package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LoadEnvFile 加载 .env 文件并设置环境变量
func LoadEnvFile(envPath string) error {
	// 如果路径为空，尝试自动查找 .env 文件
	if envPath == "" {
		envPath = findEnvFile()
		if envPath == "" {
			// 没有找到 .env 文件，不是错误
			return nil
		}
	}

	// 检查文件是否存在
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		return fmt.Errorf("环境变量文件不存在: %s", envPath)
	}

	// 读取文件
	file, err := os.Open(envPath)
	if err != nil {
		return fmt.Errorf("打开环境变量文件失败: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// 跳过空行和注释
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// 解析 key=value 格式
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue // 跳过格式不正确的行
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// 移除值两端的引号
		if len(value) >= 2 && (value[0] == '"' && value[len(value)-1] == '"') {
			value = value[1 : len(value)-1]
		}

		// 设置环境变量（如果尚未设置）
		if os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("读取环境变量文件失败: %w", err)
	}

	return nil
}

// findEnvFile 查找 .env 文件
func findEnvFile() string {
	// 获取可执行文件路径
	exePath, err := os.Executable()
	if err != nil {
		exePath = ""
	}

	// 可能的 .env 文件路径
	possiblePaths := []string{
		".env",                    // 当前工作目录
		"configs/.env",            // configs 目录
	}

	// 如果可执行文件路径可用，添加基于可执行文件的路径
	if exePath != "" {
		exeDir := filepath.Dir(exePath)
		possiblePaths = append(possiblePaths,
			filepath.Join(exeDir, ".env"),                    // 可执行文件同级目录
			filepath.Join(exeDir, "configs", ".env"),         // 可执行文件同级 configs 目录
			filepath.Join(filepath.Dir(exeDir), ".env"),      // 可执行文件上级目录
			filepath.Join(filepath.Dir(exeDir), "configs", ".env"), // 可执行文件上级 configs 目录
		)
	}

	// 检查每个可能的路径
	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

// LoadEnvFileWithLog 加载 .env 文件并记录日志
func LoadEnvFileWithLog(envPath string) error {
	// 如果路径为空，尝试自动查找
	if envPath == "" {
		envPath = findEnvFile()
		if envPath == "" {
			// 没有找到 .env 文件，记录日志但不报错
			fmt.Println("未找到 .env 文件，将使用系统环境变量")
			return nil
		}
	}

	fmt.Printf("正在加载环境变量文件: %s\n", envPath)
	err := LoadEnvFile(envPath)
	if err != nil {
		return err
	}

	fmt.Printf("环境变量文件加载成功: %s\n", envPath)
	return nil
}
