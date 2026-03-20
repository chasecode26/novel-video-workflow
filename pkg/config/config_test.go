package config

import (
	"path/filepath"
	"testing"
)

func TestLoadConfig_NormalizesBaseDirRelativePaths(t *testing.T) {
	cfg, err := LoadConfig("testdata/config-minimal.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(cfg.Paths.BaseDir) {
		t.Fatalf("expected absolute base dir, got %q", cfg.Paths.BaseDir)
	}
}

func TestLoadConfig_MapsLegacyFields(t *testing.T) {
	cfg, err := LoadConfig("testdata/config-legacy.yaml")
	if err != nil {
		t.Fatal(err)
	}

	if cfg.TTS.Provider != "indextts2" {
		t.Fatalf("expected tts provider indextts2, got %q", cfg.TTS.Provider)
	}
	if cfg.TTS.IndexTTS2.APIURL != "http://127.0.0.1:7860" {
		t.Fatalf("expected mapped tts api url, got %q", cfg.TTS.IndexTTS2.APIURL)
	}
	if cfg.Subtitle.Provider != "aegisub" {
		t.Fatalf("expected subtitle provider aegisub, got %q", cfg.Subtitle.Provider)
	}
	if cfg.Subtitle.Aegisub.ScriptPath != "./pkg/tools/aegisub/aegisub_subtitle_gen.sh" {
		t.Fatalf("expected mapped aegisub script path, got %q", cfg.Subtitle.Aegisub.ScriptPath)
	}
	if cfg.Image.Provider != "drawthings" {
		t.Fatalf("expected image provider drawthings, got %q", cfg.Image.Provider)
	}
	if cfg.Image.DrawThings.APIURL != "http://127.0.0.1:7861" {
		t.Fatalf("expected mapped image api url, got %q", cfg.Image.DrawThings.APIURL)
	}
	if cfg.Project.Provider != "capcut" {
		t.Fatalf("expected default project provider capcut, got %q", cfg.Project.Provider)
	}
}

func TestLoadConfig_LegacyValuesOverrideDefaults(t *testing.T) {
	cfg, err := LoadConfig("testdata/config-legacy-overrides.yaml")
	if err != nil {
		t.Fatal(err)
	}

	if cfg.TTS.IndexTTS2.APIURL != "http://legacy-host:9000" {
		t.Fatalf("expected legacy tts api url override, got %q", cfg.TTS.IndexTTS2.APIURL)
	}
	if cfg.TTS.IndexTTS2.TimeoutSeconds != 45 {
		t.Fatalf("expected legacy tts timeout override, got %d", cfg.TTS.IndexTTS2.TimeoutSeconds)
	}
	if cfg.TTS.IndexTTS2.MaxRetries != 7 {
		t.Fatalf("expected legacy tts max retries override, got %d", cfg.TTS.IndexTTS2.MaxRetries)
	}
	if cfg.Subtitle.Aegisub.ExecutablePath != "D:/tools/Aegisub/aegisub-cli.exe" {
		t.Fatalf("expected legacy aegisub path override, got %q", cfg.Subtitle.Aegisub.ExecutablePath)
	}
	if cfg.Subtitle.Aegisub.ScriptPath != "./custom/subtitles.lua" {
		t.Fatalf("expected legacy aegisub script override, got %q", cfg.Subtitle.Aegisub.ScriptPath)
	}
	if cfg.Image.DrawThings.APIURL != "http://legacy-drawthings:8080" {
		t.Fatalf("expected legacy image api url override, got %q", cfg.Image.DrawThings.APIURL)
	}
	if cfg.Image.DrawThings.Model != "custom-model.safetensors" {
		t.Fatalf("expected legacy image model override, got %q", cfg.Image.DrawThings.Model)
	}
	if cfg.Image.DrawThings.Scheduler != "Euler a" {
		t.Fatalf("expected legacy image scheduler override, got %q", cfg.Image.DrawThings.Scheduler)
	}
}

func TestLoadConfig_LegacyBooleanFalseOverridesDefault(t *testing.T) {
	cfg, err := LoadConfig("testdata/config-legacy-overrides.yaml")
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Subtitle.Aegisub.UseAutomation {
		t.Fatal("expected legacy subtitle.use_automation=false to override default true")
	}
}

func TestValidateConfig_OnlyChecksEnabledProviders(t *testing.T) {
	cfg := Config{
		Paths: PathsConfig{BaseDir: t.TempDir()},
		TTS: TTSConfig{
			Provider:  "mock",
			IndexTTS2: IndexTTS2Config{},
		},
		Subtitle: SubtitleConfig{
			Provider: "mock",
			Aegisub:  SubtitleAegisubConfig{},
		},
		Image: ImageConfig{
			Provider:   "drawthings",
			DrawThings: ImageDrawThingsConfig{},
		},
		Project: ProjectConfig{Provider: "mock"},
	}

	err := ValidateConfig(cfg)
	if err == nil {
		t.Fatal("expected validation error")
	}

	validationErrs, ok := err.(ValidationErrors)
	if !ok {
		t.Fatalf("expected ValidationErrors, got %T", err)
	}
	if len(validationErrs) != 1 {
		t.Fatalf("expected 1 validation error, got %d", len(validationErrs))
	}
	if validationErrs[0].Field != "image.drawthings.api_url" {
		t.Fatalf("expected drawthings validation error, got %q", validationErrs[0].Field)
	}
}

func TestValidateConfig_WindowsSubtitleProviderRequiresScriptPath(t *testing.T) {
	cfg := Config{
		Paths: PathsConfig{BaseDir: t.TempDir()},
		TTS:   TTSConfig{Provider: "mock"},
		Subtitle: SubtitleConfig{
			Provider: "windows-aegisub",
		},
		Image:   ImageConfig{Provider: "mock"},
		Project: ProjectConfig{Provider: "mock"},
	}

	err := ValidateConfig(cfg)
	if err == nil {
		t.Fatal("expected validation error")
	}

	validationErrs, ok := err.(ValidationErrors)
	if !ok {
		t.Fatalf("expected ValidationErrors, got %T", err)
	}
	if len(validationErrs) != 1 {
		t.Fatalf("expected 1 validation error, got %d", len(validationErrs))
	}
	if validationErrs[0].Field != "subtitle.aegisub.script_path" {
		t.Fatalf("expected windows subtitle script path validation error, got %q", validationErrs[0].Field)
	}
}
