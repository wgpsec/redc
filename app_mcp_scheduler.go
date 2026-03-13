package main

import (
	"fmt"
	"red-cloud/i18n"
	redc "red-cloud/mod"
	"red-cloud/mod/mcp"
	"time"
)

func (a *App) GetMCPStatus() MCPStatus {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.mcpManager == nil {
		return MCPStatus{Running: false}
	}

	status := a.mcpManager.GetStatus()
	return MCPStatus{
		Running:         status["running"].(bool),
		Mode:            status["mode"].(string),
		Address:         status["address"].(string),
		ProtocolVersion: status["protocolVersion"].(string),
	}
}

func (a *App) StartMCPServer(mode string, address string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.project == nil {
		return fmt.Errorf("%s", i18n.T("app_project_not_loaded"))
	}

	// Create manager if not exists
	if a.mcpManager == nil {
		a.mcpManager = mcp.NewMCPServerManager(a.project, a)
		a.mcpManager.SetLogCallback(a.emitLog)
	}

	// Convert mode string to TransportMode
	var transportMode mcp.TransportMode
	switch mode {
	case "sse":
		transportMode = mcp.TransportSSE
	case "stdio":
		transportMode = mcp.TransportSTDIO
	default:
		return fmt.Errorf(i18n.Tf("app_mcp_unknown_mode", mode))
	}

	if err := a.mcpManager.Start(transportMode, address); err != nil {
		return fmt.Errorf(i18n.Tf("app_mcp_start_failed", err))
	}

	a.emitLog(i18n.Tf("app_mcp_started", mode, address))
	return nil
}

func (a *App) StopMCPServer() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.mcpManager == nil {
		return fmt.Errorf("%s", i18n.T("app_mcp_not_init"))
	}

	if err := a.mcpManager.Stop(); err != nil {
		return fmt.Errorf(i18n.Tf("app_mcp_stop_failed", err))
	}

	a.emitLog(i18n.T("app_mcp_stopped"))
	return nil
}

func (a *App) ScheduleTask(caseID string, caseName string, action string, scheduledAt time.Time) (*redc.ScheduledTask, error) {
	return a.ScheduleTaskWithRepeat(caseID, caseName, action, scheduledAt, "once", 0)
}

func (a *App) ScheduleTaskWithRepeat(caseID string, caseName string, action string, scheduledAt time.Time, repeatType string, repeatInterval int) (*redc.ScheduledTask, error) {
	return a.ScheduleTaskFull(caseID, caseName, action, scheduledAt, repeatType, repeatInterval, "", false)
}

func (a *App) ScheduleTaskFull(caseID string, caseName string, action string, scheduledAt time.Time, repeatType string, repeatInterval int, sshCommand string, notifyEnabled bool) (*redc.ScheduledTask, error) {
	a.mu.Lock()
	scheduler := a.taskScheduler
	a.mu.Unlock()

	if scheduler == nil {
		return nil, fmt.Errorf("%s", i18n.T("app_scheduler_not_init"))
	}

	task, err := scheduler.AddTaskFull(caseID, caseName, action, scheduledAt, repeatType, repeatInterval, sshCommand, notifyEnabled)
	if err != nil {
		return nil, err
	}

	a.emitLog(i18n.Tf("app_cron_created", caseName, scheduledAt.Format("2006-01-02 15:04:05"), action))

	return task, nil
}

func (a *App) CancelScheduledTask(taskID string) error {
	a.mu.Lock()
	scheduler := a.taskScheduler
	a.mu.Unlock()

	if scheduler == nil {
		return fmt.Errorf("%s", i18n.T("app_scheduler_not_init"))
	}

	err := scheduler.CancelTask(taskID)
	if err != nil {
		return err
	}

	a.emitLog(i18n.Tf("app_cron_cancelled", taskID))
	return nil
}

func (a *App) GetScheduledTask(taskID string) (*redc.ScheduledTask, error) {
	a.mu.Lock()
	scheduler := a.taskScheduler
	a.mu.Unlock()

	if scheduler == nil {
		return nil, fmt.Errorf("%s", i18n.T("app_scheduler_not_init"))
	}

	return scheduler.GetTask(taskID)
}

func (a *App) ListScheduledTasks() []*redc.ScheduledTask {
	a.mu.Lock()
	scheduler := a.taskScheduler
	a.mu.Unlock()

	if scheduler == nil {
		return []*redc.ScheduledTask{}
	}

	return scheduler.ListTasks()
}

func (a *App) ListCaseScheduledTasks(caseID string) []*redc.ScheduledTask {
	a.mu.Lock()
	scheduler := a.taskScheduler
	a.mu.Unlock()

	if scheduler == nil {
		return []*redc.ScheduledTask{}
	}

	return scheduler.ListTasksByCase(caseID)
}

func (a *App) ListAllScheduledTasks() []*redc.ScheduledTask {
	a.mu.Lock()
	scheduler := a.taskScheduler
	a.mu.Unlock()

	if scheduler == nil {
		return []*redc.ScheduledTask{}
	}

	return scheduler.ListAllTasksFromDB()
}
