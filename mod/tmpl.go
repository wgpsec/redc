package mod

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"red-cloud/mod/gologger"
	"red-cloud/utils" // 保持原有引用
	"text/tabwriter"
)

const TemplateDir = "redc-templates"
const TmplCaseFile = "case.json"

type RedcTmpl struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	User        string `json:"user"`
	path        string
}

func ShowRedcTmpl() {
	l, err := ListRedcTmpl(TemplateDir)
	if err != nil {
		gologger.Error().Msgf("获取模版列表失败: %s", err)
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
	// 打印表头
	fmt.Fprintln(w, "NAME\tPATH\tUSER\tDESCRIPTION")

	for _, r := range l {
		// 格式化写入
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", r.Name, r.path, r.User, r.Description)
	}
	// 刷新缓冲区，将内容输出到终端
	w.Flush()
}

// ListRedcTmpl 获取所有镜像信息
func ListRedcTmpl(path string) ([]*RedcTmpl, error) {
	// 检查模板目录是否存在
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("模版目录：「%s」不存在", path)
	}
	_, dirs := utils.GetFilesAndDirs(path)
	var images []*RedcTmpl
	for _, dir := range dirs {
		im, err := getImageInfoByFile(dir)
		if err != nil {
			gologger.Error().Msgf("无法获取「%s」模版信息: %s", dir, err)
			continue
		}
		im.path = filepath.Base(dir)
		images = append(images, im)
	}
	return images, nil
}

// DeleteRedcTmpl 根据镜像名称删除对应的目录
func DeleteRedcTmpl(imageName string) error {
	if imageName == "" {
		return fmt.Errorf("镜像名称不能为空")
	}

	// 假设目录名就是镜像名
	targetPath := filepath.Join(TemplateDir, imageName)

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
	configPath := filepath.Join(path, TmplCaseFile)
	image := &RedcTmpl{
		path: path,
	}
	file, err := os.Open(configPath)
	if err != nil {
		return image, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	err = decoder.Decode(image)
	if err != nil {
		return nil, fmt.Errorf("JSON解码失败: %w", err)
	}

	// 如果 JSON 中没有 Name，可以使用目录名作为默认值（可选逻辑）
	if image.Name == "" {
		image.Name = filepath.Base(path)
	}

	return image, nil
}
