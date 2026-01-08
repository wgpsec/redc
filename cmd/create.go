package cmd

import (
	redc "red-cloud/mod"
	"red-cloud/mod/gologger"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	userName     string
	projectName  string
	envVars      map[string]string
	commandToRun string
)

var runCmd = &cobra.Command{
	Use:     "run [template_name]",
	Short:   "创建并立即启动一个场景",
	Example: "redc run ecs",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		templateName := args[0]
		c := createLogic(templateName)
		if err := c.TfApply(); err != nil {
			gologger.Error().Msgf("场景启动失败！%s", err.Error())
		}
		if len(args) > 1 {
			commandToRun = strings.Join(args[1:], " ")
		}
	},
}

var createCmd = &cobra.Command{
	Use:     "create [template_name]",
	Short:   "创建一个新的基础设施场景",
	Example: "redc create ecs -u team1 -n operation_alpha",
	Args:    cobra.ExactArgs(1), // 强制要求输入一个模板名，例如 pte
	Run: func(cmd *cobra.Command, args []string) {
		templateName := args[0]
		c := createLogic(templateName)
		gologger.Info().Msgf("✅「%s」%s 场景创建完成！，接下来您可以start启动该场景", c.Name, c.Id)
	},
}

func createLogic(templateName string) *redc.Case {

	// 别名处理
	if templateName == "pte" {
		templateName = "pte_arm"
	}

	pro, err := redc.ProjectParse(redc.Project, userName)
	if err != nil {
		gologger.Fatal().Msgf("项目解析失败: %s", err)
	}

	// 创建 Case
	c, err := pro.CaseCreate(templateName, userName, projectName, envVars)
	if err != nil {
		gologger.Error().Msgf("❌「%s」场景创建失败: %v", templateName, err)
		return nil
	}
	gologger.Info().Msgf("✅「%s」场景创建完成！", templateName)
	return c
}

func init() {
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(runCmd)
	CRCommonFlagSet := pflag.NewFlagSet("common", pflag.ExitOnError)

	CRCommonFlagSet.StringVarP(&userName, "user", "u", "system", "指定用户/操作员")
	CRCommonFlagSet.StringVarP(&projectName, "name", "n", "", "指定项目/任务名称")
	CRCommonFlagSet.StringToStringVarP(&envVars, "env", "e", nil, "设置环境变量 (格式: key=value)")
	createCmd.Flags().AddFlagSet(CRCommonFlagSet)
	runCmd.Flags().AddFlagSet(CRCommonFlagSet)
	// 禁用参数混排
	runCmd.Flags().SetInterspersed(false)
}
