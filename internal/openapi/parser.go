package openapi

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/mcp2rest/internal/config"
	"gopkg.in/yaml.v3"
)

// Loader 实现 OpenAPI 加载器接口
type Loader struct{}

// NewLoader 创建新的 OpenAPI 加载器
func NewLoader() *Loader {
	return &Loader{}
}

// LoadFromOpenAPI 从 OpenAPI 规范文件加载配置
func (l *Loader) LoadFromOpenAPI(filePath string) (*config.OpenAPISpec, error) {
	return ParseOpenAPISpec(filePath)
}

// ParseOpenAPISpec 解析OpenAPI规范文件
func ParseOpenAPISpec(filePath string) (*config.OpenAPISpec, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("读取OpenAPI规范文件失败: %w", err)
	}

	var spec config.OpenAPISpec
	ext := strings.ToLower(filepath.Ext(filePath))
	if ext == ".json" {
		if err := json.Unmarshal(data, &spec); err != nil {
			return nil, fmt.Errorf("解析JSON格式的OpenAPI规范失败: %w", err)
		}
	} else if ext == ".yaml" || ext == ".yml" {
		if err := yaml.Unmarshal(data, &spec); err != nil {
			return nil, fmt.Errorf("解析YAML格式的OpenAPI规范失败: %w", err)
		}
	} else {
		return nil, fmt.Errorf("不支持的文件格式: %s", ext)
	}

	return &spec, nil
}

// GetOperationByID 根据操作ID获取操作
func GetOperationByID(spec *config.OpenAPISpec, operationID string) (*config.Operation, string, string, error) {
	for path, pathItem := range spec.Paths {
		for method, operation := range pathItem {
			if !isHTTPMethod(method) {
				continue
			}
			
			// 如果操作有明确的 operationId，直接匹配
			if operation.OperationID == operationID {
				return &operation, strings.ToUpper(method), path, nil
			}
			
			// 如果没有 operationId，根据路径和方法生成操作ID进行匹配
			generatedID := generateOperationID(method, path)
			if generatedID == operationID {
				return &operation, strings.ToUpper(method), path, nil
			}
		}
	}
	
	return nil, "", "", fmt.Errorf("未找到操作ID为 %s 的操作", operationID)
}

// generateOperationID 根据HTTP方法和路径生成操作ID
func generateOperationID(method, path string) string {
	// 移除路径开头的斜杠
	path = strings.TrimPrefix(path, "/")
	
	// 将路径转换为驼峰命名
	parts := strings.Split(path, "/")
	var result []string
	
	for _, part := range parts {
		// 移除路径参数
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			continue
		}
		
		// 转换为驼峰命名
		if len(part) > 0 {
			result = append(result, strings.Title(part))
		}
	}
	
	// 组合方法名和路径
	operationID := strings.ToLower(method) + strings.Join(result, "")
	
	return operationID
}

// GetOperationByPathAndMethod 根据路径和方法获取操作
func GetOperationByPathAndMethod(spec *config.OpenAPISpec, path string, method string) (*config.Operation, error) {
	pathItem, exists := spec.Paths[path]
	if !exists {
		return nil, fmt.Errorf("未找到路径: %s", path)
	}
	
	operation, exists := pathItem[strings.ToLower(method)]
	if !exists {
		return nil, fmt.Errorf("未找到方法: %s", method)
	}
	
	return &operation, nil
}

// GetSecurityScheme 获取安全方案
func GetSecurityScheme(spec *config.OpenAPISpec, schemeName string) (*config.SecurityScheme, error) {
	if spec.Components.SecuritySchemes == nil {
		return nil, fmt.Errorf("未定义安全方案")
	}
	
	scheme, exists := spec.Components.SecuritySchemes[schemeName]
	if !exists {
		return nil, fmt.Errorf("未找到安全方案: %s", schemeName)
	}
	
	return &scheme, nil
}

// GetBaseURL 获取基础URL
func GetBaseURL(spec *config.OpenAPISpec) string {
	if len(spec.Servers) > 0 {
		return spec.Servers[0].URL
	}
	return ""
}

// isHTTPMethod 检查字符串是否为HTTP方法
func isHTTPMethod(method string) bool {
	method = strings.ToUpper(method)
	return method == "GET" || method == "POST" || method == "PUT" || method == "DELETE" ||
		method == "PATCH" || method == "HEAD" || method == "OPTIONS" || method == "TRACE"
}