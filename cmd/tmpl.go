package cmd

import (
	redc "red-cloud/mod"

	"github.com/spf13/cobra"
)

var tmplCmd = &cobra.Command{
	Use:   "image",
	Short: "管理模版信息",
	Run: func(cmd *cobra.Command, args []string) {
		// 如果用户只输入了 'redc image' 而没输 'ls'，通常打印帮助信息
		cmd.Help()
	},
}

// 3. 定义三级命令: ls
var showAll bool // 定义一个变量来接收 flag

var tmplLsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List all available images",
	Run: func(cmd *cobra.Command, args []string) {
		redc.ListRedcTmpl()
	},
}

func init() {
	rootCmd.AddCommand(tmplCmd)
	tmplCmd.AddCommand(tmplLsCmd)
}
