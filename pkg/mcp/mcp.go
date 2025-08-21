package mcp

import (
	"encoding/json"
	"fmt"
)

// MCPRequest 表示MCP请求
type MCPRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"` // 支持字符串或数字
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

// MCPResponse 表示MCP响应
type MCPResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"` // 支持字符串或数字
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

// GetIDString 获取ID的字符串表示
func (r *MCPRequest) GetIDString() string {
	if r.ID == nil {
		return ""
	}
	
	// 尝试解析为字符串
	var strID string
	if err := json.Unmarshal(r.ID, &strID); err == nil {
		return strID
	}
	
	// 尝试解析为数字
	var numID json.Number
	if err := json.Unmarshal(r.ID, &numID); err == nil {
		return numID.String()
	}
	
	// 如果都失败，返回原始字符串
	return string(r.ID)
}

// GetIDString 获取ID的字符串表示
func (r *MCPResponse) GetIDString() string {
	if r.ID == nil {
		return ""
	}
	
	// 尝试解析为字符串
	var strID string
	if err := json.Unmarshal(r.ID, &strID); err == nil {
		return strID
	}
	
	// 尝试解析为数字
	var numID json.Number
	if err := json.Unmarshal(r.ID, &numID); err == nil {
		return numID.String()
	}
	
	// 如果都失败，返回原始字符串
	return string(r.ID)
}

// SetID 设置ID
func (r *MCPResponse) SetID(id interface{}) error {
	idBytes, err := json.Marshal(id)
	if err != nil {
		return fmt.Errorf("序列化ID失败: %w", err)
	}
	r.ID = idBytes
	return nil
}

// NewSuccessResponse 创建成功响应
func NewSuccessResponse(id interface{}, result interface{}) (*MCPResponse, error) {
	resultBytes, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("序列化结果失败: %w", err)
	}

	response := &MCPResponse{
		JSONRPC: "2.0",
		Result:  resultBytes,
	}
	
	if err := response.SetID(id); err != nil {
		return nil, err
	}
	
	return response, nil
}

// NewErrorResponse 创建错误响应
func NewErrorResponse(id interface{}, code int, message string) *MCPResponse {
	response := &MCPResponse{
		JSONRPC: "2.0",
		Error: &MCPError{
			Code:    code,
			Message: message,
		},
	}
	
	response.SetID(id) // 忽略错误，因为这是错误响应
	return response
}

// ParseToolCallParams 解析工具调用参数
func ParseToolCallParams(params json.RawMessage) (*ToolCallParams, error) {
	var toolParams ToolCallParams
	if err := json.Unmarshal(params, &toolParams); err != nil {
		return nil, fmt.Errorf("解析工具调用参数失败: %w", err)
	}
	return &toolParams, nil
}