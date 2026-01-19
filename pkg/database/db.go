package database

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// InitDB 初始化数据库连接
func InitDB(dbPath string) error {
	// 确保数据库目录存在
	dbDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return fmt.Errorf("创建数据库目录失败: %v", err)
	}

	// 打开数据库连接
	newDB, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent), // 设置为静默模式，不输出日志
	})
	if err != nil {
		return fmt.Errorf("连接数据库失败: %v", err)
	}

	// 迁移数据库结构
	err = migrateDB(newDB)
	if err != nil {
		return fmt.Errorf("数据库迁移失败: %v", err)
	}

	DB = newDB
	return nil
}

// migrateDB 执行数据库迁移
func migrateDB(db *gorm.DB) error {
	// 自动迁移表结构
	err := db.AutoMigrate(&Project{}, &Chapter{}, &Scene{})
	if err != nil {
		return fmt.Errorf("自动迁移表结构失败: %v", err)
	}

	return nil
}

// HashPassword 对密码进行哈希处理
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPasswordHash 检查密码是否匹配
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// CreateProject 创建新项目
func CreateProject(name, description, globalPrompt, password string) (*Project, error) {
	hashedPassword, err := HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("密码哈希失败: %v", err)
	}

	project := &Project{
		Name:         name,
		Description:  description,
		GlobalPrompt: globalPrompt,
		PasswordHash: hashedPassword,
	}

	result := DB.Create(project)
	if result.Error != nil {
		return nil, result.Error
	}

	return project, nil
}

// GetProjectByID 根据ID获取项目
func GetProjectByID(id uint) (*Project, error) {
	var project Project
	result := DB.Preload("Chapters.Scenes").First(&project, id)
	if result.Error != nil {
		return nil, result.Error
	}
	return &project, nil
}

// GetProjectByName 根据名称获取项目
func GetProjectByName(name string) (*Project, error) {
	var project Project
	result := DB.Preload("Chapters.Scenes").Where("name = ?", name).First(&project)
	if result.Error != nil {
		return nil, result.Error
	}
	return &project, nil
}

// UpdateProject 更新项目
func UpdateProject(id uint, updates map[string]interface{}) error {
	result := DB.Model(&Project{}).Where("id = ?", id).Updates(updates)
	return result.Error
}

// DeleteProject 删除项目
func DeleteProject(id uint) error {
	result := DB.Delete(&Project{}, id)
	return result.Error
}

// CreateChapter 创建章节
func CreateChapter(projectID uint, title, content, prompt string) (*Chapter, error) {
	chapter := &Chapter{
		Title:       title,
		Content:     content,
		Prompt:      prompt,
		ProjectID:   projectID,
		ImagePaths:  "[]", // 默认空数组
	}

	result := DB.Create(chapter)
	if result.Error != nil {
		return nil, result.Error
	}

	return chapter, nil
}

// GetChapterByID 根据ID获取章节
func GetChapterByID(id uint) (*Chapter, error) {
	var chapter Chapter
	result := DB.Preload("Scenes").First(&chapter, id)
	if result.Error != nil {
		return nil, result.Error
	}
	return &chapter, nil
}

// GetChaptersByProjectID 获取项目下的所有章节
func GetChaptersByProjectID(projectID uint) ([]Chapter, error) {
	var chapters []Chapter
	result := DB.Where("project_id = ?", projectID).Preload("Scenes").Find(&chapters)
	if result.Error != nil {
		return nil, result.Error
	}
	return chapters, nil
}

// UpdateChapter 更新章节
func UpdateChapter(id uint, updates map[string]interface{}) error {
	// 如果更新了图片路径，需要确保它是有效的JSON数组
	if imagePaths, exists := updates["image_paths"]; exists {
		// 验证图片路径是有效的JSON数组
		pathsStr, ok := imagePaths.(string)
		if ok {
			var paths []string
			if err := json.Unmarshal([]byte(pathsStr), &paths); err != nil {
				return fmt.Errorf("图片路径不是有效的JSON数组: %v", err)
			}
		}
	}

	result := DB.Model(&Chapter{}).Where("id = ?", id).Updates(updates)
	return result.Error
}

// DeleteChapter 删除章节
func DeleteChapter(id uint) error {
	// 先删除相关的场景
	if err := DB.Where("chapter_id = ?", id).Delete(&Scene{}).Error; err != nil {
		return err
	}

	// 再删除章节
	result := DB.Delete(&Chapter{}, id)
	return result.Error
}

// CreateScene 创建场景
func CreateScene(chapterID uint, title, description, prompt, ollamaRequest, ollamaResponse, drawThingsConfig, drawThingsResult string, order int) (*Scene, error) {
	scene := &Scene{
		Title:            title,
		Description:      description,
		Prompt:           prompt,
		OllamaRequest:    ollamaRequest,
		OllamaResponse:   ollamaResponse,
		DrawThingsConfig: drawThingsConfig,
		DrawThingsResult: drawThingsResult,
		ChapterID:        chapterID,
		Order:            order,
	}

	result := DB.Create(scene)
	if result.Error != nil {
		return nil, result.Error
	}

	return scene, nil
}

// GetSceneByID 根据ID获取场景
func GetSceneByID(id uint) (*Scene, error) {
	var scene Scene
	result := DB.First(&scene, id)
	if result.Error != nil {
		return nil, result.Error
	}
	return &scene, nil
}

// GetScenesByChapterID 获取章节下的所有场景
func GetScenesByChapterID(chapterID uint) ([]Scene, error) {
	var scenes []Scene
	result := DB.Where("chapter_id = ?", chapterID).Order("order ASC").Find(&scenes)
	if result.Error != nil {
		return nil, result.Error
	}
	return scenes, nil
}

// UpdateScene 更新场景
func UpdateScene(id uint, updates map[string]interface{}) error {
	result := DB.Model(&Scene{}).Where("id = ?", id).Updates(updates)
	return result.Error
}

// DeleteScene 删除场景
func DeleteScene(id uint) error {
	result := DB.Delete(&Scene{}, id)
	return result.Error
}

// ValidateProjectPassword 验证项目密码
func ValidateProjectPassword(projectID uint, password string) (bool, error) {
	var project Project
	result := DB.Select("password_hash").First(&project, projectID)
	if result.Error != nil {
		return false, result.Error
	}

	return CheckPasswordHash(password, project.PasswordHash), nil
}

// ValidateChapterSharePassword 验证章节分享密码
func ValidateChapterSharePassword(shareToken, password string) (bool, error) {
	var chapter Chapter
	result := DB.Select("share_password").Where("share_token = ? AND is_shared = ?", shareToken, true).First(&chapter)
	if result.Error != nil {
		return false, result.Error
	}

	return CheckPasswordHash(password, chapter.SharePassword), nil
}

// GetChapterByShareToken 根据分享令牌获取章节（公开信息）
func GetChapterByShareToken(shareToken string) (*Chapter, error) {
	var chapter Chapter
	result := DB.Preload("Scenes").Where("share_token = ? AND is_shared = ?", shareToken, true).First(&chapter)
	if result.Error != nil {
		return nil, result.Error
	}
	return &chapter, nil
}

// SetChapterAsShared 设置章节为已分享状态
func SetChapterAsShared(chapterID uint, shareToken, password string) error {
	hashedPassword, err := HashPassword(password)
	if err != nil {
		return fmt.Errorf("密码哈希失败: %v", err)
	}

	updates := map[string]interface{}{
		"share_token":   shareToken,
		"share_password": hashedPassword,
		"is_shared":     true,
	}

	result := DB.Model(&Chapter{}).Where("id = ?", chapterID).Updates(updates)
	return result.Error
}

// RevokeChapterShare 取消章节分享
func RevokeChapterShare(chapterID uint) error {
	updates := map[string]interface{}{
		"share_token":    "",
		"share_password": "",
		"is_shared":      false,
	}

	result := DB.Model(&Chapter{}).Where("id = ?", chapterID).Updates(updates)
	return result.Error
}

// GetAllProjects 获取所有项目
func GetAllProjects() ([]Project, error) {
	var projects []Project
	result := DB.Preload("Chapters.Scenes").Find(&projects)
	if result.Error != nil {
		return nil, result.Error
	}
	return projects, nil
}
