package database

import "gorm.io/gorm"

// PromptTemplate 代表一个提示词模板
type PromptTemplate struct {
	gorm.Model
	Name        string `json:"name" gorm:"not null;Unique"`              // 模板名称
	Description string `json:"description" gorm:"type:text"`             // 模板描述
	Type        string `json:"type" gorm:"not null"`                     // 模板类型 (style, scene, character等)
	Category    string `json:"category" gorm:"not null"`                 // 模板分类 (suspense, romance, action等)
	IsActive    bool   `json:"is_active" gorm:"default:true"`            // 是否激活
	IsBuiltIn   bool   `json:"is_built_in" gorm:"default:false"`         // 是否为内置模板
	Order       int    `json:"order" gorm:"column:sort_order;default:0"` // 排序权重

	// 系统提示词部分
	SystemPrompt string `json:"system_prompt" gorm:"type:text"` // 系统提示词
	UserTemplate string `json:"user_template" gorm:"type:text"` // 用户提示词模板

	// 风格相关的附加内容
	StyleAddon     string `json:"style_addon" gorm:"type:text"`     // 风格附加描述
	NegativePrompt string `json:"negative_prompt" gorm:"type:text"` // 负面提示词
	BackgroundText string `json:"background_text" gorm:"type:text"` // 故事背景
}

// CreatePromptTemplatesTable 创建提示词模板表
func CreatePromptTemplatesTable(db *gorm.DB) error {
	return db.AutoMigrate(&PromptTemplate{})
}

// GetActivePromptTemplates 获取所有激活的提示词模板
func GetActivePromptTemplates(db *gorm.DB) ([]PromptTemplate, error) {
	var templates []PromptTemplate
	err := db.Where("is_active = ?", true).Order("sort_order ASC, created_at ASC").Find(&templates).Error
	return templates, err
}

// GetPromptTemplateByID 根据ID获取提示词模板
func GetPromptTemplateByID(db *gorm.DB, id uint) (*PromptTemplate, error) {
	var template PromptTemplate
	err := db.First(&template, id).Error
	return &template, err
}

// GetPromptTemplateByName 根据名称获取提示词模板
func GetPromptTemplateByName(db *gorm.DB, name string) (*PromptTemplate, error) {
	var template PromptTemplate
	err := db.Where("name = ?", name).First(&template).Error
	return &template, err
}

// CreatePromptTemplate 创建提示词模板
func CreatePromptTemplate(db *gorm.DB, template *PromptTemplate) error {
	return db.Create(template).Error
}

// UpdatePromptTemplate 更新提示词模板
func UpdatePromptTemplate(db *gorm.DB, template *PromptTemplate) error {
	return db.Save(template).Error
}

// DeletePromptTemplate 删除提示词模板
func DeletePromptTemplate(db *gorm.DB, id uint) error {
	return db.Delete(&PromptTemplate{}, id).Error
}

// GetPromptTemplatesByCategory 根据分类获取提示词模板
func GetPromptTemplatesByCategory(db *gorm.DB, category string) ([]PromptTemplate, error) {
	var templates []PromptTemplate
	err := db.Where("category = ? AND is_active = ?", category, true).Order("sort_order ASC").Find(&templates).Error
	return templates, err
}
