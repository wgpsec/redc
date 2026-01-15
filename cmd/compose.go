package cmd

import (
	"red-cloud/mod/compose"
	"red-cloud/mod/gologger"

	"github.com/spf13/cobra"
)

var (
	composeFile string
	profiles    []string
)

var composeCmd = &cobra.Command{
	Use:   "compose",
	Short: "redc 编排",
}

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "启动编排环境",
	Run: func(cmd *cobra.Command, args []string) {
		opts := compose.ComposeOptions{
			File:     composeFile,
			Profiles: profiles,
			Project:  redcProject,
		}

		if err := compose.RunComposeUp(opts); err != nil {
			gologger.Fatal().Msgf("编排失败: %v", err)
		}

		redcProject.SaveProject()
		gologger.Info().Msg("✨ 所有服务部署完成！")
	},
}

var downCmd = &cobra.Command{
	Use:   "down",
	Short: "销毁编排环境",
	Run: func(cmd *cobra.Command, args []string) {
		opts := compose.ComposeOptions{
			File:     composeFile,
			Profiles: profiles,
			Project:  redcProject,
		}

		if err := compose.RunComposeDown(opts); err != nil {
			gologger.Fatal().Msgf("销毁失败: %v", err)
		}

		redcProject.SaveProject()
		gologger.Info().Msg("✨ 所有服务销毁完成！")
	},
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "预览编排配置和变量解析结果 (Dry Run)",
	Long:  "解析 redc-compose.yaml，展示所有的服务裂变结果、依赖关系以及传递给 Terraform 的变量值。",
	Run: func(cmd *cobra.Command, args []string) {
		// 2. 构造选项
		// profiles 是之前的全局变量 pProfiles (需要在 root.go 或 compose.go 中定义)
		opts := compose.ComposeOptions{
			File:     "redc-compose.yaml", // 这里建议做成可配置的 flag
			Profiles: profiles,            // 引用全局 profile 变量
			Project:  redcProject,
		}

		// 3. 执行预览
		if err := compose.InspectConfig(opts); err != nil {
			gologger.Fatal().Msgf("配置解析失败: %v", err)
		}
	},
}

func init() {
	upCmd.Flags().StringVarP(&composeFile, "file", "f", "redc-compose.yaml", "配置文件路径")
	upCmd.Flags().StringSliceVarP(&profiles, "profile", "p", []string{}, "激活的 Profiles")

	downCmd.Flags().StringVarP(&composeFile, "file", "f", "redc-compose.yaml", "配置文件路径")
	downCmd.Flags().StringSliceVarP(&profiles, "profile", "p", []string{}, "激活的 Profiles")

	composeCmd.AddCommand(upCmd)
	composeCmd.AddCommand(downCmd)
	composeCmd.AddCommand(configCmd)
	rootCmd.AddCommand(composeCmd)
}
