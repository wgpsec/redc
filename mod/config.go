package mod

import (
	"fmt"
	"os"
	"path/filepath"
	"red-cloud/mod/gologger"
	"reflect"

	"gopkg.in/yaml.v3"
)

var commonCachePath = "./tf-plugin-cache" // Provider 插件缓存目录
var ProjectPath = "./redc-taskresult"
var ProjectFile = "project.json"
var planPath = "case.tfplan"

// Config 配置文件结构体，新增厂商配置也需要再这里添加
// yaml 为配置文件，env为tf的环境变量参数
type Config struct {
	Providers struct {
		Aws struct {
			AccessKey string `yaml:"AWS_ACCESS_KEY_ID" env:"AWS_ACCESS_KEY_ID"`
			SecretKey string `yaml:"AWS_SECRET_ACCESS_KEY" env:"AWS_SECRET_ACCESS_KEY" `
			Region    string `yaml:"region"`
		} `yaml:"aws"`
		Alicloud struct {
			AccessKey string `yaml:"ALICLOUD_ACCESS_KEY" env:"ALICLOUD_ACCESS_KEY" `
			SecretKey string `yaml:"ALICLOUD_SECRET_KEY" env:"ALICLOUD_SECRET_KEY"`
			Region    string `yaml:"region"`
		} `yaml:"aliyun"`
		Tencentcloud struct {
			SecretId  string `yaml:"TENCENTCLOUD_SECRET_ID" env:"TENCENTCLOUD_SECRET_ID"`
			SecretKey string `yaml:"TENCENTCLOUD_SECRET_KEY" env:"TENCENTCLOUD_SECRET_KEY"`
			Region    string `yaml:"region"`
		} `yaml:"tencentcloud"`
	} `yaml:"providers"`
}

func LoadConfig(path string) error {
	home, _ := os.UserHomeDir() // 忽略错误，home为空也没关系

	// 设置默认缓存路径
	os.Setenv("TF_PLUGIN_CACHE_DIR", filepath.Join(home, ".terraform.d", "plugin-cache"))

	// 如果指定了 path，只查 path；否则查 [用户目录, 程序目录]
	searchPaths := []string{path}
	if path == "" {
		exePath, _ := os.Executable() // 获取程序自身路径
		searchPaths = []string{
			filepath.Join(home, ".redc", "config.yaml"),         // 优先级1: 用户目录
			filepath.Join(filepath.Dir(exePath), "config.yaml"), // 优先级2: 程序旁
		}
	}

	var data []byte
	var err error
	for _, p := range searchPaths {
		if p == "" {
			continue
		}
		if data, err = os.ReadFile(p); err == nil {
			break // 读取成功，跳出循环
		}
	}

	if data == nil {
		gologger.Info().Msgf("未找到配置文件，将尝试环境变量...\n %v\n", searchPaths)
		return fmt.Errorf("配置文件未找到\n")
	}

	var conf Config
	if err := yaml.Unmarshal(data, &conf); err != nil {
		return err
	}

	// 批量配置环境变量
	bindEnv(conf)

	return nil
}

// bindEnv 递归遍历结构体，直接用 `yaml` 标签的值作为环境变量 Key
func bindEnv(v interface{}) {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	} // 处理指针

	for i := 0; i < val.NumField(); i++ {
		fieldVal := val.Field(i)
		fieldType := val.Type().Field(i)

		// 如果是嵌套结构体 (如 Providers -> Aws)，递归处理
		if fieldVal.Kind() == reflect.Struct {
			bindEnv(fieldVal.Interface())
			continue
		}

		// 如果是字符串且有值，读取 yaml 标签并 Setenv
		if tag := fieldType.Tag.Get("env"); tag != "" && fieldVal.String() != "" {
			os.Setenv(tag, fieldVal.String())
		}
	}
}
