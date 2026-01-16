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
<a href="https://github.com/wgpsec/redc/discussions"><strong>探索更多Tricks »</strong></a>
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

中文 | [English](readme_en.md)

---

Redc 基于 Terraform 封装，将红队基础设施的完整生命周期（创建、配置、销毁）进一步简化。

Redc 不仅仅是开机工具，更是对云资源的自动化调度器！

- **一条命令交付**，从购买机器到服务跑起来一条龙，无需人工干预
- **多云部署支持**，适配阿里云、腾讯云、AWS 等主流云厂商
- **场景预制封装**，红队环境 ”预制菜“，再也不用到处找资源
- **状态资源管理**，本地保存资源状态，随时销毁环境，杜绝资源费用浪费

---

## 安装配置

### redc 引擎安装
#### 下载二进制包

REDC 下载地址：https://github.com/wgpsec/redc/releases

下载系统对应的压缩文件，解压后在命令行中运行即可。

#### HomeBrew 安装 （WIP）

**安装**

```bash
brew tap wgpsec/tap
brew install wgpsec/tap/redc
```

**更新**

```bash
brew update
brew upgrade redc
```

### 模版选择

场景名称 - 对应模板仓库 https://github.com/wgpsec/redc-template

放到你 redc-templates 路径下，对应的 "文件夹名称" 就是部署时的场景名称

每个场景的具体使用和命令请查看模板仓库 https://github.com/wgpsec/redc-template 里具体场景的 readme

### 引擎配置文件

默认下 redc 会读取当前路径的 config.yaml 配置文件，格式如下
```yaml
# 多云身份凭证与默认区域
providers:
  aws:
    access_key: "AKIDXXXXXXXXXXXXXXXX"
    secret_key: "WWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWW"
    region: "us-east-1"
  aliyun:
    access_key: "AKIDXXXXXXXXXXXXXXXX"
    secret_key: "WWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWW"
    region: "cn-hangzhou"
  tencentcloud:
    access_key: "AKIDXXXXXXXXXXXXXXXX"
    secret_key: "WWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWW"
    region: "ap-guangzhou"
```

在配置文件加载失败的情况下，会尝试读取系统环境变量，使用前请配置好

---

## 快速上手

redc设计为docker like命令设计

使用 `redc -h` 可以查看常用命令帮助

**初始化模版**

首次使用模版需要运行。为了加快模版部署速度，在修改 `redc-templates` 内容后建议运行 init 选项加快后续部署速度

````bash
redc init
````

![默认init效果](./img/image.png)

> 默认只有 阿里云 ecs 单台机器场景，请自行添加模板至 redc-templates 路径下

**列出模版列表**

```bash
redc image ls
```

**创建实例并启动**

ecs 为模版文件名称

````bash
redc create --name boring_sheep_ecs  [模版名称] # 创建一个实例并plan（该过程不会创建实例，只是检查信息）
# create创建完成后会返回caseid 可使用start命令启动
redc start [caseid]
redc start [casename]
````

直接创建模版名称为 ecs 的 case 并启动

```
redc run ecs
```

![redc run ecs](./img/image2.png)

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

## 编排服务compose

redc 提供了一个编排服务

**启动编排服务**

```
redc compose up
```

**关闭compose**

````
redc compose down
````

文件名称：`redc-compose.yaml`

**compose 模版**

```yaml
version: "3.9"

# ==============================================================================
# 1. Configs: 全局配置中心
# 作用: 定义可复用的静态资源，redc 会将其注入到 Terraform 变量中
# ==============================================================================
configs:
  # [文件型] SSH 公钥
  admin_ssh_key:
    file: ~/.ssh/id_rsa.pub

  # [结构型] 安全组白名单 (将被序列化为 JSON 传递)
  global_whitelist:
    rules:
      - port: 22
        cidr: 1.2.3.4/32
        desc: "Admin Access"
      - port: 80
        cidr: 0.0.0.0/0
        desc: "HTTP Listener"
      - port: 443
        cidr: 0.0.0.0/0
        desc: "HTTPS Listener"

# ==============================================================================
# 2. Plugins: 插件服务 (非计算资源)
# 作用: 独立于服务器的云资源，如 DNS 解析、对象存储、VPC 对等连接等
# ==============================================================================
plugins:
  # 插件 A: 阿里云 DNS 解析
  # 场景: 基础设施启动后，自动将域名指向 Teamserver IP
  dns_record:
    image: plugin-dns-aliyun
    # 引用外部定义的 provider 名称
    provider: ali_hk_main
    environment:
      - domain=redteam-ops.com
      - record=cs
      - type=A
      - value=${teamserver.outputs.public_ip}

  # 插件 B: AWS S3 存储桶 (Loot Box)
  # 场景: 仅在生产环境 ('prod') 启用，用于存放回传数据
  loot_bucket:
    image: plugin-s3
    profiles:
      - prod
    provider: aws_us_east
    environment:
      - bucket_name=rt-ops-2026-logs
      - acl=private

# ==============================================================================
# 3. Services: Case场景
# ==============================================================================
services:

  # ---------------------------------------------------------------------------
  # Service A: 核心控制端 (Teamserver)
  # 特性: 总是启动 (无 profile)，包含完整生命周期钩子和文件流转
  # ---------------------------------------------------------------------------
  teamserver:
    image: ecs
    provider: ali_hk_main
    container_name: ts_leader

    # [Configs] 注入全局配置 (tf_var=config_key)
    configs:
      - ssh_public_key=admin_ssh_key
      - security_rules=global_whitelist

    environment:
      - password=StrongPassword123!
      - region=ap-southeast-1

    # [Volumes] 文件上传 (Local -> Remote)
    # 机器 SSH 连通后立即执行
    volumes:
      - ./tools/cobaltstrike.jar:/root/cs/cobaltstrike.jar
      - ./profiles/amazon.profile:/root/cs/c2.profile
      - ./scripts/init_server.sh:/root/init.sh

    # [Command] 实例内部自启动
    command: |
      chmod +x /root/init.sh
      /root/init.sh start --profile /root/cs/c2.profile

    # [Downloads] 文件回传 (Remote -> Local)
    # 启动完成后抓取凭证
    downloads:
      - /root/cs/.cobaltstrike.beacon_keys:./loot/beacon.keys
      - /root/cs/teamserver.prop:./loot/ts.prop

  # ---------------------------------------------------------------------------
  # Service B: 全球代理矩阵 (Global Redirectors)
  # 特性: 矩阵部署 (Matrix Deployment) + Profiles
  # ---------------------------------------------------------------------------
  global_redirectors:
    image: nginx-proxy

    # [Profiles] 仅在指定模式下启动 (e.g., redc up --profile prod)
    profiles:
      - prod

    # [Matrix] 多 Provider 引用
    # redc 会自动裂变出:
    # 1. global_redirectors_aws_us_east
    # 2. global_redirectors_tencent_sg
    # 3. global_redirectors_ali_jp (假设 providers.yaml 里有这个)
    provider:
      - aws_us_east
      - tencent_sg
      - ali_jp

    depends_on:
      - teamserver

    configs:
      - ingress_rules=global_whitelist

    # 注入当前 provider 的别名
    environment:
      - upstream_ip=${teamserver.outputs.public_ip}
      - node_tag=${provider.alias}

    command: docker run -d -p 80:80 -e UPSTREAM=${teamserver.outputs.public_ip} nginx-proxy

  # ---------------------------------------------------------------------------
  # Service C: 攻击/扫描节点
  # 特性: 攻击模式专用
  # ---------------------------------------------------------------------------
  scan_workers:
    image: aws-ec2-spot
    profiles:
      - attack
    deploy:
      replicas: 5
    provider: aws_us_east
    command: /app/run_scan.sh

# ==============================================================================
# 4. Setup: 联合编排 (Post-Deployment Hooks)
# 作用: 基础设施全部 Ready 后，执行跨机器的注册/交互逻辑
# 注意: redc 会根据当前激活的 Profile 自动跳过未启动服务的相关任务
# ==============================================================================
setup:

  # 任务 1: 基础检查 (总是执行)
  - name: "检查 Teamserver 状态"
    service: teamserver
    command: ./ts_cli status

  # 任务 2: 注册 AWS 代理 (仅 prod 模式有效)
  # 引用裂变后的实例名称: {service}_{provider}
  - name: "注册 AWS 代理节点"
    service: teamserver
    command: >
      ./aggressor_cmd listener_create 
      --name aws_http 
      --host ${global_redirectors_aws_us_east.outputs.public_ip} 
      --port 80

  # 任务 3: 注册 Tencent 代理 (仅 prod 模式有效)
  - name: "注册 Tencent 代理节点"
    service: teamserver
    command: >
      ./aggressor_cmd listener_create 
      --name tencent_http 
      --host ${global_redirectors_tencent_sg.outputs.public_ip} 
      --port 80

  # 任务 4: 注册 Aliyun 代理 (仅 prod 模式有效)
  - name: "注册 Aliyun 代理节点"
    service: teamserver
    command: >
      ./aggressor_cmd listener_create 
      --name ali_http 
      --host ${global_redirectors_ali_jp.outputs.public_ip} 
      --port 80

```

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

- redc 配置文件 (.redc.ini)
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
