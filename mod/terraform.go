package mod

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"red-cloud/mod/gologger"
	"runtime"
	"strings"
	"time"

	"github.com/hashicorp/hc-install/product"
	"github.com/hashicorp/hc-install/releases"
	"github.com/hashicorp/terraform-exec/tfexec"
	tfjson "github.com/hashicorp/terraform-json"
)

const (
	// TerraformTimeout is the default timeout for terraform operations
	TerraformTimeout = 30 * time.Minute
	// ExitCodeFailure is the exit code used for failures
	ExitCodeFailure = 3
	// MaxRetries is the maximum number of retries for failed operations
	MaxRetries = 3
	// InitRetries is the number of retries for init operations
	InitRetries = 2
)

// TerraformExecutor wraps terraform-exec functionality
type TerraformExecutor struct {
	tf         *tfexec.Terraform
	workingDir string
	stdout     io.Writer
	stderr     io.Writer
}

// TerraformOption configures a TerraformExecutor
type TerraformOption func(*TerraformExecutor)

// WithStdout sets a custom stdout writer for terraform output
func WithStdout(w io.Writer) TerraformOption {
	return func(te *TerraformExecutor) {
		te.stdout = w
	}
}

// WithStderr sets a custom stderr writer for terraform output
func WithStderr(w io.Writer) TerraformOption {
	return func(te *TerraformExecutor) {
		te.stderr = w
	}
}

// NewTerraformExecutor creates a new terraform executor for the given working directory
func NewTerraformExecutor(workingDir string, opts ...TerraformOption) (*TerraformExecutor, error) {
	// Determine bin directory
	binDir := ".bin"
	if RedcPath != "" {
		binDir = filepath.Join(RedcPath, "bin")
	}

	// Find terraform executable
	execPath, err := GetTerraformExecPath(context.Background(), binDir)
	if err != nil {
		return nil, fmt.Errorf("terraform executable not found: %w", err)
	}

	tf, err := tfexec.NewTerraform(workingDir, execPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create terraform executor: %w", err)
	}

	te := &TerraformExecutor{
		tf:         tf,
		workingDir: workingDir,
		stdout:     os.Stdout,
		stderr:     os.Stderr,
	}

	// Apply options
	for _, opt := range opts {
		opt(te)
	}

	// Always set stdout and stderr for better visibility and debugging
	tf.SetStdout(te.stdout)
	tf.SetStderr(te.stderr)

	// Pass all environment variables including proxy settings to terraform subprocess
	// Check if proxy is configured
	proxyVars := []string{"HTTP_PROXY", "http_proxy", "HTTPS_PROXY", "https_proxy", "NO_PROXY", "no_proxy"}
	hasProxy := false
	for _, key := range proxyVars {
		if os.Getenv(key) != "" {
			hasProxy = true
			break
		}
	}
	
	// On macOS, always set environment variables to include DYLD_FALLBACK_LIBRARY_PATH
	// This fixes the "Failed to read any lines from plugin's stdout" error with Terraform providers
	needsCustomEnv := hasProxy || runtime.GOOS == "darwin"
	
	if needsCustomEnv {
		envVars := make(map[string]string)
		// Copy all current environment variables
		for _, env := range os.Environ() {
			if idx := strings.Index(env, "="); idx > 0 {
				envVars[env[:idx]] = env[idx+1:]
			}
		}
		
		// On macOS, ensure library paths are set for provider plugins
		if runtime.GOOS == "darwin" {
			if _, exists := envVars["DYLD_FALLBACK_LIBRARY_PATH"]; !exists {
				// Set default fallback library paths for macOS
				envVars["DYLD_FALLBACK_LIBRARY_PATH"] = "/usr/local/lib:/usr/lib"
			}
			// Also set DYLD_LIBRARY_PATH to help with plugin loading
			if _, exists := envVars["DYLD_LIBRARY_PATH"]; !exists {
				envVars["DYLD_LIBRARY_PATH"] = "/usr/local/lib:/usr/lib"
			}
			// Enable Terraform plugin debug logging on macOS to diagnose issues
			if _, exists := envVars["TF_LOG_PROVIDER"]; !exists && Debug {
				envVars["TF_LOG_PROVIDER"] = "DEBUG"
			}
		}
		
		tf.SetEnv(envVars)
	}

	return te, nil
}

// Init runs terraform init with upgrade option
func (te *TerraformExecutor) Init(ctx context.Context) error {
	return te.tf.Init(ctx, tfexec.Upgrade(false))
}

// Apply runs terraform apply (auto-approve is the default behavior in terraform-exec)
func (te *TerraformExecutor) Apply(ctx context.Context, opts ...tfexec.ApplyOption) error {
	return te.tf.Apply(ctx, opts...)
}
func (te *TerraformExecutor) Plan(ctx context.Context, opts ...tfexec.PlanOption) error {
	_, err := te.tf.Plan(ctx, opts...)
	return err
}

// Destroy runs terraform destroy (auto-approve is the default behavior in terraform-exec)
func (te *TerraformExecutor) Destroy(ctx context.Context, opts ...tfexec.DestroyOption) error {
	return te.tf.Destroy(ctx, opts...)
}

// Output retrieves a terraform output value as a string
func (te *TerraformExecutor) Output(ctx context.Context) (map[string]tfexec.OutputMeta, error) {
	outputs, err := te.tf.Output(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get terraform outputs: %w", err)
	}
	return outputs, err
}

// Show runs terraform show to display current state
func (te *TerraformExecutor) Show(ctx context.Context) (*tfjson.State, error) {
	state, err := te.tf.Show(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to show terraform state: %w", err)
	}
	if state != nil && state.Values != nil {
		return state, nil
	}
	return state, fmt.Errorf("state is empty")
}

// ShowPlan 解析 Plan 文件并打印人类可读的摘要
func (te *TerraformExecutor) ShowPlan(ctx context.Context) error {
	// 1. 调用 Show 读取 Plan 文件
	plan, err := te.tf.ShowPlanFile(ctx, RedcPlanPath)
	if err != nil {
		return fmt.Errorf("读取 Plan 失败: %w", err)
	}

	// 2. 检查是否有资源变更
	if plan.ResourceChanges == nil {
		fmt.Println("没有检测到资源变更。")
		return nil
	}

	fmt.Println("=== 变更预览 ===")

	// 3. 遍历变更列表
	for _, rc := range plan.ResourceChanges {
		// rc.Address 是资源的标识符 (例如: aws_instance.web_server)
		// rc.Change.Actions 是一个字符串切片，描述动作 (["create"], ["update"], ["delete", "create"] 等)

		actions := rc.Change.Actions

		// 简单的逻辑判断动作类型
		if len(actions) == 1 {
			switch actions[0] {
			case "create":
				fmt.Printf("[+] 创建: %s\n", rc.Address)
			case "delete":
				fmt.Printf("[-] 销毁: %s\n", rc.Address)
			case "update":
				fmt.Printf("[~] 更新: %s\n", rc.Address)
			case "no-op":
				// 没有任何变更，通常不需要打印
			}
		} else if len(actions) == 2 {
			// 通常是 ["delete", "create"]，意味着重建
			if actions[0] == "delete" && actions[1] == "create" {
				fmt.Printf("[+/-] 重建 (销毁后创建): %s\n", rc.Address)
			}
		}
	}

	return nil
}

// createContextWithTimeout creates a context with a default timeout
func createContextWithTimeout() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), TerraformTimeout)
}

// retryOperation retries an operation up to maxRetries times
func retryOperation(ctx context.Context, operation func(context.Context) error, maxRetries int) error {
	var err error
	for i := 0; i < maxRetries; i++ {
		err = operation(ctx)
		if err == nil {
			return nil
		}
		if i < maxRetries-1 {
			gologger.Info().Msgf("Retrying operation (attempt %d/%d)...", i+1, maxRetries)
		}
	}
	return err
}

// GetTerraformExecPath 获取 Terraform 二进制路径
// priorityDir: 如果需要下载，指定的持久化安装目录（例如 "./bin"）
func GetTerraformExecPath(ctx context.Context, priorityDir string) (string, error) {
	// 1. 尝试从系统环境变量 PATH 中查找
	if path, err := exec.LookPath("terraform"); err == nil {
		return path, nil
	}

	// 1.5 macOS GUI 应用可能没有继承完整 PATH，尝试常见路径
	commonPaths := []string{
		"/opt/homebrew/bin/terraform",     // Homebrew on Apple Silicon
		"/usr/local/bin/terraform",         // Homebrew on Intel / manual install
		"/usr/bin/terraform",               // System
	}
	for _, p := range commonPaths {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	// 2. 检查自定义目录中是否已经存在（针对之前下载过的情况）
	// 考虑 Windows 系统的 .exe 后缀
	execName := "terraform"
	if runtime.GOOS == "windows" {
		execName = "terraform.exe"
	}

	absDir, _ := filepath.Abs(priorityDir)
	localPath := filepath.Join(absDir, execName)

	if _, err := os.Stat(localPath); err == nil {
		return localPath, nil
	}

	// 3. 如果以上都没有，执行自动下载
	gologger.Info().Msgf("未检测到 Terraform，正在下载最新版本到: %s...\n", absDir)

	// 确保目录存在
	if err := os.MkdirAll(absDir, 0755); err != nil {
		return "", fmt.Errorf("创建安装目录失败: %w", err)
	}

	//installer := &releases.ExactVersion{
	//	Product: product.Terraform,
	//	Version: version.Must(version.NewVersion("1.14.3")),
	//}

	// 默认下载最新版本
	installer := releases.LatestVersion{
		Product:    product.Terraform,
		InstallDir: absDir,
	}

	installedPath, err := installer.Install(ctx)
	if err != nil {
		return "", fmt.Errorf("下载 Terraform 失败: %w", err)
	}

	return installedPath, nil
}
