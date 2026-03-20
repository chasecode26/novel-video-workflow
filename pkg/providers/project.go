package providers

type ProjectProvider interface {
	Name() string
	Generate(ProjectRequest) (ProjectResult, error)
	HealthCheck() HealthCheckResult
}
