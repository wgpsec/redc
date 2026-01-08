package mod

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"red-cloud/utils" // 保持原有引用
	"text/tabwriter"
)

const templateDir = "redc-templates"

type RedcTmpl struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	User        string `json:"user"`
}

// ListRedcTmpl 列出所有镜像并格式化输出表格
func ListRedcTmpl() {
	// 检查模板目录是否存在
	if _, err := os.Stat(templateDir); os.IsNotExist(err) {
		fmt.Printf("模板目录不存在: %s\n", templateDir)
		return
	}

	_, dirs := utils.GetFilesAndDirs(templateDir)
	if len(dirs) == 0 {
		fmt.Println("暂无镜像数据")
		return
	}

	// 使用 tabwriter 进行格式化对齐输出
	// minwidth=0, tabwidth=8, padding=2, padchar=' ', flags=0
	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)

	// 打印表头
	fmt.Fprintln(w, "NAME\tUSER\tDESCRIPTION")

	for _, v := range dirs {
		// 假设 utils 返回的是完整路径或相对路径
		// 如果 utils 返回的只是目录名，需要 path = filepath.Join(templateDir, v)
		// 这里保留原有逻辑，假设 v 是可访问的路径
		r, err := getImageInfoByFile(v)
		if err != nil {
			// 可以选择打印错误日志，这里选择跳过无效的配置
			// fmt.Printf("跳过无效镜像 [%s]: %v\n", v, err)
			continue
		}
		// 格式化写入
		fmt.Fprintf(w, "%s\t%s\t%s\n", r.Name, r.User, r.Description)
	}

	// 刷新缓冲区，将内容输出到终端
	w.Flush()
}

// DeleteRedcTmpl 根据镜像名称删除对应的目录
func DeleteRedcTmpl(imageName string) error {
	if imageName == "" {
		return fmt.Errorf("镜像名称不能为空")
	}

	// 假设目录名就是镜像名
	targetPath := filepath.Join(templateDir, imageName)

	// 检查是否存在
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		return fmt.Errorf("镜像 '%s' 不存在", imageName)
	}

	// 删除目录及其包含的所有文件
	err := os.RemoveAll(targetPath)
	if err != nil {
		return fmt.Errorf("删除失败: %v", err)
	}

	fmt.Printf("镜像 '%s' 已成功删除\n", imageName)
	return nil
}

// getImageInfoByFile 读取并解析 case.json
func getImageInfoByFile(path string) (*RedcTmpl, error) {
	configPath := filepath.Join(path, "case.json")

	file, err := os.Open(configPath)
	if err != nil {
		return nil, err // 简化错误返回，上层决定是否打印
	}
	defer file.Close()

	var image RedcTmpl
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&image)
	if err != nil {
		return nil, fmt.Errorf("JSON解码失败: %w", err)
	}

	// 如果 JSON 中没有 Name，可以使用目录名作为默认值（可选逻辑）
	if image.Name == "" {
		image.Name = filepath.Base(path)
	}

	return &image, nil
}
