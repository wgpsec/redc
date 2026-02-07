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
	"red-cloud/mod/gologger"
	"red-cloud/mod/mcp"
	"red-cloud/mod/compose"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"github.com/projectdiscovery/gologger/levels"
	tfjson "github.com/hashicorp/terraform-json"
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
	if strings.Contains(content, "redc.wgpsec.org/tf-mirror/") {
		providers = append(providers, "wgpsec")
	}
	return providers
}

func terraformMirrorConfigContent(enabled bool, providers []string) string {
	var builder strings.Builder
	builder.WriteString("# Generated by redc-gui\n")
	builder.WriteString("plugin_cache_dir = \"$HOME/.terraform.d/plugin-cache\"\n\n")
	if !enabled || len(providers) == 0 {
		return builder.String()
	}

	providerSet := make(map[string]bool)
	for _, p := range providers {
		providerSet[p] = true
	}

	builder.WriteString("disable_checkpoint = true\n\n")
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
	if providerSet["wgpsec"] {
		builder.WriteString("  network_mirror {\n")
		builder.WriteString("    url = \"https://redc.wgpsec.org/tf-mirror/\"\n")
		builder.WriteString("    include = [\n")
		builder.WriteString("      \"registry.terraform.io/hashicorp/archive\",\n")
		builder.WriteString("      \"registry.terraform.io/hashicorp/external\",\n")
		builder.WriteString("      \"registry.terraform.io/hashicorp/http\",\n")
		builder.WriteString("      \"registry.terraform.io/hashicorp/local\",\n")
		builder.WriteString("      \"registry.terraform.io/hashicorp/null\",\n")
		builder.WriteString("      \"registry.terraform.io/hashicorp/random\",\n")
		builder.WriteString("      \"registry.terraform.io/hashicorp/time\",\n")
		builder.WriteString("      \"registry.terraform.io/hashicorp/tls\"\n")
		builder.WriteString("    ]\n")
		builder.WriteString("  }\n")
		excludes = append(excludes, "registry.terraform.io/hashicorp/archive", "registry.terraform.io/hashicorp/external", "registry.terraform.io/hashicorp/http", "registry.terraform.io/hashicorp/local", "registry.terraform.io/hashicorp/null", "registry.terraform.io/hashicorp/random", "registry.terraform.io/hashicorp/time", "registry.terraform.io/hashicorp/tls")
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
		{Name: "WGPSEC Mirror", URL: "https://redc.wgpsec.org/tf-mirror/"},
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
