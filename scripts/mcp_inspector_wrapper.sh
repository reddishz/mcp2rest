#!/bin/bash

# MCP Inspector 包装脚本
# 用于连接 MCP2REST-SSE 服务器

# 设置环境变量
export APIKEYAUTH_API_KEY="ded45a001ffb9c47b1e29fcbdd6bcec6"

# 检查 SSE 服务器是否运行
if ! curl -s http://localhost:8088/ > /dev/null 2>&1; then
    echo "错误: SSE 服务器未运行，请先启动 mcp2rest-sse"
    echo "启动命令: ./bin/mcp2rest-sse -config configs/bmc_api.yaml"
    exit 1
fi

echo "连接到 MCP2REST-SSE 服务器 (http://localhost:8088/)"
echo "使用 MCP Inspector 时，请选择 'stdio' 模式，并输入以下命令："
echo ""
echo "命令: curl"
echo "参数: -X POST -H 'Content-Type: application/json' -d '{\"jsonrpc\":\"2.0\",\"id\":\"1\",\"method\":\"initialize\",\"params\":{\"protocolVersion\":\"20241105\",\"capabilities\":{},\"clientInfo\":{\"name\":\"mcp-inspector\",\"version\":\"1.0\"}}}' http://localhost:8088/"
echo ""
echo "或者直接使用 stdio 版本："
echo "命令: ./bin/mcp2rest-stdio"
echo "参数: -config configs/bmc_api.yaml"
