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
	"novel-video-workflow/pkg/capcut"
	"novel-video-workflow/pkg/database"
	"novel-video-workflow/pkg/tools/aegisub"
	"novel-video-workflow/pkg/tools/file"
	"novel-video-workflow/pkg/tools/indextts2"
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
	mcp_pkg "novel-video-workflow/pkg/mcp"
	"novel-video-workflow/pkg/tools/drawthings"
	workflow_pkg "novel-video-workflow/pkg/workflow"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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

// 启动MCP服务器
func startMCPServer() error {
	// 创建logger
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// 创建工作流处理器
	processor, err := workflow_pkg.NewProcessor(logger)
	if err != nil {
		return fmt.Errorf("创建工作流处理器失败: %v", err)
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

func loadToolsList() {
	// 启动MCP服务器
	if err := startMCPServer(); err != nil {
		log.Printf("启动MCP服务器失败: %v", err)
		// 即使启动失败，也要提供默认工具列表
		fallbackToolList()
		return
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

func apiToolsHandler(c *gin.Context) {
	c.JSON(http.StatusOK, mcpTools)
}

func apiExecuteHandler(c *gin.Context) {
	var reqBody map[string]interface{} // 修改为interface{}以支持不同类型参数
	err := json.NewDecoder(c.Request.Body).Decode(&reqBody)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	toolName, ok := reqBody["toolName"].(string)
	if !ok || toolName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing toolName"})
		return
	}

	// 启动MCP工具执行
	go func() {
		// 检查工具是否存在
		toolExists := false
		for _, tool := range mcpTools {
			if tool.Name == toolName {
				toolExists = true
				break
			}
		}

		if !toolExists {
			return
		}
		// 对于generate_indextts2_audio工具，处理文本输入和音频生成
		if toolName == "generate_indextts2_audio" {
			text, ok := reqBody["text"].(string)
			if !ok || text == "" {
				text = "这是一个默认的测试文本。" // 默认文本
			}

			referenceAudio, ok := reqBody["reference_audio"].(string)
			if !ok || referenceAudio == "" {
				referenceAudio = "./assets/ref_audio/ref.m4a" // 默认参考音频
			}

			outputFile, ok := reqBody["output_file"].(string)
			if !ok || outputFile == "" {
				outputFile = fmt.Sprintf("./output/audio_%d.wav", time.Now().Unix()) // 默认输出文件
			}

			// 确保输出目录存在
			outputDir := filepath.Dir(outputFile)
			if err := os.MkdirAll(outputDir, 0755); err != nil {
				broadcast.GlobalBroadcastService.SendLog("indextts2", fmt.Sprintf("[%s] 创建输出目录失败: %v", toolName, err), broadcast.GetTimeStr())
				return
			}

			// 检查参考音频是否存在
			if _, err := os.Stat(referenceAudio); os.IsNotExist(err) {
				// 尝试其他可能的默认路径
				possiblePaths := []string{
					"./ref.m4a",
					"./音色.m4a",
					"./assets/ref_audio/ref.m4a",
					"./assets/ref_audio/音色.m4a",
				}

				found := false
				for _, path := range possiblePaths {
					if _, err := os.Stat(path); err == nil {
						referenceAudio = path
						found = true
						break
					}
				}

				if !found {
					broadcast.GlobalBroadcastService.SendLog("indextts2", fmt.Sprintf("[%s] 找不到参考音频文件，请确保存在默认音频文件", toolName), broadcast.GetTimeStr())

					return
				}
			}
			broadcast.GlobalBroadcastService.SendLog("indextts2", fmt.Sprintf("[%s] 使用参考音频: %s", toolName, referenceAudio), broadcast.GetTimeStr())
			broadcast.GlobalBroadcastService.SendLog("indextts2", fmt.Sprintf("[%s] 输入文本: %s", toolName, text), broadcast.GetTimeStr())

			// 检查MCP服务器实例是否存在
			if mcpServerInstance != nil {
				// 获取处理器并直接调用工具
				handler := mcpServerInstance.GetHandler()
				if handler != nil {
					// 创建MockRequest对象
					mockRequest := &mcp_pkg.MockRequest{
						Params: map[string]interface{}{
							"text":            text,
							"reference_audio": referenceAudio,
							"output_file":     outputFile,
						},
					}

					// 调用特定工具处理函数
					result, err := handler.HandleGenerateIndextts2AudioDirect(mockRequest)
					if err != nil {
						return
					}

					// 检查结果
					if success, ok := result["success"].(bool); ok && success {
					} else {
						errorMsg := "未知错误"
						if result["error"] != nil {
							if errStr, ok := result["error"].(string); ok {
								errorMsg = errStr
							}
						}
						broadcast.GlobalBroadcastService.SendLog("indextts2", fmt.Sprintf("[%s] 工具执行失败: %s", toolName, errorMsg), broadcast.GetTimeStr())

					}
				} else {
					broadcast.GlobalBroadcastService.SendLog("indextts2", fmt.Sprintf("[%s] 错误: MCP处理器未初始化", toolName), broadcast.GetTimeStr())

				}
			} else {
				// 如果没有MCP服务器实例，给出提示
				broadcast.GlobalBroadcastService.SendLog("indextts2", fmt.Sprintf("[%s] 错误: MCP服务器未启动。请确保服务已正确初始化。", toolName), broadcast.GetTimeStr())
			}
		} else {
			// 其他工具的处理 - 也需要类似处理
			if mcpServerInstance != nil {
				// 获取处理器并直接调用工具
				handler := mcpServerInstance.GetHandler()
				if handler != nil {
					// 根据工具名称调用相应的处理函数
					var result map[string]interface{}
					var err error

					// 为其他工具传递参数
					params, ok := reqBody["params"].(map[string]interface{})
					if !ok {
						params = make(map[string]interface{})
					}

					// 这里需要根据工具名称调用相应的处理函数
					switch toolName {
					case "generate_subtitles_from_indextts2":
						mockRequest := &mcp_pkg.MockRequest{Params: params}
						result, err = handler.HandleGenerateSubtitlesFromIndextts2Direct(mockRequest)
					case "file_split_novel_into_chapters":
						mockRequest := &mcp_pkg.MockRequest{Params: params}
						result, err = handler.HandleFileSplitNovelIntoChaptersDirect(mockRequest)
					case "generate_image_from_text":
						mockRequest := &mcp_pkg.MockRequest{Params: params}
						result, err = handler.HandleGenerateImageFromTextDirect(mockRequest)
					case "generate_image_from_image":
						mockRequest := &mcp_pkg.MockRequest{Params: params}
						result, err = handler.HandleGenerateImageFromImageDirect(mockRequest)
					case "generate_images_from_chapter_with_ai_prompt":
						// 处理章节图像生成（使用AI提示词）
						chapterText, ok := reqBody["chapter_text"].(string)
						if !ok {
							chapterText = "这是一个默认的章节文本。"
						}

						outputDir, ok := reqBody["output_dir"].(string)
						if !ok {
							outputDir = fmt.Sprintf("./output/chapter_images_%d", time.Now().Unix())
						}

						widthFloat, ok := reqBody["width"].(float64)
						var width int
						if ok {
							width = int(widthFloat)
						} else {
							width = 512 // 默认宽度
						}

						heightFloat, ok := reqBody["height"].(float64)
						var height int
						if ok {
							height = int(heightFloat)
						} else {
							height = 896 // 默认高度
						}

						// 确保输出目录存在
						if err := os.MkdirAll(outputDir, 0755); err != nil {
							return
						}

						// 创建一个自定义的日志记录器，将内部日志广播到前端
						logger, _ := zap.NewProduction()
						defer logger.Sync()

						// 使用自定义的广播日志适配器
						encoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
						writeSyncer := zapcore.AddSync(os.Stdout) // 输出到标准输出，同时也会被广播
						broadcastLogger := NewBroadcastLoggerAdapter(toolName, encoder, writeSyncer)
						broadcaster := zap.New(broadcastLogger)

						// 使用带广播功能的日志记录器创建章节图像生成器，使用当前全局样式
						generator := drawthings.NewChapterImageGeneratorWithStyle(broadcaster, database.DB, GlobalStyle)

						// 直接调用图像生成方法，而不是通过MCP处理器
						results, err := generator.GenerateImagesFromChapter(chapterText, outputDir, width, height, false)
						if err != nil {
							return
						}

						// 准备结果
						imageFiles := make([]string, len(results))
						paragraphs := make([]string, len(results))
						prompts := make([]string, len(results))

						for i, result := range results {
							imageFiles[i] = result.ImageFile
							paragraphs[i] = result.ParagraphText
							prompts[i] = result.ImagePrompt
						}

						result = map[string]interface{}{
							"success":               true,
							"output_dir":            outputDir,
							"chapter_text_length":   len(chapterText),
							"generated_image_count": len(results),
							"image_files":           imageFiles,
							"paragraphs":            paragraphs,
							"prompts":               prompts,
							"width":                 width,
							"height":                height,
							"is_suspense":           true,
							"tool":                  "drawthings_chapter_txt2img_with_ai_prompt",
						}
					case "generate_image_from_lyric_ai_prompt":
						// 处理歌词MV图像生成
						lyricText, ok := reqBody["lyric_text"].(string)
						if !ok || lyricText == "" {
							lyricText = "这是一首美丽的歌曲\n旋律悠扬动听\n歌词深情动人" // 默认歌词
						}

						outputDir, ok := reqBody["output_dir"].(string)
						if !ok || outputDir == "" {
							outputDir = fmt.Sprintf("./output/lyric_mv_%d", time.Now().Unix())
						}

						widthFloat, ok := reqBody["width"].(float64)
						var width int
						if ok {
							width = int(widthFloat)
						} else {
							width = 512 // 默认宽度
						}

						heightFloat, ok := reqBody["height"].(float64)
						var height int
						if ok {
							height = int(heightFloat)
						} else {
							height = 896 // 默认高度
						}

						// 确保输出目录存在
						if err := os.MkdirAll(outputDir, 0755); err != nil {
							broadcast.GlobalBroadcastService.SendLog("lyric", fmt.Sprintf("[%s] 创建输出目录失败: %v", toolName, err), broadcast.GetTimeStr())
							return
						}

						broadcast.GlobalBroadcastService.SendLog("lyric", fmt.Sprintf("[%s] 开始生成歌词MV图像", toolName), broadcast.GetTimeStr())
						broadcast.GlobalBroadcastService.SendLog("lyric", fmt.Sprintf("[%s] 歌词长度: %d 字符", toolName, len(lyricText)), broadcast.GetTimeStr())
						broadcast.GlobalBroadcastService.SendLog("lyric", fmt.Sprintf("[%s] 输出目录: %s", toolName, outputDir), broadcast.GetTimeStr())

						// 创建一个自定义的日志记录器，将内部日志广播到前端
						logger, _ := zap.NewProduction()
						defer logger.Sync()

						// 使用自定义的广播日志适配器
						encoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
						writeSyncer := zapcore.AddSync(os.Stdout) // 输出到标准输出，同时也会被广播
						broadcastLogger := NewBroadcastLoggerAdapter(toolName, encoder, writeSyncer)
						broadcaster := zap.New(broadcastLogger)

						// 使用带广播功能的日志记录器创建章节图像生成器，使用歌词MV风格
						generator := drawthings.NewChapterImageGeneratorWithStyle(broadcaster, database.DB, "音乐视频艺术风格")

						// 直接调用歌词图像生成方法
						results, err := generator.GenerateImagesFromLyric(lyricText, outputDir, width, height)
						if err != nil {
							broadcast.GlobalBroadcastService.SendLog("lyric", fmt.Sprintf("[%s] 歌词图像生成失败: %v", toolName, err), broadcast.GetTimeStr())
							return
						}

						// 准备结果
						imageFiles := make([]string, len(results))
						lyrics := make([]string, len(results))
						prompts := make([]string, len(results))

						for i, result := range results {
							imageFiles[i] = result.ImageFile
							lyrics[i] = result.ParagraphText
							prompts[i] = result.ImagePrompt
						}

						result = map[string]interface{}{
							"success":               true,
							"output_dir":            outputDir,
							"lyric_text_length":     len(lyricText),
							"generated_image_count": len(results),
							"image_files":           imageFiles,
							"lyrics":                lyrics,
							"prompts":               prompts,
							"width":                 width,
							"height":                height,
							"tool":                  "drawthings_lyric_mv_generator",
						}

						broadcast.GlobalBroadcastService.SendLog("lyric", fmt.Sprintf("[%s] 歌词MV图像生成完成，共生成 %d 张图像", toolName, len(results)), broadcast.GetTimeStr())

					default:
						return
					}

					if err != nil {
						return
					}

					// 记录执行结果
					broadcast.GlobalBroadcastService.SendLog("indextts2", fmt.Sprintf("[%s] 工具执行完成，结果: %+v", toolName, result), broadcast.GetTimeStr())
				} else {
				}
			} else {
			}
		}

	}()

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Tool execution started"})
}

func apiExecuteAllHandler(c *gin.Context) {
	// 执行所有MCP工具
	go func() {
		for _, tool := range mcpTools {

			// 检查MCP服务器实例是否存在
			if mcpServerInstance != nil {
				// 获取处理器并直接调用工具
				handler := mcpServerInstance.GetHandler()
				if handler != nil {
					// 根据工具名称调用相应的处理函数
					var result map[string]interface{}
					var err error

					// 为工具传递默认参数
					params := make(map[string]interface{})

					// 这里需要根据工具名称调用相应的处理函数
					switch tool.Name {
					case "generate_subtitles_from_indextts2":
						mockRequest := &mcp_pkg.MockRequest{Params: params}
						result, err = handler.HandleGenerateSubtitlesFromIndextts2Direct(mockRequest)
					case "file_split_novel_into_chapters":
						mockRequest := &mcp_pkg.MockRequest{Params: params}
						result, err = handler.HandleFileSplitNovelIntoChaptersDirect(mockRequest)
					case "generate_image_from_text":
						mockRequest := &mcp_pkg.MockRequest{Params: params}
						result, err = handler.HandleGenerateImageFromTextDirect(mockRequest)
					case "generate_image_from_image":
						mockRequest := &mcp_pkg.MockRequest{Params: params}
						result, err = handler.HandleGenerateImageFromImageDirect(mockRequest)
					case "generate_indextts2_audio":
						// 对于音频生成工具，使用默认参数
						defaultParams := map[string]interface{}{
							"text":            "这是一个测试音频。",
							"reference_audio": "./assets/ref_audio/ref.m4a",
							"output_file":     fmt.Sprintf("./output/test_%d.wav", time.Now().Unix()),
						}
						mockRequest := &mcp_pkg.MockRequest{Params: defaultParams}
						result, err = handler.HandleGenerateIndextts2AudioDirect(mockRequest)
					default:
						continue
					}

					if err != nil {
						broadcast.GlobalBroadcastService.SendLog("indextts2", err.Error(), broadcast.GetTimeStr())

						continue
					}
					//map转 json
					jsonData, _ := json.Marshal(result)
					broadcast.GlobalBroadcastService.SendLog("indextts2", string(jsonData), broadcast.GetTimeStr())

					// 记录执行结果
				} else {
				}

			} else {
				// 如果没有MCP服务器实例，给出提示
				broadcast.GlobalBroadcastService.SendLog("indextts2", "[提示] 请先启动MCP服务器！", broadcast.GetTimeStr())
			}

		}
	}()

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "All tools execution started"})
}

func apiProcessFolderHandler(c *gin.Context) {
	// 处理上传的文件夹
	go func() {
		broadcast.GlobalBroadcastService.SendLog("上传文件夹", "[工作流] 开始文件夹处理工作流...", broadcast.GetTimeStr())

		// 检查MCP服务器实例是否存在
		if mcpServerInstance != nil {
			// 获取处理器并直接调用工具
			handler := mcpServerInstance.GetHandler()
			if handler != nil {
				// 模拟工作流处理
				for _, tool := range mcpTools {
					broadcast.GlobalBroadcastService.SendLog("上传文件夹", fmt.Sprintf("[%s] 使用 %s 处理...", tool.Name, tool.Name), broadcast.GetTimeStr())

					// 根据工具名称调用相应的处理函数
					var result map[string]interface{}
					var err error

					// 为工具传递默认参数
					params := make(map[string]interface{})

					// 这里需要根据工具名称调用相应的处理函数
					switch tool.Name {
					case "generate_subtitles_from_indextts2":
						mockRequest := &mcp_pkg.MockRequest{Params: params}
						result, err = handler.HandleGenerateSubtitlesFromIndextts2Direct(mockRequest)
					case "file_split_novel_into_chapters":
						mockRequest := &mcp_pkg.MockRequest{Params: params}
						result, err = handler.HandleFileSplitNovelIntoChaptersDirect(mockRequest)
					case "generate_image_from_text":
						mockRequest := &mcp_pkg.MockRequest{Params: params}
						result, err = handler.HandleGenerateImageFromTextDirect(mockRequest)
					case "generate_image_from_image":
						mockRequest := &mcp_pkg.MockRequest{Params: params}
						result, err = handler.HandleGenerateImageFromImageDirect(mockRequest)
					case "generate_indextts2_audio":
						// 对于音频生成工具，使用默认参数
						defaultParams := map[string]interface{}{
							"text":            "这是文件夹处理的一部分。",
							"reference_audio": "./assets/ref_audio/ref.m4a",
							"output_file":     fmt.Sprintf("./output/folder_process_%d.wav", time.Now().Unix()),
						}
						mockRequest := &mcp_pkg.MockRequest{Params: defaultParams}
						result, err = handler.HandleGenerateIndextts2AudioDirect(mockRequest)
					default:
						broadcast.GlobalBroadcastService.SendLog("上传文件夹", fmt.Sprintf("[%s] 暂不支持直接调用工具: %s", tool.Name, tool.Name), broadcast.GetTimeStr())

						continue
					}

					if err != nil {
						broadcast.GlobalBroadcastService.SendLog("上传文件夹", fmt.Sprintf("[%s] 工具执行失败: %v", tool.Name, err), broadcast.GetTimeStr())

					} else {
						// 记录执行结果
						broadcast.GlobalBroadcastService.SendLog("上传完毕", fmt.Sprintf("[%s] 工具执行完成，结果: %+v", tool.Name, result), broadcast.GetTimeStr())

					}
				}
			} else {
				broadcast.GlobalBroadcastService.SendLog("[工作流] 错误", "MCP处理器未初始化", broadcast.GetTimeStr())

			}
		} else {
			broadcast.GlobalBroadcastService.SendLog("[工作流] 错误", "MCP服务器未启动。请确保服务已正确初始化", broadcast.GetTimeStr())
		}
		broadcast.GlobalBroadcastService.SendLog("处理完成", "[工作流] 文件夹处理完成", broadcast.GetTimeStr())

	}()

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Folder processing started"})
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

	// 构建允许的路径前缀
	allowedInputPrefix := filepath.Join(projectRoot, "input")
	allowedOutputPrefix := filepath.Join(projectRoot, "output")

	// 检查路径是否在允许的范围内
	isValidPath := strings.HasPrefix(cleanDir, allowedInputPrefix+"/") ||
		strings.HasPrefix(cleanDir, allowedOutputPrefix+"/") ||
		cleanDir == allowedInputPrefix ||
		cleanDir == allowedOutputPrefix

	if !isValidPath {
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

	// 构建允许的路径前缀
	allowedInputPrefix := filepath.Join(projectRoot, "input")
	allowedOutputPrefix := filepath.Join(projectRoot, "output")

	// 检查路径是否在允许的范围内
	isValidPath := strings.HasPrefix(cleanPath, allowedInputPrefix+"/") ||
		strings.HasPrefix(cleanPath, allowedOutputPrefix+"/") ||
		cleanPath == allowedInputPrefix ||
		cleanPath == allowedOutputPrefix

	if !isValidPath {
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

	// 构建允许的路径前缀
	allowedInputPrefix := filepath.Join(projectRoot, "input")
	allowedOutputPrefix := filepath.Join(projectRoot, "output")

	// 检查路径是否在允许的范围内
	isValidPath := strings.HasPrefix(cleanPath, allowedInputPrefix+"/") ||
		strings.HasPrefix(cleanPath, allowedOutputPrefix+"/") ||
		cleanPath == allowedInputPrefix ||
		cleanPath == allowedOutputPrefix

	if !isValidPath {
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
	loadToolsList()

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
	r.GET("/api/tools", apiToolsHandler)
	r.POST("/api/execute", apiExecuteHandler)
	r.POST("/api/execute-all", apiExecuteAllHandler)
	r.POST("/api/process-folder", apiProcessFolderHandler)
	r.POST("/api/one-click-film", oneClickFilmHandler)
	// 添加CapCut项目生成API端点
	r.GET("/api/capcut-project", capcutProjectHandler)

	// 提示词模板相关路由
	promptTemplateAPI := api_pkg.NewPromptTemplateAPI(mcpServerInstance.GetProcessor())
	r.GET("/api/prompt-templates", promptTemplateAPI.GetPromptTemplates)
	r.GET("/api/prompt-templates/:id", promptTemplateAPI.GetPromptTemplateByID)
	r.GET("/api/prompt-templates/category/:category", promptTemplateAPI.GetPromptTemplatesByCategory)
	r.POST("/api/prompt-templates", promptTemplateAPI.CreatePromptTemplate)
	r.PUT("/api/prompt-templates/:id", promptTemplateAPI.UpdatePromptTemplate)
	r.DELETE("/api/prompt-templates/:id", promptTemplateAPI.DeletePromptTemplate)

	// 工作流跟踪相关路由
	workflowTrackingAPI := api_pkg.NewWorkflowTrackingAPI(mcpServerInstance.GetProcessor())
	r.POST("/api/workflow/chapter/record-params", workflowTrackingAPI.RecordChapterWorkflowParams)
	r.GET("/api/workflow/chapter/:id/params", workflowTrackingAPI.GetChapterWorkflowParams)
	r.POST("/api/workflow/scene/record-params", workflowTrackingAPI.RecordSceneWorkflowParams)
	r.GET("/api/workflow/scene/:id/params", workflowTrackingAPI.GetSceneWorkflowParams)
	r.GET("/api/chapter/:id/scenes", workflowTrackingAPI.GetScenesByChapter)

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
}

var GlobalStyle = drawthings.DefaultSuspenseStyle
var AdditionalPrompt = ", " + drawthings.DefaultSuspenseStyle

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

		for idx, paragraph := range paragraphs {
			if strings.TrimSpace(paragraph) == "" {
				continue
			}

			// 记录Ollama生成图像提示词的过程
			promptGenerationStartTime := time.Now()
			optimizedPrompt, err := wp.drawThingsGen.OllamaClient.GenerateImagePrompt(paragraph, styleDesc)
			promptGenerationEndTime := time.Now()

			//手工拆分
			if err != nil {
				wp.logger.Warn("使用Ollama生成图像提示词失败，使用原始文本",
					zap.Int("paragraph_index", idx),
					zap.String("paragraph", paragraph),
					zap.Error(err))
				optimizedPrompt = paragraph + AdditionalPrompt
			}

			// 准备DrawThings请求参数
			drawThingsConfig := map[string]interface{}{
				"prompt":          optimizedPrompt,
				"width":           512,
				"height":          896,
				"negative_prompt": ",人脸特写，半身像，模糊，比例失调，原参考图背景，比例失调，缺肢",
				"steps":           8,
				"sampler_name":    "DPM++ 2M Trailing",
				"guidance_scale":  1.0,
				"batch_size":      1,
				"model":           "z_image_turbo_1.0_q6p.ckpt",
				"is_suspense":     true,
			}
			drawThingsConfigBytes, _ := json.Marshal(drawThingsConfig)

			imageFile := filepath.Join(imagesDir, fmt.Sprintf("paragraph_%02d.png", idx+1))
			wp.logger.Info(fmt.Sprintf("📸开始生成段落图像: %d", idx+1))
			err = wp.drawThingsGen.Client.GenerateImageFromTextWithDefaultTemplate(
				optimizedPrompt,
				imageFile,
				512,   // 缩小宽度
				896,   // 缩小高度
				false, // 风格已在提示词中处理
			)

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

		// 准备DrawThings请求参数
		drawThingsConfig := map[string]interface{}{
			"prompt":          sceneDesc,
			"width":           512,
			"height":          896,
			"negative_prompt": ",人脸特写，半身像，模糊，比例失调，原参考图背景，比例失调，缺肢",
			"steps":           8,
			"sampler_name":    "DPM++ 2M Trailing",
			"guidance_scale":  1.0,
			"batch_size":      1,
			"model":           "z_image_turbo_1.0_q6p.ckpt",
			"is_suspense":     true,
		}
		drawThingsConfigBytes, _ := json.Marshal(drawThingsConfig)

		// 使用分镜描述生成图像
		err = wp.drawThingsGen.Client.GenerateImageFromTextWithDefaultTemplate(
			sceneDesc,
			imageFile,
			512,   // 缩小宽度
			896,   // 缩小高度
			false, // 风格已在提示词中处理
		)

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

// 一键出片功能 - 完整工作流处理
func oneClickFilmHandler(c *gin.Context) {

	var reqBody map[string]interface{}
	err := json.NewDecoder(c.Request.Body).Decode(&reqBody)
	if err != nil {
		broadcast.GlobalBroadcastService.SendLog("movie", fmt.Sprintf("[一键出片] 解析请求体失败: %v", err), broadcast.GetTimeStr())
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": fmt.Sprintf("解析请求失败: %v", err)})
		return
	}

	// 获取提示词模板ID
	var selectedTemplate *database.PromptTemplate
	promptTemplateIDStr, ok := reqBody["prompt_template_id"].(string)
	if ok && promptTemplateIDStr != "" {
		// 将字符串ID转换为uint
		promptTemplateID, err := strconv.ParseUint(promptTemplateIDStr, 10, 32)
		if err != nil {
			broadcast.GlobalBroadcastService.SendLog("movie", fmt.Sprintf("[一键出片] 解析提示词模板ID失败: %v", err), broadcast.GetTimeStr())
		} else {
			// 从数据库获取模板
			template, err := database.GetPromptTemplateByID(database.DB, uint(promptTemplateID))
			if err != nil {
				broadcast.GlobalBroadcastService.SendLog("movie", fmt.Sprintf("[一键出片] 获取提示词模板失败: %v", err), broadcast.GetTimeStr())
			} else {
				selectedTemplate = template
				// 更新全局风格变量
				if selectedTemplate.StyleAddon != "" {
					GlobalStyle = selectedTemplate.StyleAddon
					AdditionalPrompt = ", " + selectedTemplate.StyleAddon
				}
				broadcast.GlobalBroadcastService.SendLog("movie", fmt.Sprintf("🫧🛠[一键出片] 使用提示词模板: %s", selectedTemplate.Name), broadcast.GetTimeStr())
			}
		}
	} else {
		broadcast.GlobalBroadcastService.SendLog("movie", "[一键出片] 未选择特定提示词模板，使用默认风格", broadcast.GetTimeStr())
	}

	// 获取项目根目录
	wd, err := os.Getwd()
	if err != nil {
		broadcast.GlobalBroadcastService.SendLog("movie", fmt.Sprintf("[一键出片] 获取工作目录失败: %v", err), broadcast.GetTimeStr())
		c.JSON(http.StatusOK, gin.H{"status": "error", "message": fmt.Sprintf("获取工作目录失败: %v", err)})
		return
	}

	projectRoot := wd
	if strings.HasSuffix(wd, "/cmd/web_server") {
		projectRoot = filepath.Dir(filepath.Dir(wd)) // 回退两级到项目根目录
	}

	inputDir := filepath.Join(projectRoot, "input")
	items, err := os.ReadDir(inputDir)
	if err != nil {
		broadcast.GlobalBroadcastService.SendLog("movie", fmt.Sprintf("[一键出片] ❌ 无法读取input目录: %v", err), broadcast.GetTimeStr())

		c.JSON(http.StatusOK, gin.H{"status": "error", "message": fmt.Sprintf("无法读取input目录: %v", err)})
		return
	}

	if len(items) == 0 {
		broadcast.GlobalBroadcastService.SendLog("movie", "[一键出片] ❌ input目录为空，请在input目录下放置小说文本文件", broadcast.GetTimeStr())
		c.JSON(http.StatusOK, gin.H{"status": "error", "message": "input目录为空，请在input目录下放置小说文本文件"})
		return
	}

	if len(items) != 1 {
		broadcast.GlobalBroadcastService.SendLog("movie", "[一键出片] ❌ input目录下有多个文件，请只放置一个小说目录", broadcast.GetTimeStr())
		c.JSON(http.StatusOK, gin.H{"status": "error", "message": "input目录下有多个文件，请只放置一个小说文件"})
		return
	}
	if !items[0].IsDir() {
		broadcast.GlobalBroadcastService.SendLog("movie", "[一键出片] ❌ input目录下有文件，请只放置一个小说目录", broadcast.GetTimeStr())
	}
	novelDir := filepath.Join(inputDir, items[0].Name())

	// 检查是否是 chapter_XX 格式的目录（即已经分割好的章节）
	if isChapterDirectory(items[0].Name()) {
		// 直接处理这个章节目录
		processChapterDirectory(novelDir, items[0].Name(), projectRoot, c, broadcast.GlobalBroadcastService)
	} else {
		// 原有的处理逻辑：在小说目录中寻找对应的小说文件
		novelFiles, err := os.ReadDir(novelDir)
		if err != nil {
			broadcast.GlobalBroadcastService.SendLog("movie", fmt.Sprintf("[一键出片] ❌ 无法读取小说目录 %s: %v", items[0].Name(), err), broadcast.GetTimeStr())
		}
		broadcast.GlobalBroadcastService.SendLog("movie", fmt.Sprintf(" 🔊 novelFiles 有几个: %d", len(novelFiles)), broadcast.GetTimeStr())

		if len(novelFiles) == 0 {
			broadcast.GlobalBroadcastService.SendLog("movie", fmt.Sprintf("[一键出片] ❌ 小说目录 %s 为空", items[0].Name()), broadcast.GetTimeStr())
			return
		}
		if len(novelFiles) > 1 {
			broadcast.GlobalBroadcastService.SendLog("movie", fmt.Sprintf("[一键出片] ❌ 小说目录 %s 中有多个文件，请手动处理", items[0].Name()), broadcast.GetTimeStr())
			return
		}
		novelFile := novelFiles[0]
		// 寻找与目录名匹配的.txt文件（例如 幽灵客栈/幽灵客栈.txt）
		expectedFileName := items[0].Name() + ".txt"
		if !novelFile.IsDir() && strings.EqualFold(novelFile.Name(), expectedFileName) {
			broadcast.GlobalBroadcastService.SendLog("movie", fmt.Sprintf(" 🔊 是文本，且文本名字与目录名字相同: %s", items[0].Name()), broadcast.GetTimeStr())
			absPath := filepath.Join(novelDir, novelFile.Name())
			broadcast.GlobalBroadcastService.SendLog("movie", "[一键出片] 🧪 开始测试章节编号解析功能...", broadcast.GetTimeStr())

			// 创建FileManager实例
			fm := file.NewFileManager()
			broadcast.GlobalBroadcastService.SendLog("movie", fmt.Sprintf("[一键出片] 📖 处理小说文件: %s", novelFile.Name()), broadcast.GetTimeStr())

			// 读取输入目录中的小说 - 这会生成chapter_XX目录
			_, err = fm.CreateInputChapterStructure(absPath)
			if err != nil {
				broadcast.GlobalBroadcastService.SendLog("movie", fmt.Sprintf("[一键出片] ❌ 处理小说文件失败: %v", err), broadcast.GetTimeStr())

				c.JSON(http.StatusOK, gin.H{"status": "error", "message": fmt.Sprintf("处理小说文件失败: %v", err)})
				return
			}
			/* 1.=================== 创建输出目录结构=======================================*/

			// 创建输出目录结构
			fm.CreateOutputChapterStructure(inputDir)
			broadcast.GlobalBroadcastService.SendLog("movie", fmt.Sprintf("[输出目录的名字] 📖 输出目录的名字: %v", inputDir), broadcast.GetTimeStr())

			// 重新读取novelDir目录，查找chapter_XX子目录
			newNovelFiles, err := os.ReadDir(novelDir)
			if err != nil {
				broadcast.GlobalBroadcastService.SendLog("movie", fmt.Sprintf("[一键出片] ❌ 无法重新读取小说目录 %s: %v", items[0].Name(), err), broadcast.GetTimeStr())
			}

			// 遍历新创建的chapter_XX目录进行处理
			for _, chapterFile := range newNovelFiles {
				if chapterFile.IsDir() && isChapterDirectory(chapterFile.Name()) {
					// 处理这个章节目录
					chapterDirPath := filepath.Join(novelDir, chapterFile.Name())
					processChapterDirectory(chapterDirPath, chapterFile.Name(), projectRoot, c, broadcast.GlobalBroadcastService)
				}
			}
		}
	}

	broadcast.GlobalBroadcastService.SendLog("workflow", "🎉🎉🎉🎊🎊🎊✅ 一键出片完整工作流执行完成！", broadcast.GetTimeStr())
	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "一键出片工作流执行完成"})
	return
}

// processChapterDirectory 处理已分割的章节目录
func processChapterDirectory(chapterDir, chapterName string, projectRoot string, c *gin.Context, broadcastService *broadcast.BroadcastService) {
	// 提取章节号
	re := regexp.MustCompile(`chapter_(\d+)`)
	matches := re.FindStringSubmatch(chapterName)
	if len(matches) < 2 {
		broadcastService.SendLog("movie", fmt.Sprintf("[一键出片] ❌ 无法从目录名提取章节号: %s", chapterName), broadcast.GetTimeStr())
		return
	}

	chapterNum, err := strconv.Atoi(matches[1])
	if err != nil {
		broadcastService.SendLog("movie", fmt.Sprintf("[一键出片] ❌ 章节号格式错误: %s", chapterName), broadcast.GetTimeStr())
		return
	}

	// 查找章节文本文件
	chapterTxtFile := filepath.Join(chapterDir, fmt.Sprintf("%s.txt", chapterName))
	if _, err := os.Stat(chapterTxtFile); os.IsNotExist(err) {
		broadcastService.SendLog("movie", fmt.Sprintf("[一键出片] ❌ 章节文本文件不存在: %s", chapterTxtFile), broadcast.GetTimeStr())
		return
	}

	// 读取章节内容
	content, err := os.ReadFile(chapterTxtFile)
	if err != nil {
		broadcastService.SendLog("movie", fmt.Sprintf("[一键出片] ❌ 读取章节文件失败: %v", err), broadcast.GetTimeStr())
		return
	}

	// 获取项目名（父目录名）
	projectName := filepath.Base(filepath.Dir(chapterDir))

	// 创建logger：设置为Debug级别以显示所有日志
	config := zap.NewDevelopmentConfig()
	config.Level = zap.NewAtomicLevelAt(zap.DebugLevel) // 设置为Debug级别以显示所有日志
	logger, err := config.Build()
	if err != nil {
		broadcastService.SendLog("movie", fmt.Sprintf("[一键出片] ❌ 创建logger失败: %v", err), broadcast.GetTimeStr())
		c.JSON(http.StatusOK, gin.H{"status": "error", "message": fmt.Sprintf("创建logger失败: %v", err)})
		return
	}
	defer logger.Sync()

	// 初始化各组件
	wp := &WorkflowProcessor{
		logger:        logger,
		fileManager:   file.NewFileManager(),
		ttsClient:     indextts2.NewIndexTTS2Client(logger, "http://localhost:7860"),
		aegisubGen:    aegisub.NewAegisubGenerator(),
		drawThingsGen: drawthings.NewChapterImageGeneratorWithStyle(logger, database.DB, GlobalStyle), // 使用带自定义风格的构造函数
	}

	broadcastService.SendLog("movie", fmt.Sprintf("[一键出片] 🔊 处理章节:第%d章", chapterNum), broadcast.GetTimeStr())
	outputDir := filepath.Join(projectRoot, "output", projectName)

	audioFile := filepath.Join(outputDir, fmt.Sprintf("chapter_%02d", chapterNum), fmt.Sprintf("chapter_%02d.wav", chapterNum))

	// 保存音频生成参数到数据库
	chapterParams := map[string]interface{}{
		"chapter_number":         chapterNum,
		"chapter_title":          fmt.Sprintf("第%d章", chapterNum),
		"chapter_content_length": len(content),
		"audio_generation": map[string]interface{}{
			"input_text":      string(content)[:min(len(string(content)), 1000)], // 只保存前1000个字符作为示例
			"reference_audio": "./assets/ref_audio/ref.m4a",
			"output_file":     audioFile,
			"timestamp":       time.Now().Format(time.RFC3339),
		},
	}

	// 使用参考音频文件
	refAudioPath := filepath.Join(projectRoot, "assets", "ref_audio", "ref.m4a")
	if _, err := os.Stat(refAudioPath); os.IsNotExist(err) {
		broadcastService.SendLog("voice", "[一键出片] ⚠️  未找到参考音频文件，跳过音频生成", broadcast.GetTimeStr())
	} else {
		/* 3.=================== 发送文字到TTS生成音频文件 =======================================*/
		err = wp.ttsClient.GenerateTTSWithAudio(refAudioPath, string(content), audioFile)
		if err != nil {
			broadcastService.SendLog("voice", fmt.Sprintf("[一键出片] ⚠️  音频生成失败: %v", err), broadcast.GetTimeStr())
			c.JSON(http.StatusOK, gin.H{"status": "error", "message": fmt.Sprintf("音频生成失败: %v", err)})
			return
		}

		// 音频生成成功
		broadcastService.SendLog("voice", fmt.Sprintf("[一键出片] ✅ 音频生成完成: %s", audioFile), broadcast.GetTimeStr())

		// 更新章节参数中的音频生成状态
		if audioParams, ok := chapterParams["audio_generation"].(map[string]interface{}); ok {
			audioParams["status"] = "completed"
			audioParams["output_file"] = audioFile
		}

		// 无论成功还是失败都需要关闭连接
		if wp.ttsClient.HTTPClient != nil {
			wp.ttsClient.HTTPClient.CloseIdleConnections()
		}
	}

	/* 4.=================== 生成台词/字幕保存srt文件 =======================================*/

	// 步骤3: 生成台词/字幕
	broadcastService.SendLog("aegisub", "[一键出片] 📜 步骤3 - 生成台词/字幕...", broadcast.GetTimeStr())

	subtitleFile := filepath.Join(outputDir, fmt.Sprintf("chapter_%02d", chapterNum), fmt.Sprintf("chapter_%02d.srt", chapterNum))

	if _, err := os.Stat(audioFile); err == nil {
		// 如果音频文件存在，生成字幕
		err = wp.aegisubGen.GenerateSubtitleFromIndextts2Audio(audioFile, string(content), subtitleFile)
		if err != nil {
			broadcastService.SendLog("aegisub", fmt.Sprintf("[一键出片] ⚠️  字幕生成失败: %v", err), broadcast.GetTimeStr())

		} else {
			broadcastService.SendLog("aegisub", fmt.Sprintf("[一键出片] ✅ 字幕生成完成: %s", subtitleFile), broadcast.GetTimeStr())

		}
	} else {
		broadcastService.SendLog("aegisub", "[一键出片] ⚠️  由于音频文件不存在，跳过字幕生成", broadcast.GetTimeStr())

	}
	broadcastService.SendLog("drawthings", "[一键出片] 🎨 步骤4 - 生成图像...", broadcast.GetTimeStr())

	// 步骤4: 生成图像
	imagesDir := filepath.Join(outputDir, fmt.Sprintf("chapter_%02d", chapterNum))
	if err := os.MkdirAll(imagesDir, 0755); err != nil {
		broadcastService.SendLog("image", fmt.Sprintf("[一键出片] ❌ 创建图像目录失败: %v", err), broadcast.GetTimeStr())
		c.JSON(http.StatusOK, gin.H{"status": "error", "message": fmt.Sprintf("创建图像目录失败: %v", err)})
		return
	}

	// 查找或创建项目
	var project database.Project
	result := database.DB.Where("name = ?", projectName).First(&project)
	if result.Error != nil {
		// 如果项目不存在，创建新项目
		project = database.Project{
			Name:        projectName,
			Description: fmt.Sprintf("项目%s的一键出片工作流", projectName),
		}
		database.DB.Create(&project)
	}

	// 更新或者创建章节
	var tempChapter database.Chapter
	result = database.DB.Where("project_id = ? AND number = ?", project.ID, chapterNum).First(&tempChapter)
	if result.Error != nil {
		tempChapter := database.Chapter{
			Title:          fmt.Sprintf("第%d章", chapterNum),
			Content:        string(content)[:min(len(string(content)), 1000)], // 只保存部分内容作为示例
			ProjectID:      project.ID,
			WorkflowParams: "{}", // 临时值
		}
		tempChapter.WorkflowParams = "{}"
		database.DB.Save(&tempChapter)
	} else {
		newChapter := database.Chapter{
			Title:          fmt.Sprintf("第%d章", chapterNum),
			Content:        string(content)[:min(len(string(content)), 1000)], // 只保存部分内容作为示例
			ProjectID:      project.ID,
			WorkflowParams: "{}", // 临时值
		}
		database.DB.Create(&newChapter)
	}

	// 估算音频时长用于分镜生成
	estimatedAudioDuration := 0
	if _, statErr := os.Stat(audioFile); statErr == nil {
		// 基于音频文件大小估算时长
		if fileInfo, err := os.Stat(audioFile); err == nil {
			fileSizeMB := float64(fileInfo.Size()) / (1024 * 1024)
			// 假设平均 1MB ≈ 10秒音频
			estimatedAudioDuration = int(fileSizeMB * 10)
			if estimatedAudioDuration < 30 { // 最少30秒
				estimatedAudioDuration = 30
			}
		}
	} else {
		// 如果没有音频文件，基于文本长度估算
		estimatedAudioDuration = len(content) * 2 / 10 // 每个字符约0.2秒
		if estimatedAudioDuration < 60 {               // 最少1分钟
			estimatedAudioDuration = 60
		}
	}

	// 使用Ollama优化的提示词生成图像
	broadcastService.SendLog("image", fmt.Sprintf("[一键出片] 开始生成图像，章节ID: %d", tempChapter.ID), broadcast.GetTimeStr())
	err = wp.generateImagesWithOllamaPrompts(string(content), imagesDir, tempChapter.ID, estimatedAudioDuration)
	if err != nil {
		broadcastService.SendLog("image", fmt.Sprintf("[一键出片] ⚠️  图像生成失败: %v", err), broadcast.GetTimeStr())
	} else {
		broadcastService.SendLog("image", fmt.Sprintf("[一键出片] ✅ 图像生成完成，保存在: %s", imagesDir), broadcast.GetTimeStr())

		// 更新章节的图像路径
		// 获取生成的图像文件列表
		imageFiles, err := wp.getImageFilesFromDir(imagesDir)
		if err == nil {
			imageFilesJSON, _ := json.Marshal(imageFiles)
			err = wp.updateChapterImages(tempChapter.ID, string(imageFilesJSON))
			if err != nil {
				wp.logger.Error("更新章节图像路径失败", zap.Uint("chapter_id", tempChapter.ID), zap.Error(err))
			}
		}
	}
}

// capcutProjectHandler 生成剪映项目
func capcutProjectHandler(c *gin.Context) {
	chapterPath := c.Query("chapter_path")

	if chapterPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing chapter_path parameter", "status": "error"})
		return
	}

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

	// 构建实际路径
	var actualPath string
	if strings.HasPrefix(chapterPath, "./") {
		actualPath = filepath.Join(projectRoot, chapterPath[2:]) // 移除开头的"./"
	} else {
		actualPath = filepath.Join(projectRoot, chapterPath)
	}

	// 确保路径安全，防止路径遍历攻击
	cleanPath := filepath.Clean(actualPath)

	// 构建允许的路径前缀
	allowedInputPrefix := filepath.Join(projectRoot, "input")
	allowedOutputPrefix := filepath.Join(projectRoot, "output")

	// 检查路径是否在允许的范围内
	isValidPath := strings.HasPrefix(cleanPath, allowedInputPrefix+"/") ||
		strings.HasPrefix(cleanPath, allowedOutputPrefix+"/") ||
		cleanPath == allowedInputPrefix ||
		cleanPath == allowedOutputPrefix

	if !isValidPath {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied", "status": "error"})
		return
	}

	// 检查目录是否存在
	if _, err := os.Stat(cleanPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chapter directory does not exist", "status": "error"})
		return
	}

	// 提取项目名称（从路径中提取小说名和章节号）
	// 例如: /path/to/output/小说名/chapter_01 -> 小说名_第01章
	relativePath, err := filepath.Rel(projectRoot, cleanPath)
	if err != nil {
		broadcast.GlobalBroadcastService.SendLog("capcut", fmt.Sprintf("[CapCut] 解析相对路径失败: %v", err), broadcast.GetTimeStr())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法解析路径", "status": "error"})
		return
	}

	// 从路径中提取小说名和章节号
	pathParts := strings.Split(relativePath, string(filepath.Separator))
	var projectName string
	if len(pathParts) >= 2 {
		novelName := pathParts[len(pathParts)-2]   // 倒数第二部分是小说名
		chapterName := pathParts[len(pathParts)-1] // 最后一部分是章节名
		projectName = fmt.Sprintf("%s_%s", novelName, chapterName)
	} else {
		projectName = filepath.Base(cleanPath)
	}

	// 启动 goroutine 生成 CapCut 项目
	go func() {
		broadcast.GlobalBroadcastService.SendLog("capcut", fmt.Sprintf("[CapCut] 开始生成剪映项目，路径: %s, 项目名: %s", cleanPath, projectName), broadcast.GetTimeStr())

		capcutGenerator := capcut.NewCapcutGenerator(nil) // 传递logger或nil
		err := capcutGenerator.GenerateAndImportProject(cleanPath, projectName)
		if err != nil {
			broadcast.GlobalBroadcastService.SendLog("capcut", fmt.Sprintf("[CapCut] 生成失败: %v", err), broadcast.GetTimeStr())
		} else {
			broadcast.GlobalBroadcastService.SendLog("capcut", fmt.Sprintf("[CapCut] 项目生成并导入完成: %s", projectName), broadcast.GetTimeStr())
		}
	}()

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "CapCut project generation started"})
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
			// 可以在这里处理各种通用设置
			// 例如：图像尺寸、质量、线程数等
			imageWidth, _ := generalSettings["image_width"]
			imageHeight, _ := generalSettings["image_height"]
			imageQuality, _ := generalSettings["image_quality"]
			threadCount, _ := generalSettings["thread_count"]

			// 在这里可以添加对这些设置的处理逻辑
			fmt.Printf("通用设置已更新: Width=%v, Height=%v, Quality=%v, Threads=%v\n",
				imageWidth, imageHeight, imageQuality, threadCount)
		}
		c.JSON(http.StatusOK, gin.H{"status": "success", "message": "通用设置已保存"})

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
