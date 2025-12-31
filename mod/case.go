package mod

import (
	"fmt"
	"math/rand"
	"os"
	"red-cloud/utils"
	"strconv"
	"text/tabwriter"
	"time"

	"github.com/gen2brain/beeep"
	uuid "github.com/satori/go.uuid"
	"gopkg.in/ini.v1"
)

func RandomName() string {
	var lastName = []string{
		"red", "blue", "yellow", "brown", "purple", "anger", "lazy", "shy", "huge", "rare",
		"fast", "stupid", "sluggish", "boring", "rigid", "rigorous", "clever", "dexterity",
		"white", "black", "dark", "idiot", "shiny", "friendly", "integrity", "happy", "sad",
		"lively", "lonely", "ugly", "leisurely", "calm", "young", "tenacious"}
	var firstName = []string{
		"pig", "cow", "sheep", "mouse", "dragon", "serpent", "tiger", "fox", "frog", "chicken",
		"fish", "shrimp", "hippocampus", "helicopter", "crab", "dolphin", "whale", "chinchilla",
		"bunny", "mole", "rabbit", "horse", "monkey", "dog", "shark", "panda", "bear", "lion",
		"rhino", "leopard", "giraffe", "deer", "wolf", "parrot", "camel", "antelope", "turtle", "zebra"}
	var lastNameLen = len(lastName)
	var firstNameLen = len(firstName)
	rand.Seed(time.Now().UnixNano())     //设置随机数种子
	var first string                     //名
	for i := 0; i <= rand.Intn(1); i++ { //随机产生2位或者3位的名
		first = fmt.Sprint(firstName[rand.Intn(firstNameLen-1)])
	}
	return fmt.Sprint(lastName[rand.Intn(lastNameLen-1)]) + first
}

func CaseCreate(ProjectPath string, CaseName string, User string, Name string) {
	// 创建新的 case 目录,这里不需要检测是否存在,因为名称是采用nanoID
	u1 := uuid.NewV4()

	// 复制tf文件
	err := utils.Dir("redc-templates/"+CaseName, ProjectPath+"/"+u1.String())
	if err != nil {
		fmt.Println("错误输入")
		os.Exit(3)
	} else {
		fmt.Println("Case 路径", u1.String())
		fmt.Println("关闭命令: ./redc -stop ", u1.String())
	}

	// 在次 init,防止万一
	TfInit2(ProjectPath + "/" + u1.String())

	fmt.Println("开始创建")

	// 部分场景单独处理
	if CaseName == "cs-49" || CaseName == "c2-new" || CaseName == "snowc2" {
		C2Apply(ProjectPath + "/" + u1.String())
	} else if CaseName == "aws-proxy" {
		AwsProxyApply(ProjectPath + "/" + u1.String())
	} else if CaseName == "aliyun-proxy" {
		AliyunProxyApply(ProjectPath + "/" + u1.String())
	} else if CaseName == "asm" {
		AsmApply(ProjectPath + "/" + u1.String())
	} else if CaseName == "asm-node" {
		AsmNodeApply(ProjectPath + "/" + u1.String())
	} else if CaseName == "dnslog" || CaseName == "xraydnslog" || CaseName == "interactsh" {
		if Domain == "360.com" {
			fmt.Printf("创建dnslog时,域名不可为默认值")
			RedcLog("创建失败,创建dnslog时,域名不可为默认值")
			os.Exit(3)
		}
		DnslogApply(ProjectPath + "/" + u1.String())
	} else if CaseName == "pss5" || CaseName == "frp" || CaseName == "frp-loki" || CaseName == "nps" {
		Base64Apply(ProjectPath + "/" + u1.String())
	} else {
		TfApply(ProjectPath + "/" + u1.String())
	}

	// 确认场景创建无误后,才会写入到配置文件中
	RedcLog("创建成功 " + ProjectPath + u1.String() + " " + CaseName)
	// case 入库
	filePath := ProjectPath + "/project.ini"
	cfg, err := ini.Load(filePath)
	if err != nil {
		fmt.Printf("Fail to read file: %v", err)
		os.Exit(3)
	}
	cfg.Section(u1.String()).Key("Operator").SetValue(User)
	cfg.Section(u1.String()).Key("Type").SetValue(CaseName)
	currentTime := time.Now().Format("2006-01-02 15:04:05")
	cfg.Section(u1.String()).Key("CreateTime").SetValue(currentTime)

	if Name == "" {
		Name = RandomName()
	}
	cfg.Section(u1.String()).Key("Name").SetValue(Name)

	// 部分场景单独处理
	if CaseName == "cs-49" || CaseName == "c2-new" || CaseName == "snowc2" {
		// 写入节点数量到ini文件
		cfg.Section(u1.String()).Key("Node").SetValue(strconv.Itoa(Node))
		cfg.Section(u1.String()).Key("Doamin").SetValue(Domain)
	} else if CaseName == "aws-proxy" {
		// 写入节点数量到ini文件
		cfg.Section(u1.String()).Key("Node").SetValue(strconv.Itoa(Node))
	} else if CaseName == "aliyun-proxy" {
		// 写入节点数量到ini文件
		cfg.Section(u1.String()).Key("Node").SetValue(strconv.Itoa(Node))
	} else if CaseName == "asm" {
		// 写入节点数量到ini文件
		cfg.Section(u1.String()).Key("Node").SetValue(strconv.Itoa(Node))
	} else if CaseName == "asm-node" {
		// 写入节点数量到ini文件
		cfg.Section(u1.String()).Key("Node").SetValue(strconv.Itoa(Node))
		cfg.Section(u1.String()).Key("Doamin").SetValue(Domain)
		cfg.Section(u1.String()).Key("Doamin2").SetValue(Domain2)
	} else if CaseName == "dnslog" || CaseName == "xraydnslog" || CaseName == "interactsh" {
		// 写入域名到ini文件
		cfg.Section(u1.String()).Key("Doamin").SetValue(Domain)
	} else if CaseName == "pss5" || CaseName == "frp" || CaseName == "frp-loki" || CaseName == "nps" {
		// 写入base64命令到ini文件
		cfg.Section(u1.String()).Key("Base64Command").SetValue(Base64Command)
	}
	err = cfg.SaveTo(filePath)
	if err != nil {
		fmt.Printf("写入 ini 时失败: %v", err)
		RedcLog("写入 ini 时失败")
		os.Exit(3)
	}

	fmt.Println("Case 路径", u1.String())
	fmt.Println("关闭命令: ./redc -stop ", u1.String())
	_ = beeep.Notify("redc", fmt.Sprintf("%v 场景创建完毕!", CaseName), "assets/information.png")

}

func CaseStop(ProjectPath string, UUID string) {

	filePath := ProjectPath + "/project.ini"
	cfg, err := ini.Load(filePath)
	if err != nil {
		fmt.Printf("Fail to read file: %v", err)
		os.Exit(3)
	}

	//fmt.Println(UUID)
	if cfg.Section(UUID).Key("Type").String() == "cs-49" || cfg.Section(UUID).Key("Type").String() == "c2-new" || cfg.Section(UUID).Key("Type").String() == "snowc2" {
		C2Destroy(ProjectPath+"/"+UUID,
			cfg.Section(UUID).Key("Node").String(),
			cfg.Section(UUID).Key("Domain").String())
	} else if cfg.Section(UUID).Key("Type").String() == "aws-proxy" {
		AwsProxyDestroy(ProjectPath+"/"+UUID,
			cfg.Section(UUID).Key("Node").String())
	} else if cfg.Section(UUID).Key("Type").String() == "aliyun-proxy" {
		AliyunProxyDestroy(ProjectPath+"/"+UUID,
			cfg.Section(UUID).Key("Node").String())
	} else if cfg.Section(UUID).Key("Type").String() == "asm" {
		AsmDestroy(ProjectPath+"/"+UUID,
			cfg.Section(UUID).Key("Node").String())
	} else if cfg.Section(UUID).Key("Type").String() == "asm-node" {
		AsmNodeDestroy(ProjectPath+"/"+UUID,
			cfg.Section(UUID).Key("Node").String(),
			cfg.Section(UUID).Key("Domain").String(),
			cfg.Section(UUID).Key("Domain2").String())
	} else if cfg.Section(UUID).Key("Type").String() == "dnslog" || cfg.Section(UUID).Key("Type").String() == "xraydnslog" || cfg.Section(UUID).Key("Type").String() == "interactsh" {
		DnslogDestroy(ProjectPath+"/"+UUID, cfg.Section(UUID).Key("Domain").String())
	} else if cfg.Section(UUID).Key("Type").String() == "pss5" || cfg.Section(UUID).Key("Type").String() == "frp" || cfg.Section(UUID).Key("Type").String() == "frp-loki" || cfg.Section(UUID).Key("Type").String() == "nps" {
		Base64Destroy(ProjectPath+"/"+UUID, cfg.Section(UUID).Key("Base64Command").String())
	} else {
		TfDestroy(ProjectPath + "/" + UUID)
	}

	// 成功销毁场景后,删除 case 文件夹
	err = os.RemoveAll(ProjectPath + "/" + UUID)
	if err != nil {
		fmt.Println(err)
		os.Exit(3)
	}

	// case 删除
	cfg2, err := ini.Load(filePath)
	if err != nil {
		fmt.Printf("Fail to read file: %v", err)
		os.Exit(3)
	}
	cfg2.DeleteSection(UUID)
	err = cfg2.SaveTo(filePath)
	if err != nil {
		fmt.Printf("修改 ini 时失败: %v", err)
		RedcLog("修改 ini 时失败")
		os.Exit(3)
	}

}

func CaseKill(ProjectPath string, UUID string) {

	filePath := ProjectPath + "/project.ini"
	cfg, err := ini.Load(filePath)
	if err != nil {
		fmt.Printf("Fail to read file: %v", err)
		os.Exit(3)
	}
	// 在次 init,防止万一
	dirs := utils.ChechDirMain(ProjectPath + "/" + UUID)
	for _, v := range dirs {
		err := utils.CheckFileName(v, "tf")
		if err {
			TfInit2(v)
		}
	}

	if cfg.Section(UUID).Key("Type").String() == "cs-49" || cfg.Section(UUID).Key("Type").String() == "c2-new" || cfg.Section(UUID).Key("Type").String() == "snowc2" {
		C2Destroy(ProjectPath+"/"+UUID,
			cfg.Section(UUID).Key("Node").String(),
			cfg.Section(UUID).Key("Domain").String())
	} else if cfg.Section(UUID).Key("Type").String() == "aws-proxy" {
		AwsProxyDestroy(ProjectPath+"/"+UUID,
			cfg.Section(UUID).Key("Node").String())
	} else if cfg.Section(UUID).Key("Type").String() == "aliyun-proxy" {
		AliyunProxyDestroy(ProjectPath+"/"+UUID,
			cfg.Section(UUID).Key("Node").String())
	} else if cfg.Section(UUID).Key("Type").String() == "asm" {
		AsmDestroy(ProjectPath+"/"+UUID,
			cfg.Section(UUID).Key("Node").String())
	} else if cfg.Section(UUID).Key("Type").String() == "asm-node" {
		AsmNodeDestroy(ProjectPath+"/"+UUID,
			cfg.Section(UUID).Key("Node").String(),
			cfg.Section(UUID).Key("Domain").String(),
			cfg.Section(UUID).Key("Domain2").String())
	} else if cfg.Section(UUID).Key("Type").String() == "dnslog" || cfg.Section(UUID).Key("Type").String() == "xraydnslog" || cfg.Section(UUID).Key("Type").String() == "interactsh" {
		DnslogDestroy(ProjectPath+"/"+UUID, cfg.Section(UUID).Key("Domain").String())
	} else if cfg.Section(UUID).Key("Type").String() == "pss5" || cfg.Section(UUID).Key("Type").String() == "frp" || cfg.Section(UUID).Key("Type").String() == "frp-loki" || cfg.Section(UUID).Key("Type").String() == "nps" {
		Base64Destroy(ProjectPath+"/"+UUID, cfg.Section(UUID).Key("Base64Command").String())
	} else {
		TfDestroy(ProjectPath + "/" + UUID)
	}

	// 成功销毁场景后,删除 case 文件夹
	err = os.RemoveAll(ProjectPath + "/" + UUID)
	if err != nil {
		fmt.Println(err)
		os.Exit(3)
	}

	// case 删除
	cfg2, err := ini.Load(filePath)
	if err != nil {
		fmt.Printf("Fail to read file: %v", err)
		os.Exit(3)
	}
	cfg2.DeleteSection(UUID)
	err = cfg2.SaveTo(filePath)
	if err != nil {
		fmt.Printf("修改 ini 时失败: %v", err)
		_ = beeep.Notify("redc", fmt.Sprintf("修改 ini 时失败: %v", err), "assets/information.png")
		os.Exit(3)
	}
}

func CaseChange(ProjectPath string, UUID string) {

	filePath := ProjectPath + "/project.ini"
	cfg, err := ini.Load(filePath)
	if err != nil {
		fmt.Printf("Fail to read file: %v", err)
		os.Exit(3)
	}

	if cfg.Section(UUID).Key("Type").String() == "cs-49" || cfg.Section(UUID).Key("Type").String() == "c2-new" || cfg.Section(UUID).Key("Type").String() == "snowc2" {
		C2Change(ProjectPath + "/" + UUID)
	} else if cfg.Section(UUID).Key("Type").String() == "aliyun-proxy" {
		AliyunProxyChange(ProjectPath + "/" + UUID)
	} else if cfg.Section(UUID).Key("Type").String() == "asm" {
		AsmChange(ProjectPath + "/" + UUID)
	} else {
		fmt.Printf("不适用与当前场景")
		os.Exit(3)
	}

}

func CaseStatus(ProjectPath string, UUID string) {
	filePath := ProjectPath + "/project.ini"
	cfg, err := ini.Load(filePath)
	if err != nil {
		fmt.Printf("Fail to read file: %v", err)
		os.Exit(3)
	}

	fmt.Println("操作人员:", cfg.Section(UUID).Key("Operator").String())
	fmt.Println("项目名称:", cfg.Section(UUID).Key("Name").String())
	fmt.Println("场景类型:", cfg.Section(UUID).Key("Type").String())
	fmt.Println("创建时间:", cfg.Section(UUID).Key("CreateTime").String())

	TfStatus(ProjectPath + "/" + UUID)

}

func CaseList(ProjectPath string) {
	filePath := ProjectPath + "/project.ini"
	cfg, err := ini.Load(filePath)
	if err != nil {
		fmt.Printf("Fail to read file: %v", err)
		os.Exit(3)
	}

	w := tabwriter.NewWriter(os.Stdout, 15, 0, 1, ' ',
		tabwriter.AlignRight)
	fmt.Fprintln(w, "UUID\tType\tName\tOperator\tCreateTime\t")
	for i := 1; i < len(cfg.SectionStrings()); i++ {
		if cfg.Section(cfg.SectionStrings()[i]).Key("Operator").String() == U || U == "system" {
			fmt.Fprintln(w, cfg.SectionStrings()[i], "\t", cfg.Section(cfg.SectionStrings()[i]).Key("Type").String(), "\t", cfg.Section(cfg.SectionStrings()[i]).Key("Name").String(), "\t", cfg.Section(cfg.SectionStrings()[i]).Key("Operator").String(), "\t", cfg.Section(cfg.SectionStrings()[i]).Key("CreateTime").String())
		}
	}
	err = w.Flush()
	if err != nil {
		fmt.Printf("打印失败: %v", err)
		os.Exit(3)
	}

	RandomName()
}

func CheckUser(ProjectPath string, UUID string) {
	filePath := ProjectPath + "/project.ini"
	cfg, err := ini.Load(filePath)
	if err != nil {
		fmt.Printf("Fail to read file: %v", err)
		os.Exit(3)
	}

	// 鉴权
	if cfg.Section(UUID).Key("Operator").String() != U && U != "system" {
		fmt.Printf("用户 %v 无权访问 %v", U, UUID)
		os.Exit(3)
	}

}
