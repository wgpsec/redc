package main

import (
	"fmt"
	"os"
	"path/filepath"
	redc "red-cloud/mod"
	"time"
)

// MCPComposePreview implements AppBridge
func (a *App) MCPComposePreview(filePath string, profiles []string) (interface{}, error) {
	return a.ComposePreview(filePath, profiles)
}

// MCPComposeUp implements AppBridge
func (a *App) MCPComposeUp(filePath string, profiles []string) error {
	return a.ComposeUp(filePath, profiles)
}

// MCPComposeDown implements AppBridge
func (a *App) MCPComposeDown(filePath string, profiles []string) error {
	return a.ComposeDown(filePath, profiles)
}

// MCPGetCostEstimate implements AppBridge
func (a *App) MCPGetCostEstimate(templateName string, variables map[string]string) (interface{}, error) {
	return a.GetCostEstimate(templateName, variables)
}

// MCPGetBalances implements AppBridge
func (a *App) MCPGetBalances(providers []string) (interface{}, error) {
	return a.GetBalances(providers)
}

// MCPGetResourceSummary implements AppBridge
func (a *App) MCPGetResourceSummary() (interface{}, error) {
	return a.GetResourceSummary()
}

// MCPGetPredictedMonthlyCost implements AppBridge
func (a *App) MCPGetPredictedMonthlyCost() (string, error) {
	return a.GetPredictedMonthlyCost()
}

// MCPGetBills implements AppBridge
func (a *App) MCPGetBills(providers []string) (interface{}, error) {
	return a.GetBills(providers)
}

// MCPGetTotalRuntime implements AppBridge
func (a *App) MCPGetTotalRuntime() (string, error) {
	return a.GetTotalRuntime()
}

// MCPListCustomDeployments implements AppBridge
func (a *App) MCPListCustomDeployments() (interface{}, error) {
	return a.ListCustomDeployments()
}

// MCPStartCustomDeployment implements AppBridge
func (a *App) MCPStartCustomDeployment(id string) error {
	return a.StartCustomDeployment(id)
}

// MCPStopCustomDeployment implements AppBridge
func (a *App) MCPStopCustomDeployment(id string) error {
	return a.StopCustomDeployment(id)
}

// MCPListProjects implements AppBridge
func (a *App) MCPListProjects() (interface{}, error) {
	return a.ListProjects()
}

// MCPSwitchProject implements AppBridge
func (a *App) MCPSwitchProject(projectName string) error {
	return a.SwitchProject(projectName)
}

// MCPListProfiles implements AppBridge
func (a *App) MCPListProfiles() (interface{}, error) {
	return a.ListProfiles()
}

// MCPGetActiveProfile implements AppBridge
func (a *App) MCPGetActiveProfile() (interface{}, error) {
	return a.GetActiveProfile()
}

// MCPSetActiveProfile implements AppBridge
func (a *App) MCPSetActiveProfile(profileID string) (interface{}, error) {
	return a.SetActiveProfile(profileID)
}

// MCPScheduleTask implements AppBridge
func (a *App) MCPScheduleTask(caseID string, caseName string, action string, scheduledAt time.Time) (interface{}, error) {
	return a.ScheduleTask(caseID, caseName, action, scheduledAt)
}

// MCPListScheduledTasks implements AppBridge
func (a *App) MCPListScheduledTasks() interface{} {
	return a.ListScheduledTasks()
}

// MCPCancelScheduledTask implements AppBridge
func (a *App) MCPCancelScheduledTask(taskID string) error {
	return a.CancelScheduledTask(taskID)
}

// MCPSaveTemplateFiles implements AppBridge
func (a *App) MCPSaveTemplateFiles(templateName string, files map[string]string) (string, error) {
	return a.SaveTemplateFiles(templateName, files)
}

// MCPSaveComposeFile implements AppBridge
func (a *App) MCPSaveComposeFile(filename string, content string) (string, error) {
	if filename == "" {
		filename = "redc-compose.yaml"
	}
	savePath := filepath.Join(redc.RedcPath, filename)
	if err := os.WriteFile(savePath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("写入 compose 文件失败: %v", err)
	}
	return savePath, nil
}
