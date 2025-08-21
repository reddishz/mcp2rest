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
echo "===== 测试 1: 建立 SSE 连接 ====="
echo "发送 GET 请求建立 SSE 连接..."
curl -N -H "Accept: text/event-stream" http://localhost:8088/sse &
SSE_PID=$!

# 等待连接建立
sleep 2

echo ""
echo "===== 测试 2: 发送 MCP 初始化请求 ====="
echo "发送 POST 请求进行 MCP 初始化..."

INIT_REQUEST='{
  "jsonrpc": "2.0",
  "id": "test_init_1",
  "method": "initialize",
  "params": {
    "protocolVersion": "20241105",
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
RESPONSE=$(curl -s -X POST http://localhost:8088/api \
  -H "Content-Type: application/json" \
  -d "$INIT_REQUEST")

echo "响应内容:"
echo "$RESPONSE" | jq '.'

echo ""
echo "===== 测试 3: 发送工具列表请求 ====="
TOOLS_REQUEST='{
  "jsonrpc": "2.0",
  "id": "test_tools_1",
  "method": "tools/list",
  "params": {}
}'

echo "发送工具列表请求..."
TOOLS_RESPONSE=$(curl -s -X POST http://localhost:8088/api \
  -H "Content-Type: application/json" \
  -d "$TOOLS_REQUEST")

echo "工具列表响应:"
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
