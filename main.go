package main

import (
	_ "embed"
	"flag"
	"fmt"
	"os"
	redc "red-cloud/mod"
	"red-cloud/mod/gologger"
	"red-cloud/utils"

	"github.com/gen2brain/beeep"
)

var ProjectPath = "./redc-taskresult"

const banner = `

██████╗  ███████╗ ██████╗   ██████╗ 
 ██╔══██╗ ██╔════╝ ██╔══██╗ ██╔════╝ 
 ██████╔╝ █████╗   ██║  ██║ ██║      
 ██╔══██╗ ██╔══╝   ██║  ██║ ██║      
 ██║  ██║ ███████╗ ██████╔╝ ╚██████╗ 
 ╚═╝  ╚═╝ ╚══════╝ ╚═════╝   ╚═════╝

`

var (
	BuiltAt   string
	GoVersion string
	GitAuthor string
	BuildSha  string
	GitTag    string
)

func Banner() {
	gologger.Print().Msgf("%sBuilt At: %s\nGo Version: %s\nAuthor: %s\nBuild SHA: %s\nVersion: %s\n\n", banner, BuiltAt, GoVersion, GitAuthor, BuildSha, GitTag)
}
func main() {
	Banner()
	flag.Parse()
	// -version 显示版本号
	if redc.V {
		fmt.Println(redc.Version)
		_ = beeep.Notify("redc", "版本"+redc.Version, "assets/information.png")
		os.Exit(0)
	}

	// 解析配置文件
	if err := redc.LoadCredentials("./config.yaml"); err != nil {
		gologger.Fatal().Msgf("配置文件加载失败! %s", err.Error())
	}

	// 解析配置(暂时不需要这一步)
	// redc.LoadConfig(configPath)

	// -init 初始化
	if redc.Init {
		redc.RedcLog("进行初始化")
		gologger.Info().Msgf("初始化中")
		// 先删除文件夹
		err := os.RemoveAll("redc-templates")
		gologger.Error().Msgf("初始化过程中删除模板文件夹失败: %s", err)
		// 释放 templates 资源
		utils.ReleaseDir("redc-templates")

		// 遍历 redc-templates 文件夹,不包括子目录
		_, dirs := utils.GetFilesAndDirs("./redc-templates")
		for _, v := range dirs {
			err = redc.TfInit(v)
			if err != nil {
				gologger.Error().Msgf("「%s」场景初始化失败\n %s", v, err)
			}
		}
		gologger.Info().Msgf("✅场景初始化任务完成！")
		return
	}

	// 解析项目名称
	redc.ProjectParse(ProjectPath+"/"+redc.Project, redc.Project, redc.U)

	// list 操作查看项目里所有 case
	if redc.List {
		redc.CaseList(ProjectPath + "/" + redc.Project)
	}

	// start 操作,去调用 case 创建方法
	if redc.Start != "" {
		redc.RedcLog("start " + redc.Start)
		if redc.Start == "pte" {
			redc.Start = "pte_arm"
		}
		//fmt.Println("step1")
		redc.CaseCreate(ProjectPath+"/"+redc.Project, redc.Start, redc.U, redc.Name)
	}

	// stop 操作,去调用 case 删除方法
	if redc.Stop != "" {
		redc.RedcLog("stop " + redc.Stop)
		redc.CheckUser(ProjectPath+"/"+redc.Project, redc.Stop)
		redc.CaseStop(ProjectPath+"/"+redc.Project, redc.Stop)
	}
	if redc.Kill != "" {
		redc.CheckUser(ProjectPath+"/"+redc.Project, redc.Kill)
		redc.CaseKill(ProjectPath+"/"+redc.Project, redc.Kill)
	}

	// change 操作,去调用 case 更改方法
	if redc.Change != "" {
		redc.RedcLog("change " + redc.Change)
		redc.CheckUser(ProjectPath+"/"+redc.Project, redc.Change)
		redc.CaseChange(ProjectPath+"/"+redc.Project, redc.Change)
	}

	// status 操作,去调用 case 状态方法
	if redc.Status != "" {
		redc.RedcLog("status" + redc.Status)
		redc.CheckUser(ProjectPath+"/"+redc.Project, redc.Status)
		redc.CaseStatus(ProjectPath+"/"+redc.Project, redc.Status)
	}

}
