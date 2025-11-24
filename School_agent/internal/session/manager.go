package session

import (
	"os"
	"path/filepath"
)

type Manager struct {
	baseDir string
}

func New(baseDir string) *Manager {
	return &Manager{baseDir: baseDir}
}

func (m *Manager) PrepareUserEnvironment(user string) {
	userDir := filepath.Join(m.baseDir, user)
	if _, err := os.Stat(userDir); os.IsNotExist(err) {
		// Тут можно добавить сложную логику ACL
		os.MkdirAll(userDir, 0700)
	}
}

func (m *Manager) Cleanup(user string) {
	// Логика очистки временных файлов
}