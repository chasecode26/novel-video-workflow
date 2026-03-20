package providers

import (
	"fmt"
	"os"
	"path/filepath"

	"novel-video-workflow/pkg/capcut"
	configpkg "novel-video-workflow/pkg/config"
	"novel-video-workflow/pkg/tools/aegisub"
	"novel-video-workflow/pkg/tools/drawthings"
	"novel-video-workflow/pkg/tools/indextts2"

	"go.uber.org/zap"
)

type ProviderBundle struct {
	TTS      TTSProvider
	Subtitle SubtitleProvider
	Image    ImageProvider
	Project  ProjectProvider
}

func BuildProviders(cfg configpkg.Config) (ProviderBundle, error) {
	bundle := ProviderBundle{}
	logger := zap.NewNop()

	var err error
	if bundle.TTS, err = buildTTSProvider(cfg, logger); err != nil {
		return ProviderBundle{}, err
	}
	if bundle.Subtitle, err = buildSubtitleProvider(cfg); err != nil {
		return ProviderBundle{}, err
	}
	if bundle.Image, err = buildImageProvider(cfg, logger); err != nil {
		return ProviderBundle{}, err
	}
	if bundle.Project, err = buildProjectProvider(cfg, logger); err != nil {
		return ProviderBundle{}, err
	}
	return bundle, nil
}

func buildTTSProvider(cfg configpkg.Config, logger *zap.Logger) (TTSProvider, error) {
	switch cfg.TTS.Provider {
	case "", "mock":
		return MockTTSProvider{baseDir: cfg.Paths.BaseDir}, nil
	case "windows-indextts2":
		return NewWindowsTTSProvider(cfg.Paths.BaseDir, cfg.TTS), nil
	case "indextts2":
		return LegacyIndexTTS2Provider{
			baseDir: cfg.Paths.BaseDir,
			client:  indextts2.NewIndexTTS2Client(logger, cfg.TTS.IndexTTS2.APIURL),
		}, nil
	default:
		return nil, NewProviderError(CategoryConfigError, fmt.Sprintf("unsupported tts provider %q", cfg.TTS.Provider), nil)
	}
}

func buildSubtitleProvider(cfg configpkg.Config) (SubtitleProvider, error) {
	switch cfg.Subtitle.Provider {
	case "", "mock":
		return MockSubtitleProvider{baseDir: cfg.Paths.BaseDir}, nil
	case "windows-aegisub":
		provider := NewWindowsSubtitleProvider(cfg.Paths.BaseDir, cfg.Subtitle)
		return provider, nil
	case "aegisub":
		return LegacyAegisubProvider{baseDir: cfg.Paths.BaseDir, integration: aegisub.NewAegisubIntegration()}, nil
	default:
		return nil, NewProviderError(CategoryConfigError, fmt.Sprintf("unsupported subtitle provider %q", cfg.Subtitle.Provider), nil)
	}
}

func buildImageProvider(cfg configpkg.Config, logger *zap.Logger) (ImageProvider, error) {
	switch cfg.Image.Provider {
	case "", "mock":
		return MockImageProvider{baseDir: cfg.Paths.BaseDir}, nil
	case "windows-drawthings":
		return NewWindowsImageProvider(cfg.Paths.BaseDir, cfg.Image), nil
	case "drawthings":
		return LegacyDrawThingsProvider{
			baseDir: cfg.Paths.BaseDir,
			width:   cfg.Image.Width,
			height:  cfg.Image.Height,
			client:  drawthings.NewChapterImageGenerator(logger),
		}, nil
	default:
		return nil, NewProviderError(CategoryConfigError, fmt.Sprintf("unsupported image provider %q", cfg.Image.Provider), nil)
	}
}

func buildProjectProvider(cfg configpkg.Config, logger *zap.Logger) (ProjectProvider, error) {
	switch cfg.Project.Provider {
	case "", "mock":
		return MockProjectProvider{baseDir: cfg.Paths.BaseDir}, nil
	case "capcut":
		return LegacyCapCutProvider{baseDir: cfg.Paths.BaseDir, generator: capcut.NewCapcutGenerator(logger)}, nil
	default:
		return nil, NewProviderError(CategoryConfigError, fmt.Sprintf("unsupported project provider %q", cfg.Project.Provider), nil)
	}
}

type MockTTSProvider struct{ baseDir string }

type MockSubtitleProvider struct{ baseDir string }

type MockImageProvider struct{ baseDir string }

type MockProjectProvider struct{ baseDir string }

func (p MockTTSProvider) Name() string { return "mock" }
func (p MockTTSProvider) HealthCheck() HealthCheckResult {
	return HealthCheckResult{Provider: p.Name(), Severity: SeverityInfo, Message: "mock tts ready"}
}
func (p MockTTSProvider) Generate(req TTSRequest) (TTSResult, error) {
	chapterDir := mockChapterDir(p.baseDir, req.ProjectID, req.ChapterNumber)
	textPath := filepath.Join(chapterDir, "text", chapterFile(req.ChapterNumber, ".txt"))
	audioPath := filepath.Join(chapterDir, "audio", chapterFile(req.ChapterNumber, ".wav"))
	tempPath := filepath.Join(chapterDir, "temp", "request.json")
	if err := ensureFile(textPath, []byte(req.Text)); err != nil {
		return TTSResult{}, err
	}
	if err := ensureFile(audioPath, []byte("mock audio for "+req.Text)); err != nil {
		return TTSResult{}, err
	}
	if err := ensureFile(tempPath, []byte(req.ReferenceAudio)); err != nil {
		return TTSResult{}, err
	}
	return TTSResult{AudioPath: audioPath, Duration: 1}, nil
}

func (p MockSubtitleProvider) Name() string { return "mock" }
func (p MockSubtitleProvider) HealthCheck() HealthCheckResult {
	return HealthCheckResult{Provider: p.Name(), Severity: SeverityInfo, Message: "mock subtitle ready"}
}
func (p MockSubtitleProvider) Generate(req SubtitleRequest) (SubtitleResult, error) {
	subtitlePath := filepath.Join(mockChapterDir(p.baseDir, req.ProjectID, req.ChapterNumber), "subtitle", chapterFile(req.ChapterNumber, ".srt"))
	content := fmt.Sprintf("1\n00:00:00,000 --> 00:00:01,000\n%s\n", req.Text)
	if err := ensureFile(subtitlePath, []byte(content)); err != nil {
		return SubtitleResult{}, err
	}
	return SubtitleResult{SubtitlePath: subtitlePath, Format: "srt"}, nil
}

func (p MockImageProvider) Name() string { return "mock" }
func (p MockImageProvider) HealthCheck() HealthCheckResult {
	return HealthCheckResult{Provider: p.Name(), Severity: SeverityInfo, Message: "mock image ready"}
}
func (p MockImageProvider) Generate(req ImageRequest) (ImageResult, error) {
	count := req.Count
	if count <= 0 {
		count = 1
	}
	paths := make([]string, 0, count)
	imagesDir := filepath.Join(mockChapterDir(p.baseDir, req.ProjectID, req.ChapterNumber), "images")
	for i := 1; i <= count; i++ {
		imagePath := filepath.Join(imagesDir, fmt.Sprintf("image_%02d.png", i))
		if err := ensureFile(imagePath, []byte(req.Prompt)); err != nil {
			return ImageResult{}, err
		}
		paths = append(paths, imagePath)
	}
	return ImageResult{ImagePaths: paths}, nil
}

func (p MockProjectProvider) Name() string { return "mock" }
func (p MockProjectProvider) HealthCheck() HealthCheckResult {
	return HealthCheckResult{Provider: p.Name(), Severity: SeverityInfo, Message: "mock project ready"}
}
func (p MockProjectProvider) Generate(req ProjectRequest) (ProjectResult, error) {
	chapterDir := req.ChapterDir
	if chapterDir == "" {
		chapterDir = mockChapterDir(p.baseDir, req.ProjectID, 0)
	}
	projectPath := filepath.Join(chapterDir, "project", "capcut_project.json")
	editListPath := filepath.Join(chapterDir, "project", "edit_list.json")
	if err := ensureFile(projectPath, []byte("{}")); err != nil {
		return ProjectResult{}, err
	}
	if err := ensureFile(editListPath, []byte("{}")); err != nil {
		return ProjectResult{}, err
	}
	return ProjectResult{ProjectPath: projectPath, EditListPath: editListPath}, nil
}

type LegacyIndexTTS2Provider struct {
	baseDir string
	client  *indextts2.IndexTTS2Client
}

type LegacyAegisubProvider struct {
	baseDir     string
	integration *aegisub.AegisubIntegration
}

type LegacyDrawThingsProvider struct {
	baseDir string
	width   int
	height  int
	client  *drawthings.ChapterImageGenerator
}

type LegacyCapCutProvider struct {
	baseDir   string
	generator *capcut.CapcutGenerator
}

func (p LegacyIndexTTS2Provider) Name() string { return "indextts2" }
func (p LegacyIndexTTS2Provider) HealthCheck() HealthCheckResult {
	return HealthCheckResult{Provider: p.Name(), Severity: SeverityInfo, Message: "legacy indextts2 configured"}
}
func (p LegacyIndexTTS2Provider) Generate(req TTSRequest) (TTSResult, error) {
	audioPath := req.OutputDir
	if audioPath == "" {
		audioPath = filepath.Join(mockChapterDir(p.baseDir, req.ProjectID, req.ChapterNumber), "audio", chapterFile(req.ChapterNumber, ".wav"))
	} else if filepath.Ext(audioPath) == "" {
		audioPath = filepath.Join(audioPath, chapterFile(req.ChapterNumber, ".wav"))
	}
	if err := ensureDir(filepath.Dir(audioPath)); err != nil {
		return TTSResult{}, err
	}
	if err := p.client.GenerateTTSWithAudio(req.ReferenceAudio, req.Text, audioPath); err != nil {
		return TTSResult{}, NewProviderError(CategoryExecutionError, "generate tts with indextts2", err)
	}
	return TTSResult{AudioPath: audioPath}, nil
}

func (p LegacyAegisubProvider) Name() string { return "aegisub" }
func (p LegacyAegisubProvider) HealthCheck() HealthCheckResult {
	return HealthCheckResult{Provider: p.Name(), Severity: SeverityInfo, Message: "legacy aegisub configured"}
}
func (p LegacyAegisubProvider) Generate(req SubtitleRequest) (SubtitleResult, error) {
	outputPath := req.OutputPath
	if outputPath == "" {
		outputPath = filepath.Join(mockChapterDir(p.baseDir, req.ProjectID, req.ChapterNumber), "subtitle", chapterFile(req.ChapterNumber, ".srt"))
	}
	if err := ensureDir(filepath.Dir(outputPath)); err != nil {
		return SubtitleResult{}, err
	}
	if err := p.integration.ProcessIndextts2OutputWithCustomName(req.AudioPath, req.Text, outputPath); err != nil {
		return SubtitleResult{}, NewProviderError(CategoryExecutionError, "generate subtitle with aegisub", err)
	}
	return SubtitleResult{SubtitlePath: outputPath, Format: "srt"}, nil
}

func (p LegacyDrawThingsProvider) Name() string { return "drawthings" }
func (p LegacyDrawThingsProvider) HealthCheck() HealthCheckResult {
	return HealthCheckResult{Provider: p.Name(), Severity: SeverityInfo, Message: "legacy drawthings configured"}
}
func (p LegacyDrawThingsProvider) Generate(req ImageRequest) (ImageResult, error) {
	outputDir := req.OutputDir
	if outputDir == "" {
		outputDir = filepath.Join(mockChapterDir(p.baseDir, req.ProjectID, req.ChapterNumber), "images")
	}
	if err := ensureDir(outputDir); err != nil {
		return ImageResult{}, err
	}
	width := p.width
	if width == 0 {
		width = 512
	}
	height := p.height
	if height == 0 {
		height = 896
	}
	paths, err := p.client.GenerateImageSequenceFromText(req.Prompt, outputDir, "image", width, height, false)
	if err != nil {
		return ImageResult{}, NewProviderError(CategoryExecutionError, "generate images with drawthings", err)
	}
	return ImageResult{ImagePaths: paths}, nil
}

func (p LegacyCapCutProvider) Name() string { return "capcut" }
func (p LegacyCapCutProvider) HealthCheck() HealthCheckResult {
	return HealthCheckResult{Provider: p.Name(), Severity: SeverityInfo, Message: "legacy capcut configured"}
}
func (p LegacyCapCutProvider) Generate(req ProjectRequest) (ProjectResult, error) {
	projectDir := filepath.Join(req.ChapterDir, "project")
	if err := ensureDir(projectDir); err != nil {
		return ProjectResult{}, err
	}
	if err := p.generator.GenerateProjectWithOutputDir(req.ChapterDir, projectDir); err != nil {
		return ProjectResult{}, NewProviderError(CategoryExecutionError, "generate project with capcut", err)
	}
	projectPath := filepath.Join(projectDir, "draft_content.json")
	return ProjectResult{ProjectPath: projectPath, EditListPath: projectPath}, nil
}

func mockChapterDir(baseDir, projectID string, chapterNumber int) string {
	return filepath.Join(baseDir, "projects", projectID, "chapters", fmt.Sprintf("%02d", chapterNumber))
}

func chapterFile(chapterNumber int, ext string) string {
	return fmt.Sprintf("chapter_%02d%s", chapterNumber, ext)
}

func ensureDir(dir string) error {
	return os.MkdirAll(dir, 0o755)
}

func ensureFile(path string, content []byte) error {
	if err := ensureDir(filepath.Dir(path)); err != nil {
		return err
	}
	return os.WriteFile(path, content, 0o644)
}
