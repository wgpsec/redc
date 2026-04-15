package main

import (
	"encoding/json"
	"fmt"
	"strings"

	redc "red-cloud/mod"
	"red-cloud/mod/plugin"
)

// PluginInfo is a serializable view of a plugin for the frontend
type PluginInfo struct {
	Name          string                       `json:"name"`
	Version       string                       `json:"version"`
	Description   string                       `json:"description"`
	DescriptionEN string                       `json:"description_en"`
	Author        string                       `json:"author"`
	Homepage      string                       `json:"homepage"`
	Category      string                       `json:"category"`
	Tags          []string                     `json:"tags"`
	Enabled       bool                         `json:"enabled"`
	Dir           string                       `json:"dir"`
	ConfigSchema  map[string]plugin.ConfigField `json:"config_schema,omitempty"`
	Config        map[string]interface{}       `json:"config,omitempty"`
}

// ListPlugins returns all installed plugins
func (a *App) ListPlugins() ([]PluginInfo, error) {
	if a.pluginMgr == nil {
		return nil, fmt.Errorf("plugin manager not initialized")
	}

	// Reload to pick up any changes
	_ = a.pluginMgr.LoadAll()

	plugins := a.pluginMgr.List()
	result := make([]PluginInfo, 0, len(plugins))
	for _, p := range plugins {
		result = append(result, pluginToInfo(p))
	}
	return result, nil
}

// InstallPlugin installs a plugin from a git URL or local path
func (a *App) InstallPlugin(source string) error {
	if a.pluginMgr == nil {
		return fmt.Errorf("plugin manager not initialized")
	}
	_, err := a.pluginMgr.Install(source)
	return err
}

// UninstallPlugin removes a plugin by name
func (a *App) UninstallPlugin(name string) error {
	if a.pluginMgr == nil {
		return fmt.Errorf("plugin manager not initialized")
	}
	return a.pluginMgr.Uninstall(name)
}

// EnablePlugin enables a plugin
func (a *App) EnablePlugin(name string) error {
	if a.pluginMgr == nil {
		return fmt.Errorf("plugin manager not initialized")
	}
	return a.pluginMgr.Enable(name)
}

// DisablePlugin disables a plugin
func (a *App) DisablePlugin(name string) error {
	if a.pluginMgr == nil {
		return fmt.Errorf("plugin manager not initialized")
	}
	return a.pluginMgr.Disable(name)
}

// UpdatePlugin updates a plugin (git pull)
func (a *App) UpdatePlugin(name string) error {
	if a.pluginMgr == nil {
		return fmt.Errorf("plugin manager not initialized")
	}
	_, err := a.pluginMgr.Update(name)
	return err
}

// GetPluginConfig returns plugin config as JSON string
func (a *App) GetPluginConfig(name string) (string, error) {
	if a.pluginMgr == nil {
		return "", fmt.Errorf("plugin manager not initialized")
	}
	p, ok := a.pluginMgr.Get(name)
	if !ok {
		return "", fmt.Errorf("plugin %s not found", name)
	}
	data, err := json.MarshalIndent(p.Config, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// SavePluginConfig saves plugin config from JSON string
func (a *App) SavePluginConfig(name string, configJSON string) error {
	if a.pluginMgr == nil {
		return fmt.Errorf("plugin manager not initialized")
	}
	var config map[string]interface{}
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return fmt.Errorf("invalid config JSON: %w", err)
	}
	return a.pluginMgr.SaveConfig(name, config)
}

// setupPluginHooks wires plugin hooks into a Case instance
func (a *App) setupPluginHooks(c *redc.Case) {
	c.SetPluginHookRunner(func(hookPoint string, cc *redc.Case) error {
		outputJSON := ""
		if outputs, err := cc.TfOutput(); err == nil {
			data, _ := json.Marshal(outputs)
			outputJSON = string(data)
		}

		// Parse allowed plugins from case
		var allowed []string
		if cc.Plugins != "" {
			for _, p := range strings.Split(cc.Plugins, ",") {
				if name := strings.TrimSpace(p); name != "" {
					allowed = append(allowed, name)
				}
			}
		}

		// Parse case -var parameters into JSON
		caseVars := ""
		if len(cc.Parameter) > 0 {
			varsMap := make(map[string]string)
			for _, p := range cc.Parameter {
				if strings.HasPrefix(p, "-var") {
					continue
				}
				if idx := strings.Index(p, "="); idx > 0 {
					varsMap[p[:idx]] = p[idx+1:]
				}
			}
			if len(varsMap) > 0 {
				data, _ := json.Marshal(varsMap)
				caseVars = string(data)
			}
		}

		ctx := &plugin.HookContext{
			CaseName:       cc.Name,
			CasePath:       cc.Path,
			CaseTemplate:   cc.Type,
			CaseState:      cc.State,
			OutputJSON:     outputJSON,
			CaseVars:       caseVars,
			AllowedPlugins: allowed,
		}
		return a.pluginMgr.RunHooks(hookPoint, ctx)
	})
}

// GetPluginsDir returns the base plugins installation directory
func (a *App) GetPluginsDir() string {
	if a.pluginMgr != nil {
		return a.pluginMgr.PluginsDir()
	}
	if d, err := plugin.DefaultPluginsDir(); err == nil {
		return d
	}
	return ""
}

// FetchPluginRegistry fetches available plugins from the remote registry
func (a *App) FetchPluginRegistry() ([]plugin.RegistryPlugin, error) {
	index, err := plugin.FetchRegistry("")
	if err != nil {
		return nil, err
	}

	// Mark installed plugins
	installed := make(map[string]bool)
	if a.pluginMgr != nil {
		for _, p := range a.pluginMgr.List() {
			installed[p.Manifest.Name] = true
		}
	}
	_ = installed // frontend uses this info via ListPlugins

	return index.Plugins, nil
}

func pluginToInfo(p *plugin.Plugin) PluginInfo {
	return PluginInfo{
		Name:          p.Manifest.Name,
		Version:       p.Manifest.Version,
		Description:   p.Manifest.Description,
		DescriptionEN: p.Manifest.DescriptionEN,
		Author:        p.Manifest.Author,
		Homepage:      p.Manifest.Homepage,
		Category:      p.Manifest.Category,
		Tags:          p.Manifest.Tags,
		Enabled:       p.Enabled,
		Dir:           p.Dir,
		ConfigSchema:  p.Manifest.ConfigSchema,
		Config:        p.Config,
	}
}
