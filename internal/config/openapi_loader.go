package config

import (
	"github.com/mcp2rest/internal/logging"
	
	"fmt"
	"path/filepath"
)

// OpenAPILoader 接口定义了从OpenAPI规范加载端点配置的方法
type OpenAPILoader interface {
	LoadFromOpenAPI(filePath string) ([]EndpointConfig, error)
}

var openAPILoaderInstance OpenAPILoader

// RegisterOpenAPILoader 注册OpenAPI加载器实例
func RegisterOpenAPILoader(loader OpenAPILoader) {
	openAPILoaderInstance = loader
}

// LoadOpenAPISpec 从OpenAPI规范文件加载端点配置
func LoadOpenAPISpec(filePath string) ([]EndpointConfig, error) {
	if openAPILoaderInstance == nil {
		return nil, fmt.Errorf("OpenAPI加载器未注册")
	}

	// 验证文件扩展名
	ext := filepath.Ext(filePath)
	if ext != ".json" && ext != ".yaml" && ext != ".yml" {
		return nil, fmt.Errorf("不支持的OpenAPI规范文件格式: %s", ext)
	}

	return openAPILoaderInstance.LoadFromOpenAPI(filePath)
}

// LoadConfigWithOpenAPI 加载服务器配置和API配置
func LoadConfigWithOpenAPI(apiConfigPath string) (*Config, error) {
	// 1. 加载服务器配置
	// 尝试从 ./configs/server.yaml 加载
	serverConfigPath := "configs/server.yaml"
	logging.Logger.Printf("尝试加载服务器配置: %s", serverConfigPath)
	server, global, err := LoadServerConfig(serverConfigPath)
	if err != nil {
		// 如果失败，尝试从 ../configs/server.yaml 加载
		serverConfigPath = "../configs/server.yaml"
		logging.Logger.Printf("尝试从上级目录加载服务器配置: %s", serverConfigPath)
		server, global, err = LoadServerConfig(serverConfigPath)
		if err != nil {
			logging.Logger.Printf("服务器配置文件未找到，使用默认配置")
			server, global = GetDefaultServerConfig()
		} else {
			logging.Logger.Printf("服务器配置加载成功: Server=%+v, Global=%+v", server, global)
		}
	} else {
		logging.Logger.Printf("服务器配置加载成功: Server=%+v, Global=%+v", server, global)
	}

	// 创建基础配置
	cfg := &Config{
		Server:    *server,
		Global:    *global,
		Endpoints: []EndpointConfig{},
	}

	// 2. 加载API配置
	logging.Logger.Printf("开始加载API配置: %s", apiConfigPath)
	if IsOpenAPISpec(apiConfigPath) {
		// 如果是OpenAPI规范文件
		logging.Logger.Printf("检测到OpenAPI规范文件: %s", apiConfigPath)
		if openAPILoaderInstance != nil {
			endpoints, err := LoadOpenAPISpec(apiConfigPath)
			if err != nil {
				return nil, fmt.Errorf("加载OpenAPI规范 %s 失败: %w", apiConfigPath, err)
			}
			cfg.Endpoints = append(cfg.Endpoints, endpoints...)
			logging.Logger.Printf("成功加载 %d 个端点配置", len(endpoints))
		}
	} else {
		// 作为普通API配置文件加载
		endpoints, err := LoadAPIConfig(apiConfigPath)
		if err != nil {
			return nil, fmt.Errorf("加载API配置文件 %s 失败: %w", apiConfigPath, err)
		}
		cfg.Endpoints = append(cfg.Endpoints, endpoints...)
		logging.Logger.Printf("成功加载 %d 个端点配置", len(endpoints))
	}

	return cfg, nil
}