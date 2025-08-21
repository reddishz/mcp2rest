package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// AuthConfigManager 认证配置管理器
type AuthConfigManager struct {
	configs map[string]*AuthConfig
}

// NewAuthConfigManager 创建新的认证配置管理器
func NewAuthConfigManager() *AuthConfigManager {
	return &AuthConfigManager{
		configs: make(map[string]*AuthConfig),
	}
}

// LoadAuthConfig 从文件加载认证配置
func (acm *AuthConfigManager) LoadAuthConfig(configPath string) error {
	// 尝试多个可能的配置文件路径
	configPaths := []string{
		configPath,
		"configs/auth_config.yaml",
		"../configs/auth_config.yaml",
	}

	// 如果可执行文件路径可用，添加基于可执行文件的路径
	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		configPaths = append(configPaths,
			filepath.Join(exeDir, "configs/auth_config.yaml"),
			filepath.Join(filepath.Dir(exeDir), "configs/auth_config.yaml"),
		)
	}

	// 尝试加载配置文件
	for _, path := range configPaths {
		if _, err := os.Stat(path); err == nil {
			return acm.loadFromFile(path)
		}
	}

	// 如果没有找到配置文件，使用默认配置
	return acm.loadDefaultConfig()
}

// loadFromFile 从文件加载配置
func (acm *AuthConfigManager) loadFromFile(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("读取认证配置文件失败: %w", err)
	}

	var config struct {
		BMCAPI    *AuthConfig `yaml:"bmc_api"`
		WeatherAPI *AuthConfig `yaml:"weather_api"`
		UserAPI   *AuthConfig `yaml:"user_api"`
	}

	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("解析认证配置文件失败: %w", err)
	}

	// 加载各个 API 的认证配置
	if config.BMCAPI != nil {
		acm.configs["bmc_api"] = config.BMCAPI
	}
	if config.WeatherAPI != nil {
		acm.configs["weather_api"] = config.WeatherAPI
	}
	if config.UserAPI != nil {
		acm.configs["user_api"] = config.UserAPI
	}

	return nil
}

// loadDefaultConfig 加载默认配置
func (acm *AuthConfigManager) loadDefaultConfig() error {
	// BMC API 默认配置
	acm.configs["bmc_api"] = &AuthConfig{
		Type:       "api_key",
		HeaderName: "X-API-Key",
		KeyEnv:     "BMC_API_KEY",
	}

	// 其他 API 的默认配置
	acm.configs["weather_api"] = &AuthConfig{
		Type:      "bearer",
		TokenEnv:  "WEATHER_API_TOKEN",
	}

	acm.configs["user_api"] = &AuthConfig{
		Type:       "basic",
		Username:   "admin",
		KeyEnv:     "USER_API_PASSWORD",
	}

	return nil
}

// GetAuthConfig 获取指定 API 的认证配置
func (acm *AuthConfigManager) GetAuthConfig(apiName string) (*AuthConfig, error) {
	config, exists := acm.configs[apiName]
	if !exists {
		return nil, fmt.Errorf("未找到 API '%s' 的认证配置", apiName)
	}
	return config, nil
}

// ValidateAuthConfig 验证认证配置
func (acm *AuthConfigManager) ValidateAuthConfig(config *AuthConfig) error {
	if config == nil {
		return fmt.Errorf("认证配置为空")
	}

	switch config.Type {
	case "api_key":
		if config.HeaderName == "" {
			return fmt.Errorf("API Key 认证需要指定 header_name")
		}
		if config.KeyEnv == "" {
			return fmt.Errorf("API Key 认证需要指定 key_env")
		}
		if os.Getenv(config.KeyEnv) == "" {
			return fmt.Errorf("环境变量 %s 未设置或为空", config.KeyEnv)
		}

	case "bearer":
		if config.TokenEnv == "" {
			return fmt.Errorf("Bearer 认证需要指定 token_env")
		}
		if os.Getenv(config.TokenEnv) == "" {
			return fmt.Errorf("环境变量 %s 未设置或为空", config.TokenEnv)
		}

	case "basic":
		if config.Username == "" && config.TokenEnv == "" {
			return fmt.Errorf("基本认证需要指定 username 或 token_env")
		}
		if config.Password == "" && config.KeyEnv == "" {
			return fmt.Errorf("基本认证需要指定 password 或 key_env")
		}

	default:
		return fmt.Errorf("不支持的身份验证类型: %s", config.Type)
	}

	return nil
}

// ListAuthConfigs 列出所有认证配置
func (acm *AuthConfigManager) ListAuthConfigs() map[string]*AuthConfig {
	return acm.configs
}

// SetAuthConfig 设置认证配置
func (acm *AuthConfigManager) SetAuthConfig(apiName string, config *AuthConfig) {
	acm.configs[apiName] = config
}

// RemoveAuthConfig 移除认证配置
func (acm *AuthConfigManager) RemoveAuthConfig(apiName string) {
	delete(acm.configs, apiName)
}
