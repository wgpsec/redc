package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"red-cloud/mod/gologger"
)

// Valid hook points
const (
	HookPrePlan     = "pre-plan"
	HookPostPlan    = "post-plan"
	HookPreApply    = "pre-apply"
	HookPostApply   = "post-apply"
	HookPreDestroy  = "pre-destroy"
	HookPostDestroy = "post-destroy"
)

// HookEntry represents a single hook script to execute
type HookEntry struct {
	PluginName string
	ScriptPath string
	PluginDir  string
	Config     map[string]interface{}
}

// HookContext provides case information to hook scripts via environment variables
type HookContext struct {
	CaseName       string
	CasePath       string
	CaseTemplate   string
	CaseState      string
	OutputJSON     string // JSON-encoded terraform outputs
	CaseVars       string // JSON-encoded case parameters (terraform -var values)
	AllowedPlugins []string // if non-empty, only run hooks from these plugins
}

const pluginOutputsFile = "plugin_outputs.json"

// RunHooks executes all hook scripts for a given hook point
func (pm *PluginManager) RunHooks(hookPoint string, ctx *HookContext) error {
	hooks := pm.GetHooks(hookPoint)
	if len(hooks) == 0 {
		return nil
	}

	// If no plugins are allowed for this case, skip all hooks
	if ctx != nil && len(ctx.AllowedPlugins) == 0 {
		return nil
	}

	// Build allowed set and order map from context
	var allowed map[string]bool
	var orderMap map[string]int
	if ctx != nil && len(ctx.AllowedPlugins) > 0 {
		allowed = make(map[string]bool, len(ctx.AllowedPlugins))
		orderMap = make(map[string]int, len(ctx.AllowedPlugins))
		for i, name := range ctx.AllowedPlugins {
			trimmed := strings.TrimSpace(name)
			allowed[trimmed] = true
			orderMap[trimmed] = i
		}
	}

	// Sort hooks by declared order in redc_plugins
	if orderMap != nil {
		sort.SliceStable(hooks, func(i, j int) bool {
			oi, okI := orderMap[hooks[i].PluginName]
			oj, okJ := orderMap[hooks[j].PluginName]
			if !okI {
				oi = len(orderMap)
			}
			if !okJ {
				oj = len(orderMap)
			}
			return oi < oj
		})
	}

	// Accumulate outputs from all hooks
	allOutputs := make(map[string]string)

	// Load existing plugin outputs (from prior hooks)
	if ctx != nil && ctx.CasePath != "" {
		if existing := LoadPluginOutputs(ctx.CasePath); existing != nil {
			for k, v := range existing {
				allOutputs[k] = v
			}
		}
	}

	for _, hook := range hooks {
		// Skip plugins not in allowed list
		if allowed != nil && !allowed[hook.PluginName] {
			continue
		}

		gologger.Info().Msgf("plugin: running %s hook from %s", hookPoint, hook.PluginName)
		hookOutputs, err := executeHook(hook, hookPoint, ctx)
		if err != nil {
			gologger.Warning().Msgf("plugin: %s hook from %s failed: %v (continuing)", hookPoint, hook.PluginName, err)
			// Continue to next hook instead of aborting
			continue
		}
		for k, v := range hookOutputs {
			allOutputs[k] = v
		}
		gologger.Info().Msgf("plugin: %s hook from %s completed", hookPoint, hook.PluginName)
	}

	// Persist accumulated outputs
	if ctx != nil && ctx.CasePath != "" && len(allOutputs) > 0 {
		savePluginOutputs(ctx.CasePath, allOutputs)
	}

	return nil
}

func executeHook(hook HookEntry, hookPoint string, hctx *HookContext) (map[string]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", hook.ScriptPath)
	cmd.Dir = hook.PluginDir

	// Build environment
	env := os.Environ()
	env = append(env,
		"REDC_HOOK_POINT="+hookPoint,
		"REDC_PLUGIN_NAME="+hook.PluginName,
		"REDC_PLUGIN_DIR="+hook.PluginDir,
	)

	if hctx != nil {
		env = append(env,
			"REDC_CASE_NAME="+hctx.CaseName,
			"REDC_CASE_PATH="+hctx.CasePath,
			"REDC_CASE_TEMPLATE="+hctx.CaseTemplate,
			"REDC_CASE_STATE="+hctx.CaseState,
		)
		if hctx.OutputJSON != "" {
			env = append(env, "REDC_OUTPUT_JSON="+hctx.OutputJSON)
		}
		if hctx.CaseVars != "" {
			env = append(env, "REDC_CASE_VARS="+hctx.CaseVars)
		}
	}

	// Inject plugin config as REDC_PLUGIN_CONFIG_<KEY>
	if hook.Config != nil {
		configJSON, _ := json.Marshal(hook.Config)
		env = append(env, "REDC_PLUGIN_CONFIG="+string(configJSON))
		for k, v := range hook.Config {
			key := "REDC_PLUGIN_CONFIG_" + strings.ToUpper(strings.ReplaceAll(k, "-", "_"))
			env = append(env, key+"="+fmt.Sprintf("%v", v))
		}
	}

	cmd.Env = env

	out, err := cmd.CombinedOutput()
	output := strings.TrimSpace(string(out))

	// Parse REDC_OUTPUT:key=value lines from stdout
	parsedOutputs := make(map[string]string)
	if output != "" {
		for _, line := range strings.Split(output, "\n") {
			if strings.HasPrefix(line, "REDC_OUTPUT:") {
				kv := strings.TrimPrefix(line, "REDC_OUTPUT:")
				if idx := strings.Index(kv, "="); idx > 0 {
					parsedOutputs[strings.TrimSpace(kv[:idx])] = strings.TrimSpace(kv[idx+1:])
					continue
				}
			}
			gologger.Info().Msgf("plugin[%s]: %s", hook.PluginName, line)
		}
	}

	if ctx.Err() == context.DeadlineExceeded {
		return parsedOutputs, fmt.Errorf("hook script timed out after 5 minutes")
	}

	return parsedOutputs, err
}

// LoadPluginOutputs reads plugin_outputs.json from the case directory
func LoadPluginOutputs(casePath string) map[string]string {
	data, err := os.ReadFile(filepath.Join(casePath, pluginOutputsFile))
	if err != nil {
		return nil
	}
	var outputs map[string]string
	if json.Unmarshal(data, &outputs) != nil {
		return nil
	}
	return outputs
}

// savePluginOutputs writes plugin_outputs.json to the case directory
func savePluginOutputs(casePath string, outputs map[string]string) {
	data, err := json.MarshalIndent(outputs, "", "  ")
	if err != nil {
		gologger.Warning().Msgf("plugin: failed to marshal outputs: %v", err)
		return
	}
	if err := os.WriteFile(filepath.Join(casePath, pluginOutputsFile), data, 0644); err != nil {
		gologger.Warning().Msgf("plugin: failed to write outputs: %v", err)
	}
}
