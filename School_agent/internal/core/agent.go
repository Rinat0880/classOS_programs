package core

import (
	"bufio"
	"encoding/json"
	"log"
	"os"
	"school_agent/internal/config"
	"school_agent/internal/ipc"
	"school_agent/internal/logger"
	"school_agent/internal/models"
	"school_agent/internal/session"
	"school_agent/internal/ws"
	"time"
)

type Agent struct {
	cfg         *config.Config
	logMgr      *logger.Manager
	wsClient    *ws.Client
	ipcServer   *ipc.Server
	sessionMgr  *session.Manager
	
	currentUser string
	ipcChan     chan models.IPCMessage
	stopChan    chan struct{}
}

func New() *Agent {
	cfg := config.Load()
	ipcChan := make(chan models.IPCMessage, 10)

	return &Agent{
		cfg:        cfg,
		logMgr:     logger.New(cfg.LogDir),
		wsClient:   ws.New(cfg.ServerURL, cfg.DeviceToken, cfg.Hostname),
		ipcServer:  ipc.New(ipcChan),
		sessionMgr: session.New(cfg.ProjectBase),
		ipcChan:    ipcChan,
		stopChan:   make(chan struct{}),
	}
}

func (a *Agent) Run() {
	a.logMgr.Start()
	a.wsClient.Start(a.stopChan)
	a.ipcServer.Start()

	hbTicker := time.NewTicker(30 * time.Second)
	uploadTicker := time.NewTicker(1 * time.Hour)

	log.Println("Core Agent logic started")

	for {
		select {
		case <-a.stopChan:
			return

		// 1. Обработка сообщений от CustomShell
		case msg := <-a.ipcChan:
			a.handleIPC(msg)

		// 2. Обработка команд от Сервера (через WS)
		case cmd := <-a.wsClient.CommandChan:
			a.handleWSCommand(cmd)

		// 3. Heartbeat
		case <-hbTicker.C:
			a.wsClient.SendHeartbeat(a.currentUser)

		// 4. Периодическая выгрузка логов
		case <-uploadTicker.C:
			a.UploadLogs()
		}
	}
}

func (a *Agent) Wait() {
    <-a.stopChan
}

func (a *Agent) Stop() {
	close(a.stopChan)
}

func (a *Agent) handleIPC(msg models.IPCMessage) {
	switch msg.Command {
	case "LOGIN":
		a.currentUser = msg.User
		a.sessionMgr.PrepareUserEnvironment(msg.User)
		a.logMgr.Add(msg.User, "system", "agent", "Session Start")
		a.wsClient.SendHeartbeat(a.currentUser) // Сразу обновить статус
	case "LOGOUT":
		a.logMgr.Add(a.currentUser, "system", "agent", "Session End")
		a.sessionMgr.Cleanup(a.currentUser)
		a.currentUser = ""
		a.wsClient.SendHeartbeat("")
	case "LOG":
		a.logMgr.Add(a.currentUser, "shell", msg.Program, msg.Action)
	}
}

func (a *Agent) handleWSCommand(cmd models.WSCommand) {
	switch cmd.Type {
	case "UPLOAD_LOGS":
		go a.UploadLogs()
	case "GET_USER":
		a.wsClient.SendHeartbeat(a.currentUser)
	}
}

func (a *Agent) UploadLogs() {
	logFile := a.logMgr.GetCurrentLogFile()
	file, err := os.Open(logFile)
	if err != nil { return }
	defer file.Close()

	var logs []models.LogEntry
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var entry models.LogEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err == nil {
			logs = append(logs, entry)
		}
	}

	if len(logs) > 0 {
		payload := map[string]interface{}{
			"type": "logs",
			"data": logs,
		}
		a.wsClient.SendJSON(payload)
	}
}