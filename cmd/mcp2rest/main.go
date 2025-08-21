package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

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
	logging.Logger.Printf("当前工作目录: %s", os.Getenv("PWD"))
	logging.Logger.Printf("环境变量 PATH: %s", os.Getenv("PATH"))
	logging.Logger.Printf("环境变量 GOPATH: %s", os.Getenv("GOPATH"))

	// 命令行参数
	apiConfigFile := flag.String("config", "configs/api_config.yaml", "API配置文件路径")
	flag.Parse()
	logging.Logger.Printf("命令行参数: config=%s", *apiConfigFile)
	flag.Parse()

	// 注册OpenAPI加载器
	openapi.RegisterLoader()

	// 加载配置
	logging.Logger.Printf("开始加载配置文件: %s", *apiConfigFile)
	cfg, err := config.LoadConfigWithOpenAPI(*apiConfigFile)
	if err != nil {
		logging.Logger.Fatalf("加载配置失败: %v", err)
	}
	logging.Logger.Printf("配置文件加载成功: 模式=%s, 主机=%s, 端口=%d", cfg.Server.Mode, cfg.Server.Host, cfg.Server.Port)
	logging.Logger.Printf("完整配置内容:\n%+v", cfg)

	// 创建服务器
	srv, err := server.NewServer(cfg)
	if err != nil {
		log.Fatalf("创建服务器失败: %v", err)
	}

	// 启动服务器
	go func() {
		if err := srv.Start(); err != nil {
			log.Fatalf("服务器启动失败: %v", err)
		}
	}()

	logging.Logger.Printf("MCP2REST 服务器已启动，模式: %s", cfg.Server.Mode)

	// 等待中断信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Println("正在关闭服务器...")
	if err := srv.Stop(); err != nil {
		log.Fatalf("服务器关闭失败: %v", err)
	}
	fmt.Println("服务器已关闭")
}