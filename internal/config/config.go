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

// Config 表示整个配置文件
type Config struct {
	Server ServerConfig `yaml:"server"`
	Global GlobalConfig `yaml:"global"`
}

// ServerConfig 表示服务器配置
type ServerConfig struct {
	Port int    `yaml:"port"`
	Host string `yaml:"host"`
	Mode string `yaml:"mode"` // "stdio" 或 "sse"
}

// GlobalConfig 表示全局设置
type GlobalConfig struct {
	Timeout        time.Duration     `yaml:"timeout"`
	MaxRequestSize string            `yaml:"max_request_size"`
	DefaultHeaders map[string]string `yaml:"default_headers"`
}

// OpenAPISpec 表示 OpenAPI 规范
type OpenAPISpec struct {
	OpenAPI    string                 `json:"openapi" yaml:"openapi"`
	Info       OpenAPIInfo            `json:"info" yaml:"info"`
	Servers    []OpenAPIServer        `json:"servers" yaml:"servers"`
	Paths      map[string]PathItem    `json:"paths" yaml:"paths"`
	Components OpenAPIComponents      `json:"components" yaml:"components"`
	Security   []map[string][]string  `json:"security" yaml:"security"`
}

// OpenAPIInfo 表示 OpenAPI 信息
type OpenAPIInfo struct {
	Title       string `json:"title" yaml:"title"`
	Description string `json:"description" yaml:"description"`
	Version     string `json:"version" yaml:"version"`
}

// OpenAPIServer 表示 OpenAPI 服务器
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

// AuthConfig 表示身份验证配置
type AuthConfig struct {
	Type       string `yaml:"type"`        // "bearer", "api_key", "basic", "oauth2"
	TokenEnv   string `yaml:"token_env"`   // 环境变量名，用于获取令牌
	HeaderName string `yaml:"header_name"` // 自定义头名称，用于API密钥
	KeyEnv     string `yaml:"key_env"`     // 环境变量名，用于获取API密钥
	Username   string `yaml:"username"`    // 用于基本身份验证
	Password   string `yaml:"password"`    // 用于基本身份验证
}

// resolveConfigPath 解析配置文件路径，支持从可执行文件目录或上一级目录查找
func resolveConfigPath(exeDir, configPath string) string {
	// 如果已经是绝对路径，直接返回
	if filepath.IsAbs(configPath) {
		return configPath
	}

	// 优先检查可执行文件目录下的路径
	joinedPath := filepath.Join(exeDir, configPath)
	if _, err := os.Stat(joinedPath); err == nil {
		return joinedPath
	}

	// 如果文件不存在且可执行文件位于 bin 目录，则尝试上一级目录
	if filepath.Base(exeDir) == "bin" {
		parentDir := filepath.Dir(exeDir)
		joinedPath = filepath.Join(parentDir, configPath)
		if _, err := os.Stat(joinedPath); err == nil {
			return joinedPath
		}
	}

	// 如果仍未找到，返回原始路径（后续逻辑会处理文件不存在的情况）
	return filepath.Join(exeDir, configPath)
}

// GetDefaultServerConfig 返回默认的服务器配置
func GetDefaultServerConfig() (*ServerConfig, *GlobalConfig) {
	server := &ServerConfig{
		Port: 8080,
		Host: "0.0.0.0",
		Mode: "sse",
	}
	
	global := &GlobalConfig{
		Timeout: 30 * time.Second,
	}
	
	return server, global
}

// TryLoadServerConfig 尝试从configs/server.yaml加载服务器配置，如果不存在则使用默认配置
func TryLoadServerConfig() (*ServerConfig, *GlobalConfig, error) {
	exePath, err := os.Executable()
	if err != nil {
		return nil, nil, fmt.Errorf("无法获取可执行文件路径: %v", err)
	}
	exeDir := filepath.Dir(exePath)
	serverConfigPath := resolveConfigPath(exeDir, "configs/server.yaml")
	
	// 检查文件是否存在
	if _, err := os.Stat(serverConfigPath); os.IsNotExist(err) {
		// 文件不存在，尝试从工作目录加载
		cwd, err := os.Getwd()
		if err != nil {
			return nil, nil, fmt.Errorf("获取当前工作目录失败: %v", err)
		}
		serverConfigPath = filepath.Join(cwd, "configs/server.yaml")
		if _, err := os.Stat(serverConfigPath); os.IsNotExist(err) {
			// 文件仍不存在，返回默认配置
			server, global := GetDefaultServerConfig()
			return server, global, nil
		}
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
		cfg.Server.Mode = "sse"
	}
	if cfg.Global.Timeout == 0 {
		cfg.Global.Timeout = 30 * time.Second
	}

	return &cfg.Server, &cfg.Global, nil
}

// LoadServerConfig 从服务器配置文件加载配置
func LoadServerConfig(filePath string) (*ServerConfig, *GlobalConfig, error) {
	if filePath == "" {
		return nil, nil, fmt.Errorf("服务器配置文件路径为空")
	}

	// 记录文件路径的绝对路径
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, nil, fmt.Errorf("获取文件绝对路径失败: %w", err)
	}

	// 检查文件是否存在
	if _, err := os.Stat(absPath); err != nil {
		return nil, nil, fmt.Errorf("服务器配置文件 %s 不存在: %w", absPath, err)
	}

	data, err := ioutil.ReadFile(absPath)
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
		cfg.Server.Mode = "sse"
	}
	if cfg.Global.Timeout == 0 {
		cfg.Global.Timeout = 30 * time.Second
	}

	return &cfg.Server, &cfg.Global, nil
}

// IsOpenAPISpec 检查文件是否为OpenAPI规范
func IsOpenAPISpec(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	return ext == ".json" || ext == ".yaml" || ext == ".yml"
}