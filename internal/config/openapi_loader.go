package config

import (
	"fmt"
	"path/filepath"

	"github.com/mcp2rest/internal/logging"
)

// OpenAPILoader 接口定义了从OpenAPI规范加载配置的方法
type OpenAPILoader interface {
	LoadFromOpenAPI(filePath string) (*OpenAPISpec, error)
}

var openAPILoaderInstance OpenAPILoader

// RegisterOpenAPILoader 注册OpenAPI加载器实例
func RegisterOpenAPILoader(loader OpenAPILoader) {
	openAPILoaderInstance = loader
}

// LoadOpenAPISpec 从OpenAPI规范文件加载配置
func LoadOpenAPISpec(filePath string) (*OpenAPISpec, error) {
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

// LoadConfigWithOpenAPI 加载OpenAPI规范
func LoadConfigWithOpenAPI(openAPIPath string) (*Config, *OpenAPISpec, error) {
	// 创建基础配置（使用默认值，具体配置由各版本自己加载）
	server, global := GetDefaultServerConfig()
	cfg := &Config{
		Server: *server,
		Global: *global,
	}

	// 加载OpenAPI规范
	logging.Logger.Printf("开始加载OpenAPI规范: %s", openAPIPath)
	
	openAPISpec, err := LoadOpenAPISpec(openAPIPath)
	if err != nil {
		return nil, nil, fmt.Errorf("加载OpenAPI规范 %s 失败: %w", openAPIPath, err)
	}
	
	logging.Logger.Printf("成功加载OpenAPI规范: %s", openAPIPath)

	return cfg, openAPISpec, nil
}