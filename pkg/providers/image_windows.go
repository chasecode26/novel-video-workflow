package providers

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	configpkg "novel-video-workflow/pkg/config"
)

type windowsImageGenerateFunc func(prompt, outputDir string, width, height int) ([]string, error)

type WindowsImageProvider struct {
	baseDir      string
	config       configpkg.ImageConfig
	generateFunc windowsImageGenerateFunc
}

func NewWindowsImageProvider(baseDir string, cfg configpkg.ImageConfig) WindowsImageProvider {
	return WindowsImageProvider{
		baseDir:      baseDir,
		config:       cfg,
		generateFunc: nil, // Will use DrawThings client in future integration
	}
}

func (p WindowsImageProvider) Name() string { return "windows-drawthings" }

func (p WindowsImageProvider) HealthCheck() HealthCheckResult {
	if strings.TrimSpace(p.config.DrawThings.APIURL) == "" {
		return HealthCheckResult{Provider: p.Name(), Severity: SeverityBlocking, Message: "drawthings api_url is not configured"}
	}
	return HealthCheckResult{
		Provider: p.Name(),
		Severity: SeverityInfo,
		Message:  "windows drawthings provider ready",
	}
}

func (p WindowsImageProvider) Generate(req ImageRequest) (ImageResult, error) {
	outputDir := req.OutputDir
	if outputDir == "" {
		outputDir = filepath.Join(p.baseDir, "projects", req.ProjectID, "chapters", fmt.Sprintf("%02d", req.ChapterNumber), "images")
	}

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return ImageResult{}, NewProviderError(CategoryExecutionError, "create images output directory", err)
	}

	width := p.config.Width
	if width == 0 {
		width = 512
	}
	height := p.config.Height
	if height == 0 {
		height = 896
	}

	generate := p.generateFunc
	if generate == nil {
		generate = func(prompt, outputDir string, width, height int) ([]string, error) {
			count := req.Count
			if count <= 0 {
				count = 1
			}
			paths := make([]string, 0, count)
			for i := 1; i <= count; i++ {
				imagePath := filepath.Join(outputDir, fmt.Sprintf("image_%02d.png", i))
				if err := writeMockImage(imagePath, prompt); err != nil {
					return nil, err
				}
				paths = append(paths, imagePath)
			}
			return paths, nil
		}
	}

	paths, err := generate(req.Prompt, outputDir, width, height)
	if err != nil {
		var providerErr ProviderError
		if errors.As(err, &providerErr) {
			return ImageResult{}, providerErr
		}
		return ImageResult{}, NewProviderError(CategoryExecutionError, "generate images with windows drawthings provider", err)
	}

	return ImageResult{ImagePaths: paths}, nil
}

func writeMockImage(outputPath, prompt string) error {
	content := fmt.Sprintf("mock image for: %s", prompt)
	return os.WriteFile(outputPath, []byte(content), 0o644)
}