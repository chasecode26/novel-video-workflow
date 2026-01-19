package workflow

import (
	aegisub "novel-video-workflow/pkg/tools/aegisub"
	drawthings "novel-video-workflow/pkg/tools/drawthings"
	"novel-video-workflow/pkg/tools/file"
	image "novel-video-workflow/pkg/tools/image"
	"novel-video-workflow/pkg/tools/indextts2"
	"novel-video-workflow/pkg/capcut"
	"novel-video-workflow/pkg/database"

	"go.uber.org/zap"
)

type ChapterParams struct {
	Text           string
	Number         int
	ReferenceAudio string
	OutputDir      string
	MaxImages      int // 添加最大图片数参数
}

type ChapterResult struct {
	ChapterDir   string
	TextFile     string
	AudioFile    string
	SubtitleFile string
	ImageFiles   []string
	Status       string
	Message      string
	VideoProject string
	EditListFile string
}

type Processor struct {
	fileTool       *file.FileManager
	ttsTool        *indextts2.IndexTTS2Client
	aegisubTool    *aegisub.AegisubIntegration
	imageTool      *image.ImageGenerator
	drawThingsTool *drawthings.ChapterImageGenerator
	capcutTool     *capcut.CapcutGenerator
	logger         *zap.Logger
}

func NewProcessor(logger *zap.Logger) (*Processor, error) {
	// 初始化数据库
	if err := database.InitDatabaseFromConfig(logger); err != nil {
		logger.Error("数据库初始化失败", zap.Error(err))
		return nil, err
	}

	// 初始化各个工具
	fileTool := file.NewFileManager()
	ttsTool := indextts2.NewIndexTTS2Client(logger, "http://localhost:7860")
	aegisubTool := aegisub.NewAegisubIntegration()
	imageTool := image.NewImageGenerator(logger)
	drawThingsTool := drawthings.NewChapterImageGenerator(logger)
	capcutTool := capcut.NewCapcutGenerator(logger)

	return &Processor{
		fileTool:       fileTool,
		ttsTool:        ttsTool,
		aegisubTool:    aegisubTool,
		imageTool:      imageTool,
		drawThingsTool: drawThingsTool,
		capcutTool:     capcutTool,
		logger:         logger,
	}, nil
}

func (p *Processor) CreateProject(name, description, globalPrompt, password string) (*database.Project, error) {
	return database.CreateProject(name, description, globalPrompt, password)
}

func (p *Processor) GetProjectByID(id uint) (*database.Project, error) {
	return database.GetProjectByID(id)
}

func (p *Processor) GetProjectByName(name string) (*database.Project, error) {
	return database.GetProjectByName(name)
}

func (p *Processor) UpdateProject(id uint, updates map[string]interface{}) error {
	return database.UpdateProject(id, updates)
}

func (p *Processor) DeleteProject(id uint) error {
	return database.DeleteProject(id)
}

func (p *Processor) CreateChapter(projectID uint, title, content, prompt string) (*database.Chapter, error) {
	return database.CreateChapter(projectID, title, content, prompt)
}

func (p *Processor) GetChapterByID(id uint) (*database.Chapter, error) {
	return database.GetChapterByID(id)
}

func (p *Processor) GetChaptersByProjectID(projectID uint) ([]database.Chapter, error) {
	return database.GetChaptersByProjectID(projectID)
}

func (p *Processor) UpdateChapter(id uint, updates map[string]interface{}) error {
	return database.UpdateChapter(id, updates)
}

func (p *Processor) DeleteChapter(id uint) error {
	return database.DeleteChapter(id)
}

func (p *Processor) CreateScene(chapterID uint, title, description, prompt, ollamaRequest, ollamaResponse, drawThingsConfig, drawThingsResult string, order int) (*database.Scene, error) {
	return database.CreateScene(chapterID, title, description, prompt, ollamaRequest, ollamaResponse, drawThingsConfig, drawThingsResult, order)
}

func (p *Processor) GetSceneByID(id uint) (*database.Scene, error) {
	return database.GetSceneByID(id)
}

func (p *Processor) GetScenesByChapterID(chapterID uint) ([]database.Scene, error) {
	return database.GetScenesByChapterID(chapterID)
}

func (p *Processor) UpdateScene(id uint, updates map[string]interface{}) error {
	return database.UpdateScene(id, updates)
}

func (p *Processor) DeleteScene(id uint) error {
	return database.DeleteScene(id)
}

func (p *Processor) SetChapterAsShared(chapterID uint, shareToken, password string) error {
	return database.SetChapterAsShared(chapterID, shareToken, password)
}

func (p *Processor) RevokeChapterShare(chapterID uint) error {
	return database.RevokeChapterShare(chapterID)
}

func (p *Processor) GetChapterByShareToken(shareToken string) (*database.Chapter, error) {
	return database.GetChapterByShareToken(shareToken)
}

func (p *Processor) ValidateProjectPassword(projectID uint, password string) (bool, error) {
	return database.ValidateProjectPassword(projectID, password)
}

func (p *Processor) ValidateChapterSharePassword(shareToken, password string) (bool, error) {
	return database.ValidateChapterSharePassword(shareToken, password)
}

// GetAllProjects 获取所有项目
func (p *Processor) GetAllProjects() ([]database.Project, error) {
	return database.GetAllProjects()
}

func (p *Processor) generateEditList(chapterDir string, chapterNum int,
	ttsResult *indextts2.TTSResult, subtitleFile string, images []string) map[string]interface{} {

	return map[string]interface{}{
		"chapter": chapterNum,
		"assets": map[string]interface{}{
			"audio":    ttsResult.AudioPath,
			"subtitle": subtitleFile,
			"images":   images,
		},
		"timeline": []map[string]interface{}{
			{
				"time": "00:00",
				"type": "audio_start",
				"file": ttsResult.AudioPath,
			},
		},
	}
}

func (p *Processor) GenerateCapcutProject(chapterDir string) error {
	// 使用 CapCut 生成器生成剪映项目
	return p.capcutTool.GenerateProject(chapterDir)
}

func (p *Processor) GetProgress() any {
	return nil
}