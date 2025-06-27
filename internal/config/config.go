package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	LogPath       string `json:"log_path"`
	WhitelistPath string `json:"whitelist_path"`

	WhitelistURL string `json:"whitelist_url,omitempty"`
	ServerURL    string `json:"server_url,omitempty"`

	UpdateInterval int `json:"update_interval"`

	LogLevel string `json:"log_level"`

	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`

	DebugMode bool `json:"debug_mode"`
	DryRun    bool `json:"dry_run"`
}

func DefaultConfig() *Config {
	return &Config{
		LogPath:        "C:\\ProgramData\\SchoolAgent\\logs\\agent.log",
		WhitelistPath:  "C:\\ProgramData\\SchoolAgent\\whitelist.json",
		UpdateInterval: 30, 
		LogLevel:       "info",
		DebugMode:      false,
		DryRun:         false,
	}
}

func LoadConfig() (*Config, error) {
	configPath := getConfigPath()

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		config := DefaultConfig()
		if err := config.Save(configPath); err != nil {
			return nil, fmt.Errorf("ошибка создания конфигурации по умолчанию: %v", err)
		}
		return config, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения файла конфигурации %s: %v", configPath, err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("ошибка парсинга конфигурации: %v", err)
	}

	fillDefaults(&config)

	return &config, nil
}

func (c *Config) Save(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("ошибка создания директории %s: %v", dir, err)
	}

	data, err := json.MarshalIndent(c, "", "    ")
	if err != nil {
		return fmt.Errorf("ошибка сериализации конфигурации: %v", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("ошибка записи файла %s: %v", path, err)
	}

	return nil
}

func getConfigPath() string {
	for i, arg := range os.Args {
		if arg == "--config" || arg == "-c" {
			if i+1 < len(os.Args) {
				return os.Args[i+1]
			}
		}
	}

	return "C:\\ProgramData\\SchoolAgent\\config\\agent.json"
}

func fillDefaults(config *Config) {
	defaults := DefaultConfig()

	if config.LogPath == "" {
		config.LogPath = defaults.LogPath
	}
	if config.WhitelistPath == "" {
		config.WhitelistPath = defaults.WhitelistPath
	}
	if config.UpdateInterval <= 0 {
		config.UpdateInterval = defaults.UpdateInterval
	}
	if config.LogLevel == "" {
		config.LogLevel = defaults.LogLevel
	}
}

func (c *Config) Validate() error {
	if c.LogPath == "" {
		return fmt.Errorf("не указан путь к файлу логов")
	}
	if c.WhitelistPath == "" {
		return fmt.Errorf("не указан путь к файлу whitelist")
	}

	if c.UpdateInterval <= 0 {
		return fmt.Errorf("интервал обновления должен быть больше 0")
	}

	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLogLevels[c.LogLevel] {
		return fmt.Errorf("недопустимый уровень логирования: %s", c.LogLevel)
	}

	if err := c.createDirectories(); err != nil {
		return fmt.Errorf("ошибка создания директорий: %v", err)
	}

	return nil
}

func (c *Config) createDirectories() error {
	dirs := []string{
		filepath.Dir(c.LogPath),
		filepath.Dir(c.WhitelistPath),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("ошибка создания директории %s: %v", dir, err)
		}
	}

	return nil
}

func (c *Config) IsDebugMode() bool {
	for _, arg := range os.Args {
		if arg == "--debug" || arg == "-d" {
			return true
		}
	}
	return c.DebugMode
}

func (c *Config) IsDryRun() bool {
	for _, arg := range os.Args {
		if arg == "--dry-run" {
			return true
		}
	}
	return c.DryRun
}
