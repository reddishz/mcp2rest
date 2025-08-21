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

// OpenAPISpec 表示OpenAPI规范
type OpenAPISpec struct {
	OpenAPI    string                 `json:"openapi" yaml:"openapi"`
	Info       OpenAPIInfo            `json:"info" yaml:"info"`
	Servers    []OpenAPIServer        `json:"servers" yaml:"servers"`
	Paths      map[string]PathItem    `json:"paths" yaml:"paths"`
	Components OpenAPIComponents      `json:"components" yaml:"components"`
	Security   []map[string][]string  `json:"security" yaml:"security"`
}

// OpenAPIInfo 表示OpenAPI信息
type OpenAPIInfo struct {
	Title       string `json:"title" yaml:"title"`
	Description string `json:"description" yaml:"description"`
	Version     string `json:"version" yaml:"version"`
}

// OpenAPIServer 表示OpenAPI服务器
type OpenAPIServer struct {
	URL         string `json:"url" yaml:"url"`
	Description string `json:"description" yaml:"description"`
}

// PathItem 表示路径项
type PathItem map[string]Operation

// Operation 表示操作
type Operation struct {
	Summary     string                 `json:"summary" yaml:"summary"`
	Description string                 `json:"description" yaml:"description"`
	OperationID string                 `json:"operationId" yaml:"operationId"`
	Tags        []string               `json:"tags" yaml:"tags"`
	Parameters  []Parameter            `json:"parameters" yaml:"parameters"`
	RequestBody RequestBody            `json:"requestBody" yaml:"requestBody"`
	Responses   map[string]Response    `json:"responses" yaml:"responses"`
	Security    []map[string][]string  `json:"security" yaml:"security"`
}

// Parameter 表示参数
type Parameter struct {
	Name        string      `json:"name" yaml:"name"`
	In          string      `json:"in" yaml:"in"`
	Description string      `json:"description" yaml:"description"`
	Required    bool        `json:"required" yaml:"required"`
	Schema      Schema      `json:"schema" yaml:"schema"`
	Example     interface{} `json:"example" yaml:"example"`
}

// RequestBody 表示请求体
type RequestBody struct {
	Description string               `json:"description" yaml:"description"`
	Required    bool                 `json:"required" yaml:"required"`
	Content     map[string]MediaType `json:"content" yaml:"content"`
}

// MediaType 表示媒体类型
type MediaType struct {
	Schema Schema `json:"schema" yaml:"schema"`
}

// Schema 表示模式
type Schema struct {
	Type       string                 `json:"type" yaml:"type"`
	Format     string                 `json:"format" yaml:"format"`
	Properties map[string]Schema      `json:"properties" yaml:"properties"`
	Required   []string               `json:"required" yaml:"required"`
	Items      *Schema                `json:"items" yaml:"items"`
	Ref        string                 `json:"$ref" yaml:"$ref"`
}

// Response 表示响应
type Response struct {
	Description string               `json:"description" yaml:"description"`
	Content     map[string]MediaType `json:"content" yaml:"content"`
}

// OpenAPIComponents 表示组件
type OpenAPIComponents struct {
	Schemas         map[string]Schema         `json:"schemas" yaml:"schemas"`
	SecuritySchemes map[string]SecurityScheme `json:"securitySchemes" yaml:"securitySchemes"`
}

// SecurityScheme 表示安全方案
type SecurityScheme struct {
	Type   string `json:"type" yaml:"type"`
	Scheme string `json:"scheme" yaml:"scheme"`
	Name   string `json:"name" yaml:"name"`
	In     string `json:"in" yaml:"in"`
}

// ParseOpenAPISpec 解析OpenAPI规范文件
func ParseOpenAPISpec(filePath string) (*OpenAPISpec, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("读取OpenAPI规范文件失败: %w", err)
	}

	var spec OpenAPISpec
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

// ConvertToEndpoints 将OpenAPI规范转换为端点配置
func ConvertToEndpoints(spec *OpenAPISpec) []config.EndpointConfig {
	var endpoints []config.EndpointConfig

	// 获取基础URL
	baseURL := ""
	if len(spec.Servers) > 0 {
		baseURL = spec.Servers[0].URL
	}

	// 处理安全方案
	securitySchemes := make(map[string]SecurityScheme)
	if spec.Components.SecuritySchemes != nil {
		securitySchemes = spec.Components.SecuritySchemes
	}

	// 处理路径
	for path, pathItem := range spec.Paths {
		for method, operation := range pathItem {
			// 跳过非HTTP方法的字段
			if !isHTTPMethod(method) {
				continue
			}

			// 创建端点配置
			endpoint := config.EndpointConfig{
				Name:        operation.OperationID,
				Description: operation.Description,
				Method:      strings.ToUpper(method),
				URLTemplate: baseURL + path,
			}

			// 处理参数
			var parameters []config.ParameterConfig
			for _, param := range operation.Parameters {
				parameter := config.ParameterConfig{
					Name:        param.Name,
					Required:    param.Required,
					Description: param.Description,
					In:          param.In,
				}
				parameters = append(parameters, parameter)
			}

			// 处理请求体参数
			if operation.RequestBody.Content != nil {
				for _, mediaType := range operation.RequestBody.Content {
					if mediaType.Schema.Properties != nil {
						for name, schema := range mediaType.Schema.Properties {
							required := false
							for _, req := range mediaType.Schema.Required {
								if req == name {
									required = true
									break
								}
							}
							parameter := config.ParameterConfig{
								Name:        name,
								Required:    required,
								Description: schema.Description,
								In:          "body",
							}
							parameters = append(parameters, parameter)
						}
					}
				}
			}

			endpoint.Parameters = parameters

			// 处理响应
			responseConfig := config.ResponseConfig{
				SuccessCode: 200,
				ErrorCodes:  make(map[int]string),
				Transform: config.TransformConfig{
					Type: "direct",
				},
			}

			for code, response := range operation.Responses {
				codeInt := 0
				if code == "default" {
					codeInt = 200
				} else {
					fmt.Sscanf(code, "%d", &codeInt)
				}

				if codeInt >= 200 && codeInt < 300 {
					responseConfig.SuccessCode = codeInt
				} else {
					responseConfig.ErrorCodes[codeInt] = response.Description
				}
			}

			endpoint.Response = responseConfig

			// 处理身份验证
			if len(operation.Security) > 0 {
				for _, securityReq := range operation.Security {
					for scheme := range securityReq {
						if securityScheme, ok := securitySchemes[scheme]; ok {
							authConfig := config.AuthConfig{}

							switch securityScheme.Type {
							case "apiKey":
								authConfig.Type = "api_key"
								authConfig.HeaderName = securityScheme.Name
								authConfig.KeyEnv = fmt.Sprintf("%s_API_KEY", strings.ToUpper(scheme))
							case "http":
								if securityScheme.Scheme == "bearer" {
									authConfig.Type = "bearer"
									authConfig.TokenEnv = fmt.Sprintf("%s_TOKEN", strings.ToUpper(scheme))
								} else if securityScheme.Scheme == "basic" {
									authConfig.Type = "basic"
									authConfig.Username = ""
									authConfig.Password = ""
								}
							case "oauth2":
								authConfig.Type = "oauth2"
								authConfig.TokenEnv = fmt.Sprintf("%s_TOKEN", strings.ToUpper(scheme))
							}

							endpoint.Authentication = authConfig
							break
						}
					}
				}
			}

			endpoints = append(endpoints, endpoint)
		}
	}

	return endpoints
}

// isHTTPMethod 检查字符串是否为HTTP方法
func isHTTPMethod(method string) bool {
	method = strings.ToUpper(method)
	return method == "GET" || method == "POST" || method == "PUT" || method == "DELETE" ||
		method == "PATCH" || method == "HEAD" || method == "OPTIONS" || method == "TRACE"
}