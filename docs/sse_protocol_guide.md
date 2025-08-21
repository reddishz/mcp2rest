# MCP 协议 SSE 传输层实现指南

## 概述

MCP (Model Context Protocol) 协议支持多种传输层，包括 stdio 和 SSE (Server-Sent Events)。本文档详细说明了 SSE 传输层的正确实现方式。

## SSE 传输层架构

### 分离端点设计

为了避免阻塞问题和协议混淆，SSE 传输层使用分离的端点：

1. **GET /sse**：建立 SSE 长连接，用于服务器向客户端推送消息
2. **POST /api**：客户端向服务器发送 MCP 消息
3. **连接管理**：服务器维护多个 SSE 连接，支持广播和定向推送

### 协议流程

```
客户端                                   服务器
   |                                       |
   |-- GET /sse (建立SSE连接) ------------>|
   |<-- data: {"type":"connected",...} ----|
   |                                       |
   |-- POST /api (发送MCP请求) ----------->|
   |<-- {"jsonrpc":"2.0",...} -------------|
   |                                       |
   |<-- data: {"type":"heartbeat",...} ----|
   |                                       |
```

## 实现细节

### 1. 连接建立 (GET /sse)

```http
GET /sse HTTP/1.1
Accept: text/event-stream
Cache-Control: no-cache
```

**服务器响应头：**
```http
HTTP/1.1 200 OK
Content-Type: text/event-stream
Cache-Control: no-cache
Connection: keep-alive
Access-Control-Allow-Origin: *
X-Accel-Buffering: no
```

**连接确认消息：**
```
data: {"type":"connected","message":"SSE连接已建立","clientId":"127.0.0.1-1234567890"}

```

### 2. 消息发送 (POST /api)

```http
POST /api HTTP/1.1
Content-Type: application/json

{
  "jsonrpc": "2.0",
  "id": "1",
  "method": "initialize",
  "params": {
    "protocolVersion": "20241105",
    "capabilities": {},
    "clientInfo": {
      "name": "test-client",
      "version": "1.0"
    }
  }
}
```

**服务器响应：**
```http
HTTP/1.1 200 OK
Content-Type: application/json

{"jsonrpc":"2.0","id":"1","result":{...}}
```

### 3. 心跳机制

服务器每 30 秒发送一次心跳消息：

```
data: {"type":"heartbeat","timestamp":"2024-01-01T12:00:00Z","clientId":"127.0.0.1-1234567890"}

```

## 连接管理

### 连接标识

每个 SSE 连接都有唯一的标识符：
- 格式：`{remote_addr}-{timestamp}`
- 示例：`127.0.0.1-1704067200000`

### 连接生命周期

1. **建立**：客户端发送 GET /sse 请求
2. **注册**：服务器将连接添加到连接池
3. **活跃**：在独立的 goroutine 中处理心跳
4. **清理**：连接断开时自动清理

### 非阻塞设计

- **GET /sse**：在 HTTP 处理函数中保持连接活跃，确保连接不关闭
- **POST /api**：独立的端点，不受 SSE 连接影响
- **并发支持**：可以同时处理多个连接和消息

## 使用示例

### 客户端实现

```javascript
// 建立 SSE 连接
const eventSource = new EventSource('http://localhost:8088/sse');

eventSource.onmessage = function(event) {
    const data = JSON.parse(event.data);
    console.log('收到消息:', data);
};

eventSource.onopen = function() {
    console.log('SSE 连接已建立');
};

// 发送 MCP 请求
async function sendMCPRequest(method, params) {
    const response = await fetch('http://localhost:8088/api', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({
            jsonrpc: '2.0',
            id: Date.now().toString(),
            method: method,
            params: params
        })
    });
    
    return response.json();
}

// 初始化连接
sendMCPRequest('initialize', {
    protocolVersion: '20241105',
    capabilities: {},
    clientInfo: {
        name: 'test-client',
        version: '1.0'
    }
});
```

### 服务器端实现

```go
// 分离的端点处理
func (s *Server) startSSEServer() error {
    mux := http.NewServeMux()
    
    // 分离端点：SSE 连接和消息处理
    mux.HandleFunc("/sse", s.handleSSEConnection)  // GET: 建立 SSE 连接
    mux.HandleFunc("/api", s.handleSSEMessage)     // POST: 处理 MCP 消息
    
    // ... 服务器配置
}

// SSE 连接处理 - 在 HTTP 处理函数中保持连接活跃
func (s *Server) handleSSEConnection(w http.ResponseWriter, r *http.Request) {
    // 设置 SSE 头
    // 创建连接
    // 注册到连接池
    
    // 发送连接确认消息
    fmt.Fprintf(w, "data: {...}\n\n")
    flusher.Flush()
    
    // 在 HTTP 处理函数中保持连接活跃
    for {
        select {
        case <-ctx.Done():
            return
        case <-time.After(30 * time.Second):
            // 发送心跳
            fmt.Fprintf(w, "data: {...}\n\n")
            flusher.Flush()
        }
    }
    // 注意：不能在这里返回，否则连接会关闭
}

// 独立的消息处理
func (s *Server) handleSSEMessage(w http.ResponseWriter, r *http.Request) {
    // 处理 MCP 消息
    // 返回 JSON 响应
}
```

## 测试

使用提供的测试脚本验证实现：

```bash
./scripts/test_sse.sh
```

测试脚本会：
1. 启动 SSE 服务器
2. 建立 SSE 连接 (GET /sse)
3. 发送 MCP 初始化请求 (POST /api)
4. 发送工具列表请求 (POST /api)
5. 验证响应格式

## 优势

### 1. 解决阻塞问题
- GET 请求立即返回，长连接在 goroutine 中处理
- POST 请求独立处理，不受 SSE 连接影响

### 2. 协议清晰分离
- SSE 连接和消息处理使用不同端点
- 职责明确，易于理解和维护

### 3. 支持并发
- 可以同时处理多个 SSE 连接
- 可以同时处理多个消息请求

### 4. 错误隔离
- SSE 连接错误不影响消息处理
- 消息处理错误不影响 SSE 连接

## 注意事项

1. **CORS 支持**：确保设置正确的 CORS 头
2. **代理兼容性**：某些代理服务器可能不支持长连接
3. **心跳间隔**：根据网络环境调整心跳间隔
4. **连接限制**：考虑服务器资源限制，合理管理连接数
5. **错误恢复**：实现客户端重连机制

## 与 stdio 模式的对比

| 特性 | SSE 模式 | stdio 模式 |
|------|----------|------------|
| 传输协议 | HTTP/SSE | 标准输入输出 |
| 连接方式 | 网络连接 | 进程间通信 |
| 并发支持 | 多连接 | 单连接 |
| 端点设计 | 分离端点 | 单一端点 |
| 适用场景 | Web 应用 | 本地工具 |
| 复杂度 | 中等 | 简单 |

## 总结

通过分离端点的设计，SSE 传输层解决了阻塞问题和协议混淆问题，为 MCP 协议提供了稳定可靠的基于 HTTP 的实时双向通信能力，特别适合 Web 应用和浏览器环境。
