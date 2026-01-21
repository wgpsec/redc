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
var RedcPath = ""
var ProjectPath = "redc-taskresult"

const ProjectFile = "project.json"
const RedcPlanPath = "case.tfplan"
const MaxTfDepth = 2

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
	if RedcPath == "" {
		RedcPath = filepath.Join(home, "redc")
	}
	defaultConfigPath := filepath.Join(RedcPath, "config.yaml")
	searchPaths := []string{path}
	if path == "" {
		exePath, _ := os.Executable() // 获取程序自身路径
		searchPaths = []string{
			filepath.Join(filepath.Dir(exePath), "config.yaml"),
			defaultConfigPath,
		}
	}

	TemplateDir = filepath.Join(RedcPath, "templates")
	ProjectPath = filepath.Join(RedcPath, "task-result")

	var data []byte
	var conf Config
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
		gologger.Info().Msgf("未找到配置文件，正在创建默认配置文件: %s\n", defaultConfigPath)
		if err := os.MkdirAll(filepath.Dir(defaultConfigPath), 0755); err != nil {
			return fmt.Errorf("创建配置目录失败: %v", err)
		}
		defaultConf := Config{}
		defaultData, err := yaml.Marshal(defaultConf)
		if err != nil {
			return fmt.Errorf("生成默认配置模版失败: %v", err)
		}
		if err := os.WriteFile(defaultConfigPath, defaultData, 0644); err != nil {
			return fmt.Errorf("创建配置文件失败: %v", err)
		}
		return nil
	}

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
