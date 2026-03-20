package providers

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	configpkg "novel-video-workflow/pkg/config"
)

func TestProviderErrorCategory_IsStable(t *testing.T) {
	cause := errors.New("dial tcp timeout")
	err := NewProviderError(CategoryConfigError, "missing endpoint", cause)
	if err.Category != CategoryConfigError {
		t.Fatalf("expected %q, got %q", CategoryConfigError, err.Category)
	}
	if err.Error() != "missing endpoint: dial tcp timeout" {
		t.Fatalf("unexpected error string %q", err.Error())
	}
	if !errors.Is(err, cause) {
		t.Fatal("expected provider error to unwrap cause")
	}
}

func TestProviderContractTypes_AreUsableTogether(t *testing.T) {
	result := HealthCheckResult{Provider: "mock", Severity: SeverityWarning, Message: "missing ffprobe"}
	if result.Severity != SeverityWarning {
		t.Fatalf("expected severity %q, got %q", SeverityWarning, result.Severity)
	}

	var _ TTSProvider = stubTTSProvider{}
	var _ SubtitleProvider = stubSubtitleProvider{}
	var _ ImageProvider = stubImageProvider{}
	var _ ProjectProvider = stubProjectProvider{}
}

func TestFactory_BuildsMockProviders(t *testing.T) {
	baseDir := t.TempDir()
	bundle, err := BuildProviders(configpkg.Config{
		Paths: configpkg.PathsConfig{BaseDir: baseDir},
		TTS:   configpkg.TTSConfig{Provider: "mock"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if bundle.TTS == nil {
		t.Fatal("expected TTS provider")
	}
}

func TestFactory_BuildsWindowsSubtitleProviderByDefault(t *testing.T) {
	baseDir := t.TempDir()
	bundle, err := BuildProviders(configpkg.Config{
		Paths:    configpkg.PathsConfig{BaseDir: baseDir},
		TTS:      configpkg.TTSConfig{Provider: "mock"},
		Subtitle: configpkg.SubtitleConfig{Provider: "windows-aegisub", Aegisub: configpkg.SubtitleAegisubConfig{ScriptPath: "./pkg/tools/aegisub/aegisub_subtitle_gen.sh"}},
		Image:    configpkg.ImageConfig{Provider: "mock"},
		Project:  configpkg.ProjectConfig{Provider: "mock"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := bundle.Subtitle.Name(); got != "windows-aegisub" {
		t.Fatalf("expected windows subtitle provider, got %q", got)
	}
}

func TestFactory_MockProvidersCreateDeterministicFiles(t *testing.T) {
	baseDir := t.TempDir()
	bundle, err := BuildProviders(configpkg.Config{
		Paths:    configpkg.PathsConfig{BaseDir: baseDir},
		TTS:      configpkg.TTSConfig{Provider: "mock"},
		Subtitle: configpkg.SubtitleConfig{Provider: "mock"},
		Image:    configpkg.ImageConfig{Provider: "mock"},
		Project:  configpkg.ProjectConfig{Provider: "mock"},
	})
	if err != nil {
		t.Fatal(err)
	}

	ttsResult, err := bundle.TTS.Generate(TTSRequest{ProjectID: "demo", ChapterNumber: 2, Text: "hello", ReferenceAudio: "ref.wav"})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := ttsResult.AudioPath, filepath.Join(baseDir, "projects", "demo", "chapters", "02", "audio", "chapter_02.wav"); got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}

	subtitleResult, err := bundle.Subtitle.Generate(SubtitleRequest{ProjectID: "demo", ChapterNumber: 2, Text: "hello", AudioPath: ttsResult.AudioPath})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := subtitleResult.SubtitlePath, filepath.Join(baseDir, "projects", "demo", "chapters", "02", "subtitle", "chapter_02.srt"); got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}

	imageResult, err := bundle.Image.Generate(ImageRequest{ProjectID: "demo", ChapterNumber: 2, Prompt: "scene", Count: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(imageResult.ImagePaths) != 2 {
		t.Fatalf("expected 2 images, got %d", len(imageResult.ImagePaths))
	}
	if got, want := imageResult.ImagePaths[0], filepath.Join(baseDir, "projects", "demo", "chapters", "02", "images", "image_01.png"); got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}

	projectResult, err := bundle.Project.Generate(ProjectRequest{ProjectID: "demo", ChapterDir: filepath.Join(baseDir, "projects", "demo", "chapters", "02"), AudioPath: ttsResult.AudioPath, SubtitlePath: subtitleResult.SubtitlePath, ImagePaths: imageResult.ImagePaths})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := projectResult.ProjectPath, filepath.Join(baseDir, "projects", "demo", "chapters", "02", "project", "capcut_project.json"); got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
	if _, err := os.Stat(filepath.Join(baseDir, "projects", "demo", "chapters", "02", "temp", "request.json")); err != nil {
		t.Fatalf("expected temp file to exist: %v", err)
	}
}

func TestFactory_RejectsUnsupportedProvider(t *testing.T) {
	_, err := BuildProviders(configpkg.Config{
		Paths: configpkg.PathsConfig{BaseDir: t.TempDir()},
		TTS:   configpkg.TTSConfig{Provider: "unknown"},
	})
	if err == nil {
		t.Fatal("expected unsupported provider error")
	}
}

type stubTTSProvider struct{}

type stubSubtitleProvider struct{}

type stubImageProvider struct{}

type stubProjectProvider struct{}

func (stubTTSProvider) Name() string                           { return "tts" }
func (stubTTSProvider) Generate(TTSRequest) (TTSResult, error) { return TTSResult{}, nil }
func (stubTTSProvider) HealthCheck() HealthCheckResult         { return HealthCheckResult{} }
func (stubSubtitleProvider) Name() string                      { return "subtitle" }
func (stubSubtitleProvider) Generate(SubtitleRequest) (SubtitleResult, error) {
	return SubtitleResult{}, nil
}
func (stubSubtitleProvider) HealthCheck() HealthCheckResult { return HealthCheckResult{} }
func (stubImageProvider) Name() string                      { return "image" }
func (stubImageProvider) Generate(ImageRequest) (ImageResult, error) {
	return ImageResult{}, nil
}
func (stubImageProvider) HealthCheck() HealthCheckResult { return HealthCheckResult{} }
func (stubProjectProvider) Name() string                 { return "project" }
func (stubProjectProvider) Generate(ProjectRequest) (ProjectResult, error) {
	return ProjectResult{}, nil
}
func (stubProjectProvider) HealthCheck() HealthCheckResult { return HealthCheckResult{} }
