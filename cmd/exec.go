package cmd

import (
	"fmt"
	"red-cloud/mod/gologger"
	"red-cloud/utils/sshutil"
	"strings"

	"github.com/spf13/cobra"
)

var (
	execInteractive bool // 是否启用交互模式
)

// GetInstanceInfoFromTF 预留的 TF 信息获取函数 (Requirement #2)
func GetInstanceInfoFromTF(id string) (*sshutil.SSHConfig, error) {
	c, err := redcProject.GetCase(id)
	if err != nil {
		return nil, fmt.Errorf("操作失败: 找不到 ID 为「%s」的场景\n错误: %s", id, err)
	}
	s, err := c.GetSSHConfig()
	return s, nil

}

var execCmd = &cobra.Command{
	Use:   "exec [id] [command]",
	Short: "在目标机器执行命令",
	Example: `  redc exec [id] whoami
  redc exec -t [id] bash`,
	//Args: cobra.MinimumNArgs(2), // 需要 ID 和 命令
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 2 {
			cmd.Help()
			return // 直接 return，返回 nil，这样 main 函数就不会打印 [ERR]
		}
		id := args[0]
		// 将剩余参数组合成命令
		commandStr := strings.Join(args[1:], " ")

		client, err := getSSHClient(id)
		if err != nil {
			gologger.Error().Msgf("连接失败: %v", err)
			return
		}
		defer client.Close()

		// 根据是否交互式调用不同函数 (Requirement #1, #4)
		if execInteractive {
			gologger.Info().Msgf("正在启动交互式命令")
			err = client.RunInteractiveShell(commandStr)
		} else {
			err = client.RunCommand(commandStr)
		}

		if err != nil {
			gologger.Error().Msgf("执行出错: %v", err)
		}
	},
}

var cpCmd = &cobra.Command{
	Use:   "cp [src] [dest]",
	Short: "在本地和远程机器间复制文件",
	Example: `  redc cp ./tool [id]:/tmp/tool
  redc cp [id]:/var/log/syslog ./local_log`,
	//Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 2 {
			cmd.Help()
			return // 直接 return，返回 nil，这样 main 函数就不会打印 [ERR]
		}
		srcArg := args[0]
		destArg := args[1]

		srcID, srcPath, srcRemote := parseCpArg(srcArg)
		destID, destPath, destRemote := parseCpArg(destArg)

		// 逻辑校验：不能两边都是远程，也不能两边都是本地 (简化版)
		if srcRemote && destRemote {
			gologger.Error().Msg("不支持远程到远程的直接复制")
			return
		}
		if !srcRemote && !destRemote {
			gologger.Error().Msg("请使用系统 cp 命令进行本地复制")
			return
		}

		// 场景 1: 上传 (Local -> Remote)
		if !srcRemote && destRemote {
			gologger.Info().Msgf("Uploading %s to %s:%s", srcArg, destID, destPath)

			client, err := getSSHClient(destID)
			if err != nil {
				gologger.Error().Msgf("连接失败: %v", err)
				return
			}
			defer client.Close()

			if err := client.Upload(srcArg, destPath); err != nil {
				gologger.Error().Msgf("上传失败: %v", err)
			} else {
				gologger.Info().Msg("上传成功")
			}
		}

		// 场景 2: 下载 (Remote -> Local)
		if srcRemote && !destRemote {
			gologger.Info().Msgf("Downloading %s:%s to %s", srcID, srcPath, destArg)

			client, err := getSSHClient(srcID)
			if err != nil {
				gologger.Error().Msgf("连接失败: %v", err)
				return
			}
			defer client.Close()

			if err := client.Download(srcPath, destArg); err != nil {
				gologger.Error().Msgf("下载失败: %v", err)
			} else {
				gologger.Info().Msg("下载成功")
			}
		}
	},
}

func init() {
	execCmd.Flags().BoolVarP(&execInteractive, "tty", "t", false, "Allocate a pseudo-TTY (Interactive mode)")
	rootCmd.AddCommand(execCmd)
	rootCmd.AddCommand(cpCmd)
}

// Helper: 获取 SSH Client
func getSSHClient(id string) (*sshutil.Client, error) {
	info, err := GetInstanceInfoFromTF(id)
	if err != nil {
		return nil, err
	}

	return sshutil.NewClient(info)
}

// parseCpArg 解析 cp 参数，判断是本地路径还是远程路径
// 格式 mimic docker: id:/path/to/file
func parseCpArg(arg string) (id string, path string, isRemote bool) {
	if strings.Contains(arg, ":") {
		parts := strings.SplitN(arg, ":", 2)
		return parts[0], parts[1], true
	}
	return "", arg, false
}
