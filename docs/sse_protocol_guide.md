# MCP 协议 SSE 传输层实现指南

## 概述

MCP (Model Context Protocol) 协议支持多种传输层，包括 stdio 和 SSE (Server-Sent Events)。本文档详细说明了 SSE 传输层的正确实现方式，严格按照 MCP 官方规范。

## SSE 传输层架构

### MCP SSE 规范设计

按照 MCP 官方规范，SSE 传输层使用以下设计：

1. **GET /sse**：建立 SSE 长连接，接收服务端推送的所有消息
2. **POST /messages/?session_id=xxx**：客户端向服务端发送消息的专用端点
3. **会话管理**：使用 session_id 维护客户端与服务端的会话状态
4. **双向异步通信**：接收通道通过 SSE 长连接，发送通道通过 HTTP POST

### 协议流程

```
客户端                                   服务器
   |                                       |
   |-- GET /sse (建立SSE连接) ------------>|
   |<-- event: endpoint                    |
   |<-- data: /messages/?session_id=xxx ---|
   |                                       |
   |-- POST /messages/?session_id=xxx ----->|
   |<-- 202 Accepted ----------------------|
   |<-- event: message                     |
   |<-- data: {"jsonrpc":"2.0",...} ------|
   |                                       |
   |<-- event: heartbeat                   |
   |<-- data: {"timestamp":"...",...} ----|
```

## 实现细节

### 1. 建立 SSE 连接 (GET /sse)

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

**专用消息端点：**
```
event: endpoint
data: /messages/?session_id=2b3c8777119444c1a1b26bc0d0f05a0a

```

### 2. 消息发送 (POST /messages/?session_id=xxx)

```http
POST /messages/?session_id=2b3c8777119444c1a1b26bc0d0f05a0a HTTP/1.1
Content-Type: application/json

{
  "jsonrpc": "2.0",
  "id": 0,
  "method": "initialize",
  "params": {
    "protocolVersion": "2024-11-05",
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
HTTP/1.1 202 Accepted
Content-Type: application/json

{"status":"Accepted"}
```

**SSE 推送的实际响应：**
```
event: message
data: {"jsonrpc":"2.0","id":0,"result":{...}}

```

### 3. 心跳机制

服务器每 30 秒发送一次心跳消息：

```
event: heartbeat
data: {"timestamp":"2024-01-01T12:00:00Z","session_id":"2b3c8777119444c1a1b26bc0d0f05a0a"}

```

## 会话管理

### 会话标识

每个 SSE 连接都有唯一的会话标识符：
- 格式：MD5 哈希值
- 示例：`2b3c8777119444c1a1b26bc0d0f05a0a`

### 会话生命周期

1. **建立**：客户端发送 GET /sse 请求
2. **分配**：服务器生成 session_id 并返回专用端点
3. **活跃**：客户端使用专用端点发送消息
4. **清理**：连接断开时自动清理会话

### 会话状态管理

```go
type MCPSession struct {
    ID           string
    ClientID     string
    Endpoint     string
    CreatedAt    time.Time
    LastActivity time.Time
}
```

## 使用示例

### 客户端实现

```javascript
// 建立 SSE 连接
const eventSource = new EventSource('http://localhost:8088/sse');
let messageEndpoint = null;

eventSource.addEventListener('endpoint', function(event) {
    messageEndpoint = event.data;
    console.log('获取到消息端点:', messageEndpoint);
});

eventSource.addEventListener('message', function(event) {
    const data = JSON.parse(event.data);
    console.log('收到消息:', data);
});

eventSource.addEventListener('heartbeat', function(event) {
    const data = JSON.parse(event.data);
    console.log('心跳:', data);
});

// 发送 MCP 请求
async function sendMCPRequest(method, params, id = null) {
    if (!messageEndpoint) {
        throw new Error('消息端点未初始化');
    }
    
    const response = await fetch(`http://localhost:8088${messageEndpoint}`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({
            jsonrpc: '2.0',
            id: id,
            method: method,
            params: params
        })
    });
    
    return response.json();
}

// 初始化流程
async function initialize() {
    // 1. 发送初始化请求
    await sendMCPRequest('initialize', {
        protocolVersion: '2024-11-05',
        capabilities: {},
        clientInfo: {
            name: 'test-client',
            version: '1.0'
        }
    }, 0);
    
    // 2. 发送初始化完成通知
    await sendMCPRequest('notifications/initialized', {}, null);
}
```

### 服务器端实现

```go
// 按照 MCP 规范设置端点
func (s *Server) startSSEServer() error {
    mux := http.NewServeMux()
    
    // 按照 MCP SSE 规范设置端点
    mux.HandleFunc("/sse", s.handleSSEConnection)           // GET: 建立 SSE 连接
    mux.HandleFunc("/messages/", s.handleMCPMessages)       // POST: 处理 MCP 消息
    
    // ... 服务器配置
}

// SSE 连接处理 - 返回专用消息端点
func (s *Server) handleSSEConnection(w http.ResponseWriter, r *http.Request) {
    // 设置 SSE 头
    // 创建会话
    // 生成 session_id
    
    // 按照 MCP 规范发送专用消息端点
    endpointMessage := fmt.Sprintf("event: endpoint\ndata: %s\n\n", session.Endpoint)
    fmt.Fprint(w, endpointMessage)
    flusher.Flush()
    
    // 保持连接活跃
    for {
        select {
        case <-ctx.Done():
            return
        case <-time.After(30 * time.Second):
            // 发送心跳
            heartbeatMessage := fmt.Sprintf("event: heartbeat\ndata: {...}\n\n")
            fmt.Fprint(w, heartbeatMessage)
            flusher.Flush()
        }
    }
}

// 消息处理 - 返回 Accepted 状态码
func (s *Server) handleMCPMessages(w http.ResponseWriter, r *http.Request) {
    // 验证 session_id
    // 处理 MCP 消息
    
    // 按照 MCP 规范，返回 "Accepted" 状态码
    w.WriteHeader(http.StatusAccepted)
    w.Write([]byte(`{"status":"Accepted"}`))
    
    // 通过 SSE 连接推送实际响应
    s.pushMessageToSession(sessionID, response)
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
3. 获取专用消息端点
4. 发送 MCP 初始化请求 (POST /messages/?session_id=xxx)
5. 发送初始化完成通知
6. 发送工具列表请求
7. 验证响应格式

## 优势

### 1. 符合 MCP 官方规范
- 严格按照 MCP 协议规范实现
- 支持会话管理和专用端点
- 正确的 SSE 事件格式

### 2. 双向异步通信
- SSE 长连接接收服务端消息
- HTTP POST 发送客户端消息
- 真正的异步通信模式

### 3. 会话状态管理
- 使用 session_id 维护会话状态
- 支持多客户端并发连接
- 自动会话清理

### 4. 标准通信协议
- 基于 JSON-RPC 2.0 规范
- 统一的通信格式
- 跨平台兼容性

## 注意事项

1. **会话端点**：所有客户端请求必须使用服务端分配的专用端点
2. **初始化流程**：必须严格按照顺序完成初始化
3. **状态码**：消息请求返回 202 Accepted，实际响应通过 SSE 推送
4. **事件格式**：使用 `event: message` 和 `data:` 格式
5. **会话管理**：及时清理断开的会话

## 与 stdio 模式的对比

| 特性 | SSE 模式 | stdio 模式 |
|------|----------|------------|
| 传输协议 | HTTP/SSE | 标准输入输出 |
| 连接方式 | 网络连接 | 进程间通信 |
| 并发支持 | 多会话 | 单连接 |
| 会话管理 | session_id | 无 |
| 端点设计 | 专用端点 | 单一端点 |
| 适用场景 | Web 应用 | 本地工具 |
| 复杂度 | 中等 | 简单 |

## 总结

通过严格按照 MCP 官方规范实现，SSE 传输层提供了完整的会话管理、双向异步通信和标准化的消息格式，为 MCP 协议提供了稳定可靠的基于 HTTP 的实时通信能力，特别适合 Web 应用和浏览器环境。
