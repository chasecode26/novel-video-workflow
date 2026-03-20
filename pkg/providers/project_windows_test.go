package providers

import (
	"os"
	"path/filepath"
	"testing"

	configpkg "novel-video-workflow/pkg/config"
)

func newTestWindowsCapcutProvider(t *testing.T) WindowsCapcutProvider {
	t.Helper()
	return WindowsCapcutProvider{
		baseDir: t.TempDir(),
		config: configpkg.ProjectConfig{
			Provider: "windows-capcut",
		},
		generateFunc: func(chapterDir, projectDir string) error {
			return writeMockProject(projectDir)
		},
	}
}

func TestWindowsCapcutProvider_GeneratesProjectWithWindowsPaths(t *testing.T) {
	provider := newTestWindowsCapcutProvider(t)
	chapterDir := filepath.Join(provider.baseDir, "projects", "demo", "chapters", "01")
	if err := os.MkdirAll(chapterDir, 0o755); err != nil {
		t.Fatal(err)
	}

	result, err := provider.Generate(ProjectRequest{
		ProjectID:  "demo",
		ChapterDir: chapterDir,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ProjectPath == "" {
		t.Fatal("expected project output path")
	}
	if filepath.IsAbs(result.ProjectPath) == false {
		t.Fatalf("expected absolute path, got %q", result.ProjectPath)
	}
}

func TestWindowsCapcutProvider_UsesConfiguredProjectDirectory(t *testing.T) {
	provider := newTestWindowsCapcutProvider(t)
	chapterDir := filepath.Join(provider.baseDir, "projects", "demo", "chapters", "02")
	if err := os.MkdirAll(chapterDir, 0o755); err != nil {
		t.Fatal(err)
	}

	result, err := provider.Generate(ProjectRequest{
		ProjectID:  "demo",
		ChapterDir: chapterDir,
	})
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(chapterDir, "project", "draft_content.json")
	if result.ProjectPath != want {
		t.Fatalf("expected %q, got %q", want, result.ProjectPath)
	}
}

func TestWindowsCapcutProvider_ReturnsCategorizedErrorWhenGenerationFails(t *testing.T) {
	provider := newTestWindowsCapcutProvider(t)
	provider.generateFunc = func(chapterDir, projectDir string) error {
		return NewProviderError(CategoryExecutionError, "project generation failed", nil)
	}
	chapterDir := filepath.Join(provider.baseDir, "projects", "demo", "chapters", "03")
	if err := os.MkdirAll(chapterDir, 0o755); err != nil {
		t.Fatal(err)
	}

	_, err := provider.Generate(ProjectRequest{
		ProjectID:  "demo",
		ChapterDir: chapterDir,
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

func TestWindowsCapcutProvider_HealthCheck_Ready(t *testing.T) {
	provider := WindowsCapcutProvider{
		baseDir: "",
		config: configpkg.ProjectConfig{
			Provider: "windows-capcut",
		},
	}
	result := provider.HealthCheck()
	if result.Severity != SeverityInfo {
		t.Fatalf("expected info severity, got %q", result.Severity)
	}
}