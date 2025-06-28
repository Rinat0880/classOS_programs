package whitelist

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"school_agent/internal/logger"
)

type Whitelist struct {
	Version   string            `json:"version"`
	Items     []string          `json:"items"`
	Hashes    map[string]string `json:"hashes,omitempty"`
	UpdatedAt time.Time         `json:"updated_at"`
}

type Manager struct {
	filePath    string
	serverURL   string
	whitelist   *Whitelist
	mu          sync.RWMutex
	httpClient  *http.Client
	initialized bool
}

func NewManager(filePath, serverURL string) *Manager {
	return &Manager{
		filePath:  filePath,
		serverURL: serverURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (m *Manager) Initialize() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	logger.Info("Инициализация whitelist менеджера...")

	if err := m.loadFromCache(); err != nil {
		logger.Warn("Не удалось загрузить whitelist из кэша: %v", err)
		
		if m.serverURL != "" {
			if err := m.fetchFromServer(); err != nil {
				logger.Warn("Не удалось получить whitelist с сервера: %v", err)
				m.createDefaultWhitelist()
			}
		} else {
			m.createDefaultWhitelist()
		}
	}

	m.initialized = true
	logger.Info("Whitelist инициализирован. Версия: %s, Элементов: %d", 
		m.whitelist.Version, len(m.whitelist.Items))
	
	return nil
}

func (m *Manager) IsAllowed(path string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.initialized || m.whitelist == nil {
		logger.Warn("Whitelist не инициализирован, разрешаем процесс: %s", path)
		return true
	}

	normalizedPath := m.normalizePath(path)
	
	for _, item := range m.whitelist.Items {
		if strings.EqualFold(m.normalizePath(item), normalizedPath) {
			logger.LogProcessAllow(0, path)
			return true
		}
	}

	for _, item := range m.whitelist.Items {
		if m.matchPattern(item, normalizedPath) {
			logger.LogProcessAllow(0, path)
			return true
		}
	}

	if m.isSystemProcess(normalizedPath) {
		logger.LogProcessAllow(0, path)
		return true
	}

	return false
}

func (m *Manager) Update() error {
	if m.serverURL == "" {
		return fmt.Errorf("URL сервера не настроен")
	}

	logger.Debug("Получение whitelist с сервера: %s", m.serverURL)

	newWhitelist, err := m.downloadWhitelist()
	if err != nil {
		return fmt.Errorf("ошибка загрузки whitelist: %v", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.whitelist != nil && m.whitelist.Version == newWhitelist.Version {
		logger.Debug("Whitelist уже актуален (версия %s)", m.whitelist.Version)
		return nil
	}

	oldVersion := ""
	if m.whitelist != nil {
		oldVersion = m.whitelist.Version
	}

	m.whitelist = newWhitelist

	if err := m.saveToCache(); err != nil {
		logger.Error("Ошибка сохранения whitelist в кэш: %v", err)
	}

	logger.LogWhitelistUpdate(m.whitelist.Version, len(m.whitelist.Items))
	logger.Info("Whitelist обновлен с версии %s на %s", oldVersion, m.whitelist.Version)

	return nil
}

func (m *Manager) loadFromCache() error {
	if _, err := os.Stat(m.filePath); os.IsNotExist(err) {
		return fmt.Errorf("файл whitelist не существует: %s", m.filePath)
	}

	data, err := os.ReadFile(m.filePath)
	if err != nil {
		return fmt.Errorf("ошибка чтения файла: %v", err)
	}

	var whitelist Whitelist
	if err := json.Unmarshal(data, &whitelist); err != nil {
		return fmt.Errorf("ошибка парсинга JSON: %v", err)
	}

	m.whitelist = &whitelist
	logger.Debug("Whitelist загружен из кэша: %s", m.filePath)
	return nil
}

func (m *Manager) saveToCache() error {
	if m.whitelist == nil {
		return fmt.Errorf("whitelist не инициализирован")
	}

	dir := filepath.Dir(m.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("ошибка создания директории: %v", err)
	}

	data, err := json.MarshalIndent(m.whitelist, "", "  ")
	if err != nil {
		return fmt.Errorf("ошибка сериализации JSON: %v", err)
	}

	if err := os.WriteFile(m.filePath, data, 0644); err != nil {
		return fmt.Errorf("ошибка записи файла: %v", err)
	}

	logger.Debug("Whitelist сохранен в кэш: %s", m.filePath)
	return nil
}

func (m *Manager) fetchFromServer() error {
	whitelist, err := m.downloadWhitelist()
	if err != nil {
		return err
	}

	m.whitelist = whitelist

	if err := m.saveToCache(); err != nil {
		logger.Warn("Не удалось сохранить whitelist в кэш: %v", err)
	}

	return nil
}

func (m *Manager) downloadWhitelist() (*Whitelist, error) {
	resp, err := m.httpClient.Get(m.serverURL)
	if err != nil {
		return nil, fmt.Errorf("ошибка HTTP запроса: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP ошибка: %d %s", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения ответа: %v", err)
	}

	var whitelist Whitelist
	if err := json.Unmarshal(body, &whitelist); err != nil {
		return nil, fmt.Errorf("ошибка парсинга JSON: %v", err)
	}

	return &whitelist, nil
}

func (m *Manager) createDefaultWhitelist() {
	m.whitelist = &Whitelist{
		Version:   "1.0.0-default",
		UpdatedAt: time.Now(),
		Items: []string{
			"C:\\Windows\\System32\\*",
			"C:\\Windows\\SysWOW64\\*",
			"C:\\Windows\\WinSxS\\*",
			
			"C:\\Windows\\System32\\notepad.exe",
			"C:\\Windows\\System32\\calc.exe",
			"C:\\Windows\\System32\\mspaint.exe",
			"C:\\Windows\\System32\\cmd.exe",
			"C:\\Windows\\System32\\powershell.exe",
			
			"C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe",
			"C:\\Program Files (x86)\\Google\\Chrome\\Application\\chrome.exe",
			"C:\\Program Files\\Mozilla Firefox\\firefox.exe",
			"C:\\Program Files (x86)\\Mozilla Firefox\\firefox.exe",
			"C:\\Program Files\\Internet Explorer\\iexplore.exe",
			"C:\\Program Files (x86)\\Internet Explorer\\iexplore.exe",
			
			"C:\\Program Files\\Microsoft Office\\*",
			"C:\\Program Files (x86)\\Microsoft Office\\*",
			
			"C:\\Program Files\\Windows Defender\\*",
			"C:\\ProgramData\\Microsoft\\Windows Defender\\*",
		},
	}

	logger.Info("Создан базовый whitelist с %d элементами", len(m.whitelist.Items))

	if err := m.saveToCache(); err != nil {
		logger.Warn("Не удалось сохранить базовый whitelist: %v", err)
	}
}

func (m *Manager) normalizePath(path string) string {
	normalized := strings.ToLower(path)
	
	normalized = strings.ReplaceAll(normalized, "/", "\\")
	
	for strings.Contains(normalized, "\\\\") {
		normalized = strings.ReplaceAll(normalized, "\\\\", "\\")
	}
	
	return normalized
}

func (m *Manager) matchPattern(pattern, path string) bool {
	pattern = m.normalizePath(pattern)
	path = m.normalizePath(path)

	// Простая поддержка wildcard '*'
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(path, prefix)
	}

	return pattern == path
}

func (m *Manager) isSystemProcess(path string) bool {
	systemPaths := []string{
		"c:\\windows\\system32\\",
		"c:\\windows\\syswow64\\",
		"c:\\windows\\winsxs\\",
		"c:\\programdata\\microsoft\\windows defender\\",
	}

	normalizedPath := m.normalizePath(path)
	
	for _, systemPath := range systemPaths {
		if strings.HasPrefix(normalizedPath, systemPath) {
			return true
		}
	}

	return false
}

func (m *Manager) GetVersion() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.whitelist == nil {
		return "unknown"
	}
	return m.whitelist.Version
}

func (m *Manager) GetItemCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.whitelist == nil {
		return 0
	}
	return len(m.whitelist.Items)
}

func (m *Manager) GetItems() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.whitelist == nil {
		return nil
	}

	items := make([]string, len(m.whitelist.Items))
	copy(items, m.whitelist.Items)
	return items
}

func (m *Manager) AddItem(path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.whitelist == nil {
		return fmt.Errorf("whitelist не инициализирован")
	}

	normalizedPath := m.normalizePath(path)
	for _, item := range m.whitelist.Items {
		if strings.EqualFold(m.normalizePath(item), normalizedPath) {
			return fmt.Errorf("элемент уже существует: %s", path)
		}
	}

	m.whitelist.Items = append(m.whitelist.Items, path)
	m.whitelist.UpdatedAt = time.Now()

	if err := m.saveToCache(); err != nil {
		return fmt.Errorf("ошибка сохранения: %v", err)
	}

	logger.Info("Добавлен элемент в whitelist: %s", path)
	return nil
}

func (m *Manager) RemoveItem(path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.whitelist == nil {
		return fmt.Errorf("whitelist не инициализирован")
	}

	normalizedPath := m.normalizePath(path)
	found := false
	newItems := make([]string, 0, len(m.whitelist.Items))

	for _, item := range m.whitelist.Items {
		if !strings.EqualFold(m.normalizePath(item), normalizedPath) {
			newItems = append(newItems, item)
		} else {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("элемент не найден: %s", path)
	}

	m.whitelist.Items = newItems
	m.whitelist.UpdatedAt = time.Now()

	if err := m.saveToCache(); err != nil {
		return fmt.Errorf("ошибка сохранения: %v", err)
	}

	logger.Info("Удален элемент из whitelist: %s", path)
	return nil
}

func (m *Manager) ValidateChecksum(path string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.whitelist == nil || m.whitelist.Hashes == nil {
		return true 
	}

	expectedHash, exists := m.whitelist.Hashes[m.normalizePath(path)]
	if !exists {
		return true 
	}

	// Вычисляем MD5 хеш файла
	actualHash, err := m.calculateMD5(path)
	if err != nil {
		logger.Warn("Ошибка вычисления хеша для %s: %v", path, err)
		return true 
	}

	match := strings.EqualFold(expectedHash, actualHash)
	if !match {
		logger.Warn("Хеш файла не совпадает: %s (ожидался: %s, получен: %s)", path, expectedHash, actualHash)
	}

	return match
}

func (m *Manager) calculateMD5(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}