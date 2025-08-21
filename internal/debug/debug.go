package debug

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/mcp2rest/internal/logging"
)

var (
	// IsDebugEnabled 是否启用调试模式
	IsDebugEnabled bool
)

// InitDebug 初始化调试模式
func InitDebug() {
	debugEnv := os.Getenv("DEBUG")
	IsDebugEnabled = debugEnv == "true" || debugEnv == "1" || debugEnv == "yes"

	if IsDebugEnabled {
		logging.Logger.Printf("=== 调试模式已启用 ===")
		logging.Logger.Printf("DEBUG 环境变量: %s", debugEnv)
	} else {
		logging.Logger.Printf("调试模式已禁用")
	}
}

// LogRequest 记录请求详情
func LogRequest(method, path string, headers map[string]string, body []byte) {
	if !IsDebugEnabled {
		return
	}

	logging.Logger.Printf("=== 请求详情 ===")
	logging.Logger.Printf("时间: %s", time.Now().Format("2006-01-02 15:04:05.000"))
	logging.Logger.Printf("方法: %s", method)
	logging.Logger.Printf("路径: %s", path)

	if len(headers) > 0 {
		logging.Logger.Printf("请求头:")
		for key, value := range headers {
			logging.Logger.Printf("  %s: %s", key, value)
		}
	}

	if len(body) > 0 {
		logging.Logger.Printf("请求体:")
		if isJSON(body) {
			// 格式化 JSON
			var prettyJSON interface{}
			if err := json.Unmarshal(body, &prettyJSON); err == nil {
				if prettyBytes, err := json.MarshalIndent(prettyJSON, "", "  "); err == nil {
					logging.Logger.Printf("  %s", string(prettyBytes))
				} else {
					logging.Logger.Printf("  %s", string(body))
				}
			} else {
				logging.Logger.Printf("  %s", string(body))
			}
		} else {
			logging.Logger.Printf("  %s", string(body))
		}
	}
	logging.Logger.Printf("=== 请求详情结束 ===")
}

// LogResponse 记录响应详情
func LogResponse(statusCode int, headers map[string]string, body []byte) {
	if !IsDebugEnabled {
		return
	}

	logging.Logger.Printf("=== 响应详情 ===")
	logging.Logger.Printf("时间: %s", time.Now().Format("2006-01-02 15:04:05.000"))
	logging.Logger.Printf("状态码: %d", statusCode)

	if len(headers) > 0 {
		logging.Logger.Printf("响应头:")
		for key, value := range headers {
			logging.Logger.Printf("  %s: %s", key, value)
		}
	} else {
		logging.Logger.Printf("响应头: 无")
	}

	if len(body) > 0 {
		logging.Logger.Printf("响应体:")
		if isJSON(body) {
			// 格式化 JSON
			var prettyJSON interface{}
			if err := json.Unmarshal(body, &prettyJSON); err == nil {
				if prettyBytes, err := json.MarshalIndent(prettyJSON, "", "  "); err == nil {
					logging.Logger.Printf("  %s", string(prettyBytes))
				} else {
					logging.Logger.Printf("  %s", string(body))
				}
			} else {
				logging.Logger.Printf("  %s", string(body))
			}
		} else {
			logging.Logger.Printf("  %s", string(body))
		}
	} else {
		logging.Logger.Printf("响应体: 空")
	}
	logging.Logger.Printf("=== 响应详情结束 ===")
}

// LogHTTPResponse 记录 HTTP 响应详情
func LogHTTPResponse(resp *http.Response) {
	if !IsDebugEnabled {
		return
	}

	logging.Logger.Printf("=== HTTP 响应详情 ===")
	logging.Logger.Printf("时间: %s", time.Now().Format("2006-01-02 15:04:05.000"))
	logging.Logger.Printf("状态码: %d", resp.StatusCode)

	if resp.Header != nil && len(resp.Header) > 0 {
		logging.Logger.Printf("响应头:")
		for key, values := range resp.Header {
			for _, value := range values {
				logging.Logger.Printf("  %s: %s", key, value)
			}
		}
	} else {
		logging.Logger.Printf("响应头: 无")
	}

	if resp.Body != nil {
		body, err := io.ReadAll(resp.Body)
		if err == nil {
			resp.Body = io.NopCloser(bytes.NewBuffer(body)) // 恢复读取后的body
			logging.Logger.Printf("响应体:")
			if isJSON(body) {
				// 格式化 JSON
				var prettyJSON interface{}
				if err := json.Unmarshal(body, &prettyJSON); err == nil {
					if prettyBytes, err := json.MarshalIndent(prettyJSON, "", "  "); err == nil {
						logging.Logger.Printf("  %s", string(prettyBytes))
					} else {
						logging.Logger.Printf("  %s", string(body))
					}
				} else {
					logging.Logger.Printf("  %s", string(body))
				}
			} else {
				logging.Logger.Printf("  %s", string(body))
			}
		} else {
			logging.Logger.Printf("读取响应体失败: %v", err)
		}
	} else {
		logging.Logger.Printf("响应体: 空")
	}
	logging.Logger.Printf("=== HTTP 响应详情结束 ===")
}

// LogMCPRequest 记录 MCP 请求详情
func LogMCPRequest(requestID string, method string, params interface{}) {
	if !IsDebugEnabled {
		return
	}

	logging.Logger.Printf("=== MCP 请求详情 ===")
	logging.Logger.Printf("时间: %s", time.Now().Format("2006-01-02 15:04:05.000"))
	logging.Logger.Printf("请求ID: %s", requestID)
	logging.Logger.Printf("方法: %s", method)

	if params != nil {
		logging.Logger.Printf("参数:")
		if prettyBytes, err := json.MarshalIndent(params, "", "  "); err == nil {
			logging.Logger.Printf("  %s", string(prettyBytes))
		} else {
			logging.Logger.Printf("  %v", params)
		}
	}
	logging.Logger.Printf("=== MCP 请求详情结束 ===")
}

// LogMCPResponse 记录 MCP 响应详情
func LogMCPResponse(requestID string, result interface{}, error interface{}) {
	if !IsDebugEnabled {
		return
	}

	logging.Logger.Printf("=== MCP 响应详情 ===")
	logging.Logger.Printf("时间: %s", time.Now().Format("2006-01-02 15:04:05.000"))
	logging.Logger.Printf("请求ID: %s", requestID)

	if error != nil {
		logging.Logger.Printf("错误:")
		if prettyBytes, err := json.MarshalIndent(error, "", "  "); err == nil {
			logging.Logger.Printf("  %s", string(prettyBytes))
		} else {
			logging.Logger.Printf("  %v", error)
		}
	} else if result != nil {
		logging.Logger.Printf("结果:")
		if prettyBytes, err := json.MarshalIndent(result, "", "  "); err == nil {
			logging.Logger.Printf("  %s", string(prettyBytes))
		} else {
			logging.Logger.Printf("  %v", result)
		}
	}
	logging.Logger.Printf("=== MCP 响应详情结束 ===")
}

// LogHTTPRequest 记录 HTTP 请求详情
func LogHTTPRequest(req interface{}) {
	if !IsDebugEnabled {
		return
	}

	logging.Logger.Printf("=== HTTP 请求详情 ===")
	logging.Logger.Printf("时间: %s", time.Now().Format("2006-01-02 15:04:05.000"))
	logging.Logger.Printf("请求对象: %+v", req)
	logging.Logger.Printf("=== HTTP 请求详情结束 ===")
}

// LogError 记录错误详情
func LogError(context string, err error) {
	if !IsDebugEnabled {
		return
	}

	logging.Logger.Printf("=== 错误详情 ===")
	logging.Logger.Printf("时间: %s", time.Now().Format("2006-01-02 15:04:05.000"))
	logging.Logger.Printf("上下文: %s", context)
	logging.Logger.Printf("错误: %v", err)
	logging.Logger.Printf("=== 错误详情结束 ===")
}

// LogInfo 记录调试信息
func LogInfo(message string, data interface{}) {
	if !IsDebugEnabled {
		return
	}

	logging.Logger.Printf("=== 调试信息 ===")
	logging.Logger.Printf("时间: %s", time.Now().Format("2006-01-02 15:04:05.000"))
	logging.Logger.Printf("消息: %s", message)
	if data != nil {
		logging.Logger.Printf("数据: %+v", data)
	}
	logging.Logger.Printf("=== 调试信息结束 ===")
}

// isJSON 检查是否为 JSON 格式
func isJSON(data []byte) bool {
	var js interface{}
	return json.Unmarshal(data, &js) == nil
}

// FormatJSON 格式化 JSON 字符串
func FormatJSON(data []byte) string {
	if !isJSON(data) {
		return string(data)
	}

	var js interface{}
	if err := json.Unmarshal(data, &js); err != nil {
		return string(data)
	}

	if prettyBytes, err := json.MarshalIndent(js, "", "  "); err == nil {
		return string(prettyBytes)
	}

	return string(data)
}
