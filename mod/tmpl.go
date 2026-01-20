package mod

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"red-cloud/mod/gologger"
	"red-cloud/utils" // 保持原有引用
	"strings"
	"text/tabwriter"
	"time"

	"github.com/schollz/progressbar/v3"
)

const TemplateDir = "redc-templates"
const TmplCaseFile = "case.json"

type RedcTmpl struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	User        string `json:"user"`
	path        string
}

// PullOptions 封装参数，方便扩展
type PullOptions struct {
	RegistryURL string
	BaseDir     string
	ImageName   string
	Tag         string
	Force       bool
	Timeout     time.Duration
}

// 内部结构定义
type remoteIndex struct {
	Templates map[string]struct {
		Latest   string              `json:"latest"`
		Versions map[string]artifact `json:"versions"`
	} `json:"templates"`
}

type artifact struct {
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
}

type localMeta struct {
	Version string `json:"version"`
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

// CheckLocalImage 检查本地镜像
func CheckLocalImage(baseDir, imageName string) (bool, string, error) {
	targetDir := filepath.Join(baseDir, imageName)
	metaPath := filepath.Join(targetDir, TmplCaseFile)

	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		return false, "", nil
	}

	data, err := os.ReadFile(metaPath)
	if err != nil {
		return true, "unknown", nil // 存在目录但无法读取版本
	}

	var local localMeta
	if err := json.Unmarshal(data, &local); err != nil {
		return true, "unknown", nil
	}

	return true, local.Version, nil
}

// PullImageWithContext 支持取消和超时的拉取操作
func PullImageWithContext(ctx context.Context, opts PullOptions) error {
	// 1. 本地状态检查
	exists, localVer, _ := CheckLocalImage(opts.BaseDir, opts.ImageName)

	gologger.Info().Msgf("🔍 Connecting to registry %s...", opts.RegistryURL)

	// 2. 获取远程索引 (带 Context)
	var idx remoteIndex
	indexURL := fmt.Sprintf("%s/index.json?t=%d", opts.RegistryURL, time.Now().Unix())
	if err := fetchJSON(ctx, indexURL, &idx); err != nil {
		return fmt.Errorf("fetch index failed: %w", err)
	}

	// 3. 解析元数据
	tmpl, ok := idx.Templates[opts.ImageName]
	if !ok {
		return fmt.Errorf("template '%s' not found", opts.ImageName)
	}

	targetTag := opts.Tag
	if targetTag == "latest" || targetTag == "" {
		if tmpl.Latest == "" {
			return fmt.Errorf("remote latest version is missing")
		}
		targetTag = tmpl.Latest
	}

	art, ok := tmpl.Versions[targetTag]
	if !ok {
		return fmt.Errorf("version '%s' not found", targetTag)
	}

	// 4. 决策逻辑
	if exists && !opts.Force {
		if localVer == targetTag {
			gologger.Info().Msgf("✅ %s:%s is already up to date.", opts.ImageName, targetTag)
			return nil
		}
		gologger.Info().Msgf("🔄 Updating %s (v%s -> v%s)...", opts.ImageName, localVer, targetTag)
	} else if exists {
		gologger.Info().Msgf("⚠️  Force pulling %s:%s...", opts.ImageName, targetTag)
	}

	// 5. 执行原子安装
	targetDir := filepath.Join(opts.BaseDir, opts.ImageName)
	return downloadAndInstall(ctx, art, targetDir)
}

// --- Helper Functions ---

func fetchJSON(ctx context.Context, url string, v interface{}) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http status %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(v)
}

// downloadAndInstall 下载并原子解压
// 修复点：在目标目录的同级创建临时目录，确保 os.Rename 100% 成功
func downloadAndInstall(ctx context.Context, art artifact, finalDest string) error {
	// 1. 创建下载用的临时文件 (Zip 包放在系统临时目录没问题，因为只读不移)
	tmpZip, err := os.CreateTemp("", "redc-dl-*.zip")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() {
		tmpZip.Close()
		os.Remove(tmpZip.Name()) // 下载完成后清理 Zip 包
	}()

	// --- 下载阶段 ---
	req, err := http.NewRequestWithContext(ctx, "GET", art.URL, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("download request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	// 进度条
	bar := progressbar.DefaultBytes(
		resp.ContentLength,
		"⬇️  Downloading",
	)

	// 计算 Hash + 写入文件 + 进度条
	hasher := sha256.New()
	writer := io.MultiWriter(tmpZip, hasher, bar)

	if _, err := io.Copy(writer, resp.Body); err != nil {
		return fmt.Errorf("download write failed: %w", err)
	}

	// 必须显式关闭文件，否则后续 unzip 读取会报错或不完整
	if err := tmpZip.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// 校验 Hash
	actualHash := hex.EncodeToString(hasher.Sum(nil))
	if !strings.EqualFold(actualHash, art.SHA256) {
		return fmt.Errorf("checksum mismatch!\nLocal: %s\nRemote: %s", actualHash, art.SHA256)
	}

	gologger.Info().Msg("📦 Extracting...")

	// 确保目标路径的父目录存在
	// 例如 finalDest = "redc-templates/aliyun/ecs"
	// 必须先创建 "redc-templates/aliyun"
	parentDir := filepath.Dir(finalDest)
	if err := os.MkdirAll(parentDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	// 在 parentDir 下创建临时解压目录
	// 作用：确保临时目录和最终目录在同一个磁盘分区，防止 os.Rename 报 "invalid cross-device link"
	tmpExtractDir, err := os.MkdirTemp(parentDir, ".tmp-install-*")
	if err != nil {
		return fmt.Errorf("failed to create temp install dir: %w", err)
	}
	// 无论成功失败，最后都尝试清理临时目录（成功Rename后它就没了，失败了则清理垃圾）
	defer os.RemoveAll(tmpExtractDir)

	// 解压到这个同级临时目录
	if err := unzip(tmpZip.Name(), tmpExtractDir); err != nil {
		return fmt.Errorf("unzip failed: %w", err)
	}

	// 1. 先移除旧版本目录 (如果存在)
	if err := os.RemoveAll(finalDest); err != nil {
		return fmt.Errorf("failed to remove old version: %w", err)
	}

	// 2. 将临时目录重命名为正式目录
	// 因为它们在同一个父目录下，这步操作是原子的，且极快
	if err := os.Rename(tmpExtractDir, finalDest); err != nil {
		return fmt.Errorf("failed to finalize installation (rename): %w", err)
	}

	return nil
}

// unzip 具体的解压逻辑
func unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	destClean := filepath.Clean(dest) + string(os.PathSeparator)

	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)
		if !strings.HasPrefix(filepath.Clean(fpath)+string(os.PathSeparator), destClean) {
			return fmt.Errorf("zip slip detected: %s", fpath)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		// 限制单个文件大小，防止解压炸弹（可选）
		io.Copy(outFile, rc)

		outFile.Close()
		rc.Close()
	}
	return nil
}
