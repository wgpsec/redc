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

const (
	// TerraformTimeout is the default timeout for terraform operations
	TerraformTimeout = 30 * time.Minute
	// ExitCodeFailure is the exit code used for failures
	ExitCodeFailure = 3
	// MaxRetries is the maximum number of retries for failed operations
	MaxRetries = 3
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

// Init runs terraform init with upgrade option
func (te *TerraformExecutor) Init(ctx context.Context) error {
	return te.tf.Init(ctx, tfexec.Upgrade(true))
}

// Apply runs terraform apply (auto-approve is the default behavior in terraform-exec)
func (te *TerraformExecutor) Apply(ctx context.Context) error {
	return te.tf.Apply(ctx)
}

// Destroy runs terraform destroy (auto-approve is the default behavior in terraform-exec)
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
	return context.WithTimeout(context.Background(), TerraformTimeout)
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

// retryOperation retries an operation up to maxRetries times
func retryOperation(ctx context.Context, operation func(context.Context) error, maxRetries int) error {
	var err error
	for i := 0; i < maxRetries; i++ {
		err = operation(ctx)
		if err == nil {
			return nil
		}
		if i < maxRetries-1 {
			fmt.Printf("Operation failed (attempt %d/%d): %v\n", i+1, maxRetries, err)
		}
	}
	return err
}
