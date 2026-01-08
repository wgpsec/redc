package mod

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"red-cloud/mod/gologger"
	"runtime"
	"time"

	"github.com/hashicorp/hc-install/product"
	"github.com/hashicorp/hc-install/releases"
	"github.com/hashicorp/terraform-exec/tfexec"
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
}

// NewTerraformExecutor creates a new terraform executor for the given working directory
func NewTerraformExecutor(workingDir string) (*TerraformExecutor, error) {
	// Find terraform executable
	execPath, err := GetTerraformExecPath(context.Background(), ".bin")
	if err != nil {
		return nil, fmt.Errorf("terraform executable not found: %w", err)
	}

	tf, err := tfexec.NewTerraform(workingDir, execPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create terraform executor: %w", err)
	}
	// Set stdout and stderr to os defaults for visibility
	if Debug {
		tf.SetStdout(os.Stdout)
		tf.SetStderr(os.Stderr)
	}

	return &TerraformExecutor{
		tf:         tf,
		workingDir: workingDir,
	}, nil
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
func (te *TerraformExecutor) Show(ctx context.Context) error {
	state, err := te.tf.Show(ctx)
	if err != nil {
		return fmt.Errorf("failed to show terraform state: %w", err)
	}
	if state != nil && state.Values != nil {
		fmt.Printf("Terraform state: %+v\n", state.Values)
	} else {
		fmt.Println("No terraform state found")
	}
	return nil
}

// ShowPlan 解析 Plan 文件并打印人类可读的摘要
func (te *TerraformExecutor) ShowPlan(ctx context.Context) error {
	// 1. 调用 Show 读取 Plan 文件
	plan, err := te.tf.ShowPlanFile(ctx, planPath)
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
