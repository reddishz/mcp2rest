package transformer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"text/template"

	"github.com/itchyny/gojq"
	"github.com/mcp2rest/internal/config"
)

// ResponseTransformer 处理API响应转换
type ResponseTransformer struct{}

// NewResponseTransformer 创建新的响应转换器
func NewResponseTransformer() (*ResponseTransformer, error) {
	return &ResponseTransformer{}, nil
}

// Transform 转换API响应
func (t *ResponseTransformer) Transform(data []byte, transformConfig *config.TransformConfig) (interface{}, error) {
	if transformConfig == nil || transformConfig.Type == "" || transformConfig.Type == "direct" {
		// 直接返回JSON解析后的响应
		var result interface{}
		if err := json.Unmarshal(data, &result); err != nil {
			return nil, fmt.Errorf("解析JSON响应失败: %w", err)
		}
		return result, nil
	}

	switch transformConfig.Type {
	case "jq":
		return t.transformWithJQ(data, transformConfig.Expression)
	case "template":
		return t.transformWithTemplate(data, transformConfig.Template)
	case "custom":
		// 自定义转换逻辑可以在这里实现
		return nil, fmt.Errorf("自定义转换尚未实现")
	default:
		return nil, fmt.Errorf("不支持的转换类型: %s", transformConfig.Type)
	}
}

// transformWithJQ 使用JQ表达式转换响应
func (t *ResponseTransformer) transformWithJQ(data []byte, expression string) (interface{}, error) {
	if expression == "" {
		return nil, fmt.Errorf("JQ表达式不能为空")
	}

	// 解析JQ表达式
	query, err := gojq.Parse(expression)
	if err != nil {
		return nil, fmt.Errorf("解析JQ表达式失败: %w", err)
	}

	// 解析JSON数据
	var input interface{}
	if err := json.Unmarshal(data, &input); err != nil {
		return nil, fmt.Errorf("解析JSON数据失败: %w", err)
	}

	// 执行JQ查询
	iter := query.Run(input)
	var result interface{}
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := v.(error); ok {
			return nil, fmt.Errorf("执行JQ表达式失败: %w", err)
		}
		result = v
	}

	return result, nil
}

// transformWithTemplate 使用模板转换响应
func (t *ResponseTransformer) transformWithTemplate(data []byte, templateStr string) (interface{}, error) {
	if templateStr == "" {
		return nil, fmt.Errorf("模板字符串不能为空")
	}

	// 解析JSON数据
	var jsonData interface{}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return nil, fmt.Errorf("解析JSON数据失败: %w", err)
	}

	// 解析模板
	tmpl, err := template.New("response").Parse(templateStr)
	if err != nil {
		return nil, fmt.Errorf("解析模板失败: %w", err)
	}

	// 执行模板
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, jsonData); err != nil {
		return nil, fmt.Errorf("执行模板失败: %w", err)
	}

	// 尝试将结果解析为JSON
	var result interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		// 如果不是有效的JSON，则返回字符串
		return buf.String(), nil
	}

	return result, nil
}