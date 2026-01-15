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
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	f.WriteString(fmt.Sprintf("\n--- Session Start: %s ---\n", timestamp))

	return &ComposeWriter{ServiceName: serviceName, File: f}, nil
}

func (w *ComposeWriter) Write(p []byte) (n int, err error) {
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
