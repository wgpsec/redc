package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	redc "red-cloud/mod"
	"red-cloud/mod/gologger"
	"red-cloud/mod/mcp"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx         context.Context
	project     *redc.RedcProject
	mu          sync.Mutex
	initError   string
	logMgr      *gologger.LogManager
	mcpManager  *mcp.MCPServerManager
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Set default values (same as CLI defaults)
	if redc.Project == "" {
		redc.Project = "default"
	}
	if redc.U == "" {
		redc.U = "system" // Use "system" to match CLI default and bypass permission check
	}

	// Initialize config using same path detection as CLI
	if err := redc.LoadConfig(""); err != nil {
		a.initError = fmt.Sprintf("配置加载失败: %v", err)
		runtime.LogErrorf(ctx, a.initError)
		return
	}

	runtime.LogInfof(ctx, "配置加载成功 - RedcPath: %s, ProjectPath: %s, TemplateDir: %s", 
		redc.RedcPath, redc.ProjectPath, redc.TemplateDir)

	// Load default project
	if p, err := redc.ProjectParse(redc.Project, redc.U); err == nil {
		a.project = p
		a.logMgr = gologger.NewLogManager(p.ProjectPath)
		runtime.LogInfof(ctx, "项目加载成功: %s", a.project.ProjectName)
	} else {
		a.initError = fmt.Sprintf("项目加载失败: %v", err)
		runtime.LogErrorf(ctx, a.initError)
	}
}

// emitLog sends a log message to the frontend and writes to file
func (a *App) emitLog(message string) {
	runtime.EventsEmit(a.ctx, "log", message)
	// Also write to GUI log file
	if a.logMgr != nil {
		if logger, err := a.logMgr.NewServiceLogger("gui"); err == nil {
			logger.Write([]byte(message + "\n"))
			logger.Close()
		}
	}
}

// emitRefresh notifies the frontend to refresh data
func (a *App) emitRefresh() {
	runtime.EventsEmit(a.ctx, "refresh", nil)
}

// createLogWriter creates an io.Writer that emits logs to the frontend and writes to file
func (a *App) createLogWriter(prefix string) io.Writer {
	eventWriter := gologger.NewEventWriter(a.emitLog, prefix)
	// If logMgr is available, create a multi-writer that also writes to file
	if a.logMgr != nil {
		if fileLogger, err := a.logMgr.NewServiceLogger(prefix); err == nil {
			return io.MultiWriter(eventWriter, fileLogger)
		}
	}
	return eventWriter
}

// CaseInfo represents case information for frontend display
type CaseInfo struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	State      string `json:"state"`
	StateTime  string `json:"stateTime"`
	CreateTime string `json:"createTime"`
	Operator   string `json:"operator"`
}

// ConfigInfo represents the configuration for frontend display
type ConfigInfo struct {
	RedcPath    string `json:"redcPath"`
	ProjectPath string `json:"projectPath"`
	LogPath     string `json:"logPath"`
	HttpProxy   string `json:"httpProxy"`
	HttpsProxy  string `json:"httpsProxy"`
	NoProxy     string `json:"noProxy"`
}

// ProviderCredential represents a single provider's credentials (masked for display)
type ProviderCredential struct {
	Name       string            `json:"name"`
	Fields     map[string]string `json:"fields"`      // field name -> masked value
	HasSecrets map[string]bool   `json:"hasSecrets"`  // field name -> has value
}

// ProvidersConfigInfo represents all providers' credentials
type ProvidersConfigInfo struct {
	ConfigPath  string               `json:"configPath"`
	Providers   []ProviderCredential `json:"providers"`
}

// GetConfig returns current configuration
func (a *App) GetConfig() ConfigInfo {
	logPath := ""
	if a.logMgr != nil {
		logPath = a.logMgr.BaseDir
	}
	return ConfigInfo{
		RedcPath:    redc.RedcPath,
		ProjectPath: redc.ProjectPath,
		LogPath:     logPath,
		HttpProxy:   os.Getenv("HTTP_PROXY"),
		HttpsProxy:  os.Getenv("HTTPS_PROXY"),
		NoProxy:     os.Getenv("NO_PROXY"),
	}
}

// SaveProxyConfig saves proxy configuration to environment variables
func (a *App) SaveProxyConfig(httpProxy, httpsProxy, noProxy string) error {
	// Set environment variables for current process
	if httpProxy != "" {
		os.Setenv("HTTP_PROXY", httpProxy)
		os.Setenv("http_proxy", httpProxy)
	} else {
		os.Unsetenv("HTTP_PROXY")
		os.Unsetenv("http_proxy")
	}

	if httpsProxy != "" {
		os.Setenv("HTTPS_PROXY", httpsProxy)
		os.Setenv("https_proxy", httpsProxy)
	} else {
		os.Unsetenv("HTTPS_PROXY")
		os.Unsetenv("https_proxy")
	}

	if noProxy != "" {
		os.Setenv("NO_PROXY", noProxy)
		os.Setenv("no_proxy", noProxy)
	} else {
		os.Unsetenv("NO_PROXY")
		os.Unsetenv("no_proxy")
	}

	a.emitLog(fmt.Sprintf("代理配置已更新 - HTTP: %s, HTTPS: %s, NO_PROXY: %s", httpProxy, httpsProxy, noProxy))
	return nil
}

// maskValue returns masked value for display (shows last 4 chars if length > 8)
func maskValue(value string) string {
	if value == "" {
		return ""
	}
	if len(value) <= 4 {
		return "****"
	}
	return "****" + value[len(value)-4:]
}

// GetProvidersConfig returns providers configuration with masked secrets
func (a *App) GetProvidersConfig(customPath string) (ProvidersConfigInfo, error) {
	conf, configPath, err := redc.ReadConfig(customPath)
	if err != nil {
		return ProvidersConfigInfo{}, err
	}

	providers := []ProviderCredential{
		{
			Name: "AWS",
			Fields: map[string]string{
				"accessKey": maskValue(conf.Providers.Aws.AccessKey),
				"secretKey": maskValue(conf.Providers.Aws.SecretKey),
				"region":    conf.Providers.Aws.Region,
			},
			HasSecrets: map[string]bool{
				"accessKey": conf.Providers.Aws.AccessKey != "",
				"secretKey": conf.Providers.Aws.SecretKey != "",
			},
		},
		{
			Name: "阿里云",
			Fields: map[string]string{
				"accessKey": maskValue(conf.Providers.Alicloud.AccessKey),
				"secretKey": maskValue(conf.Providers.Alicloud.SecretKey),
				"region":    conf.Providers.Alicloud.Region,
			},
			HasSecrets: map[string]bool{
				"accessKey": conf.Providers.Alicloud.AccessKey != "",
				"secretKey": conf.Providers.Alicloud.SecretKey != "",
			},
		},
		{
			Name: "腾讯云",
			Fields: map[string]string{
				"secretId":  maskValue(conf.Providers.Tencentcloud.SecretId),
				"secretKey": maskValue(conf.Providers.Tencentcloud.SecretKey),
				"region":    conf.Providers.Tencentcloud.Region,
			},
			HasSecrets: map[string]bool{
				"secretId":  conf.Providers.Tencentcloud.SecretId != "",
				"secretKey": conf.Providers.Tencentcloud.SecretKey != "",
			},
		},
		{
			Name: "火山引擎",
			Fields: map[string]string{
				"accessKey": maskValue(conf.Providers.Volcengine.AccessKey),
				"secretKey": maskValue(conf.Providers.Volcengine.SecretKey),
				"region":    conf.Providers.Volcengine.Region,
			},
			HasSecrets: map[string]bool{
				"accessKey": conf.Providers.Volcengine.AccessKey != "",
				"secretKey": conf.Providers.Volcengine.SecretKey != "",
			},
		},
		{
			Name: "华为云",
			Fields: map[string]string{
				"accessKey": maskValue(conf.Providers.Huaweicloud.AccessKey),
				"secretKey": maskValue(conf.Providers.Huaweicloud.SecretKey),
				"region":    conf.Providers.Huaweicloud.Region,
			},
			HasSecrets: map[string]bool{
				"accessKey": conf.Providers.Huaweicloud.AccessKey != "",
				"secretKey": conf.Providers.Huaweicloud.SecretKey != "",
			},
		},
		{
			Name: "Google Cloud",
			Fields: map[string]string{
				"credentials": maskValue(conf.Providers.Google.Credentials),
				"project":     conf.Providers.Google.Project,
				"region":      conf.Providers.Google.Region,
			},
			HasSecrets: map[string]bool{
				"credentials": conf.Providers.Google.Credentials != "",
			},
		},
		{
			Name: "Azure",
			Fields: map[string]string{
				"clientId":       maskValue(conf.Providers.Azure.ClientId),
				"clientSecret":   maskValue(conf.Providers.Azure.ClientSecret),
				"subscriptionId": maskValue(conf.Providers.Azure.SubscriptionId),
				"tenantId":       maskValue(conf.Providers.Azure.TenantId),
			},
			HasSecrets: map[string]bool{
				"clientId":       conf.Providers.Azure.ClientId != "",
				"clientSecret":   conf.Providers.Azure.ClientSecret != "",
				"subscriptionId": conf.Providers.Azure.SubscriptionId != "",
				"tenantId":       conf.Providers.Azure.TenantId != "",
			},
		},
		{
			Name: "Oracle Cloud",
			Fields: map[string]string{
				"user":        maskValue(conf.Providers.Oracle.User),
				"tenancy":     maskValue(conf.Providers.Oracle.Tenancy),
				"fingerprint": maskValue(conf.Providers.Oracle.Fingerprint),
				"keyFile":     conf.Providers.Oracle.KeyFile,
				"region":      conf.Providers.Oracle.Region,
			},
			HasSecrets: map[string]bool{
				"user":        conf.Providers.Oracle.User != "",
				"tenancy":     conf.Providers.Oracle.Tenancy != "",
				"fingerprint": conf.Providers.Oracle.Fingerprint != "",
			},
		},
		{
			Name: "Cloudflare",
			Fields: map[string]string{
				"email":  conf.Cloudflare.Email,
				"apiKey": maskValue(conf.Cloudflare.APIKey),
			},
			HasSecrets: map[string]bool{
				"apiKey": conf.Cloudflare.APIKey != "",
			},
		},
	}

	return ProvidersConfigInfo{
		ConfigPath: configPath,
		Providers:  providers,
	}, nil
}

// SaveProvidersConfig saves provider credentials (only non-empty values are updated)
func (a *App) SaveProvidersConfig(providerName string, fields map[string]string, customPath string) error {
	conf, _, err := redc.ReadConfig(customPath)
	if err != nil {
		return err
	}

	switch providerName {
	case "AWS":
		if v, ok := fields["accessKey"]; ok && v != "" {
			conf.Providers.Aws.AccessKey = v
		}
		if v, ok := fields["secretKey"]; ok && v != "" {
			conf.Providers.Aws.SecretKey = v
		}
		if v, ok := fields["region"]; ok {
			conf.Providers.Aws.Region = v
		}
	case "阿里云":
		if v, ok := fields["accessKey"]; ok && v != "" {
			conf.Providers.Alicloud.AccessKey = v
		}
		if v, ok := fields["secretKey"]; ok && v != "" {
			conf.Providers.Alicloud.SecretKey = v
		}
		if v, ok := fields["region"]; ok {
			conf.Providers.Alicloud.Region = v
		}
	case "腾讯云":
		if v, ok := fields["secretId"]; ok && v != "" {
			conf.Providers.Tencentcloud.SecretId = v
		}
		if v, ok := fields["secretKey"]; ok && v != "" {
			conf.Providers.Tencentcloud.SecretKey = v
		}
		if v, ok := fields["region"]; ok {
			conf.Providers.Tencentcloud.Region = v
		}
	case "火山引擎":
		if v, ok := fields["accessKey"]; ok && v != "" {
			conf.Providers.Volcengine.AccessKey = v
		}
		if v, ok := fields["secretKey"]; ok && v != "" {
			conf.Providers.Volcengine.SecretKey = v
		}
		if v, ok := fields["region"]; ok {
			conf.Providers.Volcengine.Region = v
		}
	case "华为云":
		if v, ok := fields["accessKey"]; ok && v != "" {
			conf.Providers.Huaweicloud.AccessKey = v
		}
		if v, ok := fields["secretKey"]; ok && v != "" {
			conf.Providers.Huaweicloud.SecretKey = v
		}
		if v, ok := fields["region"]; ok {
			conf.Providers.Huaweicloud.Region = v
		}
	case "Google Cloud":
		if v, ok := fields["credentials"]; ok && v != "" {
			conf.Providers.Google.Credentials = v
		}
		if v, ok := fields["project"]; ok {
			conf.Providers.Google.Project = v
		}
		if v, ok := fields["region"]; ok {
			conf.Providers.Google.Region = v
		}
	case "Azure":
		if v, ok := fields["clientId"]; ok && v != "" {
			conf.Providers.Azure.ClientId = v
		}
		if v, ok := fields["clientSecret"]; ok && v != "" {
			conf.Providers.Azure.ClientSecret = v
		}
		if v, ok := fields["subscriptionId"]; ok && v != "" {
			conf.Providers.Azure.SubscriptionId = v
		}
		if v, ok := fields["tenantId"]; ok && v != "" {
			conf.Providers.Azure.TenantId = v
		}
	case "Oracle Cloud":
		if v, ok := fields["user"]; ok && v != "" {
			conf.Providers.Oracle.User = v
		}
		if v, ok := fields["tenancy"]; ok && v != "" {
			conf.Providers.Oracle.Tenancy = v
		}
		if v, ok := fields["fingerprint"]; ok && v != "" {
			conf.Providers.Oracle.Fingerprint = v
		}
		if v, ok := fields["keyFile"]; ok {
			conf.Providers.Oracle.KeyFile = v
		}
		if v, ok := fields["region"]; ok {
			conf.Providers.Oracle.Region = v
		}
	case "Cloudflare":
		if v, ok := fields["email"]; ok {
			conf.Cloudflare.Email = v
		}
		if v, ok := fields["apiKey"]; ok && v != "" {
			conf.Cloudflare.APIKey = v
		}
	default:
		return fmt.Errorf("未知的云厂商: %s", providerName)
	}

	if err := redc.SaveConfig(conf, customPath); err != nil {
		return err
	}

	a.emitLog(fmt.Sprintf("凭据配置已更新: %s", providerName))
	return nil
}

// ListCases returns all cases for the current project
func (a *App) ListCases() ([]CaseInfo, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.project == nil {
		if a.initError != "" {
			return nil, fmt.Errorf(a.initError)
		}
		return nil, fmt.Errorf("项目未加载")
	}

	cases, err := redc.LoadProjectCases(a.project.ProjectName)
	if err != nil {
		return nil, err
	}

	result := make([]CaseInfo, 0, len(cases))
	for _, c := range cases {
		result = append(result, CaseInfo{
			ID:         c.Id,
			Name:       c.Name,
			Type:       c.Type,
			State:      c.State,
			StateTime:  c.StateTime,
			CreateTime: c.CreateTime,
			Operator:   c.Operator,
		})
	}
	return result, nil
}

// TemplateInfo represents template information for frontend display
type TemplateInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`
	User        string `json:"user"`
	Module      string `json:"module"`
}

// ListTemplates returns available templates
func (a *App) ListTemplates() ([]TemplateInfo, error) {
	templates, err := redc.ListLocalTemplates()
	if err != nil {
		return nil, err
	}
	result := make([]TemplateInfo, 0, len(templates))
	for _, t := range templates {
		result = append(result, TemplateInfo{
			Name:        t.Name,
			Description: t.Description,
			Version:     t.Version,
			User:        t.User,
			Module:      t.RedcModule,
		})
	}
	return result, nil
}

// TemplateVariable represents a variable definition from terraform
type TemplateVariable struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	Description  string `json:"description"`
	DefaultValue string `json:"defaultValue"`
	Required     bool   `json:"required"`
}

// GetTemplateVariables parses variables.tf and terraform.tfvars to get template variables
func (a *App) GetTemplateVariables(templateName string) ([]TemplateVariable, error) {
	templatePath := filepath.Join(redc.TemplateDir, templateName)
	
	// Parse variables.tf to get variable definitions
	variablesFile := filepath.Join(templatePath, "variables.tf")
	tfvarsFile := filepath.Join(templatePath, "terraform.tfvars")
	
	variables := make(map[string]*TemplateVariable)
	
	// Parse variables.tf
	if _, err := os.Stat(variablesFile); err == nil {
		vars, err := parseVariablesTf(variablesFile)
		if err != nil {
			return nil, fmt.Errorf("解析 variables.tf 失败: %v", err)
		}
		for _, v := range vars {
			variables[v.Name] = v
		}
	}
	
	// Parse terraform.tfvars for default values
	if _, err := os.Stat(tfvarsFile); err == nil {
		defaults, err := parseTfvars(tfvarsFile)
		if err != nil {
			return nil, fmt.Errorf("解析 terraform.tfvars 失败: %v", err)
		}
		for name, value := range defaults {
			if v, ok := variables[name]; ok {
				v.DefaultValue = value
				v.Required = false
			}
		}
	}
	
	// Convert map to slice
	result := make([]TemplateVariable, 0, len(variables))
	for _, v := range variables {
		result = append(result, *v)
	}
	return result, nil
}

// parseVariablesTf parses a variables.tf file
func parseVariablesTf(filePath string) ([]*TemplateVariable, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var variables []*TemplateVariable
	scanner := bufio.NewScanner(file)
	
	varNameRegex := regexp.MustCompile(`^variable\s+"([^"]+)"`)
	typeRegex := regexp.MustCompile(`^\s*type\s*=\s*(.+)`)
	descRegex := regexp.MustCompile(`^\s*description\s*=\s*"([^"]*)"`)
	defaultRegex := regexp.MustCompile(`^\s*default\s*=\s*(.+)`)
	
	var currentVar *TemplateVariable
	braceCount := 0
	
	for scanner.Scan() {
		line := scanner.Text()
		
		// Check for variable declaration
		if matches := varNameRegex.FindStringSubmatch(line); len(matches) > 1 {
			if currentVar != nil {
				variables = append(variables, currentVar)
			}
			currentVar = &TemplateVariable{
				Name:     matches[1],
				Required: true,
				Type:     "string",
			}
			braceCount = 1
			continue
		}
		
		if currentVar == nil {
			continue
		}
		
		// Count braces
		braceCount += strings.Count(line, "{") - strings.Count(line, "}")
		
		// Parse type
		if matches := typeRegex.FindStringSubmatch(line); len(matches) > 1 {
			currentVar.Type = strings.TrimSpace(matches[1])
		}
		
		// Parse description
		if matches := descRegex.FindStringSubmatch(line); len(matches) > 1 {
			currentVar.Description = matches[1]
		}
		
		// Parse default
		if matches := defaultRegex.FindStringSubmatch(line); len(matches) > 1 {
			currentVar.DefaultValue = strings.Trim(strings.TrimSpace(matches[1]), `"`)
			currentVar.Required = false
		}
		
		// End of variable block
		if braceCount <= 0 && currentVar != nil {
			variables = append(variables, currentVar)
			currentVar = nil
		}
	}
	
	// Add last variable if exists
	if currentVar != nil {
		variables = append(variables, currentVar)
	}
	
	return variables, scanner.Err()
}

// parseTfvars parses a terraform.tfvars file
func parseTfvars(filePath string) (map[string]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	defaults := make(map[string]string)
	scanner := bufio.NewScanner(file)
	
	// Pattern: name = "value" or name = value
	lineRegex := regexp.MustCompile(`^([a-zA-Z_][a-zA-Z0-9_]*)\s*=\s*"?([^"]*)"?`)
	
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		if matches := lineRegex.FindStringSubmatch(line); len(matches) > 2 {
			defaults[matches[1]] = matches[2]
		}
	}
	
	return defaults, scanner.Err()
}

// StartCase starts a case by ID
func (a *App) StartCase(caseID string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.project == nil {
		return fmt.Errorf("项目未加载")
	}

	c, err := a.project.GetCase(caseID)
	if err != nil {
		return fmt.Errorf("获取场景失败: %v", err)
	}

	if c == nil {
		return fmt.Errorf("场景对象为 nil")
	}

	// Validate case before starting
	if c.Path == "" {
		return fmt.Errorf("场景路径为空")
	}

	caseName := c.Name
	casePath := c.Path
	caseState := c.State

	a.emitLog(fmt.Sprintf("准备启动场景: %s, 路径: %s, 当前状态: %s", caseName, casePath, caseState))

	// Run in goroutine to avoid blocking GUI
	go func() {
		defer func() {
			if r := recover(); r != nil {
				a.emitLog(fmt.Sprintf("启动场景时发生错误: %v", r))
			}
			a.emitRefresh() // 操作完成后刷新仪表盘
		}()
		
		a.emitLog(fmt.Sprintf("正在启动场景: %s", caseName))
		if err := c.TfApply(); err != nil {
			a.emitLog(fmt.Sprintf("启动失败: %v", err))
			return
		}
		a.emitLog(fmt.Sprintf("场景启动成功: %s", caseName))
		
		// 获取并显示 outputs
		if outputs, err := c.TfOutput(); err == nil {
			for name, meta := range outputs {
				a.emitLog(fmt.Sprintf("  %s = %s", name, string(meta.Value)))
			}
		}
	}()

	return nil
}

// StopCase stops a case by ID
func (a *App) StopCase(caseID string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.project == nil {
		return fmt.Errorf("项目未加载")
	}

	c, err := a.project.GetCase(caseID)
	if err != nil {
		return err
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				a.emitLog(fmt.Sprintf("停止场景时发生错误: %v", r))
			}
			a.emitRefresh() // 操作完成后刷新仪表盘
		}()
		
		a.emitLog(fmt.Sprintf("正在停止场景: %s", c.Name))
		if err := c.Stop(); err != nil {
			a.emitLog(fmt.Sprintf("停止失败: %v", err))
			return
		}
		a.emitLog(fmt.Sprintf("场景停止成功: %s", c.Name))
	}()

	return nil
}

// RemoveCase removes a case by ID
func (a *App) RemoveCase(caseID string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.project == nil {
		return fmt.Errorf("项目未加载")
	}

	c, err := a.project.GetCase(caseID)
	if err != nil {
		return err
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				a.emitLog(fmt.Sprintf("删除场景时发生错误: %v", r))
			}
			a.emitRefresh() // 操作完成后刷新仪表盘
		}()
		
		a.emitLog(fmt.Sprintf("正在删除场景: %s", c.Name))
		if err := c.Remove(); err != nil {
			a.emitLog(fmt.Sprintf("删除失败: %v", err))
			return
		}
		a.emitLog(fmt.Sprintf("场景删除成功: %s", c.Name))
	}()

	return nil
}

// CreateCase creates a new case from a template (async)
func (a *App) CreateCase(templateName string, name string, vars map[string]string) error {
	a.mu.Lock()
	if a.project == nil {
		a.mu.Unlock()
		return fmt.Errorf("项目未加载")
	}
	project := a.project
	a.mu.Unlock()

	a.emitLog(fmt.Sprintf("正在创建场景: %s (模板: %s)", name, templateName))

	go func() {
		defer func() {
			if r := recover(); r != nil {
				a.emitLog(fmt.Sprintf("创建场景时发生错误: %v", r))
			}
			a.emitRefresh()
		}()

		a.emitLog(fmt.Sprintf("场景初始化中: %s (模板: %s)", name, templateName))
		c, err := project.CaseCreate(templateName, redc.U, name, vars)
		if err != nil {
			a.emitLog(fmt.Sprintf("场景创建失败: %v", err))
			return
		}
		a.emitLog(fmt.Sprintf("场景创建成功: %s (%s)", c.Name, c.GetId()))
	}()

	return nil
}

// CreateAndRunCase creates a new case and immediately starts it (like CLI "run" command)
func (a *App) CreateAndRunCase(templateName string, name string, vars map[string]string) error {
	a.mu.Lock()
	if a.project == nil {
		a.mu.Unlock()
		return fmt.Errorf("项目未加载")
	}
	project := a.project
	a.mu.Unlock()

	a.emitLog(fmt.Sprintf("正在创建并运行场景: %s (模板: %s)", name, templateName))

	go func() {
		defer func() {
			if r := recover(); r != nil {
				a.emitLog(fmt.Sprintf("创建场景时发生错误: %v", r))
			}
			a.emitRefresh()
		}()

		a.emitLog(fmt.Sprintf("场景初始化中: %s (模板: %s)", name, templateName))
		// Step 1: Create the case (same as planLogic in CLI)
		c, err := project.CaseCreate(templateName, redc.U, name, vars)
		if err != nil {
			a.emitLog(fmt.Sprintf("场景创建失败: %v", err))
			return
		}
		a.emitLog(fmt.Sprintf("场景创建成功: %s (%s)", c.Name, c.GetId()))

		// Step 2: Start the case immediately (same as runCmd in CLI)
		a.emitLog(fmt.Sprintf("正在启动场景: %s", c.Name))

		// Run terraform apply
		if err := c.TfApply(); err != nil {
			a.emitLog(fmt.Sprintf("启动失败: %v", err))
			return
		}

		a.emitLog(fmt.Sprintf("场景启动成功: %s", c.Name))

		// Get and display outputs
		if outputs, err := c.TfOutput(); err == nil {
			for key, meta := range outputs {
				a.emitLog(fmt.Sprintf("%s = %s", key, string(meta.Value)))
			}
		}

		a.emitRefresh()
	}()

	return nil
}

// DeployCase creates and immediately starts a case (deprecated - use CreateCase then StartCase)
func (a *App) DeployCase(templateName string, name string, vars map[string]string) error {
	// CreateCase is now async, so this method just creates the case
	// User should manually start it after creation
	return a.CreateCase(templateName, name, vars)
}

// GetCaseOutputs returns the terraform outputs for a case
func (a *App) GetCaseOutputs(caseID string) (map[string]string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.project == nil {
		return nil, fmt.Errorf("项目未加载")
	}

	c, err := a.project.GetCase(caseID)
	if err != nil {
		return nil, err
	}

	// Only get outputs for running cases
	if c.State != "running" {
		return nil, nil
	}

	outputs, err := c.TfOutput()
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for name, meta := range outputs {
		// Remove quotes from JSON string values
		value := string(meta.Value)
		if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
			value = value[1 : len(value)-1]
		}
		// Resolve relative file paths to absolute paths based on case directory
		if isRelativeFilePath(value) {
			absPath := filepath.Join(c.Path, value)
			if _, err := os.Stat(absPath); err == nil {
				value = absPath
			}
		}
		result[name] = value
	}
	return result, nil
}

// isRelativeFilePath checks if the value looks like a relative file path
func isRelativeFilePath(value string) bool {
	if value == "" {
		return false
	}
	// Check for common relative path patterns
	if strings.HasPrefix(value, "./") || strings.HasPrefix(value, "../") {
		return true
	}
	return false
}

// RegistryTemplate represents a template from the remote registry
type RegistryTemplate struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Author      string   `json:"author"`
	Latest      string   `json:"latest"`
	Versions    []string `json:"versions"`
	UpdatedAt   string   `json:"updatedAt"`
	Tags        []string `json:"tags"`
	Installed   bool     `json:"installed"`
	LocalVer    string   `json:"localVersion"`
}

// remoteTemplateInfo matches a single template in the registry
type remoteTemplateInfo struct {
	ID       string `json:"id"`
	Provider string `json:"provider"`
	Slug     string `json:"slug"`
	Latest   string `json:"latest"`
	Versions map[string]struct {
		URL       string `json:"url"`
		SHA256    string `json:"sha256"`
		UpdatedAt string `json:"updated_at"`
	} `json:"versions"`
	Metadata struct {
		Name        string `json:"name"`
		Author      string `json:"author"`
		Description string `json:"description"`
		Readme      string `json:"readme"`
	} `json:"metadata"`
}

// remoteIndexResponse matches the index.json structure from the registry
type remoteIndexResponse struct {
	UpdatedAt string                        `json:"updated_at"`
	RepoName  string                        `json:"repo_name"`
	Templates map[string]remoteTemplateInfo `json:"templates"`
}

// FetchRegistryTemplates fetches templates from the remote registry
func (a *App) FetchRegistryTemplates(registryURL string) ([]RegistryTemplate, error) {
	if registryURL == "" {
		registryURL = "https://redc.wgpsec.org"
	}

	a.emitLog(fmt.Sprintf("正在连接仓库: %s", registryURL))

	// Fetch index.json
	indexURL := fmt.Sprintf("%s/index.json?t=%d", registryURL, time.Now().Unix())
	
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(indexURL)
	if err != nil {
		return nil, fmt.Errorf("连接仓库失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("仓库返回错误: %s", resp.Status)
	}

	var idx remoteIndexResponse
	if err := json.NewDecoder(resp.Body).Decode(&idx); err != nil {
		return nil, fmt.Errorf("解析仓库索引失败: %v", err)
	}

	// Build result list
	result := make([]RegistryTemplate, 0, len(idx.Templates))
	for templateID, t := range idx.Templates {
		// Use templateID (e.g. "aliyun/ecs") as the name
		name := templateID
		if name == "" {
			name = t.ID
		}
		
		// Check if installed locally
		installed, localVer, _ := redc.CheckLocalImage(name)
		
		// Get version list
		versions := make([]string, 0, len(t.Versions))
		var updatedAt string
		for v, info := range t.Versions {
			versions = append(versions, v)
			if v == t.Latest && info.UpdatedAt != "" {
				updatedAt = info.UpdatedAt
			}
		}

		// Extract tags from provider
		var tags []string
		if t.Provider != "" {
			tags = []string{t.Provider}
		}

		result = append(result, RegistryTemplate{
			Name:        name,
			Description: t.Metadata.Description,
			Author:      t.Metadata.Author,
			Latest:      t.Latest,
			Versions:    versions,
			UpdatedAt:   updatedAt,
			Tags:        tags,
			Installed:   installed,
			LocalVer:    localVer,
		})
	}

	a.emitLog(fmt.Sprintf("已获取 %d 个模板", len(result)))
	return result, nil
}

// RemoveTemplate removes a local template (aligns with CLI `image rm`)
func (a *App) RemoveTemplate(templateName string) error {
	a.emitLog(fmt.Sprintf("正在删除模板: %s", templateName))

	if err := redc.RemoveTemplate(templateName); err != nil {
		a.emitLog(fmt.Sprintf("删除失败: %v", err))
		return err
	}

	a.emitLog(fmt.Sprintf("模板删除成功: %s", templateName))
	a.emitRefresh()
	return nil
}

// PullTemplate pulls a template from the registry
func (a *App) PullTemplate(templateName string, force bool) error {
	a.emitLog(fmt.Sprintf("正在拉取模板: %s", templateName))

	go func() {
		defer func() {
			if r := recover(); r != nil {
				a.emitLog(fmt.Sprintf("拉取模板时发生错误: %v", r))
			}
			a.emitRefresh()
		}()

		opts := redc.PullOptions{
			RegistryURL: "https://redc.wgpsec.org",
			Force:       force,
			Timeout:     120 * time.Second,
		}

		if err := redc.Pull(context.Background(), templateName, opts); err != nil {
			a.emitLog(fmt.Sprintf("拉取失败: %v", err))
			return
		}

		a.emitLog(fmt.Sprintf("模板拉取成功: %s", templateName))
	}()

	return nil
}

// MCPStatus represents the MCP server status for frontend display
type MCPStatus struct {
	Running         bool   `json:"running"`
	Mode            string `json:"mode"`
	Address         string `json:"address"`
	ProtocolVersion string `json:"protocolVersion"`
}

// GetMCPStatus returns the current MCP server status
func (a *App) GetMCPStatus() MCPStatus {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.mcpManager == nil {
		return MCPStatus{Running: false}
	}

	status := a.mcpManager.GetStatus()
	return MCPStatus{
		Running:         status["running"].(bool),
		Mode:            status["mode"].(string),
		Address:         status["address"].(string),
		ProtocolVersion: status["protocolVersion"].(string),
	}
}

// StartMCPServer starts the MCP server with the specified mode and address
func (a *App) StartMCPServer(mode string, address string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.project == nil {
		return fmt.Errorf("项目未加载")
	}

	// Create manager if not exists
	if a.mcpManager == nil {
		a.mcpManager = mcp.NewMCPServerManager(a.project)
		a.mcpManager.SetLogCallback(a.emitLog)
	}

	// Convert mode string to TransportMode
	var transportMode mcp.TransportMode
	switch mode {
	case "sse":
		transportMode = mcp.TransportSSE
	case "stdio":
		transportMode = mcp.TransportSTDIO
	default:
		return fmt.Errorf("未知的传输模式: %s", mode)
	}

	if err := a.mcpManager.Start(transportMode, address); err != nil {
		return fmt.Errorf("启动 MCP 服务器失败: %v", err)
	}

	a.emitLog(fmt.Sprintf("MCP 服务器已启动 - 模式: %s, 地址: %s", mode, address))
	return nil
}

// StopMCPServer stops the running MCP server
func (a *App) StopMCPServer() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.mcpManager == nil {
		return fmt.Errorf("MCP 服务器未初始化")
	}

	if err := a.mcpManager.Stop(); err != nil {
		return fmt.Errorf("停止 MCP 服务器失败: %v", err)
	}

	a.emitLog("MCP 服务器已停止")
	return nil
}
