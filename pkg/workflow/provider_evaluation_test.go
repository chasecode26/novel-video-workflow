package workflow

import "testing"

func TestProviderEvaluation_SubtitleGateBlocksPlanWhenSampleFails(t *testing.T) {
	result := EvaluateCandidates(EvaluationInputs{})
	if result.CanProceed {
		t.Fatal("expected evaluation gate to block")
	}
	if len(result.BlockingIssues) == 0 {
		t.Fatal("expected blocking issues when subtitle gate fails")
	}
	if result.BlockingIssues[0] != "subtitle sample gate failed" {
		t.Fatalf("expected primary blocking issue, got %q", result.BlockingIssues[0])
	}
}

func TestProviderEvaluation_SelectsPassingCandidateAndFreezesProvider(t *testing.T) {
	result := EvaluateCandidates(EvaluationInputs{
		Candidates: []CandidateEvaluation{
			{
				Name:           "aegisub-shell",
				InvocationMode: InvocationModeProcess,
				SampleResult:   CandidateSampleResult{Passed: true},
				Notes:          []string{"uses existing shell wrapper"},
			},
			{
				Name:           "whisper-http",
				InvocationMode: InvocationModeHTTP,
				SampleResult:   CandidateSampleResult{Passed: false, Error: "not configured"},
				BlockingIssues: []string{"service unavailable"},
			},
		},
	})

	if !result.CanProceed {
		t.Fatal("expected evaluation to proceed after a passing subtitle candidate")
	}
	if result.SelectedCandidateName != "aegisub-shell" {
		t.Fatalf("expected selected candidate aegisub-shell, got %q", result.SelectedCandidateName)
	}
	if result.Notes[len(result.Notes)-1] != "subtitle provider frozen for downstream Windows implementation" {
		t.Fatalf("expected freeze note, got %q", result.Notes[len(result.Notes)-1])
	}
}

func TestProviderEvaluation_AggregatesFailureDetails(t *testing.T) {
	result := EvaluateCandidates(EvaluationInputs{
		Candidates: []CandidateEvaluation{
			{
				Name:           "aegisub-gui-automation",
				InvocationMode: InvocationModeProcess,
				SampleResult:   CandidateSampleResult{Passed: false, Error: "aegisub executable not found"},
				BlockingIssues: []string{"gui automation unavailable"},
			},
			{
				Name:           "whisper-http",
				InvocationMode: InvocationModeHTTP,
				SampleResult:   CandidateSampleResult{Passed: false, Error: "provider implementation missing"},
				BlockingIssues: []string{"service unavailable"},
			},
		},
	})

	if result.CanProceed {
		t.Fatal("expected evaluation to block when all candidates fail")
	}
	if !containsString(result.BlockingIssues, "gui automation unavailable") {
		t.Fatalf("expected aggregated blocking issues, got %#v", result.BlockingIssues)
	}
	if !containsString(result.BlockingIssues, "provider implementation missing") {
		t.Fatalf("expected aggregated sample errors, got %#v", result.BlockingIssues)
	}
}

func TestProviderEvaluation_ClonesCandidateSlices(t *testing.T) {
	inputs := EvaluationInputs{
		Candidates: []CandidateEvaluation{
			{
				Name:           "aegisub-shell",
				InvocationMode: InvocationModeProcess,
				SampleResult:   CandidateSampleResult{Passed: true},
				BlockingIssues: []string{"encoding caveat"},
				Notes:          []string{"uses python fallback"},
			},
			{
				Name:           "empty-fields",
				InvocationMode: InvocationModeProcess,
				SampleResult:   CandidateSampleResult{},
			},
		},
	}

	result := EvaluateCandidates(inputs)
	inputs.Candidates[0].BlockingIssues[0] = "mutated"
	inputs.Candidates[0].Notes[0] = "mutated"

	if result.Candidates[0].BlockingIssues[0] != "encoding caveat" {
		t.Fatalf("expected cloned blocking issues, got %q", result.Candidates[0].BlockingIssues[0])
	}
	if result.Candidates[0].Notes[0] != "uses python fallback" {
		t.Fatalf("expected cloned notes, got %q", result.Candidates[0].Notes[0])
	}
	if result.Candidates[1].BlockingIssues == nil {
		t.Fatal("expected empty blocking issues slice, got nil")
	}
	if result.Candidates[1].Notes == nil {
		t.Fatal("expected empty notes slice, got nil")
	}
}

func TestProviderEvaluation_PrimaryBlockingIssue(t *testing.T) {
	candidate := CandidateEvaluation{BlockingIssues: []string{"tool missing", "ffprobe unavailable"}}
	if got := candidate.PrimaryBlockingIssue(); got != "tool missing" {
		t.Fatalf("expected first blocking issue, got %q", got)
	}
	if !candidate.HasBlockingIssue("FFPROBE unavailable") {
		t.Fatal("expected case-insensitive blocking issue lookup")
	}
}

func containsString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
