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
	Short: "列出所有模版文件",
	Run: func(cmd *cobra.Command, args []string) {
		redc.ListLocalTemplates()
	},
}
var tmplRMCmd = &cobra.Command{
	Use:   "rm [case]",
	Short: "删除模版文件",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 2 {
			cmd.Help()
			return // 直接 return，返回 nil，这样 main 函数就不会打印 [ERR]
		}
		id := args[0]
		// 将剩余参数组合成命令
		redc.RemoveTemplate(id)
	},
}

func init() {
	rootCmd.AddCommand(tmplCmd)
	tmplCmd.AddCommand(tmplLsCmd)
	tmplCmd.AddCommand(tmplRMCmd)
}
