package models

import "time"

type LogEntry struct {
	Username   string    `json:"username"`
	DeviceName string    `json:"device_name"`
	Timestamp  time.Time `json:"timestamp"`
	LogType    string    `json:"log_type"` 
	Program    string    `json:"program"`
	Action     string    `json:"action"`
}

type IPCMessage struct {
	Command string `json:"cmd"` 
	User    string `json:"user,omitempty"`
	Program string `json:"program,omitempty"`
	Action  string `json:"action,omitempty"`
}

type WSCommand struct {
	Type string `json:"type"`
}