package database

import (
	"time"

	"gorm.io/gorm"
)

// Project 项目模型，代表一部小说
type Project struct {
	gorm.Model
	Name         string    `json:"name" gorm:"not null;uniqueIndex"`
	Description  string    `json:"description"`
	GlobalPrompt string    `json:"global_prompt"` // 整体氛围提示词
	PasswordHash string    `json:"-"`             // 项目级别的密码哈希
	Chapters     []Chapter `json:"chapters" gorm:"foreignKey:ProjectID"`
}

// Chapter 章节模型，属于一个项目
type Chapter struct {
	gorm.Model
	Title          string  `json:"title"`
	Content        string  `json:"content"`
	Prompt         string  `json:"prompt"`      // 章节氛围提示词
	AudioURL       string  `json:"audio_url"`   // 章节音频URL或路径
	ImagePaths     string  `json:"image_paths"` // 章节图片路径数组（JSON格式）
	ProjectID      uint    `json:"project_id" gorm:"not null"`
	Scenes         []Scene `json:"scenes" gorm:"foreignKey:ChapterID"`
	WorkflowParams string  `json:"workflow_params" gorm:"type:text"` // 工作流参数
}

// Scene 场景模型，属于一个章节
type Scene struct {
	gorm.Model
	Title            string `json:"title"`
	Description      string `json:"description"`        // 场景文本
	Prompt           string `json:"prompt"`             // 场景提示词
	SegmentationInfo string `json:"segmentation_info"`  // 智能分镜信息
	OriginalText     string `json:"original_text"`      // 发送给Ollama的原始描述文本
	OllamaRequest    string `json:"ollama_request"`     // 发送给Ollama的请求JSON
	OllamaResponse   string `json:"ollama_response"`    // Ollama返回的结果JSON
	DrawThingsConfig string `json:"draw_things_config"` // 发送给DrawThings的配置JSON
	DrawThingsResult string `json:"draw_things_result"` // DrawThings返回的结果JSON
	ChapterID        uint   `json:"chapter_id"`
	ImageURL         string `json:"image_url"`               // 生成的图像URL
	AudioURL         string `json:"audio_url"`               // 生成的音频URL
	RetryCount       int    `json:"retry_count"`             // 重试次数
	Sort             int    `json:"sort" gorm:"column:sort"` // 场景顺序，映射到sort列
	// 新增字段用于记录工作流详细参数
	WorkflowDetails string    `json:"workflow_details" gorm:"type:text"` // 工作流详细参数，JSON格式
	Status          string    `json:"status" gorm:"default:'pending'"`   // 场景状态: pending, processing, completed, failed
	StartTime       time.Time `json:"start_time"`                        // 开始处理时间
	EndTime         time.Time `json:"end_time"`                          // 结束处理时间
}
