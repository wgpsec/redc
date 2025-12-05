package mod

import (
	"fmt"
	"github.com/gen2brain/beeep"
	"os"
	"red-cloud/mod2"
	"red-cloud/utils"
	"strconv"
	"strings"
	"time"
)

// 第一次初始化 - 使用 terraform-exec
func TfInit0(Path string) {
	ctx, cancel := createContextWithTimeout()
	defer cancel()

	fmt.Printf("Initializing terraform in %s\n", Path)

	te, err := NewTerraformExecutor(Path)
	if err != nil {
		fmt.Printf("场景初始化失败,尝试使用备用方式: %v\n", err)
		tfInit0Fallback(Path)
		return
	}

	err = te.Init(ctx)
	if err != nil {
		fmt.Println("场景初始化失败,再次尝试!", err)
		// Retry once
		err2 := te.Init(ctx)
		if err2 != nil {
			fmt.Println("场景初始化失败,请检查网络连接!", err2)
			_ = beeep.Notify("redc", fmt.Sprintf("场景初始化失败,请检查网络连接! %v", err2), "assets/information.png")
			os.Exit(3)
		}
	}
}

// tfInit0Fallback 使用bash方式的备用初始化
func tfInit0Fallback(Path string) {
	fmt.Println("cd " + Path + " && bash deploy.sh -init")
	err := utils.Command("cd " + Path + " && bash deploy.sh -init")
	if err != nil {
		fmt.Println("场景初始化失败,再次尝试!", err)

		// 如果初始化失败就再次尝试一次
		err2 := utils.Command("cd " + Path + " && bash deploy.sh -init")
		if err2 != nil {
			fmt.Println("场景初始化失败,请检查网络连接!", err)
			_ = beeep.Notify("redc", fmt.Sprintf("场景初始化失败,请检查网络连接! %v", err), "assets/information.png")
			os.Exit(3)
		}
	}
}

// 复制后的初始化 - 使用 terraform-exec
func TfInit(Path string) {
	ctx, cancel := createContextWithTimeout()
	defer cancel()

	fmt.Printf("Initializing terraform in %s\n", Path)

	te, err := NewTerraformExecutor(Path)
	if err != nil {
		fmt.Printf("场景初始化失败,尝试使用备用方式: %v\n", err)
		tfInitFallback(Path)
		return
	}

	err = te.Init(ctx)
	if err != nil {
		fmt.Println("场景初始化失败,再次尝试!", err)
		// Retry once
		err2 := te.Init(ctx)
		if err2 != nil {
			fmt.Println("场景初始化失败,请检查网络连接!", err2)
			// Remove the case folder on failure
			os.RemoveAll(Path)
			os.Exit(3)
		}
	}
}

// tfInitFallback 使用bash方式的备用初始化
func tfInitFallback(Path string) {
	fmt.Println("cd " + Path + " && bash deploy.sh -init")
	err := utils.Command("cd " + Path + " && bash deploy.sh -init")
	if err != nil {
		fmt.Println("场景初始化失败,再次尝试!", err)

		// 如果初始化失败就再次尝试一次
		err2 := utils.Command("cd " + Path + " && bash deploy.sh -init")
		if err2 != nil {
			fmt.Println("场景初始化失败,请检查网络连接!", err)

			// 无法初始化,删除 case 文件夹
			err = os.RemoveAll(Path)
			if err != nil {
				fmt.Println(err)
				os.Exit(3)
			}
			os.Exit(3)
		}
	}
}

// TfApply 使用 terraform-exec 执行 apply
func TfApply(Path string) {
	ctx, cancel := createContextWithTimeout()
	defer cancel()

	fmt.Printf("Applying terraform in %s\n", Path)

	te, err := NewTerraformExecutor(Path)
	if err != nil {
		fmt.Printf("场景创建失败,尝试使用备用方式: %v\n", err)
		tfApplyFallback(Path)
		return
	}

	err = te.Apply(ctx)
	if err != nil {
		fmt.Println("场景创建失败!尝试重新创建!")
		// Try to destroy first
		te.Destroy(ctx)
		// Retry apply
		err2 := te.Apply(ctx)
		if err2 != nil {
			fmt.Println("场景创建第二次失败!请手动排查问题")
			fmt.Println("path路径: ", Path)
			_ = beeep.Notify("redc", fmt.Sprintf("场景创建第二次失败!请手动排查问题,path路径: %v", Path), "assets/information.png")
			os.Exit(3)
		}
	}
}

// tfApplyFallback 使用bash方式的备用apply
func tfApplyFallback(Path string) {
	fmt.Println("cd " + Path + " && bash deploy.sh -start")
	err := utils.Command("cd " + Path + " && bash deploy.sh -start")
	if err != nil {
		fmt.Println("场景创建失败!尝试重新创建!")

		// 先关闭
		err2 := utils.Command("cd " + Path + " && bash deploy.sh -stop")
		if err2 != nil {
			fmt.Println("场景销毁,等待重新创建!")
			os.Exit(3)
		}

		// 重新创建
		err3 := utils.Command("cd " + Path + " && bash deploy.sh -start")
		if err3 != nil {
			fmt.Println("场景创建第二次失败!请手动排查问题")
			fmt.Println("path路径: ", Path)
			_ = beeep.Notify("redc", fmt.Sprintf("场景创建第二次失败!请手动排查问题,path路径: %v", Path), "assets/information.png")
			os.Exit(3)
		}
	}
}

// TfStatus 使用 terraform-exec 查看状态
func TfStatus(Path string) {
	ctx, cancel := createContextWithTimeout()
	defer cancel()

	fmt.Printf("Getting terraform status in %s\n", Path)

	te, err := NewTerraformExecutor(Path)
	if err != nil {
		fmt.Printf("场景状态查询失败,尝试使用备用方式: %v\n", err)
		tfStatusFallback(Path)
		return
	}

	err = te.Show(ctx)
	if err != nil {
		fmt.Println("场景状态查询失败!请手动排查问题")
		fmt.Println("path路径: ", Path)
		_ = beeep.Notify("redc", fmt.Sprintf("场景状态查询失败!请手动排查问题,path路径: %v", Path), "assets/information.png")
		os.Exit(3)
	}
}

// tfStatusFallback 使用bash方式的备用状态查询
func tfStatusFallback(Path string) {
	fmt.Println("cd " + Path + " && bash deploy.sh -status")
	err := utils.Command("cd " + Path + " && bash deploy.sh -status")
	if err != nil {
		fmt.Println("场景状态查询失败!请手动排查问题")
		fmt.Println("path路径: ", Path)
		_ = beeep.Notify("redc", fmt.Sprintf("场景状态查询失败!请手动排查问题,path路径: %v", Path), "assets/information.png")
		os.Exit(3)
	}
}

// TfDestroy 使用 terraform-exec 销毁资源
func TfDestroy(Path string) {
	ctx, cancel := createContextWithTimeout()
	defer cancel()

	fmt.Printf("Destroying terraform resources in %s\n", Path)

	te, err := NewTerraformExecutor(Path)
	if err != nil {
		fmt.Printf("场景销毁失败,尝试使用备用方式: %v\n", err)
		tfDestroyFallback(Path)
		return
	}

	err = te.Destroy(ctx)
	if err != nil {
		fmt.Println("场景销毁失败,第二次尝试!", err)
		// Retry twice more
		err2 := te.Destroy(ctx)
		if err2 != nil {
			fmt.Println("场景销毁失败,第三次尝试!", err2)
			err3 := te.Destroy(ctx)
			if err3 != nil {
				fmt.Println("场景销毁多次重试失败!请手动排查问题")
				fmt.Println("path路径: ", Path)
				_ = beeep.Notify("redc", fmt.Sprintf("场景销毁多次重试失败!请手动排查问题,path路径: %v", Path), "assets/information.png")
				os.Exit(3)
			}
		}
	}
}

// tfDestroyFallback 使用bash方式的备用销毁
func tfDestroyFallback(Path string) {
	fmt.Println("cd " + Path + " && bash deploy.sh -stop")
	err := utils.Command("cd " + Path + " && bash deploy.sh -stop")
	if err != nil {
		fmt.Println("场景销毁失败,第二次尝试!", err)

		// 如果初始化失败就再次尝试一次
		err2 := utils.Command("cd " + Path + " && bash deploy.sh -stop")
		if err2 != nil {
			fmt.Println("场景销毁失败,第三次尝试!", err)

			// 第三次
			err3 := utils.Command("cd " + Path + " && bash deploy.sh -stop")
			if err3 != nil {
				fmt.Println("场景销毁多次重试失败!请手动排查问题")
				fmt.Println("path路径: ", Path)
				_ = beeep.Notify("redc", fmt.Sprintf("场景销毁多次重试失败!请手动排查问题,path路径: %v", Path), "assets/information.png")
				os.Exit(3)
			}
		}
	}
}

func C2Apply(Path string) {

	// 先开c2
	err := utils.Command("cd " + Path + " && bash deploy.sh -step1")
	if err != nil {
		fmt.Println("场景创建失败,自动销毁场景!")
		RedcLog("场景创建失败,自动销毁场景!")
		C2Destroy(Path, strconv.Itoa(Node), Domain)
		// 成功销毁场景后,删除 case 文件夹
		err = os.RemoveAll(Path)
		os.Exit(3)
	}

	// 开rg
	if Node != 0 {
		err = utils.Command("cd " + Path + " && bash deploy.sh -step2 " + strconv.Itoa(Node) + " " + Domain)
		if err != nil {
			fmt.Println("场景创建失败,自动销毁场景!")
			RedcLog("场景创建失败,自动销毁场景!")
			C2Destroy(Path, strconv.Itoa(Node), Domain)
			// 成功销毁场景后,删除 case 文件夹
			err = os.RemoveAll(Path)
			os.Exit(3)
		}
	}

	// 获得本地几个变量 - 使用 terraform-exec 获取 output
	c2_ip, err := GetTerraformOutput(Path+"/c2-ecs", "ecs_ip")
	if err != nil {
		fmt.Printf("获取 ecs_ip 失败,尝试使用备用方式: %v\n", err)
		c2_ip = utils.Command2("cd " + Path + " && cd c2-ecs" + "&& terraform output -json ecs_ip | jq '.' -r")
	}
	c2_pass, err := GetTerraformOutput(Path+"/c2-ecs", "ecs_password")
	if err != nil {
		fmt.Printf("获取 ecs_password 失败,尝试使用备用方式: %v\n", err)
		c2_pass = utils.Command2("cd " + Path + " && cd c2-ecs" + "&& terraform output -json ecs_password | jq '.' -r")
	}

	cs_port := C2Port
	cs_pass := C2Pass
	cs_domain := Domain
	ssh_ip := c2_ip + ":22"

	// 去掉该死的换行符
	ssh_ip = strings.Replace(ssh_ip, "\n", "", -1)
	c2_pass = strings.Replace(c2_pass, "\n", "", -1)
	c2_ip = strings.Replace(c2_ip, "\n", "", -1)

	time.Sleep(time.Second * 60)

	// ssh上去起teamserver
	if Node != 0 {
		ipsum := utils.Command2("cd " + Path + "&& cd zone-node && cat ipsum.txt")
		ecs_main_ip := utils.Command2("cd " + Path + "&& cd zone-node && cat ecs_main_ip.txt")
		ipsum = strings.Replace(ipsum, "\n", "", -1)
		ecs_main_ip = strings.Replace(ecs_main_ip, "\n", "", -1)
		cscommand := "setsid ./teamserver -new " + cs_port + " " + c2_ip + " " + cs_pass + " " + cs_domain + " " + ipsum + " " + ecs_main_ip + " > /dev/null 2>&1 &"
		fmt.Println("cscommand: ", cscommand)
		err = utils.Gotossh("root", c2_pass, ssh_ip, cscommand)
		if err != nil {
			mod2.PrintOnError(err, "ssh 过程出现报错!自动销毁场景")
			RedcLog("ssh 过程出现报错!自动销毁场景")
			C2Destroy(Path, strconv.Itoa(Node), Domain)
			// 成功销毁场景后,删除 case 文件夹
			err = os.RemoveAll(Path)
			os.Exit(3)
		}
	} else {
		cscommand := "setsid ./teamserver -new " + cs_port + " " + c2_ip + " " + cs_pass + " " + cs_domain + " > /dev/null 2>&1 &"
		fmt.Println("cscommand: ", cscommand)
		err = utils.Gotossh("root", c2_pass, ssh_ip, cscommand)
		if err != nil {
			mod2.PrintOnError(err, "ssh 过程出现报错!自动销毁场景")
			RedcLog("ssh 过程出现报错!自动销毁场景")
			C2Destroy(Path, strconv.Itoa(Node), Domain)
			// 成功销毁场景后,删除 case 文件夹
			err = os.RemoveAll(Path)
			os.Exit(3)
		}
	}

	fmt.Println("ssh结束!")

	// 使用 terraform-exec 查看状态
	TfStatus(Path)

}

func C2Change(Path string) {

	// 重开rg
	fmt.Println("cd " + Path + " && bash deploy.sh -step3 " + strconv.Itoa(Node) + " " + Domain)
	err := utils.Command("cd " + Path + " && bash deploy.sh -step3 " + strconv.Itoa(Node) + " " + Domain)
	if err != nil {
		mod2.PrintOnError(err, "场景更改失败")
		os.Exit(3)
	}

	// 获得本地几个变量 - 使用 terraform-exec 获取 output
	c2_ip, err := GetTerraformOutput(Path+"/c2-ecs", "ecs_ip")
	if err != nil {
		fmt.Printf("获取 ecs_ip 失败,尝试使用备用方式: %v\n", err)
		c2_ip = utils.Command2("cd " + Path + " && cd c2-ecs" + "&& terraform output -json ecs_ip | jq '.' -r")
	}
	c2_pass, err := GetTerraformOutput(Path+"/c2-ecs", "ecs_password")
	if err != nil {
		fmt.Printf("获取 ecs_password 失败,尝试使用备用方式: %v\n", err)
		c2_pass = utils.Command2("cd " + Path + " && cd c2-ecs" + "&& terraform output -json ecs_password | jq '.' -r")
	}
	ipsum := utils.Command2("cd " + Path + "&& cd zone-node && cat ipsum.txt")
	ecs_main_ip := utils.Command2("cd " + Path + "&& cd zone-node && cat ecs_main_ip.txt")

	cs_port := C2Port
	cs_pass := C2Pass
	cs_domain := Domain
	ssh_ip := c2_ip + ":22"

	// 去掉该死的换行符
	ssh_ip = strings.Replace(ssh_ip, "\n", "", -1)
	c2_pass = strings.Replace(c2_pass, "\n", "", -1)
	c2_ip = strings.Replace(c2_ip, "\n", "", -1)
	ipsum = strings.Replace(ipsum, "\n", "", -1)
	ecs_main_ip = strings.Replace(ecs_main_ip, "\n", "", -1)
	cscommand := "setsid ./teamserver -changelistener1 " + cs_port + " " + c2_ip + " " + cs_pass + " " + cs_domain + " " + ipsum + " " + ecs_main_ip + " > /dev/null 2>&1 &"

	// ssh上去起teamserver
	utils.Gotossh("root", c2_pass, ssh_ip, cscommand)

}

func C2Destroy(Path string, Command1 string, Domain string) {

	fmt.Println("cd " + Path + " && bash deploy.sh -stop " + Command1 + " " + Domain)
	err := utils.Command("cd " + Path + " && bash deploy.sh -stop " + Command1 + " " + Domain)
	if err != nil {
		fmt.Println("场景销毁失败,第一次尝试!", err)
		RedcLog("场景销毁失败,第一次尝试!")

		// 如果初始化失败就再次尝试一次
		err2 := utils.Command("cd " + Path + " && bash deploy.sh -stop " + Command1 + " " + Domain)
		if err2 != nil {
			fmt.Println("场景销毁失败,第二次尝试!", err)
			RedcLog("场景销毁失败,第二次尝试!")

			// 第三次
			err3 := utils.Command("cd " + Path + " && bash deploy.sh -stop " + Command1 + " " + Domain)
			if err3 != nil {
				fmt.Println("场景销毁失败!")
				RedcLog("场景销毁失败")
				os.Exit(3)
			}
		}
	}

}

func AwsProxyApply(Path string) {

	fmt.Println("cd " + Path + " && bash deploy.sh -start " + strconv.Itoa(Node))
	err := utils.Command("cd " + Path + " && bash deploy.sh -start " + strconv.Itoa(Node))
	if err != nil {
		fmt.Println("场景创建失败!")
		RedcLog("场景创建失败!")
		os.Exit(3)
	}

}

func AwsProxyDestroy(Path string, Command1 string) {

	fmt.Println("cd " + Path + " && bash deploy.sh -stop " + Command1)
	err := utils.Command("cd " + Path + " && bash deploy.sh -stop " + Command1)
	if err != nil {
		fmt.Println("场景销毁失败,第一次尝试!", err)

		// 如果初始化失败就再次尝试一次
		err2 := utils.Command("cd " + Path + " && bash deploy.sh -stop " + Command1)
		if err2 != nil {
			fmt.Println("场景销毁失败,第二次尝试!", err)

			// 第三次
			err3 := utils.Command("cd " + Path + " && bash deploy.sh -stop " + Command1)
			if err3 != nil {
				fmt.Println("场景销毁失败!")
				RedcLog("场景销毁失败!")
				os.Exit(3)
			}
		}
	}

}

func AliyunProxyApply(Path string) {

	fmt.Println("cd " + Path + " && bash deploy.sh -start " + strconv.Itoa(Node))
	err := utils.Command("cd " + Path + " && bash deploy.sh -start " + strconv.Itoa(Node))
	if err != nil {
		fmt.Println("场景创建失败!")
		RedcLog("场景创建失败!")
		os.Exit(3)
	}

}

func AsmApply(Path string) {

	fmt.Println("cd " + Path + " && bash deploy.sh -start " + strconv.Itoa(Node))
	err := utils.Command("cd " + Path + " && bash deploy.sh -start " + strconv.Itoa(Node))
	if err != nil {
		fmt.Println("场景创建失败!")
		RedcLog("场景创建失败!")
		os.Exit(3)
	}

}

func AsmNodeApply(Path string) {

	fmt.Println("cd " + Path + " && bash deploy.sh -start " + strconv.Itoa(Node) + " " + Domain + " " + Domain2)
	err := utils.Command("cd " + Path + " && bash deploy.sh -start " + strconv.Itoa(Node) + " " + Domain + " " + Domain2)
	if err != nil {
		fmt.Println("场景创建失败!")
		RedcLog("场景创建失败!")
		os.Exit(3)
	}

}

func AliyunProxyChange(Path string) {

	// 重开proxy
	fmt.Println("cd " + Path + " && bash deploy.sh -change " + strconv.Itoa(Node))
	err := utils.Command("cd " + Path + " && bash deploy.sh -change " + strconv.Itoa(Node))
	if err != nil {
		fmt.Println("场景更改失败!")
		RedcLog("场景更改失败!")
		os.Exit(3)
	}

}

func AsmChange(Path string) {

	// 重开执行器
	fmt.Println("cd " + Path + " && bash deploy.sh -change " + strconv.Itoa(Node))
	err := utils.Command("cd " + Path + " && bash deploy.sh -change " + strconv.Itoa(Node))
	if err != nil {
		fmt.Println("场景更改失败!")
		RedcLog("场景更改失败!")
		os.Exit(3)
	}

}

func AliyunProxyDestroy(Path string, Command1 string) {

	fmt.Println("cd " + Path + " && bash deploy.sh -stop " + Command1)
	err := utils.Command("cd " + Path + " && bash deploy.sh -stop " + Command1)
	if err != nil {
		fmt.Println("场景销毁失败,第一次尝试!", err)

		// 如果初始化失败就再次尝试一次
		err2 := utils.Command("cd " + Path + " && bash deploy.sh -stop " + Command1)
		if err2 != nil {
			fmt.Println("场景销毁失败,第二次尝试!", err)

			// 第三次
			err3 := utils.Command("cd " + Path + " && bash deploy.sh -stop " + Command1)
			if err3 != nil {
				fmt.Println("场景销毁失败!")
				RedcLog("场景销毁失败!")
				os.Exit(3)
			}
		}
	}

}

func AsmDestroy(Path string, Command1 string) {

	fmt.Println("cd " + Path + " && bash deploy.sh -stop " + Command1)
	err := utils.Command("cd " + Path + " && bash deploy.sh -stop " + Command1)
	if err != nil {
		fmt.Println("场景销毁失败,第一次尝试!", err)

		// 如果初始化失败就再次尝试一次
		err2 := utils.Command("cd " + Path + " && bash deploy.sh -stop " + Command1)
		if err2 != nil {
			fmt.Println("场景销毁失败,第二次尝试!", err)

			// 第三次
			err3 := utils.Command("cd " + Path + " && bash deploy.sh -stop " + Command1)
			if err3 != nil {
				fmt.Println("场景销毁失败!")
				RedcLog("场景销毁失败!")
				os.Exit(3)
			}
		}
	}

}

func AsmNodeDestroy(Path string, Command1 string, Domain string, Domain2 string) {

	fmt.Println("cd " + Path + " && bash deploy.sh -stop " + Command1 + " " + Domain + " " + Domain2)
	err := utils.Command("cd " + Path + " && bash deploy.sh -stop " + Command1 + " " + Domain + " " + Domain2)
	if err != nil {
		fmt.Println("场景销毁失败,第一次尝试!", err)

		// 如果初始化失败就再次尝试一次
		err2 := utils.Command("cd " + Path + " && bash deploy.sh -stop " + Command1 + " " + Domain + " " + Domain2)
		if err2 != nil {
			fmt.Println("场景销毁失败,第二次尝试!", err)

			// 第三次
			err3 := utils.Command("cd " + Path + " && bash deploy.sh -stop " + Command1 + " " + Domain + " " + Domain2)
			if err3 != nil {
				fmt.Println("场景销毁失败!")
				RedcLog("场景销毁失败!")
				os.Exit(3)
			}
		}
	}

}

func DnslogApply(Path string) {

	fmt.Println("cd " + Path + " && bash deploy.sh -start " + Domain)
	err := utils.Command("cd " + Path + " && bash deploy.sh -start " + Domain)
	if err != nil {
		fmt.Println("场景创建失败!")
		RedcLog("场景创建失败!")
		os.Exit(3)
	}

}

func DnslogDestroy(Path string, Domain string) {

	fmt.Println("cd " + Path + " && bash deploy.sh -stop " + Domain)
	err := utils.Command("cd " + Path + " && bash deploy.sh -stop " + Domain)
	if err != nil {
		fmt.Println("场景销毁失败,第一次尝试!", err)

		// 如果初始化失败就再次尝试一次
		err2 := utils.Command("cd " + Path + " && bash deploy.sh -stop " + Domain)
		if err2 != nil {
			fmt.Println("场景销毁失败,第二次尝试!", err)

			// 第三次
			err3 := utils.Command("cd " + Path + " && bash deploy.sh -stop " + Domain)
			if err3 != nil {
				fmt.Println("场景销毁失败!")
				RedcLog("场景销毁失败!")
				os.Exit(3)
			}
		}
	}

}

func Base64Apply(Path string) {

	fmt.Println("cd " + Path + " && bash deploy.sh -start " + Base64Command)
	err := utils.Command("cd " + Path + " && bash deploy.sh -start " + Base64Command)
	if err != nil {
		fmt.Println("场景创建失败!")
		RedcLog("场景创建失败!")
		os.Exit(3)
	}

}

func Base64Destroy(Path string, Base64Command string) {

	fmt.Println("cd " + Path + " && bash deploy.sh -stop " + Base64Command)
	err := utils.Command("cd " + Path + " && bash deploy.sh -stop " + Base64Command)
	if err != nil {
		fmt.Println("场景销毁失败,第一次尝试!", err)

		// 如果初始化失败就再次尝试一次
		err2 := utils.Command("cd " + Path + " && bash deploy.sh -stop " + Base64Command)
		if err2 != nil {
			fmt.Println("场景销毁失败,第二次尝试!", err)

			// 第三次
			err3 := utils.Command("cd " + Path + " && bash deploy.sh -stop " + Base64Command)
			if err3 != nil {
				fmt.Println("场景销毁失败!")
				RedcLog("场景销毁失败!")
				os.Exit(3)
			}
		}
	}

}
