package providers

type TTSProvider interface {
	Name() string
	Generate(TTSRequest) (TTSResult, error)
	HealthCheck() HealthCheckResult
}
