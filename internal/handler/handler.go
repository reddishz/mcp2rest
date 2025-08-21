package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/mcp2rest/internal/auth"
	"github.com/mcp2rest/internal/config"
	"github.com/mcp2rest/internal/openapi"
	"github.com/mcp2rest/internal/transformer"
	"github.com/mcp2rest/pkg/mcp"
)

// RequestHandler 处理API请求
type RequestHandler struct {
	config      *config.Config
	openAPISpec *config.OpenAPISpec
	httpClient  *http.Client
	transformer *transformer.ResponseTransformer
	auth        *auth.AuthManager
}

// NewRequestHandler 创建新的请求处理器
func NewRequestHandler(cfg *config.Config, spec *config.OpenAPISpec) (*RequestHandler, error) {
	transformer, err := transformer.NewResponseTransformer()
	if err != nil {
		return nil, fmt.Errorf("创建响应转换器失败: %w", err)
	}

	authManager, err := auth.NewAuthManager()
	if err != nil {
		return nil, fmt.Errorf("创建身份验证管理器失败: %w", err)
	}

	return &RequestHandler{
		config:      cfg,
		openAPISpec: spec,
		httpClient:  &http.Client{Timeout: cfg.Global.Timeout},
		transformer: transformer,
		auth:        authManager,
	}, nil
}

// HandleRequest 处理工具调用请求
func (h *RequestHandler) HandleRequest(params *mcp.ToolCallParams) (*mcp.ToolCallResult, error) {
	// 根据操作ID查找操作
	operation, method, path, err := openapi.GetOperationByID(h.openAPISpec, params.Name)
	if err != nil {
		return nil, fmt.Errorf("查找操作失败: %w", err)
	}

	// 构建HTTP请求
	req, err := h.buildHTTPRequest(operation, method, path, params.Parameters)
	if err != nil {
		return nil, fmt.Errorf("构建HTTP请求失败: %w", err)
	}

	// 添加身份验证
	if err := h.applyAuthentication(req, operation); err != nil {
		return nil, fmt.Errorf("应用身份验证失败: %w", err)
	}

	// 添加默认头
	for key, value := range h.config.Global.DefaultHeaders {
		req.Header.Set(key, value)
	}

	// 发送请求
	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应体失败: %w", err)
	}

	// 检查状态码
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errorMsg := fmt.Sprintf("API返回错误状态码: %d", resp.StatusCode)
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			errorMsg = "客户端错误"
		} else if resp.StatusCode >= 500 {
			errorMsg = "服务器错误"
		}
		return &mcp.ToolCallResult{
			Type:   "error",
			Status: "error",
			Result: map[string]interface{}{
				"message": errorMsg,
				"code":    resp.StatusCode,
				"body":    string(body),
			},
		}, nil
	}

	// 转换响应
	result, err := h.transformer.TransformResponse(body, operation.Responses)
	if err != nil {
		return nil, fmt.Errorf("转换响应失败: %w", err)
	}

	return &mcp.ToolCallResult{
		Type:   "success",
		Status: "success",
		Result: result,
	}, nil
}

// buildHTTPRequest 构建HTTP请求
func (h *RequestHandler) buildHTTPRequest(operation *config.Operation, method, path string, params map[string]interface{}) (*http.Request, error) {
	// 获取基础URL
	baseURL := openapi.GetBaseURL(h.openAPISpec)
	if baseURL == "" {
		return nil, fmt.Errorf("OpenAPI规范中未定义服务器URL")
	}

	// 构建完整URL
	fullURL := baseURL + path

	// 处理路径参数
	for _, param := range operation.Parameters {
		if param.In == "path" {
			if value, exists := params[param.Name]; exists {
				fullURL = strings.ReplaceAll(fullURL, "{"+param.Name+"}", fmt.Sprintf("%v", value))
			} else if param.Required {
				return nil, fmt.Errorf("缺少必需的路径参数: %s", param.Name)
			}
		}
	}

	// 处理查询参数
	if method == "GET" || method == "DELETE" {
		queryParams := url.Values{}
		for _, param := range operation.Parameters {
			if param.In == "query" {
				if value, exists := params[param.Name]; exists {
					queryParams.Set(param.Name, fmt.Sprintf("%v", value))
				} else if param.Required {
					return nil, fmt.Errorf("缺少必需的查询参数: %s", param.Name)
				}
			}
		}
		if len(queryParams) > 0 {
			fullURL += "?" + queryParams.Encode()
		}
	}

	// 创建请求
	var req *http.Request
	var err error

	if method == "POST" || method == "PUT" || method == "PATCH" {
		// 处理请求体
		var body []byte
		if operation.RequestBody.Content != nil {
			// 构建请求体
			requestBody := make(map[string]interface{})
			for _, param := range operation.Parameters {
				if param.In == "body" {
					if value, exists := params[param.Name]; exists {
						requestBody[param.Name] = value
					} else if param.Required {
						return nil, fmt.Errorf("缺少必需的请求体参数: %s", param.Name)
					}
				}
			}
			
			// 如果没有从参数中获取到请求体，尝试使用整个参数对象
			if len(requestBody) == 0 && len(params) > 0 {
				requestBody = params
			}
			
			body, err = json.Marshal(requestBody)
			if err != nil {
				return nil, fmt.Errorf("序列化请求体失败: %w", err)
			}
		}
		
		req, err = http.NewRequest(method, fullURL, bytes.NewBuffer(body))
		if err != nil {
			return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
		}
		
		// 设置Content-Type
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest(method, fullURL, nil)
		if err != nil {
			return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
		}
	}

	return req, nil
}

// applyAuthentication 应用身份验证
func (h *RequestHandler) applyAuthentication(req *http.Request, operation *config.Operation) error {
	if len(operation.Security) == 0 {
		return nil // 无需身份验证
	}

	// 获取第一个安全要求
	securityReq := operation.Security[0]
	for schemeName := range securityReq {
		// 获取安全方案
		securityScheme, err := openapi.GetSecurityScheme(h.openAPISpec, schemeName)
		if err != nil {
			return fmt.Errorf("获取安全方案失败: %w", err)
		}

		// 创建认证配置
		authConfig := &config.AuthConfig{}
		switch securityScheme.Type {
		case "apiKey":
			authConfig.Type = "api_key"
			authConfig.HeaderName = securityScheme.Name
			authConfig.KeyEnv = fmt.Sprintf("%s_API_KEY", strings.ToUpper(schemeName))
		case "http":
			if securityScheme.Scheme == "bearer" {
				authConfig.Type = "bearer"
				authConfig.TokenEnv = fmt.Sprintf("%s_TOKEN", strings.ToUpper(schemeName))
			} else if securityScheme.Scheme == "basic" {
				authConfig.Type = "basic"
				authConfig.Username = ""
				authConfig.Password = ""
			}
		case "oauth2":
			authConfig.Type = "oauth2"
			authConfig.TokenEnv = fmt.Sprintf("%s_TOKEN", strings.ToUpper(schemeName))
		}

		// 应用认证
		return h.auth.ApplyAuth(req, authConfig)
	}

	return nil
}