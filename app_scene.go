package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"red-cloud/i18n"
	redc "red-cloud/mod"
)

// parseTfvars parses a terraform.tfvars file
func parseTfvars(filePath string) (map[string]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	defaults := make(map[string]string)
	scanner := bufio.NewScanner(file)

	lineRegex := regexp.MustCompile(`^([a-zA-Z_][a-zA-Z0-9_]*)\s*=\s*"?([^"]*)"?`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if matches := lineRegex.FindStringSubmatch(line); len(matches) > 2 {
			defaults[matches[1]] = matches[2]
		}
	}

	return defaults, scanner.Err()
}

// StartCase starts a case by ID
func (a *App) StartCase(caseID string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.project == nil {
		return fmt.Errorf("%s", i18n.T("app_project_not_loaded"))
	}

	c, err := a.project.GetCase(caseID)
	if err != nil {
		return fmt.Errorf(i18n.Tf("app_get_case_failed", err))
	}

	if c == nil {
		return fmt.Errorf("%s", i18n.T("app_case_nil"))
	}

	if c.Path == "" {
		return fmt.Errorf("%s", i18n.T("app_case_path_empty"))
	}

	caseName := c.Name
	casePath := c.Path
	caseState := c.State

	a.emitLog(i18n.Tf("app_scene_prepare_start", caseName, casePath, caseState))

	go func() {
		defer func() {
			if r := recover(); r != nil {
				a.emitLog(i18n.Tf("app_scene_start_error", r))
			}
			a.emitRefresh()
		}()

		a.emitLog(i18n.Tf("app_scene_starting", caseName))
		if err := c.TfApply(); err != nil {
			a.emitLog(i18n.Tf("app_scene_start_failed", err))
			if a.notificationMgr != nil {
				a.notificationMgr.SendSceneFailed(caseName, "启动")
			}
			return
		}
		a.emitLog(i18n.Tf("app_scene_start_success", caseName))

		if a.notificationMgr != nil {
			a.notificationMgr.SendSceneStarted(caseName)
		}

		if outputs, err := c.TfOutput(); err == nil {
			for name, meta := range outputs {
				a.emitLog(fmt.Sprintf("  %s = %s", name, string(meta.Value)))
			}
		}
	}()

	return nil
}

// StopCase stops a case by ID
func (a *App) StopCase(caseID string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.project == nil {
		return fmt.Errorf("%s", i18n.T("app_project_not_loaded"))
	}

	c, err := a.project.GetCase(caseID)
	if err != nil {
		return err
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				a.emitLog(i18n.Tf("app_scene_stop_error", r))
			}
			a.emitRefresh()
		}()

		a.emitLog(i18n.Tf("app_stopping_scene", c.Name))
		if err := c.Stop(); err != nil {
			a.emitLog(i18n.Tf("app_scene_stop_failed", err))
			if a.notificationMgr != nil {
				a.notificationMgr.SendSceneFailed(c.Name, "停止")
			}
			return
		}
		a.emitLog(i18n.Tf("app_scene_stop_success", c.Name))

		if a.notificationMgr != nil {
			a.notificationMgr.SendSceneStopped(c.Name)
		}
	}()

	return nil
}

// RemoveCase removes a case by ID
func (a *App) RemoveCase(caseID string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.project == nil {
		return fmt.Errorf("%s", i18n.T("app_project_not_loaded"))
	}

	c, err := a.project.GetCase(caseID)
	if err != nil {
		return err
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				a.emitLog(i18n.Tf("app_scene_delete_error", r))
			}
			a.emitRefresh()
		}()

		a.emitLog(i18n.Tf("app_deleting_scene", c.Name))
		if err := c.Remove(); err != nil {
			a.emitLog(i18n.Tf("app_scene_delete_failed", err))
			return
		}
		a.emitLog(i18n.Tf("app_scene_delete_success", c.Name))
	}()

	return nil
}

// CreateCase creates a new case from a template (async)
func (a *App) CreateCase(templateName string, name string, vars map[string]string) error {
	a.mu.Lock()
	if a.project == nil {
		a.mu.Unlock()
		return fmt.Errorf("%s", i18n.T("app_project_not_loaded"))
	}
	project := a.project
	a.mu.Unlock()

	a.emitLog(i18n.Tf("app_creating_scene", name, templateName))

	go func() {
		defer func() {
			if r := recover(); r != nil {
				a.emitLog(i18n.Tf("app_scene_init_error", r))
			}
			a.emitRefresh()
		}()

		a.emitLog(i18n.Tf("app_scene_initing", name, templateName))
		c, err := project.CaseCreate(templateName, redc.U, name, vars)
		if err != nil {
			a.emitLog(i18n.Tf("app_scene_create_failed", err))
			return
		}
		a.emitLog(i18n.Tf("app_scene_create_success", c.Name, c.GetId()))
	}()

	return nil
}

// CreateAndRunCase creates a new case and immediately starts it (like CLI "run" command)
func (a *App) CreateAndRunCase(templateName string, name string, vars map[string]string) error {
	a.mu.Lock()
	if a.project == nil {
		a.mu.Unlock()
		return fmt.Errorf("%s", i18n.T("app_project_not_loaded"))
	}
	project := a.project
	a.mu.Unlock()

	a.emitLog(i18n.Tf("app_creating_running_scene", name, templateName))

	go func() {
		defer func() {
			if r := recover(); r != nil {
				a.emitLog(i18n.Tf("app_scene_init_error", r))
			}
			a.emitRefresh()
		}()

		a.emitLog(i18n.Tf("app_scene_initing", name, templateName))
		c, err := project.CaseCreate(templateName, redc.U, name, vars)
		if err != nil {
			a.emitLog(i18n.Tf("app_scene_create_failed", err))
			return
		}
		a.emitLog(i18n.Tf("app_scene_create_success", c.Name, c.GetId()))

		a.emitLog(i18n.Tf("app_scene_starting", c.Name))

		if err := c.TfApply(); err != nil {
			a.emitLog(i18n.Tf("app_scene_start_failed", err))
			return
		}

		a.emitLog(i18n.Tf("app_scene_start_success", c.Name))

		if outputs, err := c.TfOutput(); err == nil {
			for key, meta := range outputs {
				a.emitLog(fmt.Sprintf("%s = %s", key, string(meta.Value)))
			}
		}

		a.emitRefresh()
	}()

	return nil
}

// DeployCase creates and immediately starts a case (deprecated - use CreateCase then StartCase)
func (a *App) DeployCase(templateName string, name string, vars map[string]string) error {
	return a.CreateCase(templateName, name, vars)
}

// PlanResourceChange represents a single resource change in the plan
type PlanResourceChange struct {
	Address      string            `json:"address"`
	Type         string            `json:"type"`
	Name         string            `json:"name"`
	ProviderName string            `json:"providerName"`
	Actions      []string          `json:"actions"`
	IsData       bool              `json:"isData"`
	Detail       map[string]string `json:"detail,omitempty"`
}

// PlanEdge represents a dependency edge between two resources
type PlanEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// PlanTypeSummary summarizes resource count by type
type PlanTypeSummary struct {
	Type    string `json:"type"`
	Label   string `json:"label"`
	Count   int    `json:"count"`
	Actions string `json:"actions"`
}

// PlanPreview contains the full plan preview data for topology visualization
type PlanPreview struct {
	HasChanges  bool                 `json:"hasChanges"`
	ToCreate    int                  `json:"toCreate"`
	ToUpdate    int                  `json:"toUpdate"`
	ToDelete    int                  `json:"toDelete"`
	ToRecreate  int                  `json:"toRecreate"`
	Resources   []PlanResourceChange `json:"resources"`
	Edges       []PlanEdge           `json:"edges"`
	TypeSummary []PlanTypeSummary    `json:"typeSummary"`
}

// GetCasePlanPreview returns structured plan preview data for a case
func (a *App) GetCasePlanPreview(caseID string) (*PlanPreview, error) {
	a.mu.Lock()
	if a.project == nil {
		a.mu.Unlock()
		return nil, fmt.Errorf("%s", i18n.T("app_project_not_loaded"))
	}
	project := a.project
	a.mu.Unlock()

	c, err := project.GetCase(caseID)
	if err != nil {
		return nil, fmt.Errorf("failed to get case: %w", err)
	}
	if c == nil || c.Path == "" {
		return nil, fmt.Errorf("case not found or path is empty")
	}

	return buildPlanPreview(c.Path)
}

// GetDeploymentPlanPreview returns structured plan preview data for a custom deployment
func (a *App) GetDeploymentPlanPreview(deploymentID string) (*PlanPreview, error) {
	a.mu.Lock()
	if a.project == nil {
		a.mu.Unlock()
		return nil, fmt.Errorf("%s", i18n.T("app_project_not_loaded"))
	}
	project := a.project
	a.mu.Unlock()

	deploymentPath := filepath.Join(project.ProjectPath, deploymentID)
	if _, err := os.Stat(deploymentPath); err != nil {
		return nil, fmt.Errorf("deployment path not found")
	}

	return buildPlanPreview(deploymentPath)
}

// buildPlanPreview builds plan preview data from a terraform working directory
func buildPlanPreview(workDir string) (*PlanPreview, error) {
	planFile := filepath.Join(workDir, redc.RedcPlanPath)
	if _, err := os.Stat(planFile); err != nil {
		return nil, fmt.Errorf("plan file not found")
	}

	te, err := redc.NewTerraformExecutor(workDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create terraform executor: %w", err)
	}

	ctx, cancel := redc.CreateContextWithTimeout()
	defer cancel()

	preview := &PlanPreview{
		Resources:   []PlanResourceChange{},
		Edges:       []PlanEdge{},
		TypeSummary: []PlanTypeSummary{},
	}

	resourceChanges, err := te.GetPlanResourceChanges(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse plan: %w", err)
	}

	if resourceChanges == nil || len(resourceChanges) == 0 {
		return preview, nil
	}

	// For type summary
	typeCount := make(map[string]int)
	typeAction := make(map[string]string)
	typeOrder := []string{}

	preview.HasChanges = true
	for _, rc := range resourceChanges {
		actions := make([]string, len(rc.Change.Actions))
		for i, a := range rc.Change.Actions {
			actions[i] = string(a)
		}

		if len(actions) == 1 && actions[0] == "no-op" {
			continue
		}

		isData := strings.HasPrefix(rc.Address, "data.")
		prc := PlanResourceChange{
			Address:      rc.Address,
			Type:         rc.Type,
			Name:         rc.Name,
			ProviderName: rc.ProviderName,
			Actions:      actions,
			IsData:       isData,
			Detail:       extractResourceDetail(rc.Type, rc.Change.After),
		}
		preview.Resources = append(preview.Resources, prc)

		if len(actions) == 1 {
			switch actions[0] {
			case "create":
				preview.ToCreate++
			case "update":
				preview.ToUpdate++
			case "delete":
				preview.ToDelete++
			}
		} else if len(actions) == 2 && actions[0] == "delete" && actions[1] == "create" {
			preview.ToRecreate++
		}

		// Type summary
		if _, exists := typeCount[rc.Type]; !exists {
			typeOrder = append(typeOrder, rc.Type)
		}
		typeCount[rc.Type]++
		if _, exists := typeAction[rc.Type]; !exists {
			typeAction[rc.Type] = actions[0]
		}
	}

	// Build type summary in order of first appearance
	for _, t := range typeOrder {
		preview.TypeSummary = append(preview.TypeSummary, PlanTypeSummary{
			Type:    t,
			Label:   humanizeResourceType(t),
			Count:   typeCount[t],
			Actions: typeAction[t],
		})
	}

	dot, err := te.GetGraph(ctx)
	if err == nil && dot != "" {
		preview.Edges = parseDOTEdges(dot)
	}

	return preview, nil
}

// extractResourceDetail extracts key configuration values from plan after-values
func extractResourceDetail(resType string, after interface{}) map[string]string {
	m, ok := after.(map[string]interface{})
	if !ok || m == nil {
		return nil
	}

	detail := make(map[string]string)
	getString := func(key string) string {
		if v, ok := m[key]; ok && v != nil {
			return fmt.Sprintf("%v", v)
		}
		return ""
	}

	// Instance types
	if v := getString("instance_type"); v != "" {
		detail["instance_type"] = v
	}
	if v := getString("image_id"); v != "" {
		detail["image"] = v
	}
	if v := getString("ami"); v != "" {
		detail["image"] = v
	}

	// VPC / Subnet
	if v := getString("cidr_block"); v != "" {
		detail["cidr"] = v
	}

	// Security group rules (alicloud style)
	if v := getString("port_range"); v != "" {
		proto := getString("ip_protocol")
		policy := getString("policy")
		cidr := getString("cidr_ip")
		detail["rule"] = fmt.Sprintf("%s %s %s → %s", proto, v, policy, cidr)
	}

	// Security group (AWS style - embedded ingress/egress)
	if ingress, ok := m["ingress"]; ok && ingress != nil {
		detail["ingress"] = formatSGRules(ingress)
	}
	if egress, ok := m["egress"]; ok && egress != nil {
		detail["egress"] = formatSGRules(egress)
	}

	// Tencent cloud SG rule
	if v := getString("policy_index"); v != "" {
		proto := getString("ip_protocol")
		if proto == "" {
			proto = "all"
		}
		cidr := getString("cidr_ip")
		policy := getString("policy")
		ruleType := getString("type")
		detail["rule"] = fmt.Sprintf("%s %s %s %s → %s", ruleType, proto, policy, cidr, v)
	}

	if len(detail) == 0 {
		return nil
	}
	return detail
}

// formatSGRules formats AWS-style security group ingress/egress rules
func formatSGRules(rules interface{}) string {
	ruleList, ok := rules.([]interface{})
	if !ok || len(ruleList) == 0 {
		return ""
	}
	var parts []string
	for _, r := range ruleList {
		rm, ok := r.(map[string]interface{})
		if !ok {
			continue
		}
		proto := fmt.Sprintf("%v", rm["protocol"])
		fromPort := fmt.Sprintf("%v", rm["from_port"])
		toPort := fmt.Sprintf("%v", rm["to_port"])
		var cidrs []string
		if cb, ok := rm["cidr_blocks"].([]interface{}); ok {
			for _, c := range cb {
				cidrs = append(cidrs, fmt.Sprintf("%v", c))
			}
		}
		cidr := strings.Join(cidrs, ",")
		if cidr == "" {
			cidr = "*"
		}
		if proto == "-1" {
			parts = append(parts, fmt.Sprintf("all → %s", cidr))
		} else {
			parts = append(parts, fmt.Sprintf("%s:%s-%s → %s", proto, fromPort, toPort, cidr))
		}
	}
	return strings.Join(parts, "; ")
}

// humanizeResourceType converts terraform type to human-readable label
func humanizeResourceType(resType string) string {
	parts := strings.Split(resType, "_")
	if len(parts) <= 1 {
		return resType
	}
	var words []string
	for _, p := range parts[1:] {
		if len(p) > 0 {
			words = append(words, strings.ToUpper(p[:1])+p[1:])
		}
	}
	return strings.Join(words, " ")
}

// parseDOTEdges extracts edges from DOT format string
func parseDOTEdges(dot string) []PlanEdge {
	edgeRegex := regexp.MustCompile(`"([^"]+)"\s*->\s*"([^"]+)"`)
	seen := make(map[string]bool)
	var edges []PlanEdge
	for _, line := range strings.Split(dot, "\n") {
		matches := edgeRegex.FindStringSubmatch(line)
		if len(matches) == 3 {
			from := normalizeDOTNode(matches[1])
			to := normalizeDOTNode(matches[2])
			if from == "" || to == "" || from == to {
				continue
			}
			key := from + "->" + to
			if !seen[key] {
				seen[key] = true
				edges = append(edges, PlanEdge{From: from, To: to})
			}
		}
	}
	if edges == nil {
		edges = []PlanEdge{}
	}
	return edges
}

// normalizeDOTNode strips terraform graph prefixes/suffixes to get clean resource addresses
func normalizeDOTNode(name string) string {
	name = strings.TrimPrefix(name, "[root] ")
	name = strings.TrimSuffix(name, " (expand)")
	name = strings.TrimSuffix(name, " (close)")
	return name
}

// GetCaseOutputs returns the terraform outputs for a case
func (a *App) GetCaseOutputs(caseID string) (map[string]string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.project == nil {
		return nil, fmt.Errorf("%s", i18n.T("app_project_not_loaded"))
	}

	c, err := a.project.GetCase(caseID)
	if err != nil {
		return nil, err
	}

	if c.State != "running" {
		return nil, nil
	}

	outputs, err := c.TfOutput()
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for name, meta := range outputs {
		value := string(meta.Value)
		if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
			value = value[1 : len(value)-1]
		}
		if isRelativeFilePath(value) {
			absPath := filepath.Join(c.Path, value)
			if _, err := os.Stat(absPath); err == nil {
				value = absPath
			}
		}
		result[name] = value
	}

	if c.Module != "" && (strings.Contains(c.Module, "gen_clash_config") || strings.Contains(c.Module, "upload_r2")) {
		tfvarsPath := filepath.Join(c.Path, "terraform.tfvars")
		if _, err := os.Stat(tfvarsPath); err == nil {
			if tfvars, err := parseTfvars(tfvarsPath); err == nil {
				fileName := strings.TrimSpace(tfvars["filename"])
				if fileName == "" {
					fileName = "default-config.yaml"
				}
				localConfig := filepath.Join(c.Path, "config.yaml")
				if _, err := os.Stat(localConfig); err == nil {
					result["clash_config_local"] = localConfig
				}
				bucketName := strings.TrimSpace(tfvars["buckets_name"])
				if bucketName == "" {
					bucketName = "test"
				}
				bucketPath := strings.Trim(tfvars["buckets_path"], "/")
				r2Path := fmt.Sprintf("r2:%s/%s", bucketName, fileName)
				if bucketPath != "" {
					r2Path = fmt.Sprintf("r2:%s/%s/%s", bucketName, bucketPath, fileName)
				}
				result["clash_config_r2"] = r2Path
			}
		}
	}
	return result, nil
}

// isRelativeFilePath checks if the value looks like a relative file path
func isRelativeFilePath(value string) bool {
	if value == "" {
		return false
	}
	if strings.HasPrefix(value, "./") || strings.HasPrefix(value, "../") {
		return true
	}
	return false
}
