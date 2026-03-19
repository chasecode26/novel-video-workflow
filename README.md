# novel-video-workflow

一个以 Go 实现的小说转视频工作流源码仓库。

本仓库当前 README 仅说明源码结构与开发入口，不再包含演示素材、运行产物、联系方式或宣传内容。

## 源码范围

以下目录/文件属于主要源码或开发所需配置：

- `main.go`：程序主入口
- `cmd/`：子命令与独立服务入口
- `pkg/`：核心业务实现
- `templates/`：Web 界面模板与前端静态源码
- `go.mod` / `go.sum`：Go 依赖定义
- `go.work` / `go.work.sum`：Go workspace 配置
- `config.yaml`：项目配置文件
- `generate_logo.sh`：项目脚本

## 目录说明

```text
.
├── cmd/                # 命令行入口与服务入口
├── pkg/                # 核心业务代码
├── templates/          # HTML/CSS/JS 模板
├── main.go             # 主程序入口
├── config.yaml         # 配置文件
├── go.mod              # Go module 定义
├── go.sum              # Go 依赖锁定
├── go.work             # Workspace 配置
├── go.work.sum         # Workspace 依赖校验
└── generate_logo.sh    # 辅助脚本
```

## 开发说明

### 环境要求

- Go `1.25.5` 或兼容版本

### 常见开发命令

```bash
go run main.go
```

如果需要运行特定子命令或服务入口，可查看：

- `cmd/ollama_mcp_bridge/main.go`
- `cmd/web_server/web_server.go`

## 说明

本仓库应优先保留源码、配置与开发所需脚本。

以下内容不属于核心源码范畴，后续可按需要继续清理：

- 本地产物与编译二进制
- 示例输入数据
- 数据库文件
- 截图、音视频、二维码等资源文件
- IDE 配置与系统生成垃圾文件
