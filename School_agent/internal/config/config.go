package config

import (
	"encoding/json"
	"os"
)

const (
	DefaultLogDir      = "C:\\ProgramData\\SchoolAgent\\Logs"
	DefaultConfigPath  = "C:\\ProgramData\\SchoolAgent\\config.json"
	DefaultProjectBase = "D:\\UserProjects"
	PipeName           = `\\.\pipe\SchoolAgentIPC`
)

type Config struct {
	ServerURL   string `json:"server_url"`
	DeviceToken string `json:"device_token"`
	Hostname    string `json:"hostname"`
	LogDir      string `json:"log_dir"`
	ProjectBase string `json:"project_base"`
}

func Load() *Config {
	// Дефолтные значения
	host, _ := os.Hostname()
	cfg := &Config{
		ServerURL:   "ws://localhost:8080/ws",
		DeviceToken: "unknown-device",
		Hostname:    host,
		LogDir:      DefaultLogDir,
		ProjectBase: DefaultProjectBase,
	}

	file, err := os.Open(DefaultConfigPath)
	if err == nil {
		defer file.Close()
		json.NewDecoder(file).Decode(cfg)
	}
	
	// Гарантируем, что папки существуют
	os.MkdirAll(cfg.LogDir, 0755)
	os.MkdirAll(cfg.ProjectBase, 0755)

	return cfg
}