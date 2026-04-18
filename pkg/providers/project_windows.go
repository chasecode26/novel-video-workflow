package providers

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"novel-video-workflow/pkg/capcut"
	configpkg "novel-video-workflow/pkg/config"
)

type windowsProjectGenerateFunc func(chapterDir, projectDir string) error

type WindowsCapcutProvider struct {
	baseDir      string
	config       configpkg.ProjectConfig
	generateFunc windowsProjectGenerateFunc
}

func NewWindowsCapcutProvider(baseDir string, cfg configpkg.ProjectConfig) WindowsCapcutProvider {
	provider := WindowsCapcutProvider{
		baseDir: baseDir,
		config:  cfg,
	}
	generator := capcut.NewCapcutGenerator(nil)
	provider.generateFunc = func(chapterDir, projectDir string) error {
		return generator.GenerateProjectWithOutputDir(chapterDir, projectDir)
	}
	return provider
}

func (p WindowsCapcutProvider) Name() string { return "windows-capcut" }

func (p WindowsCapcutProvider) HealthCheck() HealthCheckResult {
	return HealthCheckResult{
		Provider: p.Name(),
		Severity: SeverityInfo,
		Message:  "windows capcut provider ready",
	}
}

func (p WindowsCapcutProvider) Generate(req ProjectRequest) (ProjectResult, error) {
	projectDir := filepath.Join(req.ChapterDir, "project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		return ProjectResult{}, NewProviderError(CategoryExecutionError, "create project output directory", err)
	}

	generate := p.generateFunc
	if generate == nil {
		generate = func(chapterDir, projectDir string) error {
			return writeMockProject(projectDir)
		}
	}

	if err := generate(req.ChapterDir, projectDir); err != nil {
		var providerErr ProviderError
		if errors.As(err, &providerErr) {
			return ProjectResult{}, providerErr
		}
		return ProjectResult{}, NewProviderError(CategoryExecutionError, "generate project with windows capcut provider", err)
	}

	projectPath := filepath.Join(projectDir, "draft_content.json")
	editListPath := filepath.Join(projectDir, "edit_list.json")

	return ProjectResult{
		ProjectPath:  projectPath,
		EditListPath: editListPath,
	}, nil
}

func writeMockProject(projectDir string) error {
	// Write minimal draft_content.json
	draftContent := map[string]interface{}{
		"canvas_config": map[string]interface{}{
			"width":  1080,
			"height": 1920,
		},
		"tracks": []interface{}{},
	}
	draftContentJSON, err := json.MarshalIndent(draftContent, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(projectDir, "draft_content.json"), draftContentJSON, 0o644); err != nil {
		return err
	}

	// Write minimal edit_list.json
	editList := map[string]interface{}{
		"scenes": []interface{}{},
	}
	editListJSON, err := json.MarshalIndent(editList, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(projectDir, "edit_list.json"), editListJSON, 0o644)
}
