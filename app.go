package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"red-cloud/i18n"
	redc "red-cloud/mod"
	"red-cloud/mod/cost"
	"red-cloud/mod/gologger"
	"red-cloud/mod/mcp"
	"red-cloud/mod/plugin"

	"github.com/projectdiscovery/gologger/levels"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx                     context.Context
	project                 *redc.RedcProject
	mu                      sync.Mutex
	initError               string
	logMgr                  *gologger.LogManager
	mcpManager              *mcp.MCPServerManager
	notificationMgr         *NotificationManager
	pricingService          *cost.PricingService
	costCalculator          *cost.CostCalculator
	taskScheduler           *redc.TaskScheduler
	spotMonitor             *SpotMonitor
	customDeploymentService *redc.CustomDeploymentService
	templateManager         *redc.TemplateManager
	configStore             *redc.ConfigStore
	pluginMgr               *plugin.PluginManager
	auditStore              *redc.AuditStore
	timelineStore           *redc.TimelineStore
	disableRightClick       bool
	httpSrv                 *HTTPServer
	wailsMode               bool // true when running inside Wails desktop
	activeOps               atomic.Int32 // tracks in-flight async operations (apply/destroy/compose)
	updateState             UpdateState
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		notificationMgr: NewNotificationManager(),
	}
}

// HasActiveOperations returns true if any async operations (apply/destroy/compose) are in progress
func (a *App) HasActiveOperations() bool {
	return a.activeOps.Load() > 0
}

// beforeClose is called when the user tries to close the window
func (a *App) beforeClose(ctx context.Context) bool {
	if a.activeOps.Load() > 0 {
		result, err := runtime.MessageDialog(ctx, runtime.MessageDialogOptions{
			Type:          runtime.QuestionDialog,
			Title:         i18n.T("app_quit_confirm_title"),
			Message:       i18n.T("app_quit_confirm_message"),
			Buttons:       []string{i18n.T("app_quit_btn_cancel"), i18n.T("app_quit_btn_confirm")},
			DefaultButton: i18n.T("app_quit_btn_cancel"),
			CancelButton:  i18n.T("app_quit_btn_cancel"),
		})
		if err != nil || result != i18n.T("app_quit_btn_confirm") {
			return true // prevent close
		}
	}
	return false // allow close
}

// shutdown is called when the app is shutting down
func (a *App) shutdown(ctx context.Context) {
	// Stop spot monitor and wait for any in-progress recovery to finish
	if a.spotMonitor != nil {
		a.spotMonitor.Stop()
		a.spotMonitor = nil
	}
	if a.timelineStore != nil {
		a.timelineStore.Close()
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.wailsMode = true

	// Set default values (same as CLI defaults)
	if redc.Project == "" {
		redc.Project = "default"
	}
	if redc.U == "" {
		redc.U = "system" // Use "system" to match CLI default and bypass permission check
	}

	// Load GUI settings
	if settings, err := redc.LoadGUISettings(); err == nil {
		// Apply debug setting
		if settings.DebugEnabled {
			redc.Debug = true
			gologger.DefaultLogger.SetMaxLevel(levels.LevelDebug)
		}

		// Apply proxy settings from GUI settings
		if settings.HttpProxy != "" {
			os.Setenv("HTTP_PROXY", settings.HttpProxy)
			os.Setenv("http_proxy", settings.HttpProxy)
		}
		if settings.HttpsProxy != "" {
			os.Setenv("HTTPS_PROXY", settings.HttpsProxy)
			os.Setenv("https_proxy", settings.HttpsProxy)
		}
		if settings.Socks5Proxy != "" {
			os.Setenv("ALL_PROXY", settings.Socks5Proxy)
			os.Setenv("all_proxy", settings.Socks5Proxy)
		}
		if settings.NoProxy != "" {
			os.Setenv("NO_PROXY", settings.NoProxy)
			os.Setenv("no_proxy", settings.NoProxy)
		}

		// Restore notification enabled state
		if settings.NotificationEnabled && a.notificationMgr != nil {
			a.notificationMgr.SetEnabled(true)
		}
	}

	// Initialize config using same path detection as CLI
	if err := redc.LoadConfig(""); err != nil {
		a.initError = i18n.Tf("app_config_load_failed2", err)
		fmt.Printf("[ERROR] %s\n", a.initError)
		return
	}
	if profile, err := redc.GetActiveProfile(); err == nil {
		if _, err := redc.SetActiveProfile(profile.ID); err != nil {
			fmt.Printf("[INFO] %s\n", i18n.Tf("app_profile_init_failed", err))
		}
	} else {
		fmt.Printf("[INFO] %s\n", i18n.Tf("app_profile_init_failed", err))
	}

	fmt.Printf("[INFO] %s\n", i18n.Tf("app_config_load_success", redc.RedcPath, redc.ProjectPath, redc.TemplateDir))

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
		fmt.Printf("[INFO] %s\n", i18n.Tf("app_project_load_success", a.project.ProjectName))
	} else {
		a.initError = i18n.Tf("app_project_load_failed2", err)
		fmt.Printf("[ERROR] %s\n", a.initError)
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

	fmt.Printf("[INFO] %s\n", i18n.Tf("app_cost_init_success", pricingCacheDBPath))

	// Initialize task scheduler
	schedulerDBPath := filepath.Join(redc.RedcPath, "scheduler.db")
	a.taskScheduler = redc.NewTaskScheduler(a.project, schedulerDBPath)

	// 初始化数据库
	if err := a.taskScheduler.InitDB(); err != nil {
		fmt.Printf("[ERROR] %s\n", i18n.Tf("app_scheduler_db_init_failed", err))
	} else {
		fmt.Printf("[INFO] %s\n", i18n.Tf("app_scheduler_db_init_success", schedulerDBPath))
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
		return fmt.Errorf(i18n.Tf("app_unknown_action", action))
	})

	// SSH command callback for task center
	a.taskScheduler.SetSSHCommandCallback(func(caseID string, command string) (string, error) {
		c, err := a.project.GetCase(caseID)
		if err != nil {
			return "", fmt.Errorf("case not found: %v", err)
		}
		sshConfig, err := c.GetSSHConfig()
		if err != nil {
			return "", fmt.Errorf("SSH config error: %v", err)
		}
		result := a.execSSHCommand(sshConfig, command)
		output := result.Stdout
		if result.Stderr != "" {
			output += "\n[stderr] " + result.Stderr
		}
		if !result.Success {
			return output, fmt.Errorf("exit code %d: %s", result.ExitCode, result.Error)
		}
		return output, nil
	})

	// Notification callback for task center
	a.taskScheduler.SetNotifyCallback(func(title string, message string) {
		if a.notificationMgr != nil {
			a.notificationMgr.Send(title, message)
			if a.notificationMgr.webhookMgr != nil {
				a.notificationMgr.webhookMgr.Send(title, message, "#4a90d9")
			}
		}
	})

	a.taskScheduler.Start()

	fmt.Printf("[INFO] %s\n", i18n.T("app_scheduler_start_success"))

	// Initialize custom deployment service
	a.customDeploymentService = redc.NewCustomDeploymentService()
	a.templateManager = redc.NewTemplateManager()
	a.configStore = redc.NewConfigStore()

	// Initialize plugin manager
	a.pluginMgr = plugin.NewPluginManager("")
	if err := a.pluginMgr.LoadAll(); err != nil {
		fmt.Printf("[WARN] plugin load: %v\n", err)
	} else {
		fmt.Printf("[INFO] plugins loaded: %d\n", len(a.pluginMgr.List()))
	}

	fmt.Printf("[INFO] %s\n", i18n.T("app_deploy_service_init_success"))

	// Initialize audit log store
	if as, err := redc.NewAuditStore(); err == nil {
		a.auditStore = as
		fmt.Println("[INFO] Audit log store initialized")
	} else {
		fmt.Printf("[WARN] Audit log store init failed: %v\n", err)
	}

	// Initialize timeline store
	if ts, err := redc.NewTimelineStore(); err == nil {
		a.timelineStore = ts
		fmt.Println("[INFO] Timeline store initialized")
		a.logTimeline("system", "app_started", "", "", i18n.T("app_timeline_app_started"), "", "info")
	} else {
		fmt.Printf("[WARN] Timeline store init failed: %v\n", err)
	}

	// Start spot instance termination monitor (if enabled in settings)
	if settings, err := redc.LoadGUISettings(); err == nil && settings.SpotMonitorEnabled {
		a.spotMonitor = NewSpotMonitor(a, 120*time.Second)
		a.spotMonitor.Start()
		fmt.Printf("[INFO] %s\n", i18n.T("app_spot_monitor_start_success"))
	}

	// Check for updates in the background
	go a.CheckForUpdatesOnStartup()
}

// startupHeadless initializes the app without Wails context
func (a *App) startupHeadless() {
	a.wailsMode = false
	a.startup(context.Background())
}

// emitEvent dispatches an event to frontend (Wails or HTTP SSE)
func (a *App) emitEvent(name string, data interface{}) {
	if a.wailsMode && a.ctx != nil {
		runtime.EventsEmit(a.ctx, name, data)
	}
	if a.httpSrv != nil {
		a.httpSrv.broadcast(name, data)
	}
}

// emitLog sends a log message to the frontend and writes to file
func (a *App) emitLog(message string) {
	a.emitEvent("log", message)
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
	a.emitEvent("refresh", nil)
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
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Type           string   `json:"type"`
	State          string   `json:"state"`
	StateTime      string   `json:"stateTime"`
	CreateTime     string   `json:"createTime"`
	Operator       string   `json:"operator"`
	IsSpotInstance bool     `json:"isSpotInstance"`
	Tags           []string `json:"tags"`
	Plugins        string   `json:"plugins,omitempty"`
}

// ConfigInfo represents the configuration for frontend display
type ConfigInfo struct {
	RedcPath     string `json:"redcPath"`
	ProjectPath  string `json:"projectPath"`
	LogPath      string `json:"logPath"`
	HttpProxy    string `json:"httpProxy"`
	HttpsProxy   string `json:"httpsProxy"`
	Socks5Proxy  string `json:"socks5Proxy"`
	NoProxy      string `json:"noProxy"`
	DebugEnabled bool   `json:"debugEnabled"`
}

// TerraformMirrorConfig represents terraform mirror configuration status
type TerraformMirrorConfig struct {
	Enabled    bool     `json:"enabled"`
	ConfigPath string   `json:"configPath"`
	Managed    bool     `json:"managed"`
	FromEnv    bool     `json:"fromEnv"`
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
	Fields     map[string]string `json:"fields"`     // field name -> masked value
	HasSecrets map[string]bool   `json:"hasSecrets"` // field name -> has value
}

// ProvidersConfigInfo represents all providers' credentials
type ProvidersConfigInfo struct {
	ConfigPath string               `json:"configPath"`
	Providers  []ProviderCredential `json:"providers"`
}

// VersionCheckResult represents the result of checking for new versions
type VersionCheckResult struct {
	CurrentVersion string `json:"currentVersion"`
	LatestVersion  string `json:"latestVersion"`
	HasUpdate      bool   `json:"hasUpdate"`
	DownloadURL    string `json:"downloadURL"`
	Error          string `json:"error"`
}

// UpdateCheckResult aggregates version checks for redc, templates, and plugins
type UpdateCheckResult struct {
	Redc      VersionCheckResult    `json:"redc"`
	Templates []TemplateUpdateInfo  `json:"templates"`
	Plugins   []PluginUpdateInfo    `json:"plugins"`
}

// TemplateUpdateInfo represents a template that may have an update
type TemplateUpdateInfo struct {
	Name         string `json:"name"`
	LocalVersion string `json:"localVersion"`
	LatestVersion string `json:"latestVersion"`
	HasUpdate    bool   `json:"hasUpdate"`
}

// PluginUpdateInfo represents a plugin with its current version
type PluginUpdateInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ProjectInfo represents project information for frontend display
type ProjectInfo struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	CreateTime string `json:"createTime"`
	User       string `json:"user"`
}

// TemplateInfo represents template information for frontend display
type TemplateInfo struct {
	Name          string `json:"name"`
	Description   string `json:"description"`
	DescriptionEN string `json:"description_en"`
	Version       string `json:"version"`
	User          string `json:"user"`
	Module        string `json:"module"`
	Plugins       string `json:"plugins"`
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
	Status    string   `json:"status"` // "applied" / "destroyed" / "not_deployed"
}

// ComposeSummary represents a compose file preview
type ComposeSummary struct {
	File     string                  `json:"file"`
	Services []ComposeServiceSummary `json:"services"`
	Total    int                     `json:"total"`
}

// BillInfo represents billing information for a cloud provider
type BillInfo struct {
	Provider    string `json:"provider"`
	Month       string `json:"month"`
	TotalAmount string `json:"totalAmount"`
	Currency    string `json:"currency"`
	StartDate   string `json:"startDate"`
	EndDate     string `json:"endDate"`
	Error       string `json:"error"`
}

// TemplateVariableValidation holds validation rules for a template variable
type TemplateVariableValidation struct {
	AllowedValues []string `json:"allowed_values,omitempty"`
}

// TemplateVariable represents a variable definition from terraform
type TemplateVariable struct {
	Name         string                      `json:"name"`
	Type         string                      `json:"type"`
	Description  string                      `json:"description"`
	DefaultValue string                      `json:"defaultValue"`
	Required     bool                        `json:"required"`
	Sensitive    bool                        `json:"sensitive"`
	Validation   *TemplateVariableValidation `json:"validation,omitempty"`
}

type RegistryTemplate struct {
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	DescriptionEN string   `json:"description_en"`
	Author        string   `json:"author"`
	Latest        string   `json:"latest"`
	Versions      []string `json:"versions"`
	UpdatedAt     string   `json:"updatedAt"`
	Tags          []string `json:"tags"`
	Installed     bool     `json:"installed"`
	LocalVer      string   `json:"localVersion"`
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
		Name          string   `json:"name"`
		Author        string   `json:"author"`
		Description   string   `json:"description"`
		DescriptionEN string   `json:"description_en"`
		Readme        string   `json:"readme"`
		Tags          []string `json:"tags,omitempty"`
	} `json:"metadata"`
}

// remoteIndexResponse matches the index.json structure from the registry
type remoteIndexResponse struct {
	UpdatedAt string                        `json:"updated_at"`
	RepoName  string                        `json:"repo_name"`
	Templates map[string]remoteTemplateInfo `json:"templates"`
}

type MCPStatus struct {
	Running         bool   `json:"running"`
	Mode            string `json:"mode"`
	Address         string `json:"address"`
	ProtocolVersion string `json:"protocolVersion"`
}
