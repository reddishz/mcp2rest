# MCP 协议 SSE 传输层实现指南

## 概述

MCP (Model Context Protocol) 协议支持多种传输层，包括 stdio 和 SSE (Server-Sent Events)。本文档详细说明了 SSE 传输层的正确实现方式。

## SSE 传输层架构

### 双向通信模式

SSE 传输层通过以下方式实现双向通信：

1. **GET 请求**：建立长连接，用于服务器向客户端推送消息
2. **POST 请求**：客户端向服务器发送消息
3. **连接管理**：服务器维护多个 SSE 连接，支持广播和定向推送

### 协议流程

```
客户端                                   服务器
   |                                       |
   |-- GET / (建立SSE连接) ---------------->|
   |<-- data: {"type":"connected",...} ----|
   |                                       |
   |-- POST / (发送MCP请求) --------------->|
   |<-- data: {"jsonrpc":"2.0",...} -------|
   |                                       |
   |<-- data: {"type":"heartbeat",...} ----|
   |                                       |
```

## 实现细节

### 1. 连接建立 (GET 请求)

```http
GET / HTTP/1.1
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

### 2. 消息发送 (POST 请求)

```http
POST / HTTP/1.1
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
```
data: {"jsonrpc":"2.0","id":"1","result":{...}}

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

1. **建立**：客户端发送 GET 请求
2. **注册**：服务器将连接添加到连接池
3. **活跃**：定期发送心跳，处理消息
4. **清理**：连接断开时自动清理

### 错误处理

- **连接断开**：自动检测并清理连接
- **写入失败**：记录错误并关闭连接
- **超时处理**：心跳超时自动清理

## 使用示例

### 客户端实现

```javascript
// 建立 SSE 连接
const eventSource = new EventSource('http://localhost:8088/');

eventSource.onmessage = function(event) {
    const data = JSON.parse(event.data);
    console.log('收到消息:', data);
};

eventSource.onopen = function() {
    console.log('SSE 连接已建立');
};

// 发送 MCP 请求
async function sendMCPRequest(method, params) {
    const response = await fetch('http://localhost:8088/', {
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
    
    return response.text();
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
// 连接管理
type SSEConnection struct {
    ID         string
    Writer     http.ResponseWriter
    Flusher    http.Flusher
    Context    context.Context
    Cancel     context.CancelFunc
    RemoteAddr string
}

// 广播消息
func (s *Server) broadcastToSSE(message []byte) {
    s.sseMutex.RLock()
    defer s.sseMutex.RUnlock()

    for clientID, conn := range s.sseConnections {
        select {
        case <-conn.Context.Done():
            continue
        default:
            fmt.Fprintf(conn.Writer, "data: %s\n\n", string(message))
            conn.Flusher.Flush()
        }
    }
}
```

## 测试

使用提供的测试脚本验证实现：

```bash
./scripts/test_sse.sh
```

测试脚本会：
1. 启动 SSE 服务器
2. 建立 SSE 连接
3. 发送 MCP 初始化请求
4. 发送工具列表请求
5. 验证响应格式

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
| 适用场景 | Web 应用 | 本地工具 |
| 复杂度 | 中等 | 简单 |

## 总结

SSE 传输层为 MCP 协议提供了基于 HTTP 的实时双向通信能力，特别适合 Web 应用和浏览器环境。通过正确的连接管理和消息处理，可以实现稳定可靠的 MCP 通信。
