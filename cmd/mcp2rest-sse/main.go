package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mcp2rest/internal/config"
	"github.com/mcp2rest/internal/debug"
	"github.com/mcp2rest/internal/logging"
	"github.com/mcp2rest/internal/openapi"
	"github.com/mcp2rest/internal/server"
)

func main() {
	// 自动加载 .env 文件
	if err := config.LoadEnvFileWithLog(""); err != nil {
		log.Printf("加载环境变量文件失败: %v", err)
	}

	// 初始化日志
	if err := logging.InitLogger(); err != nil {
		log.Fatalf("初始化日志失败: %v", err)
	}

	// 初始化调试模式
	debug.InitDebug()

	// 记录启动信息
	logging.Logger.Println("===== 启动 MCP2REST-SSE 服务器 =====")
	logging.Logger.Printf("进程ID: %d", os.Getpid())
	logging.Logger.Printf("父进程ID: %d", os.Getppid())
	logging.Logger.Printf("当前工作目录: %s", os.Getenv("PWD"))

	// 命令行参数
	openAPIPath := flag.String("config", "configs/bmc_api.yaml", "OpenAPI规范文件路径")
	flag.Parse()
	logging.Logger.Printf("命令行参数: config=%s", *openAPIPath)

	// 注册OpenAPI加载器
	loader := openapi.NewLoader()
	config.RegisterOpenAPILoader(loader)

	// 加载配置
	logging.Logger.Printf("开始加载OpenAPI规范: %s", *openAPIPath)
	cfg, spec, err := config.LoadConfigWithOpenAPI(*openAPIPath)
	if err != nil {
		logging.Logger.Fatalf("加载配置失败: %v", err)
	}
	
	// 加载 sse 专用服务器配置
	serverConfig, globalConfig, err := config.LoadServerConfig("configs/sse.yaml")
	if err != nil {
		logging.Logger.Fatalf("加载服务器配置失败: %v", err)
	}
	
	// 使用 sse 专用配置
	cfg.Server = *serverConfig
	cfg.Global = *globalConfig
	
	logging.Logger.Printf("配置加载成功: 主机=%s, 端口=%d", cfg.Server.Host, cfg.Server.Port)
	logging.Logger.Printf("OpenAPI规范: %s v%s", spec.Info.Title, spec.Info.Version)

	// 创建服务器
	srv, err := server.NewServer(cfg, spec)
	if err != nil {
		log.Fatalf("创建服务器失败: %v", err)
	}

	// 启动服务器
	go func() {
		if err := srv.Start(); err != nil {
			logging.Logger.Printf("服务器启动失败: %v", err)
			os.Exit(1)
		}
	}()

	logging.Logger.Printf("MCP2REST-SSE 服务器已启动在 %s:%d", cfg.Server.Host, cfg.Server.Port)

	// 设置信号处理
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	
	// 等待信号或服务器停止
	select {
	case sig := <-sigCh:
		logging.Logger.Printf("收到信号: %v (SIGTERM/SIGINT)，开始优雅关闭", sig)
		// 立即取消上下文
		srv.Cancel()
		// 给服务器一点时间优雅关闭
		logging.Logger.Println("正在关闭服务器...")
		time.Sleep(200 * time.Millisecond)
	case <-srv.Done():
		logging.Logger.Printf("服务器已停止")
	}
	
	// 强制退出进程，确保不会有残留
	logging.Logger.Println("强制退出进程")
	os.Exit(0)
}
