package healthcheck

import "novel-video-workflow/pkg/providers"

func NewService(bundle providers.ProviderBundle) Service {
	return Service{providers: bundle}
}

func (s Service) Run() Report {
	results := []providers.HealthCheckResult{
		checkTTSProvider(s.providers.TTS),
		checkSubtitleProvider(s.providers.Subtitle),
		checkImageProvider(s.providers.Image),
		checkProjectProvider(s.providers.Project),
	}

	report := Report{CanStart: true, Results: results}
	for _, result := range results {
		switch result.Severity {
		case providers.SeverityBlocking:
			report.Blocking = append(report.Blocking, result)
			report.CanStart = false
		case providers.SeverityWarning:
			report.Warnings = append(report.Warnings, result)
		}
	}
	return report
}

func checkTTSProvider(provider providers.TTSProvider) providers.HealthCheckResult {
	if provider == nil {
		return providers.HealthCheckResult{Provider: "tts", Severity: providers.SeverityBlocking, Message: "tts provider is not configured"}
	}
	return provider.HealthCheck()
}

func checkSubtitleProvider(provider providers.SubtitleProvider) providers.HealthCheckResult {
	if provider == nil {
		return providers.HealthCheckResult{Provider: "subtitle", Severity: providers.SeverityBlocking, Message: "subtitle provider is not configured"}
	}
	return provider.HealthCheck()
}

func checkImageProvider(provider providers.ImageProvider) providers.HealthCheckResult {
	if provider == nil {
		return providers.HealthCheckResult{Provider: "image", Severity: providers.SeverityBlocking, Message: "image provider is not configured"}
	}
	return provider.HealthCheck()
}

func checkProjectProvider(provider providers.ProjectProvider) providers.HealthCheckResult {
	if provider == nil {
		return providers.HealthCheckResult{Provider: "project", Severity: providers.SeverityBlocking, Message: "project provider is not configured"}
	}
	return provider.HealthCheck()
}
