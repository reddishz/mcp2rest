# MCP2REST 测试指南

本文档介绍如何测试 MCP2REST 后端的 REST 功能是否正常工作。

## 测试程序概述

我们提供了三种测试方式：

1. **Go 测试客户端** (`cmd/test_client/main.go`) - 完整的测试套件
2. **Shell 测试脚本** (`scripts/test_mcp.sh`) - 功能测试脚本
3. **简单测试脚本** (`scripts/simple_test.sh`) - 快速验证脚本

## 前置条件

1. 确保项目已编译：
   ```bash
   go build -o bin/mcp2rest cmd/mcp2rest/main.go
   ```

2. 确保配置文件存在：
   ```bash
   ls configs/bmc_api.yaml
   ```

3. 确保服务器配置为 stdio 模式：
   ```yaml
   # configs/server.yaml
   server:
     mode: "stdio"  # 必须设置为 stdio 模式
   ```

## 测试方法

### 1. 使用 Go 测试客户端（推荐）

这是最完整的测试方式，提供了详细的测试报告和错误信息。

```bash
# 编译测试客户端
go build -o bin/test_client cmd/test_client/main.go

# 运行测试
./bin/test_client
```

**测试内容：**
- BMC 列表查询 (`list`)
- BMC 详情查询 (`detail`)
- BMC 搜索 (`search`)
- BMC 创建 (`create`)
- BMC 更新 (`update`)
- BMC 删除 (`delete`)

**输出示例：**
```
等待服务器启动...
开始运行 6 个测试用例...
============================================================
测试 1/6: 测试 BMC 列表查询
描述: 测试获取 BMC 数据列表功能
工具: list
参数: map[limit:10 order:desc page:1 sort:created]
✅ 成功 (耗时: 1.234s)
响应: map[result:map[data:map[list:[] pagination:map[limit:10 page:1 total:0]] message:success]]
----------------------------------------
...
============================================================
测试总结
============================================================
总测试数: 6
成功数: 6
失败数: 0
成功率: 100.00%
总耗时: 7.234s
平均耗时: 1.206s

🎉 所有测试都通过了！
```

### 2. 使用 Shell 测试脚本

这是一个功能性的测试脚本，适合自动化测试。

```bash
# 运行完整测试
./scripts/test_mcp.sh
```

### 3. 使用简单测试脚本

这是最快速的验证方式，适合开发时快速检查。

```bash
# 运行简单测试
./scripts/simple_test.sh
```

**输出示例：**
```
[INFO] 开始简单 MCP 测试...
[INFO] 检查依赖...
[SUCCESS] 依赖检查通过
[INFO] 测试 MCP 协议通信...
[SUCCESS] MCP 协议通信测试成功
响应: {"jsonrpc":"2.0","id":"test_001","result":{...}}
[INFO] 测试工具调用...
[SUCCESS] 工具调用测试成功
响应: {"jsonrpc":"2.0","id":"test_list_001","result":{...}}
[SUCCESS] 所有测试完成
```

## 手动测试

如果你想手动测试特定的 MCP 请求，可以使用以下方法：

### 1. 直接通过管道测试

```bash
# 测试列表查询
echo '{"jsonrpc":"2.0","id":"test_001","method":"toolCall","params":{"name":"list","parameters":{"page":1,"limit":5}}}' | ./bin/mcp2rest -config ./configs/bmc_api.yaml
```

### 2. 测试 BMC 创建

```bash
# 测试创建 BMC
echo '{"jsonrpc":"2.0","id":"test_002","method":"toolCall","params":{"name":"create","parameters":{"id":"test_001","title":"测试 BMC","description":"测试描述"}}}' | ./bin/mcp2rest -config ./configs/bmc_api.yaml
```

### 3. 测试 BMC 搜索

```bash
# 测试搜索 BMC
echo '{"jsonrpc":"2.0","id":"test_003","method":"toolCall","params":{"name":"search","parameters":{"q":"测试","page":1,"limit":10}}}' | ./bin/mcp2rest -config ./configs/bmc_api.yaml
```

## 故障排除

### 1. 服务器启动失败

**错误：** `服务器启动失败`

**解决方案：**
- 检查可执行文件是否存在：`ls -la bin/mcp2rest`
- 检查配置文件是否存在：`ls -la configs/bmc_api.yaml`
- 检查服务器配置：确保 `configs/server.yaml` 中的 `mode` 设置为 `"stdio"`

### 2. MCP 协议错误

**错误：** `MCP错误: &{Code:-32601 Message:不支持的方法}`

**解决方案：**
- 确保请求的 `method` 字段为 `"toolCall"`
- 确保 `params` 包含 `name` 和 `parameters` 字段

### 3. 工具调用失败

**错误：** `查找端点配置失败`

**解决方案：**
- 检查 OpenAPI 规范文件是否正确解析
- 确保工具名称与 OpenAPI 规范中的路径匹配
- 检查参数格式是否正确

### 4. 网络连接错误

**错误：** `发送HTTP请求失败`

**解决方案：**
- 检查目标 API 服务器是否可访问
- 检查网络连接
- 检查 API 密钥配置（如果需要）

## 测试环境配置

### 1. 设置 API 密钥（如果需要）

如果目标 API 需要认证，请设置环境变量：

```bash
export APIKEYAUTH_API_KEY="your_api_key_here"
```

### 2. 配置代理（如果需要）

```bash
export HTTP_PROXY="http://proxy.example.com:8080"
export HTTPS_PROXY="http://proxy.example.com:8080"
```

### 3. 调试模式

启用详细日志：

```bash
export MCP2REST_DEBUG=1
```

## 持续集成

可以将测试脚本集成到 CI/CD 流程中：

```yaml
# .github/workflows/test.yml
name: Test MCP2REST
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - uses: actions/setup-go@v2
        with:
          go-version: '1.19'
      - run: go build -o bin/mcp2rest cmd/mcp2rest/main.go
      - run: go build -o bin/test_client cmd/test_client/main.go
      - run: ./bin/test_client
```

## 性能测试

对于性能测试，可以修改测试客户端以支持并发测试：

```bash
# 运行并发测试（需要修改测试代码）
./bin/test_client -concurrent=10 -requests=100
```

## 总结

通过这些测试程序，你可以：

1. ✅ 验证 MCP 协议通信是否正常
2. ✅ 验证 REST API 调用是否成功
3. ✅ 验证错误处理是否正确
4. ✅ 验证响应格式是否符合预期
5. ✅ 获得详细的测试报告和性能指标

建议在开发过程中经常运行测试，确保功能正常。
