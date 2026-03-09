# MCP (Model Context Protocol) Support

[中文](MCP_CN.md) | English

---

## Overview

redc now supports the Model Context Protocol (MCP), which allows AI assistants and automation tools to interact with redc's infrastructure management capabilities programmatically.

MCP support enables:
- **AI-driven infrastructure management**: Let AI assistants help you deploy and manage red team infrastructure
- **Automated workflows**: Integrate redc with AI tools and automation platforms
- **Programmatic access**: Control redc through a standardized JSON-RPC protocol
- **Multi-modal integration**: Works with various AI clients via STDIO or SSE transports

## Features

The redc MCP server exposes the following capabilities:

### Tools (Actions)

1. **list_templates** - List all available redc templates/images
2. **search_templates** - Search for templates in the official registry by keywords (provider, name, description, etc.)
3. **pull_template** - Download a template from the registry (redc pull)
4. **list_cases** - List all running cases in the current project
5. **plan_case** - Plan a new case from a template (like terraform plan - preview resources without creating them)
6. **start_case** - Start a case by ID (this actually creates and starts the infrastructure)
7. **stop_case** - Stop a running case by ID
8. **kill_case** - Kill (destroy) a case by ID
9. **get_case_status** - Get the status of a specific case
10. **exec_command** - Execute a command on a case
11. **get_ssh_info** - Get SSH connection information for a case (IP, port, username, password, key path)
12. **upload_file** - Upload a local file to a case server via SFTP
13. **download_file** - Download a file from a case server to local machine via SFTP
14. **get_template_info** - Get detailed information about a locally installed template (metadata, variables)
15. **delete_template** - Delete a locally installed template
16. **get_case_outputs** - Get terraform outputs for a case (IP addresses, instance IDs, etc.)
17. **get_config** - Get redc current configuration (project path, proxy settings)
18. **validate_config** - Validate cloud provider configuration (credentials, region, instance type)

#### Extended Tools (GUI mode only)

The following tools are available when using MCP through redc-gui (they require AppBridge):

**Compose Orchestration:**
19. **compose_preview** - Preview compose orchestration, returns service list and dependency graph
20. **compose_up** - Start compose orchestration deployment (async)
21. **compose_down** - Destroy compose orchestration deployment (async)

**Cost & Resources:**
22. **get_cost_estimate** - Estimate deployment cost for a template
23. **get_balances** - Query cloud account balances
24. **get_resource_summary** - Get resource summary across all cloud providers
25. **get_predicted_monthly_cost** - Get predicted monthly cost
26. **get_bills** - Get cloud billing information
27. **get_total_runtime** - Get total runtime of all active cases

**Custom Deployment:**
28. **list_deployments** - List all custom deployments
29. **start_deployment** - Start a custom deployment by ID
30. **stop_deployment** - Stop a custom deployment by ID

**Project & Profile Management:**
31. **list_projects** - List all projects
32. **switch_project** - Switch to a different project
33. **list_profiles** - List all configuration profiles
34. **get_active_profile** - Get the currently active profile
35. **set_active_profile** - Set active profile by ID

**Scheduler:**
36. **schedule_task** - Schedule a future task (start/stop/kill) for a case
37. **list_scheduled_tasks** - List all pending scheduled tasks

### Resources

1. **redc://templates** - JSON list of available templates
2. **redc://cases** - JSON list of all cases in the current project
3. **redc://config** - Current redc configuration

## Transport Modes

redc MCP server supports two transport modes:

### 1. STDIO Transport

STDIO mode is ideal for local integration with AI assistants and tools. The server reads JSON-RPC requests from stdin and writes responses to stdout.

**Usage:**
```bash
redc mcp stdio
```

This mode is perfect for:
- Claude Desktop integration
- Local AI assistant tools
- Development and testing
- Pipeline automation

### 2. SSE (Server-Sent Events) Transport

SSE mode runs an HTTP server that can handle multiple clients and provides a web-accessible endpoint.

**Usage:**
```bash
# Default address (localhost:8080)
redc mcp sse

# Custom address
redc mcp sse localhost:9000

# Listen on all interfaces
redc mcp sse 0.0.0.0:8080

# Short form (port only)
redc mcp sse :8080
```

The SSE server exposes three endpoints:
- `GET /` - Server information
- `POST /message` - Send JSON-RPC messages (recommended)
- `GET /sse` - SSE streaming endpoint

This mode is perfect for:
- Web-based AI clients
- Remote access
- Multi-user environments
- Production deployments

## Usage Examples

### Initialize MCP Protocol

When connecting to the MCP server, the client must first send an `initialize` request:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "initialize",
  "params": {}
}
```

Response:
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

### List Available Tools


### Search Templates

```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/call",
  "params": {
    "name": "search_templates",
    "arguments": {
      "query": "aliyun ecs",
      "registry_url": "https://redc.wgpsec.org"
    }
  }
}
```

### Pull a Template

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

### Create a Case

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

**Note:** `plan_case` performs a terraform plan to validate the configuration without actually creating infrastructure. Use `start_case` afterwards to actually create and start the resources.

### Start a Case

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

### Execute a Command

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

### Get SSH Info

```json
{
  "jsonrpc": "2.0",
  "id": 6,
  "method": "tools/call",
  "params": {
    "name": "get_ssh_info",
    "arguments": {
      "case_id": "8a57078ee856"
    }
  }
}
```

### Upload File

```json
{
  "jsonrpc": "2.0",
  "id": 7,
  "method": "tools/call",
  "params": {
    "name": "upload_file",
    "arguments": {
      "case_id": "8a57078ee856",
      "local_path": "/local/path/to/file.txt",
      "remote_path": "/remote/path/file.txt"
    }
  }
}
```

### Download File

```json
{
  "jsonrpc": "2.0",
  "id": 8,
  "method": "tools/call",
  "params": {
    "name": "download_file",
    "arguments": {
      "case_id": "8a57078ee856",
      "remote_path": "/remote/path/file.txt",
      "local_path": "/local/path/file.txt"
    }
  }
}
```

### Get Template Info

```json
{
  "jsonrpc": "2.0",
  "id": 9,
  "method": "tools/call",
  "params": {
    "name": "get_template_info",
    "arguments": {
      "template_name": "aliyun/ecs"
    }
  }
}
```

### Delete Template

```json
{
  "jsonrpc": "2.0",
  "id": 10,
  "method": "tools/call",
  "params": {
    "name": "delete_template",
    "arguments": {
      "template_name": "aliyun/ecs"
    }
  }
}
```

### Get Case Outputs

```json
{
  "jsonrpc": "2.0",
  "id": 11,
  "method": "tools/call",
  "params": {
    "name": "get_case_outputs",
    "arguments": {
      "case_id": "8a57078ee856"
    }
  }
}
```

### Get Config

```json
{
  "jsonrpc": "2.0",
  "id": 12,
  "method": "tools/call",
  "params": {
    "name": "get_config",
    "arguments": {}
  }
}
```

### Validate Config

```json
{
  "jsonrpc": "2.0",
  "id": 13,
  "method": "tools/call",
  "params": {
    "name": "validate_config",
    "arguments": {
      "provider": "aliyun",
      "region": "cn-hangzhou",
      "instance_type": "ecs.t6-c1m1.large"
    }
  }
}
```

### Compose Preview (GUI mode)

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

### Get Balances (GUI mode)

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

### Schedule a Task (GUI mode)

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

### Read Resources

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

## Integration with AI Assistants

### Claude Desktop

Add to your Claude Desktop configuration (`~/Library/Application Support/Claude/claude_desktop_config.json` on macOS):

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

### Using curl with SSE mode

```bash
# Start the SSE server
redc mcp sse localhost:8080

# In another terminal, send requests
curl -X POST http://localhost:8080/message \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/list"
  }'
```

## Configuration

The MCP server uses the same configuration as the redc CLI:

- **Project**: Use `--project` flag to specify the project (default: "default")
- **User**: Use `--user` flag to specify the operator (default: "system")
- **Config file**: Use `--config` flag to specify a custom config file
- **Debug mode**: Use `--debug` flag to enable debug logging

Example:
```bash
redc mcp sse --project myproject --user alice --debug
```

## Security Considerations

### STDIO Mode
- Runs locally with the same permissions as the user running redc
- No network exposure
- Safe for local development

### SSE Mode
- Exposes HTTP endpoints on the network
- **WARNING**: No authentication by default
- Recommended to:
  - Bind to localhost only (default)
  - Use firewall rules to restrict access
  - Deploy behind a reverse proxy with authentication
  - Use VPN or SSH tunnels for remote access

## Troubleshooting

### Server won't start
- Check if the port is already in use
- Verify redc configuration is valid
- Ensure templates directory exists

### Commands fail
- Verify the case ID is correct
- Check case status before executing commands
- Ensure SSH connectivity to the case
- Check redc logs for detailed error messages

### Connection issues
- For STDIO: Check that JSON-RPC messages are properly formatted
- For SSE: Verify the server is running and accessible
- Use `--debug` flag for verbose logging

## Protocol Version

redc implements MCP protocol version **2024-11-05**.

## Additional Resources

- [MCP Specification](https://modelcontextprotocol.io/)
- [redc Documentation](README.md)
- [Template Repository](https://github.com/wgpsec/redc-template)

## Support

For issues or questions:
- GitHub Issues: https://github.com/wgpsec/redc/issues
- Discussions: https://github.com/wgpsec/redc/discussions
