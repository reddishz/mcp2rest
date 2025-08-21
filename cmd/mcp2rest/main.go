package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mcp2rest/internal/config"
	"github.com/mcp2rest/internal/logging"
	"github.com/mcp2rest/internal/openapi"
	"github.com/mcp2rest/internal/server"
)

func main() {
	// 初始化日志
	if err := logging.InitLogger(); err != nil {
		log.Fatalf("初始化日志失败: %v", err)
	}

	// 记录启动信息
	logging.Logger.Println("===== 启动 MCP2REST 服务器 =====")
	logging.Logger.Printf("进程ID: %d", os.Getpid())
	logging.Logger.Printf("父进程ID: %d", os.Getppid())
	logging.Logger.Printf("当前工作目录: %s", os.Getenv("PWD"))
	logging.Logger.Printf("环境变量 PATH: %s", os.Getenv("PATH"))
	logging.Logger.Printf("环境变量 GOPATH: %s", os.Getenv("GOPATH"))

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
	logging.Logger.Printf("配置加载成功: 模式=%s, 主机=%s, 端口=%d", cfg.Server.Mode, cfg.Server.Host, cfg.Server.Port)
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

	logging.Logger.Printf("MCP2REST 服务器已启动，模式: %s", cfg.Server.Mode)

	// 设置信号处理
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGHUP)
	
	// 等待信号或服务器停止
	select {
	case sig := <-sigCh:
		logging.Logger.Printf("收到信号: %v", sig)
		// 立即取消上下文
		srv.Cancel()
		// 给服务器一点时间优雅关闭
		logging.Logger.Println("正在关闭服务器...")
		time.Sleep(100 * time.Millisecond)
	case <-srv.Done():
		logging.Logger.Printf("服务器已停止")
	}
	
	// 强制退出进程，确保不会有残留
	logging.Logger.Println("强制退出进程")
	os.Exit(0)
}