package main

import (
	"encoding/json"
	"fmt"
	"github.com/projectdiscovery/gologger/levels"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"red-cloud/i18n"
	redc "red-cloud/mod"
	"red-cloud/mod/gologger"
	goruntime "runtime"
	"strings"
	"time"
)

func (a *App) GetConfig() ConfigInfo {
	logPath := ""
	if a.logMgr != nil {
		logPath = a.logMgr.BaseDir
	}

	// Try to load proxy settings from GUI settings first, fallback to env vars
	httpProxy := os.Getenv("HTTP_PROXY")
	httpsProxy := os.Getenv("HTTPS_PROXY")
	socks5Proxy := os.Getenv("ALL_PROXY")
	noProxy := os.Getenv("NO_PROXY")

	// Load from GUI settings if available
	if settings, err := redc.LoadGUISettings(); err == nil && settings != nil {
		if settings.HttpProxy != "" {
			httpProxy = settings.HttpProxy
		}
		if settings.HttpsProxy != "" {
			httpsProxy = settings.HttpsProxy
		}
		if settings.Socks5Proxy != "" {
			socks5Proxy = settings.Socks5Proxy
		}
		if settings.NoProxy != "" {
			noProxy = settings.NoProxy
		}
	}

	return ConfigInfo{
		RedcPath:     redc.RedcPath,
		ProjectPath:  redc.ProjectPath,
		LogPath:      logPath,
		HttpProxy:    httpProxy,
		HttpsProxy:   httpsProxy,
		Socks5Proxy:  socks5Proxy,
		NoProxy:      noProxy,
		DebugEnabled: redc.Debug,
	}
}

func (a *App) GetVersion() string {
	return redc.Version
}

func (a *App) CheckForUpdates() (VersionCheckResult, error) {
	result := VersionCheckResult{
		CurrentVersion: redc.Version,
		DownloadURL:    "https://github.com/wgpsec/redc/releases",
	}

	resp, err := redc.NewProxyHTTPClient(30 * time.Second).Get("https://api.github.com/repos/wgpsec/redc/releases/latest")
	if err != nil {
		result.Error = i18n.T("github_connect_failed")
		return result, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		result.Error = i18n.T("github_version_failed")
		return result, nil
	}

	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		result.Error = i18n.T("github_parse_failed")
		return result, nil
	}

	tagName, ok := data["tag_name"].(string)
	if !ok {
		result.Error = i18n.T("github_latest_failed")
		return result, nil
	}

	result.LatestVersion = tagName

	currentVer := strings.TrimPrefix(redc.Version, "v")
	latestVer := strings.TrimPrefix(tagName, "v")

	result.HasUpdate = compareVersions(currentVer, latestVer) < 0

	return result, nil
}

func compareVersions(current, latest string) int {
	currentParts := strings.Split(current, ".")
	latestParts := strings.Split(latest, ".")

	for i := 0; i < len(currentParts) || i < len(latestParts); i++ {
		var cur, lat int
		if i < len(currentParts) {
			fmt.Sscanf(currentParts[i], "%d", &cur)
		}
		if i < len(latestParts) {
			fmt.Sscanf(latestParts[i], "%d", &lat)
		}
		if cur < lat {
			return -1
		}
		if cur > lat {
			return 1
		}
	}
	return 0
}

func (a *App) SaveProxyConfig(httpProxy, httpsProxy, socks5Proxy, noProxy string) error {
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

	if socks5Proxy != "" {
		os.Setenv("ALL_PROXY", socks5Proxy)
		os.Setenv("all_proxy", socks5Proxy)
	} else {
		os.Unsetenv("ALL_PROXY")
		os.Unsetenv("all_proxy")
	}

	if noProxy != "" {
		os.Setenv("NO_PROXY", noProxy)
		os.Setenv("no_proxy", noProxy)
	} else {
		os.Unsetenv("NO_PROXY")
		os.Unsetenv("no_proxy")
	}

	// Persist to GUI settings
	settings, err := redc.LoadGUISettings()
	if err != nil {
		return fmt.Errorf(i18n.Tf("app_gui_load_failed", err))
	}

	settings.HttpProxy = httpProxy
	settings.HttpsProxy = httpsProxy
	settings.Socks5Proxy = socks5Proxy
	settings.NoProxy = noProxy

	if err := redc.SaveGUISettings(settings); err != nil {
		return fmt.Errorf(i18n.Tf("app_gui_save_failed", err))
	}

	a.emitLog(i18n.Tf("app_proxy_updated", httpProxy, httpsProxy, socks5Proxy, noProxy))
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
		a.emitLog(i18n.Tf("app_tf_mirror_written", path))
	} else {
		a.emitLog(i18n.Tf("app_tf_mirror_closed", path))
	}
	return nil
}

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
	client := redc.NewProxyHTTPClient(6 * time.Second)
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

func (a *App) SetDebugLogging(enabled bool) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	redc.Debug = enabled
	if enabled {
		gologger.DefaultLogger.SetMaxLevel(levels.LevelDebug)
		a.emitLog(i18n.T("app_debug_on"))
	} else {
		gologger.DefaultLogger.SetMaxLevel(levels.LevelInfo)
		a.emitLog(i18n.T("app_debug_off"))
	}

	// Save to GUI settings
	settings, err := redc.LoadGUISettings()
	if err != nil {
		return err
	}
	settings.DebugEnabled = enabled
	return redc.SaveGUISettings(settings)
}

func (a *App) SetNotificationEnabled(enabled bool) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.notificationMgr != nil {
		a.notificationMgr.SetEnabled(enabled)
		if enabled {
			a.emitLog(i18n.T("app_notify_on"))
		} else {
			a.emitLog(i18n.T("app_notify_off"))
		}
	}

	// Save to GUI settings
	settings, err := redc.LoadGUISettings()
	if err != nil {
		return err
	}
	settings.NotificationEnabled = enabled
	return redc.SaveGUISettings(settings)
}

func (a *App) GetNotificationEnabled() bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Load from GUI settings
	settings, err := redc.LoadGUISettings()
	if err != nil {
		if a.notificationMgr != nil {
			return a.notificationMgr.IsEnabled()
		}
		return false
	}

	if settings.NotificationEnabled {
		return true
	}

	// Fallback to notification manager
	if a.notificationMgr != nil {
		return a.notificationMgr.IsEnabled()
	}
	return false
}

func (a *App) SetDisableRightClick(enabled bool) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.disableRightClick = enabled

	// Save to GUI settings
	settings, err := redc.LoadGUISettings()
	if err != nil {
		return err
	}
	settings.DisableRightClick = enabled
	return redc.SaveGUISettings(settings)
}

func (a *App) GetDisableRightClick() bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Load from GUI settings
	settings, err := redc.LoadGUISettings()
	if err != nil {
		return true // Default to disabled
	}

	return settings.DisableRightClick
}

func (a *App) SetSpotMonitorEnabled(enabled bool) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Save to GUI settings
	settings, err := redc.LoadGUISettings()
	if err != nil {
		return err
	}
	settings.SpotMonitorEnabled = enabled
	if err := redc.SaveGUISettings(settings); err != nil {
		return err
	}

	// Start or stop the monitor
	if enabled {
		if a.spotMonitor != nil {
			// Already running, no-op
			return nil
		}
		a.spotMonitor = NewSpotMonitor(a, 120*time.Second)
		a.spotMonitor.Start()
		a.emitLog(i18n.T("app_spot_monitor_start_success"))
	} else {
		if a.spotMonitor != nil {
			a.spotMonitor.Stop()
			a.spotMonitor = nil
		}
		a.emitLog(i18n.T("app_spot_monitor_stopped"))
	}
	return nil
}

func (a *App) GetSpotMonitorEnabled() bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	settings, err := redc.LoadGUISettings()
	if err != nil {
		return false
	}
	return settings.SpotMonitorEnabled
}

func (a *App) SetSpotAutoRecoverEnabled(enabled bool) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	settings, err := redc.LoadGUISettings()
	if err != nil {
		return err
	}
	settings.SpotAutoRecoverEnabled = enabled
	return redc.SaveGUISettings(settings)
}

func (a *App) GetSpotAutoRecoverEnabled() bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	settings, err := redc.LoadGUISettings()
	if err != nil {
		return false
	}
	return settings.SpotAutoRecoverEnabled
}

func (a *App) SetLanguage(lang string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Sync with backend i18n module
	i18n.SetLang(lang)

	// Save to GUI settings
	settings, err := redc.LoadGUISettings()
	if err != nil {
		return err
	}
	settings.Language = lang
	return redc.SaveGUISettings(settings)
}

func (a *App) GetLanguage() string {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Load from GUI settings
	settings, err := redc.LoadGUISettings()
	if err != nil {
		lang := detectSystemLanguage()
		i18n.SetLang(lang)
		return lang
	}
	if settings.Language == "" {
		lang := detectSystemLanguage()
		i18n.SetLang(lang)
		return lang
	}
	i18n.SetLang(settings.Language)
	return settings.Language
}

func (a *App) SetShowWelcomeDialog(shown bool) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	settings, err := redc.LoadGUISettings()
	if err != nil {
		return err
	}
	if shown {
		settings.WelcomeDialogShown = "true"
	} else {
		settings.WelcomeDialogShown = "hidden"
	}
	return redc.SaveGUISettings(settings)
}

func (a *App) GetShowWelcomeDialog() bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	// If WelcomeDialogShown is empty or "hidden", don't show
	// Only show if it's the first time (empty string)
	settings, err := redc.LoadGUISettings()
	if err != nil {
		return true // First time, show
	}
	// Show only if it's empty (first time)
	return settings.WelcomeDialogShown == ""
}

// GetWebhookConfig returns the current webhook configuration
func (a *App) GetWebhookConfig() WebhookConfig {
	if a.notificationMgr != nil && a.notificationMgr.webhookMgr != nil {
		return a.notificationMgr.webhookMgr.GetConfig()
	}
	return WebhookConfig{}
}

// SetWebhookConfig saves webhook configuration to GUI settings and updates the in-memory manager
func (a *App) SetWebhookConfig(cfg WebhookConfig) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	settings, err := redc.LoadGUISettings()
	if err != nil {
		return err
	}
	settings.WebhookEnabled = cfg.Enabled
	settings.WebhookSlack = cfg.Slack
	settings.WebhookDingtalk = cfg.Dingtalk
	settings.WebhookDingtalkSecret = cfg.DingtalkSecret
	settings.WebhookFeishu = cfg.Feishu
	settings.WebhookFeishuSecret = cfg.FeishuSecret
	settings.WebhookDiscord = cfg.Discord
	settings.WebhookWecom = cfg.Wecom
	if err := redc.SaveGUISettings(settings); err != nil {
		return err
	}

	// Update in-memory webhook manager
	if a.notificationMgr != nil && a.notificationMgr.webhookMgr != nil {
		a.notificationMgr.webhookMgr.SetConfig(cfg)
	}
	return nil
}

// TestWebhook sends a test message to the specified platform
func (a *App) TestWebhook(platform, webhookURL, secret string) error {
	if a.notificationMgr == nil || a.notificationMgr.webhookMgr == nil {
		return fmt.Errorf("webhook manager not initialized")
	}
	return a.notificationMgr.webhookMgr.TestWebhook(platform, webhookURL, secret)
}

func detectSystemLanguage() string {
	// Try to get system locale
	lang := getSystemLocale()
	// If locale starts with "zh", use Chinese, otherwise English
	if len(lang) >= 2 && lang[:2] == "zh" {
		return "zh"
	}
	return "en"
}

func getSystemLocale() string {
	// Try to detect OS and get locale
	// For macOS: check LC_ALL, LC_MESSAGES, LANG environment variables
	// For Windows: use standard library
	// For Linux: check environment variables

	// Check common environment variables for locale
	locales := []string{"LC_ALL", "LC_MESSAGES", "LANG", "LANGUAGE"}
	for _, env := range locales {
		if val := os.Getenv(env); val != "" {
			// Parse locale like "en_US.UTF-8" or "zh_CN.UTF-8"
			parts := strings.Split(val, ".")
			if len(parts) > 0 {
				lang := strings.ToLower(parts[0])
				return lang
			}
		}
	}

	// Try runtime.GOOS specific methods
	switch goruntime.GOOS {
	case "darwin":
		// On macOS, try to get user default language using syscall
		return getMacOSLanguage()
	case "windows":
		// On Windows, try to get console code page
		return getWindowsLanguage()
	}

	return "en"
}

func getMacOSLanguage() string {
	// Try using environment variable that macOS sets
	if lang := os.Getenv("LANG"); lang != "" {
		return strings.ToLower(strings.Split(lang, "_")[0])
	}
	return "en"
}

func getWindowsLanguage() string {
	// On Windows, try to detect language from environment variables
	// Check common Windows language settings
	if lang := os.Getenv("LANG"); lang != "" {
		return strings.ToLower(strings.Split(lang, "_")[0])
	}
	// Check LC_ALL, LC_MESSAGES
	for _, env := range []string{"LC_ALL", "LC_MESSAGES"} {
		if val := os.Getenv(env); val != "" {
			parts := strings.Split(val, ".")
			if len(parts) > 0 {
				return strings.ToLower(parts[0])
			}
		}
	}
	return "en"
}

func maskValue(value string) string {
	if value == "" {
		return ""
	}
	if len(value) <= 4 {
		return "****"
	}
	return "****" + value[len(value)-4:]
}
