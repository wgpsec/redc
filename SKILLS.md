# Redc Skills - AI Operations Integration Guide

## Overview

This document provides comprehensive specifications for AI agents and automation tools to interact with the `redc` (Red Cloud) command-line tool. Redc is a Red Team Infrastructure Multi-Cloud Automated Deployment Tool built on Terraform, designed to simplify the complete lifecycle of cloud infrastructure (create, configure, destroy).

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Configuration](#configuration)
3. [Command Reference](#command-reference)
4. [Common Workflows](#common-workflows)
5. [Error Handling](#error-handling)
6. [JSON Schema Definitions](#json-schema-definitions)
7. [Best Practices](#best-practices)

---

## Prerequisites

### Installation

**Binary Download:**
```bash
# Download from: https://github.com/wgpsec/redc/releases
# Extract and ensure redc is in your PATH
wget https://github.com/wgpsec/redc/releases/latest/download/redc_<version>_<os>_<arch>.tar.gz
tar -xzf redc_<version>_<os>_<arch>.tar.gz
sudo mv redc /usr/local/bin/
```

**From Source:**
```bash
git clone https://github.com/wgpsec/redc.git
cd redc
goreleaser --snapshot --clean
# Binary will be in dist/ directory
```

### Required Configuration Files

1. **Config File Location:** `~/redc/config.yaml`
2. **Templates Directory:** `~/redc/redc-templates/`
3. **Project Directory:** `~/redc/projects/` (auto-created)

---

## Configuration

### Configuration File Structure

Create `~/redc/config.yaml` with cloud provider credentials:

```yaml
# Multi-cloud credentials and default regions
providers:
  aws:
    AWS_ACCESS_KEY_ID: "AKIDXXXXXXXXXXXXXXXX"
    AWS_SECRET_ACCESS_KEY: "WWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWW"
    region: "us-east-1"
  aliyun:
    ALICLOUD_ACCESS_KEY: "AKIDXXXXXXXXXXXXXXXX"
    ALICLOUD_SECRET_KEY: "WWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWW"
    region: "cn-hangzhou"
  tencentcloud:
    TENCENTCLOUD_SECRET_ID: "AKIDXXXXXXXXXXXXXXXX"
    TENCENTCLOUD_SECRET_KEY: "WWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWW"
    region: "ap-guangzhou"
  volcengine:
    VOLCENGINE_ACCESS_KEY: "AKIDXXXXXXXXXXXXXXXX"
    VOLCENGINE_SECRET_KEY: "WWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWW"
    region: "cn-beijing"
  huaweicloud:
    HUAWEICLOUD_ACCESS_KEY: "AKIDXXXXXXXXXXXXXXXX"
    HUAWEICLOUD_SECRET_KEY: "WWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWW"
    region: "cn-north-4"
  google:
    GOOGLE_CREDENTIALS: '{"type":"service_account","project_id":"your-project",...}'
    project: "your-project-id"
    region: "us-central1"
  azure:
    ARM_CLIENT_ID: "00000000-0000-0000-0000-000000000000"
    ARM_CLIENT_SECRET: "your-client-secret"
    ARM_SUBSCRIPTION_ID: "00000000-0000-0000-0000-000000000000"
    ARM_TENANT_ID: "00000000-0000-0000-0000-000000000000"
  oracle:
    OCI_CLI_USER: "ocid1.user.oc1..aaaaaaa..."
    OCI_CLI_TENANCY: "ocid1.tenancy.oc1..aaaaaaa..."
    OCI_CLI_FINGERPRINT: "aa:bb:cc:dd:ee:ff:00:11:22:33:44:55:66:77:88:99"
    OCI_CLI_KEY_FILE: "~/.oci/oci_api_key.pem"
    OCI_CLI_REGION: "us-ashburn-1"
  cloudflare:
    CF_EMAIL: "you@example.com"
    CF_API_KEY: "your-cloudflare-api-key"
```

### Environment Variables (Alternative)

If `config.yaml` is not available, redc will read from environment variables:

**AWS:**
```bash
export AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
export AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
```

**Alibaba Cloud:**
```bash
export ALICLOUD_ACCESS_KEY="<AccessKey ID>"
export ALICLOUD_SECRET_KEY="<AccessKey Secret>"
export ALICLOUD_SECURITY_TOKEN="<STS Token>"  # Optional
```

**Tencent Cloud:**
```bash
export TENCENTCLOUD_SECRET_ID=<YourSecretId>
export TENCENTCLOUD_SECRET_KEY=<YourSecretKey>
```

---

## Command Reference

### Global Flags

All commands support these global flags:

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--config` | string | `~/redc/config.yaml` | Path to configuration file |
| `--runpath` | string | `~/redc` | Runtime path for redc files |
| `-u, --user` | string | `system` | Operator/user identifier |
| `--project` | string | `default` | Project name |
| `--debug` | bool | `false` | Enable debug mode |
| `-v, --version` | bool | `false` | Show version information |

### Core Commands

#### 1. `redc init`

Initialize templates and prepare the environment.

**Syntax:**
```bash
redc init [global-flags]
```

**Description:**
- Scans all templates in `~/redc/redc-templates/`
- Initializes Terraform providers for each template
- Caches providers for faster deployments

**Example:**
```bash
redc init
redc init --debug
```

**Output:**
- Success: `✅「<template>」场景初始化完成`
- Error: `❌「<template>」场景初始化失败: <error>`

---

#### 2. `redc pull`

Download templates from the registry.

**Syntax:**
```bash
redc pull <image>[:tag] [flags]
```

**Flags:**
| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-r, --registry` | string | `https://redc.wgpsec.org` | Registry URL |
| `-f, --force` | bool | `false` | Force pull (overwrite existing) |
| `--timeout` | duration | `60s` | Download timeout |

**Examples:**
```bash
# Pull a template
redc pull aliyun/ecs

# Pull specific version with force
redc pull aliyun/ecs:v1.2.0 --force

# Pull from custom registry
redc pull aws/ec2 -r https://custom-registry.com
```

**Output:**
- Downloads template to `~/redc/redc-templates/<provider>/<template>/`

---

#### 3. `redc image ls`

List all available local templates.

**Syntax:**
```bash
redc image ls [global-flags]
```

**Example:**
```bash
redc image ls
```

**Output Format:**
```
Available templates:
- aliyun/ecs
- aws/ec2
- tencentcloud/cvm
```

---

#### 4. `redc image rm`

Remove a local template.

**Syntax:**
```bash
redc image rm <template-name>
```

**Example:**
```bash
redc image rm aliyun/ecs
```

---

#### 5. `redc create`

Create a new infrastructure scenario (without starting).

**Syntax:**
```bash
redc create <template-name> [flags]
```

**Flags:**
| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-u, --user` | string | `system` | User/operator identifier |
| `-n, --name` | string | auto-generated | Custom case name |
| `-e, --env` | key=value | - | Set environment variables |

**Examples:**
```bash
# Create with default settings
redc create aliyun/ecs

# Create with custom name and user
redc create aliyun/ecs -u team1 -n operation_alpha

# Create with environment variables
redc create aliyun/ecs -e password=MyPass123 -e region=us-east-1

# Create with multiple variables
redc create aws/ec2 -e instance_type=t2.micro -e key_name=mykey
```

**Output:**
```
✅「aliyun/ecs」场景创建完成！
Case ID: 8a57078ee8567cf2459a0358bc27e534
Case Name: operation_alpha
```

**Return Values:**
- `Case ID`: Unique identifier (64-char hex or 12-char prefix)
- `Case Name`: Human-readable name

---

#### 6. `redc start`

Start/apply a created scenario.

**Syntax:**
```bash
redc start <case-id> [global-flags]
```

**Examples:**
```bash
# Start by full ID
redc start 8a57078ee8567cf2459a0358bc27e534

# Start by short ID (first 12 chars)
redc start 8a57078ee856

# Start by case name
redc start operation_alpha
```

**Output:**
- Progress: Terraform apply output
- Success: `✅ start 操作执行成功: 「<name>」<id>`
- Error: `执行「start」失败，<error>`

---

#### 7. `redc run`

Create and immediately start a scenario (combines create + start).

**Syntax:**
```bash
redc run <template-name> [flags]
```

**Flags:** Same as `create` command

**Examples:**
```bash
# Quick deployment
redc run aliyun/ecs

# With custom configuration
redc run aliyun/ecs -u team1 -n quick_deploy -e password=SecurePass123

# With multiple environment variables
redc run aws/ec2 -e instance_type=t3.medium -e ami=ami-12345678
```

**Output:**
- Creation output + Start output
- Final case ID and status

---

#### 8. `redc ps`

List all scenarios in the current project.

**Syntax:**
```bash
redc ps [global-flags]
```

**Examples:**
```bash
# List all cases in default project
redc ps

# List cases in specific project
redc ps --project production

# List with debug info
redc ps --debug
```

**Output Format:**
```
CASE ID          NAME              STATUS    TEMPLATE      USER     CREATED
8a57078ee856     operation_alpha   running   aliyun/ecs    team1    2024-01-23
3b21456cd789     test_instance     stopped   aws/ec2       system   2024-01-22
```

---

#### 9. `redc status`

Check the status of a specific scenario.

**Syntax:**
```bash
redc status <case-id>
```

**Examples:**
```bash
redc status 8a57078ee856
redc status operation_alpha
```

**Output:**
- Terraform state information
- Resource status
- Instance details (IP, status, etc.)

---

#### 10. `redc stop`

Stop/destroy infrastructure for a scenario.

**Syntax:**
```bash
redc stop <case-id> [global-flags]
```

**Examples:**
```bash
redc stop 8a57078ee856
redc stop operation_alpha
```

**Output:**
- Terraform destroy progress
- Success: `✅ stop 操作执行成功: 「<name>」<id>`
- Error: `执行「stop」失败，<error>`

---

#### 11. `redc kill`

Initialize, then stop and delete a scenario completely.

**Syntax:**
```bash
redc kill <case-id> [global-flags]
```

**Examples:**
```bash
redc kill 8a57078ee856
```

**Description:**
- Runs `init` on template (if needed)
- Stops all resources
- Removes scenario from project

---

#### 12. `redc rm`

Remove a scenario (must be stopped first).

**Syntax:**
```bash
redc rm <case-id> [global-flags]
```

**Examples:**
```bash
redc rm 8a57078ee856
```

**Warning:** Ensure the scenario is stopped before removing to avoid orphaned resources.

---

#### 13. `redc change`

Modify/update a running scenario.

**Syntax:**
```bash
redc change <case-id> [flags]
```

**Flags:**
| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--rm` | bool | `false` | Destroy resources during change |

**Examples:**
```bash
# Change elastic IP or other resources
redc change 8a57078ee856

# Change with resource destruction
redc change 8a57078ee856 --rm
```

**Note:** Template must support change operations (e.g., IP rotation).

---

#### 14. `redc exec`

Execute commands on remote instances.

**Syntax:**
```bash
redc exec [flags] <case-id> <command>
```

**Flags:**
| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-t, --tty` | bool | `false` | Interactive mode (allocate pseudo-TTY) |

**Examples:**
```bash
# Execute single command
redc exec 8a57078ee856 whoami
redc exec 8a57078ee856 "cat /etc/os-release"

# Interactive shell
redc exec -t 8a57078ee856 bash
redc exec -t 8a57078ee856 /bin/sh

# Complex commands
redc exec 8a57078ee856 "ps aux | grep nginx"
```

**Output:**
- Non-interactive: Command output
- Interactive: TTY session

---

#### 15. `redc cp`

Copy files between local and remote machines.

**Syntax:**
```bash
redc cp <source> <destination>
```

**Format:**
- Local path: `/path/to/file`
- Remote path: `<case-id>:/path/to/file`

**Examples:**
```bash
# Upload file to remote
redc cp ./tool 8a57078ee856:/tmp/tool
redc cp ./config.yaml 8a57078ee856:/root/config.yaml

# Download file from remote
redc cp 8a57078ee856:/var/log/syslog ./local_log
redc cp 8a57078ee856:/root/output.txt ./

# Upload directory (recursive)
redc cp -r ./tools 8a57078ee856:/opt/
```

**Output:**
- Success: `上传成功` or `下载成功`
- Error: Connection or transfer errors

---

#### 16. `redc logs`

View logs for a scenario (if implemented).

**Syntax:**
```bash
redc logs <case-id> [flags]
```

---

#### 17. `redc compose`

Orchestrate multiple services (Work In Progress).

**Syntax:**
```bash
redc compose up [flags]
redc compose down [flags]
```

**Configuration File:** `redc-compose.yaml`

**Examples:**
```bash
# Start all services
redc compose up

# Start with specific profile
redc compose up --profile prod

# Stop all services
redc compose down
```

---

### Utility Commands

#### `redc completion`

Generate shell completion scripts.

**Syntax:**
```bash
redc completion <shell>
```

**Supported Shells:** `bash`, `zsh`, `fish`, `powershell`

**Examples:**
```bash
# Bash
source <(redc completion bash)

# Zsh
source <(redc completion zsh)

# Fish
redc completion fish | source

# PowerShell
redc completion powershell | Out-String | Invoke-Expression
```

---

## Common Workflows

### Workflow 1: Quick Deployment

```bash
# 1. Pull template
redc pull aliyun/ecs

# 2. Initialize (first time only)
redc init

# 3. Deploy instance
redc run aliyun/ecs -n my_server -e password=SecurePass123

# Output will include Case ID, e.g., 8a57078ee856

# 4. Check status
redc ps

# 5. Execute commands
redc exec 8a57078ee856 whoami
```

### Workflow 2: Controlled Deployment

```bash
# 1. Create scenario (plan without applying)
redc create aws/ec2 -n staging_server -e instance_type=t2.small

# 2. Review the plan (Case ID returned, e.g., 3b21456cd789)

# 3. Start when ready
redc start 3b21456cd789

# 4. Monitor status
redc status 3b21456cd789

# 5. Upload files
redc cp ./deploy.sh 3b21456cd789:/root/deploy.sh

# 6. Execute deployment script
redc exec 3b21456cd789 "chmod +x /root/deploy.sh && /root/deploy.sh"
```

### Workflow 3: Multi-Project Management

```bash
# Project 1: Red Team Operation
redc create aliyun/ecs --project redteam -u operator1 -n c2_server

# Project 2: Blue Team Testing
redc create aws/ec2 --project blueteam -u tester1 -n test_target

# List cases in specific project
redc ps --project redteam
redc ps --project blueteam

# Operate on specific project cases
redc start <case-id> --project redteam
```

### Workflow 4: Cleanup

```bash
# Stop instance
redc stop 8a57078ee856

# Remove case
redc rm 8a57078ee856

# Or use kill for force cleanup
redc kill 8a57078ee856
```

---

## Error Handling

### Common Errors and Solutions

#### 1. Configuration File Not Found

**Error:**
```
配置载失败！open /home/user/redc/config.yaml: no such file or directory
```

**Solution:**
```bash
# Create config file
mkdir -p ~/redc
vi ~/redc/config.yaml
# Add provider credentials as shown in Configuration section
```

#### 2. Template Not Found

**Error:**
```
❌「<template>」场景创建失败
template not found
```

**Solution:**
```bash
# Pull the template first
redc pull <template-name>

# Initialize
redc init
```

#### 3. Case ID Not Found

**Error:**
```
操作失败: 找不到 ID 为「<id>」的场景
```

**Solution:**
```bash
# List all cases to verify ID
redc ps

# Use correct case ID or name
```

#### 4. SSH Connection Failed

**Error:**
```
连接失败: dial tcp <ip>:22: i/o timeout
```

**Solutions:**
- Ensure instance is running: `redc status <case-id>`
- Check network connectivity
- Verify security group rules allow SSH (port 22)
- Wait for instance initialization to complete

#### 5. Terraform Provider Initialization Failed

**Error:**
```
❌「<template>」场景初始化失败: terraform init failed
```

**Solutions:**
```bash
# Check internet connectivity
ping terraform.io

# Configure Terraform mirror (for China users)
cat > ~/.terraformrc << EOF
plugin_cache_dir = "$HOME/.terraform.d/plugin-cache"
provider_installation {
  network_mirror {
    url = "https://mirrors.aliyun.com/terraform/"
  }
}
EOF

# Retry initialization
redc init
```

#### 6. Insufficient Permissions

**Error:**
```
Error: AccessDenied: User not authorized to perform: <action>
```

**Solution:**
- Verify cloud provider credentials in `~/redc/config.yaml`
- Ensure account has necessary permissions
- Check if access keys are valid and not expired

---

## JSON Schema Definitions

### Case Information Schema

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "Redc Case",
  "type": "object",
  "properties": {
    "id": {
      "type": "string",
      "description": "Unique case identifier (64-char hex)",
      "pattern": "^[a-f0-9]{64}$"
    },
    "short_id": {
      "type": "string",
      "description": "Short case identifier (first 12 chars)",
      "pattern": "^[a-f0-9]{12}$"
    },
    "name": {
      "type": "string",
      "description": "Human-readable case name"
    },
    "template": {
      "type": "string",
      "description": "Template identifier (e.g., aliyun/ecs)"
    },
    "user": {
      "type": "string",
      "description": "Operator/user who created the case"
    },
    "project": {
      "type": "string",
      "description": "Project name"
    },
    "status": {
      "type": "string",
      "enum": ["created", "running", "stopped", "error"],
      "description": "Current case status"
    },
    "created_at": {
      "type": "string",
      "format": "date-time",
      "description": "Creation timestamp"
    },
    "outputs": {
      "type": "object",
      "description": "Terraform outputs (e.g., IP addresses)",
      "additionalProperties": true
    }
  },
  "required": ["id", "name", "template", "status"]
}
```

### Configuration Schema

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "Redc Configuration",
  "type": "object",
  "properties": {
    "providers": {
      "type": "object",
      "description": "Cloud provider configurations",
      "properties": {
        "aws": {
          "type": "object",
          "properties": {
            "AWS_ACCESS_KEY_ID": { "type": "string" },
            "AWS_SECRET_ACCESS_KEY": { "type": "string" },
            "region": { "type": "string" }
          },
          "required": ["AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY"]
        },
        "aliyun": {
          "type": "object",
          "properties": {
            "ALICLOUD_ACCESS_KEY": { "type": "string" },
            "ALICLOUD_SECRET_KEY": { "type": "string" },
            "region": { "type": "string" }
          },
          "required": ["ALICLOUD_ACCESS_KEY", "ALICLOUD_SECRET_KEY"]
        },
        "tencentcloud": {
          "type": "object",
          "properties": {
            "TENCENTCLOUD_SECRET_ID": { "type": "string" },
            "TENCENTCLOUD_SECRET_KEY": { "type": "string" },
            "region": { "type": "string" }
          },
          "required": ["TENCENTCLOUD_SECRET_ID", "TENCENTCLOUD_SECRET_KEY"]
        }
      }
    }
  },
  "required": ["providers"]
}
```

### Environment Variables Schema

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "Template Environment Variables",
  "type": "object",
  "description": "Key-value pairs for template configuration",
  "additionalProperties": {
    "type": "string"
  },
  "examples": [
    {
      "password": "SecurePass123",
      "region": "us-east-1",
      "instance_type": "t2.micro",
      "key_name": "my-ssh-key"
    }
  ]
}
```

---

## Best Practices

### 1. ID Management

- **Use short IDs**: The first 12 characters are sufficient for most operations
- **Use case names**: Assign meaningful names with `-n` flag for easier management
- **Consistent naming**: Use a naming convention like `<project>_<purpose>_<env>`

Example:
```bash
redc create aliyun/ecs -n redteam_c2_prod
redc create aws/ec2 -n blueteam_test_staging
```

### 2. Project Organization

- **Separate projects**: Use different projects for different operations
- **User tracking**: Always specify user with `-u` flag for audit trails
- **Environment isolation**: Use different projects for dev/staging/prod

Example:
```bash
# Development
redc run aliyun/ecs --project dev -u developer1 -n dev_server

# Production
redc run aliyun/ecs --project prod -u operator1 -n prod_server
```

### 3. Security Practices

- **Secure credentials**: Never commit `config.yaml` to version control
- **Use environment variables**: For CI/CD, prefer environment variables over config files
- **Rotate keys**: Regularly rotate cloud provider access keys
- **Limit permissions**: Use least-privilege principle for cloud accounts

### 4. Resource Management

- **Always cleanup**: Use `redc stop` + `redc rm` or `redc kill` when done
- **Monitor costs**: Use `redc ps` regularly to track active instances
- **Use stop before rm**: Ensure resources are destroyed before removing cases

Example cleanup workflow:
```bash
# 1. List all cases
redc ps

# 2. Stop unwanted cases
redc stop <case-id>

# 3. Remove stopped cases
redc rm <case-id>

# Or use kill for force cleanup
redc kill <case-id>
```

### 5. Template Management

- **Initialize regularly**: Run `redc init` after pulling new templates
- **Version control**: Use specific template versions in production
- **Local cache**: Keep frequently used templates local

```bash
# Pull specific version
redc pull aliyun/ecs:v1.0.0

# Force update
redc pull aliyun/ecs --force
```

### 6. Debugging

- **Enable debug mode**: Use `--debug` flag for detailed logs
- **Check status first**: Always run `redc status <case-id>` before operations
- **Review Terraform output**: Debug messages show Terraform operations

```bash
# Debug a creation issue
redc create aliyun/ecs --debug

# Debug connection issues
redc exec 8a57078ee856 whoami --debug
```

### 7. Automation Integration

For AI agents and automation tools:

```python
import subprocess
import json

def redc_run(template, name, env_vars=None):
    """Run a redc deployment."""
    cmd = ["redc", "run", template, "-n", name]
    
    if env_vars:
        for key, value in env_vars.items():
            cmd.extend(["-e", f"{key}={value}"])
    
    try:
        result = subprocess.run(cmd, capture_output=True, text=True, check=True)
        # Parse output to extract case ID
        # Look for pattern like: 8a57078ee8567cf2459a0358bc27e534
        return result.stdout
    except subprocess.CalledProcessError as e:
        print(f"Error: {e.stderr}")
        return None

def redc_list_cases():
    """List all cases."""
    result = subprocess.run(["redc", "ps"], capture_output=True, text=True)
    return result.stdout

def redc_cleanup(case_id):
    """Stop and remove a case."""
    subprocess.run(["redc", "stop", case_id], check=True)
    subprocess.run(["redc", "rm", case_id], check=True)

# Example usage
case_output = redc_run("aliyun/ecs", "ai_deploy_001", {"password": "SecurePass123"})
print(case_output)
```

### 8. Error Recovery

Always implement retry logic for transient failures:

```bash
# Retry logic example
for i in {1..3}; do
    redc start <case-id> && break || sleep 10
done
```

### 9. Output Parsing

AI agents should parse command outputs to extract:
- Case IDs (64-char hex or 12-char prefix)
- Status indicators (`✅` for success, `❌` for error)
- IP addresses and other outputs from Terraform

Example patterns:
```
Success: ✅「<name>」<id> 场景创建完成！
Error: ❌「<name>」场景创建失败
Case ID: [a-f0-9]{64}
IP: \d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}
```

---

## Advanced Features

### Compose Orchestration (WIP)

The `redc compose` feature allows orchestrating multiple services together.

**Configuration File:** `redc-compose.yaml`

```yaml
version: "3.9"

# Global configurations
configs:
  admin_ssh_key:
    file: ~/.ssh/id_rsa.pub

# Services
services:
  teamserver:
    image: ecs
    provider: ali_hk_main
    container_name: ts_leader
    environment:
      - password=StrongPassword123!
      - region=ap-southeast-1
    volumes:
      - ./tools/app.jar:/root/app.jar
    command: |
      java -jar /root/app.jar

  redirector:
    image: nginx-proxy
    provider: aws_us_east
    depends_on:
      - teamserver
    environment:
      - upstream_ip=${teamserver.outputs.public_ip}
```

**Commands:**
```bash
# Start all services
redc compose up

# Start with profile
redc compose up --profile prod

# Stop all services
redc compose down
```

---

## Appendix

### Template Variables

Common environment variables that templates may support:

| Variable | Description | Example |
|----------|-------------|---------|
| `password` | Root/admin password | `SecurePass123!` |
| `region` | Cloud region | `us-east-1`, `cn-hangzhou` |
| `instance_type` | Instance/VM type | `t2.micro`, `ecs.n4.small` |
| `disk_size` | Disk size in GB | `40` |
| `key_name` | SSH key pair name | `my-key` |
| `security_group` | Security group ID | `sg-12345678` |
| `vpc_id` | VPC/VNet ID | `vpc-12345678` |
| `subnet_id` | Subnet ID | `subnet-12345678` |

**Note:** Specific variables depend on the template. Refer to template documentation in the `redc-templates` directory.

### Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | General error |
| `2` | Configuration error |
| `3` | Network/connectivity error |

### Command Summary Table

| Command | Purpose | Requires Case ID |
|---------|---------|------------------|
| `init` | Initialize templates | No |
| `pull` | Download template | No |
| `image ls` | List templates | No |
| `image rm` | Remove template | No |
| `create` | Create scenario | No |
| `run` | Create and start | No |
| `start` | Start scenario | Yes |
| `stop` | Stop scenario | Yes |
| `kill` | Stop and delete | Yes |
| `rm` | Remove scenario | Yes |
| `ps` | List scenarios | No |
| `status` | Check status | Yes |
| `change` | Modify scenario | Yes |
| `exec` | Execute command | Yes |
| `cp` | Copy files | Yes |
| `logs` | View logs | Yes |
| `compose up` | Start orchestration | No |
| `compose down` | Stop orchestration | No |

---

## Support and Resources

- **GitHub Repository:** https://github.com/wgpsec/redc
- **Template Repository:** https://github.com/wgpsec/redc-template
- **Online Templates:** https://redc.wgpsec.org/
- **Documentation:** https://github.com/wgpsec/redc/blob/main/README.md
- **Issues:** https://github.com/wgpsec/redc/issues
- **Discussions:** https://github.com/wgpsec/redc/discussions

---

**Version:** 1.0.0  
**Last Updated:** 2024-01-23  
**License:** Apache 2.0
