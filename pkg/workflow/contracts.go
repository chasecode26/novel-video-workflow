package workflow

// Step defines the persisted workflow step enum.
type Step string

const (
	StepTTS      Step = "tts"
	StepSubtitle Step = "subtitle"
	StepImage    Step = "image"
	StepProject  Step = "project"
)

// Status defines the persisted workflow status enum.
type Status string

const (
	StatusPending   Status = "pending"
	StatusRunning   Status = "running"
	StatusSucceeded Status = "succeeded"
	StatusFailed    Status = "failed"
	StatusCanceled  Status = "canceled"
)

// ArtifactMetadata stores provider result metadata owned by workflow persistence.
type ArtifactMetadata map[string]interface{}

func orderedSteps() []Step {
	return []Step{StepTTS, StepSubtitle, StepImage, StepProject}
}
