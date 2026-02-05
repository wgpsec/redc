package mod

import (
	"fmt"
	"os"
	"path/filepath"
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
	gologger.Info().Msgf("正在初始化场景「%s」", Path)

	// 寻找执行程序
	te, err := NewTerraformExecutor(Path)
	if err != nil {
		return fmt.Errorf("TF可执行配置失败: %w", err)
	}

	// Use retry logic with InitRetries constant
	err = retryOperation(ctx, te.Init, InitRetries)
	if err != nil {
		return fmt.Errorf("请检查网络连接: %w", err)
	}
	return nil
}

// TfInit2 复制模版后再尝试初始化
func TfInit2(Path string) error {
	if err := TfInit(Path); err != nil {
		gologger.Debug().Msgf("初始化失败！: %v", err)
		// 无法初始化,删除 case 文件夹
		if removeErr := os.RemoveAll(Path); removeErr != nil {
			gologger.Error().Msgf("删除文件夹失败: %v", removeErr)
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
		return fmt.Errorf("执行失败: %s", err.Error())
	}
	o := ToPlan(opts)
	// 增加 plan 输出文件
	o = append(o, tfexec.Out(RedcPlanPath))
	err = te.Plan(ctx, o...)
	if err != nil {
		gologger.Debug().Msgf("场景创建失败: %v", err)
		return err
	}
	err = te.ShowPlan(ctx)
	if err != nil {
		gologger.Error().Msgf("PLAN 信息展示失败！")
	}
	return nil
}
func TfApply(Path string, opts ...string) error {
	ctx, cancel := createContextWithTimeout()
	defer cancel()
	gologger.Debug().Msgf("Applying terraform in %s\n", Path)
	te, err := NewTerraformExecutor(Path)
	if err != nil {
		return fmt.Errorf("场景启动失败,terraform未找到或配置错误: %w", err)
	}
	var o []tfexec.ApplyOption
	planFile := filepath.Join(Path, RedcPlanPath)
	if _, err := os.Stat(planFile); err == nil {
		o = append(o, tfexec.DirOrPlan(RedcPlanPath))
	} else {
		o = ToApply(opts)
	}
	
	// Add stdout/stderr wrapper if needed for debugging
	// But NewTerraformExecutor already handles it via options or default os.Stdout
	
	err = te.Apply(ctx, o...)
	if err != nil {
		gologger.Debug().Msgf("场景启动失败: %s", err.Error())
		return err
	}
	return nil
}

func TfStatus(Path string) (*tfjson.State, error) {
	ctx, cancel := createContextWithTimeout()
	defer cancel()
	gologger.Debug().Msgf("Getting terraform status in %s\n", Path)
	te, err := NewTerraformExecutor(Path)
	if err != nil {
		return nil, fmt.Errorf("场景状态查询失败,terraform未找到或配置错误: %v\n", err)
	}
	s, err := te.Show(ctx)
	if err != nil {
		return nil, fmt.Errorf("场景状态查询失败!请手动排查问题,path路径: %s\n错误信息：%v\n", Path, err)
	}
	return s, nil
}

func TfOutput(Path string) (map[string]tfexec.OutputMeta, error) {
	ctx, cancel := createContextWithTimeout()
	defer cancel()
	gologger.Debug().Msgf("Getting terraform output in %s\n", Path)

	te, err := NewTerraformExecutor(Path)
	if err != nil {
		return nil, fmt.Errorf("TF可执行配置失败: %w", err)
	}

	outputs, err := te.Output(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取 Output 失败: %w", err)
	}
	return outputs, nil
}

func TfDestroy(Path string, opts []string) error {
	ctx, cancel := createContextWithTimeout()
	defer cancel()
	gologger.Debug().Msgf("Destroying terraform in %s\n", Path)
	te, err := NewTerraformExecutor(Path)
	if err != nil {
		gologger.Error().Msgf("场景销毁失败,terraform未找到或配置错误: %v", err)
		return err // Add return here!
	}
	if te == nil {
		return fmt.Errorf("TerraformExecutor is nil")
	}
	err = te.Destroy(ctx, ToDestroy(opts)...)
	if err != nil {
		gologger.Error().Msgf("场景销毁失败!请手动排查问题,path路径: %s,%v", Path, err)
	}
	return nil
}

func ToApply(v []string) []tfexec.ApplyOption {
	var opts []tfexec.ApplyOption
	for _, s := range v {
		opts = append(opts, tfexec.Var(s))
	}
	return opts
}

func ToPlan(v []string) []tfexec.PlanOption {
	var opts []tfexec.PlanOption
	for _, s := range v {
		opts = append(opts, tfexec.Var(s))
	}
	return opts
}

func ToDestroy(v []string) []tfexec.DestroyOption {
	var opts []tfexec.DestroyOption
	for _, s := range v {
		opts = append(opts, tfexec.Var(s))
	}
	return opts
}
