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

	"github.com/hashicorp/terraform-exec/tfexec"
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

// TfInit2 复制模版后再尝试初始化
func TfInit2(Path string) error {
	if err := TfInit(Path); err != nil {
		gologger.Debug().Msgf("初始化失败！: %v", err)
		// 无法初始化,删除 case 文件夹
		err = os.RemoveAll(Path)
		if err != nil {
			return fmt.Errorf("删除文件夹失败！")
		}
		return err
	}
	return nil
}

// RVar 统一转换接口，方便后续替换类型
func RVar(s ...string) []string {
	return s
}
func TfPlan(Path string, opts ...string) error {
	ctx, cancel := createContextWithTimeout()
	defer cancel()
	fmt.Printf("Planing terraform in %s\n", Path)
	te, err := NewTerraformExecutor(Path)
	if err != nil {
		return fmt.Errorf("执行失败: %s", err.Error())
	}
	o := ToPlan(opts)
	// 增加 plan 输出文件
	o = append(o, tfexec.Out(planPath))
	err = te.Plan(ctx, o...)
	if err != nil {
		gologger.Debug().Msgf("场景创建失败: %v", err)
		return err
	}
	err = te.ShowPlan(ctx)
	if err != nil {
		gologger.Error().Msgf("PLAN 信息展示失败！")
	}
	return nil
}
func TfApply(Path string, opts ...string) error {
	ctx, cancel := createContextWithTimeout()
	defer cancel()
	fmt.Printf("Applying terraform in %s\n", Path)
	te, err := NewTerraformExecutor(Path)
	if err != nil {
		return fmt.Errorf("场景启动失败,terraform未找到或配置错误: %w", err)
	}
	o := ToApply(opts)
	o = append(o, tfexec.DirOrPlan(planPath))
	err = te.Apply(ctx, ToApply(opts)...)
	if err != nil {
		gologger.Debug().Msgf("场景启动失败: %s", err.Error())
		return err
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

func TfOutput(Path string) error {
	ctx, cancel := createContextWithTimeout()
	defer cancel()

	te, err := NewTerraformExecutor(Path)
	if err != nil {
		return fmt.Errorf("TF可执行配置失败: %w", err)
	}

	outputs, err := te.Output(ctx)
	if err != nil {
		return fmt.Errorf("获取 Output 失败: %w", err)
	}

	if len(outputs) > 0 {
		gologger.Info().Msg("Terraform Outputs:")
		for k, v := range outputs {
			gologger.Info().Msgf("%s: %s", k, string(v.Value))
		}
	}
	return nil
}

func TfDestroy(Path string, opts []string) error {
	ctx, cancel := createContextWithTimeout()
	defer cancel()
	fmt.Printf("Destroying terraform resources in %s\n", Path)
	te, err := NewTerraformExecutor(Path)
	if err != nil {
		gologger.Error().Msgf("场景销毁失败,terraform未找到或配置错误: %v", err)
	}
	err = te.Destroy(ctx, ToDestroy(opts)...)
	if err != nil {
		gologger.Error().Msgf("场景销毁失败!请手动排查问题,path路径: %s,%v", Path, err)
	}
	return nil
}

func ToApply(v []string) []tfexec.ApplyOption {
	var opts []tfexec.ApplyOption
	for _, s := range v {
		opts = append(opts, tfexec.Var(s))
	}
	return opts
}

func ToPlan(v []string) []tfexec.PlanOption {
	var opts []tfexec.PlanOption
	for _, s := range v {
		opts = append(opts, tfexec.Var(s))
	}
	return opts
}

func ToDestroy(v []string) []tfexec.DestroyOption {
	var opts []tfexec.DestroyOption
	for _, s := range v {
		opts = append(opts, tfexec.Var(s))
	}
	return opts
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
