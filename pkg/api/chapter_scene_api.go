package api

import (
	"net/http"
	"strconv"

	"novel-video-workflow/pkg/database"

	"github.com/gin-gonic/gin"
)

// ChapterSceneAPI 处理章节和场景相关的API
type ChapterSceneAPI struct{}

// NewChapterSceneAPI 创建新的章节场景API实例
func NewChapterSceneAPI() *ChapterSceneAPI {
	return &ChapterSceneAPI{}
}

// GetChapters 获取所有章节
func (api *ChapterSceneAPI) GetChapters(c *gin.Context) {
	chapters, err := database.GetAllChapters()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取章节列表失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"chapters": chapters,
	})
}

// GetChapterByID 根据ID获取章节
func (api *ChapterSceneAPI) GetChapterByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的章节ID"})
		return
	}

	chapter, err := database.GetChapterByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "章节不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"chapter": chapter,
	})
}

// UpdateChapter 更新章节
func (api *ChapterSceneAPI) UpdateChapter(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的章节ID"})
		return
	}

	var req struct {
		Title            string `json:"title"`
		Content          string `json:"content"`
		SegmentationPrompt string `json:"segmentation_prompt"`
		WorkflowParams   string `json:"workflow_params"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if req.Title != "" {
		updates["title"] = req.Title
	}
	if req.Content != "" {
		updates["content"] = req.Content
	}
	if req.SegmentationPrompt != "" {
		updates["prompt"] = req.SegmentationPrompt
	}
	if req.WorkflowParams != "" {
		updates["workflow_params"] = req.WorkflowParams
	}

	err = database.UpdateChapter(uint(id), updates)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新章节失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "章节更新成功",
	})
}

// GetScenesByChapterID 根据章节ID获取所有场景
func (api *ChapterSceneAPI) GetScenesByChapterID(c *gin.Context) {
	chapterID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的章节ID"})
		return
	}

	scenes, err := database.GetScenesByChapterID(uint(chapterID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取场景列表失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"scenes": scenes,
	})
}

// GetSceneByID 根据ID获取场景
func (api *ChapterSceneAPI) GetSceneByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的场景ID"})
		return
	}

	scene, err := database.GetSceneByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "场景不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"scene": scene,
	})
}

// UpdateScene 更新场景
func (api *ChapterSceneAPI) UpdateScene(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的场景ID"})
		return
	}

	var req struct {
		Title            string `json:"title"`
		Description      string `json:"description"`
		OllamaRequest    string `json:"ollama_request"`
		OllamaResponse   string `json:"ollama_response"`
		DrawThingsConfig string `json:"draw_things_config"`
		ImagePath        string `json:"image_path"`
		Status           string `json:"status"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if req.Title != "" {
		updates["title"] = req.Title
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.OllamaRequest != "" {
		updates["ollama_request"] = req.OllamaRequest
	}
	if req.OllamaResponse != "" {
		updates["ollama_response"] = req.OllamaResponse
	}
	if req.DrawThingsConfig != "" {
		updates["draw_things_config"] = req.DrawThingsConfig
	}
	if req.ImagePath != "" {
		updates["image_url"] = req.ImagePath
	}
	if req.Status != "" {
		updates["status"] = req.Status
	}

	err = database.UpdateScene(uint(id), updates)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新场景失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "场景更新成功",
	})
}

// RetryChapter 重试章节工作流
func (api *ChapterSceneAPI) RetryChapter(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的章节ID"})
		return
	}

	// 重试章节工作流的逻辑
	err = database.ResetChapterStatus(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "重试章节失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "章节重试已启动",
	})
}

// RetryScene 重试场景
func (api *ChapterSceneAPI) RetryScene(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的场景ID"})
		return
	}

	// 更新场景状态为待处理，增加重试次数
	scene, err := database.GetSceneByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "场景不存在"})
		return
	}

	updates := map[string]interface{}{
		"status":      "pending",
		"retry_count": scene.RetryCount + 1,
	}

	err = database.UpdateScene(uint(id), updates)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "重试场景失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "场景重试已启动",
	})
}