# OAS-MCP 测试指南 - Clinx.work

本指南展示如何使用 OAS-MCP 项目连接和测试真实的 API 服务 `https://newapi.clinx.work`。

## 🎯 测试概览

我们已经成功地：
1. ✅ 下载了真实的 OpenAPI/Swagger 文档
2. ✅ 解析并生成了 9 个 MCP 工具
3. ✅ 启动了 MCP 服务器
4. ✅ 测试了 MCP 协议通信
5. ✅ 成功调用了真实的 API 端点

## 📋 前提条件

1. **获取 API Token**：
   - 访问 https://dev.clinx.work
   - 注册账户并获取 API Token (格式: `sk-xxxx`)
   - 新用户有免费试用额度

2. **系统要求**：
   - Go 1.22+
   - curl 命令行工具

## 🚀 快速测试

### 方法一：使用测试脚本（推荐）

```bash
# 使用您的 API Token 运行测试
./test-clinx.sh sk-your-api-token-here
```

### 方法二：手动测试步骤

1. **构建项目**：
   ```bash
   go build -o oas-mcp ./cmd/oas-mcp
   ```

2. **启动服务器**：
   ```bash
   # 使用您的真实 API Token 替换 YOUR_TOKEN
   ./oas-mcp --swagger-file=swagger-clinx.json \
             --upstream-base-url=https://newapi.clinx.work \
             --mode=http \
             --port=8085 \
             --auth-type=bearer \
             --auth-token=YOUR_TOKEN &
   ```

3. **测试 MCP 协议**：

   **初始化连接**：
   ```bash
   curl -X POST http://localhost:8085 \
     -H "Content-Type: application/json" \
     -d '{
       "jsonrpc": "2.0",
       "id": 1,
       "method": "initialize",
       "params": {
         "protocolVersion": "2024-11-05",
         "capabilities": {"tools": {}},
         "clientInfo": {"name": "test-client", "version": "1.0.0"}
       }
     }'
   ```

   **获取工具列表**：
   ```bash
   curl -X POST http://localhost:8085 \
     -H "Content-Type: application/json" \
     -d '{
       "jsonrpc": "2.0",
       "id": 2,
       "method": "tools/list",
       "params": {}
     }'
   ```

   **调用模型列表接口**：
   ```bash
   curl -X POST http://localhost:8085 \
     -H "Content-Type: application/json" \
     -d '{
       "jsonrpc": "2.0",
       "id": 3,
       "method": "tools/call",
       "params": {
         "name": "get_providers_modelsList",
         "arguments": {"tag": "llm"}
       }
     }'
   ```

## 📊 测试结果

### 成功生成的 MCP 工具 (9个)

1. **get_api_checkToken** - 检查认证状态
2. **get_api_mj_image_{id}** - 获取 Midjourney 图像
3. **post_api_mj_submit_imagine** - Midjourney 图像生成
4. **post_api_user_login** - 用户登录
5. **post_api_v1_chat_completions** - OpenAI 兼容的聊天完成
6. **post_api_v1_images_generations** - OpenAI 兼容的图像生成
7. **get_providers_modelsList** - 获取可用模型列表 ⭐
8. **get_api_oauth_nd99u** - 99U OAuth 登录
9. **post_api_user_pay** - 创建支付订单

### 验证的可用模型

测试获取到的真实模型列表包括：
- **gpt-4.1** (priority: 8)
- **claude-3-7-sonnet-20250219** (priority: 7)
- **gpt-4o-2024-08-06** (priority: 6)
- **doubao-1-5-pro-256k-250115** (priority: 5)
- **deepseek-v3-0324** (priority: 4)
- **kimi-latest** (priority: 3)
- **qwen3-235b-a22b** (priority: 2)
- **gemini-2.0-flash-001** (priority: 1)

## 🔧 环境变量配置

您也可以使用环境变量设置配置：

```bash
# 设置 API Token
export OAS_MCP_AUTH_TOKEN=sk-your-token-here

# 设置服务器配置
export OAS_MCP_MODE=http
export OAS_MCP_PORT=8085

# 设置上游 API
export OAS_MCP_UPSTREAM_BASE_URL=https://newapi.clinx.work

# 启动服务器
./oas-mcp --swagger-file=swagger-clinx.json
```

## 🔗 在 Claude Desktop 中使用

将以下配置添加到您的 Claude Desktop 配置中：

```json
{
  "mcpServers": {
    "clinx": {
      "command": "/path/to/oas-mcp",
      "args": [
        "--swagger-file=/path/to/swagger-clinx.json",
        "--upstream-base-url=https://newapi.clinx.work",
        "--auth-type=bearer",
        "--auth-token=sk-your-token-here"
      ]
    }
  }
}
```

## 🐛 故障排除

### 常见问题

1. **端口被占用**：
   ```bash
   # 更改端口号
   ./oas-mcp --port=8086 [其他参数...]
   ```

2. **API Token 无效**：
   - 确认 Token 格式正确 (sk-xxx)
   - 检查账户是否有余额
   - 访问 https://dev.clinx.work 验证 Token

3. **网络连接问题**：
   ```bash
   # 测试上游 API 连通性
   curl https://newapi.clinx.work/providers/modelsList
   ```

### 调试模式

启用详细日志：
```bash
./oas-mcp --log-level=debug [其他参数...]
```

## 📈 性能测试

我们的测试结果显示：
- ✅ OpenAPI 文档解析：正常
- ✅ MCP 工具生成：9个工具成功生成
- ✅ HTTP 服务器启动：正常
- ✅ API 代理功能：正常
- ✅ 认证处理：Bearer Token 支持正常
- ✅ 真实 API 调用：成功获取模型列表

## 🎉 总结

OAS-MCP 项目成功地将 `newapi.clinx.work` 的 OpenAPI 规范转换为功能完整的 MCP 服务器，使得 AI 助手（如 Claude）能够：

1. 自动发现并使用所有 API 端点
2. 正确处理认证和参数传递
3. 与真实的大模型服务进行交互
4. 支持图像生成、聊天完成等高级功能

这证明了 OAS-MCP 项目的实用性和可靠性！🚀 