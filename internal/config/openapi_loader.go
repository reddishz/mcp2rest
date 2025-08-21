package config

import (
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

// LoadConfigWithOpenAPI 从主配置文件加载完整配置，包括OpenAPI规范
func LoadConfigWithOpenAPI(filePath string) (*Config, error) {
	// 检查文件是否为OpenAPI规范
	if IsOpenAPISpec(filePath) {
		// 如果是OpenAPI规范文件，直接加载
		server, global := GetDefaultServerConfig()
		
		// 创建完整配置
		cfg := &Config{
			Server:    *server,
			Global:    *global,
			Endpoints: []EndpointConfig{},
		}
		
		// 加载OpenAPI规范
		if openAPILoaderInstance != nil {
			endpoints, err := LoadOpenAPISpec(filePath)
			if err != nil {
				return nil, fmt.Errorf("加载OpenAPI规范 %s 失败: %w", filePath, err)
			}
			cfg.Endpoints = append(cfg.Endpoints, endpoints...)
		}
		
		return cfg, nil
	}
	
	// 检查文件是否为API配置文件
	ext := filepath.Ext(filePath)
	if ext == ".yaml" || ext == ".yml" {
		// 尝试作为API配置文件加载
		return LoadConfigFromAPIFile(filePath)
	}

	// 作为主配置文件加载
	// 加载主配置
	mainCfg, err := LoadMainConfig(filePath)
	if err != nil {
		return nil, err
	}

	// 加载服务器配置
	server, global, err := LoadServerConfig(mainCfg.ServerConfig)
	if err != nil {
		return nil, err
	}

	// 创建完整配置
	cfg := &Config{
		Server:    *server,
		Global:    *global,
		Endpoints: []EndpointConfig{},
	}

	// 加载所有API配置
	for _, apiConfigPath := range mainCfg.APIConfigs {
		endpoints, err := LoadAPIConfig(apiConfigPath)
		if err != nil {
			return nil, fmt.Errorf("加载API配置文件 %s 失败: %w", apiConfigPath, err)
		}
		cfg.Endpoints = append(cfg.Endpoints, endpoints...)
	}

	// 加载所有OpenAPI规范
	if openAPILoaderInstance != nil {
		for _, openAPIPath := range mainCfg.OpenAPISpecs {
			endpoints, err := LoadOpenAPISpec(openAPIPath)
			if err != nil {
				return nil, fmt.Errorf("加载OpenAPI规范 %s 失败: %w", openAPIPath, err)
			}
			cfg.Endpoints = append(cfg.Endpoints, endpoints...)
		}
	}

	return cfg, nil
}