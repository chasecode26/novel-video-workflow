package healthcheck

import (
	"testing"

	"novel-video-workflow/pkg/providers"
)

type stubTTSProvider struct{ result providers.HealthCheckResult }
type stubSubtitleProvider struct{ result providers.HealthCheckResult }
type stubImageProvider struct{ result providers.HealthCheckResult }
type stubProjectProvider struct{ result providers.HealthCheckResult }

func (p stubTTSProvider) Name() string { return p.result.Provider }
func (p stubTTSProvider) Generate(providers.TTSRequest) (providers.TTSResult, error) {
	return providers.TTSResult{}, nil
}
func (p stubTTSProvider) HealthCheck() providers.HealthCheckResult { return p.result }

func (p stubSubtitleProvider) Name() string { return p.result.Provider }
func (p stubSubtitleProvider) Generate(providers.SubtitleRequest) (providers.SubtitleResult, error) {
	return providers.SubtitleResult{}, nil
}
func (p stubSubtitleProvider) HealthCheck() providers.HealthCheckResult { return p.result }

func (p stubImageProvider) Name() string { return p.result.Provider }
func (p stubImageProvider) Generate(providers.ImageRequest) (providers.ImageResult, error) {
	return providers.ImageResult{}, nil
}
func (p stubImageProvider) HealthCheck() providers.HealthCheckResult { return p.result }

func (p stubProjectProvider) Name() string { return p.result.Provider }
func (p stubProjectProvider) Generate(providers.ProjectRequest) (providers.ProjectResult, error) {
	return providers.ProjectResult{}, nil
}
func (p stubProjectProvider) HealthCheck() providers.HealthCheckResult { return p.result }

func TestHealthCheckService_FailsOnBlockingIssue(t *testing.T) {
	svc := NewService(providers.ProviderBundle{
		TTS:      stubTTSProvider{result: providers.HealthCheckResult{Provider: "tts", Severity: providers.SeverityInfo, Message: "ready"}},
		Subtitle: stubSubtitleProvider{result: providers.HealthCheckResult{Provider: "subtitle", Severity: providers.SeverityBlocking, Message: "missing automation dependency"}},
		Image:    stubImageProvider{result: providers.HealthCheckResult{Provider: "image", Severity: providers.SeverityWarning, Message: "fallback mode"}},
		Project:  stubProjectProvider{result: providers.HealthCheckResult{Provider: "project", Severity: providers.SeverityInfo, Message: "ready"}},
	})
	result := svc.Run()
	if result.CanStart {
		t.Fatal("expected startup to be blocked")
	}
	if len(result.Blocking) != 1 {
		t.Fatalf("expected 1 blocking issue, got %d", len(result.Blocking))
	}
}
