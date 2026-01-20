package cmd

import (
	"strings"
	"time"

	"red-cloud/mod"
	"red-cloud/mod/gologger"

	"github.com/spf13/cobra"
)

var opts struct {
	Registry, Dir string
	Force         bool
	Timeout       time.Duration // 新增超时配置
}

var pullCmd = &cobra.Command{
	Use:   "pull <image>[:tag]",
	Short: "Pull a template from registry",
	RunE:  runPull,
}

func init() {
	pullCmd.Flags().StringVarP(&opts.Registry, "registry", "r", "https://redc.wgpsec.org", "Registry URL")
	pullCmd.Flags().StringVarP(&opts.Dir, "dir", "d", mod.TemplateDir, "Output directory")
	pullCmd.Flags().BoolVarP(&opts.Force, "force", "f", false, "Force pull")
	// 新增超时参数
	pullCmd.Flags().DurationVar(&opts.Timeout, "timeout", 60*time.Second, "Download timeout")

	rootCmd.AddCommand(pullCmd)
}

func runPull(cmd *cobra.Command, args []string) error {
	imageName, tag, found := strings.Cut(args[0], ":")
	if !found || tag == "" {
		tag = "latest"
	}

	// 1. 快速反馈：本地是否有缓存
	// 这一步仅为了交互体验，不做实际逻辑判断
	exists, localVer, _ := mod.CheckLocalImage(opts.Dir, imageName)
	if exists {
		if !opts.Force && localVer != "unknown" && tag == "latest" {
			gologger.Info().Msgf("📂 Found local %s (v%s), checking for updates...", imageName, localVer)
		} else {
			gologger.Info().Msgf("📂 Found local %s (v%s)", imageName, localVer)
		}
	}

	// 2. 组装参数
	pullOpts := mod.PullOptions{
		RegistryURL: opts.Registry,
		BaseDir:     opts.Dir,
		ImageName:   imageName,
		Tag:         tag,
		Force:       opts.Force,
		Timeout:     opts.Timeout,
	}

	// 3. 执行 (传入 cmd.Context() 以响应 Ctrl+C)
	// 如果用户按 Ctrl+C，context 会取消，mod 包内的 http 请求会立即终止
	startTime := time.Now()
	err := mod.PullImageWithContext(cmd.Context(), pullOpts)
	if err != nil {
		// 如果是取消错误，友好的提示
		if strings.Contains(err.Error(), "context canceled") {
			gologger.Warning().Msg("❌ Operation canceled by user.")
			return nil
		}
		return err
	}

	// 4. 成功总结
	duration := time.Since(startTime).Round(time.Millisecond)
	if exists {
		gologger.Info().Msgf("✨ Updated %s in %s", imageName, duration)
	} else {
		gologger.Info().Msgf("✨ Installed %s in %s", imageName, duration)
	}

	return nil
}
