package mod

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// AuditLogEntry represents a single audit log entry
type AuditLogEntry struct {
	ID        int    `json:"id"`
	Timestamp string `json:"timestamp"`
	Username  string `json:"username"`
	Role      string `json:"role"`
	Method    string `json:"method"`
	Args      string `json:"args"` // JSON string of first 2 args (for context)
	Success   bool   `json:"success"`
	Error     string `json:"error,omitempty"`
	IP        string `json:"ip,omitempty"`
}

const auditMaxEntries = 5000

// AuditStore manages audit log persistence
type AuditStore struct {
	mu sync.Mutex
	db *sql.DB
}

// NewAuditStore creates and initializes an audit store
func NewAuditStore() (*AuditStore, error) {
	if err := ensureRedcPath(); err != nil {
		return nil, err
	}
	dbPath := filepath.Join(RedcPath, "audit.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open audit db: %v", err)
	}

	createSQL := `
	CREATE TABLE IF NOT EXISTS audit_log (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp DATETIME NOT NULL,
		username TEXT NOT NULL,
		role TEXT NOT NULL,
		method TEXT NOT NULL,
		args TEXT DEFAULT '',
		success INTEGER DEFAULT 1,
		error TEXT DEFAULT '',
		ip TEXT DEFAULT ''
	);
	CREATE INDEX IF NOT EXISTS idx_audit_timestamp ON audit_log(timestamp);
	CREATE INDEX IF NOT EXISTS idx_audit_username ON audit_log(username);
	CREATE INDEX IF NOT EXISTS idx_audit_method ON audit_log(method);
	`
	if _, err := db.Exec(createSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create audit table: %v", err)
	}

	return &AuditStore{db: db}, nil
}

// Close closes the database connection
func (a *AuditStore) Close() {
	if a.db != nil {
		a.db.Close()
	}
}

// Log records an audit log entry
func (a *AuditStore) Log(username, role, method, args, ip string, success bool, errMsg string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.db.Exec(
		"INSERT INTO audit_log (timestamp, username, role, method, args, success, error, ip) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		time.Now().Format("2006-01-02 15:04:05"), username, role, method, args, success, errMsg, ip,
	)

	// Enforce max entries
	a.db.Exec(`
		DELETE FROM audit_log WHERE id NOT IN (
			SELECT id FROM audit_log ORDER BY timestamp DESC LIMIT ?
		)
	`, auditMaxEntries)
}

// List returns audit log entries with optional filters
func (a *AuditStore) List(limit int, offset int, username string, method string) ([]AuditLogEntry, int, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	where := "1=1"
	args := []interface{}{}
	if username != "" {
		where += " AND username = ?"
		args = append(args, username)
	}
	if method != "" {
		where += " AND method LIKE ?"
		args = append(args, "%"+method+"%")
	}

	// Count total
	var total int
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)
	err := a.db.QueryRow("SELECT COUNT(*) FROM audit_log WHERE "+where, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	if limit <= 0 {
		limit = 100
	}

	query := fmt.Sprintf("SELECT id, timestamp, username, role, method, args, success, error, ip FROM audit_log WHERE %s ORDER BY timestamp DESC LIMIT ? OFFSET ?", where)
	args = append(args, limit, offset)

	rows, err := a.db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var entries []AuditLogEntry
	for rows.Next() {
		var e AuditLogEntry
		var success int
		if err := rows.Scan(&e.ID, &e.Timestamp, &e.Username, &e.Role, &e.Method, &e.Args, &success, &e.Error, &e.IP); err != nil {
			continue
		}
		e.Success = success != 0
		entries = append(entries, e)
	}
	return entries, total, nil
}

// ExportAll returns all audit log entries as a slice (for export)
func (a *AuditStore) ExportAll() ([]AuditLogEntry, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	rows, err := a.db.Query("SELECT id, timestamp, username, role, method, args, success, error, ip FROM audit_log ORDER BY timestamp DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []AuditLogEntry
	for rows.Next() {
		var e AuditLogEntry
		var success int
		if err := rows.Scan(&e.ID, &e.Timestamp, &e.Username, &e.Role, &e.Method, &e.Args, &success, &e.Error, &e.IP); err != nil {
			continue
		}
		e.Success = success != 0
		entries = append(entries, e)
	}
	return entries, nil
}

// Clear deletes all audit log entries
func (a *AuditStore) Clear() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	_, err := a.db.Exec("DELETE FROM audit_log")
	return err
}
