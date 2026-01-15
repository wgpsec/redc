package cmd

import (
	"fmt"
	redc "red-cloud/mod"
	"red-cloud/mod/gologger"

	"github.com/spf13/cobra"
)

var changeConfig redc.ChangeCommand

// helper: 通用的执行器
func runAction(actionType string, caseID string) {
	// 2. 查找 Case
	c, err := redcProject.GetCase(caseID)
	if err != nil {
		gologger.Error().Msgf("操作失败: 找不到 ID 为「%s」的场景\n错误: %s", caseID, err)
		return
	}

	redc.RedcLog(fmt.Sprintf("Action %s on %s", actionType, caseID))

	// 3. 执行动作
	var actionErr error
	switch actionType {
	case "stop":
		actionErr = c.Stop()
	case "start":
		actionErr = c.TfApply()
	case "kill":
		actionErr = c.Kill()
	case "change":
		actionErr = c.Change(changeConfig)
	case "status":
		actionErr = c.Status()
	case "rm":
		actionErr = c.Remove()
	}

	if actionErr != nil {
		gologger.Error().Msgf("执行「%s」失败，%v\n", actionType, actionErr)
	} else {
		if err := redcProject.SaveProject(); err != nil {
			gologger.Error().Msgf("项目状态保存失败！%s\n", err.Error())
			return
		}
		gologger.Info().Msgf("✅ %s 操作执行成功: 「%s」%s\n", actionType, c.Name, c.GetId())
	}
}

// 定义各个命令
var stopCmd = &cobra.Command{
	Use:   "stop [id]",
	Short: "停止指定场景",
	Run: func(cmd *cobra.Command, args []string) {
		runAction("stop", args[0])
	},
}

var statusCmd = &cobra.Command{
	Use:   "status [id]",
	Short: "查看场景状态",
	Run: func(cmd *cobra.Command, args []string) {
		runAction("status", args[0])
	},
}

var changeCmd = &cobra.Command{
	Use:   "change [id]",
	Short: "更改场景",
	Run: func(cmd *cobra.Command, args []string) {
		runAction("change", args[0])
	},
}

var startCmd = &cobra.Command{
	Use:   "start [id]",
	Short: "启动场景",
	Run: func(cmd *cobra.Command, args []string) {
		runAction("start", args[0])
	},
}

var killCmd = &cobra.Command{
	Use:   "kill [id]",
	Short: "销毁指定场景",
	Run: func(cmd *cobra.Command, args []string) {
		runAction("kill", args[0])
	},
}
var rmCmd = &cobra.Command{
	Use:   "rm [id]",
	Short: "删除场景 case",
	Run: func(cmd *cobra.Command, args []string) {
		runAction("rm", args[0])
	},
}

var listCmd = &cobra.Command{
	Use:   "ps",
	Short: "列出当前所有场景",
	Run: func(cmd *cobra.Command, args []string) {
		redcProject.CaseList()
	},
}

// 注册命令
func init() {
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(killCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(changeCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(rmCmd)
	changeCmd.Flags().BoolVar(&changeConfig.IsRemove, "rm", false, "更改时销毁资源")
	//listCmd.Flags().BoolVarP(&redc.ShowAll, "all", "a", false, "查看所有 case")

}
