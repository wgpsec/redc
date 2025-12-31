package mod

import (
	"fmt"
	"os"

	"gopkg.in/ini.v1"
	"gopkg.in/yaml.v3"
)

var commonCachePath = "./tf-plugin-cache" // Provider 插件缓存目录

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
		} `yaml:"alicloud"`
		Tencentcloud struct {
			SecretId  string `yaml:"secret_id"`
			SecretKey string `yaml:"secret_key"`
			Region    string `yaml:"region"`
		} `yaml:"tencentcloud"`
	} `yaml:"providers"`
}

// LoadCredentials 将配置写入环境变量，Terraform Provider 会自动读取
func LoadCredentials(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("未找到配置文件，将尝试读取系统环境变量: %v", err)
	}

	var conf Config
	if err := yaml.Unmarshal(data, &conf); err != nil {
		return err
	}

	// 设置标准 Terraform 环境变量
	os.Setenv("AWS_ACCESS_KEY_ID", conf.Providers.Aws.AccessKey)
	os.Setenv("AWS_SECRET_ACCESS_KEY", conf.Providers.Aws.SecretKey)

	os.Setenv("ALICLOUD_ACCESS_KEY", conf.Providers.Alicloud.AccessKey)
	os.Setenv("ALICLOUD_SECRET_KEY", conf.Providers.Alicloud.SecretKey)

	os.Setenv("TENCENTCLOUD_SECRET_ID", conf.Providers.Tencentcloud.SecretId)
	os.Setenv("TENCENTCLOUD_SECRET_KEY", conf.Providers.Tencentcloud.SecretKey)

	return nil
}

// LoadConfig 加载配置文件
func LoadConfig(path string) {
	_, err := os.Stat(path)
	if err != nil {
		// 没有配置文件，报错退出，提示进行修改

		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			fmt.Println("配置文件创建失败", err)
		} else {
			fmt.Println("已生成状态文件", path, "请自行修改")

			cfg, err := ini.Load(path)
			if err != nil {
				fmt.Printf("Fail to read file: %v", err)
				os.Exit(3)
			}
			cfg.Section("").Key("operator").SetValue("system")
			cfg.Section("").Key("ALICLOUD_ACCESS_KEY").SetValue("changethis")
			cfg.Section("").Key("ALICLOUD_SECRET_KEY").SetValue("changethis")
			cfg.SaveTo(path)
		}
		defer file.Close()
	}

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
