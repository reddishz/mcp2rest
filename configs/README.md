# MCP2REST 配置说明

## 配置文件结构

MCP2REST 使用简化的配置文件结构，包含以下文件：

```
configs/
├── server.yaml     # 可选的服务器配置（如不存在则使用默认值）
└── api_config.yaml # API 配置文件
```

## 配置文件说明

1. **server.yaml**：可选的服务器配置，包含端口、主机、模式等本地服务设置。如果此文件不存在，将使用默认配置。
2. **api_config.yaml**：API 端点定义，包含各种 API 的详细配置。

## 使用方法

启动服务器时，只需指定 API 配置文件路径：

```bash
mcp2rest --config ./configs/api_config.yaml
```

## 默认服务器配置

如果 `server.yaml` 不存在，将使用以下默认配置：

```yaml
server:
  port: 8080
  host: "0.0.0.0"
  mode: "websocket"

global:
  timeout: 30s
```