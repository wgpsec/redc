package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"red-cloud/mod/gologger"

	"github.com/spf13/cobra"
)

var follow bool

var logsCmd = &cobra.Command{
	Use:   "logs [service...]",
	Short: "查看服务运行日志",
	Run: func(cmd *cobra.Command, args []string) {
		logMgr := gologger.NewLogManager(redcProject.ProjectPath)

		var targets []string
		if len(args) > 0 {
			targets = args
		} else {
			// 扫描目录
			files, _ := os.ReadDir(logMgr.BaseDir)
			for _, f := range files {
				if strings.HasSuffix(f.Name(), ".log") {
					targets = append(targets, strings.TrimSuffix(f.Name(), ".log"))
				}
			}
		}

		var wg sync.WaitGroup
		for _, t := range targets {
			wg.Add(1)
			go func(name string) {
				defer wg.Done()
				path := logMgr.GetLogPath(name)
				readLog(name, path, follow)
			}(t)
		}
		wg.Wait()
	},
}

func readLog(name, path string, follow bool) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	r := bufio.NewReader(f)
	for {
		line, err := r.ReadString('\n')
		if err == nil {
			fmt.Printf("[%s] | %s", name, line)
		} else {
			if err == io.EOF {
				if follow {
					time.Sleep(500 * time.Millisecond)
					continue
				}
				break
			}
		}
	}
}

func init() {
	logsCmd.Flags().BoolVarP(&follow, "follow", "f", false, "实时跟踪日志")
	composeCmd.AddCommand(logsCmd)
}
