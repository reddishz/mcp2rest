#!/bin/bash

# SSE 测试脚本
# 用于测试 MCP2REST-SSE 服务器的功能

echo "===== MCP2REST-SSE 测试脚本 ====="

# 设置环境变量
export APIKEYAUTH_API_KEY="ded45a001ffb9c47b1e29fcbdd6bcec6"

# 检查 SSE 服务器是否运行
echo "检查 SSE 服务器状态..."
if curl -s http://localhost:8088/sse > /dev/null 2>&1; then
    echo "✅ SSE 服务器正在运行"
else
    echo "❌ SSE 服务器未运行，正在启动..."
    ./bin/mcp2rest-sse -config configs/bmc_api.yaml &
    SERVER_PID=$!
    sleep 3
    
    if ! curl -s http://localhost:8088/sse > /dev/null 2>&1; then
        echo "❌ SSE 服务器启动失败"
        exit 1
    fi
    echo "✅ SSE 服务器已启动 (PID: $SERVER_PID)"
fi

echo ""
echo "===== 测试 1: 建立 SSE 连接并获取会话端点 ====="
echo "发送 GET 请求建立 SSE 连接..."

# 启动 SSE 连接并捕获端点信息
SSE_OUTPUT=$(curl -N -H "Accept: text/event-stream" http://localhost:8088/sse 2>&1 | head -n 10) &
SSE_PID=$!

# 等待连接建立
sleep 3

# 从输出中提取会话端点
SESSION_ENDPOINT=$(echo "$SSE_OUTPUT" | grep "event: endpoint" | sed 's/event: endpoint//' | tr -d '\n\r ')
if [ -z "$SESSION_ENDPOINT" ]; then
    echo "❌ 无法获取会话端点"
    kill $SSE_PID 2>/dev/null
    exit 1
fi

echo "✅ 获取到会话端点: $SESSION_ENDPOINT"

# 构建完整的消息端点 URL
MESSAGE_URL="http://localhost:8088$SESSION_ENDPOINT"
echo "消息端点: $MESSAGE_URL"

echo ""
echo "===== 测试 2: 发送 MCP 初始化请求 ====="
echo "发送 POST 请求进行 MCP 初始化..."

INIT_REQUEST='{
  "jsonrpc": "2.0",
  "id": 0,
  "method": "initialize",
  "params": {
    "protocolVersion": "2024-11-05",
    "capabilities": {
      "tools": {
        "listChanged": true
      },
      "resources": {
        "subscribe": true,
        "unsubscribe": true
      },
      "logging": {
        "logMessage": true
      },
      "streamableHttp": {
        "request": true
      }
    },
    "clientInfo": {
      "name": "mcp2rest-test-client",
      "version": "1.0.0"
    }
  }
}'

echo "请求内容:"
echo "$INIT_REQUEST" | jq '.'

echo ""
echo "发送请求..."
RESPONSE=$(curl -s -X POST "$MESSAGE_URL" \
  -H "Content-Type: application/json" \
  -d "$INIT_REQUEST")

echo "响应内容:"
echo "$RESPONSE" | jq '.'

echo ""
echo "===== 测试 3: 发送初始化完成通知 ====="
INITIALIZED_REQUEST='{
  "jsonrpc": "2.0",
  "method": "notifications/initialized",
  "params": {}
}'

echo "发送初始化完成通知..."
INITIALIZED_RESPONSE=$(curl -s -X POST "$MESSAGE_URL" \
  -H "Content-Type: application/json" \
  -d "$INITIALIZED_REQUEST")

echo "响应内容:"
echo "$INITIALIZED_RESPONSE" | jq '.'

echo ""
echo "===== 测试 4: 发送工具列表请求 ====="
TOOLS_REQUEST='{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/list",
  "params": {}
}'

echo "发送工具列表请求..."
TOOLS_RESPONSE=$(curl -s -X POST "$MESSAGE_URL" \
  -H "Content-Type: application/json" \
  -d "$TOOLS_REQUEST")

echo "响应内容:"
echo "$TOOLS_RESPONSE" | jq '.'

echo ""
echo "===== 测试完成 ====="

# 清理
if [ ! -z "$SSE_PID" ]; then
    echo "清理 SSE 连接..."
    kill $SSE_PID 2>/dev/null
fi

if [ ! -z "$SERVER_PID" ]; then
    echo "停止 SSE 服务器..."
    kill $SERVER_PID 2>/dev/null
fi

echo "测试完成！"
