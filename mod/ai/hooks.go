package ai

import (
	"fmt"
	"strings"
)

// ToolHookAction represents what to do after a hook runs.
type ToolHookAction int

const (
	// HookContinue means proceed with tool execution.
	HookContinue ToolHookAction = iota
	// HookBlock means abort the tool call and return the message.
	HookBlock
	// HookConfirm means ask the user for confirmation before proceeding.
	HookConfirm
)

// HookResult is returned by a hook to indicate the desired action.
type HookResult struct {
	Action  ToolHookAction
	Message string // Explanation or confirmation prompt
}

// PreToolHook is called before a tool is executed.
// It receives the tool name and parsed arguments; returns a HookResult.
type PreToolHook func(toolName string, args map[string]interface{}) HookResult

// PostToolHook is called after a tool returns.
type PostToolHook func(toolName string, args map[string]interface{}, result string, success bool) string

// HookChain manages an ordered list of pre/post tool hooks.
type HookChain struct {
	preHooks  []PreToolHook
	postHooks []PostToolHook
}

// NewHookChain creates a HookChain with the standard built-in hooks.
func NewHookChain() *HookChain {
	hc := &HookChain{}
	hc.preHooks = append(hc.preHooks, DangerousOpHook)
	hc.postHooks = append(hc.postHooks, CostAnnotationPostHook)
	return hc
}

// AddPreHook appends a custom pre-tool hook.
func (hc *HookChain) AddPreHook(h PreToolHook) {
	hc.preHooks = append(hc.preHooks, h)
}

// AddPostHook appends a custom post-tool hook.
func (hc *HookChain) AddPostHook(h PostToolHook) {
	hc.postHooks = append(hc.postHooks, h)
}

// RunPreHooks runs all pre-hooks in order. Returns the first non-Continue result, or Continue.
func (hc *HookChain) RunPreHooks(toolName string, args map[string]interface{}) HookResult {
	for _, h := range hc.preHooks {
		r := h(toolName, args)
		if r.Action != HookContinue {
			return r
		}
	}
	return HookResult{Action: HookContinue}
}

// RunPostHooks runs all post-hooks in order. Returns modified result text.
func (hc *HookChain) RunPostHooks(toolName string, args map[string]interface{}, result string, success bool) string {
	for _, h := range hc.postHooks {
		if annotation := h(toolName, args, result, success); annotation != "" {
			result += "\n" + annotation
		}
	}
	return result
}

// --- Built-in Hooks ---

// dangerousTools lists tools that perform destructive/irreversible operations.
var dangerousTools = map[string]string{
	"kill_case":       "Force-kill a running case (resources may not be cleaned up)",
	"delete_template": "Permanently delete a local template",
	"compose_down":    "Destroy all services in a compose deployment",
	"stop_case":       "Stop a running case (terraform destroy)",
}

// DangerousOpHook asks for user confirmation before destructive operations.
func DangerousOpHook(toolName string, args map[string]interface{}) HookResult {
	desc, isDangerous := dangerousTools[toolName]
	if !isDangerous {
		return HookResult{Action: HookContinue}
	}

	target := ""
	if caseID, ok := args["case_id"].(string); ok && caseID != "" {
		if len(caseID) > 12 {
			target = caseID[:12] + "..."
		} else {
			target = caseID
		}
	} else if tmpl, ok := args["template"].(string); ok && tmpl != "" {
		target = tmpl
	}

	msg := fmt.Sprintf("⚠️ Dangerous operation: %s (%s)", toolName, desc)
	if target != "" {
		msg += fmt.Sprintf("\nTarget: %s", target)
	}
	msg += "\nConfirm execution?"

	return HookResult{
		Action:  HookConfirm,
		Message: msg,
	}
}

// costlyTools lists tools that may incur cloud costs.
var costlyTools = map[string]string{
	"start_case":  "Starts cloud resources (billing begins)",
	"plan_case":   "Creates a case that will provision cloud resources when started",
	"compose_up":  "Deploys multiple cloud services (billing begins for all)",
}

// CostAwareHook adds a cost warning annotation to tool calls that may incur charges.
func CostAwareHook(toolName string, args map[string]interface{}) HookResult {
	desc, isCostly := costlyTools[toolName]
	if !isCostly {
		return HookResult{Action: HookContinue}
	}

	// For start_case/compose_up, warn but don't block (agent already got user intent)
	_ = desc
	return HookResult{Action: HookContinue}
}

// CostAnnotationPostHook adds cost context to the result of cost-incurring tools.
func CostAnnotationPostHook(toolName string, args map[string]interface{}, result string, success bool) string {
	if !success {
		return ""
	}
	if _, isCostly := costlyTools[toolName]; !isCostly {
		return ""
	}
	return "💰 Note: This operation may incur cloud costs. Use get_balances or get_predicted_monthly_cost to check."
}

// SafetyRefusalPostHook detects AI safety refusal patterns in tool output and annotates.
var safetyRefusalPatterns = []string{
	"I cannot",
	"I'm not able to",
	"I apologize",
	"As an AI",
	"I must decline",
	"against my guidelines",
	"I'm unable to",
}

// DetectSafetyRefusal checks if a text contains safety refusal patterns.
func DetectSafetyRefusal(text string) bool {
	lower := strings.ToLower(text)
	for _, pattern := range safetyRefusalPatterns {
		if strings.Contains(lower, strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}
