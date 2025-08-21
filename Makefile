# MCP2REST Makefile

.PHONY: all build build-stdio build-sse build-original clean test-stdio test-sse help

# 默认目标
all: build

# 编译所有版本
build: build-stdio build-sse build-original

# 编译 stdio 版本
build-stdio:
	@echo "编译 MCP2REST-STDIO..."
	go build -o bin/mcp2rest-stdio cmd/mcp2rest-stdio/main.go
	@echo "MCP2REST-STDIO 编译完成"

# 编译 SSE 版本
build-sse:
	@echo "编译 MCP2REST-SSE..."
	go build -o bin/mcp2rest-sse cmd/mcp2rest-sse/main.go
	@echo "MCP2REST-SSE 编译完成"

# 编译原始版本
build-original:
	@echo "编译 MCP2REST（原始版本）..."
	go build -o bin/mcp2rest cmd/mcp2rest/main.go
	@echo "MCP2REST（原始版本）编译完成"

# 清理编译文件
clean:
	@echo "清理编译文件..."
	rm -f bin/mcp2rest*
	@echo "清理完成"

# 测试 stdio 版本
test-stdio: build-stdio
	@echo "测试 MCP2REST-STDIO..."
	@export APIKEYAUTH_API_KEY="ded45a001ffb9c47b1e29fcbdd6bcec6" && \
	echo '{"jsonrpc":"2.0","id":"test","method":"initialize","params":{"protocolVersion":"20241105","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}' | \
	./bin/mcp2rest-stdio -config ./configs/bmc_api.yaml

# 测试 SSE 版本
test-sse: build-sse
	@echo "测试 MCP2REST-SSE..."
	@export APIKEYAUTH_API_KEY="ded45a001ffb9c47b1e29fcbdd6bcec6" && \
	./bin/mcp2rest-sse -config ./configs/bmc_api.yaml &
	@sleep 3 && \
	curl -X POST http://localhost:8088/ \
		-H "Content-Type: application/json" \
		-d '{"jsonrpc":"2.0","id":"test","method":"initialize","params":{"protocolVersion":"20241105","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}' && \
	echo "" && \
	pkill -f mcp2rest-sse

# 显示帮助信息
help:
	@echo "MCP2REST Makefile 使用说明："
	@echo ""
	@echo "可用目标："
	@echo "  all          - 编译所有版本（默认）"
	@echo "  build        - 编译所有版本"
	@echo "  build-stdio  - 编译 stdio 版本"
	@echo "  build-sse    - 编译 SSE 版本"
	@echo "  build-original - 编译原始版本"
	@echo "  clean        - 清理编译文件"
	@echo "  test-stdio   - 测试 stdio 版本"
	@echo "  test-sse     - 测试 SSE 版本"
	@echo "  help         - 显示此帮助信息"
	@echo ""
	@echo "示例："
	@echo "  make build-stdio    # 只编译 stdio 版本"
	@echo "  make test-stdio     # 编译并测试 stdio 版本"
	@echo "  make clean          # 清理所有编译文件"