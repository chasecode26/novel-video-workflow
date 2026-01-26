package main

import (
	"context"
	"fmt"
	"net/http"
	"novel-video-workflow/cmd/web_server"
	"novel-video-workflow/pkg/mcp"
	"novel-video-workflow/pkg/tools/aegisub"
	"novel-video-workflow/pkg/tools/drawthings"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"novel-video-workflow/pkg/workflow"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func main() {
	fmt.Println("启动小说视频工作流系统...")

	// 创建用于协调服务关闭的通道
	mcpDone := make(chan bool, 1)
	webDone := make(chan bool, 1)

	// 启动 MCP 服务器
	go func() {
		runMCPModeBackground()
		mcpDone <- true
	}()

	// 启动 Web 服务器
	go func() {
		runWebModeBackground()
		webDone <- true
	}()

	fmt.Println("MCP 服务器和 Web 服务器正在后台运行...")
	fmt.Println("- MCP 服务器: 供 AI 代理和其他客户端调用")
	fmt.Println("- Web 服务器: http://localhost:8080 供用户界面操作")
	fmt.Println("按 Ctrl+C 停止服务")

	// 等待退出信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\n正在关闭服务器...")

	// 等待服务关闭完成
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
	// 检查服务可用性
	fmt.Println("正在检查服务可用性...")
	unavailableServices := runSelfCheck()
	if len(unavailableServices) > 0 {
		fmt.Printf("⚠️  以下服务不可用: %v\n", unavailableServices)
		fmt.Println("请确保相应服务已启动后再运行工作流。")
		return
	}

	// 1. 初始化日志（第一个操作，用于记录）
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// 2. 加载配置文件 - 首先尝试当前工作目录，然后尝试可执行文件目录
	var configPath string
	var err error

	// 尝试在当前工作目录查找配置文件
	wd, _ := os.Getwd()
	configPath = filepath.Join(wd, "config.yaml")

	// 如果当前工作目录没有配置文件，尝试可执行文件所在目录
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		exe, exeErr := os.Executable()
		if exeErr != nil {
			logger.Fatal("无法获取可执行文件路径", zap.Error(exeErr))
		}
		exeDir := filepath.Dir(exe)
		configPath = filepath.Join(exeDir, "config.yaml")
	}

	viper.SetConfigFile(configPath) // 关键：明确指定文件
	if err := viper.ReadInConfig(); err != nil {
		// 使用logger输出到stderr，而不是log.Fatalf或fmt.Printf
		logger.Fatal("读取配置文件失败",
			zap.String("configPath", configPath),
			zap.Error(err),
		)
	}
	// 重要：不要向stdout打印任何内容！使用logger记录到stderr。
	logger.Info("配置文件加载成功", zap.String("path", configPath))

	// 3. 创建工作流处理器和MCP服务器
	processor, err := workflow.NewProcessor(logger)
	if err != nil {
		logger.Fatal("创建工作流处理器失败", zap.Error(err))
	}

	mcpServer, err := mcp.NewServer(processor, logger)
	if err != nil {
		logger.Fatal("创建MCP服务器失败", zap.Error(err))
	}

	// 4. 启动服务器
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

// runSelfCheck 执行自检程序
func runSelfCheck() []string {
	fmt.Println("🔍 执行自检程序...")

	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Printf("创建logger失败: %v\n", err)
		return []string{"logger"}
	}
	defer logger.Sync()

	// 检查各项服务
	serviceChecks := []struct {
		name string
		fn   func() error
	}{
		{"Ollama", checkOllama},
		{"DrawThings", func() error { return checkDrawThings(logger) }},
		{"IndexTTS2", checkIndexTTS2},
		{"Aegisub脚本", checkAegisub},
		{"参考音频文件", checkRefAudio},
	}

	var unavailableServices []string
	for _, check := range serviceChecks {
		fmt.Printf("  📋 检查%s...", check.name)
		if err := check.fn(); err != nil {
			fmt.Printf(" ❌ (%v)\n", err)
			unavailableServices = append(unavailableServices, check.name)
		} else {
			fmt.Printf(" ✅\n")
		}
	}

	if len(unavailableServices) > 0 {
		fmt.Printf("⚠️  以下服务不可用: %v\n", unavailableServices)
	} else {
		fmt.Println("✅ 所有服务均正常")
	}

	return unavailableServices
}

// checkOllama 检查Ollama服务
func checkOllama() error {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("http://localhost:11434/api/tags")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("状态码: %d", resp.StatusCode)
	}

	return nil
}

// checkDrawThings 检查DrawThings服务
func checkDrawThings(logger *zap.Logger) error {
	client := drawthings.NewDrawThingsClient(logger, "http://localhost:7861")
	if !client.APIAvailable {
		return fmt.Errorf("DrawThings API不可用")
	}
	return nil
}

// checkIndexTTS2 检查IndexTTS2服务
func checkIndexTTS2() error {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("http://localhost:7860")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("状态码: %d", resp.StatusCode)
	}

	return nil
}

// checkAegisub 检查Aegisub脚本
func checkAegisub() error {
	gen := aegisub.NewAegisubGenerator()
	if _, err := os.Stat(gen.ScriptPath); os.IsNotExist(err) {
		return err
	}
	return nil
}

// checkRefAudio 检查参考音频文件
func checkRefAudio() error {
	paths := []string{
		"./assets/ref_audio/ref.m4a",
		"./assets/ref_audio/音色.m4a",
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			// 检查文件大小确保不是空文件
			info, err := os.Stat(path)
			if err != nil {
				continue
			}
			if info.Size() > 1024 { // 确保文件至少有1KB
				return nil
			}
		}
	}

	return fmt.Errorf("未找到有效的参考音频文件")
}
