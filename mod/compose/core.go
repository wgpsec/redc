package compose

import (
	"encoding/json"
	"fmt"
	"os"
	"red-cloud/mod"
	"sort"
	"strings"

	"red-cloud/mod/gologger"

	"gopkg.in/yaml.v3"
)

// --- 基础结构体定义 (移动到 Core 以便共享) ---

// ComposeConfig 对应 YAML 文件的根结构
type ComposeConfig struct {
	Version   string                 `yaml:"version"`
	Providers map[string]interface{} `yaml:"providers,omitempty"` // 预留：如果以后需要内联 Provider 定义
	Configs   map[string]ConfigItem  `yaml:"configs"`
	Plugins   map[string]ServiceSpec `yaml:"plugins"`
	Services  map[string]ServiceSpec `yaml:"services"`
	Setup     []SetupTask            `yaml:"setup"`
}

// ConfigItem 全局配置项 (configs 块)
type ConfigItem struct {
	File  string                   `yaml:"file,omitempty"`  // 引用本地文件路径
	Rules []map[string]interface{} `yaml:"rules,omitempty"` // 结构化数据 (如安全组规则)
	// 如果有其他类型的配置 (如 JSON 字符串)，也可以在这里扩展
}

// ServiceSpec 服务定义 (services/plugins 块)
type ServiceSpec struct {
	// 基础信息
	Image         string `yaml:"image"`
	ContainerName string `yaml:"container_name,omitempty"` // 自定义容器/实例名

	// 编排与控制
	Provider  interface{} `yaml:"provider,omitempty"` // 支持 string (单云) 或 []string (多云矩阵)
	Profiles  []string    `yaml:"profiles,omitempty"` // 激活环境 (prod, dev, attack)
	DependsOn []string    `yaml:"depends_on,omitempty"`
	Deploy    DeploySpec  `yaml:"deploy,omitempty"` // 部署策略 (Replicas)

	// 变量与配置注入
	Configs     []string `yaml:"configs,omitempty"`     // 格式 ["tf_var=config_key"]
	Environment []string `yaml:"environment,omitempty"` // 格式 ["key=value"]

	// SSH 后置操作
	Volumes   []string `yaml:"volumes,omitempty"`   // 上传: ["local_path:remote_path"]
	Command   string   `yaml:"command,omitempty"`   // 启动命令: "bash /root/init.sh"
	Downloads []string `yaml:"downloads,omitempty"` // 回传: ["remote_path:local_path"]
}

// DeploySpec 部署策略
type DeploySpec struct {
	Replicas int `yaml:"replicas,omitempty"` // 副本数: 用于批量创建 (如 scanner_1, scanner_2)
}

// SetupTask 后置编排任务 (setup 块)
type SetupTask struct {
	Name    string `yaml:"name"`
	Service string `yaml:"service"`         // 目标服务名
	Command string `yaml:"command"`         // 远程执行的命令
	Shell   string `yaml:"shell,omitempty"` // 指定解释器 (默认 bash)
	Outputs string `yaml:"output,omitempty"`
}

// RuntimeService 运行时服务状态
type RuntimeService struct {
	Name       string                 // 最终名称 (如 proxy_aws_1)
	RawName    string                 // YAML 中的原始服务名 (如 proxy)
	Spec       ServiceSpec            // 配置副本
	Outputs    map[string]interface{} // TF Output 缓存
	CaseRef    *mod.Case              // 关联的 Case 实例
	IsDeployed bool                   // 部署状态标记
}

// ComposeOptions 编排选项
type ComposeOptions struct {
	File     string
	Profiles []string
	Project  *mod.RedcProject
}

// ComposeContext 核心上下文，贯穿整个生命周期
type ComposeContext struct {
	RuntimeSvcs   map[string]*RuntimeService // 服务实例 Map
	SortedSvcKeys []string                   // 排序后的 Key (保证遍历顺序一致)
	GlobalConfigs map[string]string          // 解析后的 Configs
	ConfigRaw     ComposeConfig              // 原始 YAML
	LogMgr        *gologger.LogManager       // 日志管理器
	Project       *mod.RedcProject           // 项目引用
}

// --- 核心初始化逻辑 ---

// NewComposeContext 初始化上下文：读取 -> 解析 -> 过滤 -> 裂变
func NewComposeContext(opts ComposeOptions) (*ComposeContext, error) {
	// 1. 读取 YAML
	data, err := os.ReadFile(opts.File)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %v", err)
	}
	var cfg ComposeConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析 YAML 结构失败: %v", err)
	}

	// 2. 初始化组件
	logMgr := gologger.NewLogManager(opts.Project.ProjectPath)
	globalConfigs, err := resolveConfigs(cfg.Configs)
	if err != nil {
		return nil, err
	}

	// 3. 合并 Services 和 Plugins
	allSpecs := cfg.Services
	if allSpecs == nil {
		allSpecs = make(map[string]ServiceSpec)
	}
	for k, v := range cfg.Plugins {
		allSpecs[k] = v
	}

	// 4. 服务过滤与裂变 (Core Logic)
	runtimeSvcs := make(map[string]*RuntimeService)
	for name, spec := range allSpecs {
		// Profile 过滤
		if !checkProfile(spec.Profiles, opts.Profiles) {
			gologger.Debug().Msgf("Skipping service %s (profile not active)", name)
			continue
		}

		// 裂变逻辑
		expandedList := expandService(name, spec)
		for _, svc := range expandedList {
			if _, exists := runtimeSvcs[svc.Name]; exists {
				return nil, fmt.Errorf("生成服务名冲突: %s", svc.Name)
			}
			runtimeSvcs[svc.Name] = svc
		}
	}

	// 5. 生成排序 Key
	var keys []string
	for k := range runtimeSvcs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	return &ComposeContext{
		RuntimeSvcs:   runtimeSvcs,
		SortedSvcKeys: keys, // 后续遍历必须使用这个 slice
		GlobalConfigs: globalConfigs,
		ConfigRaw:     cfg,
		LogMgr:        logMgr,
		Project:       opts.Project,
	}, nil
}

// resolveConfigs 解析配置
func resolveConfigs(raw map[string]ConfigItem) (map[string]string, error) {
	res := make(map[string]string)
	for k, v := range raw {
		if v.File != "" {
			b, err := os.ReadFile(v.File)
			if err != nil {
				return nil, fmt.Errorf("read config file %s error: %v", v.File, err)
			}
			res[k] = strings.TrimSpace(string(b))
		} else {
			b, _ := json.Marshal(v.Rules)
			res[k] = string(b)
		}
	}
	return res, nil
}

// expandService 裂变逻辑，解析服务
func expandService(name string, spec ServiceSpec) []*RuntimeService {
	var providers []string
	switch v := spec.Provider.(type) {
	case string:
		if v != "" {
			providers = []string{v}
		}
	case []interface{}:
		for _, p := range v {
			providers = append(providers, fmt.Sprint(p))
		}
	}
	if len(providers) == 0 {
		providers = []string{"default"}
	}

	replicas := spec.Deploy.Replicas
	if replicas <= 0 {
		replicas = 1
	}

	var res []*RuntimeService
	for _, p := range providers {
		for i := 1; i <= replicas; i++ {
			newName := name
			if p != "default" {
				newName = fmt.Sprintf("%s_%s", newName, p)
			}
			if spec.Deploy.Replicas > 1 {
				newName = fmt.Sprintf("%s_%d", newName, i)
			}

			newSpec := spec
			newSpec.Provider = p
			res = append(res, &RuntimeService{Name: newName, RawName: name, Spec: newSpec})
		}
	}
	return res
}

// checkProfile 检查服务是否应该启动
func checkProfile(svcP, activeP []string) bool {
	// 1. 如果服务本身没有定义 Profile，它属于基础服务，总是启动
	if len(svcP) == 0 {
		return true
	}

	// 2. [修改点] 如果用户没有指定任何 Profile (命令行没传 -p)，
	// 默认视为 "All"，即启动所有带有 Profile 的服务
	if len(activeP) == 0 {
		return true
	}

	// 3. 如果用户显式指定了 Profile (如 -p prod)，则必须匹配才启动
	for _, s := range svcP {
		for _, a := range activeP {
			if s == a {
				return true
			}
		}
	}

	// 指定了 Profile 但没匹配上，不启动
	return false
}
