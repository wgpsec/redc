package main

import (
	"archive/zip"
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"red-cloud/i18n"
	redc "red-cloud/mod"
	"regexp"
	"strings"
	"time"
)

func (a *App) ListTemplates() ([]TemplateInfo, error) {
	templates, err := redc.ListLocalTemplates()
	if err != nil {
		return nil, err
	}
	result := make([]TemplateInfo, 0, len(templates))
	for _, t := range templates {
		result = append(result, TemplateInfo{
			Name:          t.Name,
			Description:   t.Description,
			DescriptionEN: t.DescriptionEN,
			Version:       t.Version,
			User:          t.User,
			Module:        t.RedcModule,
			Plugins:       t.RedcPlugins,
		})
	}
	return result, nil
}

func (a *App) ListAllTemplates() ([]TemplateInfo, error) {
	templates, err := redc.ListAllTemplates()
	if err != nil {
		return nil, err
	}
	result := make([]TemplateInfo, 0, len(templates))
	for _, t := range templates {
		result = append(result, TemplateInfo{
			Name:          t.Name,
			Description:   t.Description,
			DescriptionEN: t.DescriptionEN,
			Version:       t.Version,
			User:          t.User,
			Module:        t.RedcModule,
			Plugins:       t.RedcPlugins,
		})
	}
	return result, nil
}

// GetTemplatesDir returns the local templates directory path
func (a *App) GetTemplatesDir() string {
	return redc.TemplateDir
}

func (a *App) GetTemplateVariables(templateName string) ([]TemplateVariable, error) {
	templatePath := filepath.Join(redc.TemplateDir, templateName)

	// Parse variables.tf to get variable definitions
	variablesFile := filepath.Join(templatePath, "variables.tf")
	tfvarsFile := filepath.Join(templatePath, "terraform.tfvars")

	variables := make(map[string]*TemplateVariable)

	// Parse variables.tf
	if _, err := os.Stat(variablesFile); err == nil {
		vars, err := parseVariablesTf(variablesFile)
		if err != nil {
			return nil, fmt.Errorf(i18n.Tf("app_parse_variables_failed", err))
		}
		for _, v := range vars {
			variables[v.Name] = v
		}
	}

	// Parse terraform.tfvars for default values
	if _, err := os.Stat(tfvarsFile); err == nil {
		defaults, err := parseTfvars(tfvarsFile)
		if err != nil {
			return nil, fmt.Errorf(i18n.Tf("app_parse_tfvars_failed", err))
		}
		for name, value := range defaults {
			if v, ok := variables[name]; ok {
				v.DefaultValue = value
			}
		}
	}

	// Convert map to slice
	result := make([]TemplateVariable, 0, len(variables))
	for _, v := range variables {
		v.Required = true
		result = append(result, *v)
	}
	return result, nil
}

func parseVariablesTf(filePath string) ([]*TemplateVariable, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var variables []*TemplateVariable
	scanner := bufio.NewScanner(file)

	varNameRegex := regexp.MustCompile(`^variable\s+"([^"]+)"`)
	typeRegex := regexp.MustCompile(`^\s*type\s*=\s*(.+)`)
	descRegex := regexp.MustCompile(`^\s*description\s*=\s*"([^"]*)"`)
	defaultRegex := regexp.MustCompile(`^\s*default\s*=\s*(.+)`)
	sensitiveRegex := regexp.MustCompile(`^\s*sensitive\s*=\s*true`)

	var currentVar *TemplateVariable
	braceCount := 0

	for scanner.Scan() {
		line := scanner.Text()

		// Check for variable declaration
		if matches := varNameRegex.FindStringSubmatch(line); len(matches) > 1 {
			if currentVar != nil {
				variables = append(variables, currentVar)
			}
			currentVar = &TemplateVariable{
				Name:     matches[1],
				Required: true,
				Type:     "string",
			}
			braceCount = 1
			continue
		}

		if currentVar == nil {
			continue
		}

		// Count braces
		braceCount += strings.Count(line, "{") - strings.Count(line, "}")

		// Parse type
		if matches := typeRegex.FindStringSubmatch(line); len(matches) > 1 {
			currentVar.Type = strings.TrimSpace(matches[1])
		}

		// Parse description
		if matches := descRegex.FindStringSubmatch(line); len(matches) > 1 {
			currentVar.Description = matches[1]
		}

		// Parse default
		if matches := defaultRegex.FindStringSubmatch(line); len(matches) > 1 {
			defaultRaw := strings.TrimSpace(matches[1])
			currentVar.DefaultValue = strings.Trim(defaultRaw, `"`)
		}

		// Parse sensitive
		if sensitiveRegex.MatchString(line) {
			currentVar.Sensitive = true
		}

		// End of variable block
		if braceCount <= 0 && currentVar != nil {
			variables = append(variables, currentVar)
			currentVar = nil
		}
	}

	// Add last variable if exists
	if currentVar != nil {
		variables = append(variables, currentVar)
	}

	return variables, scanner.Err()
}

func (a *App) FetchRegistryTemplates(registryURL string) ([]RegistryTemplate, error) {
	if registryURL == "" {
		registryURL = "https://redc.wgpsec.org"
	}

	a.emitLog(i18n.Tf("app_connecting_registry", registryURL))

	// Fetch index.json
	indexURL := fmt.Sprintf("%s/index.json?t=%d", registryURL, time.Now().Unix())

	client := redc.NewProxyHTTPClient(30 * time.Second)
	resp, err := client.Get(indexURL)
	if err != nil {
		return nil, fmt.Errorf(i18n.Tf("app_registry_connect_failed", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(i18n.Tf("app_registry_status_error", resp.Status))
	}

	var idx remoteIndexResponse
	if err := json.NewDecoder(resp.Body).Decode(&idx); err != nil {
		return nil, fmt.Errorf(i18n.Tf("app_registry_parse_failed", err))
	}

	// Build result list
	result := make([]RegistryTemplate, 0, len(idx.Templates))
	for templateID, t := range idx.Templates {
		// Use templateID (e.g. "aliyun/ecs") as the name
		name := templateID
		if name == "" {
			name = t.ID
		}

		// Check if installed locally
		installed, localVer, _ := redc.CheckLocalImage(name)

		// Get version list
		versions := make([]string, 0, len(t.Versions))
		var updatedAt string
		for v, info := range t.Versions {
			versions = append(versions, v)
			if v == t.Latest && info.UpdatedAt != "" {
				updatedAt = info.UpdatedAt
			}
		}

		// Extract tags: prefer remote metadata tags, fallback to local case.json
		var tags []string
		if len(t.Metadata.Tags) > 0 {
			tags = t.Metadata.Tags
		} else if t.Provider != "" {
			tags = []string{t.Provider}
		}

		// For installed templates, read tags from local case.json
		if installed {
			localCasePath := filepath.Join(redc.TemplateDir, name, "case.json")
			if data, err := os.ReadFile(localCasePath); err == nil {
				var localCase struct {
					Tags []string `json:"tags"`
				}
				if json.Unmarshal(data, &localCase) == nil && len(localCase.Tags) > 0 {
					tags = localCase.Tags
				}
			}
		}

		result = append(result, RegistryTemplate{
			Name:          name,
			Description:   t.Metadata.Description,
			DescriptionEN: t.Metadata.DescriptionEN,
			Author:        t.Metadata.Author,
			Latest:      t.Latest,
			Versions:    versions,
			UpdatedAt:   updatedAt,
			Tags:        tags,
			Installed:   installed,
			LocalVer:    localVer,
		})
	}

	a.emitLog(i18n.Tf("app_fetched_templates", len(result)))
	return result, nil
}

func (a *App) FetchTemplateReadme(templateName string, lang string) (string, error) {
	readmeFiles := []string{}
	if lang == "en" {
		readmeFiles = []string{"README_EN.md", "README.md"}
	} else {
		readmeFiles = []string{"README.md", "README_EN.md"}
	}

	var lastErr error
	for _, readmeFile := range readmeFiles {
		readmeURL := fmt.Sprintf("https://raw.githubusercontent.com/wgpsec/redc-template/master/%s/%s", templateName, readmeFile)
		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Get(readmeURL)
		if err != nil {
			lastErr = err
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			content, err := io.ReadAll(resp.Body)
			if err != nil {
				lastErr = err
				continue
			}
			return string(content), nil
		}
		lastErr = fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return "", fmt.Errorf("failed to fetch README: %v", lastErr)
}

func (a *App) RemoveTemplate(templateName string) error {
	a.emitLog(i18n.Tf("app_deleting_template", templateName))

	if err := redc.RemoveTemplate(templateName); err != nil {
		a.emitLog(i18n.Tf("app_delete_failed", err))
		return err
	}

	a.emitLog(i18n.Tf("app_template_deleted", templateName))
	a.emitRefresh()
	return nil
}

func (a *App) PullTemplate(templateName string, force bool) error {
	a.emitLog(i18n.Tf("app_pulling_template", templateName))

	go func() {
		defer func() {
			if r := recover(); r != nil {
				a.emitLog(i18n.Tf("app_pull_error", r))
			}
			a.emitRefresh()
		}()

		opts := redc.PullOptions{
			RegistryURL: "https://redc.wgpsec.org",
			Force:       force,
			Timeout:     120 * time.Second,
		}

		if err := redc.Pull(context.Background(), templateName, opts); err != nil {
			a.emitLog(i18n.Tf("app_pull_failed", err))
			return
		}

		a.emitLog(i18n.Tf("app_template_pulled", templateName))
	}()

	return nil
}

func (a *App) CopyTemplate(sourceName string, targetName string) error {
	if err := redc.CopyTemplate(sourceName, targetName); err != nil {
		a.emitLog(i18n.Tf("app_template_copy_failed", err))
		return err
	}
	a.emitLog(i18n.Tf("app_template_copy_success", sourceName, targetName))
	a.emitRefresh()
	return nil
}

func (a *App) GetTemplateFiles(templateName string) (map[string]string, error) {
	path, err := redc.GetTemplatePath(templateName)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	files := make(map[string]string)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		// Read case.json, terraform.tfvars, *.tf, userdata files, and compose files
		if name == "case.json" || name == "terraform.tfvars" ||
			strings.HasSuffix(name, ".tf") ||
			strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml") ||
			strings.HasSuffix(name, ".sh") || strings.HasSuffix(name, ".userdata") ||
			strings.HasSuffix(name, ".md") ||
			name == "userdata" || name == "script.sh" || name == "README" {
			data, err := os.ReadFile(filepath.Join(path, name))
			if err != nil {
				continue
			}
			files[name] = string(data)
		}
	}
	return files, nil
}

func (a *App) ListUserdataTemplates() ([]redc.UserdataTemplate, error) {
	templates, err := redc.ListUserdataTemplates()
	if err != nil {
		return nil, err
	}
	result := make([]redc.UserdataTemplate, 0, len(templates))
	for _, t := range templates {
		result = append(result, *t)
	}
	return result, nil
}

func (a *App) ListComposeTemplates() ([]redc.ComposeTemplate, error) {
	templates, err := redc.ListComposeTemplates()
	if err != nil {
		return nil, err
	}
	result := make([]redc.ComposeTemplate, 0, len(templates))
	for _, t := range templates {
		result = append(result, *t)
	}
	return result, nil
}

func (a *App) SaveTemplateFiles(templateName string, files map[string]string) (string, error) {
	// 检查是否是 AI 模板（通过检查 templateName 是否包含 ai- 前缀）
	isAI := strings.HasPrefix(templateName, "ai-") || strings.HasPrefix(templateName, "AI-")
	path, err := redc.ResolveTemplatePath(templateName, isAI)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(path, 0755); err != nil {
		return "", fmt.Errorf("failed to create template directory: %w", err)
	}
	for name, content := range files {
		// Save case.json, terraform.tfvars, *.tf, userdata files, and compose files
		if name == "case.json" || name == "terraform.tfvars" ||
			strings.HasSuffix(name, ".tf") ||
			strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml") ||
			strings.HasSuffix(name, ".sh") || strings.HasSuffix(name, ".userdata") ||
			strings.HasSuffix(name, ".md") ||
			name == "userdata" || name == "script.sh" || name == "README" {
			if err := os.WriteFile(filepath.Join(path, name), []byte(content), 0644); err != nil {
				return "", err
			}
		}
	}
	absPath, _ := filepath.Abs(path)
	a.emitLog(i18n.Tf("app_template_save_success", templateName))
	return absPath, nil
}

func (a *App) ExportTemplates(templateNames []string) (string, error) {
	if len(templateNames) == 0 {
		return "", fmt.Errorf("no templates selected")
	}

	// Create temp zip file
	tmpFile, err := os.CreateTemp("", "redc-templates-*.zip")
	if err != nil {
		return "", err
	}
	tmpFile.Close()
	zipPath := tmpFile.Name()

	zipFile, err := os.OpenFile(zipPath, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		os.Remove(zipPath)
		return "", err
	}
	defer zipFile.Close()
	// Note: Don't remove zipPath here, let the frontend copy it first

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	for _, name := range templateNames {
		path, err := redc.GetTemplatePath(name)
		if err != nil {
			continue // Skip if template not found
		}

		// Walk through template directory
		err = filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}

			relPath, err := filepath.Rel(redc.TemplateDir, filePath)
			if err != nil {
				return nil
			}

			// Read file content
			content, err := os.ReadFile(filePath)
			if err != nil {
				return nil
			}

			// Add to zip - use relPath directly (already includes template name)
			w, err := zipWriter.Create(relPath)
			if err != nil {
				return nil
			}
			w.Write(content)
			return nil
		})
		if err != nil {
			continue
		}
	}

	zipWriter.Close()
	zipFile.Close()

	return zipPath, nil
}

func (a *App) ImportTemplates(zipPath string) ([]string, error) {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	imported := []string{}
	importedSet := make(map[string]bool)

	for _, file := range reader.File {
		// Skip directories
		if file.FileInfo().IsDir() {
			continue
		}

		name := file.Name
		// ZIP path is already correct, e.g., "aliyun/ecs/case.json"
		// Extract template name for tracking
		parts := strings.Split(name, "/")
		if len(parts) < 2 {
			continue
		}
		templateName := strings.Join(parts[:len(parts)-1], "/")

		// Extract file to TemplateDir
		destPath := filepath.Join(redc.TemplateDir, name)
		srcFile, err := file.Open()
		if err != nil {
			continue
		}
		defer srcFile.Close()

		// Create parent directories
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			continue
		}

		destFile, err := os.Create(destPath)
		if err != nil {
			continue
		}
		defer destFile.Close()

		if _, err := io.Copy(destFile, srcFile); err != nil {
			continue
		}

		// Track imported templates (use full template path)
		if !importedSet[templateName] {
			importedSet[templateName] = true
			imported = append(imported, templateName)
		}
	}

	return imported, nil
}

func (a *App) CopyFileTo(sourcePath string, destPath string) error {
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	// Create destination directory if not exists
	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}

	destFile, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

// validTemplateName checks if a template name is safe (allows letters, digits, -, _, /)
var validTemplateNameRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_\-/]*[a-zA-Z0-9]$|^[a-zA-Z0-9]$`)

// CreateLocalTemplate creates a new empty template directory with scaffold files
func (a *App) CreateLocalTemplate(name string, scaffold string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf(i18n.T("app_template_name_empty"))
	}
	if !validTemplateNameRe.MatchString(name) {
		return fmt.Errorf(i18n.T("app_template_name_invalid"))
	}
	// Block path traversal
	if strings.Contains(name, "..") {
		return fmt.Errorf(i18n.T("app_template_name_invalid"))
	}

	tmplPath := filepath.Join(redc.TemplateDir, name)
	if _, err := os.Stat(tmplPath); err == nil {
		return fmt.Errorf(i18n.T("app_template_already_exists"))
	}

	if err := os.MkdirAll(tmplPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write scaffold files
	files := scaffoldFiles(scaffold, name)
	for fname, content := range files {
		if err := os.WriteFile(filepath.Join(tmplPath, fname), []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", fname, err)
		}
	}

	a.emitLog(i18n.Tf("app_template_create_success", name))
	a.emitRefresh()
	return nil
}

func scaffoldFiles(scaffold string, name string) map[string]string {
	baseName := filepath.Base(name)

	makeCaseJSON := func(tmplType string) string {
		return fmt.Sprintf(`{
  "name": "%s",
  "description": "",
  "version": "1.0.0",
  "user": "",
  "template": "%s"
}`, baseName, tmplType)
	}

	mainTF := `# =============================================================================
# Terraform Main Configuration
# =============================================================================
#
# Configure your provider and resources below.
#
# Example provider block:
#   terraform {
#     required_providers {
#       alicloud = {
#         source  = "aliyun/alicloud"
#         version = "~> 1.200"
#       }
#     }
#   }
#
#   provider "alicloud" {
#     region = var.region
#   }
#
# Example resource block:
#   resource "alicloud_instance" "main" {
#     instance_type        = var.instance_type
#     image_id             = "ubuntu_22_04_x64_20G_alibase_20230815.vhd"
#     security_groups      = [alicloud_security_group.default.id]
#     vswitch_id           = alicloud_vswitch.default.id
#     instance_name        = var.instance_name
#     system_disk_category = "cloud_efficiency"
#     user_data            = file("userdata")
#   }
#
`

	variablesTF := `# =============================================================================
# Input Variables
# =============================================================================
#
# Define your input variables here.
#
# variable "region" {
#   description = "Cloud region"
#   type        = string
#   default     = "cn-hangzhou"
# }
#
# variable "instance_type" {
#   description = "Instance type"
#   type        = string
#   default     = "ecs.t6-c1m1.large"
# }
#
# variable "instance_name" {
#   description = "Instance name"
#   type        = string
#   default     = "redc-instance"
# }
#
# variable "instance_password" {
#   description = "Instance password"
#   type        = string
#   sensitive   = true
# }
`

	outputsTF := `# =============================================================================
# Outputs
# =============================================================================
#
# Define your outputs here. Outputs with "ip" in the name will be used
# by redc for SSH connections.
#
# output "ip" {
#   value = alicloud_instance.main.public_ip
# }
#
# output "password" {
#   value     = var.instance_password
#   sensitive = true
# }
`

	switch scaffold {
	case "preset":
		return map[string]string{
			"case.json":    makeCaseJSON("preset"),
			"main.tf":      mainTF,
			"variables.tf": variablesTF,
			"outputs.tf":   outputsTF,
		}
	case "preset-userdata":
		return map[string]string{
			"case.json":    makeCaseJSON("preset"),
			"main.tf":      mainTF,
			"variables.tf": variablesTF,
			"outputs.tf":   outputsTF,
			"userdata":     "#!/bin/bash\n# Add your initialization script here\necho \"Hello from redc\"\n",
		}
	case "base":
		return map[string]string{
			"case.json":    makeCaseJSON("base"),
			"main.tf":      mainTF,
			"variables.tf": variablesTF,
			"outputs.tf":   outputsTF,
		}
	case "userdata":
		userdataCaseJSON := fmt.Sprintf(`{
  "name": "%s",
  "description": "",
  "version": "1.0.0",
  "user": "",
  "type": "bash",
  "category": "custom",
  "template": "userdata"
}`, baseName)
		return map[string]string{
			"case.json": userdataCaseJSON,
			"userdata":  "#!/bin/bash\n# Add your initialization script here\necho \"Hello from redc\"\n",
		}
	case "compose":
		composeCaseJSON := fmt.Sprintf(`{
  "name": "%s",
  "description": "",
  "version": "1.0.0",
  "user": "",
  "template": "compose"
}`, baseName)
		composeYAML := `version: "3.9"

# =============================================================================
# Redc Compose - 多云编排配置
# =============================================================================
#
# 使用 redc compose up redc-compose.yaml 启动编排
# 使用 redc compose down -f redc-compose.yaml 销毁环境
# 使用 redc compose config redc-compose.yaml 预览配置
#

services:

  # 示例服务 - 修改 image 为你的模板路径 (如 aliyun/ecs)
  # my_server:
  #   image: aliyun/ecs
  #   container_name: my_ecs_instance
  #   environment:
  #     - instance_type=ecs.e-c1m2.large
  #     - password=YourPassword123
  #   command: |
  #     echo "实例初始化完成"
  #     uptime
`
		return map[string]string{
			"case.json":          composeCaseJSON,
			"redc-compose.yaml": composeYAML,
		}
	default: // backward compat — treat as preset
		return map[string]string{
			"case.json":    makeCaseJSON("preset"),
			"main.tf":      mainTF,
			"variables.tf": variablesTF,
			"outputs.tf":   outputsTF,
		}
	}
}

// DeleteTemplateFile deletes a single file from a template directory
func (a *App) DeleteTemplateFile(templateName string, fileName string) error {
	tmplPath, err := redc.GetTemplatePath(templateName)
	if err != nil {
		return err
	}

	// Safety: ensure fileName doesn't contain path traversal
	if strings.Contains(fileName, "..") || strings.Contains(fileName, "/") || strings.Contains(fileName, "\\") {
		return fmt.Errorf("invalid file name")
	}

	filePath := filepath.Join(tmplPath, fileName)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("file not found: %s", fileName)
	}

	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	a.emitLog(fmt.Sprintf("Deleted file %s from template %s", fileName, templateName))
	return nil
}

// TemplateValidateResult holds the result of a terraform validate run
type TemplateValidateResult struct {
	Valid        bool                       `json:"valid"`
	ErrorCount   int                        `json:"error_count"`
	WarningCount int                        `json:"warning_count"`
	Diagnostics  []TemplateValidateDiagnostic `json:"diagnostics"`
}

// TemplateValidateDiagnostic represents a single validation diagnostic
type TemplateValidateDiagnostic struct {
	Severity string `json:"severity"`
	Summary  string `json:"summary"`
	Detail   string `json:"detail"`
	Filename string `json:"filename"`
	Line     int    `json:"line"`
}

// ValidateTemplate runs terraform init + validate on a template to check syntax
func (a *App) ValidateTemplate(templateName string) (*TemplateValidateResult, error) {
	tmplPath, err := redc.GetTemplatePath(templateName)
	if err != nil {
		return nil, fmt.Errorf("%s", i18n.Tf("app_template_not_found", templateName))
	}

	a.emitLog(i18n.Tf("app_template_validate_start", templateName))

	// Step 1: terraform init (needed before validate)
	if err := redc.TfInit(tmplPath); err != nil {
		return &TemplateValidateResult{
			Valid:      false,
			ErrorCount: 1,
			Diagnostics: []TemplateValidateDiagnostic{
				{Severity: "error", Summary: "terraform init failed", Detail: err.Error()},
			},
		}, nil
	}

	// Step 2: terraform validate
	output, err := redc.TfValidate(tmplPath)
	if err != nil {
		return &TemplateValidateResult{
			Valid:      false,
			ErrorCount: 1,
			Diagnostics: []TemplateValidateDiagnostic{
				{Severity: "error", Summary: "terraform validate failed", Detail: err.Error()},
			},
		}, nil
	}

	result := &TemplateValidateResult{
		Valid:        output.Valid,
		ErrorCount:   output.ErrorCount,
		WarningCount: output.WarningCount,
	}

	for _, d := range output.Diagnostics {
		diag := TemplateValidateDiagnostic{
			Severity: string(d.Severity),
			Summary:  d.Summary,
			Detail:   d.Detail,
		}
		if d.Range != nil {
			diag.Filename = d.Range.Filename
			diag.Line = d.Range.Start.Line
		}
		result.Diagnostics = append(result.Diagnostics, diag)
	}

	if result.Valid {
		a.emitLog(i18n.Tf("app_template_validate_success", templateName))
	} else {
		a.emitLog(i18n.Tf("app_template_validate_failed", templateName))
	}

	return result, nil
}
