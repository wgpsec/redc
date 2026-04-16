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

	case "errorAnalysis":
		// Try to read template content for context
		templateContext := ""
		if len(messages) > 0 {
			// Extract template name from the first user message
			firstMsg := messages[0].Content
			if idx := strings.Index(firstMsg, "模板: "); idx >= 0 {
				end := strings.Index(firstMsg[idx+len("模板: "):], "\n")
				var tmplName string
				if end >= 0 {
					tmplName = firstMsg[idx+len("模板: ") : idx+len("模板: ")+end]
				} else {
					tmplName = firstMsg[idx+len("模板: "):]
				}
				tmplName = strings.TrimSpace(tmplName)
				if tmplName != "" {
					templateContext = a.gatherTemplateContext(tmplName)
				}
			}
		}
		systemPrompt = fmt.Sprintf(ai.ErrorAnalysisChatSystemPrompt, templateContext, langPrompt)

	case "free":
		systemPrompt = fmt.Sprintf(ai.FreeChatSystemPrompt, langPrompt)

	default:
		systemPrompt = fmt.Sprintf(ai.FreeChatSystemPrompt, langPrompt)
	}

	// Inject agent memory context if enabled
	enableMemory := aiConfig.EnableMemory == nil || *aiConfig.EnableMemory
	if enableMemory && a.memoryStore != nil {
		a.mu.Lock()
		projectName := ""
		if a.project != nil {
			projectName = a.project.ProjectName
		}
		a.mu.Unlock()
		if projectName != "" {
			memCtx := a.memoryStore.GetMemoryContext(projectName)
			statsCtx := redc.GetUsageStats(projectName)
			if memCtx != "" || statsCtx != "" {
				systemPrompt += "\n\n## 用户偏好与历史经验\n"
				if statsCtx != "" {
					systemPrompt += statsCtx
				}
				if memCtx != "" {
					systemPrompt += memCtx
				}
			}
		}
	}

	// Build ai.Message slice: system prompt + user-provided history
	aiMessages := make([]ai.Message, 0, len(messages)+1)
	aiMessages = append(aiMessages, ai.Message{Role: "system", Content: systemPrompt})
	for _, m := range messages {
		aiMessages = append(aiMessages, ai.Message{Role: m.Role, Content: m.Content})
	}

	// Build provider manager with failover support
	pm := buildProviderManager(aiConfig)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

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

// SmartAgentChatStream auto-classifies user intent and routes to the best specialized agent
func (a *App) SmartAgentChatStream(conversationId string, messages []AIChatMessage) error {
	intent := a.classifyIntent(messages)
	switch intent {
	case "deploy":
		return a.DeployAgentChatStream(conversationId, messages)
	case "troubleshoot":
		return a.TroubleshootAgentChatStream(conversationId, messages)
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

// ResumeAgentStream resumes an interrupted agent conversation from its last checkpoint
func (a *App) ResumeAgentStream(conversationId string) error {
	if a.memoryStore == nil {
		return fmt.Errorf("memory store not available")
	}
	cp, err := a.memoryStore.LoadCheckpoint(conversationId)
	if err != nil {
		return fmt.Errorf("无法恢复: %v", err)
	}

	// Deserialize checkpoint messages
	var aiMessages []ai.Message
	if err := json.Unmarshal([]byte(cp.Messages), &aiMessages); err != nil {
		return fmt.Errorf("checkpoint data corrupted: %v", err)
	}

	// Convert to AIChatMessage for the agent loop (only user/assistant messages)
	var chatMessages []AIChatMessage
	for _, m := range aiMessages {
		if m.Role == "user" || m.Role == "assistant" {
			chatMessages = append(chatMessages, AIChatMessage{Role: m.Role, Content: m.Content})
		}
	}

	// Notify frontend that resume is starting
	a.emitEvent("ai-chat-chunk", map[string]string{
		"conversationId": conversationId,
		"chunk":          fmt.Sprintf("\n\n🔄 从第 %d 轮检查点恢复执行...\n\n", cp.Round),
	})

	// Re-run agent loop with remaining rounds
	promptTemplate := cp.PromptTemplate
	if promptTemplate == "" {
		promptTemplate = ai.DeployAgentSystemPrompt
	}

	return a.runAgentLoop(conversationId, chatMessages, promptTemplate, 50, 15*time.Minute)
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

	// Inject agent memory context if enabled
	enableMemory := aiConfig.EnableMemory == nil || *aiConfig.EnableMemory // default true
	if enableMemory && a.memoryStore != nil {
		memCtx := a.memoryStore.GetMemoryContext(project.ProjectName)
		statsCtx := redc.GetUsageStats(project.ProjectName)
		if memCtx != "" || statsCtx != "" {
			systemPrompt += "\n\n## 用户偏好与历史经验\n"
			if statsCtx != "" {
				systemPrompt += statsCtx
			}
			if memCtx != "" {
				systemPrompt += memCtx
			}
		}

		// Inject recent incomplete tasks for checkpoint/resume
		recentTasks, err := a.memoryStore.GetRecentTasks(project.ProjectName, 3)
		if err == nil {
			var incompleteTasks []string
			for _, t := range recentTasks {
				if t.TaskStatus == "in_progress" {
					incompleteTasks = append(incompleteTasks, fmt.Sprintf("- [%s] %s (更新于 %s)", t.ConversationID[:8], t.TaskTitle, t.UpdatedAt))
				}
			}
			if len(incompleteTasks) > 0 {
				systemPrompt += "\n\n## 未完成任务（可能需要续接）\n"
				systemPrompt += "以下任务之前未完成，如用户提到继续或恢复，可参考其上下文：\n"
				for _, t := range incompleteTasks {
					systemPrompt += t + "\n"
				}
			}
		}
	}

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
	hooks.AddPostHook(ai.CostAnnotationPostHook)
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

	// Helper to save checkpoint at current state
	saveCheckpoint := func(round int) {
		if enableMemory && a.memoryStore != nil {
			if cpJSON, err := json.Marshal(aiMessages); err == nil {
				a.memoryStore.SaveCheckpoint(project.ProjectName, conversationId, string(cpJSON), round, promptTemplate)
			}
		}
	}

	// Agentic loop
	for round := 0; round < maxRounds; round++ {
		// Smart context window management: LLM-powered compaction when exceeding budget
		estimated := ai.EstimateTokens(aiMessages)
		if estimated > contextBudget {
			compactOpts := ai.CompactOptions{
				KeepRecentRounds: 4,
				ContextBudget:    contextBudget,
				MaxSummaryTokens: 2000,
			}
			aiMessages = ai.CompactWithLLM(ctx, client, aiMessages, compactOpts)
			gologger.Info().Msgf("agent: context compacted from ~%d to ~%d tokens (budget: %d)", estimated, ai.EstimateTokens(aiMessages), contextBudget)
		} else if round > 2 {
			// Light compression for older tool results even when under budget
			compressOldToolResults(aiMessages)
		}

		// Check if cancelled before each round
		if ctx.Err() != nil {
			saveCheckpoint(round)
			msg := "\n\n⏹️ 操作已被用户停止。"
			if ctx.Err() == context.DeadlineExceeded {
				msg = fmt.Sprintf("\n\n⏱️ 操作超时（已执行 %d 轮）。如需更长时间，请在 AI 配置中调整最大工具调用轮次。", round)
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
			saveCheckpoint(round)
			if ctx.Err() != nil {
				msg := "\n\n⏹️ 操作已被用户停止。"
				if ctx.Err() == context.DeadlineExceeded {
					msg = fmt.Sprintf("\n\n⏱️ 操作超时（已执行 %d 轮）。如需更长时间，请在 AI 配置中调整最大工具调用轮次。", round)
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
			// Extract memories asynchronously
			if enableMemory && a.memoryStore != nil {
				go a.extractMemories(aiConfig, aiMessages, project.ProjectName)
				a.memoryStore.CompleteTask(conversationId)
			}
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
						"question":      hookResult.Message,
						"choices":       []interface{}{"Yes, proceed", "No, cancel"},
						"allowFreeform": false,
					}, conversationId, ctx)
					if !strings.Contains(strings.ToLower(confirmResult), "yes") &&
						!strings.Contains(strings.ToLower(confirmResult), "proceed") {
						resultContent := "Operation cancelled by user."
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

		// Save checkpoint every 3 rounds for resumption after interruption
		if round > 0 && round%3 == 0 {
			saveCheckpoint(round)
		}
	}

	// Exceeded max rounds
	saveCheckpoint(maxRounds)
	// Extract memories asynchronously
	if enableMemory && a.memoryStore != nil {
		go a.extractMemories(aiConfig, aiMessages, project.ProjectName)
	}
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
			if len(content) > maxCompressedLen {
				messages[i].Content = content[:maxCompressedLen] + "... (compressed)"
			}
			compressed++
		}
	}
}

// compactMessages performs aggressive compression when context exceeds budget.
// Keeps: system prompt (first msg) + all user messages + last 3 rounds of full interactions.
// Compresses: older assistant messages to first 200 chars, older tool results to 100 chars.
func compactMessages(messages []ai.Message, budget int) []ai.Message {
	if len(messages) <= 4 {
		return messages
	}

	// Find boundary: keep last N tool+assistant messages at full length
	const keepRecentFull = 6 // last ~3 rounds (assistant + tool pairs)

	// Count non-system, non-user messages from the end
	recentCount := 0
	boundary := len(messages)
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "assistant" || messages[i].Role == "tool" {
			recentCount++
			if recentCount >= keepRecentFull {
				boundary = i
				break
			}
		}
	}

	result := make([]ai.Message, 0, len(messages))
	for i, m := range messages {
		if i == 0 && m.Role == "system" {
			result = append(result, m)
			continue
		}
		if m.Role == "user" {
			// Keep user messages full (they're usually short)
			result = append(result, m)
			continue
		}
		if i >= boundary {
			// Recent messages: keep full
			result = append(result, m)
			continue
		}
		// Older messages: compress
		compressed := m
		switch m.Role {
		case "assistant":
			if len(m.Content) > 200 {
				compressed.Content = m.Content[:200] + "... (compacted)"
			}
			// Strip tool_calls detail from old assistant messages
			if len(m.ToolCalls) > 0 {
				names := make([]string, len(m.ToolCalls))
				for j, tc := range m.ToolCalls {
					names[j] = tc.Function.Name
				}
				compressed.Content += fmt.Sprintf(" [called: %s]", strings.Join(names, ", "))
			}
		case "tool":
			if len(m.Content) > 100 {
				// For failed tools, keep more context
				if strings.Contains(m.Content, "失败") || strings.Contains(m.Content, "error") || strings.Contains(m.Content, "Error") {
					if len(m.Content) > 300 {
						compressed.Content = m.Content[:300] + "... (compacted)"
					}
				} else {
					compressed.Content = m.Content[:100] + "... (compacted)"
				}
			}
		}
		result = append(result, compressed)
	}

	return result
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

	// Persist task plan to memory store
	if a.memoryStore != nil {
		a.mu.Lock()
		proj := a.project
		a.mu.Unlock()
		if proj != nil {
			planJSON, _ := json.Marshal(args)
			a.memoryStore.SaveTaskPlan(proj.ProjectName, conversationId, title, string(planJSON))
		}
	}

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

// gatherTemplateContext reads template files and returns context for error analysis
func (a *App) gatherTemplateContext(templateName string) string {
	tmplPath, err := redc.GetTemplatePath(templateName)
	if err != nil {
		return ""
	}
	entries, err := os.ReadDir(tmplPath)
	if err != nil {
		return ""
	}

	var parts []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if name == "case.json" || strings.HasSuffix(name, ".tf") || name == "terraform.tfvars" {
			data, err := os.ReadFile(filepath.Join(tmplPath, name))
			if err != nil {
				continue
			}
			content := string(data)
			// Truncate very large files
			if len(content) > 3000 {
				content = content[:3000] + "\n... (truncated)"
			}
			parts = append(parts, fmt.Sprintf("### %s\n```\n%s\n```", name, content))
		}
	}

	if len(parts) == 0 {
		return ""
	}
	return "## 当前模板文件内容（供参考）\n\n" + strings.Join(parts, "\n\n")
}

const memoryExtractionPrompt = `你是一个经验提取器。根据以下 AI Agent 对话记录，提取值得记住的经验教训。

规则：
1. 只提取通用的、跨对话可复用的经验，不要提取一次性的具体操作细节（如具体的 IP、case ID）
2. 最多提取 5 条，没有值得记住的就返回空行
3. 每条经验一行，格式为 "category|content"，category 只能是 lesson、preference 或 failure
4. lesson: 操作中遇到的问题和解决方案、环境兼容性信息、工具使用技巧
5. preference: 用户表达的偏好（如常用的云厂商、模板、实例规格）
6. failure: 工具调用失败的结构化记录，格式为 "failure|[tool:工具名][error:错误类型][solution:解决方案]"

示例输出：
lesson|AWS t4g 系列是 ARM64 架构，VulHub 等 x86-only Docker 镜像应使用 aws/ec2-x86 模板
preference|用户偏好使用阿里云部署
failure|[tool:exec_command][error:apt lock][solution:等待 30-60 秒或 kill cloud-init 进程]

对话记录：
%s`

// extractMemories extracts reusable experience from a completed conversation
func (a *App) extractMemories(aiConfig *redc.AIConfig, messages []ai.Message, projectName string) {
	// Build conversation summary (user + assistant + failed tool messages, truncated)
	var summary strings.Builder
	for _, m := range messages {
		if m.Role == "system" {
			continue
		}
		content := m.Content
		// Include tool failures but skip successful tool results (too verbose)
		if m.Role == "tool" {
			if !strings.Contains(content, "失败") && !strings.Contains(content, "error") && !strings.Contains(content, "Error") {
				continue
			}
			content = fmt.Sprintf("[tool:%s] %s", m.Name, content)
		}
		if len(content) > 500 {
			content = content[:500] + "..."
		}
		summary.WriteString(fmt.Sprintf("[%s]: %s\n", m.Role, content))
		if summary.Len() > 5000 {
			break
		}
	}

	if summary.Len() < 100 {
		return // conversation too short, nothing to extract
	}

	prompt := fmt.Sprintf(memoryExtractionPrompt, summary.String())

	client := ai.NewClient(aiConfig.Provider, aiConfig.APIKey, aiConfig.BaseURL, aiConfig.Model)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := client.ChatWithTools(ctx, []ai.Message{
		{Role: "user", Content: prompt},
	}, nil)
	if err != nil || resp.Content == "" {
		return // silently ignore extraction failures
	}

	// Parse response lines
	for _, line := range strings.Split(resp.Content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 2)
		if len(parts) != 2 {
			continue
		}
		category := strings.TrimSpace(parts[0])
		content := strings.TrimSpace(parts[1])
		if category != "lesson" && category != "preference" && category != "failure" {
			continue
		}
		if content == "" {
			continue
		}
		a.memoryStore.AddMemory(projectName, category, content, "auto")
	}
}
