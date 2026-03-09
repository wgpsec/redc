# MCP (模型上下文协议) 支持

中文 | [English](MCP.md)

---

## 概述

redc 现已支持模型上下文协议（MCP），允许 AI 助手和自动化工具通过编程方式与 redc 的基础设施管理功能进行交互。

MCP 支持实现了：
- **AI 驱动的基础设施管理**：让 AI 助手帮助您部署和管理红队基础设施
- **自动化工作流**：将 redc 与 AI 工具和自动化平台集成
- **编程访问**：通过标准化的 JSON-RPC 协议控制 redc
- **多模式集成**：通过 STDIO 或 SSE 传输与各种 AI 客户端协同工作

## 功能特性

redc MCP 服务器提供以下能力：

### 工具（操作）

1. **list_templates** - 列出所有可用的 redc 模板/镜像
2. **pull_template** - 从仓库下载模板（redc pull）
3. **list_cases** - 列出当前项目中的所有场景
4. **plan_case** - 从模板规划新场景（类似 terraform plan - 预览资源而不实际创建）
5. **start_case** - 通过 ID 启动场景（这将实际创建并启动基础设施）
6. **stop_case** - 通过 ID 停止运行中的场景
7. **kill_case** - 通过 ID 销毁场景
8. **get_case_status** - 获取特定场景的状态
9. **exec_command** - 在场景上执行命令
10. **get_ssh_info** - 获取场景的 SSH 连接信息
11. **upload_file** - 通过 SFTP 上传文件到场景服务器
12. **download_file** - 通过 SFTP 从场景服务器下载文件
13. **get_template_info** - 获取本地模板的详细信息
14. **delete_template** - 删除本地模板
15. **get_case_outputs** - 获取场景的 terraform 输出
16. **get_config** - 获取 redc 当前配置
17. **validate_config** - 验证云厂商配置
18. **search_templates** - 在官方仓库中搜索模板

#### 扩展工具（仅 GUI 模式）

以下工具在通过 redc-gui 使用 MCP 时可用（需要 AppBridge）：

**Compose 编排：**
19. **compose_preview** - 预览 compose 编排，返回服务列表和依赖关系
20. **compose_up** - 启动 compose 编排部署（异步）
21. **compose_down** - 销毁 compose 编排部署（异步）

**成本与资源：**
22. **get_cost_estimate** - 估算模板部署成本
23. **get_balances** - 查询云账户余额
24. **get_resource_summary** - 获取各云厂商资源汇总
25. **get_predicted_monthly_cost** - 获取当月预估成本
26. **get_bills** - 获取云账单信息
27. **get_total_runtime** - 获取所有活跃场景的总运行时长

**自定义部署：**
28. **list_deployments** - 列出所有自定义部署
29. **start_deployment** - 通过 ID 启动自定义部署
30. **stop_deployment** - 通过 ID 停止自定义部署

**项目与配置管理：**
31. **list_projects** - 列出所有项目
32. **switch_project** - 切换项目
33. **list_profiles** - 列出所有配置档案
34. **get_active_profile** - 获取当前激活的配置
35. **set_active_profile** - 设置激活的配置

**定时任务：**
36. **schedule_task** - 创建定时任务（start/stop/kill）
37. **list_scheduled_tasks** - 列出所有待执行的定时任务

### 资源

1. **redc://templates** - 可用模板的 JSON 列表
2. **redc://cases** - 当前项目中所有场景的 JSON 列表
3. **redc://config** - 当前 redc 配置

## 传输模式

redc MCP 服务器支持两种传输模式：

### 1. STDIO 传输

STDIO 模式非常适合与本地 AI 助手和工具集成。服务器从 stdin 读取 JSON-RPC 请求并将响应写入 stdout。

**使用方式：**
```bash
redc mcp stdio
```

此模式适用于：
- Claude Desktop 集成
- 本地 AI 助手工具
- 开发和测试
- 管道自动化

### 2. SSE（服务器发送事件）传输

SSE 模式运行一个 HTTP 服务器，可以处理多个客户端并提供可通过 Web 访问的端点。

**使用方式：**
```bash
# 默认地址（localhost:8080）
redc mcp sse

# 自定义地址
redc mcp sse localhost:9000

# 监听所有接口
redc mcp sse 0.0.0.0:8080

# 简写形式（仅端口）
redc mcp sse :8080
```

SSE 服务器提供三个端点：
- `GET /` - 服务器信息
- `POST /message` - 发送 JSON-RPC 消息（推荐）
- `GET /sse` - SSE 流式端点

此模式适用于：
- 基于 Web 的 AI 客户端
- 远程访问
- 多用户环境
- 生产部署

## 使用示例

### 初始化 MCP 协议

连接到 MCP 服务器时，客户端必须首先发送 `initialize` 请求：

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "initialize",
  "params": {}
}
```

响应：
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "protocolVersion": "2024-11-05",
    "capabilities": {
      "tools": {},
      "resources": {}
    },
    "serverInfo": {
      "name": "redc",
      "version": "1.x.x"
    }
  }
}
```

### 列出可用工具


### 拉取模板

```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "tools/call",
  "params": {
    "name": "pull_template",
    "arguments": {
      "template": "aliyun/ecs",
      "registry_url": "https://redc.wgpsec.org",
      "force": false
    }
  }
}
```
```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/list"
}
```

### 创建场景

```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "tools/call",
  "params": {
    "name": "plan_case",
    "arguments": {
      "template": "aliyun/ecs",
      "name": "my-test-case",
      "env": {
        "region": "cn-hangzhou"
      }
    }
  }
}
```

**注意：** `plan_case` 执行 terraform plan 来验证配置而不实际创建基础设施。之后使用 `start_case` 来实际创建和启动资源。

### 启动场景

```json
{
  "jsonrpc": "2.0",
  "id": 4,
  "method": "tools/call",
  "params": {
    "name": "start_case",
    "arguments": {
      "case_id": "8a57078ee856"
    }
  }
}
```

### 执行命令

```json
{
  "jsonrpc": "2.0",
  "id": 5,
  "method": "tools/call",
  "params": {
    "name": "exec_command",
    "arguments": {
      "case_id": "8a57078ee856",
      "command": "whoami"
    }
  }
}
```

### Compose 预览（GUI 模式）

```json
{
  "jsonrpc": "2.0",
  "id": 14,
  "method": "tools/call",
  "params": {
    "name": "compose_preview",
    "arguments": {
      "file": "compose.yaml",
      "profiles": "proxy,dns"
    }
  }
}
```

### 查询余额（GUI 模式）

```json
{
  "jsonrpc": "2.0",
  "id": 15,
  "method": "tools/call",
  "params": {
    "name": "get_balances",
    "arguments": {
      "providers": "aliyun,aws"
    }
  }
}
```

### 创建定时任务（GUI 模式）

```json
{
  "jsonrpc": "2.0",
  "id": 16,
  "method": "tools/call",
  "params": {
    "name": "schedule_task",
    "arguments": {
      "case_id": "8a57078ee856",
      "case_name": "my-test-case",
      "action": "stop",
      "scheduled_at": "2025-01-15T10:30:00Z"
    }
  }
}
```

### 读取资源

```json
{
  "jsonrpc": "2.0",
  "id": 6,
  "method": "resources/read",
  "params": {
    "uri": "redc://cases"
  }
}
```

## 与 AI 助手集成

### Claude Desktop

将以下内容添加到您的 Claude Desktop 配置文件（macOS 上位于 `~/Library/Application Support/Claude/claude_desktop_config.json`）：

```json
{
  "mcpServers": {
    "redc": {
      "command": "/path/to/redc",
      "args": ["mcp", "stdio"],
      "env": {
        "REDC_PROJECT": "default"
      }
    }
  }
}
```

### 使用 curl 配合 SSE 模式

```bash
# 启动 SSE 服务器
redc mcp sse localhost:8080

# 在另一个终端中发送请求
curl -X POST http://localhost:8080/message \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/list"
  }'
```

## 配置

MCP 服务器使用与 redc CLI 相同的配置：

- **项目**：使用 `--project` 标志指定项目（默认："default"）
- **用户**：使用 `--user` 标志指定操作者（默认："system"）
- **配置文件**：使用 `--config` 标志指定自定义配置文件
- **调试模式**：使用 `--debug` 标志启用调试日志

示例：
```bash
redc mcp sse --project myproject --user alice --debug
```

## 安全注意事项

### STDIO 模式
- 在本地运行，具有与运行 redc 的用户相同的权限
- 无网络暴露
- 适合本地开发

### SSE 模式
- 在网络上暴露 HTTP 端点
- **警告**：默认情况下没有身份验证
- 建议：
  - 仅绑定到 localhost（默认）
  - 使用防火墙规则限制访问
  - 部署在带有身份验证的反向代理后面
  - 使用 VPN 或 SSH 隧道进行远程访问

## 故障排除

### 服务器无法启动
- 检查端口是否已被占用
- 验证 redc 配置是否有效
- 确保模板目录存在

### 命令执行失败
- 验证场景 ID 是否正确
- 在执行命令前检查场景状态
- 确保可以通过 SSH 连接到场景
- 检查 redc 日志以获取详细错误信息

### 连接问题
- STDIO 模式：检查 JSON-RPC 消息格式是否正确
- SSE 模式：验证服务器是否正在运行且可访问
- 使用 `--debug` 标志获取详细日志

## 协议版本

redc 实现 MCP 协议版本 **2024-11-05**。

## 其他资源

- [MCP 规范](https://modelcontextprotocol.io/)
- [redc 文档](README_CN.md)
- [模板仓库](https://github.com/wgpsec/redc-template)

## 支持

如有问题或疑问：
- GitHub Issues：https://github.com/wgpsec/redc/issues
- Discussions：https://github.com/wgpsec/redc/discussions
