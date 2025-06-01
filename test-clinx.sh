#!/bin/bash

echo "=== OAS-MCP Clinx 测试脚本 ==="
echo ""

# 检查是否提供了API token
if [ -z "$1" ]; then
    echo "用法: $0 <API_TOKEN>"
    echo ""
    echo "示例: $0 sk-your-api-token-here"
    echo ""
    echo "如果您没有API token，请访问 https://dev.clinx.work 注册获取"
    exit 1
fi

API_TOKEN="$1"
echo "使用API Token: $API_TOKEN"
echo ""

# 构建项目
echo "1. 构建项目..."
go build -o oas-mcp ./cmd/oas-mcp
if [ $? -ne 0 ]; then
    echo "构建失败！"
    exit 1
fi
echo "构建成功 ✓"
echo ""

# 启动服务器（HTTP模式）
echo "2. 启动MCP服务器（HTTP模式）..."
echo "   服务器地址: http://localhost:8085"
echo "   上游API: https://newapi.clinx.work"
echo ""

# 使用环境变量设置API token
export OAS_MCP_AUTH_TOKEN="$API_TOKEN"

# 启动服务器
./oas-mcp --config=config-clinx.yaml --port=8085 &
SERVER_PID=$!

echo "服务器已启动，PID: $SERVER_PID"
echo ""

# 等待服务器启动
sleep 3

echo "3. 测试MCP服务器..."
echo ""

# 测试MCP初始化
echo "测试1: MCP初始化"
curl -s -X POST http://localhost:8085 \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "initialize",
    "params": {
      "protocolVersion": "2024-11-05",
      "capabilities": {
        "tools": {}
      },
      "clientInfo": {
        "name": "test-client",
        "version": "1.0.0"
      }
    }
  }' | jq '.' || echo "初始化测试失败"
echo ""

# 测试工具列表
echo "测试2: 获取可用工具列表"
curl -s -X POST http://localhost:8085 \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 2,
    "method": "tools/list",
    "params": {}
  }' | jq '.' || echo "工具列表测试失败"
echo ""

# 测试API调用（获取模型列表）
echo "测试3: 调用API接口（获取模型列表）"
curl -s -X POST http://localhost:8085 \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 3,
    "method": "tools/call",
    "params": {
      "name": "get_providers_modelslist",
      "arguments": {}
    }
  }' | jq '.' || echo "API调用测试失败"
echo ""

echo "4. 测试完成！"
echo ""
echo "如需停止服务器，请运行: kill $SERVER_PID"
echo ""
echo "手动测试命令："
echo "# 初始化MCP连接"
echo "curl -X POST http://localhost:8085 -H 'Content-Type: application/json' -d '{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"initialize\",\"params\":{\"protocolVersion\":\"2024-11-05\",\"capabilities\":{\"tools\":{}},\"clientInfo\":{\"name\":\"test\",\"version\":\"1.0.0\"}}}'"
echo ""
echo "# 获取工具列表"
echo "curl -X POST http://localhost:8085 -H 'Content-Type: application/json' -d '{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/list\",\"params\":{}}'"
echo ""
echo "# 调用具体工具"
echo "curl -X POST http://localhost:8085 -H 'Content-Type: application/json' -d '{\"jsonrpc\":\"2.0\",\"id\":3,\"method\":\"tools/call\",\"params\":{\"name\":\"get_providers_modelslist\",\"arguments\":{}}}'" 