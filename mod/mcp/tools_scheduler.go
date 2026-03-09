package mcp

import (
	"encoding/json"
	"fmt"
	"time"
)

func schedulerToolSchemas() []Tool {
	return []Tool{
		{
			Name:        "schedule_task",
			Description: "Schedule a future task for a case (start or stop at a specific time)",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]Property{
					"case_id": {
						Type:        "string",
						Description: "Case ID to schedule task for",
					},
					"case_name": {
						Type:        "string",
						Description: "Case name (for display)",
					},
					"action": {
						Type:        "string",
						Description: "Action to perform",
						Enum:        []string{"start", "stop", "kill"},
					},
					"scheduled_at": {
						Type:        "string",
						Description: "Scheduled time in RFC3339 format (e.g., '2025-01-15T10:30:00Z')",
					},
				},
				Required: []string{"case_id", "action", "scheduled_at"},
			},
		},
		{
			Name:        "list_scheduled_tasks",
			Description: "List all pending scheduled tasks",
			InputSchema: ToolSchema{
				Type:       "object",
				Properties: map[string]Property{},
			},
		},
		{
			Name:        "cancel_scheduled_task",
			Description: "Cancel a pending scheduled task",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]Property{
					"task_id": {
						Type:        "string",
						Description: "Task ID to cancel",
					},
				},
				Required: []string{"task_id"},
			},
		},
	}
}

func (s *MCPServer) toolScheduleTask(caseID string, caseName string, action string, scheduledAtStr string) (ToolResult, error) {
	if s.app == nil {
		return ToolResult{}, fmt.Errorf("scheduler tools require GUI mode (AppBridge not available)")
	}
	scheduledAt, err := time.Parse(time.RFC3339, scheduledAtStr)
	if err != nil {
		return ToolResult{}, fmt.Errorf("invalid scheduled_at format (expected RFC3339): %v", err)
	}
	result, err := s.app.MCPScheduleTask(caseID, caseName, action, scheduledAt)
	if err != nil {
		return ToolResult{}, err
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	return ToolResult{
		Content: []ContentItem{{Type: "text", Text: string(data)}},
	}, nil
}

func (s *MCPServer) toolListScheduledTasks() (ToolResult, error) {
	if s.app == nil {
		return ToolResult{}, fmt.Errorf("scheduler tools require GUI mode (AppBridge not available)")
	}
	result := s.app.MCPListScheduledTasks()
	data, _ := json.MarshalIndent(result, "", "  ")
	return ToolResult{
		Content: []ContentItem{{Type: "text", Text: string(data)}},
	}, nil
}

func (s *MCPServer) toolCancelScheduledTask(taskID string) (ToolResult, error) {
	if s.app == nil {
		return ToolResult{}, fmt.Errorf("scheduler tools require GUI mode (AppBridge not available)")
	}
	if err := s.app.MCPCancelScheduledTask(taskID); err != nil {
		return ToolResult{}, err
	}
	return ToolResult{
		Content: []ContentItem{{Type: "text", Text: fmt.Sprintf("Scheduled task %s cancelled", taskID)}},
	}, nil
}
