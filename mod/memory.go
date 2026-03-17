package mod

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// MemoryItem represents a single agent memory entry
type MemoryItem struct {
	ID        int    `json:"id"`
	Project   string `json:"project"`
	Category  string `json:"category"` // "lesson" | "preference" | "note"
	Content   string `json:"content"`
	Source    string `json:"source"` // "auto" or conversationId
	CreatedAt string `json:"createdAt"`
}

const memoryMaxItems = 50

// MemoryStore manages agent memory persistence
type MemoryStore struct {
	mu sync.Mutex
	db *sql.DB
}

// NewMemoryStore creates and initializes a memory store
func NewMemoryStore() (*MemoryStore, error) {
	if err := ensureRedcPath(); err != nil {
		return nil, err
	}
	dbPath := filepath.Join(RedcPath, "memory.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open memory db: %v", err)
	}

	createSQL := `
	CREATE TABLE IF NOT EXISTS agent_memory (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		project TEXT NOT NULL,
		category TEXT NOT NULL,
		content TEXT NOT NULL,
		source TEXT DEFAULT 'auto',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_memory_project ON agent_memory(project);
	CREATE TABLE IF NOT EXISTS agent_tasks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		project TEXT NOT NULL,
		conversation_id TEXT NOT NULL,
		task_title TEXT NOT NULL,
		task_status TEXT DEFAULT 'in_progress',
		plan_json TEXT,
		checkpoint_messages TEXT,
		checkpoint_round INTEGER DEFAULT 0,
		prompt_template TEXT,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_tasks_project ON agent_tasks(project);
	`
	if _, err := db.Exec(createSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create memory table: %v", err)
	}

	// Migration: add checkpoint columns if they don't exist (for existing databases)
	for _, col := range []string{
		"ALTER TABLE agent_tasks ADD COLUMN checkpoint_messages TEXT",
		"ALTER TABLE agent_tasks ADD COLUMN checkpoint_round INTEGER DEFAULT 0",
		"ALTER TABLE agent_tasks ADD COLUMN prompt_template TEXT",
	} {
		db.Exec(col) // ignore errors (column already exists)
	}

	return &MemoryStore{db: db}, nil
}

// Close closes the database connection
func (m *MemoryStore) Close() {
	if m.db != nil {
		m.db.Close()
	}
}

// AddMemory adds a new memory entry and enforces the max items limit
func (m *MemoryStore) AddMemory(project, category, content, source string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check for duplicate content
	var count int
	err := m.db.QueryRow("SELECT COUNT(*) FROM agent_memory WHERE project = ? AND content = ?", project, content).Scan(&count)
	if err == nil && count > 0 {
		return nil // skip duplicate
	}

	_, err = m.db.Exec(
		"INSERT INTO agent_memory (project, category, content, source, created_at) VALUES (?, ?, ?, ?, ?)",
		project, category, content, source, time.Now().Format("2006-01-02 15:04:05"),
	)
	if err != nil {
		return err
	}

	// Enforce max items: delete oldest entries beyond limit
	m.db.Exec(`
		DELETE FROM agent_memory WHERE project = ? AND id NOT IN (
			SELECT id FROM agent_memory WHERE project = ? ORDER BY created_at DESC LIMIT ?
		)
	`, project, project, memoryMaxItems)

	return nil
}

// ListMemories returns all memories for a given project
func (m *MemoryStore) ListMemories(project string) ([]MemoryItem, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	rows, err := m.db.Query(
		"SELECT id, project, category, content, source, created_at FROM agent_memory WHERE project = ? ORDER BY created_at DESC",
		project,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []MemoryItem
	for rows.Next() {
		var item MemoryItem
		if err := rows.Scan(&item.ID, &item.Project, &item.Category, &item.Content, &item.Source, &item.CreatedAt); err != nil {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

// DeleteMemory deletes a specific memory by ID
func (m *MemoryStore) DeleteMemory(id int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, err := m.db.Exec("DELETE FROM agent_memory WHERE id = ?", id)
	return err
}

// ClearMemories deletes all memories for a project
func (m *MemoryStore) ClearMemories(project string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, err := m.db.Exec("DELETE FROM agent_memory WHERE project = ?", project)
	return err
}

// GetMemoryContext builds a formatted string of memories for prompt injection
func (m *MemoryStore) GetMemoryContext(project string) string {
	items, err := m.ListMemories(project)
	if err != nil || len(items) == 0 {
		return ""
	}

	var lessons, preferences, failures []string
	for _, item := range items {
		switch item.Category {
		case "lesson":
			lessons = append(lessons, item.Content)
		case "preference":
			preferences = append(preferences, item.Content)
		case "failure":
			failures = append(failures, item.Content)
		default:
			lessons = append(lessons, item.Content)
		}
	}

	result := ""
	if len(preferences) > 0 {
		result += "### 用户偏好\n"
		for _, p := range preferences {
			result += fmt.Sprintf("- %s\n", p)
		}
	}
	if len(lessons) > 0 {
		result += "### 历史经验教训\n"
		for i, l := range lessons {
			result += fmt.Sprintf("%d. %s\n", i+1, l)
		}
	}
	if len(failures) > 0 {
		result += "### 已知失败模式（遇到类似问题时直接套用解决方案）\n"
		for _, f := range failures {
			result += fmt.Sprintf("- %s\n", f)
		}
	}
	return result
}

// TaskItem represents a persisted agent task plan
type TaskItem struct {
	ID             int    `json:"id"`
	Project        string `json:"project"`
	ConversationID string `json:"conversationId"`
	TaskTitle      string `json:"taskTitle"`
	TaskStatus     string `json:"taskStatus"`
	PlanJSON       string `json:"planJson"`
	UpdatedAt      string `json:"updatedAt"`
}

// SaveTaskPlan upserts a task plan (insert or update by conversation_id)
func (m *MemoryStore) SaveTaskPlan(project, conversationId, title, planJSON string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now().Format("2006-01-02 15:04:05")

	// Check if a task for this conversation already exists
	var existingID int
	err := m.db.QueryRow(
		"SELECT id FROM agent_tasks WHERE conversation_id = ?", conversationId,
	).Scan(&existingID)

	if err == nil {
		// Update existing
		_, err = m.db.Exec(
			"UPDATE agent_tasks SET task_title = ?, plan_json = ?, updated_at = ? WHERE id = ?",
			title, planJSON, now, existingID,
		)
		return err
	}

	// Insert new
	_, err = m.db.Exec(
		"INSERT INTO agent_tasks (project, conversation_id, task_title, task_status, plan_json, updated_at) VALUES (?, ?, ?, 'in_progress', ?, ?)",
		project, conversationId, title, planJSON, now,
	)
	return err
}

// CompleteTask marks a task as completed
func (m *MemoryStore) CompleteTask(conversationId string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, err := m.db.Exec(
		"UPDATE agent_tasks SET task_status = 'completed', updated_at = ? WHERE conversation_id = ?",
		time.Now().Format("2006-01-02 15:04:05"), conversationId,
	)
	return err
}

// GetRecentTasks returns the most recent tasks for a project
func (m *MemoryStore) GetRecentTasks(project string, limit int) ([]TaskItem, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if limit <= 0 {
		limit = 10
	}

	rows, err := m.db.Query(
		"SELECT id, project, conversation_id, task_title, task_status, COALESCE(plan_json,''), updated_at FROM agent_tasks WHERE project = ? ORDER BY updated_at DESC LIMIT ?",
		project, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []TaskItem
	for rows.Next() {
		var item TaskItem
		if err := rows.Scan(&item.ID, &item.Project, &item.ConversationID, &item.TaskTitle, &item.TaskStatus, &item.PlanJSON, &item.UpdatedAt); err != nil {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

// AgentCheckpoint holds the state needed to resume an agent conversation
type AgentCheckpoint struct {
	ConversationID string `json:"conversationId"`
	Round          int    `json:"round"`
	Messages       string `json:"messages"`       // JSON-encoded compressed messages
	PromptTemplate string `json:"promptTemplate"` // which prompt template was used
}

// SaveCheckpoint persists agent loop state for resumption after interruption (upsert)
func (m *MemoryStore) SaveCheckpoint(project, conversationId string, messagesJSON string, round int, promptTemplate string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now().Format("2006-01-02 15:04:05")

	// Try update first
	res, err := m.db.Exec(
		"UPDATE agent_tasks SET checkpoint_messages = ?, checkpoint_round = ?, prompt_template = ?, updated_at = ? WHERE conversation_id = ?",
		messagesJSON, round, promptTemplate, now, conversationId,
	)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected > 0 {
		return nil
	}

	// No existing row — insert
	_, err = m.db.Exec(
		"INSERT INTO agent_tasks (project, conversation_id, task_title, task_status, checkpoint_messages, checkpoint_round, prompt_template, updated_at) VALUES (?, ?, 'Agent Task', 'in_progress', ?, ?, ?, ?)",
		project, conversationId, messagesJSON, round, promptTemplate, now,
	)
	return err
}

// LoadCheckpoint retrieves the latest checkpoint for a conversation
func (m *MemoryStore) LoadCheckpoint(conversationId string) (*AgentCheckpoint, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var cp AgentCheckpoint
	cp.ConversationID = conversationId
	var messagesJSON, promptTemplate sql.NullString
	err := m.db.QueryRow(
		"SELECT COALESCE(checkpoint_round, 0), checkpoint_messages, prompt_template FROM agent_tasks WHERE conversation_id = ? AND task_status = 'in_progress'",
		conversationId,
	).Scan(&cp.Round, &messagesJSON, &promptTemplate)
	if err != nil {
		return nil, fmt.Errorf("no checkpoint found: %v", err)
	}
	if !messagesJSON.Valid || messagesJSON.String == "" {
		return nil, fmt.Errorf("no checkpoint data for conversation %s", conversationId)
	}
	cp.Messages = messagesJSON.String
	if promptTemplate.Valid {
		cp.PromptTemplate = promptTemplate.String
	}
	return &cp, nil
}
