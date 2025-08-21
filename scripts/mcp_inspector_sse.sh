#!/bin/bash

# MCP Inspector SSE 包装脚本
# 用于连接 MCP2REST-SSE 服务器

# 设置环境变量
export APIKEYAUTH_API_KEY="ded45a001ffb9c47b1e29fcbdd6bcec6"

# 检查 SSE 服务器是否运行
if ! curl -s http://localhost:8088/ > /dev/null 2>&1; then
    echo "启动 SSE 服务器..."
    ./bin/mcp2rest-sse -config configs/bmc_api.yaml &
    SERVER_PID=$!
    
    # 等待服务器启动
    sleep 3
    
    # 检查服务器是否成功启动
    if ! curl -s http://localhost:8088/ > /dev/null 2>&1; then
        echo "错误: SSE 服务器启动失败"
        exit 1
    fi
    
    echo "SSE 服务器已启动 (PID: $SERVER_PID)"
else
    echo "SSE 服务器已在运行"
fi

# 发送初始化请求
echo "发送初始化请求..."
curl -X POST http://localhost:8088/ \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":"1","method":"initialize","params":{"protocolVersion":"20241105","capabilities":{},"clientInfo":{"name":"mcp-inspector","version":"1.0"}}}'

echo ""
echo "SSE 服务器连接信息："
echo "URL: http://localhost:8088/"
echo "方法: POST"
echo "Content-Type: application/json"
echo ""
echo "注意：MCP Inspector 可能无法直接连接 SSE 服务器。"
echo "建议使用 stdio 版本：./bin/mcp2rest-stdio -config configs/bmc_api.yaml"
