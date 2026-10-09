package mod

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"red-cloud/i18n"
	"red-cloud/mod/gologger"
	"strings"

	"github.com/hashicorp/terraform-exec/tfexec"
	tfjson "github.com/hashicorp/terraform-json"
)

// readFileContent reads a file and returns its content with newlines trimmed
func readFileContent(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// TfInit 初始化场景
func TfInit(Path string) error {
	ctx, cancel := createContextWithTimeout()
	defer cancel()
	gologger.Info().Msgf("%s", i18n.Tf("tf_init_terraform", Path))

	// 寻找执行程序
	te, err := NewTerraformExecutor(Path)
	if err != nil {
		return fmt.Errorf("%s", i18n.Tf("tf_exec_config_failed", err))
	}

	// Use retry logic with InitRetries constant
	err = retryOperation(ctx, te.Init, InitRetries)
	if err != nil {
		return fmt.Errorf("%s", i18n.Tf("tf_network_error", err))
	}
	return nil
}

// TfInit2 复制模版后再尝试初始化
func TfInit2(Path string) error {
	if err := TfInit(Path); err != nil {
		gologger.Error().Msgf("%s", i18n.Tf("tf_init_failed", err))
		// 无法初始化,删除 case 文件夹
		if removeErr := os.RemoveAll(Path); removeErr != nil {
			gologger.Error().Msgf("%s", i18n.Tf("tf_init_delete_failed", removeErr))
		}
		return err // 返回原始的初始化错误
	}
	return nil
}

// RVar 统一转换接口，方便后续替换类型
func RVar(s ...string) []string {
	return s
}

func TfPlan(Path string, opts ...string) error {
	ctx, cancel := createContextWithTimeout()
	defer cancel()
	gologger.Debug().Msgf("Planing terraform in %s\n", Path)
	te, err := NewTerraformExecutor(Path)
	if err != nil {
		return fmt.Errorf("%s", i18n.Tf("tf_exec_failed", err.Error()))
	}
	// defer 而非只在成功分支调用：terraform 在命令报错返回前就可能已经写出
	// tfstate / tfplan，失败路径同样可能留下 0644 的明文密码。
	defer SecureTfState(Path)
	o, cleanup, err := ToPlan(Path, opts)
	if err != nil {
		return err
	}
	defer cleanup()
	// 增加 plan 输出文件
	o = append(o, tfexec.Out(RedcPlanPath))
	err = te.Plan(ctx, o...)
	if err != nil {
		gologger.Error().Msgf("%s", i18n.Tf("tf_create_failed", err))
		return err
	}
	err = te.ShowPlan(ctx)
	if err != nil {
		gologger.Error().Msg(i18n.T("tf_plan_show_failed"))
	}
	return nil
}

func TfApply(Path string, opts ...string) error {
	ctx, cancel := createContextWithTimeout()
	defer cancel()
	gologger.Debug().Msgf("Applying terraform in %s\n", Path)
	te, err := NewTerraformExecutor(Path)
	if err != nil {
		return fmt.Errorf("%s", i18n.Tf("tf_start_failed_no_tf", err))
	}
	defer SecureTfState(Path)
	var o []tfexec.ApplyOption
	planFile := filepath.Join(Path, RedcPlanPath)
	if _, err := os.Stat(planFile); err == nil {
		o = append(o, tfexec.DirOrPlan(RedcPlanPath))
	} else {
		var cleanup func()
		o, cleanup, err = ToApply(Path, opts)
		if err != nil {
			return err
		}
		defer cleanup()
	}

	err = te.Apply(ctx, o...)
	if err != nil {
		gologger.Error().Msgf("%s", i18n.Tf("tf_start_failed", err.Error()))
		// 返回更详细的错误信息，包含 Terraform 的实际错误输出
		return fmt.Errorf("%s", i18n.Tf("tf_apply_failed", err))
	}
	SecureTfState(Path)
	return nil
}

func TfStatus(Path string) (*tfjson.State, error) {
	ctx, cancel := createContextWithTimeout()
	defer cancel()
	gologger.Debug().Msgf("Getting terraform status in %s\n", Path)
	te, err := NewTerraformExecutor(Path)
	if err != nil {
		return nil, fmt.Errorf("%s", i18n.Tf("tf_query_failed_no_tf", err))
	}
	s, err := te.Show(ctx)
	if err != nil {
		return nil, fmt.Errorf("%s", i18n.Tf("tf_query_failed", Path, err))
	}
	return s, nil
}

func TfOutput(Path string) (map[string]tfexec.OutputMeta, error) {
	ctx, cancel := createContextWithTimeout()
	defer cancel()
	gologger.Debug().Msgf("Getting terraform output in %s\n", Path)

	te, err := NewTerraformExecutor(Path)
	if err != nil {
		return nil, fmt.Errorf("%s", i18n.Tf("tf_exec_config_failed", err))
	}

	outputs, err := te.Output(ctx)
	if err != nil {
		return nil, fmt.Errorf("%s", i18n.Tf("tf_output_failed", err))
	}
	return outputs, nil
}

// TfValidate runs terraform validate to check syntax and configuration
func TfValidate(Path string) (*tfjson.ValidateOutput, error) {
	ctx, cancel := createContextWithTimeout()
	defer cancel()
	gologger.Debug().Msgf("Validating terraform in %s\n", Path)
	te, err := NewTerraformExecutor(Path)
	if err != nil {
		return nil, fmt.Errorf("%s", i18n.Tf("tf_exec_config_failed", err))
	}
	result, err := te.Validate(ctx)
	if err != nil {
		return nil, fmt.Errorf("terraform validate failed: %w", err)
	}
	return result, nil
}

func TfDestroy(Path string, opts []string) error {
	ctx, cancel := createContextWithTimeout()
	defer cancel()
	gologger.Debug().Msgf("Destroying terraform in %s\n", Path)
	te, err := NewTerraformExecutor(Path)
	if err != nil {
		gologger.Error().Msgf("%s", i18n.Tf("tf_destroy_failed_no_tf", err))
		return err // Add return here!
	}
	if te == nil {
		return fmt.Errorf("TerraformExecutor is nil")
	}
	defer SecureTfState(Path)
	o, cleanup, err := ToDestroy(Path, opts)
	if err != nil {
		return err
	}
	defer cleanup()
	err = te.Destroy(ctx, o...)
	if err != nil {
		gologger.Error().Msgf("%s", i18n.Tf("tf_destroy_failed", Path, err))
		return err
	}
	return nil
}

// writeVarFile materialises the -var assignments Terraform would otherwise
// receive on the command line. Passing them as argv put every generated
// instance password and provider secret into `ps aux` output for the whole
// lifetime of the terraform run, readable by any local user.
//
// A JSON variable file is used rather than TF_VAR_* environment variables
// because terraform.tfvars takes precedence over TF_VAR_*, so switching to the
// environment would silently stop overriding the template's own defaults.
// JSON also lets Terraform coerce string values into the number and bool
// variables the templates declare.
//
// The file is created 0600 and removed by the returned cleanup function.
func writeVarFile(dir string, vars []string) (string, func(), error) {
	noop := func() {}
	if len(vars) == 0 {
		return "", noop, nil
	}

	values := make(map[string]string, len(vars))
	for _, s := range vars {
		key, val, ok := strings.Cut(s, "=")
		if !ok || key == "" {
			continue
		}
		values[key] = val
	}
	if len(values) == 0 {
		return "", noop, nil
	}

	data, err := json.Marshal(values)
	if err != nil {
		return "", noop, fmt.Errorf("%s", i18n.Tf("tf_var_file_encode_failed", err))
	}

	f, err := os.CreateTemp(dir, ".redc-vars-*.auto.tfvars.json")
	if err != nil {
		return "", noop, fmt.Errorf("%s", i18n.Tf("tf_var_file_create_failed", err))
	}
	path := f.Name()
	cleanup := func() { os.Remove(path) }

	if err := f.Chmod(0600); err != nil {
		f.Close()
		cleanup()
		return "", noop, fmt.Errorf("%s", i18n.Tf("tf_var_file_create_failed", err))
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		cleanup()
		return "", noop, fmt.Errorf("%s", i18n.Tf("tf_var_file_write_failed", err))
	}
	if err := f.Close(); err != nil {
		cleanup()
		return "", noop, fmt.Errorf("%s", i18n.Tf("tf_var_file_write_failed", err))
	}
	return path, cleanup, nil
}

// SecureTfState tightens permissions on the artefacts Terraform leaves behind
// in a case directory. tfstate and the saved plan both embed generated
// instance passwords in plaintext, and the plan additionally captures the
// values of every variable that was set.
func SecureTfState(dir string) {
	for _, name := range []string{
		"terraform.tfstate",
		"terraform.tfstate.backup",
		"terraform.tfvars",
		RedcPlanPath,
	} {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err != nil {
			continue
		}
		if err := os.Chmod(path, 0600); err != nil {
			gologger.Debug().Msgf("chmod %s: %v", path, err)
		}
	}
}

func ToApply(dir string, v []string) ([]tfexec.ApplyOption, func(), error) {
	var opts []tfexec.ApplyOption
	path, cleanup, err := writeVarFile(dir, v)
	if err != nil {
		return nil, nil, err
	}
	if path != "" {
		opts = append(opts, tfexec.VarFile(path))
	}
	return opts, cleanup, nil
}

func ToPlan(dir string, v []string) ([]tfexec.PlanOption, func(), error) {
	var opts []tfexec.PlanOption
	path, cleanup, err := writeVarFile(dir, v)
	if err != nil {
		return nil, nil, err
	}
	if path != "" {
		opts = append(opts, tfexec.VarFile(path))
	}
	return opts, cleanup, nil
}

func ToDestroy(dir string, v []string) ([]tfexec.DestroyOption, func(), error) {
	var opts []tfexec.DestroyOption
	path, cleanup, err := writeVarFile(dir, v)
	if err != nil {
		return nil, nil, err
	}
	if path != "" {
		opts = append(opts, tfexec.VarFile(path))
	}
	return opts, cleanup, nil
}
