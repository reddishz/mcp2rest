package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/mcp2rest/internal/config"
)

func main() {
	// 命令行参数
	configPath := flag.String("config", "configs/auth_config.yaml", "认证配置文件路径")
	action := flag.String("action", "list", "操作类型: list, validate, set, remove")
	apiName := flag.String("api", "", "API 名称")
	authType := flag.String("type", "", "认证类型: api_key, bearer, basic")
	headerName := flag.String("header", "", "请求头名称")
	keyEnv := flag.String("key-env", "", "密钥环境变量名")
	tokenEnv := flag.String("token-env", "", "令牌环境变量名")
	username := flag.String("username", "", "用户名")
	password := flag.String("password", "", "密码")
	
	flag.Parse()

	// 创建认证配置管理器
	authManager := config.NewAuthConfigManager()

	// 加载认证配置
	if err := authManager.LoadAuthConfig(*configPath); err != nil {
		log.Fatalf("加载认证配置失败: %v", err)
	}

	switch *action {
	case "list":
		listAuthConfigs(authManager)
	case "validate":
		validateAuthConfig(authManager, *apiName)
	case "set":
		setAuthConfig(authManager, *apiName, *authType, *headerName, *keyEnv, *tokenEnv, *username, *password)
	case "remove":
		removeAuthConfig(authManager, *apiName)
	default:
		log.Fatalf("不支持的操作: %s", *action)
	}
}

// listAuthConfigs 列出所有认证配置
func listAuthConfigs(authManager *config.AuthConfigManager) {
	configs := authManager.ListAuthConfigs()
	
	if len(configs) == 0 {
		fmt.Println("没有找到认证配置")
		return
	}

	fmt.Println("认证配置列表:")
	fmt.Println(strings.Repeat("=", 50))
	
	for apiName, authConfig := range configs {
		fmt.Printf("API: %s\n", apiName)
		fmt.Printf("  类型: %s\n", authConfig.Type)
		
		switch authConfig.Type {
		case "api_key":
			fmt.Printf("  请求头: %s\n", authConfig.HeaderName)
			fmt.Printf("  环境变量: %s\n", authConfig.KeyEnv)
			if os.Getenv(authConfig.KeyEnv) != "" {
				fmt.Printf("  状态: ✅ 已设置\n")
			} else {
				fmt.Printf("  状态: ❌ 未设置\n")
			}
		case "bearer":
			fmt.Printf("  环境变量: %s\n", authConfig.TokenEnv)
			if os.Getenv(authConfig.TokenEnv) != "" {
				fmt.Printf("  状态: ✅ 已设置\n")
			} else {
				fmt.Printf("  状态: ❌ 未设置\n")
			}
		case "basic":
			fmt.Printf("  用户名: %s\n", authConfig.Username)
			if authConfig.KeyEnv != "" {
				fmt.Printf("  密码环境变量: %s\n", authConfig.KeyEnv)
				if os.Getenv(authConfig.KeyEnv) != "" {
					fmt.Printf("  状态: ✅ 已设置\n")
				} else {
					fmt.Printf("  状态: ❌ 未设置\n")
				}
			} else {
				fmt.Printf("  密码: %s\n", authConfig.Password)
				fmt.Printf("  状态: ✅ 已设置\n")
			}
		}
		fmt.Println()
	}
}

// validateAuthConfig 验证认证配置
func validateAuthConfig(authManager *config.AuthConfigManager, apiName string) {
	if apiName == "" {
		log.Fatal("请指定 API 名称")
	}

	authConfig, err := authManager.GetAuthConfig(apiName)
	if err != nil {
		log.Fatalf("获取认证配置失败: %v", err)
	}

	if err := authManager.ValidateAuthConfig(authConfig); err != nil {
		fmt.Printf("❌ 认证配置验证失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ API '%s' 的认证配置验证通过\n", apiName)
}

// setAuthConfig 设置认证配置
func setAuthConfig(authManager *config.AuthConfigManager, apiName, authType, headerName, keyEnv, tokenEnv, username, password string) {
	if apiName == "" {
		log.Fatal("请指定 API 名称")
	}
	if authType == "" {
		log.Fatal("请指定认证类型")
	}

	authConfig := &config.AuthConfig{
		Type: authType,
	}

	switch authType {
	case "api_key":
		if headerName == "" {
			log.Fatal("API Key 认证需要指定请求头名称")
		}
		if keyEnv == "" {
			log.Fatal("API Key 认证需要指定密钥环境变量名")
		}
		authConfig.HeaderName = headerName
		authConfig.KeyEnv = keyEnv
		
	case "bearer":
		if tokenEnv == "" {
			log.Fatal("Bearer 认证需要指定令牌环境变量名")
		}
		authConfig.TokenEnv = tokenEnv
		
	case "basic":
		if username == "" {
			log.Fatal("基本认证需要指定用户名")
		}
		if password == "" && keyEnv == "" {
			log.Fatal("基本认证需要指定密码或密码环境变量名")
		}
		authConfig.Username = username
		if password != "" {
			authConfig.Password = password
		} else {
			authConfig.KeyEnv = keyEnv
		}
		
	default:
		log.Fatalf("不支持的认证类型: %s", authType)
	}

	authManager.SetAuthConfig(apiName, authConfig)
	fmt.Printf("✅ 已设置 API '%s' 的认证配置\n", apiName)
}

// removeAuthConfig 移除认证配置
func removeAuthConfig(authManager *config.AuthConfigManager, apiName string) {
	if apiName == "" {
		log.Fatal("请指定 API 名称")
	}

	authManager.RemoveAuthConfig(apiName)
	fmt.Printf("✅ 已移除 API '%s' 的认证配置\n", apiName)
}
