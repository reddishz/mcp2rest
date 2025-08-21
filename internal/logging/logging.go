package logging

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

var Logger *log.Logger

func InitLogger() error {
	// 获取可执行文件路径
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("无法获取可执行文件路径: %v", err)
	}

	// 获取可执行文件所在目录
	exeDir := filepath.Dir(exePath)

	// 如果可执行文件在 bin 目录下，使用上级目录
	if filepath.Base(exeDir) == "bin" {
		exeDir = filepath.Dir(exeDir)
	}

	// 创建日志目录
	logDir := filepath.Join(exeDir, "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("无法创建日志目录: %v", err)
	}

	// 检查日志目录是否可写
	if err := os.WriteFile(filepath.Join(logDir, "test_write.log"), []byte("test"), 0644); err != nil {
		return fmt.Errorf("日志目录不可写: %v", err)
	}
	_ = os.Remove(filepath.Join(logDir, "test_write.log"))

	// 获取当前进程ID
	pid := os.Getpid()

	// 获取可执行文件名(不带路径和扩展名)
	exeName := filepath.Base(exePath)
	exeName = exeName[:len(exeName)-len(filepath.Ext(exeName))]

	// 生成按可执行文件名和进程ID命名的日志文件名
	logFile := filepath.Join(logDir, fmt.Sprintf("%s_pid_%d.log", exeName, pid))

	// 强制创建日志文件
	file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("无法创建日志文件: %v", err)
	}

	Logger = log.New(file, "", log.Ldate|log.Ltime|log.Lshortfile)
	return nil
}
