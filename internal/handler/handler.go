package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"strings"
	"text/template"

	"github.com/mcp2rest/internal/auth"
	"github.com/mcp2rest/internal/config"
	"github.com/mcp2rest/internal/transformer"
	"github.com/mcp2rest/pkg/mcp"
)

// RequestHandler 处理API请求
type RequestHandler struct {
	config      *config.Config
	httpClient  *http.Client
	transformer *transformer.ResponseTransformer
	auth        *auth.AuthManager
}

// NewRequestHandler 创建新的请求处理器
func NewRequestHandler(cfg *config.Config) (*RequestHandler, error) {
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
		httpClient:  &http.Client{Timeout: cfg.Global.Timeout},
		transformer: transformer,
		auth:        authManager,
	}, nil
}

// HandleRequest 处理工具调用请求
func (h *RequestHandler) HandleRequest(params *mcp.ToolCallParams) (*mcp.ToolCallResult, error) {
	// 查找端点配置
	endpoint, err := h.config.GetEndpointByName(params.Name)
	if err != nil {
		return nil, fmt.Errorf("查找端点配置失败: %w", err)
	}

	// 构建HTTP请求
	req, err := h.buildHTTPRequest(endpoint, params.Parameters)
	if err != nil {
		return nil, fmt.Errorf("构建HTTP请求失败: %w", err)
	}

	// 添加身份验证
	if err := h.auth.ApplyAuth(req, &endpoint.Authentication); err != nil {
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
	if resp.StatusCode != endpoint.Response.SuccessCode {
		errorMsg := fmt.Sprintf("API返回错误状态码: %d", resp.StatusCode)
		if errMsg, ok := endpoint.Response.ErrorCodes[resp.StatusCode]; ok {
			errorMsg = errMsg
		}
		return &mcp.ToolCallResult{
			Type:   "error",
			Status: "error",
			Result: map[string]interface{}{
				"message": errorMsg,
				"code":    resp.StatusCode,
			},
		}, nil
	}

	// 转换响应
	transformedResp, err := h.transformer.Transform(body, &endpoint.Response.Transform)
	if err != nil {
		return nil, fmt.Errorf("转换响应失败: %w", err)
	}

	// 返回结果
	return &mcp.ToolCallResult{
		Type:   "success",
		Status: "success",
		Result: transformedResp,
	}, nil
}

// buildHTTPRequest 构建HTTP请求
func (h *RequestHandler) buildHTTPRequest(endpoint *config.EndpointConfig, params map[string]interface{}) (*http.Request, error) {
	// 处理URL模板
	url, err := h.processURLTemplate(endpoint.URLTemplate, params)
	if err != nil {
		return nil, fmt.Errorf("处理URL模板失败: %w", err)
	}

	// 处理查询参数
	queryParams := make(map[string]string)
	for _, param := range endpoint.Parameters {
		if param.In == "query" {
			value, exists := params[param.Name]
			if !exists && param.Required {
				return nil, fmt.Errorf("缺少必需的查询参数: %s", param.Name)
			}
			if !exists && param.Default != nil {
				value = param.Default
			}
			if value != nil {
				queryParams[param.Name] = fmt.Sprintf("%v", value)
			}
		}
	}

	// 添加查询参数到URL
	if len(queryParams) > 0 {
		queryString := ""
		for key, value := range queryParams {
			if queryString == "" {
				queryString = "?"
			} else {
				queryString += "&"
			}
			queryString += fmt.Sprintf("%s=%s", key, value)
		}
		url += queryString
	}

	// 处理请求体
	var body []byte
	var contentType string
	bodyParams := make(map[string]interface{})
	for _, param := range endpoint.Parameters {
		if param.In == "body" {
			value, exists := params[param.Name]
			if !exists && param.Required {
				return nil, fmt.Errorf("缺少必需的请求体参数: %s", param.Name)
			}
			if !exists && param.Default != nil {
				value = param.Default
			}
			if value != nil {
				bodyParams[param.Name] = value
			}
		}
	}

	// 如果有请求体参数，则序列化为JSON
	if len(bodyParams) > 0 {
		var err error
		body, err = json.Marshal(bodyParams)
		if err != nil {
			return nil, fmt.Errorf("序列化请求体失败: %w", err)
		}
		contentType = "application/json"
	}

	// 创建请求
	req, err := http.NewRequest(endpoint.Method, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置内容类型
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	// 处理头参数
	for _, param := range endpoint.Parameters {
		if param.In == "header" {
			value, exists := params[param.Name]
			if !exists && param.Required {
				return nil, fmt.Errorf("缺少必需的头参数: %s", param.Name)
			}
			if !exists && param.Default != nil {
				value = param.Default
			}
			if value != nil {
				req.Header.Set(param.Name, fmt.Sprintf("%v", value))
			}
		}
	}

	return req, nil
}

// processURLTemplate 处理URL模板
func (h *RequestHandler) processURLTemplate(urlTemplate string, params map[string]interface{}) (string, error) {
	// 使用简单的字符串替换
	result := urlTemplate
	for key, value := range params {
		placeholder := fmt.Sprintf("{%s}", key)
		if strings.Contains(result, placeholder) {
			result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", value))
		}
	}

	// 检查是否还有未替换的占位符
	if strings.Contains(result, "{") && strings.Contains(result, "}") {
		// 尝试使用模板引擎进行更复杂的替换
		tmpl, err := template.New("url").Parse(urlTemplate)
		if err != nil {
			return "", fmt.Errorf("解析URL模板失败: %w", err)
		}

		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, params); err != nil {
			return "", fmt.Errorf("执行URL模板失败: %w", err)
		}

		result = buf.String()
	}

	// 检查是否还有未替换的占位符
	if strings.Contains(result, "{") && strings.Contains(result, "}") {
		return "", fmt.Errorf("URL模板中存在未替换的占位符: %s", result)
	}

	return result, nil
}