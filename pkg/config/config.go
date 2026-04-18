package config

// Config defines the typed application configuration used by the provider-based workflow.
type Config struct {
	Paths    PathsConfig    `mapstructure:"paths"`
	Workflow WorkflowConfig `mapstructure:"workflow"`
	TTS      TTSConfig      `mapstructure:"tts"`
	Subtitle SubtitleConfig `mapstructure:"subtitle"`
	Image    ImageConfig    `mapstructure:"image"`
	Project  ProjectConfig  `mapstructure:"project"`
	Database DatabaseConfig `mapstructure:"database"`
	Ollama   OllamaConfig   `mapstructure:"ollama"`
}

type PathsConfig struct {
	BaseDir        string `mapstructure:"base_dir"`
	NovelSource    string `mapstructure:"novel_source"`
	ReferenceAudio string `mapstructure:"reference_audio"`
	OutputTemplate string `mapstructure:"output_template"`
}

type WorkflowConfig struct {
	MaxConcurrent int  `mapstructure:"max_concurrent"`
	RetryAttempts int  `mapstructure:"retry_attempts"`
	CleanupTemp   bool `mapstructure:"cleanup_temp"`
}

type TTSConfig struct {
	Provider     string          `mapstructure:"provider"`
	PythonPath   string          `mapstructure:"python_path"`
	IndexTTSPath string          `mapstructure:"indextts_path"`
	VoiceModel   string          `mapstructure:"voice_model"`
	SampleRate   int             `mapstructure:"sample_rate"`
	IndexTTS2    IndexTTS2Config `mapstructure:"indextts2"`
}

type IndexTTS2Config struct {
	APIURL         string `mapstructure:"api_url"`
	TimeoutSeconds int    `mapstructure:"timeout_seconds"`
	MaxRetries     int    `mapstructure:"max_retries"`
}

type SubtitleConfig struct {
	Provider        string                `mapstructure:"provider"`
	Style           string                `mapstructure:"style"`
	FontName        string                `mapstructure:"font_name"`
	FontSize        int                   `mapstructure:"font_size"`
	PrimaryColor    string                `mapstructure:"primary_color"`
	SecondaryColor  string                `mapstructure:"secondary_color"`
	OutlineColor    string                `mapstructure:"outline_color"`
	BackColor       string                `mapstructure:"back_color"`
	Bold            bool                  `mapstructure:"bold"`
	Italic          bool                  `mapstructure:"italic"`
	Underline       bool                  `mapstructure:"underline"`
	Alignment       int                   `mapstructure:"alignment"`
	MarginL         int                   `mapstructure:"margin_l"`
	MarginR         int                   `mapstructure:"margin_r"`
	MarginV         int                   `mapstructure:"margin_v"`
	Outline         float64               `mapstructure:"outline"`
	Shadow          float64               `mapstructure:"shadow"`
	MaxCharsPerLine int                   `mapstructure:"max_chars_per_line"`
	MinDisplayTime  float64               `mapstructure:"min_display_time"`
	MaxDisplayTime  float64               `mapstructure:"max_display_time"`
	LineInterval    float64               `mapstructure:"line_interval"`
	Aegisub         SubtitleAegisubConfig `mapstructure:"aegisub"`
}

type SubtitleAegisubConfig struct {
	ExecutablePath string `mapstructure:"executable_path"`
	ScriptPath     string `mapstructure:"script_path"`
	UseAutomation  bool   `mapstructure:"use_automation"`
}

type ImageConfig struct {
	Provider       string                `mapstructure:"provider"`
	Width          int                   `mapstructure:"width"`
	Height         int                   `mapstructure:"height"`
	Steps          int                   `mapstructure:"steps"`
	CFGScale       float64               `mapstructure:"cfg_scale"`
	StylePreset    string                `mapstructure:"style_preset"`
	NegativePrompt string                `mapstructure:"negative_prompt"`
	DrawThings     ImageDrawThingsConfig `mapstructure:"drawthings"`
	ComfyUI        ImageComfyUIConfig    `mapstructure:"comfyui"`
}

type ImageDrawThingsConfig struct {
	APIURL    string `mapstructure:"api_url"`
	Model     string `mapstructure:"model"`
	Scheduler string `mapstructure:"scheduler"`
}

type ImageComfyUIConfig struct {
	APIURL         string `mapstructure:"api_url"`
	Checkpoint     string `mapstructure:"checkpoint"`
	WorkflowFile   string `mapstructure:"workflow_file"`
	OutputNodeID   string `mapstructure:"output_node_id"`
	FilenamePrefix string `mapstructure:"filename_prefix"`
}

type ProjectConfig struct {
	Provider string `mapstructure:"provider"`
}

type DatabaseConfig struct {
	Path string `mapstructure:"path"`
}

type OllamaConfig struct {
	APIURL         string  `mapstructure:"api_url"`
	Model          string  `mapstructure:"model"`
	TimeoutSeconds int     `mapstructure:"timeout_seconds"`
	MaxTokens      int     `mapstructure:"max_tokens"`
	Temperature    float64 `mapstructure:"temperature"`
}
