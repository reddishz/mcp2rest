package auth

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"os"

	"github.com/mcp2rest/internal/config"
)

// AuthManager 管理API身份验证
type AuthManager struct{}

// NewAuthManager 创建新的身份验证管理器
func NewAuthManager() (*AuthManager, error) {
	return &AuthManager{}, nil
}

// ApplyAuth 应用身份验证到请求
func (a *AuthManager) ApplyAuth(req *http.Request, authConfig *config.AuthConfig) error {
	if authConfig == nil || authConfig.Type == "" {
		return nil // 无需身份验证
	}

	switch authConfig.Type {
	case "bearer":
		return a.applyBearerAuth(req, authConfig)
	case "api_key":
		return a.applyAPIKeyAuth(req, authConfig)
	case "basic":
		return a.applyBasicAuth(req, authConfig)
	case "oauth2":
		return a.applyOAuth2Auth(req, authConfig)
	default:
		return fmt.Errorf("不支持的身份验证类型: %s", authConfig.Type)
	}
}

// applyBearerAuth 应用Bearer令牌身份验证
func (a *AuthManager) applyBearerAuth(req *http.Request, authConfig *config.AuthConfig) error {
	if authConfig.TokenEnv == "" {
		return fmt.Errorf("Bearer身份验证需要指定token_env")
	}

	token := os.Getenv(authConfig.TokenEnv)
	if token == "" {
		return fmt.Errorf("环境变量 %s 未设置或为空", authConfig.TokenEnv)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	return nil
}

// applyAPIKeyAuth 应用API密钥身份验证
func (a *AuthManager) applyAPIKeyAuth(req *http.Request, authConfig *config.AuthConfig) error {
	if authConfig.HeaderName == "" {
		return fmt.Errorf("API密钥身份验证需要指定header_name")
	}
	if authConfig.KeyEnv == "" {
		return fmt.Errorf("API密钥身份验证需要指定key_env")
	}

	apiKey := os.Getenv(authConfig.KeyEnv)
	if apiKey == "" {
		return fmt.Errorf("环境变量 %s 未设置或为空", authConfig.KeyEnv)
	}

	req.Header.Set(authConfig.HeaderName, apiKey)
	return nil
}

// applyBasicAuth 应用基本身份验证
func (a *AuthManager) applyBasicAuth(req *http.Request, authConfig *config.AuthConfig) error {
	username := authConfig.Username
	password := authConfig.Password

	// 如果用户名或密码为空，则尝试从环境变量获取
	if username == "" && authConfig.TokenEnv != "" {
		username = os.Getenv(authConfig.TokenEnv)
	}
	if password == "" && authConfig.KeyEnv != "" {
		password = os.Getenv(authConfig.KeyEnv)
	}

	if username == "" || password == "" {
		return fmt.Errorf("基本身份验证需要用户名和密码")
	}

	auth := username + ":" + password
	encodedAuth := base64.StdEncoding.EncodeToString([]byte(auth))
	req.Header.Set("Authorization", "Basic "+encodedAuth)
	return nil
}

// applyOAuth2Auth 应用OAuth2身份验证
func (a *AuthManager) applyOAuth2Auth(req *http.Request, authConfig *config.AuthConfig) error {
	// 目前简单实现，与Bearer令牌相同
	// 实际应用中可能需要处理令牌刷新等逻辑
	return a.applyBearerAuth(req, authConfig)
}