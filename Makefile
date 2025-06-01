.PHONY: build test clean help

# 默认目标
all: build

# 构建项目
build:
	@echo "Building oas-mcp..."
	go build -o oas-mcp ./cmd/oas-mcp

# 运行测试
test: build
	@echo "Running tests..."
	go test ./...

# 清理构建文件
clean:
	@echo "Cleaning up..."
	rm -f oas-mcp oas-mcp-new oas-mcp~

# 安装依赖
deps:
	@echo "Installing dependencies..."
	go mod tidy

# 运行示例（STDIO模式）
run-stdio: build
	@echo "Running in STDIO mode..."
	./oas-mcp --swagger-file=swagger.json --upstream-base-url=https://jsonplaceholder.typicode.com

# 运行示例（HTTP模式）
run-http: build
	@echo "Running in HTTP mode on port 8080..."
	./oas-mcp --swagger-file=swagger.json --upstream-base-url=https://jsonplaceholder.typicode.com --mode=http --port=8080

# 显示版本信息
version: build
	./oas-mcp --version

# 显示帮助信息
help:
	@echo "Available targets:"
	@echo "  build      - Build the project"
	@echo "  test       - Run tests"
	@echo "  clean      - Clean build files"
	@echo "  deps       - Install dependencies"
	@echo "  run-stdio  - Run in STDIO mode"
	@echo "  run-http   - Run in HTTP mode"
	@echo "  version    - Show version info"
	@echo "  help       - Show this help" 