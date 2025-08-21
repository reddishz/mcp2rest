# MCP2REST 服务器优化总结

本文档总结了 `startStdioServer` 函数的优化改进。

## 优化概述

我们对 `internal/server/server.go` 中的 `startStdioServer` 函数进行了全面优化，主要改进了以下几个方面：

1. **缓冲区管理优化**
2. **并发处理能力**
3. **错误处理机制**
4. **性能监控**
5. **超时控制**
6. **日志记录**

## 详细优化内容

### 1. 缓冲区管理优化

**优化前：**
```go
var buffer [4096]byte
n, err := os.Stdin.Read(buffer[:])
```

**优化后：**
```go
reader := bufio.NewReaderSize(os.Stdin, 64*1024) // 64KB 缓冲区
writer := bufio.NewWriterSize(os.Stdout, 64*1024) // 64KB 缓冲区
line, err := reader.ReadString('\n')
```

**改进效果：**
- ✅ 支持更大的请求数据
- ✅ 提高 I/O 效率
- ✅ 正确处理换行符分隔的 JSON 消息
- ✅ 避免数据截断问题

### 2. 并发处理能力

**新增功能：**
```go
// 创建请求通道，用于并发处理
requestChan := make(chan *requestTask, 100) // 缓冲通道

// 启动工作协程池
workerCount := 4 // 可以根据需要调整
for i := 0; i < workerCount; i++ {
    go s.stdioWorker(requestChan, writer)
}
```

**改进效果：**
- ✅ 支持并发处理多个请求
- ✅ 提高吞吐量
- ✅ 避免阻塞
- ✅ 可配置的工作协程数量

### 3. 错误处理机制

**优化前：**
```go
if err != nil {
    logging.Logger.Printf("处理MCP请求失败: %v", err)
    continue
}
```

**优化后：**
```go
// 发送错误响应
s.sendErrorResponse(writer, "", -32700, fmt.Sprintf("读取输入失败: %v", err))

// 专门的错误响应函数
func (s *Server) sendErrorResponse(writer *bufio.Writer, id string, code int, message string) {
    errResp := mcp.NewErrorResponse(id, code, message)
    response, err := json.Marshal(errResp)
    if err != nil {
        logging.Logger.Printf("序列化错误响应失败: %v", err)
        return
    }
    
    if err := s.writeResponse(writer, response); err != nil {
        logging.Logger.Printf("发送错误响应失败: %v", err)
    }
}
```

**改进效果：**
- ✅ 统一的错误响应格式
- ✅ 更好的错误信息传递
- ✅ 防止错误信息丢失
- ✅ 客户端友好的错误处理

### 4. 性能监控

**新增功能：**
```go
// 记录请求开始时间
startTime := time.Now()

// 记录请求信息
logging.Logger.Printf("收到MCP请求: ID=%s, Method=%s", request.ID, request.Method)

// 记录工具调用信息
logging.Logger.Printf("工具调用: %s, 参数: %+v", toolParams.Name, toolParams.Parameters)

// 记录处理时间
duration := time.Since(startTime)
logging.Logger.Printf("MCP请求处理完成: ID=%s, 耗时=%v", request.ID, duration)
```

**改进效果：**
- ✅ 详细的性能监控
- ✅ 请求追踪能力
- ✅ 性能瓶颈识别
- ✅ 调试信息丰富

### 5. 超时控制

**新增功能：**
```go
// 设置请求超时
ctx, cancel := context.WithTimeout(s.ctx, s.config.Global.Timeout)
defer cancel()

// 创建带超时的处理通道
done := make(chan struct{})
var response []byte
var err error

go func() {
    response, err = s.handleMCPRequest(task.data)
    close(done)
}()

// 等待处理完成或超时
select {
case <-ctx.Done():
    logging.Logger.Printf("请求处理超时")
    s.sendErrorResponse(task.writer, "", -32603, "请求处理超时")
case <-done:
    // 处理完成
}
```

**改进效果：**
- ✅ 防止请求无限等待
- ✅ 可配置的超时时间
- ✅ 优雅的超时处理
- ✅ 资源保护

### 6. 日志记录优化

**优化前：**
```go
logging.Logger.Printf("处理MCP请求失败: %v", err)
```

**优化后：**
```go
logging.Logger.Printf("解析MCP请求失败: %v, 数据: %s", err, string(data))
logging.Logger.Printf("收到MCP请求: ID=%s, Method=%s", request.ID, request.Method)
logging.Logger.Printf("工具调用: %s, 参数: %+v", toolParams.Name, toolParams.Parameters)
logging.Logger.Printf("处理工具调用失败: %v", err)
logging.Logger.Printf("MCP请求处理完成: ID=%s, 耗时=%v", request.ID, duration)
```

**改进效果：**
- ✅ 更详细的日志信息
- ✅ 请求追踪能力
- ✅ 性能监控
- ✅ 调试友好

## 性能测试

### 测试脚本

我们提供了性能测试脚本 `scripts/performance_test.sh`，可以测试：

1. **延迟测试** - 测量请求响应时间
2. **压力测试** - 测试不同并发数下的性能
3. **成功率统计** - 监控请求成功率

### 使用方法

```bash
# 运行性能测试
./scripts/performance_test.sh
```

### 测试结果示例

```
延迟测试结果:
  总请求数: 100
  成功数: 95
  最小延迟: 15ms
  最大延迟: 150ms
  平均延迟: 45ms

并发测试结果:
  总请求数: 100
  成功数: 92
  失败数: 8
  成功率: 92%
  总耗时: 4500ms
  平均耗时: 45ms
  并发数: 10
```

## 配置选项

### 工作协程数量

可以通过修改代码中的 `workerCount` 变量来调整并发处理能力：

```go
workerCount := 4 // 可以根据需要调整
```

### 缓冲区大小

可以调整缓冲区大小以适应不同的使用场景：

```go
reader := bufio.NewReaderSize(os.Stdin, 64*1024) // 64KB 缓冲区
writer := bufio.NewWriterSize(os.Stdout, 64*1024) // 64KB 缓冲区
```

### 请求通道大小

可以调整请求通道的缓冲区大小：

```go
requestChan := make(chan *requestTask, 100) // 缓冲通道
```

## 兼容性

### 向后兼容

所有优化都保持了向后兼容性：

- ✅ 支持现有的 MCP 协议格式
- ✅ 支持现有的配置文件格式
- ✅ 支持现有的命令行参数
- ✅ 支持现有的错误处理机制

### 新功能

新增的功能都是可选的，不会影响现有功能：

- ✅ 并发处理（可选）
- ✅ 性能监控（可选）
- ✅ 超时控制（可选）
- ✅ 详细日志（可选）

## 使用建议

### 1. 生产环境配置

```go
// 建议的生产环境配置
workerCount := runtime.NumCPU() // 使用 CPU 核心数
reader := bufio.NewReaderSize(os.Stdin, 128*1024) // 128KB 缓冲区
writer := bufio.NewWriterSize(os.Stdout, 128*1024) // 128KB 缓冲区
requestChan := make(chan *requestTask, 1000) // 更大的缓冲通道
```

### 2. 开发环境配置

```go
// 建议的开发环境配置
workerCount := 2 // 较少的协程数
reader := bufio.NewReaderSize(os.Stdin, 32*1024) // 较小的缓冲区
writer := bufio.NewWriterSize(os.Stdout, 32*1024) // 较小的缓冲区
requestChan := make(chan *requestTask, 50) // 较小的缓冲通道
```

### 3. 监控建议

- 定期运行性能测试脚本
- 监控日志中的性能指标
- 根据实际负载调整配置参数
- 设置合理的超时时间

## 总结

通过这次优化，MCP2REST 服务器的 `startStdioServer` 函数在以下方面得到了显著改进：

1. **性能提升** - 支持并发处理，提高吞吐量
2. **稳定性增强** - 更好的错误处理和超时控制
3. **可观测性** - 详细的日志和性能监控
4. **可维护性** - 更清晰的代码结构和错误处理
5. **可扩展性** - 可配置的参数和模块化设计

这些优化使得服务器更适合生产环境使用，同时保持了良好的开发体验。
