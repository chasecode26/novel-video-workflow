package workflow

import "testing"

func TestWorkflowRun_CanResumeFromLastSuccessfulStep(t *testing.T) {
	run := WorkflowRun{CurrentStep: StepSubtitle, Status: StatusFailed}
	next := ResumeFrom(run)
	if next != StepSubtitle {
		t.Fatalf("expected resume at subtitle, got %v", next)
	}
}

func TestWorkflowRun_PrepareResumeInvalidatesFailedAndDownstreamArtifacts(t *testing.T) {
	run := WorkflowRun{
		CurrentStep: StepImage,
		Status:      StatusFailed,
		Artifacts: map[Step]ArtifactMetadata{
			StepTTS:      {"audio_path": "chapter.wav"},
			StepSubtitle: {"subtitle_path": "chapter.srt"},
			StepImage:    {"image_paths": []string{"scene-1.png"}},
			StepProject:  {"project_path": "draft_content.json"},
		},
	}

	prepared := PrepareResume(run)

	if prepared.Status != StatusPending {
		t.Fatalf("expected status %q, got %q", StatusPending, prepared.Status)
	}
	if prepared.CurrentStep != StepImage {
		t.Fatalf("expected current step %q, got %q", StepImage, prepared.CurrentStep)
	}
	if _, ok := prepared.Artifacts[StepTTS]; !ok {
		t.Fatal("expected tts artifact to remain reusable")
	}
	if _, ok := prepared.Artifacts[StepSubtitle]; !ok {
		t.Fatal("expected subtitle artifact to remain reusable")
	}
	if _, ok := prepared.Artifacts[StepImage]; ok {
		t.Fatal("expected failed step artifact to be invalidated")
	}
	if _, ok := prepared.Artifacts[StepProject]; ok {
		t.Fatal("expected downstream project artifact to be invalidated")
	}
}
