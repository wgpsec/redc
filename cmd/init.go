package cmd

import (
	"os"
	redc "red-cloud/mod"
	"red-cloud/mod/gologger"
	"red-cloud/utils"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "初始化环境和模板",
	Run: func(cmd *cobra.Command, args []string) {
		redc.RedcLog("执行初始化")
		gologger.Info().Msg("初始化中...")

		/*
			// 清理旧目录
			os.RemoveAll(redc.TemplateDir)

			// 释放资源
			if err := utils.ReleaseDir(redc.TemplateDir); err != nil {
				gologger.Fatal().Msgf("释放模板资源失败: %s", err)
			}
		*/

		// 遍历初始化
		_, dirs := utils.GetFilesAndDirs(redc.TemplateDir)
		for _, v := range dirs {
			if err := redc.TfInit(v); err != nil {
				gologger.Error().Msgf("❌「%s」场景初始化失败: %s", v, err)
			} else {
				gologger.Info().Msgf("✅「%s」场景初始化完成", v)
			}
		}
	},
}

// completionCmd 生成命令补全脚本
var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "生成命令补全脚本",
	Long: `要在当前 Shell 中加载补全，请运行以下命令:

Bash:
  $ source <(redc completion bash)

Zsh:
  # 如果开启了 oh-my-zsh，通常可以直接运行:
  $ source <(redc completion zsh)

  # 如果没有生效，可能需要手动配置 fpath (详细参考官方文档)
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	Run: func(cmd *cobra.Command, args []string) {
		switch args[0] {
		case "bash":
			cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			cmd.Root().GenFishCompletion(os.Stdout, true)
		case "powershell":
			cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
		}
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
	rootCmd.AddCommand(initCmd)
}
