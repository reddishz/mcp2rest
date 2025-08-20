package mcp

import (
	"encoding/json"
	"fmt"
)

// MCPRequest 表示MCP请求
type MCPRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      string          `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

// MCPResponse 表示MCP响应
type MCPResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      string          `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *MCPError       `json:"error,omitempty"`
}

// MCPError 表示MCP错误
type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ToolCallParams 表示工具调用参数
type ToolCallParams struct {
	Name       string                 `json:"name"`
	Parameters map[string]interface{} `json:"parameters"`
}

// ToolCallResult 表示工具调用结果
type ToolCallResult struct {
	Type   string      `json:"type"`
	Status string      `json:"status"`
	Result interface{} `json:"result"`
}

// NewSuccessResponse 创建成功响应
func NewSuccessResponse(id string, result interface{}) (*MCPResponse, error) {
	resultBytes, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("序列化结果失败: %w", err)
	}

	return &MCPResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  resultBytes,
	}, nil
}

// NewErrorResponse 创建错误响应
func NewErrorResponse(id string, code int, message string) *MCPResponse {
	return &MCPResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &MCPError{
			Code:    code,
			Message: message,
		},
	}
}

// ParseToolCallParams 解析工具调用参数
func ParseToolCallParams(params json.RawMessage) (*ToolCallParams, error) {
	var toolParams ToolCallParams
	if err := json.Unmarshal(params, &toolParams); err != nil {
		return nil, fmt.Errorf("解析工具调用参数失败: %w", err)
	}
	return &toolParams, nil
}