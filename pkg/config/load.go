package config

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// LoadConfig reads a config file, applies legacy key mapping, normalizes paths, and validates the result.
func LoadConfig(configPath string) (Config, error) {
	loader := viper.New()
	loader.SetConfigFile(configPath)

	if err := loader.ReadInConfig(); err != nil {
		return Config{}, fmt.Errorf("read config %q: %w", configPath, err)
	}

	cfg := defaultConfig()
	if err := loader.Unmarshal(&cfg); err != nil {
		return Config{}, fmt.Errorf("decode config %q: %w", configPath, err)
	}

	applyLegacyMapping(loader, &cfg)
	if err := normalizeConfigPaths(filepath.Dir(configPath), &cfg); err != nil {
		return Config{}, err
	}
	if err := ValidateConfig(cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func defaultConfig() Config {
	return Config{
		Paths: PathsConfig{
			OutputTemplate: "chapter_{chapter:02d}",
		},
		Workflow: WorkflowConfig{
			MaxConcurrent: 2,
			RetryAttempts: 3,
			CleanupTemp:   true,
		},
		TTS: TTSConfig{
			Provider:   "indextts2",
			PythonPath: "python",
			VoiceModel: "default",
			SampleRate: 24000,
			IndexTTS2: IndexTTS2Config{
				APIURL:         "http://127.0.0.1:7860",
				TimeoutSeconds: 300,
				MaxRetries:     3,
			},
		},
		Subtitle: SubtitleConfig{
			Provider:        "aegisub",
			Style:           "Default",
			FontName:        "Microsoft YaHei",
			FontSize:        48,
			PrimaryColor:    "&H00FFFFFF",
			SecondaryColor:  "&H0000FFFF",
			OutlineColor:    "&H00000000",
			BackColor:       "&H80000000",
			Bold:            true,
			Alignment:       2,
			MarginL:         20,
			MarginR:         20,
			MarginV:         20,
			Outline:         2,
			Shadow:          1,
			MaxCharsPerLine: 40,
			MinDisplayTime:  2,
			MaxDisplayTime:  8,
			LineInterval:    0.1,
			Aegisub: SubtitleAegisubConfig{
				ExecutablePath: "C:/Program Files/Aegisub/aegisub64.exe",
				ScriptPath:     "./pkg/tools/aegisub/aegisub_subtitle_gen.sh",
				UseAutomation:  true,
			},
		},
		Image: ImageConfig{
			Provider:       "drawthings",
			Width:          512,
			Height:         896,
			Steps:          30,
			CFGScale:       7.5,
			StylePreset:    "suspense_horror",
			NegativePrompt: "low quality, blurry, distorted, bright lighting, cheerful atmosphere",
			DrawThings: ImageDrawThingsConfig{
				APIURL:    "http://127.0.0.1:7861",
				Model:     "dreamshaper_8.safetensors",
				Scheduler: "DPM++ 2M Trailing",
			},
			ComfyUI: ImageComfyUIConfig{
				APIURL:         "http://127.0.0.1:8188",
				OutputNodeID:   "9",
				FilenamePrefix: "novel_workflow",
			},
		},
		Project: ProjectConfig{
			Provider: "capcut",
		},
		Database: DatabaseConfig{
			Path: "./db.sqlite",
		},
		Ollama: OllamaConfig{
			APIURL:         "http://127.0.0.1:11434",
			Model:          "llama3:8b",
			TimeoutSeconds: 120,
			MaxTokens:      2048,
			Temperature:    0.7,
		},
	}
}

func applyLegacyMapping(loader *viper.Viper, cfg *Config) {
	if provider := normalizeProviderName(loader.GetString("tts.engine")); provider != "" {
		cfg.TTS.Provider = provider
	}
	if loader.IsSet("tts.indexTTS_path") {
		cfg.TTS.IndexTTSPath = loader.GetString("tts.indexTTS_path")
	}
	if value, ok := firstSetString(loader, "tts.indextts2.api_url", "indextts2.api_url"); ok {
		cfg.TTS.IndexTTS2.APIURL = value
	}
	if value, ok := firstSetInt(loader, "tts.indextts2.timeout_seconds", "indextts2.timeout_seconds"); ok {
		cfg.TTS.IndexTTS2.TimeoutSeconds = value
	}
	if value, ok := firstSetInt(loader, "tts.indextts2.max_retries", "indextts2.max_retries"); ok {
		cfg.TTS.IndexTTS2.MaxRetries = value
	}

	if provider := normalizeProviderName(loader.GetString("subtitle.generator")); provider != "" {
		cfg.Subtitle.Provider = provider
	}
	if loader.IsSet("subtitle.aegisub_path") {
		cfg.Subtitle.Aegisub.ExecutablePath = loader.GetString("subtitle.aegisub_path")
	}
	if loader.IsSet("subtitle.script_path") {
		cfg.Subtitle.Aegisub.ScriptPath = loader.GetString("subtitle.script_path")
	}
	if loader.IsSet("subtitle.use_automation") {
		cfg.Subtitle.Aegisub.UseAutomation = loader.GetBool("subtitle.use_automation")
	}

	if provider := normalizeProviderName(loader.GetString("image.engine")); provider != "" {
		cfg.Image.Provider = provider
	}
	if value, ok := firstSetString(loader, "image.api_url", "drawthings.api_url"); ok {
		cfg.Image.DrawThings.APIURL = value
	}
	if value, ok := firstSetString(loader, "image.drawthings_model", "drawthings.model"); ok {
		cfg.Image.DrawThings.Model = value
	}
	if value, ok := firstSetString(loader, "image.drawthings_scheduler", "drawthings.scheduler"); ok {
		cfg.Image.DrawThings.Scheduler = value
	}
	if value, ok := firstSetString(loader, "image.comfyui.api_url", "comfyui.api_url"); ok {
		cfg.Image.ComfyUI.APIURL = value
	}
	if value, ok := firstSetString(loader, "image.comfyui.checkpoint", "comfyui.checkpoint"); ok {
		cfg.Image.ComfyUI.Checkpoint = value
	}
	if value, ok := firstSetString(loader, "image.comfyui.workflow_file", "comfyui.workflow_file"); ok {
		cfg.Image.ComfyUI.WorkflowFile = value
	}
	if value, ok := firstSetString(loader, "image.comfyui.output_node_id", "comfyui.output_node_id"); ok {
		cfg.Image.ComfyUI.OutputNodeID = value
	}
	if value, ok := firstSetString(loader, "image.comfyui.filename_prefix", "comfyui.filename_prefix"); ok {
		cfg.Image.ComfyUI.FilenamePrefix = value
	}

	cfg.Project.Provider = normalizeProviderName(cfg.Project.Provider)
	if cfg.Project.Provider == "" {
		cfg.Project.Provider = "capcut"
	}
	cfg.TTS.Provider = normalizeProviderName(cfg.TTS.Provider)
	cfg.Subtitle.Provider = normalizeProviderName(cfg.Subtitle.Provider)
	cfg.Image.Provider = normalizeProviderName(cfg.Image.Provider)
}

func firstSetString(loader *viper.Viper, keys ...string) (string, bool) {
	for _, key := range keys {
		if loader.IsSet(key) {
			return loader.GetString(key), true
		}
	}
	return "", false
}

func firstSetInt(loader *viper.Viper, keys ...string) (int, bool) {
	for _, key := range keys {
		if loader.IsSet(key) {
			return loader.GetInt(key), true
		}
	}
	return 0, false
}

func normalizeConfigPaths(configDir string, cfg *Config) error {
	if cfg.Paths.BaseDir != "" && !filepath.IsAbs(cfg.Paths.BaseDir) {
		baseDir := cfg.Paths.BaseDir
		if configDir != "" && configDir != "." {
			baseDir = filepath.Join(configDir, baseDir)
		}
		absPath, err := filepath.Abs(baseDir)
		if err != nil {
			return fmt.Errorf("normalize paths.base_dir %q: %w", cfg.Paths.BaseDir, err)
		}
		cfg.Paths.BaseDir = absPath
	}
	return nil
}

func normalizeProviderName(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case "indextts", "indextts2":
		return "indextts2"
	case "aegisub":
		return "aegisub"
	case "windows-aegisub":
		return "windows-aegisub"
	case "windows-indextts2":
		return "windows-indextts2"
	case "drawthings":
		return "drawthings"
	case "windows-drawthings":
		return "windows-drawthings"
	case "comfyui":
		return "comfyui"
	case "capcut":
		return "capcut"
	case "windows-capcut":
		return "windows-capcut"
	case "mock":
		return "mock"
	default:
		return normalized
	}
}
