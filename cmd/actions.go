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
	// 1. 解析项目
	pro, err := redc.ProjectParse(redc.Project, redc.U) // 注意：这里可能需要处理 global U 或者从配置读取
	if err != nil {
		gologger.Fatal().Msgf("项目解析失败: %s", err)
	}

	// 2. 查找 Case
	c, err := pro.GetCase(caseID)
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
		gologger.Error().Msgf("执行 %s 失败: %v", actionType, actionErr)
	} else {
		if err := pro.SaveProject(); err != nil {
			gologger.Error().Msgf("项目状态保存失败！%s", err.Error())
			return
		}
		gologger.Info().Msgf("✅ %s 操作执行成功: 「%s」%s", actionType, c.Name, c.Id)
	}
}

// 定义各个命令
var stopCmd = &cobra.Command{
	Use:   "stop [id]",
	Short: "停止指定场景",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		runAction("stop", args[0])
	},
}

var statusCmd = &cobra.Command{
	Use:   "status [id]",
	Short: "查看场景状态",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		runAction("status", args[0])
	},
}

var changeCmd = &cobra.Command{
	Use:   "change [id]",
	Short: "更改场景",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		runAction("change", args[0])
	},
}

var startCmd = &cobra.Command{
	Use:   "start [id]",
	Short: "启动场景",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		runAction("start", args[0])
	},
}

var killCmd = &cobra.Command{
	Use:   "kill [id]",
	Short: "销毁指定场景",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		runAction("kill", args[0])
	},
}
var rmCmd = &cobra.Command{
	Use:   "rm [id]",
	Short: "删除场景 case",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		runAction("rm", args[0])
	},
}

var listCmd = &cobra.Command{
	Use:   "ps",
	Short: "列出当前所有场景",
	Run: func(cmd *cobra.Command, args []string) {
		pro, err := redc.ProjectParse(redc.Project, redc.U)
		if err != nil {
			gologger.Fatal().Msgf("项目解析失败: %s", err)
		}
		pro.CaseList()
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
