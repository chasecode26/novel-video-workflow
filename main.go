package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"novel-video-workflow/cmd/web_server"
	configpkg "novel-video-workflow/pkg/config"
	"novel-video-workflow/pkg/healthcheck"
	"novel-video-workflow/pkg/mcp"
	"novel-video-workflow/pkg/providers"
	"novel-video-workflow/pkg/workflow"

	"go.uber.org/zap"
)

func main() {
	fmt.Println("启动小说视频工作流系统...")

	mcpDone := make(chan bool, 1)
	webDone := make(chan bool, 1)

	go func() {
		runMCPModeBackground()
		mcpDone <- true
	}()

	go func() {
		runWebModeBackground()
		webDone <- true
	}()

	fmt.Println("MCP 服务器和 Web 服务器正在后台运行...")
	fmt.Println("- MCP 服务器: 供 AI 代理和其他客户端调用")
	fmt.Println("- Web 服务器: http://localhost:8080 供用户界面操作")
	fmt.Println("按 Ctrl+C 停止服务")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\n正在关闭服务器...")

	select {
	case <-mcpDone:
	case <-time.After(5 * time.Second):
		fmt.Println("MCP 服务器关闭超时")
	}

	select {
	case <-webDone:
	case <-time.After(5 * time.Second):
		fmt.Println("Web 服务器关闭超时")
	}

	fmt.Println("服务器已关闭")
}

func runMCPModeBackground() {
	fmt.Println("启动 MCP 服务器模式...")

	logger, _ := zap.NewProduction()
	defer logger.Sync()

	cfg, err := loadAppConfig()
	if err != nil {
		logger.Fatal("加载配置文件失败", zap.Error(err))
	}

	bundle, err := providers.BuildProviders(cfg)
	if err != nil {
		logger.Fatal("构建 providers 失败", zap.Error(err))
	}

	report := healthcheck.NewService(bundle).Run()
	for _, result := range report.Results {
		logger.Info("provider health check",
			zap.String("provider", result.Provider),
			zap.String("severity", string(result.Severity)),
			zap.String("message", result.Message),
		)
	}
	if !report.CanStart {
		logger.Fatal("启动前健康检查失败", zap.Int("blockingCount", len(report.Blocking)))
	}

	processor, err := workflow.NewProcessor(cfg, bundle, logger)
	if err != nil {
		logger.Fatal("创建工作流处理器失败", zap.Error(err))
	}

	mcpServer, err := mcp.NewServer(processor, logger)
	if err != nil {
		logger.Fatal("创建MCP服务器失败", zap.Error(err))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := mcpServer.Start(ctx); err != nil {
		logger.Fatal("MCP服务器启动失败", zap.Error(err))
	}
}

func runWebModeBackground() {
	fmt.Println("启动 Web 服务器模式...")
	web_server.StartServer()
}

func loadAppConfig() (configpkg.Config, error) {
	configPath, err := resolveConfigPath()
	if err != nil {
		return configpkg.Config{}, err
	}
	return configpkg.LoadConfig(configPath)
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
