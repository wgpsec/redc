package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"red-cloud/i18n"
	redc "red-cloud/mod"
	"red-cloud/mod/cost"
	"red-cloud/mod/gologger"
	"red-cloud/mod/mcp"

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
	customDeploymentService *redc.CustomDeploymentService
	templateManager         *redc.TemplateManager
	configStore             *redc.ConfigStore
	disableRightClick       bool
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
	}

	// Initialize config using same path detection as CLI
	if err := redc.LoadConfig(""); err != nil {
		a.initError = i18n.Tf("app_config_load_failed2", err)
		runtime.LogErrorf(ctx, a.initError)
		return
	}
	if profile, err := redc.GetActiveProfile(); err == nil {
		if _, err := redc.SetActiveProfile(profile.ID); err != nil {
			runtime.LogInfof(ctx, i18n.Tf("app_profile_init_failed", err))
		}
	} else {
		runtime.LogInfof(ctx, i18n.Tf("app_profile_init_failed", err))
	}

	runtime.LogInfof(ctx, i18n.Tf("app_config_load_success", redc.RedcPath, redc.ProjectPath, redc.TemplateDir))

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
		runtime.LogInfof(ctx, i18n.Tf("app_project_load_success", a.project.ProjectName))
	} else {
		a.initError = i18n.Tf("app_project_load_failed2", err)
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

	runtime.LogInfof(ctx, i18n.Tf("app_cost_init_success", pricingCacheDBPath))

	// Initialize task scheduler
	schedulerDBPath := filepath.Join(redc.RedcPath, "scheduler.db")
	a.taskScheduler = redc.NewTaskScheduler(a.project, schedulerDBPath)

	// 初始化数据库
	if err := a.taskScheduler.InitDB(); err != nil {
		runtime.LogErrorf(ctx, i18n.Tf("app_scheduler_db_init_failed", err))
	} else {
		runtime.LogInfof(ctx, i18n.Tf("app_scheduler_db_init_success", schedulerDBPath))
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
	a.taskScheduler.Start()

	runtime.LogInfof(ctx, i18n.T("app_scheduler_start_success"))

	// Initialize custom deployment service
	a.customDeploymentService = redc.NewCustomDeploymentService()
	a.templateManager = redc.NewTemplateManager()
	a.configStore = redc.NewConfigStore()

	runtime.LogInfof(ctx, i18n.T("app_deploy_service_init_success"))
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

// ProjectInfo represents project information for frontend display
type ProjectInfo struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	CreateTime string `json:"createTime"`
	User       string `json:"user"`
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

// TemplateVariable represents a variable definition from terraform
type TemplateVariable struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	Description  string `json:"description"`
	DefaultValue string `json:"defaultValue"`
	Required     bool   `json:"required"`
}

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

type MCPStatus struct {
	Running         bool   `json:"running"`
	Mode            string `json:"mode"`
	Address         string `json:"address"`
	ProtocolVersion string `json:"protocolVersion"`
}
