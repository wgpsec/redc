package mod

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"red-cloud/mod/gologger"

	"gopkg.in/ini.v1"
	"gopkg.in/yaml.v3"
)

var commonCachePath = "./tf-plugin-cache" // Provider 插件缓存目录
var ProjectPath = "./redc-taskresult"
var ProjectFile = "project.json"
var planPath = "case.tfplan"

// Config 结构体用于解析 YAML
type Config struct {
	Providers struct {
		Aws struct {
			AccessKey string `yaml:"access_key"`
			SecretKey string `yaml:"secret_key"`
			Region    string `yaml:"region"`
		} `yaml:"aws"`
		Alicloud struct {
			AccessKey string `yaml:"access_key"`
			SecretKey string `yaml:"secret_key"`
			Region    string `yaml:"region"`
		} `yaml:"aliyun"`
		Tencentcloud struct {
			SecretId  string `yaml:"secret_id"`
			SecretKey string `yaml:"secret_key"`
			Region    string `yaml:"region"`
		} `yaml:"tencentcloud"`
	} `yaml:"providers"`
}

// LoadConfig 将配置写入环境变量，Terraform Provider 会自动读取
func LoadConfig(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("未找到配置文件，将尝试读取系统环境变量: %v", err)
	}

	var conf Config
	if err := yaml.Unmarshal(data, &conf); err != nil {
		return err
	}
	// 设置标准 Terraform 环境变量
	usr, err := user.Current()
	err = os.Setenv("TF_PLUGIN_CACHE_DIR", filepath.Join(usr.HomeDir, ".terraform.d", "plugin-cache"))
	if err != nil {
		gologger.Error().Msgf("设置环境变量失败: %s", err)
	}
	os.Setenv("AWS_ACCESS_KEY_ID", conf.Providers.Aws.AccessKey)
	os.Setenv("AWS_SECRET_ACCESS_KEY", conf.Providers.Aws.SecretKey)

	os.Setenv("ALICLOUD_ACCESS_KEY", conf.Providers.Alicloud.AccessKey)
	os.Setenv("ALICLOUD_SECRET_KEY", conf.Providers.Alicloud.SecretKey)

	os.Setenv("TENCENTCLOUD_SECRET_ID", conf.Providers.Tencentcloud.SecretId)
	os.Setenv("TENCENTCLOUD_SECRET_KEY", conf.Providers.Tencentcloud.SecretKey)

	return nil
}

// ParseConfig 解析配置文件
func ParseConfig(path string) (string, string) {
	cfg, err := ini.Load(path)
	if err != nil {
		fmt.Printf("Fail to read file: %v", err)
		os.Exit(1)
	}

	ALICLOUD_ACCESS_KEY := cfg.Section("").Key("ALICLOUD_ACCESS_KEY").String()
	ALICLOUD_SECRET_KEY := cfg.Section("").Key("ALICLOUD_SECRET_KEY").String()

	return ALICLOUD_ACCESS_KEY, ALICLOUD_SECRET_KEY
}
