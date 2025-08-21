package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// MainConfig 表示主配置文件
type MainConfig struct {
	ServerConfig string   `yaml:"server_config"` // 服务器配置文件路径
	APIConfigs   []string `yaml:"api_configs"`   // API配置文件路径列表
	OpenAPISpecs []string `yaml:"openapi_specs"` // OpenAPI规范文件路径列表
}

// Config 表示整个配置文件
type Config struct {
	Server    ServerConfig     `yaml:"server"`
	Global    GlobalConfig     `yaml:"global"`
	Endpoints []EndpointConfig `yaml:"endpoints"`
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
	Name           string            `yaml:"name"`
	Description    string            `yaml:"description"`
	Method         string            `yaml:"method"`
	URLTemplate    string            `yaml:"url_template"`
	Authentication AuthConfig        `yaml:"authentication"`
	Parameters     []ParameterConfig `yaml:"parameters"`
	Response       ResponseConfig    `yaml:"response"`
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
	SuccessCode int             `yaml:"success_code"`
	ErrorCodes  map[int]string  `yaml:"error_codes"`
	Transform   TransformConfig `yaml:"transform"`
}

// TransformConfig 表示响应转换配置
type TransformConfig struct {
	Type       string `yaml:"type"`       // "direct", "jq", "template", "custom"
	Expression string `yaml:"expression"` // JQ表达式
	Template   string `yaml:"template"`   // 模板字符串
}

// GetDefaultServerConfig 返回默认的服务器配置
func GetDefaultServerConfig() (*ServerConfig, *GlobalConfig) {
	server := &ServerConfig{
		Port: 8080,
		Host: "0.0.0.0",
		Mode: "websocket",
	}
	
	global := &GlobalConfig{
		Timeout: 30 * time.Second,
	}
	
	return server, global
}

// TryLoadServerConfig 尝试从configs/server.yaml加载服务器配置，如果不存在则使用默认配置
func TryLoadServerConfig() (*ServerConfig, *GlobalConfig, error) {
	serverConfigPath := "configs/server.yaml"
	
	// 检查文件是否存在
	if _, err := os.Stat(serverConfigPath); os.IsNotExist(err) {
		// 文件不存在，返回默认配置
		server, global := GetDefaultServerConfig()
		return server, global, nil
	}
	
	// 文件存在，尝试加载
	data, err := ioutil.ReadFile(serverConfigPath)
	if err != nil {
		return nil, nil, fmt.Errorf("读取服务器配置文件失败: %w", err)
	}

	var cfg struct {
		Server ServerConfig `yaml:"server"`
		Global GlobalConfig `yaml:"global"`
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, nil, fmt.Errorf("解析服务器配置文件失败: %w", err)
	}

	// 设置默认值（如果配置文件中未指定）
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

	return &cfg.Server, &cfg.Global, nil
}

// LoadMainConfig 从主配置文件加载配置
func LoadMainConfig(filePath string) (*MainConfig, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("读取主配置文件失败: %w", err)
	}

	var mainCfg MainConfig
	if err := yaml.Unmarshal(data, &mainCfg); err != nil {
		return nil, fmt.Errorf("解析主配置文件失败: %w", err)
	}

	return &mainCfg, nil
}

// LoadServerConfig 从服务器配置文件加载配置
func LoadServerConfig(filePath string) (*ServerConfig, *GlobalConfig, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, nil, fmt.Errorf("读取服务器配置文件失败: %w", err)
	}

	var cfg struct {
		Server ServerConfig `yaml:"server"`
		Global GlobalConfig `yaml:"global"`
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, nil, fmt.Errorf("解析服务器配置文件失败: %w", err)
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

	return &cfg.Server, &cfg.Global, nil
}

// LoadAPIConfig 从API配置文件加载端点配置
func LoadAPIConfig(filePath string) ([]EndpointConfig, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("读取API配置文件失败: %w", err)
	}

	var cfg struct {
		Endpoints []EndpointConfig `yaml:"endpoints"`
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析API配置文件失败: %w", err)
	}

	return cfg.Endpoints, nil
}

// LoadConfig 从主配置文件加载完整配置
func LoadConfig(filePath string) (*Config, error) {
	// 加载主配置
	mainCfg, err := LoadMainConfig(filePath)
	if err != nil {
		return nil, err
	}

	// 加载服务器配置
	server, global, err := LoadServerConfig(mainCfg.ServerConfig)
	if err != nil {
		return nil, err
	}

	// 创建完整配置
	cfg := &Config{
		Server:    *server,
		Global:    *global,
		Endpoints: []EndpointConfig{},
	}

	// 加载所有API配置
	for _, apiConfigPath := range mainCfg.APIConfigs {
		endpoints, err := LoadAPIConfig(apiConfigPath)
		if err != nil {
			return nil, fmt.Errorf("加载API配置文件 %s 失败: %w", apiConfigPath, err)
		}
		cfg.Endpoints = append(cfg.Endpoints, endpoints...)
	}

	// 加载所有OpenAPI规范
	for _, openAPIPath := range mainCfg.OpenAPISpecs {
		// 这里将在后面实现OpenAPI规范的加载
		// 现在只是占位，实际实现将在openapi包中
		fmt.Printf("将加载OpenAPI规范: %s\n", openAPIPath)
	}

	return cfg, nil
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

// IsOpenAPISpec 检查文件是否为OpenAPI规范
func IsOpenAPISpec(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	return ext == ".json" || ext == ".yaml" || ext == ".yml"
}

// LoadConfigFromAPIFile 直接从API配置文件加载配置，使用默认服务器配置
func LoadConfigFromAPIFile(apiConfigPath string) (*Config, error) {
	// 尝试加载服务器配置，如果不存在则使用默认配置
	server, global, err := TryLoadServerConfig()
	if err != nil {
		return nil, err
	}
	
	// 创建完整配置
	cfg := &Config{
		Server:    *server,
		Global:    *global,
		Endpoints: []EndpointConfig{},
	}
	
	// 加载API配置
	endpoints, err := LoadAPIConfig(apiConfigPath)
	if err != nil {
		return nil, fmt.Errorf("加载API配置文件 %s 失败: %w", apiConfigPath, err)
	}
	cfg.Endpoints = append(cfg.Endpoints, endpoints...)
	
	return cfg, nil
}