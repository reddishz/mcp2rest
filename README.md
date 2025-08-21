# MCP2REST

MCP2REST 是一个将 REST API 转换为 MCP (Message Control Protocol) 的工具，支持两种运行模式。

## 项目结构

```
mcp2rest/
├── cmd/
│   ├── mcp2rest/          # 原始版本（支持 stdio 和 sse 模式）
│   ├── mcp2rest-stdio/    # 专用 stdio 版本
│   └── mcp2rest-sse/      # 专用 SSE 版本
├── bin/
│   ├── mcp2rest           # 原始可执行文件
│   ├── mcp2rest-stdio     # stdio 版本可执行文件
│   └── mcp2rest-sse       # SSE 版本可执行文件
├── configs/               # 配置文件
├── internal/              # 内部包
└── pkg/                   # 公共包
```

## 版本说明

### 1. MCP2REST-STDIO

专门用于通过标准输入/输出与 MCP 客户端通信。

**特点：**
- 使用 stdin/stdout 进行通信，符合 MCP 标准协议
- 自动跟随 MCP 客户端进程的启动和关闭
- 无需网络端口，直接进程间通信
- 高性能协程池处理并发请求

**使用场景：**
- MCP 客户端集成
- 本地工具链集成
- 需要进程生命周期管理的场景

**编译和运行：**
```bash
# 编译
go build -o bin/mcp2rest-stdio cmd/mcp2rest-stdio/main.go

# 运行
export APIKEYAUTH_API_KEY="your_api_key"
./bin/mcp2rest-stdio -config configs/bmc_api.yaml

# 测试
echo '{"jsonrpc":"2.0","id":"test","method":"initialize","params":{"protocolVersion":"20241105","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}' | ./bin/mcp2rest-stdio -config configs/bmc_api.yaml
```

### 2. MCP2REST-SSE

专门用于通过 Server-Sent Events 与 Web 客户端通信。

**特点：**
- 使用 SSE 协议进行实时通信
- 支持浏览器和 Web 客户端
- 提供标准的 HTTP 接口
- 自动发送心跳保持连接活跃

**使用场景：**
- Web 应用集成
- 浏览器扩展
- 需要实时通信的场景
- 跨网络通信

**编译和运行：**
```bash
# 编译
go build -o bin/mcp2rest-sse cmd/mcp2rest-sse/main.go

# 运行
export APIKEYAUTH_API_KEY="your_api_key"
./bin/mcp2rest-sse -config configs/bmc_api.yaml

# 测试
# 建立 SSE 连接
curl -N http://localhost:8088/

# 发送请求
curl -X POST http://localhost:8088/ \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":"test","method":"initialize","params":{"protocolVersion":"20241105","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}'
```

### 3. MCP2REST（原始版本）

支持两种模式的通用版本，通过配置文件选择运行模式。

**编译和运行：**
```bash
# 编译
go build -o bin/mcp2rest cmd/mcp2rest/main.go

# 运行（stdio 模式）
./bin/mcp2rest -config configs/bmc_api.yaml

# 运行（sse 模式）
# 修改 configs/server.yaml 中的 mode 为 "sse"
./bin/mcp2rest -config configs/bmc_api.yaml
```

## 配置

### 配置文件

配置文件位于 `configs/` 目录：
- `bmc_api.yaml`: OpenAPI 规范文件
- `stdio.yaml`: stdio 版本专用配置
- `sse.yaml`: SSE 版本专用配置

### 环境变量配置

MCP2REST 使用环境变量来配置 API 认证信息，**支持自动加载 `.env` 文件**。

#### 自动加载（推荐）

程序启动时会自动查找并加载 `.env` 文件，无需手动设置环境变量：

1. **复制环境变量模板**：
```bash
cp configs/.env.example configs/.env
```

2. **编辑 `.env` 文件**，填入实际的认证信息：
```bash
# BMC API 认证
APIKEYAUTH_API_KEY=your_actual_bmc_api_key_here

# 其他配置...
```

3. **直接运行程序**：
```bash
./bin/mcp2rest-stdio -config configs/bmc_api.yaml
```

程序会自动查找以下位置的 `.env` 文件：
- 当前工作目录：`.env`
- configs 目录：`configs/.env`
- 可执行文件同级目录：`./.env`
- 可执行文件同级 configs 目录：`./configs/.env`
- 可执行文件上级目录：`../.env`
- 可执行文件上级 configs 目录：`../configs/.env`

#### 手动设置（备选）

如果不想使用 `.env` 文件，也可以手动设置环境变量：

```bash
# 方法1：使用 source 命令
source configs/.env

# 方法2：使用 export 命令
export APIKEYAUTH_API_KEY="your_actual_bmc_api_key_here"
```

**重要说明**：
- `.env` 文件包含敏感信息，不会被提交到版本控制中
- 程序启动时会显示是否找到并加载了 `.env` 文件
- 如果找到多个 `.env` 文件，会使用第一个找到的文件
- 测试时可以使用提供的示例 API Key：`ded45a001ffb9c47b1e29fcbdd6bcec6`

## 主要改进

1. **彻底删除 WebSocket**: 移除了所有 WebSocket 相关代码
2. **实现正确的 SSE**: 使用 Server-Sent Events 协议
3. **分离版本**: 创建了专用的 stdio 和 sse 版本
4. **进程管理优化**: 改进了进程生命周期管理
5. **日志优化**: 按进程 ID 命名日志文件

## 技术栈

- **Go**: 主要编程语言
- **OpenAPI 3.0**: API 规范
- **JSON-RPC 2.0**: MCP 协议基础
- **SSE**: Server-Sent Events 协议
- **YAML**: 配置文件格式