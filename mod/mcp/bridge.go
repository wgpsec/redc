package mcp

import "time"

// AppBridge defines the interface for MCP tools to access App-layer functionality.
// Implemented by the main App struct to avoid circular dependency (mod/mcp cannot import main).
// Methods return interface{} to avoid referencing main-package types.
type AppBridge interface {
	// Compose
	MCPComposePreview(filePath string, profiles []string) (interface{}, error)
	MCPComposeUp(filePath string, profiles []string) error
	MCPComposeDown(filePath string, profiles []string) error

	// Cost & Resources
	MCPGetCostEstimate(templateName string, variables map[string]string) (interface{}, error)
	MCPGetBalances(providers []string) (interface{}, error)
	MCPGetResourceSummary() (interface{}, error)
	MCPGetPredictedMonthlyCost() (string, error)
	MCPGetBills(providers []string) (interface{}, error)
	MCPGetTotalRuntime() (string, error)

	// Custom Deployments
	MCPListCustomDeployments() (interface{}, error)
	MCPStartCustomDeployment(id string) error
	MCPStopCustomDeployment(id string) error

	// Projects & Profiles
	MCPListProjects() (interface{}, error)
	MCPSwitchProject(projectName string) error
	MCPListProfiles() (interface{}, error)
	MCPGetActiveProfile() (interface{}, error)
	MCPSetActiveProfile(profileID string) (interface{}, error)

	// Scheduler
	MCPScheduleTask(caseID string, caseName string, action string, scheduledAt time.Time) (interface{}, error)
	MCPListScheduledTasks() interface{}
	MCPCancelScheduledTask(taskID string) error

	// Template write
	MCPSaveTemplateFiles(templateName string, files map[string]string) (string, error)

	// Compose file write
	MCPSaveComposeFile(filename string, content string) (string, error)
}
