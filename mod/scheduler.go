package mod

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// ScheduledTask 定时任务
type ScheduledTask struct {
	ID             string    `json:"id"`
	CaseID         string    `json:"caseId"`
	CaseName       string    `json:"caseName"`
	Action         string    `json:"action"` // "start", "stop", "ssh_command", "auto_stop"
	ScheduledAt    time.Time `json:"scheduledAt"`
	CreatedAt      time.Time `json:"createdAt"`
	Status         string    `json:"status"` // "pending", "completed", "failed", "cancelled"
	Error          string    `json:"error,omitempty"`
	RepeatType     string    `json:"repeatType,omitempty"`     // "once", "daily", "weekly", "interval"
	RepeatInterval int       `json:"repeatInterval,omitempty"` // minutes, only for "interval" type
	CompletedAt    time.Time `json:"completedAt,omitempty"`
	SSHCommand     string    `json:"sshCommand,omitempty"`     // SSH command to execute (for "ssh_command" action)
	TaskResult     string    `json:"taskResult,omitempty"`     // execution result (e.g. SSH output)
	NotifyEnabled  bool      `json:"notifyEnabled,omitempty"`  // send notification on completion
}

// TaskScheduler 任务调度器
type TaskScheduler struct {
	tasks         map[string]*ScheduledTask
	mu            sync.RWMutex
	stopChan      chan struct{}
	project       *RedcProject
	onExecute     func(caseID string, action string) error
	onSSHCommand  func(caseID string, command string) (string, error)
	onNotify      func(title string, message string)
	db            *sql.DB
	dbPath        string
}

// NewTaskScheduler 创建新的任务调度器
func NewTaskScheduler(project *RedcProject, dbPath string) *TaskScheduler {
	return &TaskScheduler{
		tasks:    make(map[string]*ScheduledTask),
		stopChan: make(chan struct{}),
		project:  project,
		dbPath:   dbPath,
	}
}

// SetExecuteCallback 设置执行回调
func (s *TaskScheduler) SetExecuteCallback(callback func(string, string) error) {
	s.onExecute = callback
}

// SetSSHCommandCallback 设置 SSH 命令执行回调
func (s *TaskScheduler) SetSSHCommandCallback(callback func(string, string) (string, error)) {
	s.onSSHCommand = callback
}

// SetNotifyCallback 设置通知回调
func (s *TaskScheduler) SetNotifyCallback(callback func(string, string)) {
	s.onNotify = callback
}

// InitDB 初始化数据库
func (s *TaskScheduler) InitDB() error {
	db, err := sql.Open("sqlite3", s.dbPath)
	if err != nil {
		return fmt.Errorf("打开数据库失败: %v", err)
	}

	// 创建表
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS scheduled_tasks (
		id TEXT PRIMARY KEY,
		case_id TEXT NOT NULL,
		case_name TEXT NOT NULL,
		action TEXT NOT NULL,
		scheduled_at DATETIME NOT NULL,
		created_at DATETIME NOT NULL,
		status TEXT NOT NULL,
		error TEXT,
		repeat_type TEXT DEFAULT 'once',
		repeat_interval INTEGER DEFAULT 0,
		completed_at DATETIME
	);
	CREATE INDEX IF NOT EXISTS idx_case_id ON scheduled_tasks(case_id);
	CREATE INDEX IF NOT EXISTS idx_status ON scheduled_tasks(status);
	CREATE INDEX IF NOT EXISTS idx_scheduled_at ON scheduled_tasks(scheduled_at);
	`

	if _, err := db.Exec(createTableSQL); err != nil {
		db.Close()
		return fmt.Errorf("创建表失败: %v", err)
	}

	// Migrate: add columns if they don't exist (safe for existing DBs)
	for _, col := range []string{
		"ALTER TABLE scheduled_tasks ADD COLUMN repeat_type TEXT DEFAULT 'once'",
		"ALTER TABLE scheduled_tasks ADD COLUMN repeat_interval INTEGER DEFAULT 0",
		"ALTER TABLE scheduled_tasks ADD COLUMN completed_at DATETIME",
		"ALTER TABLE scheduled_tasks ADD COLUMN ssh_command TEXT DEFAULT ''",
		"ALTER TABLE scheduled_tasks ADD COLUMN task_result TEXT DEFAULT ''",
		"ALTER TABLE scheduled_tasks ADD COLUMN notify_enabled INTEGER DEFAULT 0",
	} {
		db.Exec(col) // ignore "duplicate column" errors
	}

	s.db = db

	// 从数据库加载待执行的任务
	if err := s.loadTasksFromDB(); err != nil {
		return fmt.Errorf("加载任务失败: %v", err)
	}

	return nil
}

// loadTasksFromDB 从数据库加载待执行的任务
func (s *TaskScheduler) loadTasksFromDB() error {
	rows, err := s.db.Query(`
		SELECT id, case_id, case_name, action, scheduled_at, created_at, status, error,
		       COALESCE(repeat_type, 'once'), COALESCE(repeat_interval, 0), completed_at,
		       COALESCE(ssh_command, ''), COALESCE(task_result, ''), COALESCE(notify_enabled, 0)
		FROM scheduled_tasks
		WHERE status = 'pending'
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	s.mu.Lock()
	defer s.mu.Unlock()

	for rows.Next() {
		task := &ScheduledTask{}
		var scheduledAtStr, createdAtStr string
		var errorStr, completedAtStr sql.NullString
		var notifyInt int

		err := rows.Scan(
			&task.ID,
			&task.CaseID,
			&task.CaseName,
			&task.Action,
			&scheduledAtStr,
			&createdAtStr,
			&task.Status,
			&errorStr,
			&task.RepeatType,
			&task.RepeatInterval,
			&completedAtStr,
			&task.SSHCommand,
			&task.TaskResult,
			&notifyInt,
		)
		if err != nil {
			continue
		}

		task.NotifyEnabled = notifyInt != 0
		task.ScheduledAt, _ = time.Parse(time.RFC3339, scheduledAtStr)
		task.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		if errorStr.Valid {
			task.Error = errorStr.String
		}
		if completedAtStr.Valid {
			task.CompletedAt, _ = time.Parse(time.RFC3339, completedAtStr.String)
		}
		if task.RepeatType == "" {
			task.RepeatType = "once"
		}

		s.tasks[task.ID] = task
	}

	return rows.Err()
}

// saveTaskToDB 保存任务到数据库
func (s *TaskScheduler) saveTaskToDB(task *ScheduledTask) error {
	var completedAt string
	if !task.CompletedAt.IsZero() {
		completedAt = task.CompletedAt.Format(time.RFC3339)
	}
	notifyInt := 0
	if task.NotifyEnabled {
		notifyInt = 1
	}
	_, err := s.db.Exec(`
		INSERT OR REPLACE INTO scheduled_tasks 
		(id, case_id, case_name, action, scheduled_at, created_at, status, error, repeat_type, repeat_interval, completed_at, ssh_command, task_result, notify_enabled)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		task.ID,
		task.CaseID,
		task.CaseName,
		task.Action,
		task.ScheduledAt.Format(time.RFC3339),
		task.CreatedAt.Format(time.RFC3339),
		task.Status,
		task.Error,
		task.RepeatType,
		task.RepeatInterval,
		completedAt,
		task.SSHCommand,
		task.TaskResult,
		notifyInt,
	)
	return err
}

// updateTaskStatusInDB 更新任务状态到数据库
func (s *TaskScheduler) updateTaskStatusInDB(taskID, status, errorMsg string) error {
	var completedAt string
	if status == "completed" || status == "failed" {
		completedAt = time.Now().Format(time.RFC3339)
	}
	_, err := s.db.Exec(`
		UPDATE scheduled_tasks 
		SET status = ?, error = ?, completed_at = ?
		WHERE id = ?
	`, status, errorMsg, completedAt, taskID)
	return err
}

// deleteTaskFromDB 从数据库删除任务
func (s *TaskScheduler) deleteTaskFromDB(taskID string) error {
	_, err := s.db.Exec(`DELETE FROM scheduled_tasks WHERE id = ?`, taskID)
	return err
}

// updateTaskResultInDB 更新任务执行结果到数据库
func (s *TaskScheduler) updateTaskResultInDB(taskID, result string) {
	if s.db == nil {
		return
	}
	// Truncate result to 4KB for DB storage
	if len(result) > 4096 {
		result = result[:4096] + "\n...(truncated)"
	}
	s.db.Exec(`UPDATE scheduled_tasks SET task_result = ? WHERE id = ?`, result, taskID)
}

// Start 启动调度器
func (s *TaskScheduler) Start() {
	go s.run()
	// 启动定期清理任务
	go s.periodicCleanup()
}

// Stop 停止调度器
func (s *TaskScheduler) Stop() {
	close(s.stopChan)
	if s.db != nil {
		s.db.Close()
	}
}

// periodicCleanup 定期清理已完成的任务
func (s *TaskScheduler) periodicCleanup() {
	ticker := time.NewTicker(1 * time.Hour) // 每小时清理一次
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.CleanupCompletedTasks()
		}
	}
}

// run 运行调度器主循环
func (s *TaskScheduler) run() {
	ticker := time.NewTicker(10 * time.Second) // 每10秒检查一次
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.checkAndExecuteTasks()
		}
	}
}

// checkAndExecuteTasks 检查并执行到期的任务
func (s *TaskScheduler) checkAndExecuteTasks() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for id, task := range s.tasks {
		if task.Status == "pending" && now.After(task.ScheduledAt) {
			// 执行任务
			go s.executeTask(id, task)
		}
	}
}

// executeTask 执行任务
func (s *TaskScheduler) executeTask(id string, task *ScheduledTask) {
	s.mu.Lock()
	task.Status = "executing"
	s.updateTaskStatusInDB(id, "executing", "")
	s.mu.Unlock()

	var err error
	var result string

	switch task.Action {
	case "ssh_command":
		if s.onSSHCommand != nil {
			result, err = s.onSSHCommand(task.CaseID, task.SSHCommand)
		} else {
			err = fmt.Errorf("SSH command callback not configured")
		}
	case "auto_stop":
		// auto_stop is just a stop action
		err = s.onExecute(task.CaseID, "stop")
	default:
		// "start" / "stop"
		err = s.onExecute(task.CaseID, task.Action)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	if err != nil {
		task.Status = "failed"
		task.Error = err.Error()
		task.CompletedAt = now
		task.TaskResult = result
		s.updateTaskStatusInDB(id, "failed", err.Error())
	} else {
		task.Status = "completed"
		task.CompletedAt = now
		task.TaskResult = result
		s.updateTaskStatusInDB(id, "completed", "")
	}

	// Update task_result in DB
	if result != "" {
		s.updateTaskResultInDB(id, result)
	}

	// Send notification if enabled
	if task.NotifyEnabled && s.onNotify != nil {
		actionLabel := task.Action
		statusLabel := task.Status
		msg := fmt.Sprintf("任务 [%s] %s 执行%s", task.CaseName, actionLabel, statusLabel)
		if task.Action == "ssh_command" {
			msg = fmt.Sprintf("SSH 命令任务 [%s] 执行%s", task.CaseName, statusLabel)
		}
		if err != nil {
			msg += ": " + task.Error
		}
		s.onNotify("任务中心", msg)
	}

	// Auto-renew periodic tasks (even on failure, schedule next)
	s.scheduleNextRepeat(task)
}

// AddTask 添加定时任务
func (s *TaskScheduler) AddTask(caseID, caseName, action string, scheduledAt time.Time) (*ScheduledTask, error) {
	return s.AddTaskWithRepeat(caseID, caseName, action, scheduledAt, "once", 0)
}

// AddTaskWithRepeat 添加定时任务（支持周期重复）
func (s *TaskScheduler) AddTaskWithRepeat(caseID, caseName, action string, scheduledAt time.Time, repeatType string, repeatInterval int) (*ScheduledTask, error) {
	return s.AddTaskFull(caseID, caseName, action, scheduledAt, repeatType, repeatInterval, "", false)
}

// AddTaskFull 添加完整配置的定时任务
func (s *TaskScheduler) AddTaskFull(caseID, caseName, action string, scheduledAt time.Time, repeatType string, repeatInterval int, sshCommand string, notifyEnabled bool) (*ScheduledTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 验证 action
	switch action {
	case "start", "stop", "ssh_command", "auto_stop":
	default:
		return nil, fmt.Errorf("无效的操作类型: %s", action)
	}

	// SSH command requires a command string
	if action == "ssh_command" && sshCommand == "" {
		return nil, fmt.Errorf("SSH 命令任务必须提供命令")
	}

	// 验证时间
	if scheduledAt.Before(time.Now()) {
		return nil, fmt.Errorf("计划时间不能早于当前时间")
	}

	// 验证 repeatType
	if repeatType == "" {
		repeatType = "once"
	}
	switch repeatType {
	case "once", "daily", "weekly", "interval":
	default:
		return nil, fmt.Errorf("无效的重复类型: %s", repeatType)
	}
	if repeatType == "interval" && repeatInterval <= 0 {
		return nil, fmt.Errorf("自定义间隔必须大于0分钟")
	}

	// 生成任务 ID
	taskID := fmt.Sprintf("%s-%s-%d", caseID, action, time.Now().UnixNano())

	// 创建任务
	task := &ScheduledTask{
		ID:             taskID,
		CaseID:         caseID,
		CaseName:       caseName,
		Action:         action,
		ScheduledAt:    scheduledAt,
		CreatedAt:      time.Now(),
		Status:         "pending",
		RepeatType:     repeatType,
		RepeatInterval: repeatInterval,
		SSHCommand:     sshCommand,
		NotifyEnabled:  notifyEnabled,
	}

	s.tasks[taskID] = task

	// 保存到数据库
	if err := s.saveTaskToDB(task); err != nil {
		delete(s.tasks, taskID)
		return nil, fmt.Errorf("保存任务到数据库失败: %v", err)
	}

	return task, nil
}

// scheduleNextRepeat creates the next occurrence for a periodic task (caller must hold mu)
func (s *TaskScheduler) scheduleNextRepeat(task *ScheduledTask) {
	if task.RepeatType == "" || task.RepeatType == "once" {
		return
	}

	var nextTime time.Time
	now := time.Now()
	switch task.RepeatType {
	case "daily":
		nextTime = task.ScheduledAt.Add(24 * time.Hour)
		for nextTime.Before(now) {
			nextTime = nextTime.Add(24 * time.Hour)
		}
	case "weekly":
		nextTime = task.ScheduledAt.Add(7 * 24 * time.Hour)
		for nextTime.Before(now) {
			nextTime = nextTime.Add(7 * 24 * time.Hour)
		}
	case "interval":
		interval := time.Duration(task.RepeatInterval) * time.Minute
		nextTime = task.ScheduledAt.Add(interval)
		for nextTime.Before(now) {
			nextTime = nextTime.Add(interval)
		}
	default:
		return
	}

	nextID := fmt.Sprintf("%s-%s-%d", task.CaseID, task.Action, time.Now().UnixNano())
	nextTask := &ScheduledTask{
		ID:             nextID,
		CaseID:         task.CaseID,
		CaseName:       task.CaseName,
		Action:         task.Action,
		ScheduledAt:    nextTime,
		CreatedAt:      time.Now(),
		Status:         "pending",
		RepeatType:     task.RepeatType,
		RepeatInterval: task.RepeatInterval,
		SSHCommand:     task.SSHCommand,
		NotifyEnabled:  task.NotifyEnabled,
	}

	s.tasks[nextID] = nextTask
	s.saveTaskToDB(nextTask)
}

// CancelTask 取消任务
func (s *TaskScheduler) CancelTask(taskID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, exists := s.tasks[taskID]
	if !exists {
		return fmt.Errorf("任务不存在")
	}

	if task.Status != "pending" {
		return fmt.Errorf("只能取消待执行的任务")
	}

	task.Status = "cancelled"

	// 更新数据库
	if err := s.updateTaskStatusInDB(taskID, "cancelled", ""); err != nil {
		return fmt.Errorf("更新数据库失败: %v", err)
	}

	return nil
}

// GetTask 获取任务
func (s *TaskScheduler) GetTask(taskID string) (*ScheduledTask, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	task, exists := s.tasks[taskID]
	if !exists {
		return nil, fmt.Errorf("任务不存在")
	}

	return task, nil
}

// ListTasks 列出所有任务
func (s *TaskScheduler) ListTasks() []*ScheduledTask {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tasks := make([]*ScheduledTask, 0, len(s.tasks))
	for _, task := range s.tasks {
		tasks = append(tasks, task)
	}

	return tasks
}

// ListTasksByCase 列出指定场景的任务
func (s *TaskScheduler) ListTasksByCase(caseID string) []*ScheduledTask {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tasks := make([]*ScheduledTask, 0)
	for _, task := range s.tasks {
		if task.CaseID == caseID {
			tasks = append(tasks, task)
		}
	}

	return tasks
}

// ListAllTasksFromDB 从数据库读取所有任务（含历史，最近 7 天）
func (s *TaskScheduler) ListAllTasksFromDB() []*ScheduledTask {
	if s.db == nil {
		return s.ListTasks()
	}

	cutoff := time.Now().Add(-7 * 24 * time.Hour).Format(time.RFC3339)
	rows, err := s.db.Query(`
		SELECT id, case_id, case_name, action, scheduled_at, created_at, status, error,
		       COALESCE(repeat_type, 'once'), COALESCE(repeat_interval, 0), completed_at,
		       COALESCE(ssh_command, ''), COALESCE(task_result, ''), COALESCE(notify_enabled, 0)
		FROM scheduled_tasks
		WHERE created_at > ? OR status = 'pending'
		ORDER BY scheduled_at DESC
	`, cutoff)
	if err != nil {
		return s.ListTasks()
	}
	defer rows.Close()

	var tasks []*ScheduledTask
	for rows.Next() {
		task := &ScheduledTask{}
		var scheduledAtStr, createdAtStr string
		var errorStr, completedAtStr sql.NullString
		var notifyInt int

		err := rows.Scan(
			&task.ID, &task.CaseID, &task.CaseName, &task.Action,
			&scheduledAtStr, &createdAtStr, &task.Status, &errorStr,
			&task.RepeatType, &task.RepeatInterval, &completedAtStr,
			&task.SSHCommand, &task.TaskResult, &notifyInt,
		)
		if err != nil {
			continue
		}

		task.NotifyEnabled = notifyInt != 0
		task.ScheduledAt, _ = time.Parse(time.RFC3339, scheduledAtStr)
		task.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		if errorStr.Valid {
			task.Error = errorStr.String
		}
		if completedAtStr.Valid {
			task.CompletedAt, _ = time.Parse(time.RFC3339, completedAtStr.String)
		}
		if task.RepeatType == "" {
			task.RepeatType = "once"
		}
		tasks = append(tasks, task)
	}
	return tasks
}

// CleanupCompletedTasks 清理已完成的任务（保留最近24小时的）
func (s *TaskScheduler) CleanupCompletedTasks() {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-24 * time.Hour)
	cutoffStr := cutoff.Format(time.RFC3339)

	// 从数据库删除
	if s.db != nil {
		s.db.Exec(`
			DELETE FROM scheduled_tasks 
			WHERE status IN ('completed', 'failed', 'cancelled') 
			AND created_at < ?
		`, cutoffStr)
	}

	// 从内存删除
	for id, task := range s.tasks {
		if (task.Status == "completed" || task.Status == "failed" || task.Status == "cancelled") &&
			task.CreatedAt.Before(cutoff) {
			delete(s.tasks, id)
		}
	}
}
