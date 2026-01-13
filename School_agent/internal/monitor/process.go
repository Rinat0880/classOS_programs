package monitor

import (
	"log"
	"time"

	"github.com/shirou/gopsutil/v3/process"
)

type ProcessMonitor struct {
	processes map[int32]string
	callback  func(action, program string)
}

func NewProcessMonitor(callback func(action, program string)) *ProcessMonitor {
	return &ProcessMonitor{
		processes: make(map[int32]string),
		callback:  callback,
	}
}

func (pm *ProcessMonitor) Start() {
	go pm.monitorLoop()
}

func (pm *ProcessMonitor) monitorLoop() {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		pm.checkProcesses()
	}
}

func (pm *ProcessMonitor) checkProcesses() {
	currentProcs := make(map[int32]string)
	
	procs, err := process.Processes()
	if err != nil {
		return
	}

	for _, p := range procs {
		name, err := p.Name()
		if err != nil {
			continue
		}

		if pm.isImportantProcess(name) {
			currentProcs[p.Pid] = name

			if _, exists := pm.processes[p.Pid]; !exists {
				pm.callback("Opened", name)
				log.Printf("Process started: %s (PID: %d)", name, p.Pid)
			}
		}
	}

	for pid, name := range pm.processes {
		if _, exists := currentProcs[pid]; !exists {
			pm.callback("Closed", name)
			log.Printf("Process ended: %s (PID: %d)", name, pid)
		}
	}

	pm.processes = currentProcs
}

func (pm *ProcessMonitor) isImportantProcess(name string) bool {
	important := map[string]bool{
		"chrome.exe":          true,
		"msedge.exe":          true,
		"firefox.exe":         true,
		"Code.exe":            true,
		"notepad.exe":         true,
		"notepad++.exe":       true,
		"WINWORD.EXE":         true,
		"EXCEL.EXE":           true,
		"POWERPNT.EXE":        true,
		"AcroRd32.exe":        true,
		"Acrobat.exe":         true,
		"PhotoshopCC.exe":     true,
		"Photoshop.exe":       true,
		"Illustrator.exe":     true,
		"vlc.exe":             true,
		"steam.exe":           true,
		"Discord.exe":         true,
		"Telegram.exe":        true,
		"Spotify.exe":         true,
		"cmd.exe":             true,
		"powershell.exe":      true,
		"python.exe":          true,
		"java.exe":            true,
		"javaw.exe":           true,
		"node.exe":            true,
		"git.exe":             true,
		"VisualStudio.exe":    true,
		"devenv.exe":          true,
		"Slack.exe":           true,
		"Teams.exe":           true,
		"Zoom.exe":            true,
	}

	return important[name]
}
