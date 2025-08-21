#!/bin/bash

# MCP2REST 测试脚本
# 用于测试 MCP 协议通信和 REST API 功能

set -e

# 配置
SERVER_PATH="./bin/mcp2rest"
CONFIG_PATH="./configs/bmc_api.yaml"
TEST_DATA_DIR="./test_data"

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

# 启动服务器
start_server() {
    log_info "启动 MCP2REST 服务器..."
    
    # 启动服务器进程
    $SERVER_PATH -config $CONFIG_PATH &
    SERVER_PID=$!
    
    # 等待服务器启动
    sleep 3
    
    # 检查进程是否还在运行
    if ! kill -0 $SERVER_PID 2>/dev/null; then
        log_error "服务器启动失败"
        exit 1
    fi
    
    log_success "服务器启动成功 (PID: $SERVER_PID)"
}

# 停止服务器
stop_server() {
    if [ ! -z "$SERVER_PID" ]; then
        log_info "停止服务器 (PID: $SERVER_PID)..."
        kill $SERVER_PID 2>/dev/null || true
        wait $SERVER_PID 2>/dev/null || true
        log_success "服务器已停止"
    fi
}

# 发送 MCP 请求
send_mcp_request() {
    local method="$1"
    local params="$2"
    local request_id="$3"
    
    # 构建 MCP 请求（单行JSON）
    local request="{\"jsonrpc\":\"2.0\",\"id\":\"$request_id\",\"method\":\"$method\",\"params\":$params}"
    
    echo "$request"
}

# 测试 BMC 列表查询
test_list() {
    log_info "测试 BMC 列表查询..."
    
    local params='{"name":"getList","parameters":{"page":1,"limit":5,"sort":"created","order":"desc"}}'
    
    local response=$(send_mcp_request "toolCall" "$params" "test_list_001")
    echo "$response" | $SERVER_PATH -config $CONFIG_PATH
    
    log_success "BMC 列表查询测试完成"
}

# 测试 BMC 详情查询
test_detail() {
    log_info "测试 BMC 详情查询..."
    
    local params='{"name":"getDetail","parameters":{"id":"test_bmc_001"}}'
    
    local response=$(send_mcp_request "toolCall" "$params" "test_detail_001")
    echo "$response" | $SERVER_PATH -config $CONFIG_PATH
    
    log_success "BMC 详情查询测试完成"
}

# 测试 BMC 搜索
test_search() {
    log_info "测试 BMC 搜索..."
    
    local params='{"name":"getSearch","parameters":{"q":"测试","page":1,"limit":10,"fields":"title,description"}}'
    
    local response=$(send_mcp_request "toolCall" "$params" "test_search_001")
    echo "$response" | $SERVER_PATH -config $CONFIG_PATH
    
    log_success "BMC 搜索测试完成"
}

# 测试 BMC 创建
test_create() {
    log_info "测试 BMC 创建..."
    
    local params='{"name":"postCreate","parameters":{"id":"test_bmc_001","title":"测试 BMC","description":"这是一个测试用的 BMC 数据","bmc":{"customerSegments":["企业用户","个人用户"],"valuePropositions":["高效解决方案","优质服务"],"channels":["官网","合作伙伴"],"customerRelationships":["长期合作","技术支持"],"keyResources":["技术团队","品牌声誉"],"keyActivities":["产品开发","市场推广"],"keyPartnerships":["技术供应商","渠道伙伴"],"costStructure":["研发成本","运营成本"],"revenueStreams":["产品销售","服务收费"]}}}'
    
    local response=$(send_mcp_request "toolCall" "$params" "test_create_001")
    echo "$response" | $SERVER_PATH -config $CONFIG_PATH
    
    log_success "BMC 创建测试完成"
}

# 测试 BMC 更新
test_update() {
    log_info "测试 BMC 更新..."
    
    local params='{"name":"postUpdate","parameters":{"id":"test_bmc_001","title":"更新后的测试 BMC"}}'
    
    local response=$(send_mcp_request "toolCall" "$params" "test_update_001")
    echo "$response" | $SERVER_PATH -config $CONFIG_PATH
    
    log_success "BMC 更新测试完成"
}

# 测试 BMC 删除
test_delete() {
    log_info "测试 BMC 删除..."
    
    local params='{"name":"postDelete","parameters":{"id":"test_bmc_001"}}'
    
    local response=$(send_mcp_request "toolCall" "$params" "test_delete_001")
    echo "$response" | $SERVER_PATH -config $CONFIG_PATH
    
    log_success "BMC 删除测试完成"
}

# 运行所有测试
run_all_tests() {
    log_info "开始运行所有测试..."
    
    test_list
    test_detail
    test_search
    test_create
    test_update
    test_delete
    
    log_success "所有测试完成"
}

# 主函数
main() {
    log_info "开始 MCP2REST 测试..."
    
    # 设置退出时清理
    trap stop_server EXIT
    
    # 检查依赖
    check_dependencies
    
    # 运行测试
    run_all_tests
    
    log_success "测试完成"
}

# 运行主函数
main "$@"
