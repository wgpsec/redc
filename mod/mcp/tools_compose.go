package mcp

import (
	"encoding/json"
	"fmt"
)

func composeToolSchemas() []Tool {
	return []Tool{
		{
			Name:        "save_compose_file",
			Description: "Save a redc-compose YAML file to disk. Use this to create multi-cloud orchestration deployments. The file defines services (cloud instances), their dependencies, and post-deploy setup tasks.",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]Property{
					"filename": {
						Type:        "string",
						Description: "Compose file name (default: redc-compose.yaml). Will be saved under the RedC data directory.",
					},
					"content": {
						Type:        "string",
						Description: "The full YAML content of the compose file",
					},
				},
				Required: []string{"content"},
			},
		},
		{
			Name:        "compose_preview",
			Description: "Preview a redc-compose deployment: list services, dependencies, providers, and replicas without actually deploying",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]Property{
					"file": {
						Type:        "string",
						Description: "Compose file path (default: redc-compose.yaml)",
					},
					"profiles": {
						Type:        "string",
						Description: "Comma-separated profiles to activate (e.g., 'prod,attack')",
					},
				},
			},
		},
		{
			Name:        "compose_up",
			Description: "Start a redc-compose deployment (deploys all services in dependency order)",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]Property{
					"file": {
						Type:        "string",
						Description: "Compose file path (default: redc-compose.yaml)",
					},
					"profiles": {
						Type:        "string",
						Description: "Comma-separated profiles to activate (e.g., 'prod,attack')",
					},
				},
			},
		},
		{
			Name:        "compose_down",
			Description: "Destroy a redc-compose deployment (destroys all services in reverse dependency order)",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]Property{
					"file": {
						Type:        "string",
						Description: "Compose file path (default: redc-compose.yaml)",
					},
					"profiles": {
						Type:        "string",
						Description: "Comma-separated profiles to activate (e.g., 'prod,attack')",
					},
				},
			},
		},
	}
}

func parseProfiles(raw string) []string {
	if raw == "" {
		return nil
	}
	var result []string
	for _, p := range splitCSV(raw) {
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func splitCSV(s string) []string {
	var parts []string
	for _, p := range []byte(s) {
		if p == ',' {
			parts = append(parts, "")
		} else {
			if len(parts) == 0 {
				parts = append(parts, "")
			}
			parts[len(parts)-1] += string(p)
		}
	}
	return parts
}

func (s *MCPServer) toolSaveComposeFile(filename string, content string) (ToolResult, error) {
	if s.app == nil {
		return ToolResult{}, fmt.Errorf("save_compose_file requires GUI mode (AppBridge not available)")
	}
	savedPath, err := s.app.MCPSaveComposeFile(filename, content)
	if err != nil {
		return ToolResult{}, fmt.Errorf("failed to save compose file: %v", err)
	}
	output := fmt.Sprintf("Compose file saved: %s\n\nYou can now use compose_preview to verify, then compose_up to deploy.", savedPath)
	return ToolResult{
		Content: []ContentItem{{Type: "text", Text: output}},
	}, nil
}

func (s *MCPServer) toolComposePreview(file string, profiles string) (ToolResult, error) {
	if s.app == nil {
		return ToolResult{}, fmt.Errorf("compose tools require GUI mode (AppBridge not available)")
	}
	result, err := s.app.MCPComposePreview(file, parseProfiles(profiles))
	if err != nil {
		return ToolResult{}, err
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	return ToolResult{
		Content: []ContentItem{{Type: "text", Text: string(data)}},
	}, nil
}

func (s *MCPServer) toolComposeUp(file string, profiles string) (ToolResult, error) {
	if s.app == nil {
		return ToolResult{}, fmt.Errorf("compose tools require GUI mode (AppBridge not available)")
	}
	if err := s.app.MCPComposeUp(file, parseProfiles(profiles)); err != nil {
		return ToolResult{}, err
	}
	return ToolResult{
		Content: []ContentItem{{Type: "text", Text: fmt.Sprintf("Compose deployment started (file: %s)", file)}},
	}, nil
}

func (s *MCPServer) toolComposeDown(file string, profiles string) (ToolResult, error) {
	if s.app == nil {
		return ToolResult{}, fmt.Errorf("compose tools require GUI mode (AppBridge not available)")
	}
	if err := s.app.MCPComposeDown(file, parseProfiles(profiles)); err != nil {
		return ToolResult{}, err
	}
	return ToolResult{
		Content: []ContentItem{{Type: "text", Text: fmt.Sprintf("Compose deployment destroyed (file: %s)", file)}},
	}, nil
}
