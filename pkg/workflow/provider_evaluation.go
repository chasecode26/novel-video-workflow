package workflow

import "strings"

type InvocationMode string

const (
	InvocationModeHTTP    InvocationMode = "http"
	InvocationModeProcess InvocationMode = "process"
)

type CandidateSampleResult struct {
	Passed bool
	Error  string
}

type CandidateEvaluation struct {
	Name           string
	InvocationMode InvocationMode
	SampleResult   CandidateSampleResult
	BlockingIssues []string
	Notes          []string
}

type EvaluationInputs struct {
	Candidates []CandidateEvaluation
}

type EvaluationResult struct {
	CanProceed            bool
	BlockingIssues        []string
	Notes                 []string
	Candidates            []CandidateEvaluation
	SelectedCandidateName string
}

func EvaluateCandidates(inputs EvaluationInputs) EvaluationResult {
	result := EvaluationResult{
		BlockingIssues: make([]string, 0),
		Notes: []string{
			"subtitle candidate must pass minimal sample before large follow-on work proceeds",
		},
		Candidates: cloneCandidates(inputs.Candidates),
	}

	for _, candidate := range result.Candidates {
		if candidate.SampleResult.Passed {
			result.CanProceed = true
			result.SelectedCandidateName = candidate.Name
			result.Notes = append(result.Notes, "subtitle provider frozen for downstream Windows implementation")
			return result
		}

		appendCandidateFailure(&result, candidate)
	}

	result.BlockingIssues = append([]string{"subtitle sample gate failed"}, result.BlockingIssues...)
	result.Notes = append(result.Notes, "remaining implementation tasks stay blocked until a subtitle candidate passes the minimal sample")
	return result
}

func appendCandidateFailure(result *EvaluationResult, candidate CandidateEvaluation) {
	result.BlockingIssues = append(result.BlockingIssues, candidate.BlockingIssues...)
	if candidate.SampleResult.Error != "" {
		result.BlockingIssues = append(result.BlockingIssues, candidate.SampleResult.Error)
	}
}

func cloneCandidates(candidates []CandidateEvaluation) []CandidateEvaluation {
	cloned := make([]CandidateEvaluation, len(candidates))
	for i, candidate := range candidates {
		cloned[i] = CandidateEvaluation{
			Name:           candidate.Name,
			InvocationMode: candidate.InvocationMode,
			SampleResult:   candidate.SampleResult,
			BlockingIssues: cloneStrings(candidate.BlockingIssues),
			Notes:          cloneStrings(candidate.Notes),
		}
	}
	return cloned
}

func cloneStrings(items []string) []string {
	if len(items) == 0 {
		return []string{}
	}
	return append([]string{}, items...)
}

func (c CandidateEvaluation) PrimaryBlockingIssue() string {
	if len(c.BlockingIssues) == 0 {
		return ""
	}
	return c.BlockingIssues[0]
}

func (c CandidateEvaluation) HasBlockingIssue(issue string) bool {
	for _, current := range c.BlockingIssues {
		if strings.EqualFold(current, issue) {
			return true
		}
	}
	return false
}
