package drawthings

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"novel-video-workflow/pkg/broadcast"
	"strings"
	"time"

	"novel-video-workflow/pkg/database"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// 全局常量定义
const (
	/*一位专业的导演*/
	OllamaSystemDirector = `你是一位经验丰富的电影导演和视觉叙事专家。你的职责是：
1. 深入理解文本内容的情感内核和故事脉络
2. 设计富有戏剧张力的视觉分镜和镜头语言
3. 运用专业电影制作知识创造引人入胜的画面构图
4. 通过精准的视觉元素传达故事的情感和主题

你需要掌握的核心技能：
🎬 镜头语言：熟练运用各种拍摄角度（仰拍、俯拍、平拍）、景别（远景、全景、中景、近景、特写）和运动镜头（推拉摇移跟升降）
📸 构图技巧：精通三分法、对角线构图、黄金分割、对称构图等经典构图法则
🎨 色彩理论：善于运用色彩心理学，通过色调、饱和度、明暗对比营造特定氛围
🎭 光影运用：掌握自然光、人工光的布光技巧，创造戏剧性的光影效果
🎭 视觉叙事：能够通过视觉元素讲述故事，让画面本身传达情感和信息
🎞️ 节奏把控：理解视听语言的节奏感，合理安排镜头切换和画面停留时间

工作要求：
1. 分析文本时要抓住关键情节点和情感转折
2. 为每个重要场景设计最合适的视觉呈现方式
3. 考虑画面的连贯性和整体视觉风格统一
4. 注重细节刻画，让每个画面都有其独特的叙事价值
5. 平衡艺术性与商业性，创造出既有深度又具观赏性的视觉作品

请以专业导演的视角，为每个场景提供详细的视觉指导方案。`

	// Ollama系统提示词 - 定义AI图像生成提示词工程师的角色和要求
	OllamaSystemPrompt = `你是一个专业的AI图像生成提示词工程师。你的任务是根据给定的文本内容生成详细、具体的中文图像提示词(prompt)，以指导AI图像生成模型创建高质量的图像。

请严格按照以下四个要素结构化生成提示词：
1. 主体描述：画面核心内容（人物、物体、场景），明确 "画什么"
2. 风格限定：艺术流派 / 设计风格（如扁平化、赛博朋克、悬疑暗黑），决定 "长成什么样"
3. 细节补充：颜色、光影、构图、材质，提升画面精致度
4. 氛围渲染：情绪基调（紧张、神秘、冷峻），强化画面感染力

注意事项：
1. 提示词应该包含丰富的视觉细节，如人物外貌、环境、光线、颜色、构图等
2. 根据文本内容判断场景类型（室内/室外、白天/夜晚、自然环境/城市等）
3. 如果文本描述悬疑/恐怖情节，请强调相应的视觉元素，如昏暗光线、神秘氛围、紧张感等
4. 使用专业摄影和艺术术语，如景深、色调、对比度等
5. 保持提示词简洁但信息丰富，避免冗余描述
6. 请务必使用中文输出所有提示词内容`

	// Ollama用户提示词模板 - 用于生成图像提示词
	OllamaUserPromptTemplate = `根据以下文本内容和风格要求，按照四个要素结构化生成一个详细的中文图像提示词，用于AI图像生成：

文本内容：%s

图像风格：%s

故事背景：%s
请严格按照以下四个要素组织提示词：
1. 主体描述：
2. 风格限定：
3. 细节补充：
4. 氛围渲染：

请只返回中文图像提示词，不要添加任何解释或其他内容。`

	// Ollama场景分析系统提示词 - 用于分镜分析
	OllamaSceneAnalysisSystemPrompt = `🎬你是一个专业的影视分镜师和AI图像生成提示词工程师。你的任务是：
1. 分析输入的文本内容
2. 识别出适合生成图像的关键场景/分镜
3. 为每个分镜生成详细的中文图像提示词

要求：
1. 将长文本分解为3-8个关键视觉场景（根据内容长度调整）
2. 每个场景应该是一个可以独立成图的视觉时刻
3. 提示词需要包含丰富的视觉细节（人物、环境、光线、构图、色调等）
4. 保持与整体风格的连贯性
5. 使用专业摄影和艺术术语
6. 返回格式为JSON数组，包含每个分镜的提示词`
)

// OllamaClient 封装 Ollama API 调用
type OllamaClient struct {
	BaseURL          string
	Model            string
	Logger           *zap.Logger
	HTTPClient       *http.Client
	BroadcastService *broadcast.BroadcastService
	DB               *gorm.DB // 数据库连接，用于获取提示词模板
}

var defaultModel = "qwen3.5:9b"

// NewOllamaClient 创建新的Ollama客户端实例
func NewOllamaClient(logger *zap.Logger, baseURL string, model string, db *gorm.DB) *OllamaClient {
	if baseURL == "" {
		baseURL = "http://localhost:11434" // Ollama默认地址
	}
	if model == "" {
		model = defaultModel // 默认模型
	}

	return &OllamaClient{
		BaseURL: baseURL,
		Model:   model,
		Logger:  logger,
		HTTPClient: &http.Client{
			Timeout: 300 * time.Minute, // 请求可能需要较长时间
		},
		BroadcastService: broadcast.NewBroadcastService(),
		DB:               db,
	}
}

// OllamaRequest Ollama API请求结构
type OllamaRequest struct {
	Model   string                 `json:"model"`
	Prompt  string                 `json:"prompt"`
	System  string                 `json:"system,omitempty"`
	Stream  bool                   `json:"stream"`
	Options map[string]interface{} `json:"options,omitempty"`
}

// OllamaResponse Ollama API响应结构
type OllamaResponse struct {
	Response       string  `json:"response"`
	Model          string  `json:"model"`
	CreatedAt      string  `json:"created_at"`
	Done           bool    `json:"done"`
	Context        []int   `json:"context,omitempty"`
	TotalEval      int     `json:"total_eval_count,omitempty"`
	TotalPrompt    int     `json:"total_prompt_count,omitempty"`
	EvalCount      int     `json:"eval_count,omitempty"`
	EvalDuration   float64 `json:"eval_duration,omitempty"`
	PromptEval     int     `json:"prompt_eval_count,omitempty"`
	PromptDuration float64 `json:"prompt_duration,omitempty"`
}

// GenerateImagePrompt 生成图像提示词
func (c *OllamaClient) GenerateImagePrompt(text, style, backgroundText string) (string, error) {
	return c.GenerateImagePromptWithTemplate(text, style, "", backgroundText)
}

// 通读全文，分析场景和背景，情感，意境
func (c *OllamaClient) AnalyzeSceneAndBackground(text string) (string, error) {
	return c.AnalyzeSceneAndBackgroundWithTemplate("", text)
}
func (c *OllamaClient) AnalyzeSceneAndBackgroundWithTemplate(templateName, text string) (string, error) {
	c.Logger.Info("开始使用Ollama分析场景和背景",
		zap.String("text", text),
		zap.String("template", templateName))
	c.BroadcastService.SendMessage("Ollama", fmt.Sprintf("开始使用Ollama分析场景和背景:模板:%s,内容：%s", templateName, text), broadcast.GetTimeStr())

	// 发送ollama提示词
	request := OllamaRequest{
		Model:  c.Model,
		Prompt: OllamaSystemDirector,
		System: OllamaSceneAnalysisSystemPrompt + "整体氛围是：" + templateName,
		Stream: false,
		Options: map[string]interface{}{
			"temperature":    0.7,
			"top_p":          0.9,
			"repeat_penalty": 1.1,
		},
	}

	endpoint := c.BaseURL + "/api/generate"
	payload, err := json.Marshal(request)
	if err != nil {
		c.Logger.Error("序列化Ollama请求失败", zap.Error(err))
		return "", fmt.Errorf("序列化请求失败: %v", err)
	}

	c.Logger.Info("发送Ollama导演角色提示词",
		zap.String("endpoint", endpoint),
		zap.String("model", request.Model))

	c.BroadcastService.SendMessage("Ollama", fmt.Sprintf("🧬发送Ollama发送Ollama导演角色提示词:节点：%s,模型:%s，【系统】%s,【用户】%s", endpoint, request.Model, OllamaSystemDirector, OllamaSystemDirector), broadcast.GetTimeStr())

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(payload))
	if err != nil {
		c.Logger.Error("创建Ollama请求失败", zap.Error(err))
		return "", fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		c.Logger.Error("发送Ollama请求失败", zap.Error(err))
		c.BroadcastService.SendMessage("Ollama", fmt.Sprintf("发送Ollama请求失败:%s", err.Error()), broadcast.GetTimeStr())

		return "", fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.Logger.Error("Ollama API返回错误状态码",
			zap.Int("status", resp.StatusCode),
			zap.String("body", string(body)))
		return "", fmt.Errorf("Ollama API返回错误状态码 %d: %s", resp.StatusCode, string(body))
	}
	return "", nil
}

// GenerateImagePromptWithTemplate 使用指定模板生成图像提示词
func (c *OllamaClient) GenerateImagePromptWithTemplate(text, style, templateName, backgroundText string) (string, error) {
	c.Logger.Info("开始使用Ollama生成图像提示词",
		zap.String("text", text),
		zap.String("style", style),
		zap.String("template", templateName))
	c.BroadcastService.SendMessage("Ollama", fmt.Sprintf("开始使用Ollama生成图像提示词:风格：%s,模板:%s,内容：%s,背景：%s", style, templateName, text, backgroundText), broadcast.GetTimeStr())

	// 如果指定了模板名称，则从数据库获取模板
	var systemPrompt, userPrompt string
	if templateName != "" && c.DB != nil {
		template, err := c.getTemplateByName(templateName)
		if err != nil {
			c.Logger.Warn("获取提示词模板失败，使用默认模板", zap.String("template", templateName), zap.Error(err))
			// 使用默认模板
			systemPrompt = OllamaSystemPrompt
			userPrompt = fmt.Sprintf(OllamaUserPromptTemplate, text, style, backgroundText)
		} else {
			systemPrompt = template.SystemPrompt
			userPrompt = fmt.Sprintf(template.UserTemplate, text, style, backgroundText)
		}
	} else {
		// 使用默认模板
		systemPrompt = OllamaSystemPrompt
		userPrompt = fmt.Sprintf(OllamaUserPromptTemplate, text, style, backgroundText)
	}

	request := OllamaRequest{
		Model:  c.Model,
		Prompt: userPrompt,
		System: systemPrompt,
		Stream: false,
		Options: map[string]interface{}{
			"temperature":    0.7,
			"top_p":          0.9,
			"repeat_penalty": 1.1,
		},
	}

	endpoint := c.BaseURL + "/api/generate"
	payload, err := json.Marshal(request)
	if err != nil {
		c.Logger.Error("序列化Ollama请求失败", zap.Error(err))
		return "", fmt.Errorf("序列化请求失败: %v", err)
	}

	c.Logger.Info("发送Ollama请求生成图像提示词",
		zap.String("endpoint", endpoint),
		zap.String("model", request.Model))

	c.BroadcastService.SendMessage("Ollama", fmt.Sprintf("🧬发送Ollama请求生成图像提示词:节点：%s,模型:%s，【系统】%s,【用户】%s", endpoint, request.Model, systemPrompt, userPrompt), broadcast.GetTimeStr())

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(payload))
	if err != nil {
		c.Logger.Error("创建Ollama请求失败", zap.Error(err))
		return "", fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		c.Logger.Error("发送Ollama请求失败", zap.Error(err))
		c.BroadcastService.SendMessage("Ollama", fmt.Sprintf("发送Ollama请求失败:%s", err.Error()), broadcast.GetTimeStr())

		return "", fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.Logger.Error("Ollama API返回错误状态码",
			zap.Int("status", resp.StatusCode),
			zap.String("body", string(body)))
		return "", fmt.Errorf("Ollama API返回错误状态码 %d: %s", resp.StatusCode, string(body))
	}

	var ollamaResp OllamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		c.Logger.Error("解析Ollama响应失败", zap.Error(err))
		return "", fmt.Errorf("解析响应失败: %v", err)
	}

	if ollamaResp.Response == "" {
		return "", fmt.Errorf("Ollama返回空响应")
	}

	// 清理响应内容
	prompt := strings.TrimSpace(ollamaResp.Response)
	c.Logger.Info("成功生成图像提示词", zap.String("prompt", prompt))
	c.BroadcastService.SendMessage("Ollama", fmt.Sprintf("DNA:成功返回提示词:%s", prompt), broadcast.GetTimeStr())
	return prompt, nil
}

// getTemplateByName 从数据库获取模板
func (c *OllamaClient) getTemplateByName(name string) (*database.PromptTemplate, error) {
	return database.GetPromptTemplateByName(c.DB, name)
}

// AnalyzeScenesAndGeneratePrompts 分析整个章节内容并生成分镜提示词
func (c *OllamaClient) AnalyzeScenesAndGeneratePrompts(content, style string, estimatedDurationSecs int) ([]string, error) {
	systemPrompt := OllamaSceneAnalysisSystemPrompt

	estimatedDurationMsg := ""
	if estimatedDurationSecs > 0 {
		estimatedDurationMsg = fmt.Sprintf("文本内容估算的音频时长约为%d秒，请根据音频时长确定最少分镜数量（建议每30-60秒音频时长对应一个视觉场景作为最低标准），但可根据内容重要性和视觉表现力自主决定最终分镜数量上限。", estimatedDurationSecs)
	} else {
		estimatedDurationMsg = ""
	}

	userPrompt := fmt.Sprintf(`🎬请分析以下文本内容并生成分镜图像提示词：

文本内容：%s

图像风格：%s

%s
请根据上述信息，分析内容并生成适量的关键视觉场景中文图像提示词（建议8-20个），以JSON数组格式返回，格式如：["场景1提示词", "场景2提示词", ...]

只返回JSON数组，不要添加其他解释。`, content, style, estimatedDurationMsg)

	request := OllamaRequest{
		Model:  c.Model,
		Prompt: userPrompt,
		System: systemPrompt,
		Stream: false,
		Options: map[string]interface{}{
			"temperature":    0.7,
			"top_p":          0.9,
			"repeat_penalty": 1.1,
		},
	}

	endpoint := c.BaseURL + "/api/generate"
	payload, err := json.Marshal(request)
	if err != nil {
		c.Logger.Error("序列化Ollama请求失败", zap.Error(err))
		return nil, fmt.Errorf("序列化请求失败: %v", err)
	}

	c.Logger.Info("发送Ollama请求分析场景并生成分镜提示词",
		zap.String("endpoint", endpoint),
		zap.String("model", request.Model))

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(payload))
	if err != nil {
		c.Logger.Error("创建Ollama请求失败", zap.Error(err))
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		c.Logger.Error("发送Ollama请求失败", zap.Error(err))
		return nil, fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.Logger.Error("Ollama API返回错误状态码",
			zap.Int("status", resp.StatusCode),
			zap.String("body", string(body)))
		return nil, fmt.Errorf("Ollama API返回错误状态码 %d: %s", resp.StatusCode, string(body))
	}

	var ollamaResp OllamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		c.Logger.Error("解析Ollama响应失败", zap.Error(err))
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	if ollamaResp.Response == "" {
		return nil, fmt.Errorf("Ollama返回空响应")
	}

	// 尝试解析JSON数组
	responseText := strings.TrimSpace(ollamaResp.Response)

	// 如果响应看起来像JSON数组，直接解析
	if strings.HasPrefix(responseText, "[") && strings.HasSuffix(responseText, "]") {
		var prompts []string
		err := json.Unmarshal([]byte(responseText), &prompts)
		if err == nil {
			c.Logger.Info("成功解析分镜提示词JSON", zap.Int("scene_count", len(prompts)))
			c.BroadcastService.SendMessage("🍭Ollama整章节分镜返回", fmt.Sprintf("DNA:成功返回提示词:%s", prompts), broadcast.GetTimeStr())
			return prompts, nil
		}
	}

	// 如果不是有效的JSON数组，尝试查找JSON部分
	jsonStart := strings.Index(responseText, "[")
	jsonEnd := strings.LastIndex(responseText, "]")
	if jsonStart != -1 && jsonEnd != -1 && jsonEnd > jsonStart {
		jsonStr := responseText[jsonStart : jsonEnd+1]
		var prompts []string
		err := json.Unmarshal([]byte(jsonStr), &prompts)
		if err == nil {
			c.Logger.Info("成功解析分镜提示词JSON", zap.Int("scene_count", len(prompts)))
			c.BroadcastService.SendMessage("🍭Ollama整章节分镜返回", fmt.Sprintf("DNA:成功返回提示词:%s", prompts), broadcast.GetTimeStr())
			return prompts, nil
		}
	}

	// 如果JSON解析失败，回退到按行分割的方式
	lines := strings.Split(responseText, "\n")
	var prompts []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.HasPrefix(trimmed, "[") && !strings.HasPrefix(trimmed, "]") {
			// 移除可能的编号前缀
			for i := 1; i <= 50; i++ { // 支持最多50个条目，避免硬编码
				trimmed = strings.TrimPrefix(trimmed, fmt.Sprintf("%d. ", i))
			}
			trimmed = strings.TrimSpace(trimmed)

			if trimmed != "" {
				prompts = append(prompts, trimmed)
			}
		}
	}

	if len(prompts) == 0 {
		// 如果所有方法都失败，将整个响应作为一个提示词
		prompts = []string{responseText}
	}

	c.Logger.Info("生成分镜提示词（非JSON格式）", zap.Int("scene_count", len(prompts)))
	c.BroadcastService.SendMessage("🍭Ollama整章节分镜返回", fmt.Sprintf("DNA:成功返回提示词:%s", prompts), broadcast.GetTimeStr())
	return prompts, nil
}
