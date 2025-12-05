package mod

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/hashicorp/terraform-exec/tfexec"
)

// TerraformExecutor wraps terraform-exec functionality
type TerraformExecutor struct {
	tf         *tfexec.Terraform
	workingDir string
}

// NewTerraformExecutor creates a new terraform executor for the given working directory
func NewTerraformExecutor(workingDir string) (*TerraformExecutor, error) {
	// Find terraform executable
	execPath, err := exec.LookPath("terraform")
	if err != nil {
		return nil, fmt.Errorf("terraform executable not found: %w", err)
	}

	tf, err := tfexec.NewTerraform(workingDir, execPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create terraform executor: %w", err)
	}

	// Set stdout and stderr to os defaults for visibility
	tf.SetStdout(os.Stdout)
	tf.SetStderr(os.Stderr)

	return &TerraformExecutor{
		tf:         tf,
		workingDir: workingDir,
	}, nil
}

// Init runs terraform init
func (te *TerraformExecutor) Init(ctx context.Context) error {
	return te.tf.Init(ctx, tfexec.Upgrade(true))
}

// Apply runs terraform apply with auto-approve
func (te *TerraformExecutor) Apply(ctx context.Context) error {
	return te.tf.Apply(ctx)
}

// Destroy runs terraform destroy with auto-approve
func (te *TerraformExecutor) Destroy(ctx context.Context) error {
	return te.tf.Destroy(ctx)
}

// Output retrieves a terraform output value as a string
func (te *TerraformExecutor) Output(ctx context.Context, name string) (string, error) {
	outputs, err := te.tf.Output(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get terraform outputs: %w", err)
	}

	outputValue, ok := outputs[name]
	if !ok {
		return "", fmt.Errorf("output %q not found", name)
	}

	// Parse the JSON value to string
	var result string
	if err := json.Unmarshal(outputValue.Value, &result); err != nil {
		return "", fmt.Errorf("failed to parse output value: %w", err)
	}

	return result, nil
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

// createContextWithTimeout creates a context with a default timeout
func createContextWithTimeout() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 30*time.Minute)
}

// TfExecInit0 performs first-time initialization using terraform-exec
func TfExecInit0(Path string) {
	ctx, cancel := createContextWithTimeout()
	defer cancel()

	fmt.Printf("Initializing terraform in %s\n", Path)

	te, err := NewTerraformExecutor(Path)
	if err != nil {
		fmt.Printf("场景初始化失败: %v\n", err)
		// Fallback to bash method
		TfInit0(Path)
		return
	}

	err = te.Init(ctx)
	if err != nil {
		fmt.Println("场景初始化失败,再次尝试!", err)
		// Retry once
		err2 := te.Init(ctx)
		if err2 != nil {
			fmt.Println("场景初始化失败,请检查网络连接!", err2)
			os.Exit(3)
		}
	}
}

// TfExecInit performs initialization after copying using terraform-exec
func TfExecInit(Path string) {
	ctx, cancel := createContextWithTimeout()
	defer cancel()

	fmt.Printf("Initializing terraform in %s\n", Path)

	te, err := NewTerraformExecutor(Path)
	if err != nil {
		fmt.Printf("场景初始化失败: %v\n", err)
		// Fallback to bash method
		TfInit(Path)
		return
	}

	err = te.Init(ctx)
	if err != nil {
		fmt.Println("场景初始化失败,再次尝试!", err)
		// Retry once
		err2 := te.Init(ctx)
		if err2 != nil {
			fmt.Println("场景初始化失败,请检查网络连接!", err2)
			// Remove the case folder on failure
			os.RemoveAll(Path)
			os.Exit(3)
		}
	}
}

// TfExecApply performs terraform apply using terraform-exec
func TfExecApply(Path string) {
	ctx, cancel := createContextWithTimeout()
	defer cancel()

	fmt.Printf("Applying terraform in %s\n", Path)

	te, err := NewTerraformExecutor(Path)
	if err != nil {
		fmt.Printf("场景创建失败: %v\n", err)
		// Fallback to bash method
		TfApply(Path)
		return
	}

	err = te.Apply(ctx)
	if err != nil {
		fmt.Println("场景创建失败!尝试重新创建!")
		// Try to destroy first
		te.Destroy(ctx)
		// Retry apply
		err2 := te.Apply(ctx)
		if err2 != nil {
			fmt.Println("场景创建第二次失败!请手动排查问题")
			fmt.Println("path路径: ", Path)
			os.Exit(3)
		}
	}
}

// TfExecStatus shows terraform status using terraform-exec
func TfExecStatus(Path string) {
	ctx, cancel := createContextWithTimeout()
	defer cancel()

	fmt.Printf("Getting terraform status in %s\n", Path)

	te, err := NewTerraformExecutor(Path)
	if err != nil {
		fmt.Printf("场景状态查询失败: %v\n", err)
		// Fallback to bash method
		TfStatus(Path)
		return
	}

	err = te.Show(ctx)
	if err != nil {
		fmt.Println("场景状态查询失败!请手动排查问题")
		fmt.Println("path路径: ", Path)
		os.Exit(3)
	}
}

// TfExecDestroy performs terraform destroy using terraform-exec
func TfExecDestroy(Path string) {
	ctx, cancel := createContextWithTimeout()
	defer cancel()

	fmt.Printf("Destroying terraform resources in %s\n", Path)

	te, err := NewTerraformExecutor(Path)
	if err != nil {
		fmt.Printf("场景销毁失败: %v\n", err)
		// Fallback to bash method
		TfDestroy(Path)
		return
	}

	err = te.Destroy(ctx)
	if err != nil {
		fmt.Println("场景销毁失败,第二次尝试!", err)
		// Retry twice more
		err2 := te.Destroy(ctx)
		if err2 != nil {
			fmt.Println("场景销毁失败,第三次尝试!", err2)
			err3 := te.Destroy(ctx)
			if err3 != nil {
				fmt.Println("场景销毁多次重试失败!请手动排查问题")
				fmt.Println("path路径: ", Path)
				os.Exit(3)
			}
		}
	}
}

// GetTerraformOutput retrieves a terraform output value using terraform-exec
func GetTerraformOutput(Path string, outputName string) (string, error) {
	ctx, cancel := createContextWithTimeout()
	defer cancel()

	te, err := NewTerraformExecutor(Path)
	if err != nil {
		return "", fmt.Errorf("failed to create terraform executor: %w", err)
	}

	return te.Output(ctx, outputName)
}
