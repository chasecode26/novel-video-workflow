package workflow

import (
	"encoding/json"
	"fmt"
	"time"

	"novel-video-workflow/pkg/capcut"
	configpkg "novel-video-workflow/pkg/config"
	"novel-video-workflow/pkg/database"
	"novel-video-workflow/pkg/providers"

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
	config     configpkg.Config
	providers  providers.ProviderBundle
	capcutTool *capcut.CapcutGenerator
	logger     *zap.Logger
}

func NewProcessor(cfg configpkg.Config, bundle providers.ProviderBundle, logger *zap.Logger) (*Processor, error) {
	if err := database.InitDB(cfg.Database.Path); err != nil {
		logger.Error("数据库初始化失败", zap.Error(err))
		return nil, err
	}

	return &Processor{
		config:     cfg,
		providers:  bundle,
		capcutTool: capcut.NewCapcutGenerator(logger),
		logger:     logger,
	}, nil
}

func (p *Processor) GetProviders() providers.ProviderBundle {
	return p.providers
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

func (p *Processor) ValidateProjectPassword(projectID uint, password string) (bool, error) {
	return database.ValidateProjectPassword(projectID, password)
}

// GetAllProjects 获取所有项目
func (p *Processor) GetAllProjects() ([]database.Project, error) {
	return database.GetAllProjects()
}

func (p *Processor) generateEditList(chapterDir string, chapterNum int,
	ttsResult *providers.TTSResult, subtitleFile string, images []string) map[string]interface{} {

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

func (p *Processor) UpdateSceneWithWorkflowDetails(id uint, updates map[string]interface{}) error {
	// 如果包含workflow_details，需要序列化
	if workflowDetails, exists := updates["workflow_details"]; exists {
		if workflowDetailsMap, ok := workflowDetails.(map[string]interface{}); ok {
			detailsBytes, err := json.Marshal(workflowDetailsMap)
			if err != nil {
				p.logger.Error("序列化工作流详细参数失败", zap.Error(err))
				return fmt.Errorf("序列化工作流详细参数失败: %v", err)
			}
			updates["workflow_details"] = string(detailsBytes)
		}
	}

	// 如果状态更新为completed或failed，设置结束时间
	if status, exists := updates["status"].(string); exists {
		if status == "completed" || status == "failed" {
			updates["end_time"] = time.Now()
		}
	}

	result := database.DB.Model(&database.Scene{}).Where("id = ?", id).Updates(updates)
	return result.Error
}
