package providers

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	configpkg "novel-video-workflow/pkg/config"
	"novel-video-workflow/pkg/tools/comfyui"

	"go.uber.org/zap"
)

type ComfyUIImageProvider struct {
	baseDir string
	config  configpkg.ImageConfig
	client  *comfyui.Client
}

func NewComfyUIImageProvider(baseDir string, cfg configpkg.ImageConfig, logger *zap.Logger) ComfyUIImageProvider {
	return ComfyUIImageProvider{
		baseDir: baseDir,
		config:  cfg,
		client:  comfyui.NewClient(logger, cfg.ComfyUI),
	}
}

func (p ComfyUIImageProvider) Name() string { return "comfyui" }

func (p ComfyUIImageProvider) HealthCheck() HealthCheckResult {
	if strings.TrimSpace(p.config.ComfyUI.APIURL) == "" {
		return HealthCheckResult{Provider: p.Name(), Severity: SeverityBlocking, Message: "comfyui api_url is not configured"}
	}
	if strings.TrimSpace(p.config.ComfyUI.Checkpoint) == "" {
		return HealthCheckResult{Provider: p.Name(), Severity: SeverityBlocking, Message: "comfyui checkpoint is not configured"}
	}
	if err := p.client.CheckHealth(); err != nil {
		return HealthCheckResult{Provider: p.Name(), Severity: SeverityBlocking, Message: err.Error()}
	}
	return HealthCheckResult{Provider: p.Name(), Severity: SeverityInfo, Message: "comfyui image provider ready"}
}

func (p ComfyUIImageProvider) Generate(req ImageRequest) (ImageResult, error) {
	outputDir := req.OutputDir
	if outputDir == "" {
		outputDir = filepath.Join(p.baseDir, "projects", req.ProjectID, "chapters", fmt.Sprintf("%02d", req.ChapterNumber), "images")
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return ImageResult{}, NewProviderError(CategoryExecutionError, "create comfyui output directory", err)
	}

	count := req.Count
	if count <= 0 {
		count = 1
	}

	paths := make([]string, 0, count)
	for i := 1; i <= count; i++ {
		imagePath := filepath.Join(outputDir, fmt.Sprintf("image_%02d.png", i))
		err := p.client.GenerateImage(req.Prompt, p.config.NegativePrompt, imagePath, p.config.Width, p.config.Height, p.config.Steps, p.config.CFGScale)
		if err != nil {
			var providerErr ProviderError
			if errors.As(err, &providerErr) {
				return ImageResult{}, providerErr
			}
			return ImageResult{}, NewProviderError(CategoryExecutionError, "generate image with comfyui provider", err)
		}
		paths = append(paths, imagePath)
	}

	return ImageResult{ImagePaths: paths}, nil
}
