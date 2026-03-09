package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"red-cloud/i18n"
	redc "red-cloud/mod"
	"red-cloud/mod/ai"
	"red-cloud/mod/mcp"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// AIChatMessage represents a single message in the AI chat conversation
type AIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AIChatStream handles multi-turn AI chat with streaming responses
func (a *App) AIChatStream(conversationId, mode string, messages []AIChatMessage) error {
	// Validate AI config
	profile, err := redc.GetActiveProfile()
	if err != nil || profile.AIConfig == nil {
		return fmt.Errorf("%s", i18n.T("app_ai_not_configured"))
	}

	aiConfig := profile.AIConfig
	if aiConfig.APIKey == "" || aiConfig.BaseURL == "" || aiConfig.Model == "" {
		return fmt.Errorf("%s", i18n.T("app_ai_config_incomplete"))
	}

	uiLang := a.GetLanguage()
	langPrompt := "请用中文回复"
	if uiLang == "en" {
		langPrompt = "Please reply in English"
	}

	// Determine system prompt based on mode
	var systemPrompt string
	switch mode {
	case "generate":
		systemPrompt = ai.TemplateGenerationSystemPrompt + "\n\n" + langPrompt

	case "recommend":
		localTemplates, _ := redc.ListLocalTemplates()
		templateList := make([]string, 0, len(localTemplates))
		for _, t := range localTemplates {
			templateList = append(templateList, fmt.Sprintf("- %s: %s", t.Name, t.Description))
		}
		systemPrompt = fmt.Sprintf(ai.TemplateRecommendationSystemPrompt,
			strings.Join(templateList, "\n"),
			langPrompt)

	case "cost":
		systemPrompt = fmt.Sprintf(ai.CostOptimizationSystemPrompt, langPrompt)
		// Gather running cases info and prepend to the last user message
		casesInfo, runningCount := a.gatherRunningCasesInfo()
		if runningCount > 0 {
			userPrompt := fmt.Sprintf(ai.CostOptimizationUserPrompt, runningCount, casesInfo)
			// Prepend context to the last user message
			if len(messages) > 0 {
				lastIdx := len(messages) - 1
				messages[lastIdx].Content = userPrompt + "\n\n用户额外说明：" + messages[lastIdx].Content
			}
		}

	case "free":
		systemPrompt = fmt.Sprintf(ai.FreeChatSystemPrompt, langPrompt)

	default:
		systemPrompt = fmt.Sprintf(ai.FreeChatSystemPrompt, langPrompt)
	}

	// Build ai.Message slice: system prompt + user-provided history
	aiMessages := make([]ai.Message, 0, len(messages)+1)
	aiMessages = append(aiMessages, ai.Message{Role: "system", Content: systemPrompt})
	for _, m := range messages {
		aiMessages = append(aiMessages, ai.Message{Role: m.Role, Content: m.Content})
	}

	client := ai.NewClient(aiConfig.Provider, aiConfig.APIKey, aiConfig.BaseURL, aiConfig.Model)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	err = client.ChatStream(ctx, aiMessages, func(chunk string) error {
		runtime.EventsEmit(a.ctx, "ai-chat-chunk", map[string]string{
			"conversationId": conversationId,
			"chunk":          chunk,
		})
		return nil
	})

	if err != nil {
		runtime.EventsEmit(a.ctx, "ai-chat-complete", map[string]interface{}{
			"conversationId": conversationId,
			"success":        false,
		})
		return fmt.Errorf(i18n.Tf("app_ai_analysis_failed", err))
	}

	runtime.EventsEmit(a.ctx, "ai-chat-complete", map[string]interface{}{
		"conversationId": conversationId,
		"success":        true,
	})
	return nil
}

// AgentChatStream runs the agentic loop: AI + MCP tool calling + streaming final answer
func (a *App) AgentChatStream(conversationId string, messages []AIChatMessage) error {
	profile, err := redc.GetActiveProfile()
	if err != nil || profile.AIConfig == nil {
		return fmt.Errorf("%s", i18n.T("app_ai_not_configured"))
	}
	aiConfig := profile.AIConfig
	if aiConfig.APIKey == "" || aiConfig.BaseURL == "" || aiConfig.Model == "" {
		return fmt.Errorf("%s", i18n.T("app_ai_config_incomplete"))
	}

	a.mu.Lock()
	project := a.project
	a.mu.Unlock()
	if project == nil {
		return fmt.Errorf("%s", i18n.T("app_project_not_loaded"))
	}

	uiLang := a.GetLanguage()
	langPrompt := "请用中文回复"
	if uiLang == "en" {
		langPrompt = "Please reply in English"
	}
	systemPrompt := fmt.Sprintf(ai.AgentSystemPrompt, langPrompt)

	// Build tool definitions from MCP server
	mcpServer := mcp.NewMCPServer(project, a)
	mcpTools := mcpServer.GetTools()
	toolDefs := make([]ai.ToolDefinition, 0, len(mcpTools))
	for _, t := range mcpTools {
		params := map[string]interface{}{
			"type":       t.InputSchema.Type,
			"properties": t.InputSchema.Properties,
		}
		if len(t.InputSchema.Required) > 0 {
			params["required"] = t.InputSchema.Required
		}
		toolDefs = append(toolDefs, ai.ToolDefinition{
			Type: "function",
			Function: ai.ToolFunctionDef{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  params,
			},
		})
	}

	// Build initial message list: system + history
	aiMessages := make([]ai.Message, 0, len(messages)+1)
	aiMessages = append(aiMessages, ai.Message{Role: "system", Content: systemPrompt})
	for _, m := range messages {
		aiMessages = append(aiMessages, ai.Message{Role: m.Role, Content: m.Content})
	}

	client := ai.NewClient(aiConfig.Provider, aiConfig.APIKey, aiConfig.BaseURL, aiConfig.Model)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Agentic loop: max 10 rounds of tool use
	const maxRounds = 10
	for round := 0; round < maxRounds; round++ {
		resp, err := client.ChatWithTools(ctx, aiMessages, toolDefs)
		if err != nil {
			runtime.EventsEmit(a.ctx, "ai-chat-complete", map[string]interface{}{
				"conversationId": conversationId,
				"success":        false,
			})
			return fmt.Errorf(i18n.Tf("app_ai_analysis_failed", err))
		}

		// No tool calls → stream the final answer
		if len(resp.ToolCalls) == 0 {
			// Append assistant message to history first
			aiMessages = append(aiMessages, ai.Message{Role: "assistant", Content: resp.Content})

			// Stream the final text word-by-word to give a streaming feel
			words := strings.Split(resp.Content, "")
			chunkSize := 8
			for i := 0; i < len(words); i += chunkSize {
				end := i + chunkSize
				if end > len(words) {
					end = len(words)
				}
				runtime.EventsEmit(a.ctx, "ai-chat-chunk", map[string]string{
					"conversationId": conversationId,
					"chunk":          strings.Join(words[i:end], ""),
				})
			}
			runtime.EventsEmit(a.ctx, "ai-chat-complete", map[string]interface{}{
				"conversationId": conversationId,
				"success":        true,
			})
			return nil
		}

		// Append assistant message with tool_calls to history
		aiMessages = append(aiMessages, ai.Message{
			Role:      "assistant",
			Content:   resp.Content,
			ToolCalls: resp.ToolCalls,
		})

		// Execute each tool call
		for _, tc := range resp.ToolCalls {
			// Parse tool arguments
			var args map[string]interface{}
			if tc.Function.Arguments != "" {
				if jsonErr := json.Unmarshal([]byte(tc.Function.Arguments), &args); jsonErr != nil {
					args = map[string]interface{}{}
				}
			}

			// Emit tool-call event to frontend
			runtime.EventsEmit(a.ctx, "ai-agent-tool-call", map[string]interface{}{
				"conversationId": conversationId,
				"toolCallId":     tc.ID,
				"toolName":       tc.Function.Name,
				"toolArgs":       args,
			})

			// Execute via MCP
			result, execErr := mcpServer.ExecuteTool(tc.Function.Name, args)
			var resultContent string
			success := execErr == nil
			if execErr != nil {
				resultContent = fmt.Sprintf("工具执行失败: %v", execErr)
			} else if len(result.Content) > 0 {
				var parts []string
				for _, item := range result.Content {
					parts = append(parts, item.Text)
				}
				resultContent = strings.Join(parts, "\n")
			}

			// Emit result event to frontend
			runtime.EventsEmit(a.ctx, "ai-agent-tool-result", map[string]interface{}{
				"conversationId": conversationId,
				"toolCallId":     tc.ID,
				"toolName":       tc.Function.Name,
				"success":        success,
				"content":        resultContent,
			})

			// Add tool result to message history
			aiMessages = append(aiMessages, ai.Message{
				Role:       "tool",
				Content:    resultContent,
				ToolCallID: tc.ID,
				Name:       tc.Function.Name,
			})
		}
	}

	// Exceeded max rounds
	runtime.EventsEmit(a.ctx, "ai-chat-chunk", map[string]string{
		"conversationId": conversationId,
		"chunk":          "\n\n⚠️ 已达到最大工具调用轮次（10轮），操作结束。",
	})
	runtime.EventsEmit(a.ctx, "ai-chat-complete", map[string]interface{}{
		"conversationId": conversationId,
		"success":        true,
	})
	return nil
}
func (a *App) gatherRunningCasesInfo() (string, int) {
	a.mu.Lock()
	project := a.project
	pricingService := a.pricingService
	costCalculator := a.costCalculator
	a.mu.Unlock()

	if project == nil || pricingService == nil || costCalculator == nil {
		return "", 0
	}

	cases, err := redc.LoadProjectCases(project.ProjectName)
	if err != nil {
		return "", 0
	}

	var caseInfoList []string
	runningCount := 0

	for _, c := range cases {
		if c.State != redc.StateRunning {
			continue
		}
		runningCount++

		if c.Path == "" {
			caseInfo := fmt.Sprintf(`- **%s**
  - 模板: %s
  - 状态: 运行中
  - 说明: 场景路径为空
  - 建议: 请检查场景配置`, c.Name, c.Module)
			caseInfoList = append(caseInfoList, caseInfo)
			continue
		}

		state, err := redc.TfStatus(c.Path)
		if err != nil {
			caseInfo := fmt.Sprintf(`- **%s**
  - 模板: %s
  - 状态: 运行中
  - 说明: 状态获取失败 (%v)
  - 建议: 请检查 Terraform 是否正确安装，场景是否已完成部署`, c.Name, c.Module, err)
			caseInfoList = append(caseInfoList, caseInfo)
			continue
		}

		if state == nil || state.Values == nil {
			caseInfo := fmt.Sprintf(`- **%s**
  - 模板: %s
  - 状态: 运行中
  - 说明: 状态数据为空
  - 建议: 该场景可能尚未创建资源`, c.Name, c.Module)
			caseInfoList = append(caseInfoList, caseInfo)
			continue
		}

		resources := extractResourcesFromState(state)
		if resources == nil || len(resources.Resources) == 0 {
			caseInfo := fmt.Sprintf(`- **%s**
  - 模板: %s
  - 状态: 运行中
  - 说明: 未找到资源信息
  - 建议: 该场景可能尚未创建资源，或资源已被销毁`, c.Name, c.Module)
			caseInfoList = append(caseInfoList, caseInfo)
			continue
		}

		estimate, err := costCalculator.CalculateCost(resources, pricingService)
		if err != nil {
			var resourceList []string
			for _, r := range resources.Resources {
				resourceList = append(resourceList, fmt.Sprintf("  - %s (%s)", r.Name, r.Type))
			}
			caseInfo := fmt.Sprintf(`- **%s**
  - 模板: %s
  - 状态: 运行中
  - 资源数量: %d
  - 资源列表:
%s
  - 说明: 成本计算失败 (%v)
  - 建议: 请检查定价数据是否可用`, c.Name, c.Module, len(resources.Resources), strings.Join(resourceList, "\n"), err)
			caseInfoList = append(caseInfoList, caseInfo)
			continue
		}

		var resourceDetails []string
		for _, rb := range estimate.Breakdown {
			if rb.TotalMonthly > 0 {
				resourceDetails = append(resourceDetails, fmt.Sprintf("  - %s (%s): ¥%.2f/月",
					rb.ResourceName, rb.ResourceType, rb.TotalMonthly))
			} else if !rb.Available {
				resourceDetails = append(resourceDetails, fmt.Sprintf("  - %s (%s): 定价不可用",
					rb.ResourceName, rb.ResourceType))
			}
		}

		provider := "未知"
		if len(estimate.Breakdown) > 0 {
			provider = estimate.Breakdown[0].Provider
		}

		caseInfo := fmt.Sprintf(`- **%s**
  - 模板: %s
  - 云服务商: %s
  - 月度成本: ¥%.2f
  - 资源数量: %d
  - 资源详情:
%s`, c.Name, c.Module, provider, estimate.TotalMonthlyCost, len(estimate.Breakdown), strings.Join(resourceDetails, "\n"))

		caseInfoList = append(caseInfoList, caseInfo)
	}

	return strings.Join(caseInfoList, "\n\n"), runningCount
}
