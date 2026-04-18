package providers

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	configpkg "novel-video-workflow/pkg/config"
	"novel-video-workflow/pkg/tools/indextts2"

	"go.uber.org/zap"
)

type windowsTTSGenerateFunc func(referenceAudio, text, outputPath string) error

type WindowsTTSProvider struct {
	baseDir      string
	config       configpkg.TTSConfig
	generateFunc windowsTTSGenerateFunc
}

func NewWindowsTTSProvider(baseDir string, cfg configpkg.TTSConfig) WindowsTTSProvider {
	provider := WindowsTTSProvider{
		baseDir: baseDir,
		config:  cfg,
	}
	client := indextts2.NewIndexTTS2Client(zap.NewNop(), cfg.IndexTTS2.APIURL)
	provider.generateFunc = func(referenceAudio, text, outputPath string) error {
		return client.GenerateTTSWithAudio(referenceAudio, text, outputPath)
	}
	return provider
}

func (p WindowsTTSProvider) Name() string { return "windows-indextts2" }

func (p WindowsTTSProvider) HealthCheck() HealthCheckResult {
	if strings.TrimSpace(p.config.IndexTTS2.APIURL) == "" {
		return HealthCheckResult{Provider: p.Name(), Severity: SeverityBlocking, Message: "indextts2 api_url is not configured"}
	}
	return HealthCheckResult{
		Provider: p.Name(),
		Severity: SeverityInfo,
		Message:  "windows indextts2 provider ready",
	}
}

func (p WindowsTTSProvider) Generate(req TTSRequest) (TTSResult, error) {
	audioPath := req.OutputDir
	if audioPath == "" {
		audioPath = filepath.Join(p.baseDir, "projects", req.ProjectID, "chapters", fmt.Sprintf("%02d", req.ChapterNumber), "audio", fmt.Sprintf("chapter_%02d.wav", req.ChapterNumber))
	} else if filepath.Ext(audioPath) == "" {
		audioPath = filepath.Join(audioPath, fmt.Sprintf("chapter_%02d.wav", req.ChapterNumber))
	}

	if err := os.MkdirAll(filepath.Dir(audioPath), 0o755); err != nil {
		return TTSResult{}, NewProviderError(CategoryExecutionError, "create audio output directory", err)
	}

	generate := p.generateFunc
	if generate == nil {
		generate = func(referenceAudio, text, outputPath string) error {
			return writeMockAudio(outputPath, text)
		}
	}

	if err := generate(req.ReferenceAudio, req.Text, audioPath); err != nil {
		var providerErr ProviderError
		if errors.As(err, &providerErr) {
			return TTSResult{}, providerErr
		}
		return TTSResult{}, NewProviderError(CategoryExecutionError, "generate tts with windows indextts2 provider", err)
	}

	return TTSResult{AudioPath: audioPath}, nil
}

func writeMockAudio(outputPath, text string) error {
	content := fmt.Sprintf("mock audio for: %s", text)
	return os.WriteFile(outputPath, []byte(content), 0o644)
}
