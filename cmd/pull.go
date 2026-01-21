package cmd

import (
	"red-cloud/mod"
	"red-cloud/mod/gologger"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// 定义命令行变量
var opts struct {
	Registry string
	// Dir 字段已移除，改为直接绑定 mod.TemplateDir
	Force   bool
	Timeout time.Duration
}

var pullCmd = &cobra.Command{
	Use:   "pull <image>[:tag]",
	Short: "Pull a template from registry",
	Args:  cobra.ExactArgs(1), // 必须传入 1 个参数
	RunE: func(cmd *cobra.Command, args []string) error {
		// 1. 组装配置
		pullOpts := mod.PullOptions{
			RegistryURL: opts.Registry,
			Force:       opts.Force,
			Timeout:     opts.Timeout,
		}

		// 2. 调用核心逻辑
		// 无需传递路径，mod.TemplateDir 已经通过 Flag 修改（或使用默认值）
		err := mod.Pull(cmd.Context(), args[0], pullOpts)

		if err != nil {
			// 3. 错误处理
			if strings.Contains(err.Error(), "context canceled") {
				gologger.Warning().Msg("❌ Operation canceled by user.")
				return nil
			}
			return err
		}

		return nil
	},
}

func init() {
	// 绑定 Registry 参数
	pullCmd.Flags().StringVarP(&opts.Registry, "registry", "r", "https://redc.wgpsec.org", "Registry URL")

	// 【关键】直接绑定 mod 包的全局变量 TemplateDir
	// 用户如果不传 -d，mod.TemplateDir 就是默认值 "redc-templates"
	// 用户如果传了 -d "mydir"，mod.TemplateDir 自动变为 "mydir"
	pullCmd.Flags().StringVarP(&mod.TemplateDir, "dir", "d", "redc-templates", "Output directory")

	// 其他参数
	pullCmd.Flags().BoolVarP(&opts.Force, "force", "f", false, "Force pull (overwrite)")
	pullCmd.Flags().DurationVar(&opts.Timeout, "timeout", 60*time.Second, "Download timeout")

	rootCmd.AddCommand(pullCmd)
}
