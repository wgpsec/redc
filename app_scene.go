package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"red-cloud/i18n"
	redc "red-cloud/mod"
)

// parseTfvars parses a terraform.tfvars file
func parseTfvars(filePath string) (map[string]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	defaults := make(map[string]string)
	scanner := bufio.NewScanner(file)

	lineRegex := regexp.MustCompile(`^([a-zA-Z_][a-zA-Z0-9_]*)\s*=\s*"?([^"]*)"?`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if matches := lineRegex.FindStringSubmatch(line); len(matches) > 2 {
			defaults[matches[1]] = matches[2]
		}
	}

	return defaults, scanner.Err()
}

// StartCase starts a case by ID
func (a *App) StartCase(caseID string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.project == nil {
		return fmt.Errorf("%s", i18n.T("app_project_not_loaded"))
	}

	c, err := a.project.GetCase(caseID)
	if err != nil {
		return fmt.Errorf(i18n.Tf("app_get_case_failed", err))
	}

	if c == nil {
		return fmt.Errorf("%s", i18n.T("app_case_nil"))
	}

	if c.Path == "" {
		return fmt.Errorf("%s", i18n.T("app_case_path_empty"))
	}

	caseName := c.Name
	casePath := c.Path
	caseState := c.State

	a.emitLog(i18n.Tf("app_scene_prepare_start", caseName, casePath, caseState))

	go func() {
		defer func() {
			if r := recover(); r != nil {
				a.emitLog(i18n.Tf("app_scene_start_error", r))
			}
			a.emitRefresh()
		}()

		a.emitLog(i18n.Tf("app_scene_starting", caseName))
		if err := c.TfApply(); err != nil {
			a.emitLog(i18n.Tf("app_scene_start_failed", err))
			if a.notificationMgr != nil {
				a.notificationMgr.SendSceneFailed(caseName, "启动")
			}
			return
		}
		a.emitLog(i18n.Tf("app_scene_start_success", caseName))

		if a.notificationMgr != nil {
			a.notificationMgr.SendSceneStarted(caseName)
		}

		if outputs, err := c.TfOutput(); err == nil {
			for name, meta := range outputs {
				a.emitLog(fmt.Sprintf("  %s = %s", name, string(meta.Value)))
			}
		}
	}()

	return nil
}

// StopCase stops a case by ID
func (a *App) StopCase(caseID string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.project == nil {
		return fmt.Errorf("%s", i18n.T("app_project_not_loaded"))
	}

	c, err := a.project.GetCase(caseID)
	if err != nil {
		return err
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				a.emitLog(i18n.Tf("app_scene_stop_error", r))
			}
			a.emitRefresh()
		}()

		a.emitLog(i18n.Tf("app_stopping_scene", c.Name))
		if err := c.Stop(); err != nil {
			a.emitLog(i18n.Tf("app_scene_stop_failed", err))
			if a.notificationMgr != nil {
				a.notificationMgr.SendSceneFailed(c.Name, "停止")
			}
			return
		}
		a.emitLog(i18n.Tf("app_scene_stop_success", c.Name))

		if a.notificationMgr != nil {
			a.notificationMgr.SendSceneStopped(c.Name)
		}
	}()

	return nil
}

// RemoveCase removes a case by ID
func (a *App) RemoveCase(caseID string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.project == nil {
		return fmt.Errorf("%s", i18n.T("app_project_not_loaded"))
	}

	c, err := a.project.GetCase(caseID)
	if err != nil {
		return err
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				a.emitLog(i18n.Tf("app_scene_delete_error", r))
			}
			a.emitRefresh()
		}()

		a.emitLog(i18n.Tf("app_deleting_scene", c.Name))
		if err := c.Remove(); err != nil {
			a.emitLog(i18n.Tf("app_scene_delete_failed", err))
			return
		}
		a.emitLog(i18n.Tf("app_scene_delete_success", c.Name))
	}()

	return nil
}

// CreateCase creates a new case from a template (async)
func (a *App) CreateCase(templateName string, name string, vars map[string]string) error {
	a.mu.Lock()
	if a.project == nil {
		a.mu.Unlock()
		return fmt.Errorf("%s", i18n.T("app_project_not_loaded"))
	}
	project := a.project
	a.mu.Unlock()

	a.emitLog(i18n.Tf("app_creating_scene", name, templateName))

	go func() {
		defer func() {
			if r := recover(); r != nil {
				a.emitLog(i18n.Tf("app_scene_init_error", r))
			}
			a.emitRefresh()
		}()

		a.emitLog(i18n.Tf("app_scene_initing", name, templateName))
		c, err := project.CaseCreate(templateName, redc.U, name, vars)
		if err != nil {
			a.emitLog(i18n.Tf("app_scene_create_failed", err))
			return
		}
		a.emitLog(i18n.Tf("app_scene_create_success", c.Name, c.GetId()))
	}()

	return nil
}

// CreateAndRunCase creates a new case and immediately starts it (like CLI "run" command)
func (a *App) CreateAndRunCase(templateName string, name string, vars map[string]string) error {
	a.mu.Lock()
	if a.project == nil {
		a.mu.Unlock()
		return fmt.Errorf("%s", i18n.T("app_project_not_loaded"))
	}
	project := a.project
	a.mu.Unlock()

	a.emitLog(i18n.Tf("app_creating_running_scene", name, templateName))

	go func() {
		defer func() {
			if r := recover(); r != nil {
				a.emitLog(i18n.Tf("app_scene_init_error", r))
			}
			a.emitRefresh()
		}()

		a.emitLog(i18n.Tf("app_scene_initing", name, templateName))
		c, err := project.CaseCreate(templateName, redc.U, name, vars)
		if err != nil {
			a.emitLog(i18n.Tf("app_scene_create_failed", err))
			return
		}
		a.emitLog(i18n.Tf("app_scene_create_success", c.Name, c.GetId()))

		a.emitLog(i18n.Tf("app_scene_starting", c.Name))

		if err := c.TfApply(); err != nil {
			a.emitLog(i18n.Tf("app_scene_start_failed", err))
			return
		}

		a.emitLog(i18n.Tf("app_scene_start_success", c.Name))

		if outputs, err := c.TfOutput(); err == nil {
			for key, meta := range outputs {
				a.emitLog(fmt.Sprintf("%s = %s", key, string(meta.Value)))
			}
		}

		a.emitRefresh()
	}()

	return nil
}

// DeployCase creates and immediately starts a case (deprecated - use CreateCase then StartCase)
func (a *App) DeployCase(templateName string, name string, vars map[string]string) error {
	return a.CreateCase(templateName, name, vars)
}

// GetCaseOutputs returns the terraform outputs for a case
func (a *App) GetCaseOutputs(caseID string) (map[string]string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.project == nil {
		return nil, fmt.Errorf("%s", i18n.T("app_project_not_loaded"))
	}

	c, err := a.project.GetCase(caseID)
	if err != nil {
		return nil, err
	}

	if c.State != "running" {
		return nil, nil
	}

	outputs, err := c.TfOutput()
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for name, meta := range outputs {
		value := string(meta.Value)
		if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
			value = value[1 : len(value)-1]
		}
		if isRelativeFilePath(value) {
			absPath := filepath.Join(c.Path, value)
			if _, err := os.Stat(absPath); err == nil {
				value = absPath
			}
		}
		result[name] = value
	}

	if c.Module != "" && (strings.Contains(c.Module, "gen_clash_config") || strings.Contains(c.Module, "upload_r2")) {
		tfvarsPath := filepath.Join(c.Path, "terraform.tfvars")
		if _, err := os.Stat(tfvarsPath); err == nil {
			if tfvars, err := parseTfvars(tfvarsPath); err == nil {
				fileName := strings.TrimSpace(tfvars["filename"])
				if fileName == "" {
					fileName = "default-config.yaml"
				}
				localConfig := filepath.Join(c.Path, "config.yaml")
				if _, err := os.Stat(localConfig); err == nil {
					result["clash_config_local"] = localConfig
				}
				bucketName := strings.TrimSpace(tfvars["buckets_name"])
				if bucketName == "" {
					bucketName = "test"
				}
				bucketPath := strings.Trim(tfvars["buckets_path"], "/")
				r2Path := fmt.Sprintf("r2:%s/%s", bucketName, fileName)
				if bucketPath != "" {
					r2Path = fmt.Sprintf("r2:%s/%s/%s", bucketName, bucketPath, fileName)
				}
				result["clash_config_r2"] = r2Path
			}
		}
	}
	return result, nil
}

// isRelativeFilePath checks if the value looks like a relative file path
func isRelativeFilePath(value string) bool {
	if value == "" {
		return false
	}
	if strings.HasPrefix(value, "./") || strings.HasPrefix(value, "../") {
		return true
	}
	return false
}
