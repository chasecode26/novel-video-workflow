package providers

type SubtitleProvider interface {
	Name() string
	Generate(SubtitleRequest) (SubtitleResult, error)
	HealthCheck() HealthCheckResult
}
