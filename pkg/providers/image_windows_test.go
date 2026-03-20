package providers

import (
	"errors"
	"fmt"
	"path/filepath"
	"testing"

	configpkg "novel-video-workflow/pkg/config"
)

func newTestWindowsImageProvider(t *testing.T) WindowsImageProvider {
	t.Helper()
	return WindowsImageProvider{
		baseDir: t.TempDir(),
		config: configpkg.ImageConfig{
			Provider:       "windows-drawthings",
			Width:          512,
			Height:         896,
			Steps:          30,
			CFGScale:       7.5,
			StylePreset:    "suspense_horror",
			NegativePrompt: "low quality, blurry",
			DrawThings: configpkg.ImageDrawThingsConfig{
				APIURL:    "http://127.0.0.1:7861",
				Model:     "dreamshaper_8.safetensors",
				Scheduler: "DPM++ 2M Trailing",
			},
		},
		generateFunc: func(prompt, outputDir string, width, height int) ([]string, error) {
			return []string{filepath.Join(outputDir, "image_01.png")}, nil
		},
	}
}

func TestWindowsImageProvider_GeneratesImages(t *testing.T) {
	provider := newTestWindowsImageProvider(t)
	result, err := provider.Generate(ImageRequest{
		ProjectID:     "demo",
		ChapterNumber: 1,
		Prompt:        "夜晚街道",
		Count:         1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.ImagePaths) != 1 {
		t.Fatalf("expected 1 image, got %d", len(result.ImagePaths))
	}
}

func TestWindowsImageProvider_UsesConfiguredChapterImagesDirectory(t *testing.T) {
	provider := newTestWindowsImageProvider(t)
	result, err := provider.Generate(ImageRequest{
		ProjectID:     "demo",
		ChapterNumber: 2,
		Prompt:        "夜晚街道",
		Count:         2,
	})
	if err != nil {
		t.Fatal(err)
	}
	wantDir := filepath.Join(provider.baseDir, "projects", "demo", "chapters", "02", "images")
	for _, path := range result.ImagePaths {
		if filepath.Dir(path) != wantDir {
			t.Fatalf("expected image in %q, got %q", wantDir, filepath.Dir(path))
		}
	}
}

func TestWindowsImageProvider_ReturnsCategorizedErrorWhenGenerationFails(t *testing.T) {
	provider := newTestWindowsImageProvider(t)
	provider.generateFunc = func(prompt, outputDir string, width, height int) ([]string, error) {
		return nil, NewProviderError(CategoryExecutionError, "image generation failed", nil)
	}
	_, err := provider.Generate(ImageRequest{
		ProjectID:     "demo",
		ChapterNumber: 3,
		Prompt:        "测试失败",
		Count:         1,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	providerErr, ok := err.(ProviderError)
	if !ok {
		t.Fatalf("expected ProviderError, got %T", err)
	}
	if providerErr.Category != CategoryExecutionError {
		t.Fatalf("expected execution error, got %q", providerErr.Category)
	}
}

func TestWindowsImageProvider_HealthCheck_Ready(t *testing.T) {
	provider := WindowsImageProvider{
		baseDir: "",
		config: configpkg.ImageConfig{
			Provider: "windows-drawthings",
			DrawThings: configpkg.ImageDrawThingsConfig{
				APIURL: "http://127.0.0.1:7861",
			},
		},
	}
	result := provider.HealthCheck()
	if result.Severity != SeverityInfo {
		t.Fatalf("expected info severity, got %q", result.Severity)
	}
}

func TestWindowsImageProvider_HealthCheck_BlocksOnMissingAPIURL(t *testing.T) {
	provider := WindowsImageProvider{
		baseDir: "",
		config: configpkg.ImageConfig{
			Provider: "windows-drawthings",
			DrawThings: configpkg.ImageDrawThingsConfig{
				APIURL: "",
			},
		},
	}
	result := provider.HealthCheck()
	if result.Severity != SeverityBlocking {
		t.Fatalf("expected blocking severity, got %q", result.Severity)
	}
}

func TestWindowsImageProvider_WrapsNonProviderError(t *testing.T) {
	provider := newTestWindowsImageProvider(t)
	provider.generateFunc = func(prompt, outputDir string, width, height int) ([]string, error) {
		return nil, fmt.Errorf("underlying error")
	}
	_, err := provider.Generate(ImageRequest{
		ProjectID:     "demo",
		ChapterNumber: 4,
		Prompt:        "测试错误包装",
		Count:         1,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	var providerErr ProviderError
	if !errors.As(err, &providerErr) {
		t.Fatalf("expected ProviderError, got %T", err)
	}
	if providerErr.Category != CategoryExecutionError {
		t.Fatalf("expected execution error category, got %q", providerErr.Category)
	}
}