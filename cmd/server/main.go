package main

import (
	"github.com/mcp2rest/internal/config"
	"github.com/mcp2rest/internal/logging"
	"log"
)

func main() {
	// 初始化日志
	if err := logging.InitLogger(); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	logging.Logger.Println("Logger initialized successfully")

	// 读取配置文件
	cfg, err1, err2 := config.LoadServerConfig("config_path_placeholder")
	if err1 != nil || err2 != nil {
		logging.Logger.Printf("Failed to load config: %v, %v", err1, err2)
		log.Fatalf("Failed to load config: %v, %v", err1, err2)
	}
	logging.Logger.Printf("Config loaded: mod=%s", cfg.Mod)

	// 根据 mod 启动服务
	if cfg.Mod == "stdio" {
		logging.Logger.Println("Starting server in stdio mode")
		// 启动 stdio 模式
	} else {
		logging.Logger.Println("Starting server in websocket mode")
		// 启动 websocket 模式
	}
}