package models

import "time"

// LogEntry - структура лога для сохранения и отправки
type LogEntry struct {
	User      string    `json:"user"`
	Timestamp time.Time `json:"timestamp"`
	LogType   string    `json:"log_type"` // system, shell, app
	Program   string    `json:"program"`
	Action    string    `json:"action"`
}

// IPCMessage - сообщение от CustomShell
type IPCMessage struct {
	Command string `json:"cmd"` // LOGIN, LOGOUT, LOG
	User    string `json:"user,omitempty"`
	Program string `json:"program,omitempty"`
	Action  string `json:"action,omitempty"`
}

// WSCommand - команда от сервера
type WSCommand struct {
	Type string `json:"type"`
}