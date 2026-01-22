package mod

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"red-cloud/mod/gologger"
	"red-cloud/utils"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/hashicorp/terraform-exec/tfexec"
)

type CaseState string

const (
	StateRunning CaseState = "running"
	StateStopped CaseState = "stopped"
	StateError   CaseState = "error"
	StateCreated CaseState = "created"
	StatePending CaseState = "pending"
)

func RandomName(s string) string {
	var firstName = []string{
		"red", "blue", "yellow", "brown", "purple", "anger", "lazy", "shy", "huge", "rare",
		"fast", "stupid", "sluggish", "boring", "rigid", "rigorous", "clever", "dexterity",
		"white", "black", "dark", "idiot", "shiny", "friendly", "integrity", "happy", "sad",
		"lively", "lonely", "ugly", "leisurely", "calm", "young", "tenacious", "admiring",
		"agitated", "boring", "clever", "compassionate", "condescending", "cranky", "desperate",
		"distracted", "ecstatic", "focused", "goofy", "hungry", "jolly", "modest", "naughty", "nostalgic",
		"pensive", "recursing", "sleepy", "thirsty", "xenodochial", "zen", "niubi",
	}
	var lastName = []string{
		"pig", "cow", "sheep", "mouse", "dragon", "serpent", "tiger", "fox", "frog", "chicken",
		"fish", "shrimp", "hippocampus", "helicopter", "crab", "dolphin", "whale", "chinchilla",
		"bunny", "mole", "rabbit", "horse", "monkey", "dog", "shark", "panda", "bear", "lion",
		"rhino", "leopard", "giraffe", "deer", "wolf", "parrot", "camel", "antelope", "turtle",
		"zebra", "hacker",
	}
	rand.Seed(time.Now().UnixNano())
begin:
	first := firstName[rand.Intn(len(firstName)-1)]
	last := lastName[rand.Intn(len(lastName)-1)]
	// NO NO NO ~
	if (first == "stupid" || first == "goofy") && last == "wolf" {
		goto begin
	}
	s = truncateString(s, 10)
	return fmt.Sprintf("%s_%s_%s", first, last, s)
}

// GenerateCaseID 生成 ID (64字符 hex string)
// 本质是 32 字节 (256 bit) 的随机数
func GenerateCaseID() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		// 极端情况下随机数生成失败，回退到时间戳+简单的随机，或者直接 panic
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// CaseScene 场景参数判断
func CaseScene(t string, m map[string]string) ([]string, error) {
	var par []string
	scene := filepath.Base(t)
	switch scene {
	case "cs-49", "c2-new", "snowc2":
		par = RVar(
			fmt.Sprintf("node_count=%d", Node),
			fmt.Sprintf("domain=%s", Domain),
		)
	case "proxy", "aws-proxy", "aliyun-proxy", "asm":
		par = RVar(fmt.Sprintf("node_count=%d", Node))
	case "dnslog", "xraydnslog", "interactsh":
		if Domain == "360.com" {
			return par, fmt.Errorf("创建 dnslog 时,域名不可为默认值")
		}
		tfDomain := Domain
		if !strings.HasPrefix(strings.ToLower(tfDomain), "a.") {
			tfDomain = "a." + tfDomain
		}
		par = RVar(fmt.Sprintf("domain=%s", tfDomain))
	case "pss5", "frp", "frp-loki", "nps":
		par = []string{fmt.Sprintf("base64_command=%s", Base64Command)}
	case "asm-node":
		par = RVar(
			fmt.Sprintf("node_count=%d", Node),
			fmt.Sprintf("domain2=%s", Domain2),
			fmt.Sprintf("domain=%s", Domain),
		)
	}
	// 增加自定义参数
	for k, v := range m {
		if scene == "dnslog" || scene == "xraydnslog" || scene == "interactsh" {
			if strings.EqualFold(k, "domain") {
				if !strings.HasPrefix(strings.ToLower(v), "a.") {
					v = "a." + v
				}
			}
		}
		par = append(par, fmt.Sprintf("%s=%s", k, v))
	}
	return par, nil
}

func (p *RedcProject) CaseCreate(CaseName string, User string, Name string, vars map[string]string) (*Case, error) {
	// 创建新的 case 目录,这里不需要检测是否存在,因为名称是采用nanoID
	gologger.Info().Msgf("正在创建场景 「%s」", CaseName)
	uid := GenerateCaseID()

	// 从模版文件夹复制模版
	tpPath, err := GetTemplatePath(CaseName)
	if err != nil {
		return nil, fmt.Errorf("获取模版错误！%s", err.Error())
	}
	casePath := filepath.Join(p.ProjectPath, uid)

	// 复制 tf文件
	gologger.Debug().Msgf("复制模版中 %s", uid)
	if err := utils.Dir(tpPath, casePath); err != nil {
		return nil, fmt.Errorf("复制模版出错！\n%v", err)
	}

	// 在次 init,防止万一
	if err := TfInit2(casePath); err != nil {
		gologger.Error().Msgf("二次初始化失败！%s", err.Error())
		return nil, err
	}
	// 初始化结构参数
	par, err := CaseScene(CaseName, vars)
	if err != nil {
		gologger.Error().Msgf("场景参数校验失败！%s", err.Error())
		return nil, err
	}

	// 初始化实例名称
	if Name == "" {
		Name = RandomName(CaseName)
	}

	// 读取模板元数据，获取模块名（如果有）
	meta, _ := readTemplateMeta(tpPath)
	moduleName := ""
	if meta != nil {
		moduleName = meta.RedcModule
	}

	// 初始化实例
	c := &Case{
		CreateTime: time.Now().Format("2006-01-02 15:04:05"),
		Id:         uid,
		Name:       Name,
		Operator:   User,
		Path:       casePath,
		Type:       CaseName,
		Module:     moduleName,
		Parameter:  par,
		State:      StatePending,
		project:    p, // 绑定项目引用
	}

	// 构建场景
	if err := c.TfPlan(); err != nil {
		gologger.Debug().Msgf("场景创建校验失败！%s", err.Error())
		c.Remove()
		return nil, err
	}
	gologger.Info().Msgf("场景创建成功！%s", uid)
	// 确认场景创建无误后,才会写入到配置文件中
	err = p.AddCase(c)
	err = p.SaveProject()
	if err != nil {
		gologger.Error().Msgf("项目配置保存失败！")
		return nil, err
	}
	RedcLog("创建成功 " + p.ProjectPath + "/" + uid + " " + CaseName)
	return c, nil
}

func (c *Case) TfApply() error {
	var err error
	gologger.Info().Msgf("正在启动场景：%s(%s)", c.Name, c.GetId())
	if c.State == StateRunning {
		return fmt.Errorf("场景正在运行中！")
	}
	if err = TfApply(c.Path, c.Parameter...); err != nil {
		c.StatusChange(StateError)
		// 启动失败立即销毁
		if err := c.TfDestroy(); err != nil {
			return err
		}
		return err
	}
	c.StatusChange(StateRunning)
	output, err := c.TfOutput()
	if err != nil {
		return err
	}
	for s, meta := range output {
		fmt.Println(s, string(meta.Value))
	}
	if err := c.runModuleHook(); err != nil {
		return err
	}
	return nil
}
func (c *Case) GetInstanceInfo(id string) (string, error) {
	// 1. 检查 Output 是否为 nil 防止空指针
	if c.output == nil {
		_, err := c.TfOutput()
		if err != nil {
			return "", fmt.Errorf("output 数据未初始化")
		}

	}

	// 2. 检查 key 是否存在
	val, ok := c.output[id]
	if !ok {
		return "", fmt.Errorf("未找到 ID 为 %s 的信息", id)
	}
	var str string
	// 反序列化会将 JSON 字符串解析为 Go 的 string，自动去除引号和处理转义
	if err := json.Unmarshal(val.Value, &str); err != nil {
		return "", err
	}

	// 3. 安全地获取 Value 并转换为字符串
	// 这里假设 val.Value 是 string 类型，或者可以使用 fmt.Sprint 确保兼容性
	return str, nil
}

// runModuleHook 在场景启动成功后执行模板声明的模块钩子
func (c *Case) runModuleHook() error {
	// 如果未记录模块名，尝试从 case.json 读取一次
	if c.Module == "" {
		if meta, err := readTemplateMeta(c.Path); err == nil {
			c.Module = meta.RedcModule
			if c.project != nil {
				_ = c.project.SaveProject()
			}
		}
	}
	if c.Module == "" {
		return nil
	}

	modules := strings.Split(c.Module, ",")
	for _, m := range modules {
		name := strings.TrimSpace(m)
		if name == "" {
			continue
		}
		handler, ok := moduleRegistry[name]
		if !ok {
			gologger.Warning().Msgf("未找到模块处理函数: %s", name)
			continue
		}

		gologger.Info().Msgf("执行模块: %s", name)
		if err := handler(c); err != nil {
			gologger.Error().Msgf("模块执行失败(%s): %v", name, err)
			return err
		}
	}
	return nil
}

func (c *Case) TfOutput() (map[string]tfexec.OutputMeta, error) {
	// 输出 output 信息
	o, err := TfOutput(c.Path)
	if err != nil {
		gologger.Error().Msgf("获取 Output 信息失败: %v", err)
		return nil, err
	}
	c.output = o
	return o, nil
}

func (c *Case) TfPlan() error {
	gologger.Info().Msgf("正在构建场景「%s(%s)」...", c.Name, c.GetId())
	if err := TfPlan(c.Path, c.Parameter...); err != nil {
		return err
	}
	c.StatusChange(StateCreated)
	return nil
}

func (c *Case) StatusChange(s CaseState) {
	c.State = s
	c.StateTime = time.Now().Format("2006-01-02 15:04:05")

	if c.project != nil {
		if err := c.project.UpdateCaseState(c.Id, s, c.StateTime); err != nil {
			gologger.Error().Msgf("状态保存到配置文件失败: %s \n", err)
		}
	}
}

func (c *Case) TfDestroy() error {
	gologger.Info().Msgf("正在销毁场景「%s(%s)」...", c.Name, c.GetId())
	err := TfDestroy(c.Path, c.Parameter)
	if err != nil {
		gologger.Error().Msgf("场景销毁失败！%s", err.Error())
		return err
	}
	c.StatusChange(StateStopped)
	return nil
}
func (c *Case) Remove() error {
	if c.State == StateRunning {
		return fmt.Errorf("场景正在运行中，请先停止场景后删除！")
	}
	err := os.RemoveAll(c.Path)
	if err != nil {
		return fmt.Errorf("删除场景文件失败！%s", err.Error())
	}
	if c.project != nil {
		err = c.project.HandleCase(c)
	}
	gologger.Info().Msgf("场景删除成功")
	return err
}

// Stop 停止场景
func (c *Case) Stop() error {
	if c.State != StateRunning {
		gologger.Warning().Msgf("该场景提示未运行中,不过还是为您销毁")
	}
	err := c.TfDestroy()
	if err != nil {
		return err
	}
	return nil
}

// Kill 强制销毁场景
func (c *Case) Kill() error {
	// 在次 init,防止万一
	dirs := utils.ChechDirMain(c.Path)
	for _, v := range dirs {
		err := utils.CheckFileName(v, "tf")
		if err {
			TfInit2(v)
		}
	}
	// 销毁场景
	if err := c.Stop(); err != nil {
		gologger.Error().Msgf("场景销毁失败！%s", err.Error())
		return err
	}
	return nil
}

// Change 重建场景
func (c *Case) Change(cc ChangeCommand) error {
	if cc.IsRemove {
		// 销毁场景，不删除项目
		gologger.Info().Msgf("正在销毁场景 「%s」 %s\n", c.Name, c.Id)
		if err := c.TfDestroy(); err != nil {
			return err
		}
	}
	// TODO 更改弹性公网IP等操作
	// 重新赋值
	if par, err := CaseScene(c.Type, cc.Pars); err != nil {
		c.Parameter = par
	}
	// 展示更改信息
	if err := c.TfPlan(); err != nil {
		return err
	}
	// 重建场景
	if err := c.TfApply(); err != nil {
		return err
	}
	return nil
}

func (c *Case) Status() error {
	gologger.Info().Msgf("Case「%s」当前 %s 状态", c.Name, c.State)
	state, err := TfStatus(c.Path)
	if err != nil {
		return err
	}
	// 1. 优先打印 Outputs (通常是用户最关心的，如 IP 地址)
	if len(state.Values.Outputs) > 0 {
		fmt.Println("\n--- Outputs ---")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		for key, output := range state.Values.Outputs {
			// output.Value 是 interface{}，可能是 string, map, list 等
			fmt.Fprintf(w, "%s:\t%v\n", key, output.Value)
		}
		w.Flush()
	} else {
		fmt.Println("\n--- No Outputs detected ---")
	}

	// 2. 打印资源概览 (只打印 Root Module 的资源)
	if state.Values.RootModule != nil && len(state.Values.RootModule.Resources) > 0 {
		fmt.Println("\n--- Resources ---")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "TYPE\tADDRESS\tNAME")
		for _, res := range state.Values.RootModule.Resources {
			fmt.Fprintf(w, "%s\t%s\t%s\n", res.Type, res.Address, res.Name)
		}
		w.Flush()
	}
	return nil
}
func (c *Case) GetId() string {
	if len(c.Id) > 12 {
		return c.Id[:12]
	}
	return c.Id
}

// humanDuration 计算时间差并返回 Docker 风格的字符串
// 例如: "Up 2 hours", "Up 5 minutes"
func humanDuration(t time.Time) string {
	duration := time.Since(t)
	seconds := int(duration.Seconds())

	switch {
	case seconds < 60:
		return fmt.Sprintf("%d seconds", seconds)
	case seconds < 3600:
		return fmt.Sprintf("%d minutes", seconds/60)
	case seconds < 86400:
		return fmt.Sprintf("%d hours", seconds/3600)
	default:
		return fmt.Sprintf("%d days", seconds/86400)
	}
}

// parseTime 将字符串时间转为 time.Time
func parseTime(timeStr string) time.Time {
	// 对应你代码中的 time.Now().Format("2006-01-02 15:04:05")
	layout := "2006-01-02 15:04:05"
	t, err := time.ParseInLocation(layout, timeStr, time.Local)
	if err != nil {
		return time.Now() // 解析失败则返回当前时间，避免 panic
	}
	return t
}

// truncateString 安全截断函数
func truncateString(s string, length int) string {
	s = strings.ReplaceAll(s, "/", "_")
	// 1. 将字符串转为 rune 切片（处理多字节字符的关键）
	runes := []rune(s)

	// 2. 判断字符数量是否超过限制
	if len(runes) > length {
		// 3. 截取前 length 个字符并转回 string
		return string(runes[:length])
	}

	// 4. 如果没超过，原样返回
	return s
}
