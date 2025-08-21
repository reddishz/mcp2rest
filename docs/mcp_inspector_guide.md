# MCP Inspector 连接指南

## 问题说明

MCP Inspector 报错 "The argument 'file' cannot be empty" 是因为 MCP Inspector 期望通过 stdio 连接，而不是 HTTP/SSE 连接。

## 解决方案

### 方案1：使用 stdio 版本（推荐）

MCP Inspector 最适合与 stdio 版本的 MCP2REST 连接：

1. **在 MCP Inspector 中配置**：
   - 连接类型：`stdio`
   - 命令：`./bin/mcp2rest-stdio`
   - 参数：`-config configs/bmc_api.yaml`
   - 工作目录：项目根目录

2. **确保环境变量已设置**：
   ```bash
   # 方法1：使用 .env 文件（自动加载）
   cp configs/.env.example configs/.env
   # 编辑 configs/.env，设置 APIKEYAUTH_API_KEY
   
   # 方法2：手动设置环境变量
   export APIKEYAUTH_API_KEY="ded45a001ffb9c47b1e29fcbdd6bcec6"
   ```

### 方案2：使用 SSE 版本（需要额外配置）

如果必须使用 SSE 版本，需要创建一个包装脚本：

1. **创建包装脚本** `scripts/mcp_inspector_sse.sh`：
   ```bash
   #!/bin/bash
   # 设置环境变量
   export APIKEYAUTH_API_KEY="ded45a001ffb9c47b1e29fcbdd6bcec6"
   
   # 启动 SSE 服务器
   ./bin/mcp2rest-sse -config configs/bmc_api.yaml &
   SERVER_PID=$!
   
   # 等待服务器启动
   sleep 2
   
   # 发送初始化请求
   curl -X POST http://localhost:8088/ \
     -H "Content-Type: application/json" \
     -d '{"jsonrpc":"2.0","id":"1","method":"initialize","params":{"protocolVersion":"20241105","capabilities":{},"clientInfo":{"name":"mcp-inspector","version":"1.0"}}}'
   
   # 保持脚本运行
   wait $SERVER_PID
   ```

2. **在 MCP Inspector 中配置**：
   - 连接类型：`stdio`
   - 命令：`./scripts/mcp_inspector_sse.sh`
   - 参数：（空）
   - 工作目录：项目根目录

## 推荐配置

**最佳实践是使用 stdio 版本**，因为：

1. **符合 MCP 标准**：stdio 是 MCP 协议的标准连接方式
2. **更好的兼容性**：MCP Inspector 原生支持 stdio 连接
3. **更简单的配置**：无需额外的包装脚本
4. **更好的调试**：可以直接看到输入输出

## 故障排除

### 常见问题

1. **"The argument 'file' cannot be empty"**
   - 原因：MCP Inspector 期望 stdio 连接，但收到了 HTTP 连接
   - 解决：使用 stdio 版本或创建包装脚本

2. **"Connection refused"**
   - 原因：SSE 服务器未启动
   - 解决：先启动 SSE 服务器

3. **"API Key not found"**
   - 原因：环境变量未设置
   - 解决：设置 `APIKEYAUTH_API_KEY` 环境变量

### 调试步骤

1. **测试 stdio 版本**：
   ```bash
   echo '{"jsonrpc":"2.0","id":"test","method":"initialize","params":{"protocolVersion":"20241105","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}' | ./bin/mcp2rest-stdio -config configs/bmc_api.yaml
   ```

2. **测试 SSE 版本**：
   ```bash
   ./bin/mcp2rest-sse -config configs/bmc_api.yaml &
   curl -X POST http://localhost:8088/ -H "Content-Type: application/json" -d '{"jsonrpc":"2.0","id":"test","method":"initialize","params":{"protocolVersion":"20241105","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}'
   ```

3. **检查环境变量**：
   ```bash
   echo $APIKEYAUTH_API_KEY
   ```
