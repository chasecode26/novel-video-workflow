package workflow

import "time"

// WorkflowRun stores resumable workflow execution state for a chapter.
type WorkflowRun struct {
	ChapterID     uint
	CurrentStep   Step
	Status        Status
	Artifacts     map[Step]ArtifactMetadata
	ErrorCategory string
	ErrorMessage  string
	StartedAt     *time.Time
	FinishedAt    *time.Time
}

// ResumeFrom returns the step to rerun after a failure.
func ResumeFrom(run WorkflowRun) Step {
	return run.CurrentStep
}

// PrepareResume clears the failed step and downstream artifacts while preserving successful upstream outputs.
func PrepareResume(run WorkflowRun) WorkflowRun {
	prepared := WorkflowRun{
		ChapterID:   run.ChapterID,
		CurrentStep: ResumeFrom(run),
		Status:      StatusPending,
		Artifacts:   map[Step]ArtifactMetadata{},
		StartedAt:   run.StartedAt,
	}

	invalidate := false
	for _, step := range orderedSteps() {
		if step == prepared.CurrentStep {
			invalidate = true
		}
		if invalidate {
			continue
		}
		if metadata, ok := run.Artifacts[step]; ok {
			prepared.Artifacts[step] = cloneArtifactMetadata(metadata)
		}
	}

	return prepared
}

func cloneArtifactMetadata(metadata ArtifactMetadata) ArtifactMetadata {
	if metadata == nil {
		return ArtifactMetadata{}
	}
	cloned := make(ArtifactMetadata, len(metadata))
	for key, value := range metadata {
		cloned[key] = value
	}
	return cloned
}
