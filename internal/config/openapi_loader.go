package config

import (
	"github.com/mcp2rest/internal/logging"
	
	"fmt"
	"path/filepath"
	"os"
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
	// 获取可执行文件路径
	exePath, err := os.Executable()
	if err != nil {
		logging.Logger.Printf("无法获取可执行文件路径: %v", err)
		exePath = ""
	}
	
	// 尝试多个可能的服务器配置文件路径
	serverConfigPaths := []string{
		"configs/server.yaml",                    // 当前工作目录
		"../configs/server.yaml",                 // 上级目录
	}
	
	// 如果可执行文件路径可用，添加基于可执行文件的路径
	if exePath != "" {
		exeDir := filepath.Dir(exePath)
		serverConfigPaths = append(serverConfigPaths, 
			filepath.Join(exeDir, "configs/server.yaml"),           // 可执行文件同级目录
			filepath.Join(filepath.Dir(exeDir), "configs/server.yaml"), // 可执行文件上级目录
		)
	}
	
	// 如果API配置文件路径是绝对路径，也尝试基于其目录的路径
	if filepath.IsAbs(apiConfigPath) {
		apiConfigDir := filepath.Dir(apiConfigPath)
		serverConfigPaths = append(serverConfigPaths,
			filepath.Join(apiConfigDir, "server.yaml"),             // API配置文件同级目录
			filepath.Join(filepath.Dir(apiConfigDir), "server.yaml"), // API配置文件上级目录
		)
	}
	
	var server *ServerConfig
	var global *GlobalConfig
	
	// 尝试加载服务器配置
	for _, serverConfigPath := range serverConfigPaths {
		logging.Logger.Printf("尝试加载服务器配置: %s", serverConfigPath)
		server, global, err = LoadServerConfig(serverConfigPath)
		if err == nil {
			logging.Logger.Printf("服务器配置加载成功: %s", serverConfigPath)
			logging.Logger.Printf("服务器配置: Server=%+v, Global=%+v", server, global)
			break
		}
		logging.Logger.Printf("服务器配置加载失败: %s, 错误: %v", serverConfigPath, err)
	}
	
	// 如果所有路径都失败，使用默认配置
	if server == nil || global == nil {
		logging.Logger.Printf("所有服务器配置文件路径都失败，使用默认配置")
		server, global = GetDefaultServerConfig()
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