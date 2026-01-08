package cmd

import (
	"red-cloud/mod/gologger"

	"github.com/spf13/cobra"
)

var execCmd = &cobra.Command{
	Use:     "exec [id] 命令",
	Short:   "进入机器执行命令",
	Example: "redc exec id whoami",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		gologger.Error().Msgf("该场景功能正在开发中。。。。")
	},
}

func init() {
	rootCmd.AddCommand(execCmd)
}
