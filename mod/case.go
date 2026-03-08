package mod

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"red-cloud/i18n"
	"red-cloud/mod/gologger"
	"red-cloud/utils"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/hashicorp/terraform-exec/tfexec"
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
			fmt.Sprintf("node=%d", Node),
			fmt.Sprintf("domain=%s", Domain),
		)
	case "proxy", "aws-proxy", "aliyun-proxy", "asm":
		par = RVar(fmt.Sprintf("node=%d", Node))
	case "dnslog", "xraydnslog", "interactsh":
		if Domain == "360.com" {
			return par, fmt.Errorf("%s", i18n.T("case_create_dnslog_domain_invalid"))
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
			fmt.Sprintf("node=%d", Node),
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

func providerFromTemplateName(templateName string) string {
	parts := strings.Split(templateName, "/")
	if len(parts) > 1 {
		return strings.ToLower(strings.TrimSpace(parts[0]))
	}
	return ""
}

func ensureProviderVars(templateName string, vars map[string]string) map[string]string {
	if vars == nil {
		vars = map[string]string{}
	}
	provider := providerFromTemplateName(templateName)
	
	conf, _, err := ReadConfig(ActiveConfigPath)
	if err != nil || conf == nil {
		return vars
	}
	
	// 处理阿里云
	// 阿里云使用环境变量 ALICLOUD_ACCESS_KEY 和 ALICLOUD_SECRET_KEY，不需要通过 -var 传递
	if provider == "aliyun" || provider == "alicloud" {
		// 如果模板需要 instance_password 且用户未提供，则自动生成
		if vars["instance_password"] == "" {
			vars["instance_password"] = generateInstancePassword()
			gologger.Info().Msg(i18n.T("case_auto_password_aliyun"))
		}
	}
	
	// 处理腾讯云
	if provider == "tencent" || provider == "tencentcloud" {
		if (vars["tencentcloud_secret_id"] == "" || isPlaceholderSecret(vars["tencentcloud_secret_id"])) && conf.Providers.Tencentcloud.SecretId != "" {
			vars["tencentcloud_secret_id"] = conf.Providers.Tencentcloud.SecretId
		}
		if (vars["tencentcloud_secret_key"] == "" || isPlaceholderSecret(vars["tencentcloud_secret_key"])) && conf.Providers.Tencentcloud.SecretKey != "" {
			vars["tencentcloud_secret_key"] = conf.Providers.Tencentcloud.SecretKey
		}
		// 如果模板需要 instance_password 且用户未提供，则自动生成
		if vars["instance_password"] == "" {
			vars["instance_password"] = generateInstancePassword()
			gologger.Info().Msg(i18n.T("case_auto_password_tencent"))
		}
	}

	// 处理华为云
	if provider == "huawei" || provider == "huaweicloud" {
		if (vars["huaweicloud_access_key"] == "" || isPlaceholderSecret(vars["huaweicloud_access_key"])) && conf.Providers.Huaweicloud.AccessKey != "" {
			vars["huaweicloud_access_key"] = conf.Providers.Huaweicloud.AccessKey
		}
		if (vars["huaweicloud_secret_key"] == "" || isPlaceholderSecret(vars["huaweicloud_secret_key"])) && conf.Providers.Huaweicloud.SecretKey != "" {
			vars["huaweicloud_secret_key"] = conf.Providers.Huaweicloud.SecretKey
		}
		// 如果模板需要 instance_password 且用户未提供，则自动生成
		if vars["instance_password"] == "" {
			vars["instance_password"] = generateInstancePassword()
			gologger.Info().Msg(i18n.T("case_auto_password_huawei"))
		}
	}

	// 处理火山引擎
	// 火山引擎使用环境变量 VOLCENGINE_ACCESS_KEY 和 VOLCENGINE_SECRET_KEY，不需要通过 -var 传递
	if provider == "volcengine" {
		// 如果模板需要 instance_password 且用户未提供，则自动生成
		if vars["instance_password"] == "" {
			vars["instance_password"] = generateInstancePassword()
			gologger.Info().Msg(i18n.T("case_auto_password_volc"))
		}
	}

	// 处理 UCloud
	// UCloud 使用环境变量 UCLOUD_PUBLIC_KEY 和 UCLOUD_PRIVATE_KEY，不需要通过 -var 传递
	if provider == "ucloud" {
		// 如果模板需要 instance_password 且用户未提供，则自动生成
		if vars["instance_password"] == "" {
			vars["instance_password"] = generateInstancePassword()
			gologger.Info().Msg(i18n.T("case_auto_password_ucloud"))
		}
	}

	// 处理天翼云 CTYun
	if provider == "ctyun" {
		// 如果模板需要 access_key 且用户未提供，则从配置中获取
		if (vars["access_key"] == "" || isPlaceholderSecret(vars["access_key"])) && conf.Providers.Ctyun.AccessKey != "" {
			vars["access_key"] = conf.Providers.Ctyun.AccessKey
		}
		if (vars["secret_key"] == "" || isPlaceholderSecret(vars["secret_key"])) && conf.Providers.Ctyun.SecretKey != "" {
			vars["secret_key"] = conf.Providers.Ctyun.SecretKey
		}
		// 如果模板需要 instance_password 且用户未提供，则自动生成
		if vars["instance_password"] == "" {
			vars["instance_password"] = generateInstancePassword()
			gologger.Info().Msg(i18n.T("case_auto_password_ctyun"))
		}
	}

	return vars
}

func isPlaceholderSecret(value string) bool {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return false
	}
	if strings.HasPrefix(trimmed, "****") {
		return true
	}
	for _, r := range trimmed {
		if r != 'x' && r != 'X' && r != '*' {
			return false
		}
	}
	return true
}

// generateInstancePassword 生成符合云厂商要求的实例密码
// 华为云密码规则：8~26个字符，至少包含大写、小写、数字、特殊字符中的三种
// 允许的特殊字符：!@%-_=+[]:./?
// 生成格式：R{10位hex}_+{12位hex}，长度25位，包含四种字符类型
func generateInstancePassword() string {
	// 使用 hex 编码生成随机字符串（只包含 0-9 和 a-f）
	seed := GenerateCaseID()
	// 移除连字符并取前22个字符
	seed = strings.ReplaceAll(seed, "-", "")
	if len(seed) > 22 {
		seed = seed[:22]
	}
	// 构造密码：确保包含大写、小写、数字和特殊字符
	// 格式：R(大写) + 10位hex(小写+数字) + _+(特殊字符) + 12位hex(小写+数字)
	password := fmt.Sprintf("R%s_+%s", seed[:10], seed[10:])
	return password
}

func ensureProviderParams(templateName string, params []string) []string {
	paramMap := map[string]string{}
	order := make([]string, 0, len(params))
	for _, param := range params {
		key, val, ok := strings.Cut(param, "=")
		if !ok || key == "" {
			continue
		}
		if _, exists := paramMap[key]; !exists {
			order = append(order, key)
		}
		paramMap[key] = val
	}
	paramMap = ensureProviderVars(templateName, paramMap)
	merged := make([]string, 0, len(paramMap))
	seen := map[string]bool{}
	for _, key := range order {
		if val, ok := paramMap[key]; ok {
			merged = append(merged, fmt.Sprintf("%s=%s", key, val))
			seen[key] = true
		}
	}
	for key, val := range paramMap {
		if !seen[key] {
			merged = append(merged, fmt.Sprintf("%s=%s", key, val))
		}
	}
	return merged
}

func (p *RedcProject) CaseCreate(CaseName string, User string, Name string, vars map[string]string) (*Case, error) {
	// 创建新的 case 目录,这里不需要检测是否存在,因为名称是采用nanoID
	gologger.Info().Msgf("%s", i18n.Tf("case_creating", CaseName))
	uid := GenerateCaseID()
	vars = ensureProviderVars(CaseName, vars)

	// 从模版文件夹复制模版
	tpPath, err := GetTemplatePath(CaseName)
	if err != nil {
		return nil, fmt.Errorf("%s", i18n.Tf("case_get_template_error", err.Error()))
	}
	casePath := filepath.Join(p.ProjectPath, uid)

	// 复制 tf文件
	gologger.Debug().Msgf("%s", i18n.Tf("case_copying_template", uid))
	if err := utils.Dir(tpPath, casePath); err != nil {
		return nil, fmt.Errorf("%s", i18n.Tf("case_copy_template_error", err))
	}

	// 在次 init,防止万一
	if err := TfInit2(casePath); err != nil {
		gologger.Error().Msgf("%s", i18n.Tf("case_second_init_failed", err.Error()))
		return nil, err
	}
	// 初始化结构参数
	par, err := CaseScene(CaseName, vars)
	if err != nil {
		gologger.Error().Msgf("%s", i18n.Tf("case_validate_params_failed", err.Error()))
		return nil, err
	}

	// 初始化实例名称
	if Name == "" {
		Name = RandomName(CaseName)
	}

	// 将实例名称添加到变量中，传递给 Terraform
	if vars == nil {
		vars = make(map[string]string)
	}
	vars["instance_name"] = Name

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
		ProjectID:  p.ProjectName,
		State:      StatePending,
	}
	// 绑定 project 参数
	c.bindHandlers()

	// 构建场景
	if err := c.TfPlan(); err != nil {
		gologger.Error().Msgf("%s", i18n.Tf("case_validate_create_failed", err.Error()))
		c.Remove()
		return nil, err
	}
	gologger.Info().Msgf("%s", i18n.Tf("case_create_success", uid))
	// 确认场景创建无误后,才会写入到配置文件中
	err = p.AddCase(c)
	if err != nil {
		gologger.Error().Msg(i18n.T("case_save_config_failed"))
		return nil, err
	}
	RedcLog("创建成功 " + p.ProjectPath + "/" + uid + " " + CaseName)
	return c, nil
}

func (c *Case) TfApply() error {
	var err error
	gologger.Info().Msgf("%s", i18n.Tf("case_starting", c.Name, c.GetId()))
	if c.State == StateRunning {
		return fmt.Errorf("%s", i18n.T("case_scene_running"))
	}
	
	// 设置为正在启动状态
	c.StatusChange(StateStarting)
	
	// 重新生成 plan 以确保与当前 state 一致
	gologger.Info().Msg(i18n.T("case_plan_refreshing"))
	if err = TfPlan(c.Path, c.Parameter...); err != nil {
		c.StatusChange(StateError)
		return fmt.Errorf("%s", i18n.Tf("case_plan_refresh_failed", err))
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
			return "", fmt.Errorf("%s", i18n.T("case_output_not_initialized"))
		}

	}

	// 2. 检查 key 是否存在
	val, ok := c.output[id]
	if !ok {
		return "", fmt.Errorf("%s", i18n.Tf("case_output_not_found", id))
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

// GetInstanceInfoList 获取 output 值，支持单个字符串和字符串数组两种格式
func (c *Case) GetInstanceInfoList(id string) ([]string, error) {
	if c.output == nil {
		_, err := c.TfOutput()
		if err != nil {
			return nil, fmt.Errorf("%s", i18n.T("case_output_not_initialized"))
		}
	}

	val, ok := c.output[id]
	if !ok {
		return nil, fmt.Errorf("%s", i18n.Tf("case_output_not_found", id))
	}

	// 先尝试解析为单个字符串
	var str string
	if err := json.Unmarshal(val.Value, &str); err == nil && str != "" {
		return []string{str}, nil
	}

	// 再尝试解析为字符串数组
	var arr []string
	if err := json.Unmarshal(val.Value, &arr); err == nil && len(arr) > 0 {
		return arr, nil
	}

	return nil, fmt.Errorf("无法解析 output '%s' 为字符串或字符串数组", id)
}

// runModuleHook 在场景启动成功后执行模板声明的模块钩子
func (c *Case) runModuleHook() error {
	// 如果未记录模块名，尝试从 case.json 读取一次
	if c.Module == "" {
		if meta, err := readTemplateMeta(c.Path); err == nil {
			c.Module = meta.RedcModule
			if c.saveHandler != nil {
				_ = c.saveHandler()
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
			gologger.Warning().Msgf("%s", i18n.Tf("case_module_not_found", name))
			continue
		}

		gologger.Info().Msgf("%s", i18n.Tf("case_module_executing", name))
		if err := handler(c); err != nil {
			gologger.Error().Msgf("%s", i18n.Tf("case_module_failed", name, err))
			return err
		}
	}
	return nil
}

func (c *Case) TfOutput() (map[string]tfexec.OutputMeta, error) {
	// 输出 output 信息
	o, err := TfOutput(c.Path)
	if err != nil {
		gologger.Error().Msgf("%s", i18n.Tf("case_get_output_failed", err))
		return nil, err
	}
	c.output = o
	return o, nil
}

// bindHandlers 绑定项目方法
func (c *Case) bindHandlers() {
	c.saveHandler = func() error {
		// 保存单个 Case，不再保存整个 Project
		return c.DBSave()
	}
	c.removeHandle = func() error {
		return c.DBRemove()
	}
}

func (c *Case) TfPlan() error {
	gologger.Info().Msgf("%s", i18n.Tf("case_building", c.Name, c.GetId()))
	c.Parameter = ensureProviderParams(c.Type, c.Parameter)
	if err := TfPlan(c.Path, c.Parameter...); err != nil {
		return err
	}
	c.StatusChange(StateCreated)
	return nil
}

func (c *Case) StatusChange(s string) {
	c.State = s
	// Use RFC3339 format to include timezone information
	c.StateTime = time.Now().Format(time.RFC3339)
	if c.saveHandler != nil {
		if err := c.saveHandler(); err != nil {
			gologger.Error().Msgf("%s", i18n.Tf("case_save_state_failed", err))
		}
	}
}

func (c *Case) TfDestroy() error {
	gologger.Info().Msgf("%s", i18n.Tf("case_destroying", c.Name, c.GetId()))
	
	// 设置为正在停止状态
	c.StatusChange(StateStopping)
	
	err := TfDestroy(c.Path, c.Parameter)
	if err != nil {
		gologger.Error().Msgf("%s", i18n.Tf("case_destroy_failed", err.Error()))
		c.StatusChange(StateError)
		return err
	}
	c.StatusChange(StateStopped)
	return nil
}
func (c *Case) Remove() error {
	if c.State == StateRunning {
		return fmt.Errorf("%s", i18n.T("case_delete_running"))
	}
	
	// 设置为正在删除状态
	c.StatusChange(StateRemoving)
	
	err := os.RemoveAll(c.Path)
	if err != nil {
		c.StatusChange(StateError)
		return fmt.Errorf("%s", i18n.Tf("case_delete_file_failed", err.Error()))
	}
	if err = c.removeHandle(); err != nil {
		c.StatusChange(StateError)
		return fmt.Errorf("%s", i18n.Tf("case_delete_db_failed", err))
	}
	gologger.Info().Msg(i18n.T("case_delete_success"))
	return nil
}

// Stop 停止场景
func (c *Case) Stop() error {
	if c.State != StateRunning {
		gologger.Warning().Msg(i18n.T("case_destroy_warning"))
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
		gologger.Error().Msgf("%s", i18n.Tf("case_destroy_failed", err.Error()))
		return err
	}
	return nil
}

// Change 重建场景
func (c *Case) Change(cc ChangeCommand) error {
	if cc.IsRemove {
		// 销毁场景，不删除项目
		gologger.Info().Msgf("%s", i18n.Tf("case_change_destroying", c.Name, c.Id))
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
	gologger.Info().Msgf("%s", i18n.Tf("case_status_format", c.Name, c.State))
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
