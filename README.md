# MCP2REST

MCP2REST是一个通用网关服务器，用于将MCP（Model Control Protocol）工具调用转换为REST API请求。它允许AI模型通过简单的配置文件定义与各种REST API进行交互，无需为每个API编写专门的代码。

## 特性

- 支持WebSocket和标准输入/输出两种通信模式
- 通过YAML配置文件定义API端点
- **支持直接导入OpenAPI规范（Swagger）文件**
- 支持多种身份验证方法（Bearer令牌、API密钥、基本身份验证、OAuth2）
- 灵活的参数处理（路径、查询、请求体、头）
- 强大的响应转换功能（直接、JQ表达式、模板）
- 详细的错误处理和日志记录

## 安装

```bash
go get github.com/mcp2rest
```

## 使用方法

1. 创建API配置文件（参见`examples/configs/example_config.yaml`）
2. 可选：创建服务器配置文件`configs/server.yaml`（如果不存在，将使用默认配置）
3. 启动服务器：

```bash
mcp2rest --config path/to/api_config.yaml
```

### 默认服务器配置

如果`configs/server.yaml`不存在，将使用以下默认配置：

```yaml
server:
  port: 8080
  host: "0.0.0.0"
  mode: "websocket"

global:
  timeout: 30s
```

## 配置文件格式

MCP2REST 支持两种配置方式：单一配置文件和分离配置文件。

### 单一配置文件

单一配置文件使用YAML格式，包含以下主要部分：

- `server`: 服务器配置（端口、主机、模式）
- `global`: 全局设置（超时、最大请求大小、默认头）
- `endpoints`: API端点定义列表

### 分离配置文件

分离配置文件将服务器配置和API配置分开，更适合大型项目和多环境部署：

1. **主配置文件**：指定服务器配置和API配置文件的路径
   - `server_config`: 服务器配置文件路径
   - `api_configs`: API配置文件路径列表
   - `openapi_specs`: OpenAPI规范文件路径列表

2. **服务器配置文件**：包含服务器和全局设置
   - `server`: 服务器配置（端口、主机、模式）
   - `global`: 全局设置（超时、最大请求大小、默认头）

3. **API配置文件**：包含API端点定义
   - `endpoints`: API端点定义列表

4. **OpenAPI规范文件**：标准OpenAPI（Swagger）规范文件
   - 支持JSON和YAML格式
   - 自动转换为MCP2REST端点配置

### 端点定义

每个端点定义包括：

- `name`: 端点名称（用于MCP工具调用）
- `description`: 端点描述
- `method`: HTTP方法（GET、POST、PUT、DELETE等）
- `url_template`: URL模板，支持参数替换
- `authentication`: 身份验证配置
- `parameters`: 参数定义列表
- `response`: 响应处理配置

## MCP请求格式

MCP2REST接受以下格式的JSON-RPC请求：

```json
{
  "jsonrpc": "2.0",
  "id": "request-id",
  "method": "toolCall",
  "params": {
    "name": "endpointName",
    "parameters": {
      "param1": "value1",
      "param2": "value2"
    }
  }
}
```

## 示例

### 配置文件示例

```yaml
server:
  port: 8080
  host: "0.0.0.0"
  mode: "websocket"

endpoints:
  - name: "getWeather"
    method: "GET"
    url_template: "https://api.weatherapi.com/v1/current.json"
    authentication:
      type: "api_key"
      header_name: "X-API-Key"
      key_env: "WEATHER_API_KEY"
    parameters:
      - name: "q"
        required: true
        in: "query"
    response:
      success_code: 200
      transform:
        type: "jq"
        expression: "{ location: .location.name, temp_c: .current.temp_c }"
```

### 使用OpenAPI规范示例

主配置文件：

```yaml
server_config: "configs/server_config.yaml"
api_configs:
  - "configs/custom_api.yaml"
openapi_specs:
  - "specs/weather_api.json"
  - "specs/news_api.yaml"
```

OpenAPI规范文件（JSON格式）：

```json
{
  "openapi": "3.0.0",
  "info": {
    "title": "天气API",
    "version": "1.0.0"
  },
  "servers": [
    {
      "url": "https://api.weatherapi.com/v1"
    }
  ],
  "paths": {
    "/current.json": {
      "get": {
        "operationId": "getCurrentWeather",
        "parameters": [
          {
            "name": "q",
            "in": "query",
            "required": true,
            "schema": {
              "type": "string"
            }
          }
        ],
        "security": [
          {
            "ApiKeyAuth": []
          }
        ]
      }
    }
  },
  "components": {
    "securitySchemes": {
      "ApiKeyAuth": {
        "type": "apiKey",
        "in": "query",
        "name": "key"
      }
    }
  }
}
```

### MCP请求示例

```json
{
  "jsonrpc": "2.0",
  "id": "123",
  "method": "toolCall",
  "params": {
    "name": "getWeather",
    "parameters": {
      "q": "Beijing"
    }
  }
}
```

## 许可证

MIT