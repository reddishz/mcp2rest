.PHONY: build run clean split-config

# 构建所有二进制文件
build:
	@echo "构建 MCP2REST..."
	@go build -o bin/mcp2rest ./cmd/mcp2rest
	@go build -o bin/config-splitter ./cmd/config-splitter
	@echo "构建完成"

# 运行服务器
run:
	@echo "启动 MCP2REST 服务器..."
	@./bin/mcp2rest --config ./configs/main_config.yaml

# 清理构建产物
clean:
	@echo "清理构建产物..."
	@rm -rf bin/
	@echo "清理完成"

# 分离配置文件
split-config:
	@echo "分离配置文件..."
	@./bin/config-splitter --input ./examples/configs/example_config.yaml --output ./configs
	@echo "配置文件分离完成"

# 初始化项目
init: build split-config
	@echo "项目初始化完成"