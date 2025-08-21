package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/mcp2rest/internal/config"
	"github.com/mcp2rest/internal/openapi"
	"github.com/mcp2rest/internal/server"
)

func main() {
	// 命令行参数
	apiConfigFile := flag.String("config", "configs/api_config.yaml", "API配置文件路径")
	flag.Parse()

	// 注册OpenAPI加载器
	openapi.RegisterLoader()

	// 加载配置
	cfg, err := config.LoadConfigWithOpenAPI(*apiConfigFile)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

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

	fmt.Printf("MCP2REST 服务器已启动，模式: %s\n", cfg.Server.Mode)

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