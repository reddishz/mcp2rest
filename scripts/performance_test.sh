#!/bin/bash

# MCP2REST 性能测试脚本
# 用于测试优化后的服务器性能

set -e

# 配置
SERVER_PATH="./bin/mcp2rest"
CONFIG_PATH="./configs/bmc_api.yaml"
TEST_COUNT=100
CONCURRENT_REQUESTS=10

# 设置 API 密钥
export APIKEYAUTH_API_KEY="ded45a001ffb9c47b1e29fcbdd6bcec6"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查依赖
check_dependencies() {
    log_info "检查依赖..."
    
    if [ ! -f "$SERVER_PATH" ]; then
        log_error "服务器可执行文件不存在: $SERVER_PATH"
        log_info "请先编译项目: go build -o bin/mcp2rest cmd/mcp2rest/main.go"
        exit 1
    fi
    
    if [ ! -f "$CONFIG_PATH" ]; then
        log_error "配置文件不存在: $CONFIG_PATH"
        exit 1
    fi
    
    log_success "依赖检查通过"
}

# 生成测试请求
generate_test_request() {
    local id="$1"
    cat <<EOF
{"jsonrpc":"2.0","id":"perf_test_$id","method":"toolCall","params":{"name":"getList","parameters":{"page":1,"limit":5}}}
EOF
}

# 单次性能测试
single_performance_test() {
    local request_id="$1"
    local start_time=$(date +%s%N)
    
    # 生成并发送请求
    local request=$(generate_test_request "$request_id")
    local response=$(echo "$request" | $SERVER_PATH -config $CONFIG_PATH 2>/dev/null)
    
    local end_time=$(date +%s%N)
    local duration=$(( (end_time - start_time) / 1000000 )) # 转换为毫秒
    
    # 检查响应是否成功
    if echo "$response" | grep -q '"error"'; then
        echo "FAIL:$duration:$response"
    else
        echo "SUCCESS:$duration:$response"
    fi
}

# 并发性能测试
concurrent_performance_test() {
    local concurrent_count="$1"
    local total_requests="$2"
    
    log_info "开始并发性能测试: $concurrent_count 并发, $total_requests 总请求数"
    
    local results=()
    local success_count=0
    local fail_count=0
    local total_duration=0
    
    # 创建临时文件存储结果
    local temp_file=$(mktemp)
    
    # 启动并发请求
    for ((i=1; i<=total_requests; i++)); do
        single_performance_test "$i" >> "$temp_file" &
        
        # 控制并发数
        if (( i % concurrent_count == 0 )); then
            wait
        fi
    done
    
    # 等待所有请求完成
    wait
    
    # 分析结果
    while IFS=: read -r status duration response; do
        if [ "$status" = "SUCCESS" ]; then
            ((success_count++))
        else
            ((fail_count++))
        fi
        ((total_duration += duration))
    done < "$temp_file"
    
    # 清理临时文件
    rm -f "$temp_file"
    
    # 计算统计信息
    local avg_duration=0
    if [ $((success_count + fail_count)) -gt 0 ]; then
        avg_duration=$((total_duration / (success_count + fail_count)))
    fi
    
    local success_rate=0
    if [ $((success_count + fail_count)) -gt 0 ]; then
        success_rate=$((success_count * 100 / (success_count + fail_count)))
    fi
    
    # 输出结果
    log_info "并发性能测试结果:"
    log_info "  总请求数: $((success_count + fail_count))"
    log_info "  成功请求: $success_count"
    log_info "  失败请求: $fail_count"
    log_info "  成功率: ${success_rate}%"
    log_info "  平均响应时间: ${avg_duration}ms"
    log_info "  总响应时间: ${total_duration}ms"
    
    if [ $success_rate -ge 90 ]; then
        log_success "性能测试通过"
    else
        log_warning "性能测试警告: 成功率低于90%"
    fi
}

# 压力测试
stress_test() {
    log_info "开始压力测试..."
    
    # 逐步增加并发数
    for concurrent in 1 5 10 20 50; do
        log_info "测试并发数: $concurrent"
        concurrent_performance_test "$concurrent" 100
        echo
    done
}

# 延迟测试
latency_test() {
    log_info "开始延迟测试..."
    
    local total_duration=0
    local min_duration=999999
    local max_duration=0
    local count=0
    
    for ((i=1; i<=50; i++)); do
        local start_time=$(date +%s%N)
        local request=$(generate_test_request "$i")
        local response=$(echo "$request" | $SERVER_PATH -config $CONFIG_PATH 2>/dev/null)
        local end_time=$(date +%s%N)
        local duration=$(( (end_time - start_time) / 1000000 ))
        
        ((total_duration += duration))
        ((count++))
        
        if [ $duration -lt $min_duration ]; then
            min_duration=$duration
        fi
        if [ $duration -gt $max_duration ]; then
            max_duration=$duration
        fi
    done
    
    local avg_duration=$((total_duration / count))
    
    log_info "延迟测试结果:"
    log_info "  测试次数: $count"
    log_info "  最小延迟: ${min_duration}ms"
    log_info "  最大延迟: ${max_duration}ms"
    log_info "  平均延迟: ${avg_duration}ms"
    
    if [ $avg_duration -lt 100 ]; then
        log_success "延迟测试通过: 平均延迟低于100ms"
    else
        log_warning "延迟测试警告: 平均延迟高于100ms"
    fi
}

# 主函数
main() {
    log_info "开始 MCP2REST 性能测试..."
    
    # 检查依赖
    check_dependencies
    
    # 运行延迟测试
    latency_test
    echo
    
    # 运行压力测试
    stress_test
    
    log_success "性能测试完成"
}

# 运行主函数
main "$@"
