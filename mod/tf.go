package mod

import (
	"fmt"
	"os"
	"red-cloud/mod/gologger"
	"red-cloud/mod2"
	"red-cloud/utils"
	"strconv"
	"strings"
	"time"
)

// readFileContent reads a file and returns its content with newlines trimmed
func readFileContent(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// TfInit 初始化场景
func TfInit(Path string) error {
	ctx, cancel := createContextWithTimeout()
	defer cancel()
	gologger.Info().Msgf("正在初始化场景「%s」", Path)

	// 寻找执行程序
	te, err := NewTerraformExecutor(Path)
	if err != nil {
		return fmt.Errorf("TF可执行配置失败: %w", err)
	}

	// Use retry logic with InitRetries constant
	err = retryOperation(ctx, te.Init, InitRetries)
	if err != nil {
		return fmt.Errorf("请检查网络连接: %w", err)
	}
	return nil
}

// TfInit2 复制后的初始化 遗留函数
func TfInit2(Path string) {

	if err := TfInit(Path); err != nil {
		// 无法初始化,删除 case 文件夹
		err = os.RemoveAll(Path)
		if err != nil {
			fmt.Println(err)
			os.Exit(3)
		}
		os.Exit(3)
	}
}

func TfApply(Path string) error {
	ctx, cancel := createContextWithTimeout()
	defer cancel()
	fmt.Printf("Applying terraform in %s\n", Path)
	te, err := NewTerraformExecutor(Path)
	if err != nil {
		return fmt.Errorf("场景创建失败,terraform未找到或配置错误: %w", err)
	}

	err = te.Apply(ctx)
	if err != nil {
		gologger.Error().Msgf("场景创建失败，正在尝试第二次创建: %v", err)
		err = te.Destroy(ctx)
		if err != nil {
			gologger.Error().Msgf("场景删除失败！: %v", err)
			return err
		}
		// Retry apply
		err2 := te.Apply(ctx)
		if err2 != nil {
			return fmt.Errorf("场景创建第二次失败!请手动排查问题,path路径: %s : %w", Path, err2)
		}
	}
	return nil
}

func TfStatus(Path string) {
	ctx, cancel := createContextWithTimeout()
	defer cancel()

	fmt.Printf("Getting terraform status in %s\n", Path)

	te, err := NewTerraformExecutor(Path)
	if err != nil {
		gologger.Error().Msgf("场景状态查询失败,terraform未找到或配置错误: %v", err)
	}

	err = te.Show(ctx)
	if err != nil {
		gologger.Error().Msgf("场景状态查询失败!请手动排查问题,path路径: %s,%v", Path, err)
	}
}

func TfDestroy(Path string) {
	ctx, cancel := createContextWithTimeout()
	defer cancel()
	fmt.Printf("Destroying terraform resources in %s\n", Path)
	te, err := NewTerraformExecutor(Path)
	if err != nil {
		gologger.Error().Msgf("场景销毁失败,terraform未找到或配置错误: %v", err)
	}
	// Use retry logic with MaxRetries
	err = retryOperation(ctx, te.Destroy, MaxRetries)
	if err != nil {
		gologger.Error().Msgf("场景销毁失败!请手动排查问题,path路径: %s,%v", Path, err)
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

	// 获得本地几个变量
	c2_ip := utils.Command2("cd " + Path + " && cd c2-ecs" + "&& terraform output -json ecs_ip | jq '.' -r")
	c2_pass := utils.Command2("cd " + Path + " && cd c2-ecs" + "&& terraform output -json ecs_password | jq '.' -r")

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

	err = utils.Command("cd " + Path + " && bash deploy.sh -status")

	if err != nil {
		mod2.PrintOnError(err, "场景创建失败")
		RedcLog("场景创建失败")
		os.Exit(3)
	}

}

func C2Change(Path string) {

	// 重开rg
	fmt.Println("cd " + Path + " && bash deploy.sh -step3 " + strconv.Itoa(Node) + " " + Domain)
	err := utils.Command("cd " + Path + " && bash deploy.sh -step3 " + strconv.Itoa(Node) + " " + Domain)
	if err != nil {
		mod2.PrintOnError(err, "场景更改失败")
		os.Exit(3)
	}

	// 获得本地几个变量
	c2_ip := utils.Command2("cd " + Path + " && cd c2-ecs" + "&& terraform output -json ecs_ip | jq '.' -r")
	c2_pass := utils.Command2("cd " + Path + " && cd c2-ecs" + "&& terraform output -json ecs_password | jq '.' -r")
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
