# Redc Skills - AI 运维集成指南

## 概述

本文档为 AI 代理和自动化工具提供与 `redc`（Red Cloud）命令行工具交互的全面规范。Redc 是一个基于 Terraform 构建的红队基础设施多云自动化部署工具，旨在简化云基础设施的完整生命周期（创建、配置、销毁）。

## 目录

1. [前置要求](#前置要求)
2. [配置说明](#配置说明)
3. [命令参考](#命令参考)
4. [常用工作流](#常用工作流)
5. [错误处理](#错误处理)
6. [JSON Schema 定义](#json-schema-定义)
7. [最佳实践](#最佳实践)

---

## 前置要求

### 安装方式

**二进制包下载：**
```bash
# 下载地址：https://github.com/wgpsec/redc/releases
# 解压后确保 redc 在 PATH 中
wget https://github.com/wgpsec/redc/releases/latest/download/redc_<version>_<os>_<arch>.tar.gz
tar -xzf redc_<version>_<os>_<arch>.tar.gz
sudo mv redc /usr/local/bin/
```

**从源码编译：**
```bash
git clone https://github.com/wgpsec/redc.git
cd redc
goreleaser --snapshot --clean
# 编译后的二进制文件在 dist/ 目录
```

### 必需的配置文件

1. **配置文件位置：** `~/redc/config.yaml`
2. **模板目录：** `~/redc/redc-templates/`
3. **项目目录：** `~/redc/projects/`（自动创建）

---

## 配置说明

### 配置文件结构

创建 `~/redc/config.yaml` 并配置云服务商凭证：

```yaml
# 多云身份凭证与默认区域
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

### 环境变量（备选方案）

如果 `config.yaml` 不可用，redc 会从环境变量读取配置：

**AWS：**
```bash
export AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
export AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
```

**阿里云：**
```bash
export ALICLOUD_ACCESS_KEY="<AccessKey ID>"
export ALICLOUD_SECRET_KEY="<AccessKey Secret>"
export ALICLOUD_SECURITY_TOKEN="<STS Token>"  # 可选
```

**腾讯云：**
```bash
export TENCENTCLOUD_SECRET_ID=您的SecretId
export TENCENTCLOUD_SECRET_KEY=您的SecretKey
```

---

## 命令参考

### 全局参数

所有命令都支持以下全局参数：

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `--config` | string | `~/redc/config.yaml` | 配置文件路径 |
| `--runpath` | string | `~/redc` | redc 文件运行路径 |
| `-u, --user` | string | `system` | 操作员/用户标识 |
| `--project` | string | `default` | 项目名称 |
| `--debug` | bool | `false` | 启用调试模式 |
| `-v, --version` | bool | `false` | 显示版本信息 |

### 核心命令

#### 1. `redc init`

初始化模板并准备环境。

**语法：**
```bash
redc init [全局参数]
```

**说明：**
- 扫描 `~/redc/redc-templates/` 中的所有模板
- 为每个模板初始化 Terraform providers
- 缓存 providers 以加快部署速度

**示例：**
```bash
redc init
redc init --debug
```

**输出：**
- 成功：`✅「<模板名>」场景初始化完成`
- 失败：`❌「<模板名>」场景初始化失败: <错误信息>`

---

#### 2. `redc pull`

从仓库下载模板。

**语法：**
```bash
redc pull <镜像名>[:标签] [参数]
```

**参数：**
| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `-r, --registry` | string | `https://redc.wgpsec.org` | 仓库地址 |
| `-f, --force` | bool | `false` | 强制拉取（覆盖已有） |
| `--timeout` | duration | `60s` | 下载超时时间 |

**示例：**
```bash
# 拉取模板
redc pull aliyun/ecs

# 拉取特定版本并强制覆盖
redc pull aliyun/ecs:v1.2.0 --force

# 从自定义仓库拉取
redc pull aws/ec2 -r https://custom-registry.com
```

**输出：**
- 模板下载到 `~/redc/redc-templates/<提供商>/<模板>/`

---

#### 3. `redc image ls`

列出所有本地可用模板。

**语法：**
```bash
redc image ls [全局参数]
```

**示例：**
```bash
redc image ls
```

**输出格式：**
```
可用模板：
- aliyun/ecs
- aws/ec2
- tencentcloud/cvm
```

---

#### 4. `redc image rm`

删除本地模板。

**语法：**
```bash
redc image rm <模板名>
```

**示例：**
```bash
redc image rm aliyun/ecs
```

---

#### 5. `redc create`

创建新的基础设施场景（不启动）。

**语法：**
```bash
redc create <模板名> [参数]
```

**参数：**
| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `-u, --user` | string | `system` | 用户/操作员标识 |
| `-n, --name` | string | 自动生成 | 自定义场景名称 |
| `-e, --env` | key=value | - | 设置环境变量 |

**示例：**
```bash
# 使用默认设置创建
redc create aliyun/ecs

# 使用自定义名称和用户创建
redc create aliyun/ecs -u team1 -n operation_alpha

# 使用环境变量创建
redc create aliyun/ecs -e password=MyPass123 -e region=us-east-1

# 使用多个变量创建
redc create aws/ec2 -e instance_type=t2.micro -e key_name=mykey
```

**输出：**
```
✅「aliyun/ecs」场景创建完成！
Case ID: 8a57078ee8567cf2459a0358bc27e534
Case Name: operation_alpha
```

**返回值：**
- `Case ID`：唯一标识符（64字符十六进制或12字符前缀）
- `Case Name`：人类可读的名称

---

#### 6. `redc start`

启动/应用已创建的场景。

**语法：**
```bash
redc start <case-id> [全局参数]
```

**示例：**
```bash
# 使用完整 ID 启动
redc start 8a57078ee8567cf2459a0358bc27e534

# 使用短 ID 启动（前12个字符）
redc start 8a57078ee856

# 使用场景名称启动
redc start operation_alpha
```

**输出：**
- 进度：Terraform apply 输出
- 成功：`✅ start 操作执行成功: 「<名称>」<id>`
- 失败：`执行「start」失败，<错误>`

---

#### 7. `redc run`

创建并立即启动场景（相当于 create + start）。

**语法：**
```bash
redc run <模板名> [参数]
```

**参数：** 与 `create` 命令相同

**示例：**
```bash
# 快速部署
redc run aliyun/ecs

# 使用自定义配置
redc run aliyun/ecs -u team1 -n quick_deploy -e password=SecurePass123

# 使用多个环境变量
redc run aws/ec2 -e instance_type=t3.medium -e ami=ami-12345678
```

**输出：**
- 创建输出 + 启动输出
- 最终 case ID 和状态

---

#### 8. `redc ps`

列出当前项目中的所有场景。

**语法：**
```bash
redc ps [全局参数]
```

**示例：**
```bash
# 列出默认项目中的所有场景
redc ps

# 列出特定项目中的场景
redc ps --project production

# 带调试信息列出
redc ps --debug
```

**输出格式：**
```
CASE ID          NAME              STATUS    TEMPLATE      USER     CREATED
8a57078ee856     operation_alpha   running   aliyun/ecs    team1    2024-01-23
3b21456cd789     test_instance     stopped   aws/ec2       system   2024-01-22
```

---

#### 9. `redc status`

检查特定场景的状态。

**语法：**
```bash
redc status <case-id>
```

**示例：**
```bash
redc status 8a57078ee856
redc status operation_alpha
```

**输出：**
- Terraform 状态信息
- 资源状态
- 实例详情（IP、状态等）

---

#### 10. `redc stop`

停止/销毁场景的基础设施。

**语法：**
```bash
redc stop <case-id> [全局参数]
```

**示例：**
```bash
redc stop 8a57078ee856
redc stop operation_alpha
```

**输出：**
- Terraform destroy 进度
- 成功：`✅ stop 操作执行成功: 「<名称>」<id>`
- 失败：`执行「stop」失败，<错误>`

---

#### 11. `redc kill`

初始化后停止并完全删除场景。

**语法：**
```bash
redc kill <case-id> [全局参数]
```

**示例：**
```bash
redc kill 8a57078ee856
```

**说明：**
- 运行模板的 `init`（如果需要）
- 停止所有资源
- 从项目中删除场景

---

#### 12. `redc rm`

删除场景（必须先停止）。

**语法：**
```bash
redc rm <case-id> [全局参数]
```

**示例：**
```bash
redc rm 8a57078ee856
```

**警告：** 删除前确保场景已停止，以避免遗留资源。

---

#### 13. `redc change`

修改/更新运行中的场景。

**语法：**
```bash
redc change <case-id> [参数]
```

**参数：**
| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `--rm` | bool | `false` | 更改时销毁资源 |

**示例：**
```bash
# 更改弹性 IP 或其他资源
redc change 8a57078ee856

# 带资源销毁的更改
redc change 8a57078ee856 --rm
```

**注意：** 模板必须支持更改操作（例如 IP 轮换）。

---

#### 14. `redc exec`

在远程实例上执行命令。

**语法：**
```bash
redc exec [参数] <case-id> <命令>
```

**参数：**
| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `-t, --tty` | bool | `false` | 交互模式（分配伪终端） |

**示例：**
```bash
# 执行单条命令
redc exec 8a57078ee856 whoami
redc exec 8a57078ee856 "cat /etc/os-release"

# 交互式 shell
redc exec -t 8a57078ee856 bash
redc exec -t 8a57078ee856 /bin/sh

# 复杂命令
redc exec 8a57078ee856 "ps aux | grep nginx"
```

**输出：**
- 非交互模式：命令输出
- 交互模式：TTY 会话

---

#### 15. `redc cp`

在本地和远程机器之间复制文件。

**语法：**
```bash
redc cp <源路径> <目标路径>
```

**格式：**
- 本地路径：`/path/to/file`
- 远程路径：`<case-id>:/path/to/file`

**示例：**
```bash
# 上传文件到远程
redc cp ./tool 8a57078ee856:/tmp/tool
redc cp ./config.yaml 8a57078ee856:/root/config.yaml

# 从远程下载文件
redc cp 8a57078ee856:/var/log/syslog ./local_log
redc cp 8a57078ee856:/root/output.txt ./

# 上传目录（递归）
redc cp -r ./tools 8a57078ee856:/opt/
```

**输出：**
- 成功：`上传成功` 或 `下载成功`
- 失败：连接或传输错误

---

#### 16. `redc logs`

查看场景日志（如已实现）。

**语法：**
```bash
redc logs <case-id> [参数]
```

---

#### 17. `redc compose`

编排多个服务（开发中）。

**语法：**
```bash
redc compose up [参数]
redc compose down [参数]
```

**配置文件：** `redc-compose.yaml`

**示例：**
```bash
# 启动所有服务
redc compose up

# 使用特定配置文件启动
redc compose up --profile prod

# 停止所有服务
redc compose down
```

---

### 实用命令

#### `redc completion`

生成 shell 补全脚本。

**语法：**
```bash
redc completion <shell>
```

**支持的 Shell：** `bash`, `zsh`, `fish`, `powershell`

**示例：**
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

## 常用工作流

### 工作流 1：快速部署

```bash
# 1. 拉取模板
redc pull aliyun/ecs

# 2. 初始化（仅首次）
redc init

# 3. 部署实例
redc run aliyun/ecs -n my_server -e password=SecurePass123

# 输出会包含 Case ID，例如：8a57078ee856

# 4. 检查状态
redc ps

# 5. 执行命令
redc exec 8a57078ee856 whoami
```

### 工作流 2：受控部署

```bash
# 1. 创建场景（计划但不应用）
redc create aws/ec2 -n staging_server -e instance_type=t2.small

# 2. 审查计划（返回 Case ID，例如：3b21456cd789）

# 3. 准备好后启动
redc start 3b21456cd789

# 4. 监控状态
redc status 3b21456cd789

# 5. 上传文件
redc cp ./deploy.sh 3b21456cd789:/root/deploy.sh

# 6. 执行部署脚本
redc exec 3b21456cd789 "chmod +x /root/deploy.sh && /root/deploy.sh"
```

### 工作流 3：多项目管理

```bash
# 项目 1：红队行动
redc create aliyun/ecs --project redteam -u operator1 -n c2_server

# 项目 2：蓝队测试
redc create aws/ec2 --project blueteam -u tester1 -n test_target

# 列出特定项目中的场景
redc ps --project redteam
redc ps --project blueteam

# 对特定项目的场景进行操作
redc start <case-id> --project redteam
```

### 工作流 4：清理

```bash
# 停止实例
redc stop 8a57078ee856

# 删除场景
redc rm 8a57078ee856

# 或使用 kill 强制清理
redc kill 8a57078ee856
```

---

## 错误处理

### 常见错误及解决方案

#### 1. 配置文件未找到

**错误：**
```
配置加载失败！open /home/user/redc/config.yaml: no such file or directory
```

**解决方案：**
```bash
# 创建配置文件
mkdir -p ~/redc
vi ~/redc/config.yaml
# 按照配置说明部分添加云服务商凭证
```

#### 2. 模板未找到

**错误：**
```
❌「<模板名>」场景创建失败
template not found
```

**解决方案：**
```bash
# 先拉取模板
redc pull <模板名>

# 初始化
redc init
```

#### 3. Case ID 未找到

**错误：**
```
操作失败: 找不到 ID 为「<id>」的场景
```

**解决方案：**
```bash
# 列出所有场景以验证 ID
redc ps

# 使用正确的 case ID 或名称
```

#### 4. SSH 连接失败

**错误：**
```
连接失败: dial tcp <ip>:22: i/o timeout
```

**解决方案：**
- 确保实例正在运行：`redc status <case-id>`
- 检查网络连接
- 验证安全组规则允许 SSH（端口 22）
- 等待实例初始化完成

#### 5. Terraform Provider 初始化失败

**错误：**
```
❌「<模板名>」场景初始化失败: terraform init failed
```

**解决方案：**
```bash
# 检查网络连接
ping terraform.io

# 配置 Terraform 镜像（中国用户）
cat > ~/.terraformrc << EOF
plugin_cache_dir = "$HOME/.terraform.d/plugin-cache"
provider_installation {
  network_mirror {
    url = "https://mirrors.aliyun.com/terraform/"
  }
}
EOF

# 重试初始化
redc init
```

#### 6. 权限不足

**错误：**
```
Error: AccessDenied: User not authorized to perform: <action>
```

**解决方案：**
- 验证 `~/redc/config.yaml` 中的云服务商凭证
- 确保账户具有必要的权限
- 检查访问密钥是否有效且未过期

---

## JSON Schema 定义

### Case 信息 Schema

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "Redc Case",
  "type": "object",
  "properties": {
    "id": {
      "type": "string",
      "description": "唯一的场景标识符（64字符十六进制）",
      "pattern": "^[a-f0-9]{64}$"
    },
    "short_id": {
      "type": "string",
      "description": "短场景标识符（前12个字符）",
      "pattern": "^[a-f0-9]{12}$"
    },
    "name": {
      "type": "string",
      "description": "人类可读的场景名称"
    },
    "template": {
      "type": "string",
      "description": "模板标识符（例如：aliyun/ecs）"
    },
    "user": {
      "type": "string",
      "description": "创建场景的操作员/用户"
    },
    "project": {
      "type": "string",
      "description": "项目名称"
    },
    "status": {
      "type": "string",
      "enum": ["created", "running", "stopped", "error"],
      "description": "当前场景状态"
    },
    "created_at": {
      "type": "string",
      "format": "date-time",
      "description": "创建时间戳"
    },
    "outputs": {
      "type": "object",
      "description": "Terraform 输出（例如：IP 地址）",
      "additionalProperties": true
    }
  },
  "required": ["id", "name", "template", "status"]
}
```

### 配置 Schema

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "Redc Configuration",
  "type": "object",
  "properties": {
    "providers": {
      "type": "object",
      "description": "云服务商配置",
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

### 环境变量 Schema

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "Template Environment Variables",
  "type": "object",
  "description": "模板配置的键值对",
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

## 最佳实践

### 1. ID 管理

- **使用短 ID**：前12个字符足以用于大多数操作
- **使用场景名称**：使用 `-n` 参数分配有意义的名称，便于管理
- **一致的命名**：使用命名约定如 `<项目>_<用途>_<环境>`

示例：
```bash
redc create aliyun/ecs -n redteam_c2_prod
redc create aws/ec2 -n blueteam_test_staging
```

### 2. 项目组织

- **分离项目**：为不同的操作使用不同的项目
- **用户追踪**：始终使用 `-u` 参数指定用户以进行审计
- **环境隔离**：为 dev/staging/prod 使用不同的项目

示例：
```bash
# 开发环境
redc run aliyun/ecs --project dev -u developer1 -n dev_server

# 生产环境
redc run aliyun/ecs --project prod -u operator1 -n prod_server
```

### 3. 安全实践

- **保护凭证**：永远不要将 `config.yaml` 提交到版本控制
- **使用环境变量**：对于 CI/CD，优先使用环境变量而非配置文件
- **轮换密钥**：定期轮换云服务商访问密钥
- **限制权限**：对云账户使用最小权限原则

### 4. 资源管理

- **始终清理**：完成后使用 `redc stop` + `redc rm` 或 `redc kill`
- **监控成本**：定期使用 `redc ps` 跟踪活动实例
- **先停止再删除**：在删除场景前确保资源已销毁

清理工作流示例：
```bash
# 1. 列出所有场景
redc ps

# 2. 停止不需要的场景
redc stop <case-id>

# 3. 删除已停止的场景
redc rm <case-id>

# 或使用 kill 强制清理
redc kill <case-id>
```

### 5. 模板管理

- **定期初始化**：拉取新模板后运行 `redc init`
- **版本控制**：在生产环境使用特定的模板版本
- **本地缓存**：保持常用模板的本地副本

```bash
# 拉取特定版本
redc pull aliyun/ecs:v1.0.0

# 强制更新
redc pull aliyun/ecs --force
```

### 6. 调试

- **启用调试模式**：使用 `--debug` 参数获取详细日志
- **先检查状态**：操作前始终运行 `redc status <case-id>`
- **查看 Terraform 输出**：调试消息显示 Terraform 操作

```bash
# 调试创建问题
redc create aliyun/ecs --debug

# 调试连接问题
redc exec 8a57078ee856 whoami --debug
```

### 7. 自动化集成

对于 AI 代理和自动化工具：

```python
import subprocess
import json
import re

def redc_run(template, name, env_vars=None):
    """运行 redc 部署。"""
    cmd = ["redc", "run", template, "-n", name]
    
    if env_vars:
        for key, value in env_vars.items():
            cmd.extend(["-e", f"{key}={value}"])
    
    try:
        result = subprocess.run(cmd, capture_output=True, text=True, check=True)
        # 解析输出以提取 case ID
        # 查找类似模式：8a57078ee8567cf2459a0358bc27e534
        match = re.search(r'[a-f0-9]{64}', result.stdout)
        if match:
            return match.group(0)
        return None
    except subprocess.CalledProcessError as e:
        print(f"错误：{e.stderr}")
        return None

def redc_list_cases():
    """列出所有场景。"""
    result = subprocess.run(["redc", "ps"], capture_output=True, text=True)
    return result.stdout

def redc_cleanup(case_id):
    """停止并删除场景。"""
    subprocess.run(["redc", "stop", case_id], check=True)
    subprocess.run(["redc", "rm", case_id], check=True)

# 使用示例
case_id = redc_run("aliyun/ecs", "ai_deploy_001", {"password": "SecurePass123"})
if case_id:
    print(f"部署成功，Case ID：{case_id}")
```

### 8. 错误恢复

对于临时故障始终实施重试逻辑：

```bash
# 重试逻辑示例
for i in {1..3}; do
    redc start <case-id> && break || sleep 10
done
```

### 9. 输出解析

AI 代理应解析命令输出以提取：
- Case ID（64字符十六进制或12字符前缀）
- 状态指示符（`✅` 表示成功，`❌` 表示错误）
- Terraform 的 IP 地址和其他输出

示例模式：
```
成功：✅「<名称>」<id> 场景创建完成！
错误：❌「<名称>」场景创建失败
Case ID: [a-f0-9]{64}
IP: \d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}
```

---

## 高级功能

### Compose 编排（开发中）

`redc compose` 功能允许一起编排多个服务。

**配置文件：** `redc-compose.yaml`

```yaml
version: "3.9"

# 全局配置
configs:
  admin_ssh_key:
    file: ~/.ssh/id_rsa.pub

# 服务
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

**命令：**
```bash
# 启动所有服务
redc compose up

# 使用配置文件启动
redc compose up --profile prod

# 停止所有服务
redc compose down
```

---

## 附录

### 模板变量

模板可能支持的常见环境变量：

| 变量 | 说明 | 示例 |
|------|------|------|
| `password` | Root/admin 密码 | `SecurePass123!` |
| `region` | 云区域 | `us-east-1`, `cn-hangzhou` |
| `instance_type` | 实例/虚拟机类型 | `t2.micro`, `ecs.n4.small` |
| `disk_size` | 磁盘大小（GB） | `40` |
| `key_name` | SSH 密钥对名称 | `my-key` |
| `security_group` | 安全组 ID | `sg-12345678` |
| `vpc_id` | VPC/VNet ID | `vpc-12345678` |
| `subnet_id` | 子网 ID | `subnet-12345678` |

**注意：** 具体变量取决于模板。请参考 `redc-templates` 目录中的模板文档。

### 退出码

| 代码 | 含义 |
|------|------|
| `0` | 成功 |
| `1` | 一般错误 |
| `2` | 配置错误 |
| `3` | 网络/连接错误 |

### 命令摘要表

| 命令 | 用途 | 需要 Case ID |
|---------|---------|------------------|
| `init` | 初始化模板 | 否 |
| `pull` | 下载模板 | 否 |
| `image ls` | 列出模板 | 否 |
| `image rm` | 删除模板 | 否 |
| `create` | 创建场景 | 否 |
| `run` | 创建并启动 | 否 |
| `start` | 启动场景 | 是 |
| `stop` | 停止场景 | 是 |
| `kill` | 停止并删除 | 是 |
| `rm` | 删除场景 | 是 |
| `ps` | 列出场景 | 否 |
| `status` | 检查状态 | 是 |
| `change` | 修改场景 | 是 |
| `exec` | 执行命令 | 是 |
| `cp` | 复制文件 | 是 |
| `logs` | 查看日志 | 是 |
| `compose up` | 启动编排 | 否 |
| `compose down` | 停止编排 | 否 |

---

## 支持与资源

- **GitHub 仓库：** https://github.com/wgpsec/redc
- **模板仓库：** https://github.com/wgpsec/redc-template
- **在线模板：** https://redc.wgpsec.org/
- **文档：** https://github.com/wgpsec/redc/blob/main/README_CN.md
- **问题反馈：** https://github.com/wgpsec/redc/issues
- **讨论区：** https://github.com/wgpsec/redc/discussions

---

**版本：** 1.0.0  
**最后更新：** 2024-01-23  
**许可证：** Apache 2.0
