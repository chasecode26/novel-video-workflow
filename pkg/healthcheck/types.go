package healthcheck

import "novel-video-workflow/pkg/providers"

type Report struct {
	CanStart bool
	Results  []providers.HealthCheckResult
	Warnings []providers.HealthCheckResult
	Blocking []providers.HealthCheckResult
}

type Service struct {
	providers providers.ProviderBundle
}
