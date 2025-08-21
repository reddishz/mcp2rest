#!/bin/bash

# Goroutine 监控脚本

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 服务器路径
SERVER_PATH="./bin/mcp2rest"
CONFIG_PATH="./configs/bmc_api.yaml"

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

# 获取进程的协程数量
get_goroutine_count() {
    local pid=$1
    if [ -z "$pid" ]; then
        echo "0"
        return
    fi
    
    # 使用 /proc/pid/status 获取线程数
    if [ -f "/proc/$pid/status" ]; then
        local threads=$(grep "Threads:" /proc/$pid/status | awk '{print $2}')
        echo "${threads:-0}"
    else
        echo "0"
    fi
}

# 监控单个进程
monitor_process() {
    local pid=$1
    local duration=${2:-10}
    
    log_info "监控进程 $pid，持续 $duration 秒"
    
    local start_time=$(date +%s)
    local end_time=$((start_time + duration))
    
    echo "时间,进程ID,线程数,内存(KB),CPU%"
    
    while [ $(date +%s) -lt $end_time ]; do
        if ! kill -0 $pid 2>/dev/null; then
            log_warn "进程 $pid 已退出"
            break
        fi
        
        local timestamp=$(date '+%H:%M:%S')
        local threads=$(get_goroutine_count $pid)
        local memory=$(ps -p $pid -o rss= | tr -d ' ')
        local cpu=$(ps -p $pid -o pcpu= | tr -d ' ')
        
        echo "$timestamp,$pid,$threads,${memory:-0},${cpu:-0}"
        sleep 1
    done
}

# 测试协程泄漏
test_goroutine_leak() {
    log_info "测试协程泄漏"
    
    # 设置环境变量
    export APIKEYAUTH_API_KEY="ded45a001ffb9c47b1e29fcbdd6bcec6"
    
    # 启动服务器
    $SERVER_PATH -config $CONFIG_PATH > /tmp/goroutine_test.log 2>&1 &
    local server_pid=$!
    
    # 等待服务器启动
    sleep 3
    
    if ! kill -0 $server_pid 2>/dev/null; then
        log_error "服务器启动失败"
        return 1
    fi
    
    log_info "服务器已启动，PID: $server_pid"
    
    # 监控初始状态
    local initial_threads=$(get_goroutine_count $server_pid)
    log_info "初始线程数: $initial_threads"
    
    # 发送多个请求
    log_info "发送测试请求..."
    for i in {1..10}; do
        echo '{"jsonrpc":"2.0","id":"'$i'","method":"tools/list","params":{}}' | timeout 5s $SERVER_PATH -config $CONFIG_PATH > /dev/null 2>&1 || true
        sleep 0.5
    done
    
    # 检查线程数变化
    sleep 2
    local current_threads=$(get_goroutine_count $server_pid)
    log_info "处理后线程数: $current_threads"
    
    # 发送退出信号
    log_info "发送退出信号..."
    kill -TERM $server_pid
    
    # 等待进程退出
    local timeout=10
    local count=0
    while kill -0 $server_pid 2>/dev/null && [ $count -lt $timeout ]; do
        sleep 1
        count=$((count + 1))
    done
    
    if kill -0 $server_pid 2>/dev/null; then
        log_error "进程未在 $timeout 秒内退出"
        kill -9 $server_pid 2>/dev/null || true
        return 1
    else
        log_info "进程已正常退出"
    fi
    
    # 分析结果
    local thread_diff=$((current_threads - initial_threads))
    if [ $thread_diff -gt 5 ]; then
        log_warn "检测到可能的协程泄漏: 线程数增加了 $thread_diff"
    else
        log_info "未检测到明显的协程泄漏"
    fi
}

# 压力测试
stress_test() {
    log_info "协程压力测试"
    
    # 设置环境变量
    export APIKEYAUTH_API_KEY="ded45a001ffb9c47b1e29fcbdd6bcec6"
    
    # 启动服务器
    $SERVER_PATH -config $CONFIG_PATH > /tmp/stress_test.log 2>&1 &
    local server_pid=$!
    
    # 等待服务器启动
    sleep 3
    
    if ! kill -0 $server_pid 2>/dev/null; then
        log_error "服务器启动失败"
        return 1
    fi
    
    log_info "服务器已启动，PID: $server_pid"
    
    # 监控进程
    monitor_process $server_pid 30 > /tmp/goroutine_monitor.csv &
    local monitor_pid=$!
    
    # 并发发送请求
    log_info "开始压力测试..."
    for i in {1..50}; do
        (
            echo '{"jsonrpc":"2.0","id":"stress_'$i'","method":"tools/list","params":{}}' | \
            timeout 10s $SERVER_PATH -config $CONFIG_PATH > /dev/null 2>&1 || true
        ) &
        
        # 控制并发数
        if [ $((i % 10)) -eq 0 ]; then
            wait
            sleep 1
        fi
    done
    
    wait
    
    # 停止监控
    kill $monitor_pid 2>/dev/null || true
    
    # 停止服务器
    kill -TERM $server_pid
    sleep 5
    kill -9 $server_pid 2>/dev/null || true
    
    # 显示监控结果
    if [ -f "/tmp/goroutine_monitor.csv" ]; then
        log_info "监控结果:"
        cat /tmp/goroutine_monitor.csv | tail -10
    fi
}

# 主函数
main() {
    case "${1:-test}" in
        "test")
            test_goroutine_leak
            ;;
        "stress")
            stress_test
            ;;
        "monitor")
            if [ -z "$2" ]; then
                log_error "请提供进程ID"
                echo "用法: $0 monitor <PID> [duration]"
                exit 1
            fi
            monitor_process $2 ${3:-60}
            ;;
        "help"|"-h"|"--help")
            echo "用法: $0 [命令]"
            echo ""
            echo "命令:"
            echo "  test      - 测试协程泄漏 (默认)"
            echo "  stress    - 压力测试"
            echo "  monitor   - 监控指定进程"
            echo "  help      - 显示此帮助信息"
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
