package providers

type ImageProvider interface {
	Name() string
	Generate(ImageRequest) (ImageResult, error)
	HealthCheck() HealthCheckResult
}
