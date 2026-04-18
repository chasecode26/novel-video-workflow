package web_server

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"novel-video-workflow/pkg/broadcast"
	configpkg "novel-video-workflow/pkg/config"
	"novel-video-workflow/pkg/database"
	"novel-video-workflow/pkg/healthcheck"
	mcp_pkg "novel-video-workflow/pkg/mcp"
	"novel-video-workflow/pkg/providers"
	"novel-video-workflow/pkg/tools/aegisub"
	"novel-video-workflow/pkg/tools/comfyui"
	"novel-video-workflow/pkg/tools/drawthings"
	"novel-video-workflow/pkg/tools/file"
	"novel-video-workflow/pkg/tools/indextts2"
	workflow_pkg "novel-video-workflow/pkg/workflow"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	api_pkg "novel-video-workflow/pkg/api"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v3"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// 存储WebSocket连接
var clients = make(map[*websocket.Conn]bool)

// ToolInfo 结构存储MCP工具的信息
type ToolInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Path        string `json:"path"`
}

// 存储所有可用的MCP工具
var mcpTools []ToolInfo
var mcpServerInstance *mcp_pkg.Server

func loadAppComponents(configPath string) (configpkg.Config, providers.ProviderBundle, *workflow_pkg.Processor, error) {
	cfg, err := configpkg.LoadConfig(configPath)
	if err != nil {
		return configpkg.Config{}, providers.ProviderBundle{}, nil, fmt.Errorf("加载配置文件失败: %v", err)
	}

	bundle, err := providers.BuildProviders(cfg)
	if err != nil {
		return configpkg.Config{}, providers.ProviderBundle{}, nil, fmt.Errorf("构建 providers 失败: %v", err)
	}

	report := healthcheck.NewService(bundle).Run()
	if !report.CanStart {
		return configpkg.Config{}, providers.ProviderBundle{}, nil, fmt.Errorf("启动前健康检查失败: %d blocking issues", len(report.Blocking))
	}

	logger := zap.NewNop()
	processor, err := workflow_pkg.NewProcessor(cfg, bundle, logger)
	if err != nil {
		return configpkg.Config{}, providers.ProviderBundle{}, nil, fmt.Errorf("创建工作流处理器失败: %v", err)
	}

	return cfg, bundle, processor, nil
}

func buildWorkflowAPIs(cfg configpkg.Config, bundle providers.ProviderBundle) (*api_pkg.SystemCheckAPI, *api_pkg.WorkflowRunAPI) {
	if err := database.InitDB(cfg.Database.Path); err != nil {
		log.Printf("初始化 workflow run 数据库失败: %v", err)
		return nil, nil
	}

	runStorage := workflow_pkg.NewRunStorage(database.DB)
	runExecutor := workflow_pkg.NewExecutor(bundle, runStorage, cfg.Paths.BaseDir, cfg.Paths.ReferenceAudio)
	runExecutor.SetEventPublisher(&api_pkg.BroadcastEventPublisher{})
	return api_pkg.NewSystemCheckAPI(bundle), api_pkg.NewWorkflowRunAPI(runExecutor, runStorage)
}

func registerWorkflowRoutes(r *gin.Engine, cfg configpkg.Config, bundle providers.ProviderBundle) bool {
	systemCheckAPI, workflowRunAPI := buildWorkflowAPIs(cfg, bundle)
	if systemCheckAPI == nil || workflowRunAPI == nil {
		registerUnavailableWorkflowRoutes(r)
		return false
	}

	systemCheckAPI.RegisterRoutes(r)
	workflowRunAPI.RegisterRoutes(r)
	return true
}

func registerWorkflowSupportRoutes(r *gin.Engine) {
	configPath := mustResolveConfigPath()
	if strings.TrimSpace(configPath) == "" {
		log.Printf("加载 workflow 依赖失败: 配置文件路径不可用")
		registerWorkflowSupportUnavailableRoutes(r)
		registerUnavailableWorkflowRoutes(r)
		return
	}

	cfg, bundle, processor, err := loadAppComponents(configPath)
	if err != nil {
		log.Printf("加载 workflow 依赖失败: %v", err)
		registerWorkflowSupportUnavailableRoutes(r)
		cfgOnly, cfgErr := configpkg.LoadConfig(configPath)
		if cfgErr == nil {
			if bundleOnly, bundleErr := providers.BuildProviders(cfgOnly); bundleErr == nil {
				registerWorkflowRoutes(r, cfgOnly, bundleOnly)
				return
			}
		}
		registerUnavailableWorkflowRoutes(r)
		return
	}

	registerWorkflowRoutes(r, cfg, bundle)

	promptTemplateAPI := api_pkg.NewPromptTemplateAPI(processor)
	r.GET("/api/prompt-templates", promptTemplateAPI.GetPromptTemplates)
	r.GET("/api/prompt-templates/:id", promptTemplateAPI.GetPromptTemplateByID)
	r.GET("/api/prompt-templates/category/:category", promptTemplateAPI.GetPromptTemplatesByCategory)
	r.POST("/api/prompt-templates", promptTemplateAPI.CreatePromptTemplate)
	r.PUT("/api/prompt-templates/:id", promptTemplateAPI.UpdatePromptTemplate)
	r.DELETE("/api/prompt-templates/:id", promptTemplateAPI.DeletePromptTemplate)

	workflowTrackingAPI := api_pkg.NewWorkflowTrackingAPI(processor)
	r.POST("/api/workflow/chapter/record-params", workflowTrackingAPI.RecordChapterWorkflowParams)
	r.GET("/api/workflow/chapter/:id/params", workflowTrackingAPI.GetChapterWorkflowParams)
	r.POST("/api/workflow/scene/record-params", workflowTrackingAPI.RecordSceneWorkflowParams)
	r.GET("/api/workflow/scene/:id/params", workflowTrackingAPI.GetSceneWorkflowParams)
	r.GET("/api/chapter/:id/scenes", workflowTrackingAPI.GetScenesByChapter)
}

func mustResolveConfigPath() string {
	configPath, err := resolveConfigPath()
	if err != nil {
		return ""
	}
	return configPath
}

func registerProcessorBackedRoutes(r *gin.Engine) bool {
	if mcpServerInstance == nil {
		log.Printf("MCP 服务未初始化，相关 API 以 503 降级返回")
		registerUnavailableProcessorRoutes(r)
		return false
	}

	processor := mcpServerInstance.GetProcessor()
	if processor == nil {
		log.Printf("MCP processor 不可用，相关 API 以 503 降级返回")
		registerUnavailableProcessorRoutes(r)
		return false
	}

	promptTemplateAPI := api_pkg.NewPromptTemplateAPI(processor)
	r.GET("/api/prompt-templates", promptTemplateAPI.GetPromptTemplates)
	r.GET("/api/prompt-templates/:id", promptTemplateAPI.GetPromptTemplateByID)
	r.GET("/api/prompt-templates/category/:category", promptTemplateAPI.GetPromptTemplatesByCategory)
	r.POST("/api/prompt-templates", promptTemplateAPI.CreatePromptTemplate)
	r.PUT("/api/prompt-templates/:id", promptTemplateAPI.UpdatePromptTemplate)
	r.DELETE("/api/prompt-templates/:id", promptTemplateAPI.DeletePromptTemplate)

	workflowTrackingAPI := api_pkg.NewWorkflowTrackingAPI(processor)
	r.POST("/api/workflow/chapter/record-params", workflowTrackingAPI.RecordChapterWorkflowParams)
	r.GET("/api/workflow/chapter/:id/params", workflowTrackingAPI.GetChapterWorkflowParams)
	r.POST("/api/workflow/scene/record-params", workflowTrackingAPI.RecordSceneWorkflowParams)
	r.GET("/api/workflow/scene/:id/params", workflowTrackingAPI.GetSceneWorkflowParams)
	r.GET("/api/chapter/:id/scenes", workflowTrackingAPI.GetScenesByChapter)
	return true
}

func registerUnavailableWorkflowRoutes(r *gin.Engine) {
	unavailable := func(c *gin.Context) {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "error",
			"message": "workflow services unavailable",
		})
	}

	r.GET("/api/system/check", unavailable)
	r.POST("/api/workflow/runs", unavailable)
	r.GET("/api/workflow/runs/:id", unavailable)
}

func registerWorkflowSupportUnavailableRoutes(r *gin.Engine) {
	unavailable := func(c *gin.Context) {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "error",
			"message": "MCP processor unavailable",
		})
	}

	r.GET("/api/prompt-templates", unavailable)
	r.GET("/api/prompt-templates/:id", unavailable)
	r.GET("/api/prompt-templates/category/:category", unavailable)
	r.POST("/api/prompt-templates", unavailable)
	r.PUT("/api/prompt-templates/:id", unavailable)
	r.DELETE("/api/prompt-templates/:id", unavailable)

	r.POST("/api/workflow/chapter/record-params", unavailable)
	r.GET("/api/workflow/chapter/:id/params", unavailable)
	r.POST("/api/workflow/scene/record-params", unavailable)
	r.GET("/api/workflow/scene/:id/params", unavailable)
	r.GET("/api/chapter/:id/scenes", unavailable)
}

func registerUnavailableProcessorRoutes(r *gin.Engine) {
	registerWorkflowSupportUnavailableRoutes(r)
}

func isAllowedProjectPath(cleanPath, projectRoot string) bool {
	allowedPrefixes := []string{
		filepath.Join(projectRoot, "input"),
		filepath.Join(projectRoot, "output"),
	}

	for _, prefix := range allowedPrefixes {
		if cleanPath == prefix {
			return true
		}
		if strings.HasPrefix(cleanPath, prefix+string(filepath.Separator)) {
			return true
		}
	}

	return false
}

func extractToolParams(reqBody map[string]interface{}) map[string]interface{} {
	if params, ok := reqBody["params"].(map[string]interface{}); ok {
		return params
	}

	params := make(map[string]interface{}, len(reqBody))
	for key, value := range reqBody {
		if key == "toolName" || key == "params" {
			continue
		}
		params[key] = value
	}

	return params
}

// 启动MCP服务器
func startMCPServer() error {
	// 创建logger
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	configPath, err := resolveConfigPath()
	if err != nil {
		return fmt.Errorf("解析配置文件路径失败: %v", err)
	}

	_, _, processor, err := loadAppComponents(configPath)
	if err != nil {
		return err
	}

	// 创建MCP服务器，使用当前全局样式
	server, err := mcp_pkg.NewServerWithGlobalStyle(processor, logger, GlobalStyle)
	if err != nil {
		return fmt.Errorf("创建MCP服务器失败: %v", err)
	}
	mcpServerInstance = server

	// 获取可用工具列表
	availableTools := server.GetHandler().GetToolNames()
	log.Printf("MCP服务器启动成功，加载了 %d 个工具", len(availableTools))

	// 为每个工具创建描述信息
	for _, toolName := range availableTools {
		description := getToolDescription(toolName) // 获取工具描述
		mcpTools = append(mcpTools, ToolInfo{
			Name:        toolName,
			Description: description,
			Path:        fmt.Sprintf("./mcp-tools/%s.yaml", toolName),
		})
	}

	log.Printf("Loaded %d MCP tools", len(mcpTools))

	return nil
}

func initializeMCPServer() {
	// 启动MCP服务器
	if err := startMCPServer(); err != nil {
		log.Printf("启动MCP服务器失败: %v", err)
		// 即使启动失败，也要提供默认工具列表
		fallbackToolList()
		return
	}
}

func resolveConfigPath() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	configPath := filepath.Join(wd, "config.yaml")
	if _, err := os.Stat(configPath); err == nil {
		return configPath, nil
	}

	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	configPath = filepath.Join(filepath.Dir(exe), "config.yaml")
	if _, err := os.Stat(configPath); err != nil {
		return "", err
	}
	return configPath, nil
}

func loadWebConfig() (configpkg.Config, error) {
	configPath, err := resolveConfigPath()
	if err != nil {
		return configpkg.Config{}, err
	}
	return configpkg.LoadConfig(configPath)
}

func ensureConfigMap(root map[string]interface{}, key string) map[string]interface{} {
	if existing, ok := root[key].(map[string]interface{}); ok {
		return existing
	}
	child := map[string]interface{}{}
	root[key] = child
	return child
}

func updateConfigFile(mutator func(map[string]interface{}) error) error {
	configPath, err := resolveConfigPath()
	if err != nil {
		return err
	}

	raw, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	cfgMap := map[string]interface{}{}
	if len(raw) > 0 {
		if err := yaml.Unmarshal(raw, &cfgMap); err != nil {
			return err
		}
	}

	if err := mutator(cfgMap); err != nil {
		return err
	}

	out, err := yaml.Marshal(cfgMap)
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, out, 0o644)
}

func intFromSetting(value interface{}) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int32:
		return int(v), true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	case json.Number:
		n, err := v.Int64()
		return int(n), err == nil
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(v))
		return n, err == nil
	default:
		return 0, false
	}
}

// fallbackToolList 提供备用工具列表
func fallbackToolList() {
	descriptions := map[string]string{
		"generate_indextts2_audio":                    "使用IndexTTS2生成音频文件，具有高级语音克隆功能",
		"generate_subtitles_from_indextts2":           "使用Aegisub从IndexTTS2音频和提供的文本生成字幕(SRT)",
		"file_split_novel_into_chapters":              "根据章节标记将小说文件拆分为单独的章节文件夹和文件",
		"generate_image_from_text":                    "使用DrawThings API根据文本生成图像，采用悬疑风格",
		"generate_image_from_image":                   "使用DrawThings API根据参考图像生成图像，采用悬疑风格",
		"generate_images_from_chapter":                "使用DrawThings API根据章节文本生成图像，采用悬疑风格",
		"generate_images_from_chapter_with_ai_prompt": "使用AI生成提示词和DrawThings API根据章节文本生成图像，采用悬疑风格",
		"generate_image_from_lyric_ai_prompt":         "使用歌词按每句歌词文本生成图像，",
	}

	defaultTools := []string{
		"generate_indextts2_audio",
		"generate_subtitles_from_indextts2",
		"file_split_novel_into_chapters",
		"generate_image_from_text",
		"generate_image_from_image",
		"generate_images_from_chapter",
		"generate_images_from_chapter_with_ai_prompt",
		"generate_image_from_lyric_ai_prompt",
	}

	for _, toolName := range defaultTools {
		description, exists := descriptions[toolName]
		if !exists {
			description = fmt.Sprintf("%s", toolName)
		}
		mcpTools = append(mcpTools, ToolInfo{
			Name:        toolName,
			Description: description,
			Path:        fmt.Sprintf("./mcp-tools/%s.yaml", toolName),
		})
	}

	log.Printf("加载了 %d 个备用工具", len(mcpTools))
}

// getToolDescription 根据工具名称获取描述
func getToolDescription(toolName string) string {
	descriptions := map[string]string{
		"generate_indextts2_audio":                    "使用IndexTTS2生成音频文件，具有高级语音克隆功能",
		"generate_subtitles_from_indextts2":           "使用Aegisub从IndexTTS2音频和提供的文本生成字幕(SRT)",
		"file_split_novel_into_chapters":              "根据章节标记将小说文件拆分为单独的章节文件夹和文件",
		"generate_image_from_text":                    "使用DrawThings API根据文本生成图像，采用悬疑风格",
		"generate_image_from_image":                   "使用DrawThings API根据参考图像生成图像，采用悬疑风格",
		"generate_images_from_chapter":                "使用DrawThings API根据章节文本生成图像，采用悬疑风格",
		"generate_images_from_chapter_with_ai_prompt": "使用AI生成提示词和DrawThings API根据章节文本生成图像，采用悬疑风格",
		"generate_image_from_lyric_ai_prompt":         "使用歌词按每句歌词文本生成图像",
	}

	if desc, exists := descriptions[toolName]; exists {
		return desc
	}

	return fmt.Sprintf("%s", toolName)
}

// Gin路由处理函数
func homePage(c *gin.Context) {
	// 获取项目根目录 - 使用固定路径方式
	wd, err := os.Getwd()
	if err != nil {
		log.Printf("无法获取当前工作目录: %v", err)
		c.String(http.StatusInternalServerError, "服务器内部错误")
		return
	}

	// 检查当前工作目录是否包含templates目录
	templatePath := filepath.Join(wd, "templates", "index.html")
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		// 如果当前工作目录下没有，尝试上两级目录（项目根目录）
		projectRoot := filepath.Dir(filepath.Dir(wd))
		templatePath = filepath.Join(projectRoot, "templates", "index.html")

		if _, err := os.Stat(templatePath); os.IsNotExist(err) {
			log.Printf("模板文件不存在: %s", templatePath)
			c.String(http.StatusInternalServerError, "模板文件不存在")
			return
		}
	}

	// 使用template.Must安全地解析模板
	tmpl := template.Must(template.New("index.html").ParseFiles(templatePath))

	if err := tmpl.Execute(c.Writer, nil); err != nil {
		log.Printf("执行模板失败: %v", err)
		c.String(http.StatusInternalServerError, "模板执行失败")
		return
	}
}

func wsEndpoint(c *gin.Context) {
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	// 确保在函数结束时关闭连接
	defer func() {
		log.Printf("WebSocket connection closing")
		ws.Close()
	}()

	// 添加客户端到全局广播服务
	clientChan := broadcast.GlobalBroadcastService.RegisterClient(ws)

	// 确保在函数结束时注销客户端
	defer func() {
		log.Printf("Unregistering client from broadcast service")
		client := &broadcast.Client{Conn: ws, Send: clientChan}
		broadcast.GlobalBroadcastService.UnregisterClient(client)
	}()

	// 启动goroutine处理来自广播服务的消息
	messageGoroutineDone := make(chan bool, 1)
	go func() {
		defer func() {
			log.Printf("Goroutine handling broadcast messages ending")
			messageGoroutineDone <- true
		}()

		for {
			select {
			case message, ok := <-clientChan:
				if !ok {
					// 通道已关闭，退出循环
					log.Printf("Client channel closed, exiting message loop")
					return
				}
				// 直接发送消息，因为现在BroadcastMessage已经包含了前端期望的字段
				if err := ws.WriteJSON(message); err != nil {
					log.Printf("Error sending message to client: %v", err)
					return
				}
			}
		}
	}()

	// 处理从客户端接收的消息
	for {
		var msg map[string]interface{}
		err := ws.ReadJSON(&msg)
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			break
		}
		// 可以处理从客户端发送的消息，如果需要的话
	}

	// 等待消息处理goroutine结束
	<-messageGoroutineDone
	log.Printf("WebSocket connection fully closed and cleaned up")
}

func resolveProjectRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	if strings.HasSuffix(wd, filepath.Join("cmd", "web_server")) {
		return filepath.Dir(filepath.Dir(wd)), nil
	}

	return wd, nil
}

func syncUploadedContent(projectRoot string) (int, int, error) {
	inputDir := filepath.Join(projectRoot, "input")
	if err := os.MkdirAll(inputDir, 0o755); err != nil {
		return 0, 0, err
	}

	entries, err := os.ReadDir(inputDir)
	if err != nil {
		return 0, 0, err
	}

	projectCount := 0
	chapterCount := 0
	for _, entry := range entries {
		var imported int

		switch {
		case entry.IsDir():
			imported, err = syncProjectDirectory(filepath.Join(inputDir, entry.Name()), entry.Name())
		case strings.EqualFold(filepath.Ext(entry.Name()), ".txt"):
			projectName := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
			imported, err = syncStandaloneNovelFile(filepath.Join(inputDir, entry.Name()), projectName)
		default:
			continue
		}

		if err != nil {
			return projectCount, chapterCount, err
		}
		if imported > 0 {
			projectCount++
			chapterCount += imported
		}
	}

	return projectCount, chapterCount, nil
}

func syncStandaloneNovelFile(novelPath, projectName string) (int, error) {
	fm := file.NewFileManager()
	if _, err := fm.CreateInputChapterStructure(novelPath); err != nil {
		return 0, err
	}

	return importChapterDirectories(filepath.Dir(novelPath), projectName)
}

func syncProjectDirectory(projectDir, projectName string) (int, error) {
	chapterDirs, err := findChapterDirectories(projectDir)
	if err != nil {
		return 0, err
	}
	if len(chapterDirs) == 0 {
		novelPath, err := selectNovelFile(projectDir, projectName)
		if err != nil {
			return 0, err
		}
		if novelPath == "" {
			return 0, nil
		}

		fm := file.NewFileManager()
		if _, err := fm.CreateInputChapterStructure(novelPath); err != nil {
			return 0, err
		}
	}

	return importChapterDirectories(projectDir, projectName)
}

func selectNovelFile(projectDir, projectName string) (string, error) {
	entries, err := os.ReadDir(projectDir)
	if err != nil {
		return "", err
	}

	expectedName := projectName + ".txt"
	var firstTxt string
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".txt") {
			continue
		}

		candidate := filepath.Join(projectDir, entry.Name())
		if strings.EqualFold(entry.Name(), expectedName) {
			return candidate, nil
		}
		if firstTxt == "" {
			firstTxt = candidate
		}
	}

	return firstTxt, nil
}

func findChapterDirectories(projectDir string) ([]string, error) {
	entries, err := os.ReadDir(projectDir)
	if err != nil {
		return nil, err
	}

	dirs := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() && isChapterDirectory(entry.Name()) {
			dirs = append(dirs, filepath.Join(projectDir, entry.Name()))
		}
	}

	return dirs, nil
}

func importChapterDirectories(projectDir, projectName string) (int, error) {
	project, err := ensureProjectRecord(projectName)
	if err != nil {
		return 0, err
	}

	entries, err := os.ReadDir(projectDir)
	if err != nil {
		return 0, err
	}

	imported := 0
	for _, entry := range entries {
		if !entry.IsDir() || !isChapterDirectory(entry.Name()) {
			continue
		}

		if err := upsertChapterRecord(project.ID, filepath.Join(projectDir, entry.Name()), entry.Name()); err != nil {
			return imported, err
		}
		imported++
	}

	return imported, nil
}

func ensureProjectRecord(projectName string) (*database.Project, error) {
	project, err := database.GetProjectByName(projectName)
	if err == nil {
		return project, nil
	}

	return database.CreateProject(projectName, "uploaded from input folder", "", "secret")
}

func upsertChapterRecord(projectID uint, chapterDir, chapterDirName string) error {
	chapterFile := filepath.Join(chapterDir, chapterDirName+".txt")
	content, err := os.ReadFile(chapterFile)
	if err != nil {
		return err
	}

	title := buildChapterTitle(chapterDirName, string(content))
	workflowParamsBytes, _ := json.Marshal(map[string]interface{}{
		"source_dir":  chapterDir,
		"chapter_dir": chapterDirName,
		"synced_at":   time.Now().Format(time.RFC3339),
	})

	var chapter database.Chapter
	queryErr := database.DB.Where("project_id = ? AND title = ?", projectID, title).First(&chapter).Error
	if queryErr == nil {
		return database.UpdateChapter(chapter.ID, map[string]interface{}{
			"content":         string(content),
			"workflow_params": string(workflowParamsBytes),
		})
	}

	created, err := database.CreateChapter(projectID, title, string(content), "")
	if err != nil {
		return err
	}

	return database.UpdateChapter(created.ID, map[string]interface{}{
		"workflow_params": string(workflowParamsBytes),
	})
}

func buildChapterTitle(chapterDirName, content string) string {
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			return trimmed
		}
	}

	return chapterDirName
}

// fileListHandler 返回指定目录中的文件列表
func fileListHandler(c *gin.Context) {
	dir := c.Query("dir")

	// 获取项目根目录
	wd, err := os.Getwd()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法获取当前工作目录", "status": "error"})
		return
	}

	projectRoot := wd
	if strings.HasSuffix(wd, "/cmd/web_server") {
		projectRoot = filepath.Dir(filepath.Dir(wd)) // 回退两级到项目根目录
	}

	if dir == "" {
		// 默认目录使用项目根路径
		dir = filepath.Join(projectRoot, "input")
	} else {
		// 解码URL参数
		decodedDir, err := url.QueryUnescape(dir)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid directory path", "status": "error"})
			return
		}

		// 如果是相对路径格式（如./input），将其转换为绝对路径
		if strings.HasPrefix(decodedDir, "./") {
			if strings.HasPrefix(decodedDir, "./input") {
				dir = filepath.Join(projectRoot, decodedDir[2:]) // 移除开头的"./"
			} else if strings.HasPrefix(decodedDir, "./output") {
				dir = filepath.Join(projectRoot, decodedDir[2:]) // 移除开头的"./"
			} else {
				c.JSON(http.StatusForbidden, gin.H{"error": "Invalid directory path", "status": "error"})
				return
			}
		} else {
			// 如果已经是绝对路径，直接使用
			dir = decodedDir
		}
	}

	// 确保路径安全，防止路径遍历攻击
	cleanDir := filepath.Clean(dir)

	// 检查路径是否在允许的范围内
	if !isAllowedProjectPath(cleanDir, projectRoot) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied", "status": "error"})
		return
	}

	files, err := os.ReadDir(cleanDir)
	if err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Directory not found", "status": "error"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "status": "error"})
		return
	}

	var fileList []map[string]interface{}
	for _, file := range files {
		filePath := filepath.Join(cleanDir, file.Name())
		info, err := os.Stat(filePath)
		if err != nil {
			continue
		}

		fileInfo := map[string]interface{}{
			"name":    file.Name(),
			"size":    info.Size(),
			"modTime": info.ModTime().Format(time.RFC3339),
			"isDir":   file.IsDir(),
			"type":    getFileType(file.Name()),
		}
		fileList = append(fileList, fileInfo)
	}

	c.JSON(http.StatusOK, gin.H{"files": fileList, "directory": cleanDir})
}

// fileContentHandler 返回文件的内容
func fileContentHandler(c *gin.Context) {
	pathParam := c.Query("path")

	// 获取项目根目录
	wd, err := os.Getwd()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法获取当前工作目录", "status": "error"})
		return
	}

	projectRoot := wd
	if strings.HasSuffix(wd, "/cmd/web_server") {
		projectRoot = filepath.Dir(filepath.Dir(wd)) // 回退两级到项目根目录
	}

	if pathParam == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File path is required", "status": "error"})
		return
	}

	// 解码URL参数
	decodedPath, err := url.QueryUnescape(pathParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file path", "status": "error"})
		return
	}

	// 如果是相对路径格式（如./input/file.txt），将其转换为绝对路径
	var cleanPath string
	if strings.HasPrefix(decodedPath, "./") {
		if strings.HasPrefix(decodedPath, "./input") || strings.HasPrefix(decodedPath, "./output") {
			cleanPath = filepath.Join(projectRoot, decodedPath[2:]) // 移除开头的"./"
		} else {
			c.JSON(http.StatusForbidden, gin.H{"error": "Invalid file path", "status": "error"})
			return
		}
	} else {
		// 如果已经是绝对路径，直接使用
		cleanPath = decodedPath
	}

	// 确保路径安全，防止路径遍历攻击
	cleanPath = filepath.Clean(cleanPath)

	// 检查路径是否在允许的范围内
	if !isAllowedProjectPath(cleanPath, projectRoot) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied", "status": "error"})
		return
	}

	// 检查文件类型，只允许预览特定类型的文件
	fileExt := strings.ToLower(filepath.Ext(cleanPath))
	allowedExts := map[string]bool{
		".txt":  true,
		".md":   true,
		".json": true,
		".yaml": true,
		".yml":  true,
		".xml":  true,
		".csv":  true,
		".log":  true,
	}

	if !allowedExts[fileExt] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File type not supported for preview", "status": "error"})
		return
	}

	content, err := os.ReadFile(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "File not found", "status": "error"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "status": "error"})
		return
	}

	c.Data(http.StatusOK, "text/plain; charset=utf-8", content)
}

// fileDeleteHandler 删除指定的文件或目录
func fileDeleteHandler(c *gin.Context) {
	pathParam := c.Query("path")

	// 获取项目根目录
	wd, err := os.Getwd()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法获取当前工作目录", "status": "error"})
		return
	}

	projectRoot := wd
	if strings.HasSuffix(wd, "/cmd/web_server") {
		projectRoot = filepath.Dir(filepath.Dir(wd)) // 回退两级到项目根目录
	}

	if pathParam == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File path is required", "status": "error"})
		return
	}

	// 解码URL参数
	decodedPath, err := url.QueryUnescape(pathParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file path", "status": "error"})
		return
	}

	// 如果是相对路径格式（如./input/file.txt），将其转换为绝对路径
	var cleanPath string
	if strings.HasPrefix(decodedPath, "./") {
		if strings.HasPrefix(decodedPath, "./input") || strings.HasPrefix(decodedPath, "./output") {
			cleanPath = filepath.Join(projectRoot, decodedPath[2:]) // 移除开头的"./"
		} else {
			c.JSON(http.StatusForbidden, gin.H{"error": "Invalid file path", "status": "error"})
			return
		}
	} else {
		// 如果已经是绝对路径，直接使用
		cleanPath = decodedPath
	}

	// 确保路径安全，防止路径遍历攻击
	cleanPath = filepath.Clean(cleanPath)

	// 检查路径是否在允许的范围内
	if !isAllowedProjectPath(cleanPath, projectRoot) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied", "status": "error"})
		return
	}

	// 确认文件或目录存在
	if _, err := os.Stat(cleanPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "File or directory does not exist", "status": "error"})
		return
	}

	err = os.RemoveAll(cleanPath) // 使用RemoveAll可以删除非空目录
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete file or directory: " + err.Error(), "status": "error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "File or directory deleted successfully"})
}

// fileUploadHandler 上传文件到指定目录
func fileUploadHandler(c *gin.Context) {
	// 解析 multipart form (32MB max)
	err := c.Request.ParseMultipartForm(32 << 20) // 32MB max memory
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unable to parse form", "status": "error"})
		return
	}

	dir := c.PostForm("dir")
	if dir == "" {
		dir = "./input" // 默认目录
	}

	// 确保路径安全，防止路径遍历攻击
	cleanDir := filepath.Clean(dir)

	// 获取当前工作目录作为基础路径
	workDir, err := os.Getwd()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to get working directory", "status": "error"})
		return
	}

	// 尝试找到项目根目录
	projectRoot := workDir
	// 如果当前在cmd/web_server目录下，向上两级到达项目根目录
	if strings.HasSuffix(workDir, "/cmd/web_server") {
		projectRoot = filepath.Dir(filepath.Dir(workDir))
	}

	// 检查目录是否在允许的范围内 - 只允许input目录
	allowedInputDir := filepath.Join(projectRoot, "input")

	// 处理相对路径和绝对路径的情况
	var cleanTargetDir string
	if strings.HasPrefix(cleanDir, "./") {
		// 如果是 ./ 开头的相对路径，转换为绝对路径
		cleanTargetDir = filepath.Clean(filepath.Join(projectRoot, cleanDir[2:]))
	} else if strings.HasPrefix(cleanDir, "input/") || cleanDir == "input" {
		// 如果是 input/ 开头的相对路径，转换为绝对路径
		cleanTargetDir = filepath.Clean(filepath.Join(projectRoot, cleanDir))
	} else {
		// 其他情况直接使用 cleanDir
		cleanTargetDir = filepath.Clean(cleanDir)
	}

	// 检查路径是否在允许的目录内 - 只允许上传到input目录
	isInInputDir := strings.HasPrefix(cleanTargetDir, allowedInputDir+string(filepath.Separator)) || cleanTargetDir == allowedInputDir

	if !isInInputDir {
		c.JSON(http.StatusForbidden, gin.H{"error": "文件上传路径不被允许，只能上传到input目录", "status": "error", "details": "目标路径: " + cleanTargetDir})
		return
	}

	// 确保目录存在
	if err := os.MkdirAll(dir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to create directory", "status": "error"})
		return
	}

	file, handler, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error retrieving file", "status": "error"})
		return
	}
	defer file.Close()

	filePath := filepath.Join(dir, handler.Filename)
	dest, err := os.Create(filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creating file", "status": "error"})
		return
	}
	defer dest.Close()

	_, err = io.Copy(dest, file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error saving file", "status": "error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "filename": handler.Filename, "message": "File uploaded successfully"})
}

// getFileType 根据文件扩展名确定文件类型
func getFileType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".bmp", ".svg", ".webp":
		return "image"
	case ".mp4", ".avi", ".mov", ".wmv", ".flv", ".mkv":
		return "video"
	case ".mp3", ".wav", ".flac", ".aac", ".ogg":
		return "audio"
	case ".txt", ".md", ".json", ".yaml", ".yml", ".xml", ".csv", ".log":
		return "text"
	case ".pdf":
		return "pdf"
	case ".zip", ".rar", ".tar", ".gz", ".7z":
		return "archive"
	default:
		return "unknown"
	}
}

// BroadcastLoggerAdapter 是一个自定义的zapcore.Core实现，用于将日志广播到WebSocket
type BroadcastLoggerAdapter struct {
	toolName string
	zapcore.Core
}

// NewBroadcastLoggerAdapter 创建一个新的广播日志适配器
func NewBroadcastLoggerAdapter(toolName string, encoder zapcore.Encoder, writeSyncer zapcore.WriteSyncer) *BroadcastLoggerAdapter {
	core := zapcore.NewCore(encoder, writeSyncer, zapcore.DebugLevel)
	return &BroadcastLoggerAdapter{
		toolName: toolName,
		Core:     core,
	}
}

// With 添加字段并返回新的Core
func (b *BroadcastLoggerAdapter) With(fields []zapcore.Field) zapcore.Core {
	newCore := b.Core.With(fields)
	return &BroadcastLoggerAdapter{
		toolName: b.toolName,
		Core:     newCore,
	}
}

// Check 检查日志级别是否启用
func (b *BroadcastLoggerAdapter) Check(entry zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if b.Core.Enabled(entry.Level) {
		return ce.AddCore(entry, b)
	}
	return ce
}

// Write 将日志条目写入并广播到WebSocket
func (b *BroadcastLoggerAdapter) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	// 首先让底层core处理日志
	err := b.Core.Write(entry, fields)

	// 构建日志消息
	// 创建一个临时编码器来生成日志消息
	encoder := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
	buffer, err2 := encoder.EncodeEntry(entry, fields)
	if err2 != nil {
		// 如果编码失败，使用简单消息
		return err
	}

	message := strings.TrimSpace(string(buffer.Bytes()))

	// 广播到WebSocket
	logType := "info"
	switch entry.Level {
	case zapcore.ErrorLevel:
		logType = "error"
	case zapcore.WarnLevel:
		logType = "error" // 使用error类型显示警告
	case zapcore.DPanicLevel, zapcore.PanicLevel, zapcore.FatalLevel:
		logType = "error"
	}

	broadcast.GlobalBroadcastService.SendMessage(logType, fmt.Sprintf("[%s] %s", b.toolName, message), broadcast.GetTimeStr())

	return err
}

// updateChapterAudio 更新章节的音频URL
func (wp *WorkflowProcessor) updateChapterAudio(chapterID uint, audioURL string) error {
	return database.DB.Model(&database.Chapter{}).Where("id = ?", chapterID).Update("audio_url", audioURL).Error
}

// updateChapterImages 更新章节的图像路径
func (wp *WorkflowProcessor) updateChapterImages(chapterID uint, imagePaths string) error {
	return database.DB.Model(&database.Chapter{}).Where("id = ?", chapterID).Update("image_paths", imagePaths).Error
}

// getImageFilesFromDir 从目录中获取图像文件列表
func (wp *WorkflowProcessor) getImageFilesFromDir(dir string) ([]string, error) {
	var imageFiles []string
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if !file.IsDir() {
			ext := strings.ToLower(filepath.Ext(file.Name()))
			if ext == ".png" || ext == ".jpg" || ext == ".jpeg" {
				imagePath := filepath.Join(dir, file.Name())
				imageFiles = append(imageFiles, imagePath)
			}
		}
	}

	return imageFiles, nil
}

func webServerMain() {
	initializeMCPServer()

	// 初始化全局广播服务
	broadcast.GlobalBroadcastService = broadcast.NewBroadcastService()
	var wg sync.WaitGroup
	wg.Add(1)
	go broadcast.GlobalBroadcastService.Start(&wg)

	// 设置Gin为发布模式以获得更好的性能
	gin.SetMode(gin.ReleaseMode)
	//设置gin的超时时间
	r := gin.Default()

	// 获取项目根目录的绝对路径
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal("无法获取当前工作目录:", err)
	}
	projectRoot := wd

	// 如果是从子目录运行的，需要调整到项目根目录
	if strings.HasSuffix(wd, "/cmd/web_server") {
		projectRoot = filepath.Dir(filepath.Dir(wd)) // 回退两级到项目根目录
	}

	// 注册路由
	r.GET("/", homePage)
	r.GET("/ws", wsEndpoint)

	// workflow 主路由独立注册；提示词模板与参数跟踪按 processor 可用性降级
	registerWorkflowSupportRoutes(r)

	// 章节和场景管理相关路由
	chapterSceneAPI := api_pkg.NewChapterSceneAPI()
	r.GET("/api/chapters", chapterSceneAPI.GetChapters)
	r.GET("/api/chapters/:id", chapterSceneAPI.GetChapterByID)
	r.PUT("/api/chapters/:id", chapterSceneAPI.UpdateChapter)
	r.GET("/api/chapters/:id/scenes", chapterSceneAPI.GetScenesByChapterID)
	r.GET("/api/scenes/:id", chapterSceneAPI.GetSceneByID)
	r.PUT("/api/scenes/:id", chapterSceneAPI.UpdateScene)
	r.POST("/api/chapters/:id/retry", chapterSceneAPI.RetryChapter)
	r.POST("/api/scenes/:id/retry", chapterSceneAPI.RetryScene)

	// 添加文件管理API端点
	r.GET("/api/files/list", fileListHandler)
	r.GET("/api/files/content", fileContentHandler)
	r.DELETE("/api/files/delete", fileDeleteHandler)
	r.POST("/api/files/upload", fileUploadHandler)

	// 添加设置API端点
	r.GET("/api/settings", getSettingsHandler)
	r.POST("/api/settings", settingsHandler)
	// 添加调试API端点，用于检查全局变量
	r.GET("/api/debug/global-style", debugGlobalStyleHandler)

	// 添加静态文件服务，用于提供input和output目录的文件访问
	// 使用项目根路径确保正确访问input和output目录
	inputPath := filepath.Join(projectRoot, "input")
	outputPath := filepath.Join(projectRoot, "output")
	assetsPath := filepath.Join(projectRoot, "assets")
	templatePath := filepath.Join(projectRoot, "templates")

	// 确保目录存在
	os.MkdirAll(inputPath, 0755)
	os.MkdirAll(outputPath, 0755)
	os.MkdirAll(assetsPath, 0755)
	os.MkdirAll(templatePath, 0755)

	r.Static("/files/input", inputPath)
	r.Static("/files/output", outputPath)
	r.Static("assets", assetsPath)
	r.Static("css", filepath.Join(templatePath, "css"))
	r.Static("js", filepath.Join(templatePath, "js"))

	// 从环境变量获取端口，默认为8080
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// 一键出片功能 - 完整工作流处理
	go func() {
		broadcast.GlobalBroadcastService.SendLog("movie", "[一键出片] 服务器启动完成，准备处理一键出片任务", broadcast.GetTimeStr())
	}()

	log.Println("服务器启动在 :" + port)
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      r,
		ReadTimeout:  15 * time.Minute, // 读取请求头最大耗时
		WriteTimeout: 15 * time.Minute, // 写响应最大耗时
		IdleTimeout:  15 * time.Second, // 空闲连接保持时间
	}

	// 为服务器关闭创建一个goroutine
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("服务器运行出错: %v", err)
		}
	}()

	// 等待中断信号来优雅关闭服务器
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("接收到关闭信号，正在关闭服务器...")

	// 创建关闭上下文，最多等待5秒
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 优雅关闭服务器
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("服务器关闭失败: %v", err)
	} else {
		log.Println("服务器已成功关闭")
	}
}

// StartServer 启动Web服务器
func StartServer() {
	webServerMain()
}

// WorkflowProcessor 工作流处理器
type WorkflowProcessor struct {
	logger        *zap.Logger
	fileManager   *file.FileManager
	ttsClient     *indextts2.IndexTTS2Client
	aegisubGen    *aegisub.AegisubGenerator
	drawThingsGen *drawthings.ChapterImageGenerator
	comfyClient   *comfyui.Client
	imageConfig   configpkg.ImageConfig
}

var GlobalStyle = drawthings.DefaultSuspenseStyle
var AdditionalPrompt = ", " + drawthings.DefaultSuspenseStyle

func (wp *WorkflowProcessor) imageBackendName() string {
	if strings.EqualFold(wp.imageConfig.Provider, "comfyui") {
		return "comfyui"
	}
	return "drawthings"
}

func (wp *WorkflowProcessor) renderImage(promptText, outputFile string, width, height int) error {
	if strings.EqualFold(wp.imageConfig.Provider, "comfyui") {
		if wp.comfyClient == nil {
			return fmt.Errorf("comfyui client is not configured")
		}
		return wp.comfyClient.GenerateImage(promptText, wp.imageConfig.NegativePrompt, outputFile, width, height, wp.imageConfig.Steps, wp.imageConfig.CFGScale)
	}

	return wp.drawThingsGen.Client.GenerateImageFromTextWithDefaultTemplate(promptText, outputFile, width, height, false)
}

// generateImagesWithOllamaPrompts 使用Ollama优化的提示词生成图像
func (wp *WorkflowProcessor) generateImagesWithOllamaPrompts(content, imagesDir string, chapterID uint, audioDurationSecs int) error {
	// 使用Ollama分析整个章节内容并生成分镜提示词
	styleDesc := GlobalStyle

	// 使用实际音频时长，如果未提供则估算
	estimatedDurationSecs := audioDurationSecs
	if estimatedDurationSecs <= 0 {
		// 估算音频时长（假设每分钟300字，即每个字符约0.2秒）
		estimatedDurationSecs = len(content) * 2 / 10 // 简化估算，大约每个字符0.2秒
		if estimatedDurationSecs < 60 {               // 最少1分钟
			estimatedDurationSecs = 60
		}
	}

	// 让Ollama分析整个章节并生成分镜
	wp.logger.Info("📸开始Ollama分镜分析", zap.Uint("chapter_id", chapterID), zap.Int("content_length", len(content)), zap.Int("estimated_duration_secs", estimatedDurationSecs))
	sceneDescriptions, err := wp.drawThingsGen.OllamaClient.AnalyzeScenesAndGeneratePrompts(content, styleDesc, estimatedDurationSecs)
	if err != nil {
		wp.logger.Warn("使用Ollama分析场景并生成分镜提示词失败",
			zap.Error(err))

		// 如果Ollama场景分析失败，回退到原来的段落处理方式
		wp.logger.Info("Ollama分镜分析失败，回退到段落处理方式")
		paragraphs := wp.splitChapterIntoParagraphsWithMerge(content)
		config_str, err := database.GetConfigAddStr(wp.drawThingsGen.Client.DB)
		if err != nil {
			wp.logger.Warn("获取配置失败",
				zap.Error(err))
		}
		// 先分析整个内容确定基调
		wp.drawThingsGen.OllamaClient.AnalyzeSceneAndBackground(content)

		// 再为每一个分镜进行出图
		for idx, paragraph := range paragraphs {
			if strings.TrimSpace(paragraph) == "" {
				continue
			}

			// 记录Ollama生成图像提示词的过程
			promptGenerationStartTime := time.Now()
			optimizedPrompt, err := wp.drawThingsGen.OllamaClient.GenerateImagePrompt(paragraph, styleDesc, config_str)
			promptGenerationEndTime := time.Now()

			//手工拆分
			if err != nil {
				wp.logger.Warn("使用Ollama生成图像提示词失败，使用原始文本",
					zap.Int("paragraph_index", idx),
					zap.String("paragraph", paragraph),
					zap.Error(err))
				optimizedPrompt = paragraph + AdditionalPrompt
			}

			// 记录图像生成请求参数
			drawThingsConfig := map[string]interface{}{
				"backend":         wp.imageBackendName(),
				"prompt":          optimizedPrompt,
				"width":           wp.imageConfig.Width,
				"height":          wp.imageConfig.Height,
				"negative_prompt": wp.imageConfig.NegativePrompt,
				"steps":           wp.imageConfig.Steps,
				"guidance_scale":  wp.imageConfig.CFGScale,
				"batch_size":      1,
				"model":           wp.imageConfig.DrawThings.Model,
				"checkpoint":      wp.imageConfig.ComfyUI.Checkpoint,
				"is_suspense":     true,
			}
			drawThingsConfigBytes, _ := json.Marshal(drawThingsConfig)

			imageFile := filepath.Join(imagesDir, fmt.Sprintf("paragraph_%02d.png", idx+1))
			wp.logger.Info(fmt.Sprintf("📸开始生成段落图像: %d", idx+1))
			err = wp.renderImage(optimizedPrompt, imageFile, wp.imageConfig.Width, wp.imageConfig.Height)

			drawThingsResult := map[string]interface{}{
				"image_file": imageFile,
				"success":    err == nil,
				"error":      err,
			}
			drawThingsResultBytes, _ := json.Marshal(drawThingsResult)

			if err != nil {
				wp.logger.Warn("生成图像失败", zap.String("paragraph", paragraph[:min(len(paragraph), 50)]), zap.Error(err))
				fmt.Printf("⚠️ 段落图像生成失败: %v\n", err)
			} else {
				fmt.Printf("✅ 段落图像生成完成: %s\n", imageFile)
			}

			// 准备Ollama请求参数
			ollamaRequest := map[string]interface{}{
				"text":      paragraph,
				"style":     styleDesc,
				"timestamp": promptGenerationStartTime.Format(time.RFC3339),
			}
			ollamaRequestBytes, _ := json.Marshal(ollamaRequest)

			// 即使在回退情况下，也将段落作为场景保存到数据库
			scene := &database.Scene{
				Title:            fmt.Sprintf("段落场景 %d", idx+1),
				Description:      paragraph,
				Prompt:           optimizedPrompt,
				SegmentationInfo: fmt.Sprintf("段落信息: %s", paragraph),
				OriginalText:     content,
				OllamaRequest:    string(ollamaRequestBytes),
				OllamaResponse:   optimizedPrompt,
				DrawThingsConfig: string(drawThingsConfigBytes),
				DrawThingsResult: string(drawThingsResultBytes),
				ChapterID:        chapterID, // 使用传入的章节ID
				ImageURL:         imageFile,
				AudioURL:         "",
				RetryCount:       0,
				Sort:             idx + 1,
				StartTime:        promptGenerationStartTime,
				EndTime:          time.Now(),
				Status:           "completed",
			}

			// 记录工作流详细参数
			workflowDetails := map[string]interface{}{
				"ollama_analysis": map[string]interface{}{
					"request": map[string]interface{}{
						"text":      paragraph[:min(len(paragraph), 200)],
						"style":     styleDesc,
						"timestamp": promptGenerationStartTime.Format(time.RFC3339),
					},
					"response": map[string]interface{}{
						"prompt":    optimizedPrompt,
						"timestamp": promptGenerationEndTime.Format(time.RFC3339),
						"duration":  promptGenerationEndTime.Sub(promptGenerationStartTime).Seconds(),
					},
				},
				"drawthings_generation": map[string]interface{}{
					"request":   drawThingsConfig,
					"response":  drawThingsResult,
					"timestamp": time.Now().Format(time.RFC3339),
					"duration":  time.Since(promptGenerationEndTime).Seconds(),
				},
				"execution_info": map[string]interface{}{
					"paragraph_index": idx,
					"paragraph_text":  paragraph[:min(len(paragraph), 200)],
					"chapter_id":      chapterID,
					"image_file":      imageFile,
					"status":          "completed",
				},
			}

			workflowDetailsBytes, _ := json.Marshal(workflowDetails)
			scene.WorkflowDetails = string(workflowDetailsBytes)

			result := database.DB.Create(scene)
			if result.Error != nil {
				wp.logger.Error("保存段落场景到数据库失败", zap.Error(result.Error), zap.String("paragraph", paragraph[:min(len(paragraph), 100)]))
				fmt.Printf("❌ 保存段落场景到数据库失败: %v\n", result.Error)
			} else {
				wp.logger.Info("段落场景已保存到数据库", zap.Uint("scene_id", scene.ID), zap.Uint("chapter_id", chapterID))
				fmt.Printf("✅ 段落场景已保存到数据库，ID: %d\n", scene.ID)
			}
		}

		return nil
	}

	// 如果Ollama分镜分析成功，使用生成的分镜描述生成图像
	wp.logger.Info("Ollama分镜分析成功", zap.Int("scene_count", len(sceneDescriptions)))

	// 将场景数据保存到数据库
	for idx, sceneDesc := range sceneDescriptions {
		imageFile := filepath.Join(imagesDir, fmt.Sprintf("scene_%02d.png", idx+1))

		// 记录图像生成请求参数
		drawThingsConfig := map[string]interface{}{
			"backend":         wp.imageBackendName(),
			"prompt":          sceneDesc,
			"width":           wp.imageConfig.Width,
			"height":          wp.imageConfig.Height,
			"negative_prompt": wp.imageConfig.NegativePrompt,
			"steps":           wp.imageConfig.Steps,
			"guidance_scale":  wp.imageConfig.CFGScale,
			"batch_size":      1,
			"model":           wp.imageConfig.DrawThings.Model,
			"checkpoint":      wp.imageConfig.ComfyUI.Checkpoint,
			"is_suspense":     true,
		}
		drawThingsConfigBytes, _ := json.Marshal(drawThingsConfig)

		// 使用分镜描述生成图像
		err = wp.renderImage(sceneDesc, imageFile, wp.imageConfig.Width, wp.imageConfig.Height)

		drawThingsResult := map[string]interface{}{
			"image_file": imageFile,
			"success":    err == nil,
			"error":      err,
		}
		drawThingsResultBytes, _ := json.Marshal(drawThingsResult)

		if err != nil {
			wp.logger.Warn("生成分镜图像失败", zap.String("scene", sceneDesc[:min(len(sceneDesc), 50)]), zap.Error(err))
			fmt.Printf("⚠️  分镜图像生成失败: %v\n", err)
		} else {
			fmt.Printf("✅ 分镜图像生成完成: %s\n", imageFile)
		}

		// 准备Ollama请求参数（分镜分析）
		sceneAnalysisStartTime := time.Now() // 假设分析时间就是现在
		ollamaRequest := map[string]interface{}{
			"content":      content[:min(len(content), 200)],
			"style":        styleDesc,
			"duration_sec": estimatedDurationSecs,
			"timestamp":    sceneAnalysisStartTime.Format(time.RFC3339),
		}
		ollamaRequestBytes, _ := json.Marshal(ollamaRequest)

		// 创建场景记录并保存到数据库
		scene := &database.Scene{
			Title:            fmt.Sprintf("场景 %d", idx+1),
			Description:      sceneDesc,
			Prompt:           sceneDesc,
			SegmentationInfo: fmt.Sprintf("分镜信息: %s", sceneDesc),
			OriginalText:     content,
			OllamaRequest:    string(ollamaRequestBytes),
			OllamaResponse:   sceneDesc,
			DrawThingsConfig: string(drawThingsConfigBytes),
			DrawThingsResult: string(drawThingsResultBytes),
			ChapterID:        chapterID, // 使用传入的章节ID
			ImageURL:         imageFile,
			AudioURL:         "",
			RetryCount:       0,
			Sort:             idx + 1,
			StartTime:        sceneAnalysisStartTime,
			EndTime:          time.Now(),
			Status:           "completed",
		}

		// 记录工作流详细参数
		workflowDetails := map[string]interface{}{
			"ollama_analysis": map[string]interface{}{
				"request": map[string]interface{}{
					"content":      content[:min(len(content), 200)],
					"style":        styleDesc,
					"duration_sec": estimatedDurationSecs,
					"timestamp":    sceneAnalysisStartTime.Format(time.RFC3339),
				},
				"response": map[string]interface{}{
					"scene_descriptions": sceneDescriptions,
					"timestamp":          time.Now().Format(time.RFC3339),
				},
			},
			"drawthings_generation": map[string]interface{}{
				"request":   drawThingsConfig,
				"response":  drawThingsResult,
				"timestamp": time.Now().Format(time.RFC3339),
			},
			"execution_info": map[string]interface{}{
				"scene_index":  idx,
				"scene_text":   sceneDesc[:min(len(sceneDesc), 200)],
				"chapter_id":   chapterID,
				"image_file":   imageFile,
				"status":       "completed",
				"total_scenes": len(sceneDescriptions),
			},
		}

		workflowDetailsBytes, _ := json.Marshal(workflowDetails)
		scene.WorkflowDetails = string(workflowDetailsBytes)

		result := database.DB.Create(scene)
		if result.Error != nil {
			wp.logger.Error("保存场景到数据库失败", zap.Error(result.Error), zap.String("scene_desc", sceneDesc[:min(len(sceneDesc), 100)]))
			fmt.Printf("❌ 保存场景到数据库失败: %v\n", result.Error)
		} else {
			wp.logger.Info("场景已保存到数据库", zap.Uint("scene_id", scene.ID), zap.Uint("chapter_id", chapterID))
			fmt.Printf("✅ 场景已保存到数据库(%d/%d)，ID: %d\n", idx+1, len(sceneDescriptions), scene.ID)
			wp.drawThingsGen.OllamaClient.BroadcastService.SendMessage("分镜", fmt.Sprintf("🖥 场景已保存到数据库(%d/%d)，ID: %d\n", idx, len(sceneDescriptions), scene.ID), broadcast.GetTimeStr())
		}
		wp.drawThingsGen.OllamaClient.BroadcastService.SendMessage("分镜", fmt.Sprintf("场景 %s 生成完成", fmt.Sprintf("scene_%02d.png", idx+1)), broadcast.GetTimeStr())
	}

	return nil
}

// splitChapterIntoParagraphsWithMerge 将章节文本分割为段落，并对短段落进行合并
func (wp *WorkflowProcessor) splitChapterIntoParagraphsWithMerge(text string) []string {
	// 按换行符分割文本
	lines := strings.Split(text, "\n")

	var rawParagraphs []string
	var currentParagraph strings.Builder

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		if trimmedLine == "" {
			// 遇到空行，结束当前段落
			if currentParagraph.Len() > 0 {
				rawParagraphs = append(rawParagraphs, strings.TrimSpace(currentParagraph.String()))
				currentParagraph.Reset()
			}
		} else {
			// 添加到当前段落
			if currentParagraph.Len() > 0 {
				currentParagraph.WriteString(" ")
			}
			currentParagraph.WriteString(trimmedLine)
		}
	}

	// 处理最后一个段落
	if currentParagraph.Len() > 0 {
		rawParagraphs = append(rawParagraphs, strings.TrimSpace(currentParagraph.String()))
	}

	// 合并短段落
	var mergedParagraphs []string
	minLength := 50 // 设定最小长度阈值，低于此值的段落将与相邻段落合并

	for i := 0; i < len(rawParagraphs); i++ {
		currentPara := rawParagraphs[i]

		// 如果当前段落太短，考虑与下一个段落合并
		if len(currentPara) < minLength && i < len(rawParagraphs)-1 {
			// 与下一个段落合并
			merged := currentPara + " " + rawParagraphs[i+1]
			mergedParagraphs = append(mergedParagraphs, merged)
			i++ // 跳过下一个段落，因为它已经被合并了
		} else {
			// 检查是否当前段落太短但已经是最后一段
			if len(currentPara) < minLength && len(mergedParagraphs) > 0 {
				// 将其与前一段落合并
				lastIdx := len(mergedParagraphs) - 1
				mergedParagraphs[lastIdx] = mergedParagraphs[lastIdx] + " " + currentPara
			} else {
				// 添加正常段落
				mergedParagraphs = append(mergedParagraphs, currentPara)
			}
		}
	}

	// 过滤掉过短的段落（比如只有标点符号）
	var filtered []string
	for _, para := range mergedParagraphs {
		// 只保留非空且有一定长度的段落
		if len(strings.TrimSpace(para)) > 3 { // 至少3个字符
			filtered = append(filtered, para)
		}
	}

	return filtered
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// isChapterDirectory 检查目录名是否为 chapter_XX 格式
func isChapterDirectory(dirName string) bool {
	matched, err := regexp.MatchString(`^chapter_\d+$`, dirName)
	if err != nil {
		return false
	}
	return matched
}

// 设置处理函数
func getSettingsHandler(c *gin.Context) {
	cfg, err := loadWebConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": fmt.Sprintf("加载配置失败: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"general": gin.H{
				"image_width":  cfg.Image.Width,
				"image_height": cfg.Image.Height,
			},
			"tts": gin.H{
				"provider":        cfg.TTS.Provider,
				"reference_audio": cfg.Paths.ReferenceAudio,
				"api_url":         cfg.TTS.IndexTTS2.APIURL,
				"timeout_seconds": cfg.TTS.IndexTTS2.TimeoutSeconds,
				"max_retries":     cfg.TTS.IndexTTS2.MaxRetries,
				"voice_model":     cfg.TTS.VoiceModel,
				"sample_rate":     cfg.TTS.SampleRate,
				"python_path":     cfg.TTS.PythonPath,
				"indextts_path":   cfg.TTS.IndexTTSPath,
			},
			"subtitle": gin.H{
				"provider":        cfg.Subtitle.Provider,
				"style":           cfg.Subtitle.Style,
				"font_name":       cfg.Subtitle.FontName,
				"font_size":       cfg.Subtitle.FontSize,
				"script_path":     cfg.Subtitle.Aegisub.ScriptPath,
				"executable_path": cfg.Subtitle.Aegisub.ExecutablePath,
				"use_automation":  cfg.Subtitle.Aegisub.UseAutomation,
			},
			"project": gin.H{
				"provider": cfg.Project.Provider,
			},
			"image": gin.H{
				"provider":        cfg.Image.Provider,
				"width":           cfg.Image.Width,
				"height":          cfg.Image.Height,
				"steps":           cfg.Image.Steps,
				"cfg_scale":       cfg.Image.CFGScale,
				"negative_prompt": cfg.Image.NegativePrompt,
				"comfyui": gin.H{
					"api_url":         cfg.Image.ComfyUI.APIURL,
					"checkpoint":      cfg.Image.ComfyUI.Checkpoint,
					"output_node_id":  cfg.Image.ComfyUI.OutputNodeID,
					"filename_prefix": cfg.Image.ComfyUI.FilenamePrefix,
				},
			},
			"ollama": gin.H{
				"api_url":         cfg.Ollama.APIURL,
				"model":           cfg.Ollama.Model,
				"timeout_seconds": cfg.Ollama.TimeoutSeconds,
			},
			"image_provider": cfg.Image.Provider,
		},
	})
}

func settingsHandler(c *gin.Context) {
	var reqBody map[string]interface{}
	err := json.NewDecoder(c.Request.Body).Decode(&reqBody)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": fmt.Sprintf("解析请求失败: %v", err)})
		return
	}

	settingType, ok := reqBody["setting_type"].(string)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "缺少setting_type参数"})
		return
	}

	// 根据设置类型处理不同的设置
	switch settingType {
	case "style_template":
		// 处理风格模板设置
		settingValue, exists := reqBody["setting_value"]
		if !exists {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "缺少setting_value参数"})
			return
		}

		// 将设置值转换为字符串
		templateIdStr := ""
		if settingValue != nil {
			if templateId, ok := settingValue.(string); ok {
				templateIdStr = templateId
			} else {
				c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "setting_value必须是字符串类型"})
				return
			}
		}

		// 如果模板ID不为空，则获取模板并更新全局设置
		if templateIdStr != "" && templateIdStr != "undefined" {
			promptTemplateID, err := strconv.ParseUint(templateIdStr, 10, 32)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": fmt.Sprintf("解析模板ID失败: %v", err)})
				return
			}

			// 从数据库获取模板
			template, err := database.GetPromptTemplateByID(database.DB, uint(promptTemplateID))
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": fmt.Sprintf("获取模板失败: %v", err)})
				return
			}

			// 更新全局风格变量
			if template.StyleAddon != "" {
				GlobalStyle = template.StyleAddon
				AdditionalPrompt = ", " + template.StyleAddon

				// 如果MCP服务器实例存在，也更新其全局样式
				if mcpServerInstance != nil {
					handler := mcpServerInstance.GetHandler()
					if handler != nil {
						handler.UpdateGlobalStyle(template.StyleAddon)
					}
				}
			}
		} else {
			// 如果模板ID为空或undefined，重置为默认值
			GlobalStyle = drawthings.DefaultSuspenseStyle
			AdditionalPrompt = ", " + drawthings.DefaultSuspenseStyle

			// 如果MCP服务器实例存在，也更新其全局样式为默认值
			if mcpServerInstance != nil {
				handler := mcpServerInstance.GetHandler()
				if handler != nil {
					handler.UpdateGlobalStyle(drawthings.DefaultSuspenseStyle)
				}
			}
		}

		c.JSON(http.StatusOK, gin.H{"status": "success", "message": "风格设置已保存"})

	case "general":
		// 处理通用设置
		settingValue, exists := reqBody["setting_value"]
		if !exists {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "缺少setting_value参数"})
			return
		}

		// 这里可以处理图像尺寸、质量等通用设置
		// 目前我们只返回成功响应，实际的设置处理可以根据需要扩展
		// 将settingValue转换为map以备将来使用
		if generalSettings, ok := settingValue.(map[string]interface{}); ok {
			imageWidth, hasWidth := intFromSetting(generalSettings["image_width"])
			imageHeight, hasHeight := intFromSetting(generalSettings["image_height"])
			if hasWidth || hasHeight {
				if err := updateConfigFile(func(cfgMap map[string]interface{}) error {
					imageCfg := ensureConfigMap(cfgMap, "image")
					if hasWidth {
						imageCfg["width"] = imageWidth
					}
					if hasHeight {
						imageCfg["height"] = imageHeight
					}
					return nil
				}); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": fmt.Sprintf("保存通用设置失败: %v", err)})
					return
				}
			}
		}
		c.JSON(http.StatusOK, gin.H{"status": "success", "message": "通用设置已保存"})

	case "ollama":
		settingValue, exists := reqBody["setting_value"]
		if !exists {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "缺少setting_value参数"})
			return
		}

		ollamaSettings, ok := settingValue.(map[string]interface{})
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "setting_value必须是对象类型"})
			return
		}

		apiURL := strings.TrimSpace(fmt.Sprint(ollamaSettings["api_url"]))
		model := strings.TrimSpace(fmt.Sprint(ollamaSettings["model"]))
		timeoutSeconds, ok := intFromSetting(ollamaSettings["timeout_seconds"])
		if !ok {
			timeoutSeconds = 120
		}
		if apiURL == "" || model == "" {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "api_url 和 model 不能为空"})
			return
		}

		if err := updateConfigFile(func(cfgMap map[string]interface{}) error {
			ollamaCfg := ensureConfigMap(cfgMap, "ollama")
			ollamaCfg["api_url"] = apiURL
			ollamaCfg["model"] = model
			ollamaCfg["timeout_seconds"] = timeoutSeconds
			return nil
		}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": fmt.Sprintf("保存 Ollama 配置失败: %v", err)})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Ollama 配置已保存"})

	case "tts":
		settingValue, exists := reqBody["setting_value"]
		if !exists {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "缺少setting_value参数"})
			return
		}
		ttsSettings, ok := settingValue.(map[string]interface{})
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "setting_value必须是对象类型"})
			return
		}
		provider := strings.TrimSpace(fmt.Sprint(ttsSettings["provider"]))
		apiURL := strings.TrimSpace(fmt.Sprint(ttsSettings["api_url"]))
		referenceAudio := strings.TrimSpace(fmt.Sprint(ttsSettings["reference_audio"]))
		voiceModel := strings.TrimSpace(fmt.Sprint(ttsSettings["voice_model"]))
		pythonPath := strings.TrimSpace(fmt.Sprint(ttsSettings["python_path"]))
		indexTTSPath := strings.TrimSpace(fmt.Sprint(ttsSettings["indextts_path"]))
		timeoutSeconds, ok := intFromSetting(ttsSettings["timeout_seconds"])
		if !ok {
			timeoutSeconds = 300
		}
		maxRetries, ok := intFromSetting(ttsSettings["max_retries"])
		if !ok {
			maxRetries = 3
		}
		sampleRate, ok := intFromSetting(ttsSettings["sample_rate"])
		if !ok {
			sampleRate = 24000
		}
		if provider == "" || apiURL == "" {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "provider 和 api_url 不能为空"})
			return
		}
		if err := updateConfigFile(func(cfgMap map[string]interface{}) error {
			pathsCfg := ensureConfigMap(cfgMap, "paths")
			ttsCfg := ensureConfigMap(cfgMap, "tts")
			indexCfg := ensureConfigMap(ttsCfg, "indextts2")
			pathsCfg["reference_audio"] = referenceAudio
			ttsCfg["provider"] = provider
			ttsCfg["voice_model"] = voiceModel
			ttsCfg["python_path"] = pythonPath
			ttsCfg["indextts_path"] = indexTTSPath
			ttsCfg["sample_rate"] = sampleRate
			indexCfg["api_url"] = apiURL
			indexCfg["timeout_seconds"] = timeoutSeconds
			indexCfg["max_retries"] = maxRetries
			return nil
		}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": fmt.Sprintf("保存 TTS 配置失败: %v", err)})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "success", "message": "TTS 配置已保存"})

	case "subtitle":
		settingValue, exists := reqBody["setting_value"]
		if !exists {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "缺少setting_value参数"})
			return
		}
		subtitleSettings, ok := settingValue.(map[string]interface{})
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "setting_value必须是对象类型"})
			return
		}
		provider := strings.TrimSpace(fmt.Sprint(subtitleSettings["provider"]))
		style := strings.TrimSpace(fmt.Sprint(subtitleSettings["style"]))
		fontName := strings.TrimSpace(fmt.Sprint(subtitleSettings["font_name"]))
		scriptPath := strings.TrimSpace(fmt.Sprint(subtitleSettings["script_path"]))
		executablePath := strings.TrimSpace(fmt.Sprint(subtitleSettings["executable_path"]))
		fontSize, ok := intFromSetting(subtitleSettings["font_size"])
		if !ok {
			fontSize = 48
		}
		useAutomation, _ := subtitleSettings["use_automation"].(bool)
		if provider == "" || scriptPath == "" {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "provider 和 script_path 不能为空"})
			return
		}
		if err := updateConfigFile(func(cfgMap map[string]interface{}) error {
			subtitleCfg := ensureConfigMap(cfgMap, "subtitle")
			aegisubCfg := ensureConfigMap(subtitleCfg, "aegisub")
			subtitleCfg["provider"] = provider
			subtitleCfg["style"] = style
			subtitleCfg["font_name"] = fontName
			subtitleCfg["font_size"] = fontSize
			aegisubCfg["script_path"] = scriptPath
			aegisubCfg["executable_path"] = executablePath
			aegisubCfg["use_automation"] = useAutomation
			return nil
		}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": fmt.Sprintf("保存字幕配置失败: %v", err)})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "success", "message": "字幕配置已保存"})

	case "project":
		settingValue, exists := reqBody["setting_value"]
		if !exists {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "缺少setting_value参数"})
			return
		}
		projectSettings, ok := settingValue.(map[string]interface{})
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "setting_value必须是对象类型"})
			return
		}
		provider := strings.TrimSpace(fmt.Sprint(projectSettings["provider"]))
		if provider == "" {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "provider 不能为空"})
			return
		}
		if err := updateConfigFile(func(cfgMap map[string]interface{}) error {
			projectCfg := ensureConfigMap(cfgMap, "project")
			projectCfg["provider"] = provider
			return nil
		}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": fmt.Sprintf("保存成片配置失败: %v", err)})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "success", "message": "成片配置已保存"})

	case "image":
		settingValue, exists := reqBody["setting_value"]
		if !exists {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "缺少setting_value参数"})
			return
		}
		imageSettings, ok := settingValue.(map[string]interface{})
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "setting_value必须是对象类型"})
			return
		}
		provider := strings.TrimSpace(fmt.Sprint(imageSettings["provider"]))
		negativePrompt := strings.TrimSpace(fmt.Sprint(imageSettings["negative_prompt"]))
		width, ok := intFromSetting(imageSettings["width"])
		if !ok {
			width = 512
		}
		height, ok := intFromSetting(imageSettings["height"])
		if !ok {
			height = 896
		}
		steps, ok := intFromSetting(imageSettings["steps"])
		if !ok {
			steps = 30
		}
		cfgScale := 7.5
		switch v := imageSettings["cfg_scale"].(type) {
		case float64:
			cfgScale = v
		case int:
			cfgScale = float64(v)
		case string:
			if parsed, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil {
				cfgScale = parsed
			}
		}
		comfySettings, _ := imageSettings["comfyui"].(map[string]interface{})
		comfyAPIURL := strings.TrimSpace(fmt.Sprint(comfySettings["api_url"]))
		comfyCheckpoint := strings.TrimSpace(fmt.Sprint(comfySettings["checkpoint"]))
		comfyOutputNodeID := strings.TrimSpace(fmt.Sprint(comfySettings["output_node_id"]))
		comfyFilenamePrefix := strings.TrimSpace(fmt.Sprint(comfySettings["filename_prefix"]))
		if provider == "" {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "image provider 不能为空"})
			return
		}
		if err := updateConfigFile(func(cfgMap map[string]interface{}) error {
			imageCfg := ensureConfigMap(cfgMap, "image")
			comfyCfg := ensureConfigMap(imageCfg, "comfyui")
			imageCfg["provider"] = provider
			imageCfg["width"] = width
			imageCfg["height"] = height
			imageCfg["steps"] = steps
			imageCfg["cfg_scale"] = cfgScale
			imageCfg["negative_prompt"] = negativePrompt
			comfyCfg["api_url"] = comfyAPIURL
			comfyCfg["checkpoint"] = comfyCheckpoint
			comfyCfg["output_node_id"] = comfyOutputNodeID
			comfyCfg["filename_prefix"] = comfyFilenamePrefix
			return nil
		}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": fmt.Sprintf("保存图像配置失败: %v", err)})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "success", "message": "图像配置已保存"})

	default:
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "未知的设置类型"})
	}
}

// 调试全局样式处理函数
func debugGlobalStyleHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"global_style":      GlobalStyle,
		"additional_prompt": AdditionalPrompt,
	})
}
