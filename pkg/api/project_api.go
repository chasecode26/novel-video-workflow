package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"novel-video-workflow/pkg/workflow"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type ProjectAPI struct {
	processor *workflow.Processor
}

func NewProjectAPI(processor *workflow.Processor) *ProjectAPI {
	return &ProjectAPI{
		processor: processor,
	}
}

// CreateProject godoc
// @Summary 创建新项目
// @Description 创建一个新的小说项目
// @Tags 项目
// @Accept json
// @Produce json
// @Param project body map[string]interface{} true "项目信息"
// @Success 200 {object} database.Project
// @Router /projects [post]
func (api *ProjectAPI) CreateProject(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		GlobalPrompt string `json:"global_prompt"`
		Password    string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	project, err := api.processor.CreateProject(req.Name, req.Description, req.GlobalPrompt, req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, project)
}

// GetProjectByID godoc
// @Summary 获取项目详情
// @Description 根据ID获取项目详情
// @Tags 项目
// @Accept json
// @Produce json
// @Param id path int true "项目ID"
// @Success 200 {object} database.Project
// @Router /projects/{id} [get]
func (api *ProjectAPI) GetProjectByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的项目ID"})
		return
	}

	project, err := api.processor.GetProjectByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "项目不存在"})
		return
	}

	c.JSON(http.StatusOK, project)
}

// GetProjectByName godoc
// @Summary 获取项目详情
// @Description 根据名称获取项目详情
// @Tags 项目
// @Accept json
// @Produce json
// @Param name path string true "项目名称"
// @Success 200 {object} database.Project
// @Router /projects/name/{name} [get]
func (api *ProjectAPI) GetProjectByName(c *gin.Context) {
	name := c.Param("name")

	project, err := api.processor.GetProjectByName(name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "项目不存在"})
		return
	}

	c.JSON(http.StatusOK, project)
}

// UpdateProject godoc
// @Summary 更新项目
// @Description 更新项目信息
// @Tags 项目
// @Accept json
// @Produce json
// @Param id path int true "项目ID"
// @Param project body map[string]interface{} true "项目更新信息"
// @Success 200 {object} database.Project
// @Router /projects/{id} [put]
func (api *ProjectAPI) UpdateProject(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的项目ID"})
		return
	}

	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 如果更新了密码，需要重新哈希
	if newPassword, exists := req["password"]; exists {
		passwordStr, ok := newPassword.(string)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "密码必须是字符串"})
			return
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(passwordStr), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "密码哈希失败"})
			return
		}

		req["password_hash"] = string(hashedPassword)
		delete(req, "password") // 删除明文密码
	}

	err = api.processor.UpdateProject(uint(id), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 返回更新后的项目
	project, err := api.processor.GetProjectByID(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, project)
}

// DeleteProject godoc
// @Summary 删除项目
// @Description 删除指定ID的项目
// @Tags 项目
// @Accept json
// @Produce json
// @Param id path int true "项目ID"
// @Success 200 {object} map[string]string
// @Router /projects/{id} [delete]
func (api *ProjectAPI) DeleteProject(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的项目ID"})
		return
	}

	err = api.processor.DeleteProject(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "项目删除成功"})
}

// CreateChapter godoc
// @Summary 创建章节
// @Description 为指定项目创建新章节
// @Tags 章节
// @Accept json
// @Produce json
// @Param project_id path int true "项目ID"
// @Param chapter body map[string]interface{} true "章节信息"
// @Success 200 {object} database.Chapter
// @Router /projects/{project_id}/chapters [post]
func (api *ProjectAPI) CreateChapter(c *gin.Context) {
	projectID, err := strconv.ParseUint(c.Param("projectId"), 10, 32)  // 更新参数名与路由匹配
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的项目ID"})
		return
	}

	var req struct {
		Title   string `json:"title" binding:"required"`
		Content string `json:"content"`
		Prompt  string `json:"prompt"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	chapter, err := api.processor.CreateChapter(uint(projectID), req.Title, req.Content, req.Prompt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, chapter)
}

// GetChapterByID godoc
// @Summary 获取章节详情
// @Description 根据ID获取章节详情
// @Tags 章节
// @Accept json
// @Produce json
// @Param id path int true "章节ID"
// @Success 200 {object} database.Chapter
// @Router /chapters/{id} [get]
func (api *ProjectAPI) GetChapterByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("chapterId"), 10, 32)  // 更新参数名与路由匹配
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的章节ID"})
		return
	}

	chapter, err := api.processor.GetChapterByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "章节不存在"})
		return
	}

	c.JSON(http.StatusOK, chapter)
}

// GetChaptersByProjectID godoc
// @Summary 获取项目下的所有章节
// @Description 获取指定项目下的所有章节
// @Tags 章节
// @Accept json
// @Produce json
// @Param project_id path int true "项目ID"
// @Success 200 {array} database.Chapter
// @Router /projects/{project_id}/chapters [get]
func (api *ProjectAPI) GetChaptersByProjectID(c *gin.Context) {
	projectID, err := strconv.ParseUint(c.Param("projectId"), 10, 32)  // 更新参数名与路由匹配
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的项目ID"})
		return
	}

	chapters, err := api.processor.GetChaptersByProjectID(uint(projectID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, chapters)
}

// UpdateChapter godoc
// @Summary 更新章节
// @Description 更新章节信息
// @Tags 章节
// @Accept json
// @Produce json
// @Param id path int true "章节ID"
// @Param chapter body map[string]interface{} true "章节更新信息"
// @Success 200 {object} database.Chapter
// @Router /chapters/{id} [put]
func (api *ProjectAPI) UpdateChapter(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("chapterId"), 10, 32)  // 更新参数名与路由匹配
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的章节ID"})
		return
	}

	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 如果更新了图片路径，需要确保它是有效的JSON数组
	if imagePaths, exists := req["image_paths"]; exists {
		pathsStr, ok := imagePaths.(string)
		if ok {
			var paths []string
			if err := json.Unmarshal([]byte(pathsStr), &paths); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "图片路径不是有效的JSON数组"})
				return
			}
		}
	}

	err = api.processor.UpdateChapter(uint(id), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 返回更新后的章节
	chapter, err := api.processor.GetChapterByID(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, chapter)
}

// DeleteChapter godoc
// @Summary 删除章节
// @Description 删除指定ID的章节
// @Tags 章节
// @Accept json
// @Produce json
// @Param id path int true "章节ID"
// @Success 200 {object} map[string]string
// @Router /chapters/{id} [delete]
func (api *ProjectAPI) DeleteChapter(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("chapterId"), 10, 32)  // 更新参数名与路由匹配
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的章节ID"})
		return
	}

	err = api.processor.DeleteChapter(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "章节删除成功"})
}

// CreateScene godoc
// @Summary 创建场景
// @Description 为指定章节创建新场景
// @Tags 场景
// @Accept json
// @Produce json
// @Param chapter_id path int true "章节ID"
// @Param scene body map[string]interface{} true "场景信息"
// @Success 200 {object} database.Scene
// @Router /chapters/{chapter_id}/scenes [post]
func (api *ProjectAPI) CreateScene(c *gin.Context) {
	chapterID, err := strconv.ParseUint(c.Param("chapterId"), 10, 32)  // 更新参数名与路由匹配
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的章节ID"})
		return
	}

	var req struct {
		Title            string `json:"title" binding:"required"`
		Description      string `json:"description"`
		Prompt           string `json:"prompt"`
		OllamaRequest    string `json:"ollama_request"`
		OllamaResponse   string `json:"ollama_response"`
		DrawThingsConfig string `json:"draw_things_config"`
		DrawThingsResult string `json:"draw_things_result"`
		Order            int    `json:"order"`
		ImageURL         string `json:"image_url"`
		AudioURL         string `json:"audio_url"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	scene, err := api.processor.CreateScene(uint(chapterID), req.Title, req.Description, req.Prompt, req.OllamaRequest, req.OllamaResponse, req.DrawThingsConfig, req.DrawThingsResult, req.Order)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 如果提供了图片或音频URL，更新场景
	updates := make(map[string]interface{})
	if req.ImageURL != "" {
		updates["image_url"] = req.ImageURL
	}
	if req.AudioURL != "" {
		updates["audio_url"] = req.AudioURL
	}

	if len(updates) > 0 {
		api.processor.UpdateScene(scene.ID, updates)
		// 重新获取场景以返回最新数据
		scene, _ = api.processor.GetSceneByID(scene.ID)
	}

	c.JSON(http.StatusOK, scene)
}

// GetSceneByID godoc
// @Summary 获取场景详情
// @Description 根据ID获取场景详情
// @Tags 场景
// @Accept json
// @Produce json
// @Param id path int true "场景ID"
// @Success 200 {object} database.Scene
// @Router /scenes/{id} [get]
func (api *ProjectAPI) GetSceneByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("sceneId"), 10, 32)  // 更新参数名与路由匹配
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的场景ID"})
		return
	}

	scene, err := api.processor.GetSceneByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "场景不存在"})
		return
	}

	c.JSON(http.StatusOK, scene)
}

// GetScenesByChapterID godoc
// @Summary 获取章节下的所有场景
// @Description 获取指定章节下的所有场景
// @Tags 场景
// @Accept json
// @Produce json
// @Param chapter_id path int true "章节ID"
// @Success 200 {array} database.Scene
// @Router /chapters/{chapter_id}/scenes [get]
func (api *ProjectAPI) GetScenesByChapterID(c *gin.Context) {
	chapterID, err := strconv.ParseUint(c.Param("chapterId"), 10, 32)  // 更新参数名与路由匹配
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的章节ID"})
		return
	}

	scenes, err := api.processor.GetScenesByChapterID(uint(chapterID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, scenes)
}

// UpdateScene godoc
// @Summary 更新场景
// @Description 更新场景信息，允许用户修改提示词和DrawThings配置
// @Tags 场景
// @Accept json
// @Produce json
// @Param id path int true "场景ID"
// @Param scene body map[string]interface{} true "场景更新信息"
// @Success 200 {object} database.Scene
// @Router /scenes/{id} [put]
func (api *ProjectAPI) UpdateScene(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("sceneId"), 10, 32)  // 更新参数名与路由匹配
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的场景ID"})
		return
	}

	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err = api.processor.UpdateScene(uint(id), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 返回更新后的场景
	scene, err := api.processor.GetSceneByID(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, scene)
}

// DeleteScene godoc
// @Summary 删除场景
// @Description 删除指定ID的场景
// @Tags 场景
// @Accept json
// @Produce json
// @Param id path int true "场景ID"
// @Success 200 {object} map[string]string
// @Router /scenes/{id} [delete]
func (api *ProjectAPI) DeleteScene(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("sceneId"), 10, 32)  // 更新参数名与路由匹配
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的场景ID"})
		return
	}

	err = api.processor.DeleteScene(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "场景删除成功"})
}

// ShareChapter godoc
// @Summary 分享章节
// @Description 为章节生成分享链接，需要密码保护
// @Tags 分享
// @Accept json
// @Produce json
// @Param id path int true "章节ID"
// @Param share_info body map[string]interface{} true "分享信息"
// @Success 200 {object} map[string]interface{}
// @Router /chapters/{id}/share [post]
func (api *ProjectAPI) ShareChapter(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("chapterId"), 10, 32)  // 更新参数名与路由匹配
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的章节ID"})
		return
	}

	var req struct {
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 生成分享令牌
	shareToken := uuid.New().String()

	err = api.processor.SetChapterAsShared(uint(id), shareToken, req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "章节分享成功",
		"share_token": shareToken,
		"share_link":  fmt.Sprintf("%s/chapters/share/%s", c.Request.Host, shareToken),
	})
}

// UnshareChapter godoc
// @Summary 取消分享章节
// @Description 取消章节的分享状态
// @Tags 分享
// @Accept json
// @Produce json
// @Param id path int true "章节ID"
// @Success 200 {object} map[string]string
// @Router /chapters/{id}/unshare [post]
func (api *ProjectAPI) UnshareChapter(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("chapterId"), 10, 32)  // 更新参数名与路由匹配
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的章节ID"})
		return
	}

	err = api.processor.RevokeChapterShare(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "章节分享已取消"})
}

// GetSharedChapter godoc
// @Summary 获取分享的章节
// @Description 根据分享令牌获取章节信息，需要提供密码
// @Tags 分享
// @Accept json
// @Produce json
// @Param token path string true "分享令牌"
// @Param password body map[string]string true "密码"
// @Success 200 {object} database.Chapter
// @Router /chapters/share/{token} [post]
func (api *ProjectAPI) GetSharedChapter(c *gin.Context) {
	token := c.Param("token")

	var req struct {
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 验证密码
	valid, err := api.processor.ValidateChapterSharePassword(token, req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "验证失败"})
		return
	}

	if !valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误"})
		return
	}

	// 获取章节信息
	chapter, err := api.processor.GetChapterByShareToken(token)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "章节不存在或未分享"})
		return
	}

	c.JSON(http.StatusOK, chapter)
}

// GetPublicSharedChapter godoc
// @Summary 获取分享的章节（公开）
// @Description 根据分享令牌获取章节信息（仅返回公共可见字段）
// @Tags 分享
// @Accept json
// @Produce json
// @Param token path string true "分享令牌"
// @Success 200 {object} map[string]interface{}
// @Router /chapters/share/public/{token} [get]
func (api *ProjectAPI) GetPublicSharedChapter(c *gin.Context) {
	token := c.Param("token")

	chapter, err := api.processor.GetChapterByShareToken(token)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "章节不存在或未分享"})
		return
	}

	// 返回公共信息
	publicChapter := gin.H{
		"id":          chapter.ID,
		"title":       chapter.Title,
		"content":     chapter.Content,
		"prompt":      chapter.Prompt,
		"created_at":  chapter.CreatedAt,
		"updated_at":  chapter.UpdatedAt,
		"scenes":      chapter.Scenes,
		"image_paths": chapter.ImagePaths,
		"audio_url":   chapter.AudioURL,
	}

	c.JSON(http.StatusOK, publicChapter)
}

// ValidateProjectPassword godoc
// @Summary 验证项目密码
// @Description 验证项目访问密码
// @Tags 项目
// @Accept json
// @Produce json
// @Param id path int true "项目ID"
// @Param password body map[string]string true "密码"
// @Success 200 {object} map[string]bool
// @Router /projects/{id}/validate-password [post]
func (api *ProjectAPI) ValidateProjectPassword(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("projectId"), 10, 32)  // 更新参数名与路由匹配
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的项目ID"})
		return
	}

	var req struct {
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	valid, err := api.processor.ValidateProjectPassword(uint(id), req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "验证失败"})
		return
	}

	if valid {
		c.JSON(http.StatusOK, gin.H{"valid": true})
	} else {
		c.JSON(http.StatusUnauthorized, gin.H{"valid": false})
	}
}

// GetAllProjects godoc
// @Summary 获取所有项目
// @Description 获取所有项目列表
// @Tags 项目
// @Accept json
// @Produce json
// @Success 200 {array} database.Project
// @Router /projects [get]
func (api *ProjectAPI) GetAllProjects(c *gin.Context) {
	projects, err := api.processor.GetAllProjects()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, projects)
}
