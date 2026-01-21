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
	"red-cloud/utils" // 需确保此包存在或自行实现 GetFilesAndDirs
	"strings"
	"text/tabwriter"
	"time"

	"github.com/schollz/progressbar/v3"
)

// TemplateDir 全局配置：默认模版存放路径
// 这是一个导出变量，CLI 可以通过 flag (如 -d) 直接修改这个变量
var TemplateDir = "redc-templates"

const TmplCaseFile = "case.json"

// RedcTmpl 对应 case.json 的结构
type RedcTmpl struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	User        string `json:"user"`
	Version     string `json:"version"`
	Path        string `json:"-"`
}

// PullOptions 配置项 (移除了 BaseDir，因为使用了全局 TemplateDir)
type PullOptions struct {
	RegistryURL string
	Force       bool
	Timeout     time.Duration
}

// 内部使用的远程索引结构
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

// =============================================================================
//  核心功能：Pull (下载/更新)
// =============================================================================

// Pull 执行拉取流程
func Pull(ctx context.Context, imageRef string, opts PullOptions) error {
	startTime := time.Now()

	// 1. 解析参数 (name:tag)
	imageName, tag, found := strings.Cut(imageRef, ":")
	if !found || tag == "" {
		tag = "latest"
	}

	// 2. 检查本地 (使用全局 TemplateDir)
	exists, localVer, _ := CheckLocalImage(imageName)
	if exists {
		if !opts.Force && localVer != "unknown" && tag == "latest" {
			gologger.Info().Msgf("📂 Found local %s (v%s), checking for updates...", imageName, localVer)
		} else {
			gologger.Info().Msgf("📂 Found local %s (v%s)", imageName, localVer)
		}
	}

	// 3. 设置超时
	var cancel context.CancelFunc
	if opts.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	// 4. 执行核心下载逻辑
	downloaded, err := pullCore(ctx, imageName, tag, localVer, exists, opts)
	if err != nil {
		return err
	}

	// 5. 结果反馈
	duration := time.Since(startTime).Round(time.Millisecond)
	if downloaded {
		if exists {
			gologger.Info().Msgf("✨ Updated %s in %s", imageName, duration)
		} else {
			gologger.Info().Msgf("✨ Installed %s in %s", imageName, duration)
		}
	}
	return nil
}

// pullCore 处理网络请求和决策
func pullCore(ctx context.Context, imageName, tag, localVer string, exists bool, opts PullOptions) (bool, error) {
	gologger.Info().Msgf("🔍 Connecting to registry %s...", opts.RegistryURL)

	// 1. 获取远程索引
	var idx remoteIndex
	indexURL := fmt.Sprintf("%s/index.json?t=%d", opts.RegistryURL, time.Now().Unix())
	if err := fetchJSON(ctx, indexURL, &idx); err != nil {
		return false, fmt.Errorf("fetch index failed: %w", err)
	}

	// 2. 查找模版
	tmpl, ok := idx.Templates[imageName]
	if !ok {
		return false, fmt.Errorf("template '%s' not found in registry", imageName)
	}

	// 3. 解析版本
	targetTag := tag
	if targetTag == "latest" || targetTag == "" {
		if tmpl.Latest == "" {
			return false, fmt.Errorf("remote latest version is missing")
		}
		targetTag = tmpl.Latest
	}

	art, ok := tmpl.Versions[targetTag]
	if !ok {
		return false, fmt.Errorf("version '%s' not found", targetTag)
	}

	// 4. 决策
	if exists && !opts.Force {
		if localVer == targetTag {
			gologger.Info().Msgf("✅ %s:%s is already up to date.", imageName, targetTag)
			return false, nil
		}
		gologger.Info().Msgf("🔄 Updating %s (v%s -> v%s)...", imageName, localVer, targetTag)
	} else if exists {
		gologger.Info().Msgf("⚠️  Force pulling %s:%s...", imageName, targetTag)
	}

	// 5. 下载并原子安装 (拼接全局 TemplateDir)
	targetDir := filepath.Join(TemplateDir, imageName)
	if err := downloadAndInstall(ctx, art, targetDir); err != nil {
		return false, err
	}

	return true, nil
}

// =============================================================================
//  本地管理功能：List (列表) & Remove (删除)
// =============================================================================

// ShowLocalTemplates 打印表格形式的列表
func ShowLocalTemplates() {
	// 使用全局 TemplateDir
	list, err := ListLocalTemplates()
	if err != nil {
		gologger.Error().Msgf("Failed to list templates: %v", err)
		return
	}

	if len(list) == 0 {
		gologger.Info().Msgf("No templates found in directory: %s", TemplateDir)
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 8, 4, ' ', 0)
	fmt.Fprintln(w, "NAME\tVERSION\tUSER\tDESCRIPTION")

	for _, tmpl := range list {
		desc := tmpl.Description
		if len(desc) > 50 {
			desc = desc[:47] + "..."
		}
		ver := tmpl.Version
		if ver == "" {
			ver = "unknown"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", tmpl.Name, ver, tmpl.User, desc)
	}
	w.Flush()
}

// ListLocalTemplates 返回结构化数据
func ListLocalTemplates() ([]*RedcTmpl, error) {
	// 使用全局 TemplateDir
	if _, err := os.Stat(TemplateDir); os.IsNotExist(err) {
		return nil, nil
	}

	_, dirs := utils.GetFilesAndDirs(TemplateDir)
	var templates []*RedcTmpl

	for _, dirPath := range dirs {
		t, err := readTemplateMeta(dirPath)
		if err != nil {
			t = &RedcTmpl{Name: filepath.Base(dirPath), Description: "[Error reading metadata]"}
		}
		t.Path = dirPath
		templates = append(templates, t)
	}
	return templates, nil
}

// RemoveTemplate 删除指定模版
func RemoveTemplate(imageName string) error {
	if imageName == "" {
		return fmt.Errorf("image name cannot be empty")
	}

	// 使用全局 TemplateDir 拼接
	targetPath := filepath.Join(TemplateDir, imageName)

	// 安全检查：防止路径穿越 (../../)
	cleanBase := filepath.Clean(TemplateDir)
	cleanTarget := filepath.Clean(targetPath)
	if !strings.HasPrefix(cleanTarget, cleanBase) {
		return fmt.Errorf("invalid path: %s", imageName)
	}

	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		return fmt.Errorf("template '%s' not found", imageName)
	}

	gologger.Info().Msgf("🗑️  Removing template: %s", imageName)
	if err := os.RemoveAll(targetPath); err != nil {
		return fmt.Errorf("failed to remove: %w", err)
	}

	gologger.Info().Msg("✅ Successfully removed.")
	return nil
}

// =============================================================================
//  辅助函数 / Utils
// =============================================================================

// CheckLocalImage 检查本地 (使用全局 TemplateDir)
func CheckLocalImage(imageName string) (bool, string, error) {
	targetDir := filepath.Join(TemplateDir, imageName)

	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		return false, "", nil
	}

	meta, err := readTemplateMeta(targetDir)
	if err != nil || meta.Version == "" {
		return true, "unknown", nil
	}
	return true, meta.Version, nil
}

// readTemplateMeta 读取 case.json
func readTemplateMeta(dirPath string) (*RedcTmpl, error) {
	configPath := filepath.Join(dirPath, TmplCaseFile)
	tmpl := &RedcTmpl{}
	data, err := os.ReadFile(configPath)
	if err != nil {
		return tmpl, err
	}
	if err := json.Unmarshal(data, tmpl); err != nil {
		return nil, err
	}
	// 如果 Name 为空，用目录名兜底
	if tmpl.Name == "" {
		tmpl.Name = filepath.Base(dirPath)
	}
	return tmpl, nil
}

// fetchJSON 通用 GET 请求
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

// downloadAndInstall 下载并解压 (原子操作)
func downloadAndInstall(ctx context.Context, art artifact, finalDest string) error {
	// 1. 创建临时 ZIP 文件
	tmpZip, err := os.CreateTemp("", "redc-dl-*.zip")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() {
		tmpZip.Close()
		os.Remove(tmpZip.Name())
	}()

	// 2. 下载
	req, err := http.NewRequestWithContext(ctx, "GET", art.URL, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http status: %d", resp.StatusCode)
	}

	// 3. 进度条 + Hash
	bar := progressbar.DefaultBytes(resp.ContentLength, "⬇️  Downloading")
	hasher := sha256.New()
	writer := io.MultiWriter(tmpZip, hasher, bar)

	if _, err := io.Copy(writer, resp.Body); err != nil {
		return fmt.Errorf("write failed: %w", err)
	}
	tmpZip.Close() // 必须显式关闭才能被 zip reader 读取

	// 4. 校验 Hash
	actualHash := hex.EncodeToString(hasher.Sum(nil))
	if !strings.EqualFold(actualHash, art.SHA256) {
		return fmt.Errorf("checksum mismatch!\nLocal: %s\nRemote: %s", actualHash, art.SHA256)
	}

	gologger.Info().Msg("📦 Extracting...")

	// 5. 准备解压目录结构
	parentDir := filepath.Dir(finalDest)
	if err := os.MkdirAll(parentDir, os.ModePerm); err != nil {
		return fmt.Errorf("mkdir parent failed: %w", err)
	}

	// 创建一个同级的临时目录用于解压，确保 rename 是原子操作
	tmpExtractDir, err := os.MkdirTemp(parentDir, ".tmp-install-*")
	if err != nil {
		return fmt.Errorf("mkdir temp failed: %w", err)
	}
	// 无论成功与否，最后都清理掉这个临时文件夹
	defer os.RemoveAll(tmpExtractDir)

	// 解压到临时目录
	if err := unzip(tmpZip.Name(), tmpExtractDir); err != nil {
		return fmt.Errorf("unzip failed: %w", err)
	}

	// 6. 原子替换：删除旧目录 -> 移动新目录
	if err := os.RemoveAll(finalDest); err != nil {
		return fmt.Errorf("remove old version failed: %w", err)
	}
	if err := os.Rename(tmpExtractDir, finalDest); err != nil {
		return fmt.Errorf("rename failed: %w", err)
	}

	return nil
}

// unzip 标准解压函数 + Zip Slip 防护
func unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	destClean := filepath.Clean(dest) + string(os.PathSeparator)

	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)

		// 安全检查: Zip Slip
		if !strings.HasPrefix(filepath.Clean(fpath)+string(os.PathSeparator), destClean) {
			return fmt.Errorf("zip slip detected: %s", f.Name)
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

		// 限制文件大小，可选，防止压缩包炸弹
		io.Copy(outFile, rc)

		outFile.Close()
		rc.Close()
	}
	return nil
}
