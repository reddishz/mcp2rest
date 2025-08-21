#!/bin/bash

# 简单的 MCP 测试脚本
# 用于快速验证 MCP 协议通信

set -e

# 配置
SERVER_PATH="./bin/mcp2rest"
CONFIG_PATH="./configs/bmc_api.yaml"

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

# 测试 MCP 通信
test_mcp_communication() {
    log_info "测试 MCP 协议通信..."
    
    # 创建测试请求
    local test_request='{
        "jsonrpc": "2.0",
        "id": "test_001",
        "method": "toolCall",
        "params": {
            "name": "list",
            "parameters": {
                "page": 1,
                "limit": 5
            }
        }
    }'
    
    # 发送请求并获取响应
    local response=$(echo "$test_request" | $SERVER_PATH -config $CONFIG_PATH)
    
    # 检查响应
    if echo "$response" | grep -q '"jsonrpc":"2.0"'; then
        log_success "MCP 协议通信测试成功"
        echo "响应: $response"
    else
        log_error "MCP 协议通信测试失败"
        echo "响应: $response"
        exit 1
    fi
}

# 测试工具调用
test_tool_call() {
    log_info "测试工具调用..."
    
    # 测试列表查询
    local list_request='{
        "jsonrpc": "2.0",
        "id": "test_list_001",
        "method": "toolCall",
        "params": {
            "name": "list",
            "parameters": {
                "page": 1,
                "limit": 3,
                "sort": "created",
                "order": "desc"
            }
        }
    }'
    
    local response=$(echo "$list_request" | $SERVER_PATH -config $CONFIG_PATH)
    
    if echo "$response" | grep -q '"result"'; then
        log_success "工具调用测试成功"
        echo "响应: $response"
    else
        log_error "工具调用测试失败"
        echo "响应: $response"
        exit 1
    fi
}

# 主函数
main() {
    log_info "开始简单 MCP 测试..."
    
    # 检查依赖
    check_dependencies
    
    # 测试 MCP 通信
    test_mcp_communication
    
    # 测试工具调用
    test_tool_call
    
    log_success "所有测试完成"
}

# 运行主函数
main "$@"
