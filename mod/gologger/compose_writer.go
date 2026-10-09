package gologger

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var consoleMu sync.Mutex

type LogManager struct {
	BaseDir string
}

func NewLogManager(projectPath string) *LogManager {
	logDir := filepath.Join(projectPath, "logs")
	_ = os.MkdirAll(logDir, 0755)
	return &LogManager{BaseDir: logDir}
}

func (m *LogManager) GetLogPath(name string) string {
	return filepath.Join(m.BaseDir, fmt.Sprintf("%s.log", name))
}

type ComposeWriter struct {
	ServiceName string
	File        *os.File
}

func (m *LogManager) NewServiceLogger(serviceName string) (*ComposeWriter, error) {
	path := m.GetLogPath(serviceName)
	// 日志会包含 provider 报错、SSH 命令等敏感内容，仅限属主可读。
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return nil, err
	}
	// 历史文件可能由旧版本以 0644 创建，OpenFile 不会改变已有文件的权限。
	if err := f.Chmod(0600); err != nil {
		f.Close()
		return nil, err
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	f.WriteString(fmt.Sprintf("\n--- Session Start: %s ---\n", timestamp))

	return &ComposeWriter{ServiceName: serviceName, File: f}, nil
}

func (w *ComposeWriter) Write(p []byte) (n int, err error) {
	// The GUI's emitLog writes through this writer directly, bypassing
	// Logger.Log, so redaction must be applied here too or those lines land in
	// the log file untouched.
	p = []byte(Redact(string(p)))
	if w.File != nil {
		w.File.Write(p)
	}

	scanner := bufio.NewScanner(bytes.NewReader(p))
	consoleMu.Lock()
	defer consoleMu.Unlock()

	for scanner.Scan() {
		text := scanner.Text()
		if text != "" {
			fmt.Printf("[%s] | %s\n", w.ServiceName, text)
		}
	}
	return len(p), nil
}

func (w *ComposeWriter) Close() {
	if w.File != nil {
		if err := w.File.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "failed to close log file for %s: %v\n", w.ServiceName, err)
		}
	}
}

// LogEmitter is a callback function type for emitting log events
type LogEmitter func(message string)

// EventWriter is an io.Writer that emits logs to a callback function
// This is useful for Wails applications to stream logs to the frontend
type EventWriter struct {
	emitter LogEmitter
	prefix  string
}

// NewEventWriter creates a new EventWriter with the given emitter callback
func NewEventWriter(emitter LogEmitter, prefix string) *EventWriter {
	return &EventWriter{
		emitter: emitter,
		prefix:  prefix,
	}
}

// Write implements io.Writer and emits each line to the callback
func (w *EventWriter) Write(p []byte) (n int, err error) {
	if w.emitter == nil {
		return len(p), nil
	}
	scanner := bufio.NewScanner(bytes.NewReader(p))
	for scanner.Scan() {
		text := scanner.Text()
		if text != "" {
			if w.prefix != "" {
				w.emitter(fmt.Sprintf("[%s] %s", w.prefix, text))
			} else {
				w.emitter(text)
			}
		}
	}
	return len(p), nil
}

// MultiWriter combines multiple io.Writers into one
type MultiWriter struct {
	writers []interface{ Write([]byte) (int, error) }
}

// NewMultiWriter creates a writer that writes to all provided writers
func NewMultiWriter(writers ...interface{ Write([]byte) (int, error) }) *MultiWriter {
	return &MultiWriter{writers: writers}
}

// Write writes to all underlying writers
func (mw *MultiWriter) Write(p []byte) (n int, err error) {
	for _, w := range mw.writers {
		n, err = w.Write(p)
		if err != nil {
			return
		}
	}
	return len(p), nil
}
