package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"red-cloud/i18n"
	redc "red-cloud/mod"
	"red-cloud/utils/sshutil"
)

// ansiRegex matches ANSI escape sequences for stripping from terminal output
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// F8xInstallRecord records a f8x installation
type F8xInstallRecord struct {
	ID         string `json:"id"`
	CaseID     string `json:"caseID"`
	Flags      string `json:"flags"`
	Status     string `json:"status"` // running, success, failed
	Output     string `json:"output"`
	StartedAt  string `json:"startedAt"`
	FinishedAt string `json:"finishedAt,omitempty"`
}

// F8xStatus represents f8x deployment status on a VPS
type F8xStatus struct {
	Deployed bool   `json:"deployed"`
	Version  string `json:"version,omitempty"`
	Error    string `json:"error,omitempty"`
}

var (
	f8xInstallHistory   []F8xInstallRecord
	f8xInstallHistoryMu sync.Mutex
	f8xRunningTasks     = make(map[string]*F8xInstallRecord)
	f8xRunningTasksMu   sync.Mutex
)

// GetF8xCatalog returns the full f8x tool catalog
func (a *App) GetF8xCatalog() []redc.F8xModule {
	return redc.GetF8xCatalog()
}

// GetF8xCategories returns category metadata with counts
func (a *App) GetF8xCategories() []redc.F8xCategoryInfo {
	return redc.GetF8xCategories()
}

// GetF8xPresets returns preset install combinations
func (a *App) GetF8xPresets() []redc.F8xPreset {
	return redc.GetF8xPresets()
}

// GetF8xTools returns the list of individually installable tools
func (a *App) GetF8xTools() []redc.F8xTool {
	return redc.GetF8xTools()
}

// GetInstalledTools detects installed tools on a target VPS via SSH
func (a *App) GetInstalledTools(caseID string) (*redc.F8xInstalledFile, error) {
	sshConfig, err := a.getSSHConfigForCase(caseID)
	if err != nil {
		return nil, fmt.Errorf("SSH config error: %v", err)
	}

	// Try reading the installed.json file first
	result := a.execSSHCommand(sshConfig, "cat /opt/.f8x/installed.json 2>/dev/null")
	if result.Success && strings.TrimSpace(result.Stdout) != "" {
		var installed redc.F8xInstalledFile
		if err := json.Unmarshal([]byte(result.Stdout), &installed); err == nil {
			return &installed, nil
		}
	}

	// Fallback: detect common tools via which
	tools := redc.GetF8xTools()
	if len(tools) == 0 {
		return &redc.F8xInstalledFile{Tools: map[string]redc.F8xInstalledInfo{}}, nil
	}

	// Batch check in groups of 50
	installed := &redc.F8xInstalledFile{
		Tools:       make(map[string]redc.F8xInstalledInfo),
		LastUpdated: time.Now().UTC().Format(time.RFC3339),
	}

	batchSize := 50
	for i := 0; i < len(tools); i += batchSize {
		end := i + batchSize
		if end > len(tools) {
			end = len(tools)
		}
		var names []string
		for _, t := range tools[i:end] {
			names = append(names, t.ID)
		}
		cmd := "for t in " + strings.Join(names, " ") + "; do command -v $t >/dev/null 2>&1 && echo $t; done"
		checkResult := a.execSSHCommand(sshConfig, cmd)
		if checkResult.Success {
			for _, line := range strings.Split(strings.TrimSpace(checkResult.Stdout), "\n") {
				line = strings.TrimSpace(line)
				if line != "" {
					installed.Tools[line] = redc.F8xInstalledInfo{
						InstalledAt: "unknown",
						Method:      "detected",
					}
				}
			}
		}
	}

	return installed, nil
}

// InstallF8xTool installs a single tool on a target VPS via f8x -install
func (a *App) InstallF8xTool(caseID string, toolID string) string {
	return a.RunF8xInstall(caseID, []string{"-install", toolID})
}

// RefreshF8xCatalog forces a refresh of the remote catalog cache
func (a *App) RefreshF8xCatalog() map[string]interface{} {
	redc.InvalidateF8xCache()
	remote, err := redc.FetchF8xRemoteCatalog()
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   err.Error(),
			"source":  "none",
			"count":   0,
		}
	}
	return map[string]interface{}{
		"success":   true,
		"source":    "remote",
		"version":   remote.Version,
		"updatedAt": remote.UpdatedAt,
		"count":     len(remote.Modules),
	}
}

// GetF8xStatus checks if f8x is deployed on a VPS
func (a *App) GetF8xStatus(caseID string) F8xStatus {
	sshConfig, err := a.getSSHConfigForCase(caseID)
	if err != nil {
		return F8xStatus{Error: err.Error()}
	}

	// Check f8x status: try which, known paths, and /ffffffff0x work dir
	sudo := ""
	if sshConfig.User != "root" {
		sudo = "sudo "
	}
	checkCmd := fmt.Sprintf(
		`%ssh -c 'for p in $(which f8x 2>/dev/null) /usr/local/bin/f8x /tmp/f8x; do [ -f "$p" ] && head -c 500 "$p" 2>/dev/null | grep -o "F8x_Version=\"[^\"]*\"" && exit 0; done; [ -d /ffffffff0x ] && echo "DEPLOYED" || echo "NOT_FOUND"'`,
		sudo)
	result := a.execSSHCommand(sshConfig, checkCmd)

	output := strings.TrimSpace(result.Stdout)
	if output == "" || strings.Contains(output, "NOT_FOUND") {
		return F8xStatus{Deployed: false}
	}
	if output == "DEPLOYED" {
		return F8xStatus{Deployed: true, Version: "installed"}
	}

	version := ""
	if strings.Contains(output, "F8x_Version=") {
		version = strings.Trim(strings.SplitN(output, "=", 2)[1], "\"")
	}
	return F8xStatus{Deployed: true, Version: version}
}

// EnsureF8x deploys f8x to target VPS if not present
func (a *App) EnsureF8x(caseID string) ExecCommandResult {
	sshConfig, err := a.getSSHConfigForCase(caseID)
	if err != nil {
		return ExecCommandResult{Error: err.Error(), Success: false}
	}

	// Check if already present (both deploy and install locations)
	checkResult := a.execSSHCommand(sshConfig, "test -f /tmp/f8x -o -f /usr/local/bin/f8x && echo 'EXISTS' || echo 'MISSING'")
	if checkResult.Success && strings.Contains(checkResult.Stdout, "EXISTS") {
		return ExecCommandResult{Success: true, Stdout: "f8x already deployed"}
	}

	// Download f8x from CDN
	cmd := fmt.Sprintf("wget -q -O /tmp/f8x %s && chmod +x /tmp/f8x && echo 'OK'", redc.F8xDefaultURL)
	result := a.execSSHCommand(sshConfig, cmd)
	if result.Success && strings.Contains(result.Stdout, "OK") {
		return ExecCommandResult{Success: true, Stdout: "f8x deployed successfully"}
	}

	// Fallback to GitHub raw
	cmd = fmt.Sprintf("wget -q -O /tmp/f8x %s && chmod +x /tmp/f8x && echo 'OK'", redc.F8xFallbackURL)
	result = a.execSSHCommand(sshConfig, cmd)
	if result.Success && strings.Contains(result.Stdout, "OK") {
		return ExecCommandResult{Success: true, Stdout: "f8x deployed (fallback)"}
	}

	// Try curl as last resort
	cmd = fmt.Sprintf("curl -sL -o /tmp/f8x %s && chmod +x /tmp/f8x && echo 'OK'", redc.F8xDefaultURL)
	result = a.execSSHCommand(sshConfig, cmd)
	if result.Success && strings.Contains(result.Stdout, "OK") {
		return ExecCommandResult{Success: true, Stdout: "f8x deployed (curl)"}
	}

	return ExecCommandResult{Error: i18n.T("f8x_deploy_failed"), Success: false}
}

// BuildF8xCommand returns the shell command string for f8x installation.
// The command includes: f8x download (if needed) and the install flags.
// User can interact with f8x prompts directly in the SSH terminal.
func (a *App) BuildF8xCommand(flags []string) string {
	flagStr := strings.Join(flags, " ")
	return fmt.Sprintf(
		"test -f /tmp/f8x -o -f /usr/local/bin/f8x || (wget -q -O /tmp/f8x %s || curl -sSL -o /tmp/f8x %s) && chmod +x /tmp/f8x 2>/dev/null; "+
			"F8X=$(which f8x 2>/dev/null || test -f /tmp/f8x && echo /tmp/f8x || echo /usr/local/bin/f8x) && "+
			"sudo bash \"$F8X\" %s",
		redc.F8xDefaultURL, redc.F8xFallbackURL, flagStr,
	)
}

// RunF8xInstall executes f8x with given flags on target VPS
// Returns taskID for tracking; output is streamed via events
func (a *App) RunF8xInstall(caseID string, flags []string) string {
	taskID := fmt.Sprintf("f8x-%s-%d", caseID, time.Now().UnixMilli())

	record := &F8xInstallRecord{
		ID:        taskID,
		CaseID:    caseID,
		Flags:     strings.Join(flags, " "),
		Status:    "running",
		StartedAt: time.Now().Format(time.RFC3339),
	}

	f8xRunningTasksMu.Lock()
	f8xRunningTasks[taskID] = record
	f8xRunningTasksMu.Unlock()

	go a.runF8xInstallAsync(taskID, caseID, flags, record)

	return taskID
}

func (a *App) runF8xInstallAsync(taskID, caseID string, flags []string, record *F8xInstallRecord) {
	defer func() {
		f8xRunningTasksMu.Lock()
		delete(f8xRunningTasks, taskID)
		f8xRunningTasksMu.Unlock()

		f8xInstallHistoryMu.Lock()
		f8xInstallHistory = append(f8xInstallHistory, *record)
		// Keep last 100 records
		if len(f8xInstallHistory) > 100 {
			f8xInstallHistory = f8xInstallHistory[len(f8xInstallHistory)-100:]
		}
		f8xInstallHistoryMu.Unlock()
	}()

	// Ensure f8x is deployed
	a.emitEvent("f8x:output", map[string]interface{}{
		"taskID": taskID, "type": "info",
		"text": "Checking f8x deployment...",
	})

	ensureResult := a.EnsureF8x(caseID)
	if !ensureResult.Success {
		record.Status = "failed"
		record.Output = ensureResult.Error
		record.FinishedAt = time.Now().Format(time.RFC3339)
		a.emitEvent("f8x:output", map[string]interface{}{
			"taskID": taskID, "type": "error",
			"text": "Failed to deploy f8x: " + ensureResult.Error,
		})
		a.emitEvent("f8x:done", map[string]interface{}{
			"taskID": taskID, "status": "failed",
		})
		return
	}

	// Get SSH config and run f8x with streaming output
	sshConfig, err := a.getSSHConfigForCase(caseID)
	if err != nil {
		record.Status = "failed"
		record.Output = err.Error()
		record.FinishedAt = time.Now().Format(time.RFC3339)
		a.emitEvent("f8x:done", map[string]interface{}{
			"taskID": taskID, "status": "failed",
		})
		return
	}

	// touch /tmp/IS_CI to skip interactive prompts (f8x CI mode)
	// Use sudo for non-root users (e.g., AWS ec2-user, admin)
	// f8x may be at /tmp/f8x (freshly deployed) or /usr/local/bin/f8x (self-installed)
	flagStr := strings.Join(flags, " ")
	sudo := ""
	if sshConfig.User != "root" {
		sudo = "sudo "
	}
	cmd := fmt.Sprintf("touch /tmp/IS_CI && F8X=$(which f8x 2>/dev/null || test -f /tmp/f8x && echo /tmp/f8x || echo /usr/local/bin/f8x) && %sbash \"$F8X\" %s", sudo, flagStr)

	a.emitEvent("f8x:output", map[string]interface{}{
		"taskID": taskID, "type": "info",
		"text": fmt.Sprintf("Running: %s", cmd),
	})

	client, err := sshutil.NewClient(sshConfig)
	if err != nil {
		record.Status = "failed"
		record.Output = err.Error()
		record.FinishedAt = time.Now().Format(time.RFC3339)
		a.emitEvent("f8x:done", map[string]interface{}{
			"taskID": taskID, "status": "failed",
		})
		return
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		record.Status = "failed"
		record.Output = err.Error()
		record.FinishedAt = time.Now().Format(time.RFC3339)
		a.emitEvent("f8x:done", map[string]interface{}{
			"taskID": taskID, "status": "failed",
		})
		return
	}
	defer session.Close()

	// Stream stdout
	stdout, _ := session.StdoutPipe()
	stderr, _ := session.StderrPipe()

	if err := session.Start(cmd); err != nil {
		record.Status = "failed"
		record.Output = err.Error()
		record.FinishedAt = time.Now().Format(time.RFC3339)
		a.emitEvent("f8x:done", map[string]interface{}{
			"taskID": taskID, "status": "failed",
		})
		return
	}

	var outputBuf strings.Builder

	// Read stdout in goroutine
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := stdout.Read(buf)
			if n > 0 {
				text := ansiRegex.ReplaceAllString(string(buf[:n]), "")
				outputBuf.WriteString(text)
				a.emitEvent("f8x:output", map[string]interface{}{
					"taskID": taskID, "type": "stdout", "text": text,
				})
			}
			if err != nil {
				break
			}
		}
	}()

	// Read stderr in goroutine
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := stderr.Read(buf)
			if n > 0 {
				text := ansiRegex.ReplaceAllString(string(buf[:n]), "")
				outputBuf.WriteString(text)
				a.emitEvent("f8x:output", map[string]interface{}{
					"taskID": taskID, "type": "stderr", "text": text,
				})
			}
			if err != nil {
				break
			}
		}
	}()

	// Wait for completion
	err = session.Wait()
	record.FinishedAt = time.Now().Format(time.RFC3339)

	// Keep last 10000 chars of output
	output := outputBuf.String()
	if len(output) > 10000 {
		output = output[len(output)-10000:]
	}
	record.Output = output

	if err != nil {
		record.Status = "failed"
		a.emitEvent("f8x:done", map[string]interface{}{
			"taskID": taskID, "status": "failed", "error": err.Error(),
		})
	} else {
		record.Status = "success"
		a.emitEvent("f8x:done", map[string]interface{}{
			"taskID": taskID, "status": "success",
		})
	}
}

// GetF8xInstallHistory returns install history for a case (or all)
func (a *App) GetF8xInstallHistory(caseID string) []F8xInstallRecord {
	f8xInstallHistoryMu.Lock()
	defer f8xInstallHistoryMu.Unlock()

	if caseID == "" {
		result := make([]F8xInstallRecord, len(f8xInstallHistory))
		copy(result, f8xInstallHistory)
		return result
	}

	var filtered []F8xInstallRecord
	for _, r := range f8xInstallHistory {
		if r.CaseID == caseID {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

// GetF8xRunningTasks returns currently running f8x install tasks
func (a *App) GetF8xRunningTasks() []F8xInstallRecord {
	f8xRunningTasksMu.Lock()
	defer f8xRunningTasksMu.Unlock()

	var result []F8xInstallRecord
	for _, r := range f8xRunningTasks {
		result = append(result, *r)
	}
	return result
}

// helper to get SSH config for a case (or custom deployment)
func (a *App) getSSHConfigForCase(caseID string) (*sshutil.SSHConfig, error) {
	a.mu.Lock()
	project := a.project
	service := a.customDeploymentService
	a.mu.Unlock()

	if project == nil {
		return nil, fmt.Errorf(i18n.T("app_project_not_loaded"))
	}

	c, caseErr := project.GetCase(caseID)
	if caseErr == nil {
		return c.GetSSHConfig()
	}

	if service != nil {
		return a.getDeploymentSSHConfig(caseID)
	}

	return nil, fmt.Errorf(i18n.T("app_case_not_found"), caseErr)
}
