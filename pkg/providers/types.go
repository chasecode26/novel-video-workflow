package providers

type ProviderErrorCategory string

type HealthCheckSeverity string

const (
	CategoryConfigError     ProviderErrorCategory = "config_error"
	CategoryDependencyError ProviderErrorCategory = "dependency_error"
	CategoryExecutionError  ProviderErrorCategory = "execution_error"
	CategoryValidationError ProviderErrorCategory = "validation_error"
	SeverityInfo            HealthCheckSeverity   = "info"
	SeverityWarning         HealthCheckSeverity   = "warning"
	SeverityBlocking        HealthCheckSeverity   = "blocking"
)

type HealthCheckResult struct {
	Provider string
	Severity HealthCheckSeverity
	Message  string
}

type TTSRequest struct {
	ProjectID      string
	ChapterNumber  int
	Text           string
	ReferenceAudio string
	OutputDir      string
}

type TTSResult struct {
	AudioPath string
	Duration  float64
}

type SubtitleRequest struct {
	ProjectID     string
	ChapterNumber int
	Text          string
	AudioPath     string
	OutputPath    string
}

type SubtitleResult struct {
	SubtitlePath string
	Format       string
}

type ImageRequest struct {
	ProjectID     string
	ChapterNumber int
	Prompt        string
	OutputDir     string
	Count         int
}

type ImageResult struct {
	ImagePaths []string
}

type ProjectRequest struct {
	ProjectID    string
	ChapterDir   string
	AudioPath    string
	SubtitlePath string
	ImagePaths   []string
}

type ProjectResult struct {
	ProjectPath  string
	EditListPath string
}
