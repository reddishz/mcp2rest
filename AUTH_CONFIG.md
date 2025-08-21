# MCP2REST 认证配置指南

本文档介绍如何在 MCP2REST 中配置 API 认证。

## 认证类型支持

MCP2REST 支持以下认证类型：

1. **API Key 认证** - 在请求头中传递 API 密钥
2. **Bearer Token 认证** - 在 Authorization 头中传递 Bearer 令牌
3. **Basic 认证** - 使用用户名和密码的基本认证

## 配置方法

### 方法 1: 环境变量配置（推荐）

#### BMC API 配置

对于 `configs/bmc_api.yaml` 中定义的 API，需要设置环境变量：

```bash
# 设置 BMC API 的 API Key
export BMC_API_KEY="your_bmc_api_key_here"
```

#### 其他 API 配置示例

```bash
# Bearer Token 认证
export WEATHER_API_TOKEN="your_weather_api_token_here"

# Basic 认证密码
export USER_API_PASSWORD="your_user_api_password_here"
```

### 方法 2: 认证配置文件

创建 `configs/auth_config.yaml` 文件：

```yaml
# BMC API 认证配置
bmc_api:
  type: "api_key"
  header_name: "X-API-Key"
  key_env: "BMC_API_KEY"
  description: "BMC 数据管理 API 的认证密钥"

# 天气 API 认证配置
weather_api:
  type: "bearer"
  token_env: "WEATHER_API_TOKEN"
  description: "天气 API 的 Bearer Token"

# 用户 API 认证配置
user_api:
  type: "basic"
  username: "admin"
  key_env: "USER_API_PASSWORD"
  description: "用户 API 的基本认证"
```

### 方法 3: 使用认证配置工具

#### 编译认证配置工具

```bash
go build -o bin/auth_config cmd/auth_config/main.go
```

#### 查看当前认证配置

```bash
./bin/auth_config -action list
```

#### 验证认证配置

```bash
./bin/auth_config -action validate -api bmc_api
```

#### 设置认证配置

```bash
# 设置 API Key 认证
./bin/auth_config -action set -api bmc_api -type api_key -header "X-API-Key" -key-env "BMC_API_KEY"

# 设置 Bearer Token 认证
./bin/auth_config -action set -api weather_api -type bearer -token-env "WEATHER_API_TOKEN"

# 设置 Basic 认证
./bin/auth_config -action set -api user_api -type basic -username "admin" -key-env "USER_API_PASSWORD"
```

## 认证配置详解

### API Key 认证

```yaml
bmc_api:
  type: "api_key"
  header_name: "X-API-Key"  # 请求头名称
  key_env: "BMC_API_KEY"    # 环境变量名
```

**环境变量设置：**
```bash
export BMC_API_KEY="your_api_key_here"
```

**请求示例：**
```http
GET /api/endpoint
X-API-Key: your_api_key_here
```

### Bearer Token 认证

```yaml
weather_api:
  type: "bearer"
  token_env: "WEATHER_API_TOKEN"  # 环境变量名
```

**环境变量设置：**
```bash
export WEATHER_API_TOKEN="your_bearer_token_here"
```

**请求示例：**
```http
GET /api/endpoint
Authorization: Bearer your_bearer_token_here
```

### Basic 认证

```yaml
user_api:
  type: "basic"
  username: "admin"              # 用户名
  key_env: "USER_API_PASSWORD"   # 密码环境变量名
```

**环境变量设置：**
```bash
export USER_API_PASSWORD="your_password_here"
```

**请求示例：**
```http
GET /api/endpoint
Authorization: Basic YWRtaW46eW91cl9wYXNzd29yZF9oZXJl
```

## 配置文件位置

认证配置文件会按以下顺序查找：

1. 指定的配置文件路径
2. `configs/auth_config.yaml`
3. `../configs/auth_config.yaml`
4. 可执行文件目录下的 `configs/auth_config.yaml`
5. 可执行文件上级目录下的 `configs/auth_config.yaml`

## 环境变量文件

可以创建 `.env` 文件来管理环境变量：

```bash
# 复制示例文件
cp configs/env.example .env

# 编辑 .env 文件
nano .env
```

`.env` 文件内容示例：
```bash
# BMC API 认证
BMC_API_KEY=your_bmc_api_key_here

# 其他 API 认证
WEATHER_API_TOKEN=your_weather_api_token_here
USER_API_PASSWORD=your_user_api_password_here

# 服务器配置
MCP2REST_DEBUG=false
MCP2REST_LOG_LEVEL=info
```

## 验证配置

### 1. 使用认证配置工具验证

```bash
# 验证 BMC API 配置
./bin/auth_config -action validate -api bmc_api

# 验证所有配置
./bin/auth_config -action list
```

### 2. 使用测试脚本验证

```bash
# 运行测试脚本
./scripts/simple_test.sh
```

### 3. 手动测试

```bash
# 设置环境变量
export BMC_API_KEY="your_api_key_here"

# 测试请求
echo '{"jsonrpc":"2.0","id":"test_001","method":"toolCall","params":{"name":"list","parameters":{"page":1,"limit":5}}}' | ./bin/mcp2rest -config ./configs/bmc_api.yaml
```

## 故障排除

### 1. 环境变量未设置

**错误信息：**
```
环境变量 BMC_API_KEY 未设置或为空
```

**解决方案：**
```bash
export BMC_API_KEY="your_api_key_here"
```

### 2. 认证配置错误

**错误信息：**
```
API Key 认证需要指定 header_name
```

**解决方案：**
检查认证配置文件中的 `header_name` 字段是否正确设置。

### 3. 认证失败

**错误信息：**
```
API返回错误状态码: 403
```

**解决方案：**
- 检查 API Key 是否正确
- 检查 API Key 是否有效
- 检查 API Key 是否有足够的权限

### 4. 配置文件未找到

**错误信息：**
```
读取认证配置文件失败
```

**解决方案：**
- 检查配置文件路径是否正确
- 确保配置文件存在
- 检查文件权限

## 安全建议

1. **不要在代码中硬编码认证信息**
2. **使用环境变量存储敏感信息**
3. **定期轮换 API 密钥**
4. **使用最小权限原则**
5. **监控 API 调用日志**
6. **使用 HTTPS 传输**

## 示例配置

### 完整的认证配置示例

```yaml
# configs/auth_config.yaml
bmc_api:
  type: "api_key"
  header_name: "X-API-Key"
  key_env: "BMC_API_KEY"
  description: "BMC 数据管理 API 的认证密钥"

weather_api:
  type: "bearer"
  token_env: "WEATHER_API_TOKEN"
  description: "天气 API 的 Bearer Token"

user_api:
  type: "basic"
  username: "admin"
  key_env: "USER_API_PASSWORD"
  description: "用户 API 的基本认证"

# 全局配置
global:
  timeout: "30s"
  retry:
    max_attempts: 3
    backoff_delay: "1s"
  cache:
    enabled: true
    ttl: "5m"
```

### 环境变量配置示例

```bash
# .env 文件
BMC_API_KEY=sk-1234567890abcdef
WEATHER_API_TOKEN=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
USER_API_PASSWORD=secure_password_123
MCP2REST_DEBUG=false
MCP2REST_LOG_LEVEL=info
```

通过这些配置，你可以安全地管理各种 API 的认证信息，确保 MCP2REST 能够正确访问需要认证的 REST API。
