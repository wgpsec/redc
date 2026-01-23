package mod

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"red-cloud/mod/gologger"

	"gopkg.in/yaml.v3"
)

// moduleRegistry 注册所有可用的模块钩子
var moduleRegistry = map[string]func(*Case) error{
	"gen_clash_config": genClashConfig,
	"upload_r2":        uploadR2,
	"chang_dns":        changDNS,
}

// genClashConfig 根据模板 outputs 与 tfvars 生成 Clash 配置并上传 R2
func genClashConfig(c *Case) error {
	// 确保 output 已加载
	if c.output == nil {
		if _, err := c.TfOutput(); err != nil {
			return fmt.Errorf("获取场景 output 失败: %w", err)
		}
	}

	ips, err := collectIPs(c)
	if err != nil {
		return err
	}
	if len(ips) == 0 {
		return fmt.Errorf("未获取到任何节点 IP")
	}

	vars, err := loadTfVars(filepath.Join(c.Path, "terraform.tfvars"))
	if err != nil {
		return err
	}
	ssPort, okP := vars["port"]
	ssPass, okS := vars["password"]
	if !okP || !okS {
		return fmt.Errorf("terraform.tfvars 缺少 port 或 password")
	}
	fileName := vars["filename"]
	if fileName == "" {
		fileName = "default-config.yaml"
	}

	configContent, err := buildClashConfig(ips, ssPort, ssPass)
	if err != nil {
		return err
	}

	// 写入本地文件
	configPath := filepath.Join(c.Path, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		return fmt.Errorf("写入 config.yaml 失败: %w", err)
	}

	// 生成上传副本（文件名可配置）
	uploadName := filepath.Join(c.Path, fileName)
	if err := os.WriteFile(uploadName, []byte(configContent), 0644); err != nil {
		return fmt.Errorf("写入 %s 失败: %w", fileName, err)
	}

	gologger.Info().Msgf("Clash 配置生成完成，文件位置: %s", configPath)
	return nil
}

func collectIPs(c *Case) ([]string, error) {
	// 优先用 Terraform Output 中的 ecs_ip（list），兼容 public_ip（string/array）
	tryKeys := []string{"ecs_ip", "public_ip"}
	for _, key := range tryKeys {
		if meta, ok := c.output[key]; ok {
			var arr []string
			if err := json.Unmarshal(meta.Value, &arr); err == nil {
				return arr, nil
			}
			var single string
			if err := json.Unmarshal(meta.Value, &single); err == nil && single != "" {
				return []string{single}, nil
			}
		}
	}
	return nil, fmt.Errorf("未在 Terraform Output 中找到 ecs_ip/public_ip")
}

func loadTfVars(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("读取 terraform.tfvars 失败: %w", err)
	}
	defer f.Close()

	res := make(map[string]string)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.Trim(strings.TrimSpace(parts[1]), "\"')")
		val = strings.Trim(val, "\"")
		res[key] = val
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return res, nil
}

func buildClashConfig(ips []string, port string, password string) (string, error) {
	decode := func(s string) (string, error) {
		b, err := base64.StdEncoding.DecodeString(s)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}

	part1, err := decode("bWl4ZWQtcG9ydDogNjQyNzcKYWxsb3ctbGFuOiB0cnVlCmJpbmQtYWRkcmVzczogJyonCm1vZGU6IHJ1bGUKbG9nLWxldmVsOiBpbmZvCmlwdjY6IGZhbHNlCmV4dGVybmFsLWNvbnRyb2xsZXI6IDEyNy4wLjAuMTo5MDkwCnNlY3JldDogdHh0dHh0eHQyc3h0eHR4dGRkeHR4dDExMTExMTEKcm91dGluZy1tYXJrOiA2NjY2Cmhvc3RzOgoKcHJvZmlsZToKICBzdG9yZS1zZWxlY3RlZDogZmFsc2UKICBzdG9yZS1mYWtlLWlwOiB0cnVlCgpkbnM6CiAgZW5hYmxlOiBmYWxzZQogIGxpc3RlbjogMC4wLjAuMDo1MwogIGRlZmF1bHQtbmFtZXNlcnZlcjoKICAgIC0gMjIzLjUuNS41CiAgICAtIDExOS4yOS4yOS4yOQogIGVuaGFuY2VkLW1vZGU6IGZha2UtaXAgIyBvciByZWRpci1ob3N0IChub3QgcmVjb21tZW5kZWQpCiAgZmFrZS1pcC1yYW5nZTogMTk4LjE4LjAuMS8xNiAjIEZha2UgSVAgYWRkcmVzc2VzIHBvb2wgQ0lEUgogIG5hbWVzZXJ2ZXI6CiAgICAtIDIyMy41LjUuNSAjIGRlZmF1bHQgdmFsdWUKICAgIC0gMTE5LjI5LjI5LjI5ICMgZGVmYXVsdCB2YWx1ZQogICAgLSB0bHM6Ly9kbnMucnVieWZpc2guY246ODUzICMgRE5TIG92ZXIgVExTCiAgICAtIGh0dHBzOi8vMS4xLjEuMS9kbnMtcXVlcnkgIyBETlMgb3ZlciBIVFRQUwogICAgLSBkaGNwOi8vZW4wICMgZG5zIGZyb20gZGhjcAogICAgIyAtICc4LjguOC44I2VuMCcKCnByb3hpZXM6Cg==")
	if err != nil {
		return "", err
	}

	part3, err := decode("cHJveHktZ3JvdXBzOgogIC0gbmFtZTogInRlc3QiCiAgICB0eXBlOiBsb2FkLWJhbGFuY2UKICAgIHByb3hpZXM6Cg==")
	if err != nil {
		return "", err
	}

	part5, err := decode("ICAgIHVybDogJ2h0dHA6Ly93d3cuZ3N0YXRpYy5jb20vZ2VuZXJhdGVfMjA0JwogICAgaW50ZXJ2YWw6IDI0MDAKICAgIHN0cmF0ZWd5OiByb3VuZC1yb2JpbgoKcnVsZXM6CiAgLSBET01BSU4tU1VGRklYLGdvb2dsZS5jb20sdGVzdAogIC0gRE9NQUlOLUtFWVdPUkQsZ29vZ2xlLHRlc3QKICAtIERPTUFJTixnb29nbGUuY29tLHRlc3QKICAtIEdFT0lQLENOLHRlc3QKICAtIE1BVENILHRlc3QKICAtIFNSQy1JUC1DSURSLDE5Mi4xNjguMS4yMDEvMzIsRElSRUNUCiAgLSBJUC1DSURSLDEyNy4wLjAuMC84LERJUkVDVAogIC0gRE9NQUlOLVNVRkZJWCxhZC5jb20sUkVKRUNUCg==")
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	sb.WriteString(part1)

	// part2: proxies 列表
	for _, ip := range ips {
		sb.WriteString("  - name: \"")
		sb.WriteString(ip)
		sb.WriteString("\"\n    type: ss\n    server: ")
		sb.WriteString(ip)
		sb.WriteString("\n    port: ")
		sb.WriteString(port)
		sb.WriteString("\n    cipher: chacha20-ietf-poly1305\n    password: \"")
		sb.WriteString(password)
		sb.WriteString("\"\n\n")
	}

	// part3
	sb.WriteString(part3)

	// part4: proxy-groups 内容
	for _, ip := range ips {
		sb.WriteString("      - ")
		sb.WriteString(ip)
		sb.WriteString("\n")
	}

	// part5
	sb.WriteString(part5)

	return sb.String(), nil
}

func runRcloneUpload(workdir, filePath, bucketName, bucketPath string) error {
	// 允许 tfvars 配置桶与路径，未配置时沿用旧默认
	if bucketName == "" {
		bucketName = "test"
	}
	trimmed := strings.Trim(bucketPath, "/")
	if trimmed != "" {
		trimmed += "/"
	}
	remoteDir := fmt.Sprintf("r2:%s/%s", bucketName, trimmed)

	// 先删除，再上传
	delCmd := exec.Command("bash", "-c", fmt.Sprintf("cd %s && rclone deletefile %s%s", workdir, remoteDir, filepath.Base(filePath)))
	if err := delCmd.Run(); err != nil {
		gologger.Debug().Msgf("rclone deletefile 失败: %v", err)
	}

	upCmd := exec.Command("bash", "-c", fmt.Sprintf("cd %s && rclone copy %s %s", workdir, filepath.Base(filePath), remoteDir))
	if out, err := upCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("rclone copy 失败: %v, output: %s", err, string(out))
	}

	gologger.Info().Msgf("R2 上传完成: %s%s", remoteDir, filepath.Base(filePath))
	return nil
}

// uploadR2 仅负责将已生成的配置文件上传到 R2
func uploadR2(c *Case) error {
	vars, _ := loadTfVars(filepath.Join(c.Path, "terraform.tfvars"))
	fileName := vars["filename"]
	if fileName == "" {
		fileName = "default-config.yaml"
	}
	bucketName := vars["buckets_name"]
	bucketPath := vars["buckets_path"]
	file := filepath.Join(c.Path, fileName)
	if _, err := os.Stat(file); err != nil {
		return fmt.Errorf("上传失败，未找到文件: %s", file)
	}
	return runRcloneUpload(c.Path, file, bucketName, bucketPath)
}

// changDNS 根据模板参数自动更新 Cloudflare A 记录，逻辑参考 deploy.sh 中的 CFAddRecords
func changDNS(c *Case) error {
	// 确保 output 已加载
	if c.output == nil {
		if _, err := c.TfOutput(); err != nil {
			return fmt.Errorf("获取场景 output 失败: %w", err)
		}
	}

	params := parseVarMap(c.Parameter)
	domain := strings.TrimSpace(params["domain"])
	if domain == "" {
		return fmt.Errorf("chang_dns: 未找到 domain 参数")
	}

	gologger.Info().Msgf("chang_dns: 输入 domain=%s", domain)

	// 获取实例公网 IP（取第一个）
	ips, err := collectIPs(c)
	if err != nil {
		return err
	}
	if len(ips) == 0 {
		return fmt.Errorf("chang_dns: 未获取到实例 IP")
	}
	targetIP := ips[0]
	gologger.Info().Msgf("chang_dns: 目标 IP=%s", targetIP)

	client, cfConf, ok := newCFClientFromConfig()
	gologger.Info().Msg("chang_dns: 已调用 newCFClientFromConfig")
	if !ok {
		gologger.Warning().Msg("未配置 Cloudflare API（需要 CF_EMAIL 与 CF_API_KEY），跳过 DNS 修改")
		return nil
	}
	gologger.Info().Msgf("chang_dns: Cloudflare email=%s (key 已加载=%v)", cfConf.Email, cfConf.APIKey != "")

	var zoneName string
	zoneName = extractZoneName(domain)
	if zoneName == "" {
		zoneName = strings.TrimSpace(cfConf.Zone)
	}
	if zoneName == "" {
		return fmt.Errorf("chang_dns: 未确定 zone，domain/CF_ZONE 均为空")
	}
	recordName := defaultRecordName(zoneName, cfConf.Record)
	gologger.Info().Msgf("chang_dns: zone=%s record=%s", zoneName, recordName)

	zoneID, err := client.getZoneID(zoneName)
	if err != nil {
		return fmt.Errorf("获取 Cloudflare zone 失败: %w", err)
	}
	gologger.Info().Msgf("chang_dns: zoneID=%s", zoneID)

	if err := client.upsertARecord(zoneID, recordName, targetIP); err != nil {
		return fmt.Errorf("更新 Cloudflare A 记录失败: %w", err)
	}

	gologger.Info().Msgf("Cloudflare DNS 已更新: %s -> %s", recordName, targetIP)
	return nil
}

func parseVarMap(vars []string) map[string]string {
	res := make(map[string]string, len(vars))
	for _, raw := range vars {
		parts := strings.SplitN(raw, "=", 2)
		if len(parts) != 2 {
			continue
		}
		res[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}
	return res
}

// extractZoneName 默认取 domain 的后两段
func extractZoneName(domain string) string {
	domain = strings.TrimSuffix(strings.ToLower(domain), ".")
	parts := strings.Split(domain, ".")
	if len(parts) >= 2 {
		return strings.Join(parts[len(parts)-2:], ".")
	}
	return domain
}

func defaultRecordName(zone string, override string) string {
	if name := strings.TrimSpace(override); name != "" {
		return name
	}
	return "ns1." + zone
}

type cfAPIError struct {
	Message string `json:"message"`
}

type cfClient struct {
	email      string
	key        string
	httpClient *http.Client
}

type cfSettings struct {
	Email  string
	APIKey string
	Zone   string
	Record string
}

// newCFClientFromConfig 优先从 redc 配置文件读取 Cloudflare 凭证，其次再回退到环境变量
func newCFClientFromConfig() (*cfClient, cfSettings, bool) {
	gologger.Info().Msg("chang_dns: 读取 Cloudflare 配置")
	cfConf := loadCFSettings()
	gologger.Info().Msgf("chang_dns: 配置文件读取完成 email=%s apiKey?=%v", cfConf.Email, cfConf.APIKey != "")

	email := strings.TrimSpace(cfConf.Email)
	key := strings.TrimSpace(cfConf.APIKey)

	// 环境变量兜底
	if email == "" {
		email = strings.TrimSpace(os.Getenv("CF_EMAIL"))
		if email == "" {
			email = strings.TrimSpace(os.Getenv("CF_API_EMAIL"))
		}
	}
	if key == "" {
		key = strings.TrimSpace(os.Getenv("CF_API_KEY"))
	}
	if key == "" {
		key = strings.TrimSpace(os.Getenv("CF_KEY"))
	}
	gologger.Info().Msgf("chang_dns: 环境变量加载 email=%s apiKey?=%v", email, key != "")

	// 区域/记录兜底
	if cfConf.Zone == "" {
		cfConf.Zone = strings.TrimSpace(os.Getenv("CF_ZONE"))
	}
	if cfConf.Record == "" {
		cfConf.Record = strings.TrimSpace(os.Getenv("CF_RECORD"))
	}

	cfConf.Email = email
	cfConf.APIKey = key

	if email == "" || key == "" {
		gologger.Warning().Msg("chang_dns: Cloudflare 凭证缺失，返回 not ok")
		return nil, cfConf, false
	}

	return &cfClient{
		email:      email,
		key:        key,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}, cfConf, true
}

// loadCFSettings 读取 redc 配置文件中的 Cloudflare 配置，尽量兼容大小写 key
func loadCFSettings() cfSettings {
	var conf cfSettings
	gologger.Info().Msg("chang_dns: loadCFSettings begin")

	cfgPath := filepath.Join(RedcPath, "config.yaml")
	if RedcPath == "" {
		if home, err := os.UserHomeDir(); err == nil {
			cfgPath = filepath.Join(home, "redc", "config.yaml")
		}
	}
	gologger.Info().Msgf("chang_dns: loadCFSettings path=%s", cfgPath)

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		gologger.Info().Msgf("chang_dns: loadCFSettings read err=%v", err)
		return conf
	}

	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		gologger.Info().Msgf("chang_dns: loadCFSettings unmarshal err=%v", err)
		return conf
	}

	cfRaw, ok := raw["cloudflare"]
	if !ok {
		// 兼容配置写在 providers.cloudflare 下的情况
		if provRaw, okProv := raw["providers"]; okProv {
			if provMap, okMap := provRaw.(map[string]interface{}); okMap {
				cfRaw, ok = provMap["cloudflare"]
				if ok {
					gologger.Info().Msg("chang_dns: loadCFSettings 使用 providers.cloudflare")
				}
			} else {
				gologger.Info().Msg("chang_dns: loadCFSettings providers 节点类型不符")
			}
		}
	}
	if !ok {
		gologger.Info().Msg("chang_dns: loadCFSettings 未找到 cloudflare 节点")
		return conf
	}
	m, ok := cfRaw.(map[string]interface{})
	if !ok {
		gologger.Info().Msg("chang_dns: loadCFSettings cloudflare 节点类型不符")
		return conf
	}

	conf.Email = pickCFValue(m, "email", "cf_email", "CF_EMAIL")
	conf.APIKey = pickCFValue(m, "api_key", "cf_api_key", "CF_API_KEY", "apiKey")
	return conf
}

func pickCFValue(m map[string]interface{}, keys ...string) string {
	for _, k := range keys {
		for variant, val := range m {
			if strings.EqualFold(variant, k) {
				if s, ok := val.(string); ok {
					return strings.TrimSpace(s)
				}
			}
		}
	}
	return ""
}

func (c *cfClient) applyAuth(req *http.Request) {
	req.Header.Set("X-Auth-Email", c.email)
	req.Header.Set("X-Auth-Key", c.key)
	req.Header.Set("Content-Type", "application/json")
}

func (c *cfClient) do(method, url string, payload interface{}, out interface{}) error {
	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return err
	}
	c.applyAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("Cloudflare API 返回错误(%d): %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	if out != nil {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("解析 Cloudflare 响应失败: %w", err)
		}
	}
	return nil
}

func (c *cfClient) getZoneID(zone string) (string, error) {
	var res struct {
		Success bool         `json:"success"`
		Errors  []cfAPIError `json:"errors"`
		Result  []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"result"`
	}

	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones?name=%s", zone)
	if err := c.do(http.MethodGet, url, nil, &res); err != nil {
		return "", err
	}
	if !res.Success {
		return "", fmt.Errorf("查询 zone 失败: %s", joinCFErrors(res.Errors))
	}
	if len(res.Result) == 0 {
		return "", fmt.Errorf("未找到 zone: %s", zone)
	}
	return res.Result[0].ID, nil
}

func (c *cfClient) findARecord(zoneID, recordName string) (string, error) {
	var res struct {
		Success bool         `json:"success"`
		Errors  []cfAPIError `json:"errors"`
		Result  []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"result"`
	}

	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records?type=A&name=%s", zoneID, recordName)
	if err := c.do(http.MethodGet, url, nil, &res); err != nil {
		return "", err
	}
	if !res.Success {
		return "", fmt.Errorf("查询 A 记录失败: %s", joinCFErrors(res.Errors))
	}
	if len(res.Result) == 0 {
		return "", nil
	}
	return res.Result[0].ID, nil
}

func (c *cfClient) upsertARecord(zoneID, recordName, ip string) error {
	recordID, err := c.findARecord(zoneID, recordName)
	if err != nil {
		return err
	}

	payload := map[string]interface{}{
		"type":    "A",
		"name":    recordName,
		"content": ip,
		"ttl":     3600,
		"proxied": false,
	}

	if recordID == "" {
		var res struct {
			Success bool         `json:"success"`
			Errors  []cfAPIError `json:"errors"`
		}
		url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records", zoneID)
		if err := c.do(http.MethodPost, url, payload, &res); err != nil {
			return err
		}
		if !res.Success {
			return fmt.Errorf("创建 A 记录失败: %s", joinCFErrors(res.Errors))
		}
		return nil
	}

	var res struct {
		Success bool         `json:"success"`
		Errors  []cfAPIError `json:"errors"`
	}
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records/%s", zoneID, recordID)
	if err := c.do(http.MethodPut, url, payload, &res); err != nil {
		return err
	}
	if !res.Success {
		return fmt.Errorf("更新 A 记录失败: %s", joinCFErrors(res.Errors))
	}
	return nil
}

func joinCFErrors(errs []cfAPIError) string {
	if len(errs) == 0 {
		return ""
	}
	var parts []string
	for _, e := range errs {
		if strings.TrimSpace(e.Message) != "" {
			parts = append(parts, e.Message)
		}
	}
	return strings.Join(parts, "; ")
}
