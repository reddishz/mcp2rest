package logging

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

var Logger *log.Logger

func InitLogger() error {
	// 获取当前工作目录
	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("无法获取工作目录: %v", err)
	}
	exeDir := workDir

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

	// 生成带时间戳的日志文件名
	timestamp := time.Now().Format("20060102_150405")
	logFile := filepath.Join(logDir, "server_" + timestamp + ".log")

	// 强制创建日志文件
	file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("无法创建日志文件: %v", err)
	}

	Logger = log.New(file, "", log.Ldate|log.Ltime|log.Lshortfile)
	return nil
}

