package cmd

import (
	redc "red-cloud/mod"
	"red-cloud/mod/gologger"

	"github.com/spf13/cobra"
)

var tmplCmd = &cobra.Command{
	Use:   "image",
	Short: "管理模版信息",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

// 3. 定义三级命令: ls
var showAll bool // 定义一个变量来接收 flag

var tmplLsCmd = &cobra.Command{
	Use:   "ls",
	Short: "列出所有模版文件",
	Run: func(cmd *cobra.Command, args []string) {
		redc.ShowLocalTemplates()
	},
}
var tmplRMCmd = &cobra.Command{
	Use:   "rm [case]",
	Short: "删除模版文件",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		if err := redc.RemoveTemplate(id); err != nil {
			gologger.Error().Msgf("remove template failed: %v", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(tmplCmd)
	tmplCmd.AddCommand(tmplLsCmd)
	tmplCmd.AddCommand(tmplRMCmd)
}
