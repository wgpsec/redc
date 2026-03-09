package mcp

import (
	"encoding/json"
	"fmt"
)

func deploymentToolSchemas() []Tool {
	return []Tool{
		{
			Name:        "list_deployments",
			Description: "List all custom deployments in the current project",
			InputSchema: ToolSchema{
				Type:       "object",
				Properties: map[string]Property{},
			},
		},
		{
			Name:        "start_deployment",
			Description: "Start a custom deployment by ID",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]Property{
					"deployment_id": {
						Type:        "string",
						Description: "Custom deployment ID to start",
					},
				},
				Required: []string{"deployment_id"},
			},
		},
		{
			Name:        "stop_deployment",
			Description: "Stop a custom deployment by ID",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]Property{
					"deployment_id": {
						Type:        "string",
						Description: "Custom deployment ID to stop",
					},
				},
				Required: []string{"deployment_id"},
			},
		},
	}
}

func (s *MCPServer) toolListDeployments() (ToolResult, error) {
	if s.app == nil {
		return ToolResult{}, fmt.Errorf("deployment tools require GUI mode (AppBridge not available)")
	}
	result, err := s.app.MCPListCustomDeployments()
	if err != nil {
		return ToolResult{}, err
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	return ToolResult{
		Content: []ContentItem{{Type: "text", Text: string(data)}},
	}, nil
}

func (s *MCPServer) toolStartDeployment(id string) (ToolResult, error) {
	if s.app == nil {
		return ToolResult{}, fmt.Errorf("deployment tools require GUI mode (AppBridge not available)")
	}
	if err := s.app.MCPStartCustomDeployment(id); err != nil {
		return ToolResult{}, err
	}
	return ToolResult{
		Content: []ContentItem{{Type: "text", Text: fmt.Sprintf("Custom deployment %s started", id)}},
	}, nil
}

func (s *MCPServer) toolStopDeployment(id string) (ToolResult, error) {
	if s.app == nil {
		return ToolResult{}, fmt.Errorf("deployment tools require GUI mode (AppBridge not available)")
	}
	if err := s.app.MCPStopCustomDeployment(id); err != nil {
		return ToolResult{}, err
	}
	return ToolResult{
		Content: []ContentItem{{Type: "text", Text: fmt.Sprintf("Custom deployment %s stopped", id)}},
	}, nil
}
