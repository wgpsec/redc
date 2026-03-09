package mcp

import (
	"encoding/json"
	"fmt"
)

func projectToolSchemas() []Tool {
	return []Tool{
		{
			Name:        "list_projects",
			Description: "List all redc projects",
			InputSchema: ToolSchema{
				Type:       "object",
				Properties: map[string]Property{},
			},
		},
		{
			Name:        "switch_project",
			Description: "Switch to a different redc project",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]Property{
					"project_name": {
						Type:        "string",
						Description: "Project name to switch to",
					},
				},
				Required: []string{"project_name"},
			},
		},
		{
			Name:        "list_profiles",
			Description: "List all cloud provider profiles (credential sets)",
			InputSchema: ToolSchema{
				Type:       "object",
				Properties: map[string]Property{},
			},
		},
		{
			Name:        "get_active_profile",
			Description: "Get the currently active cloud provider profile",
			InputSchema: ToolSchema{
				Type:       "object",
				Properties: map[string]Property{},
			},
		},
		{
			Name:        "set_active_profile",
			Description: "Switch the active cloud provider profile",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]Property{
					"profile_id": {
						Type:        "string",
						Description: "Profile ID to activate",
					},
				},
				Required: []string{"profile_id"},
			},
		},
	}
}

func (s *MCPServer) toolListProjects() (ToolResult, error) {
	if s.app == nil {
		return ToolResult{}, fmt.Errorf("project tools require GUI mode (AppBridge not available)")
	}
	result, err := s.app.MCPListProjects()
	if err != nil {
		return ToolResult{}, err
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	return ToolResult{
		Content: []ContentItem{{Type: "text", Text: string(data)}},
	}, nil
}

func (s *MCPServer) toolSwitchProject(name string) (ToolResult, error) {
	if s.app == nil {
		return ToolResult{}, fmt.Errorf("project tools require GUI mode (AppBridge not available)")
	}
	if err := s.app.MCPSwitchProject(name); err != nil {
		return ToolResult{}, err
	}
	return ToolResult{
		Content: []ContentItem{{Type: "text", Text: fmt.Sprintf("Switched to project: %s", name)}},
	}, nil
}

func (s *MCPServer) toolListProfiles() (ToolResult, error) {
	if s.app == nil {
		return ToolResult{}, fmt.Errorf("profile tools require GUI mode (AppBridge not available)")
	}
	result, err := s.app.MCPListProfiles()
	if err != nil {
		return ToolResult{}, err
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	return ToolResult{
		Content: []ContentItem{{Type: "text", Text: string(data)}},
	}, nil
}

func (s *MCPServer) toolGetActiveProfile() (ToolResult, error) {
	if s.app == nil {
		return ToolResult{}, fmt.Errorf("profile tools require GUI mode (AppBridge not available)")
	}
	result, err := s.app.MCPGetActiveProfile()
	if err != nil {
		return ToolResult{}, err
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	return ToolResult{
		Content: []ContentItem{{Type: "text", Text: string(data)}},
	}, nil
}

func (s *MCPServer) toolSetActiveProfile(profileID string) (ToolResult, error) {
	if s.app == nil {
		return ToolResult{}, fmt.Errorf("profile tools require GUI mode (AppBridge not available)")
	}
	result, err := s.app.MCPSetActiveProfile(profileID)
	if err != nil {
		return ToolResult{}, err
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	return ToolResult{
		Content: []ContentItem{{Type: "text", Text: string(data)}},
	}, nil
}
