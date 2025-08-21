package openapi

import (
	"github.com/mcp2rest/internal/config"
)

// Loader 实现了config.OpenAPILoader接口
type Loader struct{}

// LoadFromOpenAPI 从OpenAPI规范文件加载端点配置
func (l *Loader) LoadFromOpenAPI(filePath string) ([]config.EndpointConfig, error) {
	// 解析OpenAPI规范
	spec, err := ParseOpenAPISpec(filePath)
	if err != nil {
		return nil, err
	}

	// 转换为端点配置
	endpoints := ConvertToEndpoints(spec)
	return endpoints, nil
}

// RegisterLoader 注册OpenAPI加载器
func RegisterLoader() {
	loader := &Loader{}
	config.RegisterOpenAPILoader(loader)
}