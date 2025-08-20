package config

import (
	"fmt"
	"io/ioutil"
	"time"

	"gopkg.in/yaml.v3"
)

// Config 表示整个配置文件
type Config struct {
	Server    ServerConfig            `yaml:"server"`
	Global    GlobalConfig            `yaml:"global"`
	Endpoints []EndpointConfig        `yaml:"endpoints"`
}

// ServerConfig 表示服务器配置
type ServerConfig struct {
	Port int    `yaml:"port"`
	Host string `yaml:"host"`
	Mode string `yaml:"mode"` // "websocket" 或 "stdio"
}

// GlobalConfig 表示全局设置
type GlobalConfig struct {
	Timeout        time.Duration     `yaml:"timeout"`
	MaxRequestSize string            `yaml:"max_request_size"`
	DefaultHeaders map[string]string `yaml:"default_headers"`
}

// EndpointConfig 表示API端点配置
type EndpointConfig struct {
	Name           string                 `yaml:"name"`
	Description    string                 `yaml:"description"`
	Method         string                 `yaml:"method"`
	URLTemplate    string                 `yaml:"url_template"`
	Authentication AuthConfig             `yaml:"authentication"`
	Parameters     []ParameterConfig      `yaml:"parameters"`
	Response       ResponseConfig         `yaml:"response"`
}

// AuthConfig 表示身份验证配置
type AuthConfig struct {
	Type       string `yaml:"type"`        // "bearer", "api_key", "basic", "oauth2"
	TokenEnv   string `yaml:"token_env"`   // 环境变量名，用于获取令牌
	HeaderName string `yaml:"header_name"` // 自定义头名称，用于API密钥
	KeyEnv     string `yaml:"key_env"`     // 环境变量名，用于获取API密钥
	Username   string `yaml:"username"`    // 用于基本身份验证
	Password   string `yaml:"password"`    // 用于基本身份验证
}

// ParameterConfig 表示参数配置
type ParameterConfig struct {
	Name        string      `yaml:"name"`
	Required    bool        `yaml:"required"`
	Default     interface{} `yaml:"default"`
	Description string      `yaml:"description"`
	In          string      `yaml:"in"` // "path", "query", "body", "header"
	Sensitive   bool        `yaml:"sensitive"`
}

// ResponseConfig 表示响应处理配置
type ResponseConfig struct {
	SuccessCode int               `yaml:"success_code"`
	ErrorCodes  map[int]string    `yaml:"error_codes"`
	Transform   TransformConfig   `yaml:"transform"`
}

// TransformConfig 表示响应转换配置
type TransformConfig struct {
	Type       string `yaml:"type"`       // "direct", "jq", "template", "custom"
	Expression string `yaml:"expression"` // JQ表达式
	Template   string `yaml:"template"`   // 模板字符串
}

// LoadConfig 从文件加载配置
func LoadConfig(filePath string) (*Config, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 设置默认值
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if cfg.Server.Host == "" {
		cfg.Server.Host = "0.0.0.0"
	}
	if cfg.Server.Mode == "" {
		cfg.Server.Mode = "websocket"
	}
	if cfg.Global.Timeout == 0 {
		cfg.Global.Timeout = 30 * time.Second
	}

	return &cfg, nil
}

// GetEndpointByName 根据名称获取端点配置
func (c *Config) GetEndpointByName(name string) (*EndpointConfig, error) {
	for _, endpoint := range c.Endpoints {
		if endpoint.Name == name {
			return &endpoint, nil
		}
	}
	return nil, fmt.Errorf("未找到名为 %s 的端点配置", name)
}