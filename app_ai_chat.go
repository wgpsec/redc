package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"red-cloud/i18n"
	redc "red-cloud/mod"
	"red-cloud/mod/ai"
	"red-cloud/mod/gologger"
	"red-cloud/mod/mcp"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// agentCancelMap stores cancel functions for active agent conversations
var agentCancelMap = struct {
	sync.Mutex
	m map[string]context.CancelFunc
}{m: make(map[string]context.CancelFunc)}

// askUserChannels stores channels for ask_user tool responses, keyed by conversationId
var askUserChannels = struct {
	sync.Mutex
	m map[string]chan string
}{m: make(map[string]chan string)}

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

	// System prompt for free chat mode (only mode used via AIChatStream)
	systemPrompt := fmt.Sprintf(ai.FreeChatSystemPrompt, langPrompt)

	// Build ai.Message slice: system prompt + user-provided history
	aiMessages := make([]ai.Message, 0, len(messages)+1)
	aiMessages = append(aiMessages, ai.Message{Role: "system", Content: systemPrompt})
	for _, m := range messages {
		aiMessages = append(aiMessages, ai.Message{Role: m.Role, Content: m.Content})
	}

	// Context window management: compact if exceeding budget
	contextBudget := 108000
	if aiConfig.ContextWindow > 0 {
		contextBudget = aiConfig.ContextWindow * 9 / 10
	}
	if estimated := ai.EstimateTokens(aiMessages); estimated > contextBudget {
		client := buildProviderManager(aiConfig).CurrentClient()
		compactCtx, compactCancel := context.WithTimeout(context.Background(), 35*time.Second)
		aiMessages = ai.CompactWithLLM(compactCtx, client, aiMessages, ai.CompactOptions{
			KeepRecentRounds: 4,
			ContextBudget:    contextBudget,
			MaxSummaryTokens: 2000,
		})
		compactCancel()
		newEstimated := ai.EstimateTokens(aiMessages)
		gologger.Info().Msgf("chat: context compacted from ~%d to ~%d tokens", estimated, newEstimated)
		a.emitEvent("ai-chat-compact", map[string]interface{}{
			"conversationId": conversationId,
			"before":         estimated,
			"after":          newEstimated,
			"budget":         contextBudget,
		})
	}

	// Build provider manager with failover support
	pm := buildProviderManager(aiConfig)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Register cancel so StopAgentStream can abort free chat too
	agentCancelMap.Lock()
	agentCancelMap.m[conversationId] = cancel
	agentCancelMap.Unlock()
	defer func() {
		agentCancelMap.Lock()
		delete(agentCancelMap.m, conversationId)
		agentCancelMap.Unlock()
	}()

	// Try with failover: retry on transient/permanent errors by switching providers
	maxAttempts := pm.Count() + 1
	for attempt := 0; attempt < maxAttempts; attempt++ {
		client := pm.CurrentClient()
		err = client.ChatStream(ctx, aiMessages, func(chunk string) error {
			a.emitEvent("ai-chat-chunk", map[string]string{
				"conversationId": conversationId,
				"chunk":          chunk,
			})
			return nil
		})

		if err == nil {
			break
		}

		// Try failover on recoverable errors
		if ai.ShouldFailover(err.Error()) && pm.Failover(err.Error()) {
			newProvider := pm.Current()
			gologger.Info().Msgf("ai-chat: failover to %s after error: %s", newProvider.Name, err.Error())
			// Notify frontend via both event and inline chunk
			a.emitEvent("ai-chat-failover", map[string]string{
				"conversationId": conversationId,
				"provider":       newProvider.Name,
				"model":          newProvider.Model,
				"error":          err.Error(),
			})
			a.emitEvent("ai-chat-chunk", map[string]string{
				"conversationId": conversationId,
				"chunk":          fmt.Sprintf("\n\n> ⚠️ **Provider Failover**: %s → %s (%s)\n\n", "primary", newProvider.Name, newProvider.Model),
			})
			continue
		}
		break // non-recoverable or no more providers
	}

	if err != nil {
		// User-initiated stop: context was cancelled via StopAgentStream
		if ctx.Err() == context.Canceled {
			a.emitEvent("ai-chat-chunk", map[string]string{
				"conversationId": conversationId,
				"chunk":          "\n\n⏹️ " + i18n.T("app_ai_user_stopped"),
			})
			a.emitEvent("ai-chat-complete", map[string]interface{}{
				"conversationId": conversationId,
				"success":        true,
			})
			return nil
		}
		a.emitEvent( "ai-chat-complete", map[string]interface{}{
			"conversationId": conversationId,
			"success":        false,
		})
		return fmt.Errorf(i18n.Tf("app_ai_analysis_failed", err))
	}

	a.emitEvent( "ai-chat-complete", map[string]interface{}{
		"conversationId": conversationId,
		"success":        true,
	})
	return nil
}

// AgentChatStream runs the agentic loop: AI + MCP tool calling + streaming final answer
func (a *App) AgentChatStream(conversationId string, messages []AIChatMessage) error {
	return a.runAgentLoop(conversationId, messages, ai.AgentSystemPrompt, 50, 10*time.Minute)
}

// DeployAgentChatStream runs the deploy agent loop with specialized system prompt
func (a *App) DeployAgentChatStream(conversationId string, messages []AIChatMessage) error {
	return a.runAgentLoop(conversationId, messages, ai.DeployAgentSystemPrompt, 50, 15*time.Minute)
}

// TroubleshootAgentChatStream runs the troubleshoot agent loop
func (a *App) TroubleshootAgentChatStream(conversationId string, messages []AIChatMessage) error {
	return a.runAgentLoop(conversationId, messages, ai.TroubleshootAgentSystemPrompt, 30, 10*time.Minute)
}

// SmartAgentChatStream auto-classifies user intent and routes to the best specialized agent.
// For generate/recommend/cost intents, it injects contextual data into messages before routing to AgentChatStream.
func (a *App) SmartAgentChatStream(conversationId string, messages []AIChatMessage) error {
	intent := a.classifyIntent(messages)
	switch intent {
	case "deploy":
		return a.DeployAgentChatStream(conversationId, messages)
	case "troubleshoot":
		return a.TroubleshootAgentChatStream(conversationId, messages)
	case "generate":
		// Inject hint into system context so Agent knows to generate a template
		if len(messages) > 0 {
			lastIdx := len(messages) - 1
			messages[lastIdx].Content = "[用户意图: 生成 RedC 场景模板]\n\n" + messages[lastIdx].Content +
				"\n\n请先调用 list_templates 和 search_templates 了解已有模板结构，然后为用户生成完整的 RedC 模板文件（case.json + main.tf + variables.tf + outputs.tf + versions.tf）。"
		}
		return a.AgentChatStream(conversationId, messages)
	case "recommend":
		// Inject available templates context
		localTemplates, _ := redc.ListLocalTemplates()
		if len(localTemplates) > 0 {
			templateList := make([]string, 0, len(localTemplates))
			for _, t := range localTemplates {
				templateList = append(templateList, fmt.Sprintf("- %s: %s", t.Name, t.Description))
			}
			if len(messages) > 0 {
				lastIdx := len(messages) - 1
				messages[lastIdx].Content = "[用户意图: 推荐场景模板]\n\n" + messages[lastIdx].Content +
					"\n\n以下是用户本地已有的模板列表，请优先从中推荐，也可调用 search_templates 搜索更多：\n" +
					strings.Join(templateList, "\n")
			}
		}
		return a.AgentChatStream(conversationId, messages)
	case "cost":
		// Inject running cases cost context
		casesInfo, runningCount := a.gatherRunningCasesInfo()
		if runningCount > 0 && len(messages) > 0 {
			lastIdx := len(messages) - 1
			messages[lastIdx].Content = fmt.Sprintf("[用户意图: 成本优化分析]\n\n%s\n\n当前有 %d 个运行中的场景：\n%s\n\n请分析上述场景并提供成本优化建议。也可调用 list_cases 获取最新数据。",
				messages[lastIdx].Content, runningCount, casesInfo)
		}
		return a.AgentChatStream(conversationId, messages)
	default:
		return a.AgentChatStream(conversationId, messages)
	}
}

// classifyIntent uses a lightweight LLM call to classify the user's intent
func (a *App) classifyIntent(messages []AIChatMessage) string {
	profile, err := redc.GetActiveProfile()
	if err != nil || profile.AIConfig == nil {
		return "ops"
	}
	aiConfig := profile.AIConfig
	if aiConfig.APIKey == "" {
		return "ops"
	}

	// Find the last user message
	var lastUserMsg string
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			lastUserMsg = messages[i].Content
			break
		}
	}
	if lastUserMsg == "" {
		return "ops"
	}

	// Truncate long messages
	if len(lastUserMsg) > 500 {
		lastUserMsg = lastUserMsg[:500]
	}

	prompt := fmt.Sprintf(ai.IntentClassificationPrompt, lastUserMsg)
	client := ai.NewClient(aiConfig.Provider, aiConfig.APIKey, aiConfig.BaseURL, aiConfig.Model)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := client.ChatWithTools(ctx, []ai.Message{
		{Role: "user", Content: prompt},
	}, nil)
	if err != nil || resp.Content == "" {
		return "ops"
	}

	result := strings.TrimSpace(strings.ToLower(resp.Content))
	switch {
	case strings.Contains(result, "deploy"):
		return "deploy"
	case strings.Contains(result, "troubleshoot"):
		return "troubleshoot"
	case strings.Contains(result, "generate"):
		return "generate"
	case strings.Contains(result, "recommend"):
		return "recommend"
	case strings.Contains(result, "cost"):
		return "cost"
	default:
		return "ops"
	}
}

// StopAgentStream cancels a running agent conversation
func (a *App) StopAgentStream(conversationId string) {
	agentCancelMap.Lock()
	if cancel, ok := agentCancelMap.m[conversationId]; ok {
		cancel()
		delete(agentCancelMap.m, conversationId)
	}
	agentCancelMap.Unlock()
}

// SubmitAskUserResponse sends user's answer back to a waiting ask_user tool call
func (a *App) SubmitAskUserResponse(conversationId string, answer string) {
	askUserChannels.Lock()
	ch, ok := askUserChannels.m[conversationId]
	askUserChannels.Unlock()
	if ok {
		select {
		case ch <- answer:
		default:
		}
	}
}

// ExportChatLog saves chat log content to a user-selected file
func (a *App) ExportChatLog(content string) error {
	filename := fmt.Sprintf("redc-chat-%s.md", time.Now().Format("20060102-150405"))
	filePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "导出对话日志",
		DefaultFilename: filename,
		Filters: []runtime.FileFilter{
			{DisplayName: "Markdown", Pattern: "*.md"},
			{DisplayName: "All Files", Pattern: "*.*"},
		},
	})
	if err != nil {
		return err
	}
	if filePath == "" {
		return nil
	}
	return os.WriteFile(filePath, []byte(content), 0644)
}

// ExportConsoleLogs exports console log content to a file via native save dialog
func (a *App) ExportConsoleLogs(content string) error {
	filename := fmt.Sprintf("redc-console-%s.log", time.Now().Format("20060102-150405"))
	filePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "导出控制台日志",
		DefaultFilename: filename,
		Filters: []runtime.FileFilter{
			{DisplayName: "Log Files", Pattern: "*.log"},
			{DisplayName: "Text Files", Pattern: "*.txt"},
			{DisplayName: "All Files", Pattern: "*.*"},
		},
	})
	if err != nil {
		return err
	}
	if filePath == "" {
		return nil
	}
	return os.WriteFile(filePath, []byte(content), 0644)
}

// runAgentLoop is the shared agentic loop used by AgentChatStream and DeployAgentChatStream
func (a *App) runAgentLoop(conversationId string, messages []AIChatMessage, promptTemplate string, defaultMaxRounds int, timeout time.Duration) error {
	profile, err := redc.GetActiveProfile()
	if err != nil || profile.AIConfig == nil {
		return fmt.Errorf("%s", i18n.T("app_ai_not_configured"))
	}
	aiConfig := profile.AIConfig
	if aiConfig.APIKey == "" || aiConfig.BaseURL == "" || aiConfig.Model == "" {
		return fmt.Errorf("%s", i18n.T("app_ai_config_incomplete"))
	}

	// Use user-configured max rounds if set, otherwise use default (cap at 200)
	maxRounds := defaultMaxRounds
	if aiConfig.MaxToolRounds > 0 {
		maxRounds = aiConfig.MaxToolRounds
	}
	if maxRounds > 200 {
		maxRounds = 200
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
	systemPrompt := fmt.Sprintf(promptTemplate, langPrompt)

	// Inject current time context for time-aware operations
	now := time.Now()
	zone, offset := now.Zone()
	offsetHours := offset / 3600
	systemPrompt += fmt.Sprintf("\n\n## 当前时间\n当前时间: %s (时区: %s, UTC%+d:00)",
		now.Format("2006-01-02 15:04:05"), zone, offsetHours)

	// Build tool definitions from MCP server
	mcpServer := mcp.NewMCPServer(project, a)
	mcpTools := mcpServer.GetTools()
	enableAskUser := aiConfig.EnableAskUser == nil || *aiConfig.EnableAskUser // default true
	toolDefs := make([]ai.ToolDefinition, 0, len(mcpTools))
	for _, t := range mcpTools {
		if t.Name == "ask_user" && !enableAskUser {
			continue
		}
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

	// Build provider manager with failover support
	pm := buildProviderManager(aiConfig)
	client := pm.CurrentClient()

	// Inject skills suggestions into system prompt based on user query context
	if len(messages) > 0 {
		lastUserMsg := ""
		for i := len(messages) - 1; i >= 0; i-- {
			if messages[i].Role == "user" {
				lastUserMsg = messages[i].Content
				break
			}
		}
		if lastUserMsg != "" {
			skillsDir := ""
			if redc.RedcPath != "" {
				skillsDir = redc.RedcPath + "/skills"
			}
			engine := ai.NewSkillsEngine(skillsDir)
			suggestions := engine.Suggest(lastUserMsg, 3)
			if skillsBlock := ai.FormatSuggestions(suggestions); skillsBlock != "" {
				aiMessages[0].Content += skillsBlock
			}
		}
	}

	// Initialize safety hooks
	hooks := ai.NewHookChain()
	// Dynamic timeout: base timeout + extra per round (each round may involve API call + tool execution)
	dynamicTimeout := timeout + time.Duration(maxRounds)*30*time.Second
	ctx, cancel := context.WithTimeout(context.Background(), dynamicTimeout)
	defer cancel()

	// Register cancel function so frontend can stop this conversation
	agentCancelMap.Lock()
	agentCancelMap.m[conversationId] = cancel
	agentCancelMap.Unlock()
	defer func() {
		agentCancelMap.Lock()
		delete(agentCancelMap.m, conversationId)
		agentCancelMap.Unlock()
	}()

	// Register exec timeout callback: when exec_command/exec_userdata times out,
	// ask user whether to continue (extend 10 min) or abort
	mcpServer.RegisterExecTimeoutAsk(conversationId, func(command string, elapsed time.Duration, partialOutput string) bool {
		// Show last 300 chars of output for context
		truncatedOutput := partialOutput
		if len(truncatedOutput) > 300 {
			truncatedOutput = "..." + truncatedOutput[len(truncatedOutput)-300:]
		}
		question := fmt.Sprintf("命令执行已超时（%s）：\n`%s`\n\n最近输出：\n```\n%s\n```\n\n是否继续等待？", elapsed.Round(time.Second), command, truncatedOutput)

		ch := make(chan string, 1)
		askUserChannels.Lock()
		askUserChannels.m[conversationId] = ch
		askUserChannels.Unlock()
		defer func() {
			askUserChannels.Lock()
			delete(askUserChannels.m, conversationId)
			askUserChannels.Unlock()
		}()

		a.emitEvent("ai-agent-ask-user", map[string]interface{}{
			"conversationId": conversationId,
			"toolCallId":     "exec-timeout-" + fmt.Sprintf("%d", time.Now().UnixMilli()),
			"question":       question,
			"choices":        []string{"继续等待（延长10分钟）", "终止命令"},
			"allowFreeform":  false,
		})

		select {
		case answer := <-ch:
			return strings.Contains(answer, "继续")
		case <-ctx.Done():
			return false
		}
	})
	defer mcpServer.UnregisterExecTimeoutAsk(conversationId)

	// Determine context window budget (use 90% of configured window, reserve for output)
	contextBudget := 108000 // default ~90% of 120K
	if aiConfig.ContextWindow > 0 {
		contextBudget = aiConfig.ContextWindow * 9 / 10
	}

	// Accumulate token usage across all rounds
	var totalUsage ai.TokenUsage

	// Agentic loop
	for round := 0; round < maxRounds; round++ {
		// Smart context window management: LLM-powered compaction when exceeding budget
		estimated := ai.EstimateTokens(aiMessages)
		if estimated > contextBudget {
			// Save full transcript before compaction for audit trail
			if redc.RedcPath != "" {
				transcriptDir := filepath.Join(redc.RedcPath, "transcripts")
				os.MkdirAll(transcriptDir, 0755)
				transcriptFile := filepath.Join(transcriptDir, fmt.Sprintf("%s-round%d.json", conversationId, round))
				if err := ai.SaveTranscript(aiMessages, transcriptFile); err != nil {
					gologger.Warning().Msgf("agent: failed to save transcript before compaction: %v", err)
				}
			}
			compactOpts := ai.CompactOptions{
				KeepRecentRounds: 4,
				ContextBudget:    contextBudget,
				MaxSummaryTokens: 2000,
			}
			aiMessages = ai.CompactWithLLM(ctx, client, aiMessages, compactOpts)
			newEstimated := ai.EstimateTokens(aiMessages)
			gologger.Info().Msgf("agent: context compacted from ~%d to ~%d tokens (budget: %d)", estimated, newEstimated, contextBudget)
			a.emitEvent("ai-chat-compact", map[string]interface{}{
				"conversationId": conversationId,
				"before":         estimated,
				"after":          newEstimated,
				"budget":         contextBudget,
			})
		} else if round > 2 {
			// Light compression for older tool results even when under budget
			compressOldToolResults(aiMessages)
		}

		// Check if cancelled before each round
		if ctx.Err() != nil {
			msg := "\n\n⏹️ " + i18n.T("app_ai_user_stopped")
			if ctx.Err() == context.DeadlineExceeded {
				msg = "\n\n⏱️ " + i18n.Tf("app_ai_timeout", round)
			}
			a.emitEvent("ai-chat-chunk", map[string]string{
				"conversationId": conversationId,
				"chunk":          msg,
			})
			a.emitEvent("ai-chat-complete", map[string]interface{}{
				"conversationId": conversationId,
				"success":        true,
				"usage":          totalUsage,
			})
			return nil
		}

		// Use streaming tool call so users can see AI thinking in real-time
		resp, err := client.ChatWithToolsStream(ctx, aiMessages, toolDefs, func(chunk string) error {
			a.emitEvent("ai-chat-chunk", map[string]string{
				"conversationId": conversationId,
				"chunk":          chunk,
			})
			return nil
		})
		if err != nil {
			// Try provider failover on API errors
			if ai.ShouldFailover(err.Error()) && pm.Failover(err.Error()) {
				client = pm.CurrentClient()
				newProvider := pm.Current()
				gologger.Info().Msgf("agent: failover to %s after error: %s", newProvider.Name, err.Error())
				a.emitEvent("ai-chat-failover", map[string]string{
					"conversationId": conversationId,
					"provider":       newProvider.Name,
					"model":          newProvider.Model,
					"error":          err.Error(),
				})
				a.emitEvent("ai-chat-chunk", map[string]string{
					"conversationId": conversationId,
					"chunk":          fmt.Sprintf("\n\n> ⚠️ **Provider Failover**: %s → %s (%s)\n\n", "primary", newProvider.Name, newProvider.Model),
				})
				continue // Retry this round with new provider
			}
			if ctx.Err() != nil {
				msg := "\n\n⏹️ " + i18n.T("app_ai_user_stopped")
				if ctx.Err() == context.DeadlineExceeded {
					msg = "\n\n⏱️ " + i18n.Tf("app_ai_timeout", round)
				}
				a.emitEvent("ai-chat-chunk", map[string]string{
					"conversationId": conversationId,
					"chunk":          msg,
				})
				a.emitEvent("ai-chat-complete", map[string]interface{}{
					"conversationId": conversationId,
					"success":        true,
					"usage":          totalUsage,
				})
				return nil
			}
			a.emitEvent("ai-chat-complete", map[string]interface{}{
				"conversationId": conversationId,
				"success":        false,
				"usage":          totalUsage,
			})
			return fmt.Errorf(i18n.Tf("app_ai_analysis_failed", err))
		}

		// Accumulate token usage
		totalUsage.PromptTokens += resp.Usage.PromptTokens
		totalUsage.CompletionTokens += resp.Usage.CompletionTokens
		totalUsage.TotalTokens += resp.Usage.TotalTokens

		// No tool calls → final answer (already streamed via callback)
		if len(resp.ToolCalls) == 0 {
			aiMessages = append(aiMessages, ai.Message{Role: "assistant", Content: resp.Content})
			a.notifyAgentComplete(i18n.T("notify_agent_complete"), i18n.Tf("notify_agent_complete_msg", round))
			a.emitEvent("ai-chat-complete", map[string]interface{}{
				"conversationId": conversationId,
				"success":        true,
				"usage":          totalUsage,
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
		// Determine if we can parallelize (multiple calls, none are interactive/write)
		parallelizable := len(resp.ToolCalls) > 1 && canParallelizeToolCalls(resp.ToolCalls)

		type toolExecResult struct {
			tc            ai.ToolCall
			args          map[string]interface{}
			resultContent string
			success       bool
		}

		var execResults []toolExecResult

		if parallelizable {
			// Parallel execution for independent read-only tools
			execResults = make([]toolExecResult, len(resp.ToolCalls))
			var wg sync.WaitGroup
			for i, tc := range resp.ToolCalls {
				wg.Add(1)
				go func(idx int, call ai.ToolCall) {
					defer wg.Done()
					res := toolExecResult{tc: call}
					res.args = parseToolArgs(call)
					res.resultContent, res.success = a.executeSingleTool(call, res.args, mcpServer, conversationId, ctx)
					execResults[idx] = res
				}(i, tc)
			}
			wg.Wait()

			// Emit events for all parallel results
			for _, res := range execResults {
				a.emitEvent("ai-agent-tool-call", map[string]interface{}{
					"conversationId": conversationId,
					"toolCallId":     res.tc.ID,
					"toolName":       res.tc.Function.Name,
					"toolArgs":       res.args,
				})

				// Truncate large results
				content := res.resultContent
				const maxToolResultLen = 8000
				if len(content) > maxToolResultLen {
					content = content[:maxToolResultLen] + "\n\n... (output truncated, total " + fmt.Sprintf("%d", len(content)) + " bytes)"
				}

				a.emitEvent("ai-agent-tool-result", map[string]interface{}{
					"conversationId": conversationId,
					"toolCallId":     res.tc.ID,
					"toolName":       res.tc.Function.Name,
					"success":        res.success,
					"content":        content,
				})

				aiMessages = append(aiMessages, ai.Message{
					Role:       "tool",
					Content:    content,
					ToolCallID: res.tc.ID,
					Name:       res.tc.Function.Name,
				})
			}
		} else {
			// Serial execution (original path + retry enhancement)
			for _, tc := range resp.ToolCalls {
				args := parseToolArgs(tc)

				// Run safety pre-hooks
				hookResult := hooks.RunPreHooks(tc.Function.Name, args)
				if hookResult.Action == ai.HookBlock {
					resultContent := fmt.Sprintf("⛔ Blocked by safety hook: %s", hookResult.Message)
					a.emitEvent("ai-agent-tool-result", map[string]interface{}{
						"conversationId": conversationId,
						"toolCallId":     tc.ID,
						"toolName":       tc.Function.Name,
						"success":        false,
						"content":        resultContent,
					})
					aiMessages = append(aiMessages, ai.Message{
						Role: "tool", Content: resultContent, ToolCallID: tc.ID, Name: tc.Function.Name,
					})
					continue
				}
				if hookResult.Action == ai.HookConfirm {
					// Use ask_user mechanism for confirmation
					confirmResult, _ := a.handleAskUser(map[string]interface{}{
						"question":       hookResult.Message,
						"choices":        []interface{}{"Yes, proceed", "No, cancel"},
						"allow_freeform": false,
					}, conversationId, ctx)
					if !strings.Contains(strings.ToLower(confirmResult), "yes") &&
						!strings.Contains(strings.ToLower(confirmResult), "proceed") {
						resultContent := "Operation cancelled by user."
						a.emitEvent("ai-agent-tool-result", map[string]interface{}{
							"conversationId": conversationId,
							"toolCallId":     tc.ID,
							"toolName":       tc.Function.Name,
							"success":        false,
							"content":        resultContent,
						})
						aiMessages = append(aiMessages, ai.Message{
							Role: "tool", Content: resultContent, ToolCallID: tc.ID, Name: tc.Function.Name,
						})
						continue
					}
				}

				a.emitEvent("ai-agent-tool-call", map[string]interface{}{
					"conversationId": conversationId,
					"toolCallId":     tc.ID,
					"toolName":       tc.Function.Name,
					"toolArgs":       args,
				})

				resultContent, success := a.executeSingleTool(tc, args, mcpServer, conversationId, ctx)

				// Run post-hooks (annotations)
				resultContent = hooks.RunPostHooks(tc.Function.Name, args, resultContent, success)

				// Truncate large tool results to prevent context window overflow
				const maxToolResultLen = 8000
				if len(resultContent) > maxToolResultLen {
					resultContent = resultContent[:maxToolResultLen] + "\n\n... (output truncated, total " + fmt.Sprintf("%d", len(resultContent)) + " bytes)"
				}

				a.emitEvent("ai-agent-tool-result", map[string]interface{}{
					"conversationId": conversationId,
					"toolCallId":     tc.ID,
					"toolName":       tc.Function.Name,
					"success":        success,
					"content":        resultContent,
				})

				aiMessages = append(aiMessages, ai.Message{
					Role:       "tool",
					Content:    resultContent,
					ToolCallID: tc.ID,
					Name:       tc.Function.Name,
				})
			}
		}
	}

	// Exceeded max rounds
	a.notifyAgentComplete(i18n.T("notify_agent_max_rounds"), i18n.Tf("notify_agent_max_rounds_msg", maxRounds))
	a.emitEvent( "ai-chat-chunk", map[string]string{
		"conversationId": conversationId,
		"chunk":          fmt.Sprintf("\n\n⚠️ 已达到最大工具调用轮次（%d轮），操作结束。", maxRounds),
	})
	a.emitEvent( "ai-chat-complete", map[string]interface{}{
		"conversationId": conversationId,
		"success":        true,
		"usage":          totalUsage,
	})
	return nil
}

func (a *App) notifyAgentComplete(title, message string) {
	if a.notificationMgr != nil {
		a.notificationMgr.Send(title, message)
	}
}

// compressOldToolResults shortens tool result messages beyond the most recent round
// to prevent context window bloat over many rounds.
// It keeps the last 4 tool-role messages at full length and compresses older ones.
func compressOldToolResults(messages []ai.Message) {
	const maxCompressedLen = 200
	const keepRecentToolMessages = 4

	// Count total tool messages
	toolCount := 0
	for i := range messages {
		if messages[i].Role == "tool" {
			toolCount++
		}
	}
	if toolCount <= keepRecentToolMessages {
		return
	}

	// Compress older tool messages (keep last keepRecentToolMessages intact)
	skipCount := toolCount - keepRecentToolMessages
	compressed := 0
	for i := 0; i < len(messages); i++ {
		if messages[i].Role == "tool" && compressed < skipCount {
			content := messages[i].Content
			runes := []rune(content)
			if len(runes) > maxCompressedLen {
				messages[i].Content = string(runes[:maxCompressedLen]) + "... (compressed)"
			}
			compressed++
		}
	}
}

// parseToolArgs parses tool call arguments JSON, handling streaming edge cases
func parseToolArgs(tc ai.ToolCall) map[string]interface{} {
	args := map[string]interface{}{}
	if tc.Function.Arguments != "" {
		if jsonErr := json.Unmarshal([]byte(tc.Function.Arguments), &args); jsonErr != nil {
			trimmed := strings.TrimSpace(tc.Function.Arguments)
			if trimmed != "{" && trimmed != "" {
				args["_parse_error"] = jsonErr.Error()
				args["_raw_arguments"] = tc.Function.Arguments
			}
		}
	}
	return args
}

// canParallelizeToolCalls checks if multiple tool calls can run in parallel.
// Interactive tools (ask_user, update_plan) and write operations must run serially.
func canParallelizeToolCalls(calls []ai.ToolCall) bool {
	for _, tc := range calls {
		switch tc.Function.Name {
		case "ask_user", "update_plan",
			"start_case", "stop_case", "kill_case", "plan_case", "delete_case",
			"compose_up", "compose_down",
			"exec_command", "exec_userdata",
			"save_compose_file", "save_template_files",
			"pull_template", "delete_template",
			"schedule_task":
			return false
		}
	}
	return true
}

// isRetryableError checks if an error is transient and worth retrying automatically
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	retryablePatterns := []string{
		"connection refused",
		"connection reset",
		"i/o timeout",
		"broken pipe",
		"no route to host",
		"failed to create SSH client",
		"failed to create SSH session",
		"dial tcp",
	}
	for _, pattern := range retryablePatterns {
		if strings.Contains(msg, pattern) {
			return true
		}
	}
	return false
}

// executeSingleTool handles execution of a single tool call with retry for transient errors
func (a *App) executeSingleTool(tc ai.ToolCall, args map[string]interface{}, mcpServer *mcp.MCPServer, conversationId string, ctx context.Context) (string, bool) {
	// Check for parse error
	if parseErr, ok := args["_parse_error"]; ok {
		raw, _ := args["_raw_arguments"].(string)
		return fmt.Sprintf("工具参数 JSON 解析失败: %v\n原始参数: %s", parseErr, raw), false
	}

	if tc.Function.Name == "ask_user" {
		return a.handleAskUser(args, conversationId, ctx)
	}

	if tc.Function.Name == "update_plan" {
		return a.handleUpdatePlan(args, conversationId)
	}

	// Generic tool execution with auto-retry for transient errors
	args["_conversation_id"] = conversationId
	result, execErr := mcpServer.ExecuteTool(tc.Function.Name, args)
	delete(args, "_conversation_id")

	if execErr != nil && isRetryableError(execErr) {
		gologger.Info().Msgf("agent: retrying %s after transient error: %v", tc.Function.Name, execErr)
		time.Sleep(5 * time.Second)
		result, execErr = mcpServer.ExecuteTool(tc.Function.Name, args)
		delete(args, "_conversation_id")
	}

	if execErr != nil {
		return fmt.Sprintf("工具执行失败: %v", execErr), false
	}
	if len(result.Content) > 0 {
		var parts []string
		for _, item := range result.Content {
			parts = append(parts, item.Text)
		}
		return strings.Join(parts, "\n"), true
	}
	return "", true
}

// handleAskUser handles the ask_user tool call
func (a *App) handleAskUser(args map[string]interface{}, conversationId string, ctx context.Context) (string, bool) {
	question, _ := args["question"].(string)
	var choices []string
	if rawChoices, ok := args["choices"].([]interface{}); ok {
		for _, c := range rawChoices {
			if s, ok := c.(string); ok {
				choices = append(choices, s)
			}
		}
	}
	allowFreeform := true
	if af, ok := args["allow_freeform"].(bool); ok {
		allowFreeform = af
	}

	ch := make(chan string, 1)
	askUserChannels.Lock()
	askUserChannels.m[conversationId] = ch
	askUserChannels.Unlock()
	defer func() {
		askUserChannels.Lock()
		delete(askUserChannels.m, conversationId)
		askUserChannels.Unlock()
	}()

	a.emitEvent("ai-agent-ask-user", map[string]interface{}{
		"conversationId": conversationId,
		"toolCallId":     fmt.Sprintf("ask-%d", time.Now().UnixMilli()),
		"question":       question,
		"choices":        choices,
		"allowFreeform":  allowFreeform,
	})

	select {
	case answer := <-ch:
		return answer, true
	case <-ctx.Done():
		return "用户未回答，操作已取消", false
	}
}

// handleUpdatePlan handles the update_plan tool call
func (a *App) handleUpdatePlan(args map[string]interface{}, conversationId string) (string, bool) {
	title, _ := args["title"].(string)
	steps, _ := args["steps"]
	currentStep := 0
	if cs, ok := args["current_step"].(float64); ok {
		currentStep = int(cs)
	}

	// Normalize step field: AI may use "content" instead of "name"
	if stepsArr, ok := steps.([]interface{}); ok {
		for _, s := range stepsArr {
			if stepMap, ok := s.(map[string]interface{}); ok {
				if _, hasName := stepMap["name"]; !hasName {
					if content, hasContent := stepMap["content"]; hasContent {
						stepMap["name"] = content
					}
				}
			}
		}
	}

	a.emitEvent("ai-agent-plan", map[string]interface{}{
		"conversationId": conversationId,
		"title":          title,
		"steps":          steps,
		"currentStep":    currentStep,
	})

	return "Plan updated and displayed to user.", true
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
