package mod

import (
	"fmt"
	"github.com/gen2brain/beeep"
	"os"
	"path/filepath"
	"red-cloud/mod2"
	"red-cloud/utils"
	"strconv"
	"strings"
	"time"
)

// notifyError sends a notification and exits with failure code
func notifyError(message string, err error) {
	fmt.Printf("%s: %v\n", message, err)
	_ = beeep.Notify("redc", fmt.Sprintf("%s: %v", message, err), "assets/information.png")
	os.Exit(ExitCodeFailure)
}

// readFileContent reads a file and returns its content with newlines trimmed
func readFileContent(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// 第一次初始化 - 使用 terraform-exec (无fallback)
func TfInit0(Path string) {
	ctx, cancel := createContextWithTimeout()
	defer cancel()

	fmt.Printf("Initializing terraform in %s\n", Path)

	te, err := NewTerraformExecutor(Path)
	if err != nil {
		notifyError("场景初始化失败,terraform未找到或配置错误", err)
	}

	// Use retry logic with InitRetries constant
	err = retryOperation(ctx, te.Init, InitRetries)
	if err != nil {
		notifyError("场景初始化失败,请检查网络连接!", err)
	}
}

// 复制后的初始化 - 使用 terraform-exec (无fallback)
func TfInit(Path string) {
	ctx, cancel := createContextWithTimeout()
	defer cancel()

	fmt.Printf("Initializing terraform in %s\n", Path)

	te, err := NewTerraformExecutor(Path)
	if err != nil {
		os.RemoveAll(Path)
		notifyError("场景初始化失败,terraform未找到或配置错误", err)
	}

	// Use retry logic with InitRetries constant
	err = retryOperation(ctx, te.Init, InitRetries)
	if err != nil {
		fmt.Println("场景初始化失败,请检查网络连接!", err)
		// Remove the case folder on failure
		os.RemoveAll(Path)
		os.Exit(ExitCodeFailure)
	}
}

// TfApply 使用 terraform-exec 执行 apply (无fallback)
func TfApply(Path string) {
	ctx, cancel := createContextWithTimeout()
	defer cancel()

	fmt.Printf("Applying terraform in %s\n", Path)

	te, err := NewTerraformExecutor(Path)
	if err != nil {
		notifyError("场景创建失败,terraform未找到或配置错误", err)
	}

	err = te.Apply(ctx)
	if err != nil {
		fmt.Println("场景创建失败!尝试重新创建!")
		// Try to destroy first
		te.Destroy(ctx)
		// Retry apply
		err2 := te.Apply(ctx)
		if err2 != nil {
			notifyError("场景创建第二次失败!请手动排查问题,path路径: "+Path, err2)
		}
	}
}

// TfStatus 使用 terraform-exec 查看状态 (无fallback)
func TfStatus(Path string) {
	ctx, cancel := createContextWithTimeout()
	defer cancel()

	fmt.Printf("Getting terraform status in %s\n", Path)

	te, err := NewTerraformExecutor(Path)
	if err != nil {
		notifyError("场景状态查询失败,terraform未找到或配置错误", err)
	}

	err = te.Show(ctx)
	if err != nil {
		notifyError("场景状态查询失败!请手动排查问题,path路径: "+Path, err)
	}
}

// TfDestroy 使用 terraform-exec 销毁资源 (无fallback)
func TfDestroy(Path string) {
	ctx, cancel := createContextWithTimeout()
	defer cancel()

	fmt.Printf("Destroying terraform resources in %s\n", Path)

	te, err := NewTerraformExecutor(Path)
	if err != nil {
		notifyError("场景销毁失败,terraform未找到或配置错误", err)
	}

	// Use retry logic with MaxRetries
	err = retryOperation(ctx, te.Destroy, MaxRetries)
	if err != nil {
		notifyError("场景销毁多次重试失败!请手动排查问题,path路径: "+Path, err)
	}
}

// applyTerraformInDir applies terraform in a specific subdirectory
func applyTerraformInDir(baseDir string, subDir string) error {
	ctx, cancel := createContextWithTimeout()
	defer cancel()

	path := baseDir + "/" + subDir
	fmt.Printf("Applying terraform in %s\n", path)

	te, err := NewTerraformExecutor(path)
	if err != nil {
		return fmt.Errorf("failed to create executor for %s: %w", path, err)
	}

	return te.Apply(ctx)
}

// destroyTerraformInDir destroys terraform resources in a specific subdirectory
func destroyTerraformInDir(baseDir string, subDir string) error {
	ctx, cancel := createContextWithTimeout()
	defer cancel()

	path := baseDir + "/" + subDir
	fmt.Printf("Destroying terraform in %s\n", path)

	te, err := NewTerraformExecutor(path)
	if err != nil {
		return fmt.Errorf("failed to create executor for %s: %w", path, err)
	}

	return retryOperation(ctx, te.Destroy, MaxRetries)
}

func C2Apply(Path string) {
	// 先开c2 (step1 - apply in c2-ecs directory)
	err := applyTerraformInDir(Path, "c2-ecs")
	if err != nil {
		fmt.Println("场景创建失败,自动销毁场景!", err)
		RedcLog("场景创建失败,自动销毁场景!")
		C2Destroy(Path, strconv.Itoa(Node), Domain)
		// 成功销毁场景后,删除 case 文件夹
		os.RemoveAll(Path)
		os.Exit(ExitCodeFailure)
	}

	// 开rg (step2 - apply in zone-node directory if Node != 0)
	if Node != 0 {
		err = applyTerraformInDir(Path, "zone-node")
		if err != nil {
			fmt.Println("场景创建失败,自动销毁场景!", err)
			RedcLog("场景创建失败,自动销毁场景!")
			C2Destroy(Path, strconv.Itoa(Node), Domain)
			// 成功销毁场景后,删除 case 文件夹
			os.RemoveAll(Path)
			os.Exit(ExitCodeFailure)
		}
	}

	// 获得本地几个变量 - 使用 terraform-exec 获取 output
	c2_ip, err := GetTerraformOutput(Path+"/c2-ecs", "ecs_ip")
	if err != nil {
		fmt.Printf("获取 ecs_ip 失败: %v\n", err)
		RedcLog("获取 ecs_ip 失败")
		os.Exit(ExitCodeFailure)
	}
	c2_pass, err := GetTerraformOutput(Path+"/c2-ecs", "ecs_password")
	if err != nil {
		fmt.Printf("获取 ecs_password 失败: %v\n", err)
		RedcLog("获取 ecs_password 失败")
		os.Exit(ExitCodeFailure)
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
		ipsum, err := readFileContent(filepath.Join(Path, "zone-node", "ipsum.txt"))
		if err != nil {
			fmt.Printf("读取 ipsum.txt 失败: %v\n", err)
			RedcLog("读取 ipsum.txt 失败")
			os.Exit(ExitCodeFailure)
		}
		ecs_main_ip, err := readFileContent(filepath.Join(Path, "zone-node", "ecs_main_ip.txt"))
		if err != nil {
			fmt.Printf("读取 ecs_main_ip.txt 失败: %v\n", err)
			RedcLog("读取 ecs_main_ip.txt 失败")
			os.Exit(ExitCodeFailure)
		}
		cscommand := "setsid ./teamserver -new " + cs_port + " " + c2_ip + " " + cs_pass + " " + cs_domain + " " + ipsum + " " + ecs_main_ip + " > /dev/null 2>&1 &"
		fmt.Println("cscommand: ", cscommand)
		err = utils.Gotossh("root", c2_pass, ssh_ip, cscommand)
		if err != nil {
			mod2.PrintOnError(err, "ssh 过程出现报错!自动销毁场景")
			RedcLog("ssh 过程出现报错!自动销毁场景")
			C2Destroy(Path, strconv.Itoa(Node), Domain)
			// 成功销毁场景后,删除 case 文件夹
			os.RemoveAll(Path)
			os.Exit(ExitCodeFailure)
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
			os.RemoveAll(Path)
			os.Exit(ExitCodeFailure)
		}
	}

	fmt.Println("ssh结束!")

	// 使用 terraform-exec 查看状态
	TfStatus(Path)

}

func C2Change(Path string) {
	// 重开rg (step3 - recreate zone-node)
	if Node != 0 {
		// Destroy first
		err := destroyTerraformInDir(Path, "zone-node")
		if err != nil {
			fmt.Printf("场景更改失败(销毁): %v\n", err)
			mod2.PrintOnError(err, "场景更改失败")
			os.Exit(ExitCodeFailure)
		}
		// Apply again
		err = applyTerraformInDir(Path, "zone-node")
		if err != nil {
			fmt.Printf("场景更改失败(创建): %v\n", err)
			mod2.PrintOnError(err, "场景更改失败")
			os.Exit(ExitCodeFailure)
		}
	}

	// 获得本地几个变量 - 使用 terraform-exec 获取 output
	c2_ip, err := GetTerraformOutput(Path+"/c2-ecs", "ecs_ip")
	if err != nil {
		fmt.Printf("获取 ecs_ip 失败: %v\n", err)
		RedcLog("获取 ecs_ip 失败")
		os.Exit(ExitCodeFailure)
	}
	c2_pass, err := GetTerraformOutput(Path+"/c2-ecs", "ecs_password")
	if err != nil {
		fmt.Printf("获取 ecs_password 失败: %v\n", err)
		RedcLog("获取 ecs_password 失败")
		os.Exit(ExitCodeFailure)
	}
	ipsum, err := readFileContent(filepath.Join(Path, "zone-node", "ipsum.txt"))
	if err != nil {
		fmt.Printf("读取 ipsum.txt 失败: %v\n", err)
		RedcLog("读取 ipsum.txt 失败")
		os.Exit(ExitCodeFailure)
	}
	ecs_main_ip, err := readFileContent(filepath.Join(Path, "zone-node", "ecs_main_ip.txt"))
	if err != nil {
		fmt.Printf("读取 ecs_main_ip.txt 失败: %v\n", err)
		RedcLog("读取 ecs_main_ip.txt 失败")
		os.Exit(ExitCodeFailure)
	}

	cs_port := C2Port
	cs_pass := C2Pass
	cs_domain := Domain
	ssh_ip := c2_ip + ":22"

	// 去掉该死的换行符
	ssh_ip = strings.Replace(ssh_ip, "\n", "", -1)
	c2_pass = strings.Replace(c2_pass, "\n", "", -1)
	c2_ip = strings.Replace(c2_ip, "\n", "", -1)
	cscommand := "setsid ./teamserver -changelistener1 " + cs_port + " " + c2_ip + " " + cs_pass + " " + cs_domain + " " + ipsum + " " + ecs_main_ip + " > /dev/null 2>&1 &"

	// ssh上去起teamserver
	utils.Gotossh("root", c2_pass, ssh_ip, cscommand)

}

func C2Destroy(Path string, Command1 string, Domain string) {
	// Parse the node count
	nodeCount, _ := strconv.Atoi(Command1)

	// Destroy zone-node if it exists
	if nodeCount != 0 {
		err := destroyTerraformInDir(Path, "zone-node")
		if err != nil {
			fmt.Println("zone-node销毁失败,第一次尝试!", err)
			RedcLog("zone-node销毁失败,第一次尝试!")
			// Retry
			err2 := destroyTerraformInDir(Path, "zone-node")
			if err2 != nil {
				fmt.Println("zone-node销毁失败,第二次尝试!", err2)
				RedcLog("zone-node销毁失败,第二次尝试!")
				// One more try
				err3 := destroyTerraformInDir(Path, "zone-node")
				if err3 != nil {
					fmt.Println("zone-node销毁失败!")
					RedcLog("zone-node销毁失败")
					// Continue to destroy c2-ecs anyway
				}
			}
		}
	}

	// Destroy c2-ecs
	err := destroyTerraformInDir(Path, "c2-ecs")
	if err != nil {
		fmt.Println("c2-ecs销毁失败,第一次尝试!", err)
		RedcLog("c2-ecs销毁失败,第一次尝试!")
		// Retry
		err2 := destroyTerraformInDir(Path, "c2-ecs")
		if err2 != nil {
			fmt.Println("c2-ecs销毁失败,第二次尝试!", err2)
			RedcLog("c2-ecs销毁失败,第二次尝试!")
			// One more try
			err3 := destroyTerraformInDir(Path, "c2-ecs")
			if err3 != nil {
				fmt.Println("场景销毁失败!")
				RedcLog("场景销毁失败")
				os.Exit(ExitCodeFailure)
			}
		}
	}

}

func AwsProxyApply(Path string) {
	TfApply(Path)
}

func AwsProxyDestroy(Path string, Command1 string) {
	TfDestroy(Path)
}

func AliyunProxyApply(Path string) {
	TfApply(Path)
}

func AsmApply(Path string) {
	TfApply(Path)
}

func AsmNodeApply(Path string) {
	TfApply(Path)
}

func AliyunProxyChange(Path string) {
	// 重开proxy - destroy and apply
	TfDestroy(Path)
	TfApply(Path)
}

func AsmChange(Path string) {
	// 重开执行器 - destroy and apply
	TfDestroy(Path)
	TfApply(Path)
}

func AliyunProxyDestroy(Path string, Command1 string) {
	TfDestroy(Path)
}

func AsmDestroy(Path string, Command1 string) {
	TfDestroy(Path)
}

func AsmNodeDestroy(Path string, Command1 string, Domain string, Domain2 string) {
	TfDestroy(Path)
}

func DnslogApply(Path string) {
	TfApply(Path)
}

func DnslogDestroy(Path string, Domain string) {
	TfDestroy(Path)
}

func Base64Apply(Path string) {
	TfApply(Path)
}

func Base64Destroy(Path string, Base64Command string) {
	TfDestroy(Path)
}
