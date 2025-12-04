# redc

---

## 编译

> 编译前 按需从 https://github.com/wgpsec/redc-template 仓库中把你需要的场景 复制到 redc/utils/redc-templates/ 路径下 !!!

复制后，进行编译
```
goreleaser --snapshot --skip-publish --rm-dist
```

## 本地依赖工具

**mac**

aliyun-cli
```bash
brew install aliyun-cli
```

terraform
```bash
brew tap hashicorp/tap
brew install hashicorp/tap/terraform
brew upgrade hashicorp/tap/terraform
```

jq
```bash
brew install jq
```

aws-cli
- https://docs.aws.amazon.com/zh_cn/cli/latest/userguide/getting-started-install.html


**linux**
```bash
# terraform
mkdir -p /tmp/terraform && cd /tmp/terraform && wget -O terraform_1.6.6_linux_amd64.zip 'https://releases.hashicorp.com/terraform/1.6.6/terraform_1.6.6_linux_amd64.zip'
unzip terraform_1.6.6_linux_amd64.zip
mv --force terraform /usr/local/bin/terraform > /dev/null 2>&1 && chmod +x /usr/local/bin/terraform
rm -rf /tmp/terraform

# 如果是 arm64 机器
# mkdir -p /tmp/terraform && cd /tmp/terraform && wget -O terraform_1.6.6_linux_arm64.zip 'https://releases.hashicorp.com/terraform/1.6.6/terraform_1.6.6_linux_arm64.zip'
# unzip terraform_1.6.6_linux_arm64.zip
# mv --force terraform /usr/local/bin/terraform > /dev/null 2>&1 && chmod +x /usr/local/bin/terraform
# rm -rf /tmp/terraform

cd /tmp
terraform -version

# aliyun
mkdir -p /tmp/aliyuncli && cd /tmp/aliyuncli && wget -O aliyun-cli-linux-latest-amd64.tgz 'https://aliyuncli.alicdn.com/aliyun-cli-linux-latest-amd64.tgz?file=aliyun-cli-linux-latest-amd64.tgz'
tar -xzvf aliyun-cli-linux-latest-amd64.tgz
mv --force aliyun /usr/local/bin/aliyun > /dev/null 2>&1 && chmod +x /usr/local/bin/aliyun
rm -rf /tmp/aliyuncli

apt install jq || yum install jq

# aws
https://docs.aws.amazon.com/zh_cn/cli/latest/userguide/getting-started-install.html
aws configure
```

## 依赖配置

**aliyun cli 配置**
```bash
aliyun configure set --profile cloud-tool --mode AK --region cn-beijing --access-key-id xxxxxxxxxxxxxx --access-key-secret xxxxxxxxxxxxxx
```

**aws-cli 配置**
```
aws configure
AKIAXXXXXXXXXXXXX
XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
ap-east-1
text
```

**配置rclone** (如果用代理池传r2 就配置,如果不用可以不用配置)
```
rclone config
s3
Cloudflare R2 Storage
XXXXXXXXXXXXXXXXXXX
XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
https://XXXXXXXXXXXXXXXXXXXXXXXXXXX.r2.cloudflarestorage.com
auto

rclone lsf r2:test
```

配置tf插件缓存路径
```bash
echo 'plugin_cache_dir = "$HOME/.terraform.d/plugin-cache"' > ~/.terraformrc
```

使用前需初始化redc，将自动下载tf模块依赖 (如果重新对模板打包,则再次编译后还需要进行初始化)
```
./redc -init
```

## 交互使用

```bash
# 开启 awvs 场景
redc -start awvs -u zhangsan
.........
.........
场景uuid:xxxxxxxxx

# 查看指定场景的状态
redc -status [uuid]

# 关闭指定场景
redc -stop [uuid]

# 查看所有场景
redc -list
uuid        type    createtime      operator
xxxxxxxxx   awvs    2022.02.22      system
bbbbbbbbb   file    2022.02.22      system
```

场景名称 - 对应模板仓库 https://github.com/wgpsec/redc-template

按你放到redc/utils/redc-templates/ 路径下的"文件夹名称"来

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
