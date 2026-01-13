package logger

import (
	"encoding/json"
	"os"
	"path/filepath"
	"school_agent/internal/models"
	"time"
)

type Manager struct {
	logDir   string
	hostname string
	queue    chan models.LogEntry
}

func New(logDir, hostname string) *Manager {
	return &Manager{
		logDir:   logDir,
		hostname: hostname,
		queue:    make(chan models.LogEntry, 100),
	}
}

// Start запускает воркер записи в файл
func (m *Manager) Start() {
	go func() {
		for entry := range m.queue {
			m.writeToFile(entry)
		}
	}()
}

func (m *Manager) Add(user, lType, prog, action string) {
	if user == "" {
		user = "system"
	}
	m.queue <- models.LogEntry{
		Username:   user,
		DeviceName: m.hostname,
		Timestamp:  time.Now(),
		LogType:    lType,
		Program:    prog,
		Action:     action,
	}
}

func (m *Manager) writeToFile(entry models.LogEntry) {
	filename := filepath.Join(m.logDir, time.Now().Format("2006-01-02")+".jsonl")
	f, _ := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer f.Close()
	
	data, _ := json.Marshal(entry)
	f.Write(data)
	f.WriteString("\n")
}

// GetCurrentLogFile возвращает путь к текущему логу (для отправки)
func (m *Manager) GetCurrentLogFile() string {
	return filepath.Join(m.logDir, time.Now().Format("2006-01-02")+".jsonl")
}