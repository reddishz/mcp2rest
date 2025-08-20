# MCP2REST

MCP2REST is a general-purpose gateway server for converting MCP (Model Control Protocol) tool calls to REST API requests. It allows AI models to interact with various REST APIs through simple configuration files, without the need to write specialized code for each API.

## Features

- Supports both WebSocket and standard input/output communication modes
- Defines API endpoints through YAML configuration files
- Supports multiple authentication methods (Bearer token, API key, Basic authentication, OAuth2)
- Flexible parameter handling (path, query, request body, headers)
- Powerful response transformation capabilities (direct, JQ expressions, templates)
- Detailed error handling and logging

## Installation

```bash
go get github.com/mcp2rest
```

## Usage

1. Create a configuration file (see `examples/configs/example_config.yaml`)
2. Start the server:

```bash
mcp2rest --config path/to/config.yaml
```

## Configuration File Format

The configuration file uses YAML format and contains the following main sections:

- `server`: Server configuration (port, host, mode)
- `global`: Global settings (timeout, maximum request size, default headers)
- `endpoints`: List of API endpoint definitions

Each endpoint definition includes:

- `name`: Endpoint name (used for MCP tool calls)
- `description`: Endpoint description
- `method`: HTTP method (GET, POST, PUT, DELETE, etc.)
- `url_template`: URL template, supports parameter substitution
- `authentication`: Authentication configuration
- `parameters`: List of parameter definitions
- `response`: Response handling configuration

## MCP Request Format

MCP2REST accepts JSON-RPC requests in the following format:

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

## Examples

### Configuration File Example

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

### MCP Request Example

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

## License

MIT