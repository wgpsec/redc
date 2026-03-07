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
			Name:        t.Name,
			Description: t.Description,
			Version:     t.Version,
			User:        t.User,
			Module:      t.RedcModule,
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
			Name:        t.Name,
			Description: t.Description,
			Version:     t.Version,
			User:        t.User,
			Module:      t.RedcModule,
		})
	}
	return result, nil
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

		// Extract tags from provider
		var tags []string
		if t.Provider != "" {
			tags = []string{t.Provider}
		}

		result = append(result, RegistryTemplate{
			Name:        name,
			Description: t.Metadata.Description,
			Author:      t.Metadata.Author,
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
		resp, err := http.Get(readmeURL)
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
