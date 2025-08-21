#!/bin/bash

# 进程泄漏调试脚本

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_debug() {
    echo -e "${BLUE}[DEBUG]${NC} $1"
}

# 监控进程数量
monitor_processes() {
    local count=$(ps aux | grep mcp2rest | grep -v grep | wc -l)
    echo "$count"
}

# 显示进程详情
show_processes() {
    echo "当前 mcp2rest 进程:"
    ps aux | grep mcp2rest | grep -v grep | while read line; do
        echo "  $line"
    done
}

# 监控日志文件
monitor_logs() {
    local log_file=$(ls -t logs/ | head -1)
    if [ -n "$log_file" ]; then
        echo "最新日志文件: logs/$log_file"
        echo "最后 20 行日志:"
        tail -20 "logs/$log_file"
    else
        echo "没有找到日志文件"
    fi
}

# 测试进程泄漏
test_process_leak() {
    log_info "开始测试进程泄漏..."
    
    # 清理现有进程
    pkill -9 -f mcp2rest 2>/dev/null || true
    sleep 1
    
    # 记录初始进程数
    local initial_count=$(monitor_processes)
    log_info "初始进程数: $initial_count"
    
    # 启动服务器并立即停止
    log_info "启动服务器..."
    export APIKEYAUTH_API_KEY="ded45a001ffb9c47b1e29fcbdd6bcec6"
    timeout 10s ./bin/mcp2rest -config ./configs/bmc_api.yaml &
    local server_pid=$!
    sleep 3
    
    # 检查进程是否启动
    local running_count=$(monitor_processes)
    log_info "启动后进程数: $running_count"
    
    if [ $running_count -gt 0 ]; then
        # 发送 SIGTERM 信号
        log_info "发送 SIGTERM 信号到进程 $server_pid"
        kill -TERM $server_pid
        
        # 等待进程退出
        local timeout=10
        local count=0
        while [ $count -lt $timeout ]; do
            sleep 1
            count=$((count + 1))
            local current_count=$(monitor_processes)
            log_info "等待 $count 秒，当前进程数: $current_count"
            
            if [ $current_count -eq 0 ]; then
                log_info "进程已正常退出"
                break
            fi
        done
        
        # 检查最终状态
        local final_count=$(monitor_processes)
        if [ $final_count -gt 0 ]; then
            log_error "进程泄漏！最终进程数: $final_count"
            show_processes
            monitor_logs
            return 1
        else
            log_info "✅ 进程正常退出，无泄漏"
        fi
    else
        log_error "服务器启动失败"
        return 1
    fi
}

# 监控模式
monitor_mode() {
    log_info "进入监控模式，按 Ctrl+C 退出"
    while true; do
        local count=$(monitor_processes)
        local timestamp=$(date '+%H:%M:%S')
        echo "[$timestamp] mcp2rest 进程数: $count"
        
        if [ $count -gt 0 ]; then
            show_processes
        fi
        
        sleep 5
    done
}

# 主函数
main() {
    case "${1:-test}" in
        "test")
            test_process_leak
            ;;
        "monitor")
            monitor_mode
            ;;
        "status")
            local count=$(monitor_processes)
            log_info "当前 mcp2rest 进程数: $count"
            if [ $count -gt 0 ]; then
                show_processes
                monitor_logs
            fi
            ;;
        "cleanup")
            log_info "清理所有 mcp2rest 进程..."
            pkill -9 -f mcp2rest 2>/dev/null || true
            sleep 1
            local count=$(monitor_processes)
            log_info "清理后进程数: $count"
            ;;
        "help"|"-h"|"--help")
            echo "用法: $0 [命令]"
            echo ""
            echo "命令:"
            echo "  test     - 测试进程泄漏 (默认)"
            echo "  monitor  - 持续监控进程"
            echo "  status   - 显示当前状态"
            echo "  cleanup  - 清理所有进程"
            echo "  help     - 显示此帮助信息"
            ;;
        *)
            log_error "未知命令: $1"
            echo "使用 '$0 help' 查看帮助"
            exit 1
            ;;
    esac
}

# 运行主函数
main "$@"
