package database

import (
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
	Title         string  `json:"title"`
	Content       string  `json:"content"`
	Prompt        string  `json:"prompt"`      // 章节氛围提示词
	AudioURL      string  `json:"audio_url"`   // 章节音频URL或路径
	ImagePaths    string  `json:"image_paths"` // 章节图片路径数组（JSON格式）
	ProjectID     uint    `json:"project_id" gorm:"not null"`
	Scenes        []Scene `json:"scenes" gorm:"foreignKey:ChapterID"`
	ShareToken    string  `json:"share_token"` // 章节分享令牌
	SharePassword string  `json:"-"`           // 章节分享密码哈希
	IsShared      bool    `json:"is_shared"`   // 是否已分享
}

// Scene 场景模型，属于一个章节
type Scene struct {
	gorm.Model
	Title            string `json:"title"`
	Description      string `json:"description"`        // 场景文本
	Prompt           string `json:"prompt"`             // 场景提示词
	OllamaRequest    string `json:"ollama_request"`     // 发送给Ollama的请求JSON
	OllamaResponse   string `json:"ollama_response"`    // Ollama返回的结果JSON
	DrawThingsConfig string `json:"draw_things_config"` // 发送给DrawThings的配置JSON
	DrawThingsResult string `json:"draw_things_result"` // DrawThings返回的结果JSON
	ChapterID        uint   `json:"chapter_id" gorm:"not null"`
	ImageURL         string `json:"image_url"` // 生成的图像URL
	AudioURL         string `json:"audio_url"` // 生成的音频URL
	Order            int    `json:"order"`     // 场景顺序
}
