package core

import (
	"bufio"
	"encoding/json"
	"log"
	"os"
	"school_agent/internal/config"
	"school_agent/internal/logger"
	"school_agent/internal/models"
	"school_agent/internal/monitor"
	"school_agent/internal/session"
	"school_agent/internal/sysuser"
	"school_agent/internal/ws"
	"strings"
	"time"
)

type Agent struct {
	cfg         *config.Config
	logMgr      *logger.Manager
	wsClient    *ws.Client
	sessionMgr  *session.Manager
	
	procMonitor    *monitor.ProcessMonitor
	browserMonitor *monitor.BrowserMonitor
	
	currentUser string
	stopChan    chan struct{}
}

func New() *Agent {
	cfg := config.Load()

	agent := &Agent{
		cfg:        cfg,
		logMgr:     logger.New(cfg.LogDir, cfg.Hostname),
		wsClient:   ws.New(cfg.ServerURL, cfg.DeviceToken, cfg.Hostname),
		sessionMgr: session.New(cfg.ProjectBase),
		stopChan:   make(chan struct{}),
	}

	agent.procMonitor = monitor.NewProcessMonitor(func(action, program string) {
		agent.logMgr.Add(agent.currentUser, "process", program, action)
	})

	agent.browserMonitor = monitor.NewBrowserMonitor("", func(browser, action string) {
		agent.logMgr.Add(agent.currentUser, "browser", browser, action)
	})

	return agent
}

func (a *Agent) Run() {
	a.logMgr.Start()
	a.wsClient.Start(a.stopChan)

	a.detectAndUpdateUser()

	a.procMonitor.Start()
	a.browserMonitor.Start()

	hbTicker := time.NewTicker(30 * time.Second)
	uploadTicker := time.NewTicker(10 * time.Minute)
	userCheckTicker := time.NewTicker(30 * time.Second)

	log.Println("Core Agent logic started")

	for {
		select {
		case <-a.stopChan:
			return

		case cmd := <-a.wsClient.CommandChan:
			a.handleWSCommand(cmd)

		case <-hbTicker.C:
			a.wsClient.SendHeartbeat(a.currentUser)

		case <-uploadTicker.C:
			a.UploadLogs()

		case <-userCheckTicker.C:
			a.detectAndUpdateUser()
		}
	}
}

func (a *Agent) Wait() {
    <-a.stopChan
}

func (a *Agent) Stop() {
	close(a.stopChan)
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
		for i := range logs {
			if logs[i].DeviceName == "" {
				logs[i].DeviceName = a.cfg.Hostname
			}
		}

		payload := map[string]interface{}{
			"type": "logs",
			"data": logs,
		}
		a.wsClient.SendJSON(payload)
		log.Printf("Uploaded %d logs to server", len(logs))
	}
}

func (a *Agent) detectAndUpdateUser() {
	user, err := sysuser.GetActiveUser()
	if err != nil {
		log.Printf("Could not detect active user: %v", err)
		return
	}

	user = a.cleanUsername(user)

	if user == "" {
		if a.currentUser != "" {
			log.Printf("User logged out: %s", a.currentUser)
			a.logMgr.Add(a.currentUser, "system", "agent", "Session End")
			a.currentUser = ""
			a.browserMonitor.UpdateUsername("")
			a.wsClient.SendHeartbeat("")
		}
		return
	}

	if user != a.currentUser {
		if a.currentUser != "" {
			log.Printf("User changed: %s -> %s", a.currentUser, user)
			a.logMgr.Add(a.currentUser, "system", "agent", "Session End")
		} else {
			log.Printf("User logged in: %s", user)
		}
		
		a.currentUser = user
		a.browserMonitor.UpdateUsername(user)
		a.sessionMgr.PrepareUserEnvironment(user)
		a.logMgr.Add(user, "system", "agent", "Session Start")
		a.wsClient.SendHeartbeat(a.currentUser)
	}
}
func (a *Agent) cleanUsername(username string) string {
	if idx := strings.Index(username, "\\"); idx != -1 {
		return username[idx+1:]
	}
	return username
}
