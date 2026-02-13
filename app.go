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
	"sort"
	"strings"
	"sync"
	"time"
	goruntime "runtime"

	redc "red-cloud/mod"
	"red-cloud/mod/ai"
	"red-cloud/mod/cost"
	"red-cloud/mod/gologger"
	"red-cloud/mod/mcp"
	"red-cloud/mod/compose"
	"red-cloud/utils/sshutil"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"github.com/projectdiscovery/gologger/levels"
	tfjson "github.com/hashicorp/terraform-json"
	"golang.org/x/crypto/ssh"
)

// App struct
type App struct {
	ctx                context.Context
	project            *redc.RedcProject
	mu                 sync.Mutex
	initError          string
	logMgr             *gologger.LogManager
	mcpManager         *mcp.MCPServerManager
	notificationMgr    *NotificationManager
	pricingService     *cost.PricingService
	costCalculator     *cost.CostCalculator
	taskScheduler      *redc.TaskScheduler
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		notificationMgr: NewNotificationManager(),
	}
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
	if profile, err := redc.GetActiveProfile(); err == nil {
		if _, err := redc.SetActiveProfile(profile.ID); err != nil {
			runtime.LogInfof(ctx, "Profile 初始化失败: %v", err)
		}
	} else {
		runtime.LogInfof(ctx, "Profile 初始化失败: %v", err)
	}

	runtime.LogInfof(ctx, "配置加载成功 - RedcPath: %s, ProjectPath: %s, TemplateDir: %s", 
		redc.RedcPath, redc.ProjectPath, redc.TemplateDir)

	// Load default project
	if p, err := redc.ProjectParse(redc.Project, redc.U); err == nil {
		a.project = p
		a.logMgr = gologger.NewLogManager(p.ProjectPath)
		gologger.DefaultLogger.SetWriter(&guiWriter{out: a.createLogWriter("core")})
		if redc.Debug {
			gologger.DefaultLogger.SetMaxLevel(levels.LevelDebug)
		} else {
			gologger.DefaultLogger.SetMaxLevel(levels.LevelInfo)
		}
		runtime.LogInfof(ctx, "项目加载成功: %s", a.project.ProjectName)
	} else {
		a.initError = fmt.Sprintf("项目加载失败: %v", err)
		runtime.LogErrorf(ctx, a.initError)
	}

	// Initialize cost estimation components
	pricingCacheDBPath := filepath.Join(redc.RedcPath, "pricing_cache.db")
	a.pricingService = cost.NewPricingService(pricingCacheDBPath)
	a.costCalculator = cost.NewCostCalculator()
	
	// Set credential provider for cost estimation
	// This function reads credentials from the active config file
	credProvider := func(provider string) (accessKey, secretKey, region string, err error) {
		conf, _, err := redc.ReadConfig(redc.ActiveConfigPath)
		if err != nil {
			return "", "", "", fmt.Errorf("failed to read config: %w", err)
		}
		
		switch provider {
		case "alicloud":
			return conf.Providers.Alicloud.AccessKey, conf.Providers.Alicloud.SecretKey, conf.Providers.Alicloud.Region, nil
		case "tencentcloud":
			return conf.Providers.Tencentcloud.SecretId, conf.Providers.Tencentcloud.SecretKey, conf.Providers.Tencentcloud.Region, nil
		case "aws":
			return conf.Providers.Aws.AccessKey, conf.Providers.Aws.SecretKey, conf.Providers.Aws.Region, nil
		case "volcengine":
			return conf.Providers.Volcengine.AccessKey, conf.Providers.Volcengine.SecretKey, conf.Providers.Volcengine.Region, nil
		default:
			return "", "", "", fmt.Errorf("unsupported provider: %s", provider)
		}
	}
	
	a.pricingService.SetCredentialProvider(credProvider)
	
	// Also set global credential provider for data source resolution
	cost.SetGlobalCredentialProvider(credProvider)
	
	// Start background cache cleanup (runs every hour)
	a.pricingService.StartCacheCleanup(1 * time.Hour)
	
	runtime.LogInfof(ctx, "成本估算服务初始化成功 - 缓存路径: %s", pricingCacheDBPath)
	
	// Initialize task scheduler
	schedulerDBPath := filepath.Join(redc.RedcPath, "scheduler.db")
	a.taskScheduler = redc.NewTaskScheduler(a.project, schedulerDBPath)
	
	// 初始化数据库
	if err := a.taskScheduler.InitDB(); err != nil {
		runtime.LogErrorf(ctx, "任务调度器数据库初始化失败: %v", err)
	} else {
		runtime.LogInfof(ctx, "任务调度器数据库初始化成功: %s", schedulerDBPath)
	}
	
	a.taskScheduler.SetExecuteCallback(func(caseID string, action string) error {
		if action == "start" {
			err := a.StartCase(caseID)
			if err == nil {
				a.emitRefresh()
			}
			return err
		} else if action == "stop" {
			err := a.StopCase(caseID)
			if err == nil {
				a.emitRefresh()
			}
			return err
		}
		return fmt.Errorf("未知操作: %s", action)
	})
	a.taskScheduler.Start()
	
	runtime.LogInfof(ctx, "任务调度器启动成功")
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
	DebugEnabled bool  `json:"debugEnabled"`
}

// TerraformMirrorConfig represents terraform mirror configuration status
type TerraformMirrorConfig struct {
	Enabled    bool   `json:"enabled"`
	ConfigPath string `json:"configPath"`
	Managed    bool   `json:"managed"`
	FromEnv    bool   `json:"fromEnv"`
	Providers  []string `json:"providers"`
}

// EndpointCheck represents a connectivity check result
type EndpointCheck struct {
	Name      string `json:"name"`
	URL       string `json:"url"`
	OK        bool   `json:"ok"`
	Status    int    `json:"status"`
	Error     string `json:"error"`
	LatencyMs int64  `json:"latencyMs"`
	CheckedAt string `json:"checkedAt"`
}

// guiWriter adapts an io.Writer to gologger writer.Writer
type guiWriter struct {
	out io.Writer
}

func (w *guiWriter) Write(data []byte, level levels.Level) {
	if w.out == nil {
		return
	}
	_, _ = w.out.Write(data)
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
		DebugEnabled: redc.Debug,
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

func defaultTerraformConfigPath() (string, bool, error) {
	if envPath := strings.TrimSpace(os.Getenv("TF_CLI_CONFIG_FILE")); envPath != "" {
		return envPath, true, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", false, err
	}
	if goruntime.GOOS == "windows" {
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		return filepath.Join(appData, "terraform.rc"), false, nil
	}
	return filepath.Join(home, ".terraformrc"), false, nil
}

func parseTerraformMirrorProviders(content string) []string {
	providers := []string{}
	if strings.Contains(content, "registry.terraform.io/aliyun/alicloud") || strings.Contains(content, "registry.terraform.io/hashicorp/alicloud") {
		providers = append(providers, "aliyun")
	}
	if strings.Contains(content, "registry.terraform.io/tencentcloudstack/") {
		providers = append(providers, "tencent")
	}
	if strings.Contains(content, "registry.terraform.io/volcengine/") {
		providers = append(providers, "volc")
	}
	return providers
}

func terraformMirrorConfigContent(enabled bool, providers []string) string {
	var builder strings.Builder
	builder.WriteString("# Generated by redc-gui\n")
	builder.WriteString("plugin_cache_dir = \"$HOME/.terraform.d/plugin-cache\"\n")
	builder.WriteString("disable_checkpoint = true\n")
	// 始终优先使用本地缓存，即使网络不可达也能使用已缓存的 provider
	builder.WriteString("plugin_cache_may_break_dependency_lock_file = true\n\n")
	
	if !enabled || len(providers) == 0 {
		return builder.String()
	}

	providerSet := make(map[string]bool)
	for _, p := range providers {
		providerSet[p] = true
	}

	builder.WriteString("provider_installation {\n")

	excludes := []string{}
	if providerSet["aliyun"] {
		builder.WriteString("  network_mirror {\n")
		builder.WriteString("    url = \"https://mirrors.aliyun.com/terraform/\"\n")
		builder.WriteString("    include = [\n")
		builder.WriteString("      \"registry.terraform.io/aliyun/alicloud\",\n")
		builder.WriteString("      \"registry.terraform.io/hashicorp/alicloud\"\n")
		builder.WriteString("    ]\n")
		builder.WriteString("  }\n")
		excludes = append(excludes, "registry.terraform.io/aliyun/alicloud", "registry.terraform.io/hashicorp/alicloud")
	}
	if providerSet["tencent"] {
		builder.WriteString("  network_mirror {\n")
		builder.WriteString("    url = \"https://mirrors.tencent.com/terraform/\"\n")
		builder.WriteString("    include = [\n")
		builder.WriteString("      \"registry.terraform.io/tencentcloudstack/*\"\n")
		builder.WriteString("    ]\n")
		builder.WriteString("  }\n")
		excludes = append(excludes, "registry.terraform.io/tencentcloudstack/*")
	}
	if providerSet["volc"] {
		builder.WriteString("  network_mirror {\n")
		builder.WriteString("    url = \"https://mirrors.volces.com/terraform/\"\n")
		builder.WriteString("    include = [\n")
		builder.WriteString("      \"registry.terraform.io/volcengine/*\"\n")
		builder.WriteString("    ]\n")
		builder.WriteString("  }\n")
		excludes = append(excludes, "registry.terraform.io/volcengine/*")
	}

	if len(excludes) > 0 {
		builder.WriteString("  direct {\n")
		builder.WriteString("    exclude = [\n")
		for i, item := range excludes {
			if i < len(excludes)-1 {
				builder.WriteString(fmt.Sprintf("      \"%s\",\n", item))
			} else {
				builder.WriteString(fmt.Sprintf("      \"%s\"\n", item))
			}
		}
		builder.WriteString("    ]\n")
		builder.WriteString("  }\n")
	}
	builder.WriteString("}\n")
	return builder.String()
}

// GetTerraformMirrorConfig returns current terraform mirror configuration status
func (a *App) GetTerraformMirrorConfig() (TerraformMirrorConfig, error) {
	configPath, fromEnv, err := defaultTerraformConfigPath()
	if err != nil {
		return TerraformMirrorConfig{}, err
	}
	result := TerraformMirrorConfig{
		Enabled:    false,
		ConfigPath: configPath,
		Managed:    false,
		FromEnv:    fromEnv,
		Providers:  []string{},
	}
	content, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return result, nil
		}
		return result, err
	}
	text := string(content)
	result.Managed = strings.Contains(text, "redc-gui")
	result.Providers = parseTerraformMirrorProviders(text)
	result.Enabled = len(result.Providers) > 0
	return result, nil
}

// SaveTerraformMirrorConfig writes terraform mirror configuration
func (a *App) SaveTerraformMirrorConfig(enabled bool, providers []string, configPath string, setEnv bool) error {
	path := strings.TrimSpace(configPath)
	if path == "" {
		p, _, err := defaultTerraformConfigPath()
		if err != nil {
			return err
		}
		path = p
	}
	if setEnv {
		os.Setenv("TF_CLI_CONFIG_FILE", path)
	}
	content := terraformMirrorConfigContent(enabled, providers)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return err
	}
	if enabled {
		a.emitLog(fmt.Sprintf("Terraform 镜像配置已写入: %s", path))
	} else {
		a.emitLog(fmt.Sprintf("Terraform 镜像配置已关闭: %s", path))
	}
	return nil
}

// TestTerraformEndpoints checks connectivity for terraform endpoints
func (a *App) TestTerraformEndpoints() ([]EndpointCheck, error) {
	endpoints := []struct {
		Name string
		URL  string
	}{
		{Name: "Terraform Registry", URL: "https://registry.terraform.io/.well-known/terraform.json"},
		{Name: "Alibaba Cloud Mirror", URL: "https://mirrors.aliyun.com/terraform/"},
		{Name: "Tencent Cloud Mirror", URL: "https://mirrors.tencent.com/terraform/"},
		{Name: "Volcengine Mirror", URL: "https://mirrors.volces.com/terraform/"},
	}
	client := &http.Client{Timeout: 6 * time.Second}
	results := make([]EndpointCheck, 0, len(endpoints))
	for _, ep := range endpoints {
		start := time.Now()
		status := 0
		ok := false
		errMsg := ""
		req, err := http.NewRequest("GET", ep.URL, nil)
		if err != nil {
			errMsg = err.Error()
		} else {
			req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36")
			req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
			req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
			req.Header.Set("Cache-Control", "no-cache")
			req.Header.Set("Pragma", "no-cache")
			resp, err := client.Do(req)
			if err != nil {
				errMsg = err.Error()
			} else {
				status = resp.StatusCode
				if resp.Body != nil {
					_, _ = io.Copy(io.Discard, resp.Body)
					resp.Body.Close()
				}
				ok = status >= 200 && status < 400
				if status == 403 {
					ok = false
					if errMsg == "" {
						errMsg = "403 Forbidden"
					}
				}
			}
		}
		results = append(results, EndpointCheck{
			Name:      ep.Name,
			URL:       ep.URL,
			OK:        ok,
			Status:    status,
			Error:     errMsg,
			LatencyMs: time.Since(start).Milliseconds(),
			CheckedAt: time.Now().Format(time.RFC3339),
		})
	}
	return results, nil
}

// SetDebugLogging enables or disables debug logging for GUI
func (a *App) SetDebugLogging(enabled bool) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	redc.Debug = enabled
	if enabled {
		gologger.DefaultLogger.SetMaxLevel(levels.LevelDebug)
		a.emitLog("调试日志已开启")
	} else {
		gologger.DefaultLogger.SetMaxLevel(levels.LevelInfo)
		a.emitLog("调试日志已关闭")
	}
	return nil
}

// SetNotificationEnabled enables or disables system notifications
func (a *App) SetNotificationEnabled(enabled bool) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.notificationMgr != nil {
		a.notificationMgr.SetEnabled(enabled)
		if enabled {
			a.emitLog("系统通知已开启")
		} else {
			a.emitLog("系统通知已关闭")
		}
	}
	return nil
}

// GetNotificationEnabled returns whether system notifications are enabled
func (a *App) GetNotificationEnabled() bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.notificationMgr != nil {
		return a.notificationMgr.IsEnabled()
	}
	return false
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
			Name: "Vultr",
			Fields: map[string]string{
				"apiKey": maskValue(conf.Providers.Vultr.ApiKey),
			},
			HasSecrets: map[string]bool{
				"apiKey": conf.Providers.Vultr.ApiKey != "",
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
	case "Vultr":
		if v, ok := fields["apiKey"]; ok && v != "" {
			conf.Providers.Vultr.ApiKey = v
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

// ListProfiles returns all available profiles
func (a *App) ListProfiles() ([]redc.ProfileInfo, error) {
	return redc.ListProfiles()
}

// GetActiveProfile returns the active profile
func (a *App) GetActiveProfile() (redc.ProfileInfo, error) {
	return redc.GetActiveProfile()
}

// SetActiveProfile switches the active profile
func (a *App) SetActiveProfile(profileID string) (redc.ProfileInfo, error) {
	return redc.SetActiveProfile(profileID)
}

// CreateProfile creates a new profile
func (a *App) CreateProfile(name string, configPath string, templateDir string) (redc.ProfileInfo, error) {
	return redc.CreateProfile(name, configPath, templateDir)
}

// UpdateProfile updates an existing profile
func (a *App) UpdateProfile(profileID string, name string, configPath string, templateDir string) (redc.ProfileInfo, error) {
	return redc.UpdateProfile(profileID, name, configPath, templateDir)
}

// DeleteProfile removes a profile
func (a *App) DeleteProfile(profileID string) error {
	return redc.DeleteProfile(profileID)
}

// UpdateProfileAIConfig updates AI configuration for a profile
func (a *App) UpdateProfileAIConfig(profileID string, provider string, apiKey string, baseUrl string, model string) error {
	aiConfig := &redc.AIConfig{
		Provider: provider,
		APIKey:   apiKey,
		BaseURL:  baseUrl,
		Model:    model,
	}
	return redc.UpdateProfileAIConfig(profileID, aiConfig)
}

// ProjectInfo represents project information for frontend display
type ProjectInfo struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	CreateTime string `json:"createTime"`
	User       string `json:"user"`
}

// ListProjects returns all projects
func (a *App) ListProjects() ([]ProjectInfo, error) {
	projects, err := redc.ListAllProjects()
	if err != nil {
		return nil, err
	}

	result := make([]ProjectInfo, 0, len(projects))
	for _, p := range projects {
		result = append(result, ProjectInfo{
			Name:       p.ProjectName,
			Path:       p.ProjectPath,
			CreateTime: p.CreateTime,
			User:       p.User,
		})
	}
	return result, nil
}

// GetCurrentProject returns the current project name
func (a *App) GetCurrentProject() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.project == nil {
		return ""
	}
	return a.project.ProjectName
}

// SwitchProject switches to a different project
func (a *App) SwitchProject(projectName string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Parse and load the new project
	p, err := redc.ProjectParse(projectName, redc.U)
	if err != nil {
		return fmt.Errorf("切换项目失败: %v", err)
	}

	// Update project reference
	a.project = p

	// Update log manager to use new project path
	a.logMgr = gologger.NewLogManager(p.ProjectPath)

	// Update global project variable
	redc.Project = projectName

	// Emit log and refresh
	a.emitLog(fmt.Sprintf("已切换到项目: %s", projectName))
	a.emitRefresh()

	return nil
}

// CreateProject creates a new project
func (a *App) CreateProject(name string) error {
	_, err := redc.NewProjectConfig(name, redc.U)
	if err != nil {
		return fmt.Errorf("创建项目失败: %v", err)
	}
	a.emitLog(fmt.Sprintf("已创建新项目: %s", name))
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

// GetResourceSummary aggregates terraform resources by type across all cases
func (a *App) GetResourceSummary() ([]ResourceSummary, error) {
	a.mu.Lock()
	project := a.project
	a.mu.Unlock()

	if project == nil {
		if a.initError != "" {
			return nil, fmt.Errorf(a.initError)
		}
		return nil, fmt.Errorf("项目未加载")
	}

	cases, err := redc.LoadProjectCases(project.ProjectName)
	if err != nil {
		return nil, err
	}

	counts := make(map[string]int)
	for _, c := range cases {
		if c.State != redc.StateRunning {
			continue
		}
		if c.Path == "" {
			continue
		}
		state, err := redc.TfStatus(c.Path)
		if err != nil || state == nil || state.Values == nil {
			continue
		}
		addModuleResources(counts, state.Values.RootModule)
	}

	result := make([]ResourceSummary, 0, len(counts))
	for typ, count := range counts {
		result = append(result, ResourceSummary{Type: typ, Count: count})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Type < result[j].Type })
	return result, nil
}

func addModuleResources(counts map[string]int, module *tfjson.StateModule) {
	if module == nil {
		return
	}
	for _, res := range module.Resources {
		if res.Type != "" {
			counts[res.Type]++
		}
	}
	for _, child := range module.ChildModules {
		addModuleResources(counts, child)
	}
}

// ComposePreview parses a compose file and returns services summary
func (a *App) ComposePreview(filePath string, profiles []string) (ComposeSummary, error) {
	a.mu.Lock()
	project := a.project
	a.mu.Unlock()

	if project == nil {
		if a.initError != "" {
			return ComposeSummary{}, fmt.Errorf(a.initError)
		}
		return ComposeSummary{}, fmt.Errorf("项目未加载")
	}

	if strings.TrimSpace(filePath) == "" {
		filePath = "redc-compose.yaml"
	}

	ctx, err := compose.NewComposeContext(compose.ComposeOptions{
		File:     filePath,
		Profiles: profiles,
		Project:  project,
	})
	if err != nil {
		return ComposeSummary{}, err
	}

	services := make([]ComposeServiceSummary, 0, len(ctx.SortedSvcKeys))
	for _, name := range ctx.SortedSvcKeys {
		svc := ctx.RuntimeSvcs[name]
		services = append(services, ComposeServiceSummary{
			Name:      svc.Name,
			RawName:   svc.RawName,
			Template:  svc.Spec.Image,
			Provider:  formatComposeProvider(svc.Spec.Provider),
			Profiles:  svc.Spec.Profiles,
			DependsOn: svc.Spec.DependsOn,
			Replicas:  svc.Spec.Deploy.Replicas,
		})
	}

	return ComposeSummary{
		File:     filePath,
		Services: services,
		Total:    len(services),
	}, nil
}

// ComposeUp starts compose deployment asynchronously
func (a *App) ComposeUp(filePath string, profiles []string) error {
	a.mu.Lock()
	project := a.project
	a.mu.Unlock()

	if project == nil {
		if a.initError != "" {
			return fmt.Errorf(a.initError)
		}
		return fmt.Errorf("项目未加载")
	}

	if strings.TrimSpace(filePath) == "" {
		filePath = "redc-compose.yaml"
	}

	opts := compose.ComposeOptions{
		File:     filePath,
		Profiles: profiles,
		Project:  project,
	}

	a.emitLog(fmt.Sprintf("开始执行 compose up: %s", filePath))
	go func() {
		defer a.emitRefresh()
		if err := compose.RunComposeUp(opts); err != nil {
			a.emitLog(fmt.Sprintf("compose up 失败: %v", err))
			return
		}
		a.emitLog("compose up 完成")
	}()
	return nil
}

// ComposeDown destroys compose deployment asynchronously
func (a *App) ComposeDown(filePath string, profiles []string) error {
	a.mu.Lock()
	project := a.project
	a.mu.Unlock()

	if project == nil {
		if a.initError != "" {
			return fmt.Errorf(a.initError)
		}
		return fmt.Errorf("项目未加载")
	}

	if strings.TrimSpace(filePath) == "" {
		filePath = "redc-compose.yaml"
	}

	opts := compose.ComposeOptions{
		File:     filePath,
		Profiles: profiles,
		Project:  project,
	}

	a.emitLog(fmt.Sprintf("开始执行 compose down: %s", filePath))
	go func() {
		defer a.emitRefresh()
		if err := compose.RunComposeDown(opts); err != nil {
			a.emitLog(fmt.Sprintf("compose down 失败: %v", err))
			return
		}
		a.emitLog("compose down 完成")
	}()
	return nil
}

func formatComposeProvider(provider interface{}) string {
	if provider == nil {
		return ""
	}
	switch v := provider.(type) {
	case string:
		return v
	case []string:
		return strings.Join(v, ",")
	case []interface{}:
		items := make([]string, 0, len(v))
		for _, item := range v {
			items = append(items, fmt.Sprintf("%v", item))
		}
		return strings.Join(items, ",")
	default:
		return fmt.Sprintf("%v", provider)
	}
}

// TemplateInfo represents template information for frontend display
type TemplateInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`
	User        string `json:"user"`
	Module      string `json:"module"`
}

// ResourceSummary represents aggregated resource counts by type
type ResourceSummary struct {
	Type  string `json:"type"`
	Count int    `json:"count"`
}

// BalanceInfo represents account balance result
type BalanceInfo struct {
	Provider  string `json:"provider"`
	Amount    string `json:"amount"`
	Currency  string `json:"currency"`
	UpdatedAt string `json:"updatedAt"`
	Error     string `json:"error"`
}

// ComposeServiceSummary represents a compose service preview
type ComposeServiceSummary struct {
	Name      string   `json:"name"`
	RawName   string   `json:"rawName"`
	Template  string   `json:"template"`
	Provider  string   `json:"provider"`
	Profiles  []string `json:"profiles"`
	DependsOn []string `json:"dependsOn"`
	Replicas  int      `json:"replicas"`
}

// ComposeSummary represents a compose file preview
type ComposeSummary struct {
	File     string                  `json:"file"`
	Services []ComposeServiceSummary `json:"services"`
	Total    int                     `json:"total"`
}

// GetBalances returns account balances for selected providers (manual trigger)
func (a *App) GetBalances(providers []string) ([]BalanceInfo, error) {
	if len(providers) == 0 {
		providers = []string{"aliyun", "tencentcloud", "volcengine", "huaweicloud"}
	}

	conf, _, err := redc.ReadConfig(redc.ActiveConfigPath)
	if err != nil {
		return nil, err
	}

	results := make([]BalanceInfo, 0, len(providers))
	for _, p := range providers {
		result := BalanceInfo{
			Provider:  p,
			Amount:    "-",
			Currency:  "-",
			UpdatedAt: time.Now().Format("2006-01-02 15:04:05"),
		}
		switch p {
		case "aliyun":
			amount, currency, err := redc.QueryAliyunBalance(conf.Providers.Alicloud.AccessKey, conf.Providers.Alicloud.SecretKey, conf.Providers.Alicloud.Region)
			if err != nil {
				result.Error = err.Error()
			} else {
				result.Amount = amount
				result.Currency = currency
			}
		case "tencentcloud":
			amount, currency, err := redc.QueryTencentBalance(conf.Providers.Tencentcloud.SecretId, conf.Providers.Tencentcloud.SecretKey, conf.Providers.Tencentcloud.Region)
			if err != nil {
				result.Error = err.Error()
			} else {
				result.Amount = amount
				result.Currency = currency
			}
		case "volcengine":
			amount, currency, err := redc.QueryVolcengineBalance(conf.Providers.Volcengine.AccessKey, conf.Providers.Volcengine.SecretKey, conf.Providers.Volcengine.Region)
			if err != nil {
				result.Error = err.Error()
			} else {
				result.Amount = amount
				result.Currency = currency
			}
		case "huaweicloud":
			amount, currency, err := redc.QueryHuaweiBalance(conf.Providers.Huaweicloud.AccessKey, conf.Providers.Huaweicloud.SecretKey, conf.Providers.Huaweicloud.Region)
			if err != nil {
				result.Error = err.Error()
			} else {
				result.Amount = amount
				result.Currency = currency
			}
		default:
			result.Error = "不支持的云厂商"
		}
		results = append(results, result)
	}
	return results, nil
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
			}
		}
	}
	
	// Convert map to slice
	result := make([]TemplateVariable, 0, len(variables))
	for _, v := range variables {
		v.Required = true
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
			defaultRaw := strings.TrimSpace(matches[1])
			currentVar.DefaultValue = strings.Trim(defaultRaw, `"`)
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
			a.emitRefresh()
		}()
		
		a.emitLog(fmt.Sprintf("正在启动场景: %s", caseName))
		if err := c.TfApply(); err != nil {
			a.emitLog(fmt.Sprintf("启动失败: %v", err))
			if a.notificationMgr != nil {
				a.notificationMgr.SendSceneFailed(caseName, "启动")
			}
			return
		}
		a.emitLog(fmt.Sprintf("场景启动成功: %s", caseName))
		
		if a.notificationMgr != nil {
			a.notificationMgr.SendSceneStarted(caseName)
		}
		
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
			a.emitRefresh()
		}()
		
		a.emitLog(fmt.Sprintf("正在停止场景: %s", c.Name))
		if err := c.Stop(); err != nil {
			a.emitLog(fmt.Sprintf("停止失败: %v", err))
			if a.notificationMgr != nil {
				a.notificationMgr.SendSceneFailed(c.Name, "停止")
			}
			return
		}
		a.emitLog(fmt.Sprintf("场景停止成功: %s", c.Name))
		
		if a.notificationMgr != nil {
			a.notificationMgr.SendSceneStopped(c.Name)
		}
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

	// Add clash config paths if generated by module hooks
	// Only add these outputs if the case actually uses the CLASH_CONFIG_R2 module
	if c.Module != "" && strings.Contains(c.Module, "CLASH_CONFIG_R2") {
		tfvarsPath := filepath.Join(c.Path, "terraform.tfvars")
		if _, err := os.Stat(tfvarsPath); err == nil {
			if tfvars, err := parseTfvars(tfvarsPath); err == nil {
				fileName := strings.TrimSpace(tfvars["filename"])
				if fileName == "" {
					fileName = "default-config.yaml"
				}
				localConfig := filepath.Join(c.Path, "config.yaml")
				if _, err := os.Stat(localConfig); err == nil {
					result["clash_config_local"] = localConfig
				}
				bucketName := strings.TrimSpace(tfvars["buckets_name"])
				if bucketName == "" {
					bucketName = "test"
				}
				bucketPath := strings.Trim(tfvars["buckets_path"], "/")
				r2Path := fmt.Sprintf("r2:%s/%s", bucketName, fileName)
				if bucketPath != "" {
					r2Path = fmt.Sprintf("r2:%s/%s/%s", bucketName, bucketPath, fileName)
				}
				result["clash_config_r2"] = r2Path
			}
		}
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

// TemplateRecommendation represents a template recommendation result
type TemplateRecommendation struct {
	Template    string   `json:"template"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Match       int      `json:"match"`
	Tags        []string `json:"tags"`
	Provider    string   `json:"provider"`
	Version     string   `json:"version"`
	Installed   bool     `json:"installed"`
}

// RecommendTemplates searches and recommends templates based on user query
func (a *App) RecommendTemplates(query string) ([]TemplateRecommendation, error) {
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("搜索关键词不能为空")
	}

	opts := redc.PullOptions{
		RegistryURL: "https://redc.wgpsec.org",
		Timeout:     30 * time.Second,
	}

	results, err := redc.Search(context.Background(), query, opts)
	if err != nil {
		return nil, fmt.Errorf("搜索失败: %v", err)
	}

	localTemplates, _ := redc.ListLocalTemplates()
	installedMap := make(map[string]bool)
	for _, t := range localTemplates {
		installedMap[t.Name] = true
	}

	recommendations := make([]TemplateRecommendation, 0, len(results))
	for _, r := range results {
		maxScore := 1000
		if len(results) > 0 && results[0].Score > 0 {
			maxScore = results[0].Score
		}
		matchPercent := 50
		if r.Score >= maxScore {
			matchPercent = 95
		} else if r.Score > 0 {
			matchPercent = 50 + (r.Score*45)/maxScore
		}
		if matchPercent > 100 {
			matchPercent = 100
		}

		tags := []string{r.Provider}
		if r.Author != "" {
			tags = append(tags, r.Author)
		}

		name := r.Key
		if parts := strings.Split(r.Key, "/"); len(parts) == 2 {
			name = parts[1]
		}

		recommendations = append(recommendations, TemplateRecommendation{
			Template:    r.Key,
			Name:        name,
			Description: r.Description,
			Match:       matchPercent,
			Tags:        tags,
			Provider:    r.Provider,
			Version:     r.Version,
			Installed:   installedMap[r.Key],
		})
	}

	if len(recommendations) > 10 {
		recommendations = recommendations[:10]
	}

	return recommendations, nil
}

// AIRecommendTemplates uses AI to recommend templates based on user query with streaming
func (a *App) AIRecommendTemplates(query string) error {
	if strings.TrimSpace(query) == "" {
		return fmt.Errorf("搜索关键词不能为空")
	}

	// Get active profile and AI config
	profile, err := redc.GetActiveProfile()
	if err != nil || profile.AIConfig == nil {
		return fmt.Errorf("请先配置 AI 服务")
	}

	aiConfig := profile.AIConfig
	if aiConfig.APIKey == "" || aiConfig.BaseURL == "" || aiConfig.Model == "" {
		return fmt.Errorf("AI 配置不完整，请检查 API Key、Base URL 和 Model")
	}

	// Get available templates
	localTemplates, _ := redc.ListLocalTemplates()
	templateList := make([]string, 0, len(localTemplates))
	for _, t := range localTemplates {
		templateList = append(templateList, fmt.Sprintf("- %s: %s", t.Name, t.Description))
	}

	// Create AI client
	client := ai.NewClient(aiConfig.Provider, aiConfig.APIKey, aiConfig.BaseURL, aiConfig.Model)

	// Prepare system prompt
	systemPrompt := `你是一个云资源场景推荐助手。用户会描述他们的需求，你需要根据可用的模板列表推荐最合适的场景。

可用的模板列表：
` + strings.Join(templateList, "\n") + `

请根据用户需求，推荐最合适的模板，并说明推荐理由。如果没有完全匹配的模板，可以推荐相近的模板并说明如何调整使用。

请用简洁、友好的语言回复，直接给出推荐结果和理由。`

	messages := []ai.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: query},
	}

	// Stream response to frontend with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	
	err = client.ChatStream(ctx, messages, func(chunk string) error {
		// Emit chunk to frontend
		runtime.EventsEmit(a.ctx, "ai-recommend-chunk", chunk)
		return nil
	})

	if err != nil {
		return fmt.Errorf("AI 推荐失败: %v", err)
	}

	// Emit completion event
	runtime.EventsEmit(a.ctx, "ai-recommend-complete", true)
	return nil
}

// AICostOptimization uses AI to analyze running cases and provide cost optimization suggestions
func (a *App) AICostOptimization() error {
	// Get active profile and AI config
	profile, err := redc.GetActiveProfile()
	if err != nil || profile.AIConfig == nil {
		return fmt.Errorf("请先配置 AI 服务")
	}

	aiConfig := profile.AIConfig
	if aiConfig.APIKey == "" || aiConfig.BaseURL == "" || aiConfig.Model == "" {
		return fmt.Errorf("AI 配置不完整，请检查 API Key、Base URL 和 Model")
	}

	// Get running cases and their cost information
	a.mu.Lock()
	project := a.project
	pricingService := a.pricingService
	costCalculator := a.costCalculator
	logMgr := a.logMgr
	a.mu.Unlock()

	if project == nil {
		return fmt.Errorf("项目未加载")
	}

	if pricingService == nil || costCalculator == nil {
		return fmt.Errorf("成本估算服务未初始化")
	}

	cases, err := redc.LoadProjectCases(project.ProjectName)
	if err != nil {
		return fmt.Errorf("加载 case 列表失败: %v", err)
	}

	// Log analysis start
	if logMgr != nil {
		if logger, logErr := logMgr.NewServiceLogger("cost-optimization"); logErr == nil {
			logger.Write([]byte(fmt.Sprintf("[INFO] Starting AI cost optimization analysis\n")))
			logger.Write([]byte(fmt.Sprintf("[INFO] Total cases: %d\n", len(cases))))
			logger.Close()
		}
	}

	// Collect running cases with cost information
	var caseInfoList []string
	runningCount := 0
	
	for _, c := range cases {
		if c.State != redc.StateRunning {
			continue
		}
		runningCount++

		// Log case processing
		if logMgr != nil {
			if logger, logErr := logMgr.NewServiceLogger("cost-optimization"); logErr == nil {
				logger.Write([]byte(fmt.Sprintf("[INFO] Processing case: %s (path: %s)\n", c.Name, c.Path)))
				logger.Close()
			}
		}

		// Check path
		if c.Path == "" {
			if logMgr != nil {
				if logger, logErr := logMgr.NewServiceLogger("cost-optimization"); logErr == nil {
					logger.Write([]byte(fmt.Sprintf("[WARN] Case %s has empty path, skipping\n", c.Name)))
					logger.Close()
				}
			}
			caseInfo := fmt.Sprintf(`- **%s**
  - 模板: %s
  - 状态: 运行中
  - 说明: 场景路径为空
  - 建议: 请检查场景配置`, c.Name, c.Module)
			caseInfoList = append(caseInfoList, caseInfo)
			continue
		}

		// Get terraform state using TfStatus (proper way)
		state, err := redc.TfStatus(c.Path)
		if err != nil {
			// Log error
			if logMgr != nil {
				if logger, logErr := logMgr.NewServiceLogger("cost-optimization"); logErr == nil {
					logger.Write([]byte(fmt.Sprintf("[ERROR] Failed to get terraform state for %s: %v\n", c.Name, err)))
					logger.Close()
				}
			}
			caseInfo := fmt.Sprintf(`- **%s**
  - 模板: %s
  - 状态: 运行中
  - 说明: 状态获取失败 (%v)
  - 建议: 请检查 Terraform 是否正确安装，场景是否已完成部署`, c.Name, c.Module, err)
			caseInfoList = append(caseInfoList, caseInfo)
			continue
		}

		// Check if state is valid
		if state == nil || state.Values == nil {
			if logMgr != nil {
				if logger, logErr := logMgr.NewServiceLogger("cost-optimization"); logErr == nil {
					logger.Write([]byte(fmt.Sprintf("[WARN] Case %s has nil state or values, skipping\n", c.Name)))
					logger.Close()
				}
			}
			caseInfo := fmt.Sprintf(`- **%s**
  - 模板: %s
  - 状态: 运行中
  - 说明: 状态数据为空
  - 建议: 该场景可能尚未创建资源`, c.Name, c.Module)
			caseInfoList = append(caseInfoList, caseInfo)
			continue
		}

		// Extract resources
		resources := extractResourcesFromState(state)
		if resources == nil || len(resources.Resources) == 0 {
			if logMgr != nil {
				if logger, logErr := logMgr.NewServiceLogger("cost-optimization"); logErr == nil {
					logger.Write([]byte(fmt.Sprintf("[WARN] Case %s has no resources\n", c.Name)))
					logger.Close()
				}
			}
			// Provide basic info even if no resources found
			caseInfo := fmt.Sprintf(`- **%s**
  - 模板: %s
  - 状态: 运行中
  - 说明: 未找到资源信息
  - 建议: 该场景可能尚未创建资源，或资源已被销毁`, c.Name, c.Module)
			caseInfoList = append(caseInfoList, caseInfo)
			continue
		}

		if logMgr != nil {
			if logger, logErr := logMgr.NewServiceLogger("cost-optimization"); logErr == nil {
				logger.Write([]byte(fmt.Sprintf("[INFO] Case %s has %d resources\n", c.Name, len(resources.Resources))))
				logger.Close()
			}
		}

		// Calculate cost
		estimate, err := costCalculator.CalculateCost(resources, pricingService)
		if err != nil {
			if logMgr != nil {
				if logger, logErr := logMgr.NewServiceLogger("cost-optimization"); logErr == nil {
					logger.Write([]byte(fmt.Sprintf("[ERROR] Failed to calculate cost for %s: %v\n", c.Name, err)))
					logger.Close()
				}
			}
			// Provide resource info even if cost calculation fails
			var resourceList []string
			for _, r := range resources.Resources {
				resourceList = append(resourceList, fmt.Sprintf("  - %s (%s)", r.Name, r.Type))
			}
			caseInfo := fmt.Sprintf(`- **%s**
  - 模板: %s
  - 状态: 运行中
  - 资源数量: %d
  - 资源列表:
%s
  - 说明: 成本计算失败 (%v)
  - 建议: 请检查定价数据是否可用`, c.Name, c.Module, len(resources.Resources), strings.Join(resourceList, "\n"), err)
			caseInfoList = append(caseInfoList, caseInfo)
			continue
		}

		if logMgr != nil {
			if logger, logErr := logMgr.NewServiceLogger("cost-optimization"); logErr == nil {
				logger.Write([]byte(fmt.Sprintf("[INFO] Cost calculated for %s: ¥%.2f/month\n", c.Name, estimate.TotalMonthlyCost)))
				logger.Close()
			}
		}

		// Build case info with full details
		var resourceDetails []string
		for _, rb := range estimate.Breakdown {
			if rb.TotalMonthly > 0 {
				resourceDetails = append(resourceDetails, fmt.Sprintf("  - %s (%s): ¥%.2f/月", 
					rb.ResourceName, rb.ResourceType, rb.TotalMonthly))
			} else if !rb.Available {
				resourceDetails = append(resourceDetails, fmt.Sprintf("  - %s (%s): 定价不可用", 
					rb.ResourceName, rb.ResourceType))
			}
		}

		// Get provider from first resource
		provider := "未知"
		if len(estimate.Breakdown) > 0 {
			provider = estimate.Breakdown[0].Provider
		}

		caseInfo := fmt.Sprintf(`- **%s**
  - 模板: %s
  - 云服务商: %s
  - 月度成本: ¥%.2f
  - 资源数量: %d
  - 资源详情:
%s`, c.Name, c.Module, provider, estimate.TotalMonthlyCost, len(estimate.Breakdown), strings.Join(resourceDetails, "\n"))

		caseInfoList = append(caseInfoList, caseInfo)
	}

	if runningCount == 0 {
		return fmt.Errorf("当前没有运行中的场景")
	}

	// Create AI client
	client := ai.NewClient(aiConfig.Provider, aiConfig.APIKey, aiConfig.BaseURL, aiConfig.Model)

	// Prepare system prompt
	systemPrompt := `你是一个云成本优化专家。用户会提供当前运行中的云资源场景及其成本信息，你需要分析并提供成本优化建议。

**重要说明**：
- 某些场景可能因为状态文件问题无法获取完整信息
- 对于信息不完整的场景，请基于已知信息提供方向性建议
- 对于有完整成本信息的场景，请提供详细的优化建议

**分析维度**：
1. **实例规格优化**：是否可以降低配置或使用更经济的实例类型
2. **使用模式优化**：是否可以使用竞价实例、预留实例、定时开关机等策略
3. **资源利用率**：识别可能的资源浪费（如过度配置、闲置资源）
4. **存储优化**：存储类型是否合理，是否有优化空间
5. **网络优化**：带宽配置是否合理

**输出格式**：
对每个场景，请提供：
- 当前状态分析
- 具体的优化建议（可操作的）
- 预计可节省的成本（如果有成本数据）
- 优化的优先级（高/中/低）

**特殊情况处理**：
- 如果场景状态文件读取失败，建议检查部署状态
- 如果无法获取成本信息，提供通用的优化方向
- 如果资源信息不完整，基于模板类型给出建议

请用清晰、专业的语言回复，给出实用的建议。`

	casesInfo := strings.Join(caseInfoList, "\n\n")
	userPrompt := fmt.Sprintf(`请分析以下 %d 个运行中的云资源场景，并提供成本优化建议：

%s

请为每个场景提供详细的优化建议。`, runningCount, casesInfo)

	messages := []ai.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	// Stream response to frontend with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	
	err = client.ChatStream(ctx, messages, func(chunk string) error {
		// Emit chunk to frontend
		runtime.EventsEmit(a.ctx, "ai-cost-chunk", chunk)
		return nil
	})

	if err != nil {
		return fmt.Errorf("AI 成本分析失败: %v", err)
	}

	// Emit completion event
	runtime.EventsEmit(a.ctx, "ai-cost-complete", true)
	return nil
}

// CopyTemplate creates an editable local copy of a template
func (a *App) CopyTemplate(sourceName string, targetName string) error {
	if err := redc.CopyTemplate(sourceName, targetName); err != nil {
		a.emitLog(fmt.Sprintf("模板复制失败: %v", err))
		return err
	}
	a.emitLog(fmt.Sprintf("模板复制成功: %s -> %s", sourceName, targetName))
	a.emitRefresh()
	return nil
}

// GetTemplateFiles reads editable files from a template directory
func (a *App) GetTemplateFiles(templateName string) (map[string]string, error) {
	path, err := redc.GetTemplatePath(templateName)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	files := make(map[string]string)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if name == "case.json" || name == "terraform.tfvars" || strings.HasSuffix(name, ".tf") {
			data, err := os.ReadFile(filepath.Join(path, name))
			if err != nil {
				return nil, err
			}
			files[name] = string(data)
		}
	}
	return files, nil
}

// SaveTemplateFiles writes editable files to a template directory
func (a *App) SaveTemplateFiles(templateName string, files map[string]string) error {
	path, err := redc.GetTemplatePath(templateName)
	if err != nil {
		return err
	}
	for name, content := range files {
		if name == "case.json" || name == "terraform.tfvars" || strings.HasSuffix(name, ".tf") {
			if err := os.WriteFile(filepath.Join(path, name), []byte(content), 0644); err != nil {
				return err
			}
		}
	}
	a.emitLog(fmt.Sprintf("模板保存成功: %s", templateName))
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

// GetCostEstimate calculates cost estimate for a template
func (a *App) GetCostEstimate(templateName string, variables map[string]string) (*cost.CostEstimate, error) {
	a.mu.Lock()
	pricingService := a.pricingService
	costCalculator := a.costCalculator
	logMgr := a.logMgr
	a.mu.Unlock()

	// Validate that cost estimation components are initialized
	if pricingService == nil || costCalculator == nil {
		err := fmt.Errorf("成本估算服务未初始化")
		if logMgr != nil {
			if logger, logErr := logMgr.NewServiceLogger("cost-estimation"); logErr == nil {
				logger.Write([]byte(fmt.Sprintf("[ERROR] Cost estimation service not initialized for template: %s\n", templateName)))
				logger.Close()
			}
		}
		return nil, err
	}

	// Log the start of cost estimation
	if logMgr != nil {
		if logger, logErr := logMgr.NewServiceLogger("cost-estimation"); logErr == nil {
			logger.Write([]byte(fmt.Sprintf("[INFO] Starting cost estimation for template: %s\n", templateName)))
			if len(variables) > 0 {
				logger.Write([]byte(fmt.Sprintf("[INFO] Variables provided: %d\n", len(variables))))
			}
			logger.Close()
		}
	}

	// 1. Get template path
	templatePath, err := redc.GetTemplatePath(templateName)
	if err != nil {
		// Log template not found error with context
		if logMgr != nil {
			if logger, logErr := logMgr.NewServiceLogger("cost-estimation"); logErr == nil {
				logger.Write([]byte(fmt.Sprintf("[ERROR] Template not found: %s, error: %v\n", templateName, err)))
				logger.Close()
			}
		}
		return nil, fmt.Errorf("模板未找到: %w", err)
	}

	// Log successful template path resolution
	if logMgr != nil {
		if logger, logErr := logMgr.NewServiceLogger("cost-estimation"); logErr == nil {
			logger.Write([]byte(fmt.Sprintf("[INFO] Template path resolved: %s\n", templatePath)))
			logger.Close()
		}
	}

	// 2. Parse template to extract resources
	resources, err := cost.ParseTemplate(templatePath, variables)
	if err != nil {
		// Log parsing error with detailed context
		if logMgr != nil {
			if logger, logErr := logMgr.NewServiceLogger("cost-estimation"); logErr == nil {
				logger.Write([]byte(fmt.Sprintf("[ERROR] Template parsing failed for: %s\n", templateName)))
				logger.Write([]byte(fmt.Sprintf("[ERROR] Template path: %s\n", templatePath)))
				logger.Write([]byte(fmt.Sprintf("[ERROR] Parse error: %v\n", err)))
				logger.Close()
			}
		}
		return nil, fmt.Errorf("模板解析失败: %w", err)
	}

	// Log successful parsing with resource count
	if logMgr != nil {
		if logger, logErr := logMgr.NewServiceLogger("cost-estimation"); logErr == nil {
			logger.Write([]byte(fmt.Sprintf("[INFO] Template parsed successfully: %d resources found\n", len(resources.Resources))))
			if resources.Provider != "" {
				logger.Write([]byte(fmt.Sprintf("[INFO] Primary provider: %s\n", resources.Provider)))
			}
			if resources.Region != "" {
				logger.Write([]byte(fmt.Sprintf("[INFO] Primary region: %s\n", resources.Region)))
			}
			logger.Close()
		}
	}

	// 3. Calculate costs
	estimate, err := costCalculator.CalculateCost(resources, pricingService)
	if err != nil {
		// Log calculation error with context
		if logMgr != nil {
			if logger, logErr := logMgr.NewServiceLogger("cost-estimation"); logErr == nil {
				logger.Write([]byte(fmt.Sprintf("[ERROR] Cost calculation failed for template: %s\n", templateName)))
				logger.Write([]byte(fmt.Sprintf("[ERROR] Resource count: %d\n", len(resources.Resources))))
				logger.Write([]byte(fmt.Sprintf("[ERROR] Calculation error: %v\n", err)))
				logger.Close()
			}
		}
		return nil, fmt.Errorf("成本计算失败: %w", err)
	}

	// Log successful cost estimation with summary
	if logMgr != nil {
		if logger, logErr := logMgr.NewServiceLogger("cost-estimation"); logErr == nil {
			logger.Write([]byte(fmt.Sprintf("[INFO] Cost estimation completed successfully for template: %s\n", templateName)))
			logger.Write([]byte(fmt.Sprintf("[INFO] Total hourly cost: %.4f %s\n", estimate.TotalHourlyCost, estimate.Currency)))
			logger.Write([]byte(fmt.Sprintf("[INFO] Total monthly cost: %.2f %s\n", estimate.TotalMonthlyCost, estimate.Currency)))
			logger.Write([]byte(fmt.Sprintf("[INFO] Resources in breakdown: %d\n", len(estimate.Breakdown))))
			if estimate.UnavailableCount > 0 {
				logger.Write([]byte(fmt.Sprintf("[WARN] Resources with unavailable pricing: %d\n", estimate.UnavailableCount)))
			}
			if len(estimate.Warnings) > 0 {
				logger.Write([]byte(fmt.Sprintf("[WARN] Warnings generated: %d\n", len(estimate.Warnings))))
				for i, warning := range estimate.Warnings {
					logger.Write([]byte(fmt.Sprintf("[WARN]   %d. %s\n", i+1, warning)))
				}
			}
			logger.Close()
		}
	}

	return estimate, nil
}

// SelectComposeFile opens a file dialog to select a compose file
func (a *App) SelectComposeFile() (string, error) {
	filePath, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择 Compose 文件",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "YAML 文件 (*.yaml, *.yml)",
				Pattern:     "*.yaml;*.yml",
			},
			{
				DisplayName: "所有文件 (*.*)",
				Pattern:     "*.*",
			},
		},
	})
	if err != nil {
		return "", err
	}
	return filePath, nil
}

// GetTotalRuntime calculates total runtime of all running cases
func (a *App) GetTotalRuntime() (string, error) {
	a.mu.Lock()
	project := a.project
	logMgr := a.logMgr
	a.mu.Unlock()

	if project == nil {
		return "0h", fmt.Errorf("项目未加载")
	}

	cases, err := redc.LoadProjectCases(project.ProjectName)
	if err != nil {
		return "0h", err
	}

	totalMinutes := 0
	now := time.Now()

	for _, c := range cases {
		if c.State == redc.StateRunning {
			// Try multiple time formats to parse StateTime
			var stateTime time.Time
			var parseErr error
			
			// Try RFC3339 format first (e.g., "2006-01-02T15:04:05Z" or "2006-01-02T15:04:05+08:00")
			stateTime, parseErr = time.Parse(time.RFC3339, c.StateTime)
			if parseErr != nil {
				// Try format with timezone (e.g., "2006-01-02 15:04:05 +08:00")
				stateTime, parseErr = time.Parse("2006-01-02 15:04:05 -07:00", c.StateTime)
				if parseErr != nil {
					// Try format without timezone (e.g., "2006-01-02 15:04:05")
					// Assume local timezone
					stateTime, parseErr = time.ParseInLocation("2006-01-02 15:04:05", c.StateTime, time.Local)
					if parseErr != nil {
						// Log parsing error for debugging
						if logMgr != nil {
							if logger, logErr := logMgr.NewServiceLogger("runtime"); logErr == nil {
								logger.Write([]byte(fmt.Sprintf("[WARN] Failed to parse StateTime for case %s: %s (error: %v)\n", c.Name, c.StateTime, parseErr)))
								logger.Close()
							}
						}
						continue
					}
				}
			}
			
			// COMPATIBILITY FIX: If the time ends with 'Z' (UTC) but results in negative duration,
			// it's likely a bug where local time was incorrectly stored as UTC.
			// Re-parse it as local time.
			duration := now.Sub(stateTime)
			if duration < 0 && strings.HasSuffix(c.StateTime, "Z") {
				// Extract the time part without 'Z' and parse as local time
				timeStr := strings.TrimSuffix(c.StateTime, "Z")
				timeStr = strings.Replace(timeStr, "T", " ", 1)
				stateTime, parseErr = time.ParseInLocation("2006-01-02 15:04:05", timeStr, time.Local)
				if parseErr == nil {
					duration = now.Sub(stateTime)
				}
			}
			
			minutes := int(duration.Minutes())
			
			// Log for debugging
			if logMgr != nil {
				if logger, logErr := logMgr.NewServiceLogger("runtime"); logErr == nil {
					logger.Write([]byte(fmt.Sprintf("[DEBUG] Case %s: StateTime=%s, Now=%s, Duration=%v, Minutes=%d\n", 
						c.Name, stateTime.Format(time.RFC3339), now.Format(time.RFC3339), duration, minutes)))
					logger.Close()
				}
			}
			
			// Only add positive durations (ignore cases with future StateTime)
			if minutes > 0 {
				totalMinutes += minutes
			}
		}
	}

	// Format as hours
	hours := totalMinutes / 60
	return fmt.Sprintf("%dh", hours), nil
}

// GetPredictedMonthlyCost calculates predicted monthly cost for all running cases
func (a *App) GetPredictedMonthlyCost() (string, error) {
	a.mu.Lock()
	project := a.project
	pricingService := a.pricingService
	costCalculator := a.costCalculator
	logMgr := a.logMgr
	a.mu.Unlock()

	if project == nil {
		return "¥0.00", fmt.Errorf("项目未加载")
	}

	if pricingService == nil || costCalculator == nil {
		return "¥0.00", fmt.Errorf("成本估算服务未初始化")
	}

	cases, err := redc.LoadProjectCases(project.ProjectName)
	if err != nil {
		return "¥0.00", err
	}

	totalMonthlyCost := 0.0
	currency := "CNY"
	runningCount := 0

	// Log start
	if logMgr != nil {
		if logger, logErr := logMgr.NewServiceLogger("cost-prediction"); logErr == nil {
			logger.Write([]byte(fmt.Sprintf("[INFO] Starting predicted monthly cost calculation\n")))
			logger.Write([]byte(fmt.Sprintf("[INFO] Total cases: %d\n", len(cases))))
			logger.Close()
		}
	}

	// Calculate cost for each running case
	for _, c := range cases {
		if c.State != redc.StateRunning {
			continue
		}
		runningCount++
		
		if logMgr != nil {
			if logger, logErr := logMgr.NewServiceLogger("cost-prediction"); logErr == nil {
				logger.Write([]byte(fmt.Sprintf("[INFO] Processing running case: %s (path: %s)\n", c.Name, c.Path)))
				logger.Close()
			}
		}
		
		if c.Path == "" {
			if logMgr != nil {
				if logger, logErr := logMgr.NewServiceLogger("cost-prediction"); logErr == nil {
					logger.Write([]byte(fmt.Sprintf("[WARN] Case %s has empty path, skipping\n", c.Name)))
					logger.Close()
				}
			}
			continue
		}

		// Get terraform state
		state, err := redc.TfStatus(c.Path)
		if err != nil {
			if logMgr != nil {
				if logger, logErr := logMgr.NewServiceLogger("cost-prediction"); logErr == nil {
					logger.Write([]byte(fmt.Sprintf("[ERROR] Failed to get terraform state for %s: %v\n", c.Name, err)))
					logger.Close()
				}
			}
			continue
		}
		
		if state == nil || state.Values == nil {
			if logMgr != nil {
				if logger, logErr := logMgr.NewServiceLogger("cost-prediction"); logErr == nil {
					logger.Write([]byte(fmt.Sprintf("[WARN] Case %s has nil state or values, skipping\n", c.Name)))
					logger.Close()
				}
			}
			continue
		}

		// Extract resources from state and convert to cost.Resource format
		resources := extractResourcesFromState(state)
		if len(resources.Resources) == 0 {
			if logMgr != nil {
				if logger, logErr := logMgr.NewServiceLogger("cost-prediction"); logErr == nil {
					logger.Write([]byte(fmt.Sprintf("[WARN] Case %s has no resources, skipping\n", c.Name)))
					logger.Close()
				}
			}
			continue
		}

		if logMgr != nil {
			if logger, logErr := logMgr.NewServiceLogger("cost-prediction"); logErr == nil {
				logger.Write([]byte(fmt.Sprintf("[INFO] Case %s has %d resources, calculating cost...\n", c.Name, len(resources.Resources))))
				logger.Write([]byte(fmt.Sprintf("[DEBUG] Provider: %s, Region: %s\n", resources.Provider, resources.Region)))
				// Log first few resources for debugging
				for i, res := range resources.Resources {
					if i >= 3 {
						break
					}
					instanceType := "N/A"
					if it, ok := res.Attributes["instance_type"].(string); ok {
						instanceType = it
					}
					region := "N/A"
					if r, ok := res.Attributes["region"].(string); ok {
						region = r
					}
					zone := "N/A"
					if z, ok := res.Attributes["zone"].(string); ok {
						zone = z
					}
					availabilityZone := "N/A"
					if az, ok := res.Attributes["availability_zone"].(string); ok {
						availabilityZone = az
					}
					zoneId := "N/A"
					if zid, ok := res.Attributes["zone_id"].(string); ok {
						zoneId = zid
					}
					logger.Write([]byte(fmt.Sprintf("[DEBUG] Resource %d: Type=%s, Name=%s, InstanceType=%s, Region=%s, Zone=%s, AvailabilityZone=%s, ZoneId=%s, ResourceRegion=%s\n", 
						i+1, res.Type, res.Name, instanceType, region, zone, availabilityZone, zoneId, res.Region)))
				}
				logger.Close()
			}
		}

		// Calculate cost for this case
		estimate, err := costCalculator.CalculateCost(resources, pricingService)
		if err != nil {
			if logMgr != nil {
				if logger, logErr := logMgr.NewServiceLogger("cost-prediction"); logErr == nil {
					logger.Write([]byte(fmt.Sprintf("[ERROR] Failed to calculate cost for %s: %v\n", c.Name, err)))
					logger.Close()
				}
			}
			continue
		}

		if logMgr != nil {
			if logger, logErr := logMgr.NewServiceLogger("cost-prediction"); logErr == nil {
				logger.Write([]byte(fmt.Sprintf("[INFO] Case %s monthly cost: %.2f %s\n", c.Name, estimate.TotalMonthlyCost, estimate.Currency)))
				logger.Close()
			}
		}

		totalMonthlyCost += estimate.TotalMonthlyCost
		if estimate.Currency != "" {
			currency = estimate.Currency
		}
	}

	// Log summary
	if logMgr != nil {
		if logger, logErr := logMgr.NewServiceLogger("cost-prediction"); logErr == nil {
			logger.Write([]byte(fmt.Sprintf("[INFO] Prediction complete - Running cases: %d, Total monthly cost: %.2f %s\n", runningCount, totalMonthlyCost, currency)))
			logger.Close()
		}
	}

	// Format the result
	var symbol string
	switch currency {
	case "CNY":
		symbol = "¥"
	case "USD":
		symbol = "$"
	default:
		symbol = currency + " "
	}

	return fmt.Sprintf("%s%.2f", symbol, totalMonthlyCost), nil
}

// extractResourcesFromState converts terraform state to cost.TemplateResources
func extractResourcesFromState(state *tfjson.State) *cost.TemplateResources {
	resources := &cost.TemplateResources{
		Resources: []cost.ResourceSpec{},
	}

	if state.Values == nil || state.Values.RootModule == nil {
		return resources
	}

	// Extract provider from first resource if available
	if len(state.Values.RootModule.Resources) > 0 {
		firstResource := state.Values.RootModule.Resources[0]
		if firstResource.ProviderName != "" {
			// Extract short provider name from full registry path
			// e.g., "registry.terraform.io/volcengine/volcengine" -> "volcengine"
			providerName := extractShortProviderName(firstResource.ProviderName)
			resources.Provider = providerName
		}
	}

	// Recursively extract resources from modules
	extractModuleResources(state.Values.RootModule, resources)
	
	// Try to extract region from any resource that has it
	// Some resources like security groups don't have region, but compute instances do
	for _, res := range resources.Resources {
		if region, ok := res.Attributes["region"].(string); ok && region != "" {
			resources.Region = region
			break
		} else if availabilityZone, ok := res.Attributes["availability_zone"].(string); ok && availabilityZone != "" {
			// Extract region from availability zone (e.g., "cn-beijing-a" -> "cn-beijing")
			if len(availabilityZone) > 2 {
				lastDash := strings.LastIndex(availabilityZone, "-")
				if lastDash > 0 {
					resources.Region = availabilityZone[:lastDash]
					break
				}
			}
		} else if zone, ok := res.Attributes["zone"].(string); ok && zone != "" {
			// Some providers use "zone" instead of "availability_zone"
			if len(zone) > 2 {
				lastDash := strings.LastIndex(zone, "-")
				if lastDash > 0 {
					resources.Region = zone[:lastDash]
					break
				}
			}
		} else if zoneId, ok := res.Attributes["zone_id"].(string); ok && zoneId != "" {
			// Volcengine uses "zone_id" (e.g., "cn-beijing-a" -> "cn-beijing")
			if len(zoneId) > 2 {
				lastDash := strings.LastIndex(zoneId, "-")
				if lastDash > 0 {
					resources.Region = zoneId[:lastDash]
					break
				}
			}
		}
	}

	return resources
}

// extractShortProviderName extracts the short provider name from full registry path
// e.g., "registry.terraform.io/volcengine/volcengine" -> "volcengine"
// e.g., "registry.terraform.io/aliyun/alicloud" -> "alicloud"
func extractShortProviderName(fullName string) string {
	// Split by "/"
	parts := strings.Split(fullName, "/")
	if len(parts) >= 3 {
		// Return the last part (provider name)
		return parts[len(parts)-1]
	}
	return fullName
}

// extractModuleResources recursively extracts resources from a terraform module
func extractModuleResources(module *tfjson.StateModule, resources *cost.TemplateResources) {
	if module == nil {
		return
	}

	// Extract resources from current module
	for _, res := range module.Resources {
		if res.Type == "" {
			continue
		}

		// Extract short provider name
		providerName := extractShortProviderName(res.ProviderName)

		// Convert state resource to cost.ResourceSpec
		costRes := cost.ResourceSpec{
			Type:       res.Type,
			Name:       res.Name,
			Provider:   providerName,
			Count:      1, // Default count
			Attributes: make(map[string]interface{}),
		}

		// Copy all attributes from state values
		if res.AttributeValues != nil {
			for key, value := range res.AttributeValues {
				costRes.Attributes[key] = value
			}
			
			// Extract region from resource attributes
			if region, ok := res.AttributeValues["region"].(string); ok && region != "" {
				costRes.Region = region
			} else if availabilityZone, ok := res.AttributeValues["availability_zone"].(string); ok && availabilityZone != "" {
				// Extract region from availability zone (e.g., "cn-beijing-a" -> "cn-beijing")
				if len(availabilityZone) > 2 {
					lastDash := strings.LastIndex(availabilityZone, "-")
					if lastDash > 0 {
						costRes.Region = availabilityZone[:lastDash]
					}
				}
			} else if zone, ok := res.AttributeValues["zone"].(string); ok && zone != "" {
				// Some providers use "zone" instead of "availability_zone"
				if len(zone) > 2 {
					lastDash := strings.LastIndex(zone, "-")
					if lastDash > 0 {
						costRes.Region = zone[:lastDash]
					}
				}
			} else if zoneId, ok := res.AttributeValues["zone_id"].(string); ok && zoneId != "" {
				// Volcengine uses "zone_id" (e.g., "cn-beijing-a" -> "cn-beijing")
				if len(zoneId) > 2 {
					lastDash := strings.LastIndex(zoneId, "-")
					if lastDash > 0 {
						costRes.Region = zoneId[:lastDash]
					}
				}
			}
			
			// Ensure instance_type is properly extracted for compute resources
			if res.Type == "alicloud_instance" || res.Type == "aws_instance" || 
			   res.Type == "tencentcloud_instance" || res.Type == "volcengine_ecs_instance" {
				// Make sure instance_type is available
				if instanceType, ok := res.AttributeValues["instance_type"].(string); ok && instanceType != "" {
					costRes.Attributes["instance_type"] = instanceType
				}
			}
		}

		resources.Resources = append(resources.Resources, costRes)
	}

	// Recursively process child modules
	for _, child := range module.ChildModules {
		extractModuleResources(child, resources)
	}
}

// ============================================================================
// SSH Operations API
// ============================================================================

// ExecCommandResult 执行命令的结果
type ExecCommandResult struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exitCode"`
	Success  bool   `json:"success"`
	Error    string `json:"error,omitempty"`
}

// ExecCommand 在指定场景上执行命令并返回结果
func (a *App) ExecCommand(caseID string, command string) ExecCommandResult {
	a.mu.Lock()
	project := a.project
	a.mu.Unlock()

	result := ExecCommandResult{}

	if project == nil {
		result.Error = "项目未加载"
		result.Success = false
		return result
	}

	c, err := project.GetCase(caseID)
	if err != nil {
		result.Error = fmt.Sprintf("找不到场景: %v", err)
		result.Success = false
		return result
	}

	sshConfig, err := c.GetSSHConfig()
	if err != nil {
		result.Error = fmt.Sprintf("获取 SSH 配置失败: %v", err)
		result.Success = false
		return result
	}

	client, err := sshutil.NewClient(sshConfig)
	if err != nil {
		result.Error = fmt.Sprintf("SSH 连接失败: %v", err)
		result.Success = false
		return result
	}
	defer client.Close()

	var stdoutBuf, stderrBuf strings.Builder
	session, err := client.NewSession()
	if err != nil {
		result.Error = fmt.Sprintf("创建 SSH 会话失败: %v", err)
		result.Success = false
		return result
	}
	defer session.Close()

	session.Stdout = &stdoutBuf
	session.Stderr = &stderrBuf

	err = session.Run(command)
	result.Stdout = stdoutBuf.String()
	result.Stderr = stderrBuf.String()

	if err != nil {
		if exitErr, ok := err.(*ssh.ExitError); ok {
			result.ExitCode = exitErr.ExitStatus()
		} else {
			result.ExitCode = -1
			result.Error = err.Error()
		}
		result.Success = false
	} else {
		result.ExitCode = 0
		result.Success = true
	}

	return result
}

// FileTransferResult 文件传输结果
type FileTransferResult struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// UploadFile 上传文件到远程服务器
func (a *App) UploadFile(caseID string, localPath string, remotePath string) FileTransferResult {
	a.mu.Lock()
	project := a.project
	a.mu.Unlock()

	result := FileTransferResult{}

	if project == nil {
		result.Error = "项目未加载"
		return result
	}

	c, err := project.GetCase(caseID)
	if err != nil {
		result.Error = fmt.Sprintf("找不到场景: %v", err)
		return result
	}

	sshConfig, err := c.GetSSHConfig()
	if err != nil {
		result.Error = fmt.Sprintf("获取 SSH 配置失败: %v", err)
		return result
	}

	client, err := sshutil.NewClient(sshConfig)
	if err != nil {
		result.Error = fmt.Sprintf("SSH 连接失败: %v", err)
		return result
	}
	defer client.Close()

	if err := client.Upload(localPath, remotePath); err != nil {
		result.Error = fmt.Sprintf("上传失败: %v", err)
		return result
	}

	result.Success = true
	return result
}

// DownloadFile 从远程服务器下载文件
func (a *App) DownloadFile(caseID string, remotePath string, localPath string) FileTransferResult {
	a.mu.Lock()
	project := a.project
	a.mu.Unlock()

	result := FileTransferResult{}

	if project == nil {
		result.Error = "项目未加载"
		return result
	}

	c, err := project.GetCase(caseID)
	if err != nil {
		result.Error = fmt.Sprintf("找不到场景: %v", err)
		return result
	}

	sshConfig, err := c.GetSSHConfig()
	if err != nil {
		result.Error = fmt.Sprintf("获取 SSH 配置失败: %v", err)
		return result
	}

	client, err := sshutil.NewClient(sshConfig)
	if err != nil {
		result.Error = fmt.Sprintf("SSH 连接失败: %v", err)
		return result
	}
	defer client.Close()

	if err := client.Download(remotePath, localPath); err != nil {
		result.Error = fmt.Sprintf("下载失败: %v", err)
		return result
	}

	result.Success = true
	return result
}

// SelectFile 打开文件选择对话框
func (a *App) SelectFile(title string) (string, error) {
	file, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: title,
	})
	return file, err
}

// SelectDirectory 打开目录选择对话框
func (a *App) SelectDirectory(title string) (string, error) {
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: title,
	})
	return dir, err
}

// SelectSaveFile 打开保存文件对话框
func (a *App) SelectSaveFile(title string, defaultFilename string) (string, error) {
	file, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           title,
		DefaultFilename: defaultFilename,
	})
	return file, err
}

// ListRemoteFiles 列出远程目录下的文件
// ============================================================================
// SSH Terminal & File Manager
// ============================================================================

// SSHTerminalSession 存储活动的终端会话
var (
	terminalSessions   = make(map[string]*sshutil.TerminalSession)
	terminalSessionsMu sync.Mutex
)

// StartSSHTerminal 启动 SSH 终端会话
func (a *App) StartSSHTerminal(caseID string, rows, cols int) (string, error) {
	a.mu.Lock()
	project := a.project
	a.mu.Unlock()

	if project == nil {
		return "", fmt.Errorf("项目未加载")
	}

	c, err := project.GetCase(caseID)
	if err != nil {
		return "", fmt.Errorf("找不到场景: %v", err)
	}

	sshConfig, err := c.GetSSHConfig()
	if err != nil {
		return "", fmt.Errorf("获取 SSH 配置失败: %v", err)
	}

	client, err := sshutil.NewClient(sshConfig)
	if err != nil {
		return "", fmt.Errorf("SSH 连接失败: %v", err)
	}

	// 创建终端会话
	session, err := client.NewTerminalSession(rows, cols)
	if err != nil {
		client.Close()
		return "", fmt.Errorf("创建终端会话失败: %v", err)
	}

	// 生成会话 ID
	sessionID := fmt.Sprintf("%s-%d", caseID, time.Now().Unix())

	// 存储会话
	terminalSessionsMu.Lock()
	terminalSessions[sessionID] = session
	terminalSessionsMu.Unlock()

	// 启动输出读取协程
	go a.readTerminalOutput(sessionID, session)

	return sessionID, nil
}

// readTerminalOutput 读取终端输出并发送到前端
func (a *App) readTerminalOutput(sessionID string, session *sshutil.TerminalSession) {
	buf := make([]byte, 4096)
	for {
		n, err := session.Read(buf)
		if err != nil {
			if err != io.EOF {
				runtime.EventsEmit(a.ctx, "terminal-error-"+sessionID, err.Error())
			}
			// 会话结束，清理
			terminalSessionsMu.Lock()
			delete(terminalSessions, sessionID)
			terminalSessionsMu.Unlock()
			runtime.EventsEmit(a.ctx, "terminal-closed-"+sessionID, true)
			break
		}

		if n > 0 {
			// 发送输出到前端
			runtime.EventsEmit(a.ctx, "terminal-output-"+sessionID, string(buf[:n]))
		}
	}
}

// WriteToTerminal 向终端写入数据
func (a *App) WriteToTerminal(sessionID string, data string) error {
	terminalSessionsMu.Lock()
	session, exists := terminalSessions[sessionID]
	terminalSessionsMu.Unlock()

	if !exists {
		return fmt.Errorf("终端会话不存在")
	}

	return session.Write([]byte(data))
}

// ResizeTerminal 调整终端大小
func (a *App) ResizeTerminal(sessionID string, rows, cols int) error {
	terminalSessionsMu.Lock()
	session, exists := terminalSessions[sessionID]
	terminalSessionsMu.Unlock()

	if !exists {
		return fmt.Errorf("终端会话不存在")
	}

	return session.Resize(rows, cols)
}

// CloseTerminal 关闭终端会话
func (a *App) CloseTerminal(sessionID string) error {
	terminalSessionsMu.Lock()
	session, exists := terminalSessions[sessionID]
	if exists {
		delete(terminalSessions, sessionID)
	}
	terminalSessionsMu.Unlock()

	if !exists {
		return nil // 已经关闭
	}

	return session.Close()
}

// ListRemoteFiles 列出远程目录文件
func (a *App) ListRemoteFiles(caseID string, remotePath string) ([]sshutil.FileInfo, error) {
	a.mu.Lock()
	project := a.project
	a.mu.Unlock()

	if project == nil {
		return nil, fmt.Errorf("项目未加载")
	}

	c, err := project.GetCase(caseID)
	if err != nil {
		return nil, fmt.Errorf("找不到场景: %v", err)
	}

	sshConfig, err := c.GetSSHConfig()
	if err != nil {
		return nil, fmt.Errorf("获取 SSH 配置失败: %v", err)
	}

	client, err := sshutil.NewClient(sshConfig)
	if err != nil {
		return nil, fmt.Errorf("SSH 连接失败: %v", err)
	}
	defer client.Close()

	return client.ListFiles(remotePath)
}

// CreateRemoteDirectory 创建远程目录
func (a *App) CreateRemoteDirectory(caseID string, remotePath string) error {
	a.mu.Lock()
	project := a.project
	a.mu.Unlock()

	if project == nil {
		return fmt.Errorf("项目未加载")
	}

	c, err := project.GetCase(caseID)
	if err != nil {
		return fmt.Errorf("找不到场景: %v", err)
	}

	sshConfig, err := c.GetSSHConfig()
	if err != nil {
		return fmt.Errorf("获取 SSH 配置失败: %v", err)
	}

	client, err := sshutil.NewClient(sshConfig)
	if err != nil {
		return fmt.Errorf("SSH 连接失败: %v", err)
	}
	defer client.Close()

	return client.CreateDirectory(remotePath)
}

// DeleteRemoteFile 删除远程文件或目录
func (a *App) DeleteRemoteFile(caseID string, remotePath string) error {
	a.mu.Lock()
	project := a.project
	a.mu.Unlock()

	if project == nil {
		return fmt.Errorf("项目未加载")
	}

	c, err := project.GetCase(caseID)
	if err != nil {
		return fmt.Errorf("找不到场景: %v", err)
	}

	sshConfig, err := c.GetSSHConfig()
	if err != nil {
		return fmt.Errorf("获取 SSH 配置失败: %v", err)
	}

	client, err := sshutil.NewClient(sshConfig)
	if err != nil {
		return fmt.Errorf("SSH 连接失败: %v", err)
	}
	defer client.Close()

	return client.DeleteFile(remotePath)
}

// RenameRemoteFile 重命名远程文件或目录
func (a *App) RenameRemoteFile(caseID string, oldPath, newPath string) error {
	a.mu.Lock()
	project := a.project
	a.mu.Unlock()

	if project == nil {
		return fmt.Errorf("项目未加载")
	}

	c, err := project.GetCase(caseID)
	if err != nil {
		return fmt.Errorf("找不到场景: %v", err)
	}

	sshConfig, err := c.GetSSHConfig()
	if err != nil {
		return fmt.Errorf("获取 SSH 配置失败: %v", err)
	}

	client, err := sshutil.NewClient(sshConfig)
	if err != nil {
		return fmt.Errorf("SSH 连接失败: %v", err)
	}
	defer client.Close()

	return client.RenameFile(oldPath, newPath)
}

// GetRemoteFileContent 获取远程文件内容（用于预览）
func (a *App) GetRemoteFileContent(caseID string, remotePath string) (string, error) {
	a.mu.Lock()
	project := a.project
	a.mu.Unlock()

	if project == nil {
		return "", fmt.Errorf("项目未加载")
	}

	c, err := project.GetCase(caseID)
	if err != nil {
		return "", fmt.Errorf("找不到场景: %v", err)
	}

	sshConfig, err := c.GetSSHConfig()
	if err != nil {
		return "", fmt.Errorf("获取 SSH 配置失败: %v", err)
	}

	client, err := sshutil.NewClient(sshConfig)
	if err != nil {
		return "", fmt.Errorf("SSH 连接失败: %v", err)
	}
	defer client.Close()

	// 限制预览文件大小为 1MB
	content, err := client.GetFileContent(remotePath, 1024*1024)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

// WriteRemoteFileContent 写入远程文件内容
func (a *App) WriteRemoteFileContent(caseID string, remotePath string, content string) error {
	a.mu.Lock()
	project := a.project
	a.mu.Unlock()

	if project == nil {
		return fmt.Errorf("项目未加载")
	}

	c, err := project.GetCase(caseID)
	if err != nil {
		return fmt.Errorf("找不到场景: %v", err)
	}

	sshConfig, err := c.GetSSHConfig()
	if err != nil {
		return fmt.Errorf("获取 SSH 配置失败: %v", err)
	}

	client, err := sshutil.NewClient(sshConfig)
	if err != nil {
		return fmt.Errorf("SSH 连接失败: %v", err)
	}
	defer client.Close()

	return client.WriteFileContent(remotePath, []byte(content))
}

// ============================================================================
// Task Scheduler APIs
// ============================================================================

// ScheduleTask 创建定时任务
func (a *App) ScheduleTask(caseID string, caseName string, action string, scheduledAt time.Time) (*redc.ScheduledTask, error) {
	a.mu.Lock()
	scheduler := a.taskScheduler
	a.mu.Unlock()

	if scheduler == nil {
		return nil, fmt.Errorf("任务调度器未初始化")
	}

	task, err := scheduler.AddTask(caseID, caseName, action, scheduledAt)
	if err != nil {
		return nil, err
	}

	// 发送通知
	a.emitLog(fmt.Sprintf("已创建定时任务: %s 将在 %s %s", caseName, scheduledAt.Format("2006-01-02 15:04:05"), action))

	return task, nil
}

// CancelScheduledTask 取消定时任务
func (a *App) CancelScheduledTask(taskID string) error {
	a.mu.Lock()
	scheduler := a.taskScheduler
	a.mu.Unlock()

	if scheduler == nil {
		return fmt.Errorf("任务调度器未初始化")
	}

	err := scheduler.CancelTask(taskID)
	if err != nil {
		return err
	}

	a.emitLog(fmt.Sprintf("已取消定时任务: %s", taskID))
	return nil
}

// GetScheduledTask 获取定时任务
func (a *App) GetScheduledTask(taskID string) (*redc.ScheduledTask, error) {
	a.mu.Lock()
	scheduler := a.taskScheduler
	a.mu.Unlock()

	if scheduler == nil {
		return nil, fmt.Errorf("任务调度器未初始化")
	}

	return scheduler.GetTask(taskID)
}

// ListScheduledTasks 列出所有定时任务
func (a *App) ListScheduledTasks() []*redc.ScheduledTask {
	a.mu.Lock()
	scheduler := a.taskScheduler
	a.mu.Unlock()

	if scheduler == nil {
		return []*redc.ScheduledTask{}
	}

	return scheduler.ListTasks()
}

// ListCaseScheduledTasks 列出指定场景的定时任务
func (a *App) ListCaseScheduledTasks(caseID string) []*redc.ScheduledTask {
	a.mu.Lock()
	scheduler := a.taskScheduler
	a.mu.Unlock()

	if scheduler == nil {
		return []*redc.ScheduledTask{}
	}

	return scheduler.ListTasksByCase(caseID)
}
