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
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"red-cloud/mod/gologger"
	"red-cloud/utils"

	"github.com/schollz/progressbar/v3"
)

// TemplateDir 全局配置：默认模版存放路径
var TemplateDir = "redc-templates"

const TmplCaseFile = "case.json"

// RedcTmpl 对应本地 case.json 的结构
type RedcTmpl struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	User        string `json:"user"`
	Version     string `json:"version"`
	RedcModule  string `json:"redc_module"`
	Path        string `json:"-"`
}

// PullOptions 配置项
type PullOptions struct {
	RegistryURL string
	Force       bool
	Timeout     time.Duration
}

// =============================================================================
//  远程索引数据结构 (JSON Mapping)
// =============================================================================

// RemoteIndex 对应 index.json
type RemoteIndex struct {
	UpdatedAt string                  `json:"updated_at"`
	RepoName  string                  `json:"repo_name"`
	Templates map[string]TemplateItem `json:"templates"`
}

// TemplateItem 对应 templates 下的具体项
type TemplateItem struct {
	ID       string                     `json:"id"`       // e.g. "aliyun/ecs"
	Provider string                     `json:"provider"` // e.g. "aliyun"
	Slug     string                     `json:"slug"`     // e.g. "ecs"
	Latest   string                     `json:"latest"`   // e.g. "1.0.1"
	Versions map[string]TemplateVersion `json:"versions"`
	Metadata TemplateMetadata           `json:"metadata"`
}

// TemplateMetadata 元数据信息
type TemplateMetadata struct {
	Name        string `json:"name"`
	Author      string `json:"author"`
	Description string `json:"description"`
	Readme      string `json:"readme"`
}

// TemplateVersion 具体版本信息
type TemplateVersion struct {
	URL       string `json:"url"`
	SHA256    string `json:"sha256"`
	UpdatedAt string `json:"updated_at"`
}

// SearchResult 搜索结果结构
type SearchResult struct {
	Key         string
	Version     string
	Description string
	Author      string
	Provider    string
	Score       int
}

// =============================================================================
//  网络层：索引获取
// =============================================================================

// GetRemoteIndex 获取并解析远程索引 (独立函数，便于复用)
func GetRemoteIndex(ctx context.Context, registryURL string) (*RemoteIndex, error) {
	var idx RemoteIndex
	// 添加时间戳防止 CDN 缓存
	indexURL := fmt.Sprintf("%s/index.json?t=%d", registryURL, time.Now().Unix())
	if err := fetchJSON(ctx, indexURL, &idx); err != nil {
		return nil, fmt.Errorf("fetch index failed: %w", err)
	}
	return &idx, nil
}

// =============================================================================
//  逻辑层：智能搜索算法
// =============================================================================

// SearchFromIndex 在内存中的索引进行搜索 (纯 CPU 计算)
// 支持多关键词、权重打分、长度惩罚排序
func SearchFromIndex(idx *RemoteIndex, query string) []SearchResult {
	var results []SearchResult
	query = strings.ToLower(strings.TrimSpace(query))
	tokens := strings.Fields(query) // 分词

	for key, tmpl := range idx.Templates {
		// 预处理字段 (全部归一化为小写)
		fields := struct {
			Key, Provider, Slug, Name, Author, Desc string
		}{
			Key:      strings.ToLower(key),
			Provider: strings.ToLower(tmpl.Provider),
			Slug:     strings.ToLower(tmpl.Slug),
			Name:     strings.ToLower(tmpl.Metadata.Name),
			Author:   strings.ToLower(tmpl.Metadata.Author),
			Desc:     strings.ToLower(tmpl.Metadata.Description),
		}

		score := 0
		allTokensMatched := true

		// 核心评分逻辑
		if len(tokens) > 0 {
			for _, token := range tokens {
				tokenScore := 0

				// 规则 A: 完整 Key 精确匹配 (最高权重)
				if fields.Key == token {
					tokenScore += 1000
				}
				// 规则 B: Slug/Name 精确匹配 (次高权重, e.g. 搜 "ecs" 命中 "aliyun/ecs")
				if fields.Slug == token || fields.Name == token {
					tokenScore += 500
				}
				// 规则 C: Provider 精确匹配
				if fields.Provider == token {
					tokenScore += 200
				}
				// 规则 D: 字段包含匹配
				if strings.Contains(fields.Key, token) {
					tokenScore += 50
				} else if strings.Contains(fields.Author, token) {
					tokenScore += 30
				} else if strings.Contains(fields.Desc, token) {
					tokenScore += 10
				}

				if tokenScore == 0 {
					allTokensMatched = false
					break
				}
				score += tokenScore
			}
		} else {
			// 无关键词列出所有，默认低分
			score = 1
		}

		if allTokensMatched {
			results = append(results, SearchResult{
				Key:         key,
				Version:     tmpl.Latest,
				Description: tmpl.Metadata.Description,
				Author:      tmpl.Metadata.Author,
				Provider:    tmpl.Provider,
				Score:       score,
			})
		}
	}

	// 结果排序：分数高 > 名字短 > 字母序
	sort.Slice(results, func(i, j int) bool {
		// 优先级 1: 分数
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		// 优先级 2: 长度 (越短越基础，越靠前)
		if len(results[i].Key) != len(results[j].Key) {
			return len(results[i].Key) < len(results[j].Key)
		}
		// 优先级 3: 字母序
		return results[i].Key < results[j].Key
	})

	return results
}

// Search 对外暴露的完整搜索接口 (网络 + 计算)
func Search(ctx context.Context, query string, opts PullOptions) ([]SearchResult, error) {
	// 1. 获取远程索引
	idx, err := GetRemoteIndex(ctx, opts.RegistryURL)
	if err != nil {
		return nil, err
	}
	// 2. 内存搜索
	return SearchFromIndex(idx, query), nil
}

// =============================================================================
//  业务层：Pull 流程
// =============================================================================

// Pull 执行拉取流程
func Pull(ctx context.Context, imageRef string, opts PullOptions) error {
	startTime := time.Now()

	// 1. 解析参数 (name:tag)
	imageName, tag, found := strings.Cut(imageRef, ":")
	if !found || tag == "" {
		tag = "latest"
	}

	// 2. 检查本地
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
	idx, err := GetRemoteIndex(ctx, opts.RegistryURL)
	if err != nil {
		return false, err
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

	verData, ok := tmpl.Versions[targetTag]
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

	// 5. 下载并原子安装
	targetDir, err := resolveSafePath(imageName)
	if err != nil {
		return false, fmt.Errorf("invalid install path: %w", err)
	}

	if err := downloadAndInstall(ctx, verData, targetDir); err != nil {
		return false, err
	}

	return true, nil
}

// =============================================================================
//  本地管理功能
// =============================================================================

// GetTemplatePath 根据镜像名称查找并返回本地路径
func GetTemplatePath(imageName string) (string, error) {
	path, err := resolveSafePath(imageName)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return "", fmt.Errorf("template '%s' not found", imageName)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("path '%s' exists but is not a directory", path)
	}
	configPath := filepath.Join(path, TmplCaseFile)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return "", fmt.Errorf("template broken: missing %s in %s", TmplCaseFile, imageName)
	}
	return path, nil
}

// CheckLocalImage 检查本地是否存在指定模版
func CheckLocalImage(imageName string) (bool, string, error) {
	path, err := GetTemplatePath(imageName)
	if err != nil {
		return false, "", nil
	}
	meta, err := readTemplateMeta(path)
	if err != nil || meta.Version == "" {
		return true, "unknown", nil
	}
	return true, meta.Version, nil
}

// RemoveTemplate 删除指定模版
func RemoveTemplate(imageName string) error {
	targetPath, err := resolveSafePath(imageName)
	if err != nil {
		return err
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

// CopyTemplate copies a local template to a new template name
func CopyTemplate(sourceName string, targetName string) error {
	if strings.TrimSpace(sourceName) == "" || strings.TrimSpace(targetName) == "" {
		return fmt.Errorf("template name cannot be empty")
	}
	if sourceName == targetName {
		return fmt.Errorf("target template name must be different")
	}
	sourcePath, err := GetTemplatePath(sourceName)
	if err != nil {
		return err
	}
	targetPath, err := resolveSafePath(targetName)
	if err != nil {
		return err
	}
	if _, err := os.Stat(targetPath); err == nil {
		return fmt.Errorf("template '%s' already exists", targetName)
	}
	if err := utils.Dir(sourcePath, targetPath); err != nil {
		return fmt.Errorf("copy template failed: %w", err)
	}
	return nil
}

// ShowLocalTemplates 打印表格形式的列表
func ShowLocalTemplates() {
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
	fmt.Fprintln(w, "NAME\tVERSION\tUSER\tMODULE\tDESCRIPTION")
	for _, tmpl := range list {
		desc := tmpl.Description
		if len(desc) > 100 {
			desc = desc[:100] + "..."
		}
		ver := tmpl.Version
		if ver == "" {
			ver = "unknown"
		}
		module := tmpl.RedcModule
		if module == "" {
			module = "-"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", tmpl.Name, ver, tmpl.User, module, desc)
	}
	w.Flush()
}

// ListLocalTemplates 返回结构化数据
func ListLocalTemplates() ([]*RedcTmpl, error) {
	if _, err := os.Stat(TemplateDir); os.IsNotExist(err) {
		return nil, nil
	}
	// 假设最大深度为 3，根据需要调整
	dirs, err := ScanTemplateDirs(TemplateDir, 3)
	if err != nil {
		return nil, err
	}
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

// =============================================================================
//  通用辅助函数 / Utils
// =============================================================================

// resolveSafePath 核心路径处理函数
func resolveSafePath(imageName string) (string, error) {
	if imageName == "" {
		return "", fmt.Errorf("image name cannot be empty")
	}
	localImageName := filepath.FromSlash(imageName)
	targetPath := filepath.Join(TemplateDir, localImageName)
	absBase, err := filepath.Abs(TemplateDir)
	if err != nil {
		return "", fmt.Errorf("resolve base path failed: %w", err)
	}
	absTarget, err := filepath.Abs(targetPath)
	if err != nil {
		return "", fmt.Errorf("resolve target path failed: %w", err)
	}
	if !strings.HasPrefix(absTarget, absBase+string(os.PathSeparator)) && absTarget != absBase {
		return "", fmt.Errorf("security violation: invalid path traversal detected in '%s'", imageName)
	}
	return targetPath, nil
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
	relPath, relErr := filepath.Rel(TemplateDir, dirPath)
	if relErr != nil {
		relPath = filepath.Base(dirPath)
	}
	finalName := filepath.ToSlash(relPath)
	tmpl.Name = finalName
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

// downloadAndInstall 下载并解压 (适配新的 TemplateVersion 结构)
func downloadAndInstall(ctx context.Context, verData TemplateVersion, finalDest string) error {
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
	req, err := http.NewRequestWithContext(ctx, "GET", verData.URL, nil)
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
	tmpZip.Close()

	// 4. 校验 Hash
	actualHash := hex.EncodeToString(hasher.Sum(nil))
	if !strings.EqualFold(actualHash, verData.SHA256) {
		return fmt.Errorf("checksum mismatch!\nLocal: %s\nRemote: %s", actualHash, verData.SHA256)
	}

	gologger.Info().Msg("📦 Extracting...")

	// 5. 准备解压目录结构
	parentDir := filepath.Dir(finalDest)
	if err := os.MkdirAll(parentDir, os.ModePerm); err != nil {
		return fmt.Errorf("mkdir parent failed: %w", err)
	}
	tmpExtractDir, err := os.MkdirTemp(parentDir, ".tmp-install-*")
	if err != nil {
		return fmt.Errorf("mkdir temp failed: %w", err)
	}
	defer os.RemoveAll(tmpExtractDir)

	// 解压到临时目录
	if err := unzip(tmpZip.Name(), tmpExtractDir); err != nil {
		return fmt.Errorf("unzip failed: %w", err)
	}

	// 6. 原子替换
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
		io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()
	}
	return nil
}

// ScanTemplateDirs 扫描指定目录寻找模版
func ScanTemplateDirs(rootDir string, maxDepth int) ([]string, error) {
	var validPaths []string
	hasConfigFile := func(dirPath string) bool {
		configPath := filepath.Join(dirPath, TmplCaseFile)
		_, err := os.Stat(configPath)
		return err == nil
	}
	var scan func(currentPath string, currentDepth int)
	scan = func(currentPath string, currentDepth int) {
		if currentDepth > maxDepth {
			return
		}
		entries, err := os.ReadDir(currentPath)
		if err != nil {
			return
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			fullPath := filepath.Join(currentPath, entry.Name())
			if hasConfigFile(fullPath) {
				validPaths = append(validPaths, fullPath)
				continue
			}
			scan(fullPath, currentDepth+1)
		}
	}
	scan(rootDir, 1)
	return validPaths, nil
}
