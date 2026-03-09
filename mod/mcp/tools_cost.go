package mcp

import (
	"encoding/json"
	"fmt"
	"strings"
)

func costToolSchemas() []Tool {
	return []Tool{
		{
			Name:        "get_cost_estimate",
			Description: "Estimate deployment cost for a template (hourly and monthly cost breakdown by resource)",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]Property{
					"template": {
						Type:        "string",
						Description: "Template name (e.g., 'aliyun/ecs')",
					},
				},
				Required: []string{"template"},
			},
		},
		{
			Name:        "get_balances",
			Description: "Query cloud account balances for configured providers",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]Property{
					"providers": {
						Type:        "string",
						Description: "Comma-separated provider names (e.g., 'aliyun,aws'). Empty = all providers",
					},
				},
			},
		},
		{
			Name:        "get_resource_summary",
			Description: "Get a summary of cloud resources across all configured providers (instance counts, running status, etc.)",
			InputSchema: ToolSchema{
				Type:       "object",
				Properties: map[string]Property{},
			},
		},
		{
			Name:        "get_predicted_monthly_cost",
			Description: "Get predicted total monthly cost based on currently running resources",
			InputSchema: ToolSchema{
				Type:       "object",
				Properties: map[string]Property{},
			},
		},
		{
			Name:        "get_bills",
			Description: "Get cloud billing information for configured providers",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]Property{
					"providers": {
						Type:        "string",
						Description: "Comma-separated provider names (e.g., 'aliyun,aws'). Empty = all providers",
					},
				},
			},
		},
		{
			Name:        "get_total_runtime",
			Description: "Get total runtime of all running cases",
			InputSchema: ToolSchema{
				Type:       "object",
				Properties: map[string]Property{},
			},
		},
	}
}

func parseProviders(raw string) []string {
	if raw == "" {
		return nil
	}
	var result []string
	for _, p := range strings.Split(raw, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func (s *MCPServer) toolGetCostEstimate(template string) (ToolResult, error) {
	if s.app == nil {
		return ToolResult{}, fmt.Errorf("cost tools require GUI mode (AppBridge not available)")
	}
	result, err := s.app.MCPGetCostEstimate(template, nil)
	if err != nil {
		return ToolResult{}, err
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	return ToolResult{
		Content: []ContentItem{{Type: "text", Text: string(data)}},
	}, nil
}

func (s *MCPServer) toolGetBalances(providers string) (ToolResult, error) {
	if s.app == nil {
		return ToolResult{}, fmt.Errorf("cost tools require GUI mode (AppBridge not available)")
	}
	result, err := s.app.MCPGetBalances(parseProviders(providers))
	if err != nil {
		return ToolResult{}, err
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	return ToolResult{
		Content: []ContentItem{{Type: "text", Text: string(data)}},
	}, nil
}

func (s *MCPServer) toolGetResourceSummary() (ToolResult, error) {
	if s.app == nil {
		return ToolResult{}, fmt.Errorf("cost tools require GUI mode (AppBridge not available)")
	}
	result, err := s.app.MCPGetResourceSummary()
	if err != nil {
		return ToolResult{}, err
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	return ToolResult{
		Content: []ContentItem{{Type: "text", Text: string(data)}},
	}, nil
}

func (s *MCPServer) toolGetPredictedMonthlyCost() (ToolResult, error) {
	if s.app == nil {
		return ToolResult{}, fmt.Errorf("cost tools require GUI mode (AppBridge not available)")
	}
	result, err := s.app.MCPGetPredictedMonthlyCost()
	if err != nil {
		return ToolResult{}, err
	}
	return ToolResult{
		Content: []ContentItem{{Type: "text", Text: result}},
	}, nil
}

func (s *MCPServer) toolGetBills(providers string) (ToolResult, error) {
	if s.app == nil {
		return ToolResult{}, fmt.Errorf("cost tools require GUI mode (AppBridge not available)")
	}
	result, err := s.app.MCPGetBills(parseProviders(providers))
	if err != nil {
		return ToolResult{}, err
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	return ToolResult{
		Content: []ContentItem{{Type: "text", Text: string(data)}},
	}, nil
}

func (s *MCPServer) toolGetTotalRuntime() (ToolResult, error) {
	if s.app == nil {
		return ToolResult{}, fmt.Errorf("cost tools require GUI mode (AppBridge not available)")
	}
	result, err := s.app.MCPGetTotalRuntime()
	if err != nil {
		return ToolResult{}, err
	}
	return ToolResult{
		Content: []ContentItem{{Type: "text", Text: result}},
	}, nil
}
