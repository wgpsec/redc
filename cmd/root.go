package cmd

import (
	"os"
	redc "red-cloud/mod"
	"red-cloud/mod/gologger"

	"github.com/projectdiscovery/gologger/levels"
	"github.com/spf13/cobra"
)

var (
	cfgFile     string
	showVer     bool
	redcProject *redc.RedcProject
)

const BannerArt = `
██████╗  ███████╗ ██████╗   ██████╗ 
 ██╔══██╗ ██╔════╝ ██╔══██╗ ██╔════╝ 
 ██████╔╝ █████╗   ██║  ██║ ██║      
 ██╔══██╗ ██╔══╝   ██║  ██║ ██║      
 ██║  ██║ ███████╗ ██████╔╝ ╚██████╗ 
 ╚═╝  ╚═╝ ╚══════╝ ╚═════╝   ╚═════╝
`

// rootCmd
var rootCmd = &cobra.Command{
	Use:   "redc",
	Short: "Red Cloud - 红队基础设施自动化工具",
	Long:  BannerArt + "\nRed Cloud 是一个用于快速部署和管理红队云基础设施的工具。",
	// PersistentPreRun 在任何子命令执行前都会运行
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// 如果是查版本，就不加载配置
		if showVer {
			return
		}
		// 统一加载配置
		if err := redc.LoadConfig(cfgFile); err != nil {
			gologger.Fatal().Msgf("配置载失败！%s\n", err.Error())
		}
		if redc.Debug {
			gologger.DefaultLogger.SetMaxLevel(levels.LevelDebug)
			gologger.Debug().Msgf("当前已开始 DEBUG 模式！")
		}
		// 加载项目
		if p, err := redc.ProjectParse(redc.Project, redc.U); err == nil {
			redcProject = p
		} else {
			gologger.Fatal().Msgf("项目加载失败: %v", err)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		if showVer {
			gologger.Print().Msgf("%s\nVersion: %s\n", BannerArt, redc.Version)
			return
		}
		// 如果没参数也没flag，打印帮助
		cmd.Help()
	},
}

// Execute 是 main.go 调用的入口
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		gologger.Error().Msgf(err.Error())
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().BoolVarP(&showVer, "version", "v", false, "显示版本信息")
	// 全局 Flag
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "配置文件路径")
	rootCmd.PersistentFlags().StringVar(&redc.RedcPath, "runpath", "", "运行路径，默认在user路径下")
	rootCmd.PersistentFlags().StringVarP(&redc.U, "user", "u", "system", "操作者")
	rootCmd.PersistentFlags().StringVar(&redc.Project, "project", "default", "项目名称")
	rootCmd.PersistentFlags().BoolVar(&redc.Debug, "debug", false, "开启调试模式")
}
