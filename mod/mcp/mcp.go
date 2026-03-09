package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	redc "red-cloud/mod"
	"red-cloud/mod/gologger"
	"red-cloud/utils/sshutil"
)

// MCP Protocol Version
const MCPVersion = "2025-11-25"

// Supported protocol versions (for negotiation)
var supportedVersions = []string{"2025-11-25", "2024-11-05"}

// MCP Message types
type MCPRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type MCPResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *MCPError   `json:"error,omitempty"`
}

type MCPError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// MCP Server capabilities and info
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type Implementation struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type InitializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	ServerInfo      ServerInfo         `json:"serverInfo"`
	Instructions    string             `json:"instructions,omitempty"`
}

type ServerCapabilities struct {
	Tools     *ToolsCapability     `json:"tools,omitempty"`
	Resources *ResourcesCapability `json:"resources,omitempty"`
}

type ToolsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

type ResourcesCapability struct {
	Subscribe   bool `json:"subscribe,omitempty"`
	ListChanged bool `json:"listChanged,omitempty"`
}

// Tool definitions
type Tool struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	InputSchema ToolSchema `json:"inputSchema"`
}

type ToolSchema struct {
	Type       string              `json:"type"`
	Properties map[string]Property `json:"properties"`
	Required   []string            `json:"required,omitempty"`
}

type Property struct {
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Enum        []string `json:"enum,omitempty"`
}

// Tool call parameters
type CallToolParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

type ToolResult struct {
	Content []ContentItem `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

type ContentItem struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// Resource definitions
type Resource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

type ResourceContents struct {
	URI      string        `json:"uri"`
	MimeType string        `json:"mimeType,omitempty"`
	Contents []ContentItem `json:"contents"`
}

// LogCallback is a function type for logging
type LogCallback func(message string)

// MCPServer handles MCP protocol requests
type MCPServer struct {
	project   *redc.RedcProject
	app       AppBridge
	logWriter LogCallback
}

// NewMCPServer creates a new MCP server instance
func NewMCPServer(project *redc.RedcProject, app AppBridge) *MCPServer {
	return &MCPServer{
		project: project,
		app:     app,
	}
}

// SetLogCallback sets a callback for log messages
func (s *MCPServer) SetLogCallback(callback LogCallback) {
	s.logWriter = callback
}

func (s *MCPServer) log(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if s.logWriter != nil {
		s.logWriter(msg)
	}
	gologger.Info().Msg(msg)
}

// HandleRequest processes an MCP request and returns a response
func (s *MCPServer) HandleRequest(req *MCPRequest) *MCPResponse {
	resp := &MCPResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
	}

	switch req.Method {
	// Notifications — no response needed (return nil)
	case "notifications/initialized", "notifications/cancelled":
		return nil

	case "initialize":
		// Version negotiation: parse client's requested version
		negotiatedVersion := MCPVersion
		if req.Params != nil {
			var initParams struct {
				ProtocolVersion string `json:"protocolVersion"`
			}
			if err := json.Unmarshal(req.Params, &initParams); err == nil && initParams.ProtocolVersion != "" {
				// Check if we support the client's requested version
				clientSupported := false
				for _, v := range supportedVersions {
					if v == initParams.ProtocolVersion {
						clientSupported = true
						negotiatedVersion = initParams.ProtocolVersion
						break
					}
				}
				if !clientSupported {
					// Respond with our latest supported version
					negotiatedVersion = MCPVersion
				}
			}
		}
		resp.Result = InitializeResult{
			ProtocolVersion: negotiatedVersion,
			Capabilities: ServerCapabilities{
				Tools: &ToolsCapability{
					ListChanged: false,
				},
				Resources: &ResourcesCapability{
					Subscribe:   false,
					ListChanged: false,
				},
			},
			ServerInfo: ServerInfo{
				Name:    "redc",
				Version: redc.Version,
			},
			Instructions: "RedC MCP Server - Cloud resource deployment and management via Terraform",
		}

	case "tools/list":
		resp.Result = map[string]interface{}{
			"tools": s.getTools(),
		}

	case "tools/call":
		var params CallToolParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			resp.Error = &MCPError{
				Code:    -32602,
				Message: "Invalid params",
				Data:    err.Error(),
			}
		} else {
			result, err := s.executeTool(params.Name, params.Arguments)
			if err != nil {
				resp.Result = ToolResult{
					Content: []ContentItem{{
						Type: "text",
						Text: fmt.Sprintf("Error: %v", err),
					}},
					IsError: true,
				}
			} else {
				resp.Result = result
			}
		}

	case "resources/list":
		resp.Result = map[string]interface{}{
			"resources": s.getResources(),
		}

	case "resources/read":
		var params struct {
			URI string `json:"uri"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			resp.Error = &MCPError{
				Code:    -32602,
				Message: "Invalid params",
				Data:    err.Error(),
			}
		} else {
			result, err := s.readResource(params.URI)
			if err != nil {
				resp.Error = &MCPError{
					Code:    -32603,
					Message: "Failed to read resource",
					Data:    err.Error(),
				}
			} else {
				resp.Result = result
			}
		}

	case "ping":
		resp.Result = map[string]interface{}{}

	default:
		resp.Error = &MCPError{
			Code:    -32601,
			Message: "Method not found",
		}
	}

	return resp
}

func (s *MCPServer) getTools() []Tool {
	tools := []Tool{
		{
			Name:        "list_templates",
			Description: "List all available redc templates/images",
			InputSchema: ToolSchema{
				Type:       "object",
				Properties: map[string]Property{},
			},
		},
		{
			Name:        "search_templates",
			Description: "Search for templates in the official registry by keywords (provider, name, description, etc.)",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]Property{
					"query": {
						Type:        "string",
						Description: "Search query (e.g., 'aliyun', 'ecs', 'network', 'huawei/vpc')",
					},
					"registry_url": {
						Type:        "string",
						Description: "Registry base URL (optional, default: https://redc.wgpsec.org)",
					},
				},
				Required: []string{"query"},
			},
		},
		{
			Name:        "pull_template",
			Description: "Download a template from the registry (redc pull)",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]Property{
					"template": {
						Type:        "string",
						Description: "Template name (e.g., 'aliyun/ecs' or 'aliyun/ecs:1.0.1')",
					},
					"registry_url": {
						Type:        "string",
						Description: "Registry base URL (optional, default: https://redc.wgpsec.org)",
					},
					"force": {
						Type:        "boolean",
						Description: "Force re-download even if template exists (optional)",
					},
				},
				Required: []string{"template"},
			},
		},
		{
			Name:        "list_cases",
			Description: "List all running cases in the current project",
			InputSchema: ToolSchema{
				Type:       "object",
				Properties: map[string]Property{},
			},
		},
		{
			Name:        "plan_case",
			Description: "Plan a new case from a template (like terraform plan - preview resources without creating them)",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]Property{
					"template": {
						Type:        "string",
						Description: "Template name (e.g., 'aliyun/ecs')",
					},
					"name": {
						Type:        "string",
						Description: "Case name (optional, auto-generated if not provided)",
					},
					"env": {
						Type:        "object",
						Description: "Environment variables for the template (optional)",
					},
				},
				Required: []string{"template"},
			},
		},
		{
			Name:        "start_case",
			Description: "Start a case by ID",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]Property{
					"case_id": {
						Type:        "string",
						Description: "Case ID to start",
					},
				},
				Required: []string{"case_id"},
			},
		},
		{
			Name:        "stop_case",
			Description: "Stop a running case by ID",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]Property{
					"case_id": {
						Type:        "string",
						Description: "Case ID to stop",
					},
				},
				Required: []string{"case_id"},
			},
		},
		{
			Name:        "kill_case",
			Description: "Kill (destroy) a case by ID",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]Property{
					"case_id": {
						Type:        "string",
						Description: "Case ID to kill",
					},
				},
				Required: []string{"case_id"},
			},
		},
		{
			Name:        "get_case_status",
			Description: "Get the status of a specific case",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]Property{
					"case_id": {
						Type:        "string",
						Description: "Case ID to check",
					},
				},
				Required: []string{"case_id"},
			},
		},
		{
			Name:        "exec_command",
			Description: "Execute a command on a case",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]Property{
					"case_id": {
						Type:        "string",
						Description: "Case ID to execute command on",
					},
					"command": {
						Type:        "string",
						Description: "Command to execute",
					},
				},
				Required: []string{"case_id", "command"},
			},
		},
		{
			Name:        "get_ssh_info",
			Description: "Get SSH connection information for a case (IP, port, username)",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]Property{
					"case_id": {
						Type:        "string",
						Description: "Case ID to get SSH info",
					},
				},
				Required: []string{"case_id"},
			},
		},
		{
			Name:        "upload_file",
			Description: "Upload a local file to a case server via SCP/SFTP",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]Property{
					"case_id": {
						Type:        "string",
						Description: "Case ID to upload file to",
					},
					"local_path": {
						Type:        "string",
						Description: "Local file path to upload",
					},
					"remote_path": {
						Type:        "string",
						Description: "Remote destination path on the server",
					},
				},
				Required: []string{"case_id", "local_path", "remote_path"},
			},
		},
		{
			Name:        "download_file",
			Description: "Download a file from a case server to local machine via SCP/SFTP",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]Property{
					"case_id": {
						Type:        "string",
						Description: "Case ID to download file from",
					},
					"remote_path": {
						Type:        "string",
						Description: "Remote file path on the server",
					},
					"local_path": {
						Type:        "string",
						Description: "Local destination path",
					},
				},
				Required: []string{"case_id", "remote_path", "local_path"},
			},
		},
		{
			Name:        "get_template_info",
			Description: "Get detailed information about a locally installed template (including metadata, variables, files)",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]Property{
					"template_name": {
						Type:        "string",
						Description: "Template name (e.g., aliyun/ecs)",
					},
				},
				Required: []string{"template_name"},
			},
		},
		{
			Name:        "delete_template",
			Description: "Delete a locally installed template",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]Property{
					"template_name": {
						Type:        "string",
						Description: "Template name to delete (e.g., aliyun/ecs)",
					},
				},
				Required: []string{"template_name"},
			},
		},
		{
			Name:        "get_case_outputs",
			Description: "Get terraform outputs for a case (IP addresses, instance IDs, etc.)",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]Property{
					"case_id": {
						Type:        "string",
						Description: "Case ID to get outputs",
					},
				},
				Required: []string{"case_id"},
			},
		},
		{
			Name:        "get_config",
			Description: "Get redc current configuration (project path, proxy settings, etc.)",
			InputSchema: ToolSchema{
				Type:       "object",
				Properties: map[string]Property{},
			},
		},
		{
			Name:        "validate_config",
			Description: "Validate cloud provider configuration (check if credentials, region, instance type are valid)",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]Property{
					"provider": {
						Type:        "string",
						Description: "Cloud provider name (e.g., aliyun, tencentcloud, aws, volcengine, huaweicloud)",
					},
					"region": {
						Type:        "string",
						Description: "Region ID (e.g., cn-hangzhou)",
					},
					"instance_type": {
						Type:        "string",
						Description: "Instance type (e.g., ecs.t6-c1m1.large)",
					},
				},
				Required: []string{"provider"},
			},
		},
	}

	// Append extended tools (require AppBridge)
	if s.app != nil {
		tools = append(tools, composeToolSchemas()...)
		tools = append(tools, costToolSchemas()...)
		tools = append(tools, deploymentToolSchemas()...)
		tools = append(tools, projectToolSchemas()...)
		tools = append(tools, schedulerToolSchemas()...)
	}

	return tools
}

func (s *MCPServer) getResources() []Resource {
	return []Resource{
		{
			URI:         "redc://templates",
			Name:        "Available Templates",
			Description: "List of all available redc templates",
			MimeType:    "application/json",
		},
		{
			URI:         "redc://cases",
			Name:        "Running Cases",
			Description: "List of all cases in the current project",
			MimeType:    "application/json",
		},
		{
			URI:         "redc://config",
			Name:        "Configuration",
			Description: "Current redc configuration",
			MimeType:    "application/json",
		},
	}
}

func (s *MCPServer) readResource(uri string) (interface{}, error) {
	switch uri {
	case "redc://templates":
		dirs, err := redc.ScanTemplateDirs(redc.TemplateDir, redc.MaxTfDepth)
		if err != nil {
			return nil, err
		}
		data, err := json.MarshalIndent(dirs, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal templates: %v", err)
		}
		return ResourceContents{
			URI:      uri,
			MimeType: "application/json",
			Contents: []ContentItem{{
				Type: "text",
				Text: string(data),
			}},
		}, nil

	case "redc://cases":
		cases, err := redc.LoadProjectCases(s.project.ProjectName)
		if err != nil {
			return nil, err
		}
		caseList := make([]map[string]string, 0)
		for _, c := range cases {
			caseList = append(caseList, map[string]string{
				"id":     c.GetId(),
				"name":   c.Name,
				"status": c.State,
				"type":   c.Type,
			})
		}
		data, err := json.MarshalIndent(caseList, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal cases: %v", err)
		}
		return ResourceContents{
			URI:      uri,
			MimeType: "application/json",
			Contents: []ContentItem{{
				Type: "text",
				Text: string(data),
			}},
		}, nil

	case "redc://config":
		config := map[string]interface{}{
			"project":      redc.Project,
			"user":         redc.U,
			"template_dir": redc.TemplateDir,
			"redc_path":    redc.RedcPath,
		}
		data, err := json.MarshalIndent(config, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal config: %v", err)
		}
		return ResourceContents{
			URI:      uri,
			MimeType: "application/json",
			Contents: []ContentItem{{
				Type: "text",
				Text: string(data),
			}},
		}, nil

	default:
		return nil, fmt.Errorf("unknown resource URI: %s", uri)
	}
}

func (s *MCPServer) executeTool(name string, args map[string]interface{}) (ToolResult, error) {
	switch name {
	case "list_templates":
		return s.toolListTemplates()

	case "search_templates":
		query, ok := args["query"].(string)
		if !ok {
			return ToolResult{}, fmt.Errorf("missing or invalid 'query' parameter")
		}
		registryURL, _ := args["registry_url"].(string)
		return s.toolSearchTemplates(query, registryURL)

	case "pull_template":
		template, ok := args["template"].(string)
		if !ok {
			return ToolResult{}, fmt.Errorf("missing or invalid 'template' parameter")
		}
		registryURL, _ := args["registry_url"].(string)
		force, _ := args["force"].(bool)
		return s.toolPullTemplate(template, registryURL, force)

	case "list_cases":
		return s.toolListCases()

	case "plan_case":
		template, ok := args["template"].(string)
		if !ok {
			return ToolResult{}, fmt.Errorf("missing or invalid 'template' parameter")
		}
		caseName, _ := args["name"].(string)
		env, _ := args["env"].(map[string]interface{})
		return s.toolPlanCase(template, caseName, env)

	case "start_case":
		caseID, ok := args["case_id"].(string)
		if !ok {
			return ToolResult{}, fmt.Errorf("missing or invalid 'case_id' parameter")
		}
		return s.toolStartCase(caseID)

	case "stop_case":
		caseID, ok := args["case_id"].(string)
		if !ok {
			return ToolResult{}, fmt.Errorf("missing or invalid 'case_id' parameter")
		}
		return s.toolStopCase(caseID)

	case "kill_case":
		caseID, ok := args["case_id"].(string)
		if !ok {
			return ToolResult{}, fmt.Errorf("missing or invalid 'case_id' parameter")
		}
		return s.toolKillCase(caseID)

	case "get_case_status":
		caseID, ok := args["case_id"].(string)
		if !ok {
			return ToolResult{}, fmt.Errorf("missing or invalid 'case_id' parameter")
		}
		return s.toolGetCaseStatus(caseID)

	case "exec_command":
		caseID, ok := args["case_id"].(string)
		if !ok {
			return ToolResult{}, fmt.Errorf("missing or invalid 'case_id' parameter")
		}
		command, ok := args["command"].(string)
		if !ok {
			return ToolResult{}, fmt.Errorf("missing or invalid 'command' parameter")
		}
		return s.toolExecCommand(caseID, command)

	case "get_ssh_info":
		caseID, ok := args["case_id"].(string)
		if !ok {
			return ToolResult{}, fmt.Errorf("missing or invalid 'case_id' parameter")
		}
		return s.toolGetSSHInfo(caseID)

	case "upload_file":
		caseID, ok := args["case_id"].(string)
		if !ok {
			return ToolResult{}, fmt.Errorf("missing or invalid 'case_id' parameter")
		}
		localPath, ok := args["local_path"].(string)
		if !ok {
			return ToolResult{}, fmt.Errorf("missing or invalid 'local_path' parameter")
		}
		remotePath, ok := args["remote_path"].(string)
		if !ok {
			return ToolResult{}, fmt.Errorf("missing or invalid 'remote_path' parameter")
		}
		return s.toolUploadFile(caseID, localPath, remotePath)

	case "download_file":
		caseID, ok := args["case_id"].(string)
		if !ok {
			return ToolResult{}, fmt.Errorf("missing or invalid 'case_id' parameter")
		}
		remotePath, ok := args["remote_path"].(string)
		if !ok {
			return ToolResult{}, fmt.Errorf("missing or invalid 'remote_path' parameter")
		}
		localPath, ok := args["local_path"].(string)
		if !ok {
			return ToolResult{}, fmt.Errorf("missing or invalid 'local_path' parameter")
		}
		return s.toolDownloadFile(caseID, remotePath, localPath)

	case "get_template_info":
		templateName, ok := args["template_name"].(string)
		if !ok {
			return ToolResult{}, fmt.Errorf("missing or invalid 'template_name' parameter")
		}
		return s.toolGetTemplateInfo(templateName)

	case "delete_template":
		templateName, ok := args["template_name"].(string)
		if !ok {
			return ToolResult{}, fmt.Errorf("missing or invalid 'template_name' parameter")
		}
		return s.toolDeleteTemplate(templateName)

	case "get_case_outputs":
		caseID, ok := args["case_id"].(string)
		if !ok {
			return ToolResult{}, fmt.Errorf("missing or invalid 'case_id' parameter")
		}
		return s.toolGetCaseOutputs(caseID)

	case "get_config":
		return s.toolGetConfig()

	case "validate_config":
		provider, ok := args["provider"].(string)
		if !ok {
			return ToolResult{}, fmt.Errorf("missing or invalid 'provider' parameter")
		}
		region, _ := args["region"].(string)
		instanceType, _ := args["instance_type"].(string)
		return s.toolValidateConfig(provider, region, instanceType)

	// --- Compose tools ---
	case "compose_preview":
		file, _ := args["file"].(string)
		profiles, _ := args["profiles"].(string)
		return s.toolComposePreview(file, profiles)

	case "compose_up":
		file, _ := args["file"].(string)
		profiles, _ := args["profiles"].(string)
		return s.toolComposeUp(file, profiles)

	case "compose_down":
		file, _ := args["file"].(string)
		profiles, _ := args["profiles"].(string)
		return s.toolComposeDown(file, profiles)

	// --- Cost & Resource tools ---
	case "get_cost_estimate":
		template, ok := args["template"].(string)
		if !ok {
			return ToolResult{}, fmt.Errorf("missing or invalid 'template' parameter")
		}
		return s.toolGetCostEstimate(template)

	case "get_balances":
		providers, _ := args["providers"].(string)
		return s.toolGetBalances(providers)

	case "get_resource_summary":
		return s.toolGetResourceSummary()

	case "get_predicted_monthly_cost":
		return s.toolGetPredictedMonthlyCost()

	case "get_bills":
		providers, _ := args["providers"].(string)
		return s.toolGetBills(providers)

	case "get_total_runtime":
		return s.toolGetTotalRuntime()

	// --- Custom Deployment tools ---
	case "list_deployments":
		return s.toolListDeployments()

	case "start_deployment":
		id, ok := args["deployment_id"].(string)
		if !ok {
			return ToolResult{}, fmt.Errorf("missing or invalid 'deployment_id' parameter")
		}
		return s.toolStartDeployment(id)

	case "stop_deployment":
		id, ok := args["deployment_id"].(string)
		if !ok {
			return ToolResult{}, fmt.Errorf("missing or invalid 'deployment_id' parameter")
		}
		return s.toolStopDeployment(id)

	// --- Project & Profile tools ---
	case "list_projects":
		return s.toolListProjects()

	case "switch_project":
		name, ok := args["project_name"].(string)
		if !ok {
			return ToolResult{}, fmt.Errorf("missing or invalid 'project_name' parameter")
		}
		return s.toolSwitchProject(name)

	case "list_profiles":
		return s.toolListProfiles()

	case "get_active_profile":
		return s.toolGetActiveProfile()

	case "set_active_profile":
		profileID, ok := args["profile_id"].(string)
		if !ok {
			return ToolResult{}, fmt.Errorf("missing or invalid 'profile_id' parameter")
		}
		return s.toolSetActiveProfile(profileID)

	// --- Scheduler tools ---
	case "schedule_task":
		caseID, ok := args["case_id"].(string)
		if !ok {
			return ToolResult{}, fmt.Errorf("missing or invalid 'case_id' parameter")
		}
		caseName, _ := args["case_name"].(string)
		action, ok := args["action"].(string)
		if !ok {
			return ToolResult{}, fmt.Errorf("missing or invalid 'action' parameter")
		}
		scheduledAt, ok := args["scheduled_at"].(string)
		if !ok {
			return ToolResult{}, fmt.Errorf("missing or invalid 'scheduled_at' parameter")
		}
		return s.toolScheduleTask(caseID, caseName, action, scheduledAt)

	case "list_scheduled_tasks":
		return s.toolListScheduledTasks()

	case "cancel_scheduled_task":
		taskID, ok := args["task_id"].(string)
		if !ok {
			return ToolResult{}, fmt.Errorf("missing or invalid 'task_id' parameter")
		}
		return s.toolCancelScheduledTask(taskID)

	default:
		return ToolResult{}, fmt.Errorf("unknown tool: %s", name)
	}
}

// ExecuteTool is the public API for calling MCP tools directly (used by AI Agent)
func (s *MCPServer) ExecuteTool(name string, args map[string]interface{}) (ToolResult, error) {
	return s.executeTool(name, args)
}

// GetTools returns the list of available MCP tools (used by AI Agent to build tool schema)
func (s *MCPServer) GetTools() []Tool {
	return s.getTools()
}

// GetMCPServer returns the underlying MCPServer from the manager (used by AI Agent)
func (m *MCPServerManager) GetMCPServer() *MCPServer {
	return m.server
}

// Tool implementations
func (s *MCPServer) toolListTemplates() (ToolResult, error) {
	dirs, err := redc.ScanTemplateDirs(redc.TemplateDir, redc.MaxTfDepth)
	if err != nil {
		return ToolResult{}, err
	}

	output := "Available templates:\n"
	for _, dir := range dirs {
		output += fmt.Sprintf("- %s\n", dir)
	}

	return ToolResult{
		Content: []ContentItem{{
			Type: "text",
			Text: output,
		}},
	}, nil
}

func (s *MCPServer) toolSearchTemplates(query string, registryURL string) (ToolResult, error) {
	if strings.TrimSpace(query) == "" {
		return ToolResult{}, fmt.Errorf("query cannot be empty")
	}
	if strings.TrimSpace(registryURL) == "" {
		registryURL = "https://redc.wgpsec.org"
	}

	opts := redc.PullOptions{
		RegistryURL: registryURL,
		Timeout:     30 * time.Second,
	}

	results, err := redc.Search(context.Background(), query, opts)
	if err != nil {
		return ToolResult{}, fmt.Errorf("failed to search templates: %v", err)
	}

	if len(results) == 0 {
		return ToolResult{
			Content: []ContentItem{{
				Type: "text",
				Text: fmt.Sprintf("No templates found for query: '%s'\n\nTry searching with different keywords like:\n- Provider names: 'aliyun', 'tencent', 'huawei', 'aws'\n- Resource types: 'ecs', 'vpc', 'network', 'database'\n- Full paths: 'aliyun/ecs', 'tencent/cvm'", query),
			}},
		}, nil
	}

	output := fmt.Sprintf("Found %d template(s) for query '%s':\n\n", len(results), query)
	for i, result := range results {
		output += fmt.Sprintf("%d. %s\n", i+1, result.Key)
		output += fmt.Sprintf("   Version: %s\n", result.Version)
		output += fmt.Sprintf("   Provider: %s\n", result.Provider)
		if result.Author != "" {
			output += fmt.Sprintf("   Author: %s\n", result.Author)
		}
		if result.Description != "" {
			desc := result.Description
			if len(desc) > 100 {
				desc = desc[:100] + "..."
			}
			output += fmt.Sprintf("   Description: %s\n", desc)
		}
		output += "\n"
	}

	output += "To pull a template, use the 'pull_template' tool with the template name (e.g., 'aliyun/ecs').\n"

	return ToolResult{
		Content: []ContentItem{{
			Type: "text",
			Text: output,
		}},
	}, nil
}

func (s *MCPServer) toolPullTemplate(template string, registryURL string, force bool) (ToolResult, error) {
	if strings.TrimSpace(template) == "" {
		return ToolResult{}, fmt.Errorf("template cannot be empty")
	}
	if strings.TrimSpace(registryURL) == "" {
		registryURL = "https://redc.wgpsec.org"
	}

	opts := redc.PullOptions{
		RegistryURL: registryURL,
		Force:       force,
		Timeout:     120 * time.Second,
	}

	if err := redc.Pull(context.Background(), template, opts); err != nil {
		return ToolResult{}, fmt.Errorf("failed to pull template: %v", err)
	}

	output := fmt.Sprintf("Template pulled successfully:\n- Template: %s\n- Registry: %s\n", template, registryURL)
	if force {
		output += "- Force: true\n"
	}

	return ToolResult{
		Content: []ContentItem{{
			Type: "text",
			Text: output,
		}},
	}, nil
}

func (s *MCPServer) toolListCases() (ToolResult, error) {
	cases, err := redc.LoadProjectCases(s.project.ProjectName)
	if err != nil {
		return ToolResult{}, err
	}

	output := fmt.Sprintf("Cases in project '%s':\n", redc.Project)

	if len(cases) == 0 {
		output += "No cases found.\n"
	} else {
		for _, c := range cases {
			output += fmt.Sprintf("- ID: %s, Name: %s, Status: %s, Type: %s\n", c.GetId(), c.Name, c.State, c.Type)
		}
	}

	return ToolResult{
		Content: []ContentItem{{
			Type: "text",
			Text: output,
		}},
	}, nil
}

func (s *MCPServer) toolPlanCase(template string, name string, env map[string]interface{}) (ToolResult, error) {
	vars := make(map[string]string)
	if env != nil {
		for k, v := range env {
			vars[k] = fmt.Sprintf("%v", v)
		}
	}

	c, err := s.project.CaseCreate(template, redc.U, name, vars)
	if err != nil {
		return ToolResult{}, fmt.Errorf("failed to plan case: %v", err)
	}

	output := fmt.Sprintf("Case planned successfully (terraform plan completed):\n")
	output += fmt.Sprintf("- ID: %s\n", c.GetId())
	output += fmt.Sprintf("- Name: %s\n", c.Name)
	output += fmt.Sprintf("- Template: %s\n", template)
	output += fmt.Sprintf("\nThe case has been validated but not started yet.\n")
	output += fmt.Sprintf("Use 'start_case' with ID '%s' to actually create and start the infrastructure.\n", c.GetId())

	return ToolResult{
		Content: []ContentItem{{
			Type: "text",
			Text: output,
		}},
	}, nil
}

func (s *MCPServer) toolStartCase(caseID string) (ToolResult, error) {
	c, err := s.project.GetCase(caseID)
	if err != nil {
		return ToolResult{}, fmt.Errorf("case not found: %v", err)
	}

	if err := c.TfApply(); err != nil {
		return ToolResult{}, fmt.Errorf("failed to start case: %v", err)
	}

	output := fmt.Sprintf("Case '%s' (%s) started successfully.\n", c.Name, c.GetId())
	return ToolResult{
		Content: []ContentItem{{
			Type: "text",
			Text: output,
		}},
	}, nil
}

func (s *MCPServer) toolStopCase(caseID string) (ToolResult, error) {
	c, err := s.project.GetCase(caseID)
	if err != nil {
		return ToolResult{}, fmt.Errorf("case not found: %v", err)
	}

	if err := c.Stop(); err != nil {
		return ToolResult{}, fmt.Errorf("failed to stop case: %v", err)
	}

	output := fmt.Sprintf("Case '%s' (%s) stopped successfully.\n", c.Name, c.GetId())
	return ToolResult{
		Content: []ContentItem{{
			Type: "text",
			Text: output,
		}},
	}, nil
}

func (s *MCPServer) toolKillCase(caseID string) (ToolResult, error) {
	c, err := s.project.GetCase(caseID)
	if err != nil {
		return ToolResult{}, fmt.Errorf("case not found: %v", err)
	}

	if err := c.Kill(); err != nil {
		return ToolResult{}, fmt.Errorf("failed to kill case: %v", err)
	}

	output := fmt.Sprintf("Case '%s' (%s) killed successfully.\n", c.Name, c.GetId())
	return ToolResult{
		Content: []ContentItem{{
			Type: "text",
			Text: output,
		}},
	}, nil
}

func (s *MCPServer) toolGetCaseStatus(caseID string) (ToolResult, error) {
	c, err := s.project.GetCase(caseID)
	if err != nil {
		return ToolResult{}, fmt.Errorf("case not found: %v", err)
	}

	output := fmt.Sprintf("Case Status:\n")
	output += fmt.Sprintf("- ID: %s\n", c.GetId())
	output += fmt.Sprintf("- Name: %s\n", c.Name)
	output += fmt.Sprintf("- Status: %s\n", c.State)
	output += fmt.Sprintf("- Template: %s\n", c.Type)

	return ToolResult{
		Content: []ContentItem{{
			Type: "text",
			Text: output,
		}},
	}, nil
}

func (s *MCPServer) toolExecCommand(caseID string, command string) (ToolResult, error) {
	c, err := s.project.GetCase(caseID)
	if err != nil {
		return ToolResult{}, fmt.Errorf("case not found: %v", err)
	}

	sshConfig, err := c.GetSSHConfig()
	if err != nil {
		return ToolResult{}, fmt.Errorf("failed to get SSH config: %v", err)
	}

	client, err := sshutil.NewClient(sshConfig)
	if err != nil {
		return ToolResult{}, fmt.Errorf("failed to create SSH client: %v", err)
	}
	defer client.Close()

	var outputBuf strings.Builder
	session, err := client.Client.NewSession()
	if err != nil {
		return ToolResult{}, fmt.Errorf("failed to create SSH session: %v", err)
	}
	defer session.Close()

	session.Stdout = &outputBuf
	session.Stderr = &outputBuf

	if err := session.Run(command); err != nil {
		return ToolResult{}, fmt.Errorf("failed to execute command: %v\nOutput: %s", err, outputBuf.String())
	}

	output := fmt.Sprintf("Command executed on case '%s' (%s):\n", c.Name, c.GetId())
	output += fmt.Sprintf("\nOutput:\n%s", outputBuf.String())

	return ToolResult{
		Content: []ContentItem{{
			Type: "text",
			Text: output,
		}},
	}, nil
}

func (s *MCPServer) toolGetSSHInfo(caseID string) (ToolResult, error) {
	c, err := s.project.GetCase(caseID)
	if err != nil {
		return ToolResult{}, fmt.Errorf("case not found: %v", err)
	}

	sshConfig, err := c.GetSSHConfig()
	if err != nil {
		return ToolResult{}, fmt.Errorf("failed to get SSH config: %v", err)
	}

	output := fmt.Sprintf("SSH Connection Info for Case '%s':\n\n", c.Name)
	output += fmt.Sprintf("| Field | Value |\n")
	output += fmt.Sprintf("|-------|-------|\n")
	output += fmt.Sprintf("| IP Address | %s |\n", sshConfig.Host)
	output += fmt.Sprintf("| Port | %d |\n", sshConfig.Port)
	output += fmt.Sprintf("| Username | %s |\n", sshConfig.User)
	output += fmt.Sprintf("| Password | %s |\n", sshConfig.Password)
	output += fmt.Sprintf("| Key Path | %s |\n", sshConfig.KeyPath)

	return ToolResult{
		Content: []ContentItem{{
			Type: "text",
			Text: output,
		}},
	}, nil
}

func (s *MCPServer) toolUploadFile(caseID string, localPath string, remotePath string) (ToolResult, error) {
	c, err := s.project.GetCase(caseID)
	if err != nil {
		return ToolResult{}, fmt.Errorf("case not found: %v", err)
	}

	sshConfig, err := c.GetSSHConfig()
	if err != nil {
		return ToolResult{}, fmt.Errorf("failed to get SSH config: %v", err)
	}

	client, err := sshutil.NewClient(sshConfig)
	if err != nil {
		return ToolResult{}, fmt.Errorf("failed to create SSH client: %v", err)
	}
	defer client.Close()

	err = client.Upload(localPath, remotePath)
	if err != nil {
		return ToolResult{}, fmt.Errorf("failed to upload file: %v", err)
	}

	output := fmt.Sprintf("File uploaded successfully!\n\n")
	output += fmt.Sprintf("- Case: %s\n", c.Name)
	output += fmt.Sprintf("- Local Path: %s\n", localPath)
	output += fmt.Sprintf("- Remote Path: %s\n", remotePath)

	return ToolResult{
		Content: []ContentItem{{
			Type: "text",
			Text: output,
		}},
	}, nil
}

func (s *MCPServer) toolDownloadFile(caseID string, remotePath string, localPath string) (ToolResult, error) {
	c, err := s.project.GetCase(caseID)
	if err != nil {
		return ToolResult{}, fmt.Errorf("case not found: %v", err)
	}

	sshConfig, err := c.GetSSHConfig()
	if err != nil {
		return ToolResult{}, fmt.Errorf("failed to get SSH config: %v", err)
	}

	client, err := sshutil.NewClient(sshConfig)
	if err != nil {
		return ToolResult{}, fmt.Errorf("failed to create SSH client: %v", err)
	}
	defer client.Close()

	err = client.Download(remotePath, localPath)
	if err != nil {
		return ToolResult{}, fmt.Errorf("failed to download file: %v", err)
	}

	output := fmt.Sprintf("File downloaded successfully!\n\n")
	output += fmt.Sprintf("- Case: %s\n", c.Name)
	output += fmt.Sprintf("- Remote Path: %s\n", remotePath)
	output += fmt.Sprintf("- Local Path: %s\n", localPath)

	return ToolResult{
		Content: []ContentItem{{
			Type: "text",
			Text: output,
		}},
	}, nil
}

func (s *MCPServer) toolGetTemplateInfo(templateName string) (ToolResult, error) {
	_, err := redc.GetTemplatePath(templateName)
	if err != nil {
		return ToolResult{}, fmt.Errorf("template not found: %v", err)
	}

	tm := redc.NewTemplateManager()
	templates, err := redc.ListLocalTemplates()
	if err != nil {
		return ToolResult{}, fmt.Errorf("failed to list templates: %v", err)
	}

	var baseTemplate *redc.RedcTmpl
	for _, t := range templates {
		if t.Name == templateName {
			baseTemplate = t
			break
		}
	}
	if baseTemplate == nil {
		return ToolResult{}, fmt.Errorf("template '%s' not found in local templates", templateName)
	}

	templateVariables, err := tm.GetTemplateVariables(templateName)
	if err != nil {
		templateVariables = []redc.TemplateVariable{}
	}

	output := fmt.Sprintf("# Template: %s\n\n", templateName)
	output += fmt.Sprintf("| Field | Value |\n")
	output += fmt.Sprintf("|-------|-------|\n")
	output += fmt.Sprintf("| Name | %s |\n", baseTemplate.Name)
	output += fmt.Sprintf("| Description | %s |\n", baseTemplate.Description)
	output += fmt.Sprintf("| User | %s |\n", baseTemplate.User)
	output += fmt.Sprintf("| Version | %s |\n", baseTemplate.Version)
	output += fmt.Sprintf("| Template Type | %s |\n", baseTemplate.TemplateType)

	if len(templateVariables) > 0 {
		output += fmt.Sprintf("\n## Variables\n\n")
		output += fmt.Sprintf("| Name | Type | Required | Default | Description |\n")
		output += fmt.Sprintf("|------|------|----------|---------|-------------|\n")
		for _, v := range templateVariables {
			output += fmt.Sprintf("| %s | %s | %v | %s | %s |\n", v.Name, v.Type, v.Required, v.DefaultValue, v.Description)
		}
	}

	return ToolResult{
		Content: []ContentItem{{
			Type: "text",
			Text: output,
		}},
	}, nil
}

func (s *MCPServer) toolDeleteTemplate(templateName string) (ToolResult, error) {
	err := redc.RemoveTemplate(templateName)
	if err != nil {
		return ToolResult{}, fmt.Errorf("failed to delete template: %v", err)
	}

	output := fmt.Sprintf("Template '%s' deleted successfully!\n", templateName)

	return ToolResult{
		Content: []ContentItem{{
			Type: "text",
			Text: output,
		}},
	}, nil
}

func (s *MCPServer) toolGetCaseOutputs(caseID string) (ToolResult, error) {
	c, err := s.project.GetCase(caseID)
	if err != nil {
		return ToolResult{}, fmt.Errorf("case not found: %v", err)
	}

	if c.State != "running" {
		return ToolResult{}, fmt.Errorf("case is not running, current state: %s", c.State)
	}

	outputs, err := c.TfOutput()
	if err != nil {
		return ToolResult{}, fmt.Errorf("failed to get outputs: %v", err)
	}

	if len(outputs) == 0 {
		return ToolResult{}, fmt.Errorf("no outputs found for this case")
	}

	result := fmt.Sprintf("# Terraform Outputs for Case '%s'\n\n", c.Name)
	result += fmt.Sprintf("| Key | Value |\n")
	result += fmt.Sprintf("|-----|-------|\n")
	for k, v := range outputs {
		result += fmt.Sprintf("| %s | %s |\n", k, v)
	}

	return ToolResult{
		Content: []ContentItem{{
			Type: "text",
			Text: result,
		}},
	}, nil
}

func (s *MCPServer) toolGetConfig() (ToolResult, error) {
	result := fmt.Sprintf("# Redc Configuration\n\n")
	result += fmt.Sprintf("| Field | Value |\n")
	result += fmt.Sprintf("|-------|-------|\n")
	result += fmt.Sprintf("| Redc Path | %s |\n", redc.RedcPath)
	result += fmt.Sprintf("| Project Path | %s |\n", redc.ProjectPath)

	httpProxy := os.Getenv("HTTP_PROXY")
	httpsProxy := os.Getenv("HTTPS_PROXY")
	socks5Proxy := os.Getenv("ALL_PROXY")
	noProxy := os.Getenv("NO_PROXY")

	if httpProxy != "" {
		result += fmt.Sprintf("| HTTP Proxy | %s |\n", httpProxy)
	}
	if httpsProxy != "" {
		result += fmt.Sprintf("| HTTPS Proxy | %s |\n", httpsProxy)
	}
	if socks5Proxy != "" {
		result += fmt.Sprintf("| SOCKS5 Proxy | %s |\n", socks5Proxy)
	}
	if noProxy != "" {
		result += fmt.Sprintf("| No Proxy | %s |\n", noProxy)
	}

	return ToolResult{
		Content: []ContentItem{{
			Type: "text",
			Text: result,
		}},
	}, nil
}

func (s *MCPServer) toolValidateConfig(provider string, region string, instanceType string) (ToolResult, error) {
	validator := redc.NewConfigValidator()

	result := fmt.Sprintf("# Config Validation Results\n\n")

	err := validator.ValidateProvider(provider)
	if err != nil {
		result += fmt.Sprintf("| Provider | ❌ Invalid |\n")
		result += fmt.Sprintf("| Error | %s |\n", err.Error())
	} else {
		result += fmt.Sprintf("| Provider | ✅ Valid (%s) |\n", provider)
	}

	if region != "" {
		err = validator.ValidateRegion(provider, region)
		if err != nil {
			result += fmt.Sprintf("| Region | ❌ Invalid |\n")
			result += fmt.Sprintf("| Error | %s |\n", err.Error())
		} else {
			result += fmt.Sprintf("| Region | ✅ Valid (%s) |\n", region)
		}
	}

	if instanceType != "" && region != "" {
		err = validator.ValidateInstanceType(provider, region, instanceType)
		if err != nil {
			result += fmt.Sprintf("| Instance Type | ❌ Invalid |\n")
			result += fmt.Sprintf("| Error | %s |\n", err.Error())
		} else {
			result += fmt.Sprintf("| Instance Type | ✅ Valid (%s) |\n", instanceType)
		}
	}

	return ToolResult{
		Content: []ContentItem{{
			Type: "text",
			Text: result,
		}},
	}, nil
}

// Transport mode
type TransportMode string

const (
	TransportSTDIO TransportMode = "stdio"
	TransportSSE   TransportMode = "sse"
)

// MCPServerManager manages the MCP server lifecycle
type MCPServerManager struct {
	server     *MCPServer
	mode       TransportMode
	address    string
	httpServer *http.Server
	cancel     context.CancelFunc
	running    bool
	mu         sync.Mutex
	logWriter  LogCallback
}

// NewMCPServerManager creates a new server manager
func NewMCPServerManager(project *redc.RedcProject, app AppBridge) *MCPServerManager {
	return &MCPServerManager{
		server: NewMCPServer(project, app),
	}
}

// SetLogCallback sets a callback for log messages
func (m *MCPServerManager) SetLogCallback(callback LogCallback) {
	m.logWriter = callback
	m.server.SetLogCallback(callback)
}

func (m *MCPServerManager) log(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if m.logWriter != nil {
		m.logWriter(msg)
	}
	gologger.Info().Msg(msg)
}

// Start starts the MCP server with the specified transport mode
func (m *MCPServerManager) Start(mode TransportMode, address string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return fmt.Errorf("MCP server is already running")
	}

	m.mode = mode
	m.address = address

	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel

	switch mode {
	case TransportSTDIO:
		go m.runStdioServer(ctx)
	case TransportSSE:
		if err := m.runSSEServer(ctx, address); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown transport mode: %s", mode)
	}

	m.running = true
	return nil
}

// Stop stops the MCP server
func (m *MCPServerManager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return fmt.Errorf("MCP server is not running")
	}

	if m.cancel != nil {
		m.cancel()
	}

	if m.httpServer != nil {
		if err := m.httpServer.Shutdown(context.Background()); err != nil {
			return fmt.Errorf("failed to shutdown HTTP server: %v", err)
		}
		m.httpServer = nil
	}

	m.running = false
	m.log("MCP server stopped")
	return nil
}

// IsRunning returns whether the server is running
func (m *MCPServerManager) IsRunning() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.running
}

// GetStatus returns the current server status
func (m *MCPServerManager) GetStatus() map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	return map[string]interface{}{
		"running":         m.running,
		"mode":            string(m.mode),
		"address":         m.address,
		"protocolVersion": MCPVersion,
	}
}

// STDIO transport
func (m *MCPServerManager) runStdioServer(ctx context.Context) {
	m.log("🚀 Starting MCP STDIO Server...")
	m.log("   Protocol version: %s", MCPVersion)
	m.log("   Project: %s", redc.Project)

	decoder := json.NewDecoder(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			var req MCPRequest
			if err := decoder.Decode(&req); err != nil {
				if err == io.EOF {
					return
				}
				gologger.Error().Msgf("Failed to decode request: %v", err)
				continue
			}

			resp := m.server.HandleRequest(&req)
			if resp == nil {
				// Notification — no response to send
				continue
			}
			if err := encoder.Encode(resp); err != nil {
				gologger.Error().Msgf("Failed to encode response: %v", err)
			}
		}
	}
}

// SSE transport (with Streamable HTTP support per 2025-11-25 spec)
func (m *MCPServerManager) runSSEServer(ctx context.Context, addr string) error {
	// Normalize address
	if strings.HasPrefix(addr, ":") || !strings.Contains(addr, ":") {
		if strings.HasPrefix(addr, ":") {
			addr = "localhost" + addr
		} else {
			addr = "localhost:" + addr
		}
	}

	mux := http.NewServeMux()

	// CORS preflight handler
	setCORSHeaders := func(w http.ResponseWriter) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept, MCP-Session-Id")
		w.Header().Set("Access-Control-Expose-Headers", "MCP-Session-Id")
	}

	// Streamable HTTP endpoint (2025-11-25 spec)
	mux.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
		setCORSHeaders(w)
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if r.Method == "GET" {
			// SSE stream for server-initiated messages
			flusher, ok := w.(http.Flusher)
			if !ok {
				http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")
			fmt.Fprintf(w, ": connected\n\n")
			flusher.Flush()

			// Keep connection open until client disconnects or context cancelled
			select {
			case <-r.Context().Done():
			case <-ctx.Done():
			}
			return
		}

		if r.Method == "POST" {
			var req MCPRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(MCPResponse{
					JSONRPC: "2.0",
					Error: &MCPError{
						Code:    -32700,
						Message: "Parse error",
						Data:    err.Error(),
					},
				})
				return
			}

			resp := m.server.HandleRequest(&req)
			if resp == nil {
				// Notification — accepted, no body
				w.WriteHeader(http.StatusAccepted)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}

		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	})

	// Legacy SSE endpoint (kept for backward compatibility)
	mux.HandleFunc("/sse", func(w http.ResponseWriter, r *http.Request) {
		setCORSHeaders(w)
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		if r.Method == "POST" {
			var req MCPRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			resp := m.server.HandleRequest(&req)
			if resp == nil {
				w.WriteHeader(http.StatusAccepted)
				return
			}
			data, _ := json.Marshal(resp)

			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
			return
		}

		fmt.Fprintf(w, "data: {\"type\":\"connected\"}\n\n")
		flusher.Flush()
	})

	// Legacy message endpoint (kept for backward compatibility)
	mux.HandleFunc("/message", func(w http.ResponseWriter, r *http.Request) {
		setCORSHeaders(w)
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req MCPRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		resp := m.server.HandleRequest(&req)
		if resp == nil {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		setCORSHeaders(w)
		fmt.Fprintf(w, "redc MCP Server v%s (protocol %s)\n", redc.Version, MCPVersion)
		fmt.Fprintf(w, "Endpoints:\n")
		fmt.Fprintf(w, "  POST /mcp     - Streamable HTTP endpoint (2025-11-25)\n")
		fmt.Fprintf(w, "  POST /message - JSON-RPC messages (legacy)\n")
		fmt.Fprintf(w, "  GET  /sse     - SSE endpoint (legacy)\n")
	})

	m.httpServer = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		m.log("🚀 MCP Server listening on %s", addr)
		m.log("   Streamable HTTP: http://%s/mcp (2025-11-25)", addr)
		m.log("   Legacy POST:     http://%s/message", addr)
		m.log("   Legacy SSE:      http://%s/sse", addr)
		m.log("   Protocol version: %s", MCPVersion)

		if err := m.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			gologger.Error().Msgf("Failed to start server: %v", err)
			m.mu.Lock()
			m.running = false
			m.mu.Unlock()
		}
	}()

	return nil
}
