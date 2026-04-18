package providers

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	configpkg "novel-video-workflow/pkg/config"
	"novel-video-workflow/pkg/tools/aegisub"
)

type windowsSubtitleGenerateFunc func(audioPath, text, outputPath string) error

type WindowsSubtitleProvider struct {
	baseDir      string
	config       configpkg.SubtitleConfig
	generateFunc windowsSubtitleGenerateFunc
}

func NewWindowsSubtitleProvider(baseDir string, cfg configpkg.SubtitleConfig) WindowsSubtitleProvider {
	provider := WindowsSubtitleProvider{
		baseDir: baseDir,
		config:  cfg,
	}
	integration := aegisub.NewAegisubIntegration()
	provider.generateFunc = func(audioPath, text, outputPath string) error {
		return integration.ProcessIndextts2OutputWithCustomName(audioPath, text, outputPath)
	}
	return provider
}

func (p WindowsSubtitleProvider) Name() string { return "windows-aegisub" }

func (p WindowsSubtitleProvider) HealthCheck() HealthCheckResult {
	if strings.TrimSpace(p.config.Aegisub.ScriptPath) == "" {
		return HealthCheckResult{Provider: p.Name(), Severity: SeverityBlocking, Message: "aegisub script path is not configured"}
	}
	return HealthCheckResult{Provider: p.Name(), Severity: SeverityInfo, Message: "windows aegisub subtitle provider ready"}
}

func (p WindowsSubtitleProvider) Generate(req SubtitleRequest) (SubtitleResult, error) {
	outputPath := req.OutputPath
	if outputPath == "" {
		outputPath = filepath.Join(p.baseDir, "projects", req.ProjectID, "chapters", fmt.Sprintf("%02d", req.ChapterNumber), "subtitle", fmt.Sprintf("chapter_%02d.srt", req.ChapterNumber))
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return SubtitleResult{}, NewProviderError(CategoryExecutionError, "create subtitle output directory", err)
	}
	generate := p.generateFunc
	if generate == nil {
		generate = p.generateWithAegisub
	}
	if err := generate(req.AudioPath, req.Text, outputPath); err != nil {
		var providerErr ProviderError
		if errors.As(err, &providerErr) {
			return SubtitleResult{}, providerErr
		}
		return SubtitleResult{}, NewProviderError(CategoryExecutionError, "generate subtitle with windows aegisub provider", err)
	}
	return SubtitleResult{SubtitlePath: outputPath, Format: "srt"}, nil
}

func (p WindowsSubtitleProvider) generateWithAegisub(audioPath, text, outputPath string) error {
	integration := aegisub.NewAegisubIntegration()
	return integration.ProcessIndextts2OutputWithCustomName(audioPath, text, outputPath)
}

func writeWindowsSubtitleSRT(outputPath, text string) error {
	content := fmt.Sprintf("1\n00:00:00,000 --> 00:00:02,000\n%s\n\n", text)
	return os.WriteFile(outputPath, []byte(content), 0o644)
}
