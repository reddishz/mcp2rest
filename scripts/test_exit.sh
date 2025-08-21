#!/bin/bash

# 测试 MCP 服务器进程退出功能

set -e

# 设置环境变量
export APIKEYAUTH_API_KEY="ded45a001ffb9c47b1e29fcbdd6bcec6"

# 服务器路径
SERVER_PATH="./bin/mcp2rest"
CONFIG_PATH="./configs/bmc_api.yaml"

# 日志函数
log_info() {
    echo "[INFO] $1"
}

log_success() {
    echo "[SUCCESS] $1"
}

log_error() {
    echo "[ERROR] $1"
}

# 检查服务器是否运行
check_server_running() {
    local pid=$1
    if kill -0 $pid 2>/dev/null; then
        return 0
    else
        return 1
    fi
}

# 测试 1: 正常初始化后退出
test_normal_exit() {
    log_info "测试 1: 正常初始化后退出"
    
    # 启动服务器进程
    $SERVER_PATH -config $CONFIG_PATH > /tmp/test_exit.log 2>&1 &
    local server_pid=$!
    
    # 等待服务器启动
    sleep 2
    
    # 检查进程是否运行
    if ! check_server_running $server_pid; then
        log_error "服务器进程未启动"
        return 1
    fi
    
    log_info "服务器进程已启动，PID: $server_pid"
    
    # 发送初始化请求
    echo '{"jsonrpc":"2.0","id":"init_001","method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}' > /proc/$server_pid/fd/0
    
    # 等待响应
    sleep 1
    
    # 发送退出请求
    echo '{"jsonrpc":"2.0","id":"exit_001","method":"exit","params":{}}' > /proc/$server_pid/fd/0
    
    # 等待进程退出
    local timeout=10
    local count=0
    while check_server_running $server_pid && [ $count -lt $timeout ]; do
        sleep 1
        count=$((count + 1))
    done
    
    if check_server_running $server_pid; then
        log_error "服务器进程未在 $timeout 秒内退出"
        kill -9 $server_pid 2>/dev/null || true
        return 1
    else
        log_success "服务器进程已正常退出"
    fi
}

# 测试 2: 信号终止
test_signal_exit() {
    log_info "测试 2: 信号终止"
    
    # 启动服务器进程
    $SERVER_PATH -config $CONFIG_PATH > /tmp/test_signal.log 2>&1 &
    local server_pid=$!
    
    # 等待服务器启动
    sleep 2
    
    # 检查进程是否运行
    if ! check_server_running $server_pid; then
        log_error "服务器进程未启动"
        return 1
    fi
    
    log_info "服务器进程已启动，PID: $server_pid"
    
    # 发送 SIGTERM 信号
    kill -TERM $server_pid
    
    # 等待进程退出
    local timeout=10
    local count=0
    while check_server_running $server_pid && [ $count -lt $timeout ]; do
        sleep 1
        count=$((count + 1))
    done
    
    if check_server_running $server_pid; then
        log_error "服务器进程未在 $timeout 秒内退出"
        kill -9 $server_pid 2>/dev/null || true
        return 1
    else
        log_success "服务器进程已响应信号退出"
    fi
}

# 测试 3: 检查进程数量
test_process_count() {
    log_info "测试 3: 检查进程数量"
    
    # 获取当前 mcp2rest 进程数量
    local process_count=$(pgrep -f "mcp2rest" | wc -l)
    log_info "当前 mcp2rest 进程数量: $process_count"
    
    if [ $process_count -gt 0 ]; then
        log_error "发现残留的 mcp2rest 进程"
        pgrep -f "mcp2rest" | xargs ps -p
        return 1
    else
        log_success "没有发现残留的 mcp2rest 进程"
    fi
}

# 主测试函数
main() {
    log_info "开始测试 MCP 服务器进程退出功能"
    
    # 清理之前的进程
    pkill -f "mcp2rest" 2>/dev/null || true
    sleep 2
    
    # 运行测试
    test_normal_exit
    test_signal_exit
    test_process_count
    
    log_success "所有测试通过"
}

# 运行主函数
main "$@"
