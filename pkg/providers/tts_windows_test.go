package providers

import (
	"errors"
	"fmt"
	"path/filepath"
	"testing"

	configpkg "novel-video-workflow/pkg/config"
)

func newTestWindowsTTSProvider(t *testing.T) WindowsTTSProvider {
	t.Helper()
	return WindowsTTSProvider{
		baseDir: t.TempDir(),
		config: configpkg.TTSConfig{
			Provider:   "windows-indextts2",
			PythonPath: "python",
			VoiceModel: "default",
			SampleRate: 24000,
			IndexTTS2: configpkg.IndexTTS2Config{
				APIURL:         "http://127.0.0.1:7860",
				TimeoutSeconds: 300,
				MaxRetries:     3,
			},
		},
		generateFunc: func(referenceAudio, text, outputPath string) error {
			return writeMockAudio(outputPath, text)
		},
	}
}

func TestWindowsTTSProvider_GeneratesAudioArtifact(t *testing.T) {
	provider := newTestWindowsTTSProvider(t)
	result, err := provider.Generate(TTSRequest{
		ProjectID:     "demo",
		ChapterNumber: 1,
		Text:          "测试文本",
		ReferenceAudio: "testdata/workflow/reference.wav",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.AudioPath == "" {
		t.Fatal("expected audio path")
	}
	if filepath.Ext(result.AudioPath) != ".wav" {
		t.Fatalf("expected .wav extension, got %q", filepath.Ext(result.AudioPath))
	}
}

func TestWindowsTTSProvider_UsesConfiguredChapterAudioDirectory(t *testing.T) {
	provider := newTestWindowsTTSProvider(t)
	result, err := provider.Generate(TTSRequest{
		ProjectID:     "demo",
		ChapterNumber: 2,
		Text:          "第二章测试",
		ReferenceAudio: "testdata/workflow/reference.wav",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(provider.baseDir, "projects", "demo", "chapters", "02", "audio", "chapter_02.wav")
	if result.AudioPath != want {
		t.Fatalf("expected %q, got %q", want, result.AudioPath)
	}
}

func TestWindowsTTSProvider_ReturnsCategorizedErrorWhenGenerationFails(t *testing.T) {
	provider := newTestWindowsTTSProvider(t)
	provider.generateFunc = func(referenceAudio, text, outputPath string) error {
		return NewProviderError(CategoryExecutionError, "tts generation failed", nil)
	}
	_, err := provider.Generate(TTSRequest{
		ProjectID:     "demo",
		ChapterNumber: 3,
		Text:          "测试失败",
		ReferenceAudio: "testdata/workflow/reference.wav",
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

func TestWindowsTTSProvider_HealthCheck_Ready(t *testing.T) {
	provider := WindowsTTSProvider{
		baseDir: "",
		config: configpkg.TTSConfig{
			Provider: "windows-indextts2",
			IndexTTS2: configpkg.IndexTTS2Config{
				APIURL: "http://127.0.0.1:7860",
			},
		},
	}
	result := provider.HealthCheck()
	if result.Severity != SeverityInfo {
		t.Fatalf("expected info severity, got %q", result.Severity)
	}
}

func TestWindowsTTSProvider_HealthCheck_BlocksOnMissingAPIURL(t *testing.T) {
	provider := WindowsTTSProvider{
		baseDir: "",
		config: configpkg.TTSConfig{
			Provider: "windows-indextts2",
			IndexTTS2: configpkg.IndexTTS2Config{
				APIURL: "",
			},
		},
	}
	result := provider.HealthCheck()
	if result.Severity != SeverityBlocking {
		t.Fatalf("expected blocking severity, got %q", result.Severity)
	}
}

func TestWindowsTTSProvider_WrapsNonProviderError(t *testing.T) {
	provider := newTestWindowsTTSProvider(t)
	provider.generateFunc = func(referenceAudio, text, outputPath string) error {
		return fmt.Errorf("underlying error")
	}
	_, err := provider.Generate(TTSRequest{
		ProjectID:     "demo",
		ChapterNumber: 4,
		Text:          "测试错误包装",
		ReferenceAudio: "testdata/workflow/reference.wav",
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