package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"

	configpkg "novel-video-workflow/pkg/config"
	"novel-video-workflow/pkg/mcp"

	"go.uber.org/zap"
)

func loadBridgeConfig(configPath string) (configpkg.Config, error) {
	return configpkg.LoadConfig(configPath)
}

func main() {
	// 命令行参数
	mode := flag.String("mode", "server", "运行模式: server, list-tools, call-tool")
	toolName := flag.String("tool", "", "要调用的工具名称 (call-tool模式)")
	toolArgs := flag.String("args", "", "工具参数，JSON格式 (call-tool模式)")
	flag.Parse()

	// 初始化日志
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	if _, err := loadBridgeConfig("config.yaml"); err != nil {
		logger.Warn("加载 typed config 失败，继续启动桥接器", zap.Error(err))
	}

	// 创建MCP适配器
	adapter := mcp.NewMCPAdapter(logger)

	switch *mode {
	case "server":
		// 以MCP服务模式运行
		fmt.Println("🚀 启动MCP服务桥接器...")
		ctx := context.Background()
		if err := adapter.RunAsMCPService(ctx); err != nil {
			log.Fatalf("MCP服务运行失败: %v", err)
		}
	case "list-tools":
		// 列出可用工具
		tools := adapter.GetAvailableTools()
		fmt.Println("📋 可用MCP工具:")
		for _, tool := range tools {
			fmt.Printf("- %s\n", tool)
		}
	case "call-tool":
		// 调用特定工具
		if *toolName == "" {
			log.Fatal("使用call-tool模式时必须指定--tool参数")
		}

		// 解析工具参数
		var args map[string]interface{}
		if *toolArgs != "" {
			if err := json.Unmarshal([]byte(*toolArgs), &args); err != nil {
				log.Fatalf("解析工具参数失败: %v", err)
			}
		} else {
			args = make(map[string]interface{})
		}

		fmt.Printf("🔧 调用工具: %s\n", *toolName)
		fmt.Printf("📝 参数: %+v\n", args)

		result, err := adapter.ProcessWithOllamaDesktop(*toolName, args)
		if err != nil {
			log.Fatalf("工具调用失败: %v", err)
		}

		// 输出结果
		resultJSON, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			log.Fatalf("序列化结果失败: %v", err)
		}
		fmt.Printf("✅ 工具调用成功:\n%s\n", string(resultJSON))
	default:
		log.Fatalf("未知模式: %s，支持的模式: server, list-tools, call-tool", *mode)
	}
}