package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// 配置结构
type Config struct {
	Server    ServerConfig     `yaml:"server"`
	Global    GlobalConfig     `yaml:"global"`
	Endpoints []EndpointConfig `yaml:"endpoints"`
}

type ServerConfig struct {
	Port int    `yaml:"port"`
	Host string `yaml:"host"`
	Mode string `yaml:"mode"`
}

type GlobalConfig struct {
	Timeout        string            `yaml:"timeout"`
	MaxRequestSize string            `yaml:"max_request_size"`
	DefaultHeaders map[string]string `yaml:"default_headers"`
}

type EndpointConfig struct {
	Name           string            `yaml:"name"`
	Description    string            `yaml:"description"`
	Method         string            `yaml:"method"`
	URLTemplate    string            `yaml:"url_template"`
	Authentication interface{}       `yaml:"authentication"`
	Parameters     []interface{}     `yaml:"parameters"`
	Response       interface{}       `yaml:"response"`
}

// 主配置结构
type MainConfig struct {
	ServerConfig string   `yaml:"server_config"`
	APIConfigs   []string `yaml:"api_configs"`
}

// 服务器配置结构
type ServerOnlyConfig struct {
	Server ServerConfig `yaml:"server"`
	Global GlobalConfig `yaml:"global"`
}

// API配置结构
type APIOnlyConfig struct {
	Endpoints []EndpointConfig `yaml:"endpoints"`
}

func main() {
	// 命令行参数
	inputFile := flag.String("input", "", "输入配置文件路径")
	outputDir := flag.String("output", "", "输出目录路径")
	flag.Parse()

	if *inputFile == "" || *outputDir == "" {
		fmt.Println("用法: config-splitter --input <输入配置文件> --output <输出目录>")
		os.Exit(1)
	}

	// 读取输入文件
	data, err := ioutil.ReadFile(*inputFile)
	if err != nil {
		fmt.Printf("读取配置文件失败: %v\n", err)
		os.Exit(1)
	}

	// 解析配置
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		fmt.Printf("解析配置文件失败: %v\n", err)
		os.Exit(1)
	}

	// 创建输出目录
	serverDir := filepath.Join(*outputDir, "server")
	apiDir := filepath.Join(*outputDir, "api")

	if err := os.MkdirAll(serverDir, 0755); err != nil {
		fmt.Printf("创建服务器配置目录失败: %v\n", err)
		os.Exit(1)
	}

	if err := os.MkdirAll(apiDir, 0755); err != nil {
		fmt.Printf("创建API配置目录失败: %v\n", err)
		os.Exit(1)
	}

	// 创建服务器配置
	serverCfg := ServerOnlyConfig{
		Server: cfg.Server,
		Global: cfg.Global,
	}

	serverData, err := yaml.Marshal(serverCfg)
	if err != nil {
		fmt.Printf("序列化服务器配置失败: %v\n", err)
		os.Exit(1)
	}

	if err := ioutil.WriteFile(filepath.Join(serverDir, "server_config.yaml"), serverData, 0644); err != nil {
		fmt.Printf("写入服务器配置文件失败: %v\n", err)
		os.Exit(1)
	}

	// 创建API配置
	apiCfg := APIOnlyConfig{
		Endpoints: cfg.Endpoints,
	}

	apiData, err := yaml.Marshal(apiCfg)
	if err != nil {
		fmt.Printf("序列化API配置失败: %v\n", err)
		os.Exit(1)
	}

	if err := ioutil.WriteFile(filepath.Join(apiDir, "api_config.yaml"), apiData, 0644); err != nil {
		fmt.Printf("写入API配置文件失败: %v\n", err)
		os.Exit(1)
	}

	// 创建主配置
	mainCfg := MainConfig{
		ServerConfig: "./server/server_config.yaml",
		APIConfigs:   []string{"./api/api_config.yaml"},
	}

	mainData, err := yaml.Marshal(mainCfg)
	if err != nil {
		fmt.Printf("序列化主配置失败: %v\n", err)
		os.Exit(1)
	}

	if err := ioutil.WriteFile(filepath.Join(*outputDir, "main_config.yaml"), mainData, 0644); err != nil {
		fmt.Printf("写入主配置文件失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("配置文件分离完成！")
	fmt.Printf("- 主配置文件: %s\n", filepath.Join(*outputDir, "main_config.yaml"))
	fmt.Printf("- 服务器配置文件: %s\n", filepath.Join(serverDir, "server_config.yaml"))
	fmt.Printf("- API配置文件: %s\n", filepath.Join(apiDir, "api_config.yaml"))
}