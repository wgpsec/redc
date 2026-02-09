<p align="center">
  <a href="https://github.com/wgpsec/redc">
    <img src="./img/banner.jpg" width="100%" alt="redc">
  </a>
  <p align="center">
    红队基础设施多云自动化部署工具
    <br />
    <br />
<a href="https://github.com/wgpsec/redc/stargazers"><img alt="GitHub stars" src="https://img.shields.io/github/stars/wgpsec/redc"/></a>
<a href="https://github.com/wgpsec/redc/releases"><img alt="GitHub releases" src="https://img.shields.io/github/release/wgpsec/redc"/></a>
<a href="https://github.com/wgpsec/redc/blob/main/LICENSE"><img alt="License" src="https://img.shields.io/badge/License-Apache%202.0-blue.svg"/></a>
<a href="https://github.com/wgpsec/redc/releases"><img alt="Downloads" src="https://img.shields.io/github/downloads/wgpsec/redc/total?color=brightgreen"/></a>
<a href="https://goreportcard.com/report/github.com/wgpsec/redc"><img alt="Go Report Card" src="https://goreportcard.com/badge/github.com/wgpsec/redc"/></a>
<a href="https://twitter.com/wgpsec"><img alt="Twitter" src="https://img.shields.io/twitter/follow/wgpsec?label=Followers&style=social" /></a>
<br>
<br>
<a href="https://redc.wgpsec.org/"><strong>探索更多模板 »</strong></a>
    <br/>
    <br />
      <a href="https://github.com/wgpsec/redc?tab=readme-ov-file#%E4%BD%BF%E7%94%A8%E6%8C%87%E5%8D%97">🧐如何使用</a>
      ·
    <a href="https://github.com/wgpsec/redc/releases">⬇️下载程序</a>
    ·
    <a href="https://github.com/wgpsec/redc/issues">❔反馈Bug</a>
    ·
    <a href="https://github.com/wgpsec/redc/discussions">🍭提交需求</a>
  </p>

中文 | [English](README.md)

---

## 文档

- **[使用指南](README_CN.md)** - 完整的安装和使用指南
- **[AI 运维 Skills](.claude/skills/useage/SKILL_CN.md)** - AI 代理和自动化工具的综合指南
- **[MCP 协议支持](doc/MCP_CN.md)** - AI 助手的模型上下文协议集成
- **[Compose 编排指南](doc/Compose_CN.md)** - 多服务编排部署最佳实践
- **[模板仓库](https://github.com/wgpsec/redc-template)** - 预配置的基础设施模板
- **[在线模板](https://redc.wgpsec.org/)** - 浏览和下载模板

---

Redc 基于 Terraform 封装，将红队基础设施的完整生命周期（创建、配置、销毁）进一步简化。

Redc 不仅仅是开机工具，更是对云资源的自动化调度器！

- **一条命令交付**，从购买机器到服务跑起来一条龙，无需人工干预
- **多云部署支持**，适配阿里云、腾讯云、AWS 等主流云厂商
- **场景预制封装**，红队环境 ”预制菜“，再也不用到处找资源
- **状态资源管理**，本地保存资源状态，随时销毁环境，杜绝资源费用浪费

---

## GUI 开发与调试

redc GUI 基于 Wails + Svelte + Vite。

![gui](./img/gui.png)

### 开发环境准备

1. 安装 Go（建议 1.21+）
2. 安装 Node.js（建议 18+）
3. 安装 Wails CLI

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

### 启动开发调试

在项目根目录执行：

```bash
wails dev
```

常见问题排查：

- 前端依赖未安装：在 [frontend/](frontend/) 目录执行 `npm install`
- 前端热更新异常：尝试删除 [frontend/node_modules](frontend/node_modules) 后重新安装

---

## GUI 编译与发布

在项目根目录执行：

```bash
wails build

# windows
wails build -platform windows/amd64
```

构建产物输出到 [build/bin](build/bin)。

> 目前可在 releases 下载 gui 版本

如需指定平台或架构，可参考 Wails 官方文档：
https://wails.io/docs/guides/building/

---

## CLI 安装配置

### redc 引擎安装 (第一步)
#### 下载二进制包

REDC 下载地址：https://github.com/wgpsec/redc/releases

下载系统对应的压缩文件，解压后在命令行中运行即可。

#### HomeBrew 安装 （WIP）

**安装**

```bash
brew install wgpsec/tap/redc
```

**更新**

```bash
brew update
brew upgrade redc
```

#### 从源码编译安装

**goreleaser**
```bash
git clone https://github.com/wgpsec/redc.git
cd redc
goreleaser --snapshot --clean

# 编译成功后会在 dist 路径下
```

### 模版选择 (第二步)

默认下 redc 会读取用户目录下的 ~/redc/redc-templates 模板文件夹，对应的 "文件夹名称" 就是部署时的场景名称

可以自行下载模板场景，场景名称对应模板仓库 https://github.com/wgpsec/redc-template

在线地址：https://redc.wgpsec.org/

例如，一键拉取ecs场景
```bash
redc pull aliyun/ecs

# 此时，模板会下载到 ~/redc/redc-templates 目录下
```

![redc pull](./img/image9.png)

每个场景的具体使用和命令请查看模板仓库 https://github.com/wgpsec/redc-template 里具体场景的 readme

### 引擎配置文件 (第三步)

redc 开启机器需要依靠 aksk

默认下 redc 会读取用户路径的 config.yaml 配置文件，格式如下
```
vim ~/redc/config.yaml
```

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

在配置文件加载失败的情况下，会尝试读取系统环境变量，使用前请配置好

**AWS 环境变量**
- 详情参考 : https://docs.aws.amazon.com/sdkref/latest/guide/feature-static-credentials.html

Linux/macOS 示例通过命令行设置环境变量：
```bash
export AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
export AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
```

Windows 示例通过命令行设置环境变量：
```powershell
setx AWS_ACCESS_KEY_ID AKIAIOSFODNN7EXAMPLE
setx AWS_SECRET_ACCESS_KEY wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
```

**阿里云环境变量**
- 详情参考 : https://help.aliyun.com/zh/terraform/terraform-authentication

Linux/macOS 系统
> 使用 export 命令配置的临时环境变量仅对当前 Shell 会话有效。如需长期保留，可将 export 命令写入 Shell 的启动配置文件（如 .bash_profile 或 .zshrc）。
```
# AccessKey ID
$ export ALICLOUD_ACCESS_KEY="<AccessKey ID>"
# AccessKey Secret
$ export ALICLOUD_SECRET_KEY="<AccessKey Secret>"
# 如果使用 STS 凭证，需配置 security_token
$ export ALICLOUD_SECURITY_TOKEN="<STS Token>"
```

Windows 系统
```
在桌面右键单击 此电脑，选择 属性 > 高级系统设置 > 环境变量。
在 系统变量 或 用户变量 中，单击 新建，创建以下环境变量：ALICLOUD_ACCESS_KEY、ALICLOUD_SECRET_KEY、ALICLOUD_SECURITY_TOKEN（可选）。
```

**腾讯云环境变量**
- 详情参考 : https://cloud.tencent.com/document/product/1278/85305

Linux/macOS 系统
```
export TENCENTCLOUD_SECRET_ID=您的SecretId
export TENCENTCLOUD_SECRET_KEY=您的SecretKey
```

Windows 系统
```
set TENCENTCLOUD_SECRET_ID=您的SecretId
set TENCENTCLOUD_SECRET_KEY=您的SecretKey
```

**火山引擎环境变量**
- 详情参考 : https://www.volcengine.com/docs/6291/65568

Linux/macOS 系统
```
export VOLCENGINE_ACCESS_KEY=您的AccessKey
export VOLCENGINE_SECRET_KEY=您的SecretKey
```

Windows 系统
```
set VOLCENGINE_ACCESS_KEY=您的AccessKey
set VOLCENGINE_SECRET_KEY=您的SecretKey
```

---

## 快速上手

redc设计为docker like命令设计

使用 `redc -h` 可以查看常用命令帮助

**初始化模版**

首次使用模版需要运行。为了加快模版部署速度，建议运行 init 选项加快后续部署速度

````bash
redc init
````

![默认init效果](./img/image.png)

默认会 init 在 ~/redc/redc-templates 路径下的所有场景，作用就是刷一遍 tf provider 的 cache

**列出模版列表**

```bash
redc image ls
```

默认会列出在 ~/redc/redc-templates 路径下的所有场景

![redc image ls](./img/image10.png)

**创建实例并启动**

ecs 为模版文件名称

````bash
redc plan --name boring_sheep_ecs  [模版名称] # 规划一个实例（该过程验证配置但不会创建基础设施）
# plan完成后会返回caseid 可使用start命令实际创建基础设施
redc start [caseid]
redc start [casename]
````

也可以直接启动模版

```bash
redc run aliyun/ecs
```

![redc run aliyun/ecs](./img/image11.png)

> 开启后会给出 case id ，这是标识场景唯一性的识别 id，后续操作都需要用到 case id
> 例如 8a57078ee8567cf2459a0358bc27e534cb87c8a02eadc637ce8335046c16cb3c 可以用 8a57078ee856 效果一样

使用`-e` 参数可配置变量

```
redc run -e xxx=xxx ecs
```

停止实例

````bash
redc stop [caseid] # 停止实例
redc rm [caseid] # 删除实例（删除前确认实例是否已经停止）
redc kill [caseid] # init模版后停止实例并删除
````

![redc stop [caseid]](./img/image7.png)

**查看case情况**

````
redc ps
````

![redc ps](./img/image8.png)

**执行命令**

直接执行命令并返回结果

````
redc exec [caseid] whoami
````

![redc exec [caseid] whoami](./img/image3.png)

进入交互式命令

````
redc exec -t [caseid] bash
````

![redc exec -t [caseid] bash](./img/image4.png)

复制文件到服务器

```
redc cp test.txt [caseid]:/root/
```

![redc cp test.txt [caseid]:/root/](./img/image5.png)

下载文件到本地

```
redc cp [caseid]:/root/test.txt ./
```

![redc cp [caseid]:/root/test.txt ./](./img/image6.png)

**更改服务**

这个需要模版支持更改，可实现更换弹性公网ip

````
redc change [caseid]
````

## MCP（模型上下文协议）支持

redc 现已支持模型上下文协议，可与 AI 助手和自动化工具无缝集成。

### 主要特性

- **两种传输模式**：STDIO 用于本地集成，SSE 用于基于 Web 的访问
- **全面的工具**：创建、管理和在基础设施上执行命令
- **AI 友好**：支持 Claude Desktop、自定义 AI 工具和自动化平台
- **安全性**：STDIO 在本地运行无网络暴露；SSE 可限制为 localhost

### 快速开始

**启动 STDIO 服务器**（用于 Claude Desktop 集成）：
```bash
redc mcp stdio
```

**启动 SSE 服务器**（用于基于 Web 的客户端）：
```bash
# 默认（localhost:8080）
redc mcp sse

# 自定义端口
redc mcp sse localhost:9000
```

### 可用工具

- `list_templates` - 列出所有可用模板
- `list_cases` - 列出项目中的所有场景
- `plan_case` - 从模板规划新场景（预览资源而不实际创建）
- `start_case` / `stop_case` / `kill_case` - 管理场景生命周期
- `get_case_status` - 检查场景状态
- `exec_command` - 在场景上执行命令

### 示例：与 Chrerry stdio 集成

以Chrerry stdio为例 填入 http://localhost:9000/sse 即可获得到工具信息

![mcp](./img/image12.png)

使用示例
- 测试

  ![测试](./img/image13.png)

- 开启机器

  ![开启机器](./img/image14.png)

- 执行命令

  ![执行命令](./img/image15.png)

- 关闭机器

  ![关闭机器](./img/image16.png)

### 示例：与 Claude Desktop 集成

添加到 `~/Library/Application Support/Claude/claude_desktop_config.json`：
```json
{
  "mcpServers": {
    "redc": {
      "command": "/path/to/redc",
      "args": ["mcp", "stdio"]
    }
  }
}
```

详细文档请参阅 **[MCP_CN.md](doc/MCP_CN.md)**。

---

## 编排服务 Compose

redc 提供了一个编排服务，可以通过 YAML 配置文件同时管理多个云服务实例，实现复杂的基础设施部署。

### 快速开始

**配置文件示例** ([完整示例](doc/redc-compose.yaml))

```yaml
version: "3.9"

# 服务定义
services:
  # 阿里云 ECS 实例
  aliyun_server:
    image: aliyun/ecs
    container_name: my_aliyun_ecs
    command: |
      echo "阿里云 ECS 初始化完成"
      uptime
  
  # 火山云 ECS 实例
  volcengine_server:
    image: volcengine/ecs
    container_name: my_volcengine_ecs
    command: |
      echo "火山云 ECS 初始化完成"
      uptime

# 后置编排任务
setup:
  - name: "检查阿里云实例"
    service: aliyun_server
    command: hostname && ip addr show

  - name: "检查火山云实例"
    service: volcengine_server
    command: hostname && ip addr show
```

**常用命令**

```bash
# 预览配置
redc compose config redc-compose.yaml

# 启动编排服务
redc compose up redc-compose.yaml

# 关闭编排服务
redc compose down redc-compose.yaml
```

**详细文档**

完整的使用说明、高级功能和最佳实践,请参阅 **[Compose 编排指南](doc/Compose_CN.md)**。

---

## 配置缓存和加速

仅配置缓存地址：

```bash
echo 'plugin_cache_dir = "$HOME/.terraform.d/plugin-cache"' > ~/.terraformrc
```

配置阿里云加速 修改 `/.terraformrc` 文件

```
plugin_cache_dir  = "$HOME/.terraform.d/plugin-cache"
disable_checkpoint = true
provider_installation {
  network_mirror {
    url = "https://mirrors.aliyun.com/terraform/"
    # 限制只有阿里云相关 Provider 从国内镜像源下载
    include = ["registry.terraform.io/aliyun/alicloud",
               "registry.terraform.io/hashicorp/alicloud",
              ]
  }
  direct {
    # 声明除了阿里云相关Provider, 其它Provider保持原有的下载链路
    exclude = ["registry.terraform.io/aliyun/alicloud",
               "registry.terraform.io/hashicorp/alicloud",
              ]
  }
}
```

---

## 设计规划

1. 先创建新项目
2. 指定项目下要创建场景会从场景库复制一份场景文件夹到项目文件夹下
3. 不同项目下创建同一场景互不干扰
4. 同一项目下创建同一场景互不干扰
5. 多用户操作互不干扰(本地有做鉴权,但这个实际上要在平台上去做)

- redc 配置文件 (~/redc/config.yaml)
- 项目1 (./project1)
    - 场景1 (./project1/[uuid1])
        - main.tf
        - version.tf
        - output.tf
    - 场景2 (./project1/[uuid2])
        - main.tf
        - version.tf
        - output.tf
    - 项目状态文件 (project.ini)
- 项目2 (./project2)
    - 场景1 (./project2/[uuid1])
        - main.tf
        - version.tf
        - output.tf
    - 场景2 (./project2/[uuid2])
        - ...
    - 项目状态文件 (project.ini)
- 项目3 (./project3)
    - ...

## 文章介绍

- https://mp.weixin.qq.com/s/JH-IlL_GFgZp3xXeOFzZeQ

## FAQ

### MacOS安装问题

使用编译的 app 包可能会遇到
- xxx已损坏，无法打开，您应该将它移到废纸篓
- 打不开xxx，因为 Apple 无法检查其是否包含恶意软件
- 打不开 xxx，因为它来自身份不明的开发者

苹果系统有一个 GateKeeper 保护机制。从互联网上下载来的文件，会被自动打上 com.apple.quarantine 标志，我们可以理解为 "免疫隔离"。系统根据这个附加属性对这个文件作出限制。

随着版本不同，MacOS 对 com.apple.quarantine 的限制越来越严格，在较新 的 MacOS 中，会直接提示 "映像损坏" 或 "应用损坏" 这类很激进的策略。

我们可以通过手动移除该选项来解决此问题,执行下面的命令:
```
sudo xattr -r -d com.apple.quarantine /Applications/redc-gui.app
```
