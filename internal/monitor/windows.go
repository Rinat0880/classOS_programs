package monitor

import (
	"context"
	"fmt"
	"time"

	"school_agent/internal/logger"

	"github.com/StackExchange/wmi"
	"golang.org/x/sys/windows"
)

type ProcessCallback func(pid uint32, path string)

type WindowsMonitor struct {
	callback   ProcessCallback
	ctx        context.Context
	cancel     context.CancelFunc
	isRunning  bool
	knownPIDs  map[uint32]bool
	scanTicker *time.Ticker
	dryRun     bool
}

type Win32_Process struct {
	ProcessId      uint32
	Name           string
	ExecutablePath *string
}

func NewWindowsMonitor(callback ProcessCallback) *WindowsMonitor {
	return &WindowsMonitor{
		callback:  callback,
		knownPIDs: make(map[uint32]bool),
		dryRun:    isDryRunMode(),
	}
}

func (m *WindowsMonitor) Start(ctx context.Context) error {
	m.ctx, m.cancel = context.WithCancel(ctx)

	logger.Info("Запуск мониторинга процессов Windows...")

	if err := m.enableDebugPrivilege(); err != nil {
		logger.Warn("Не удалось включить SeDebugPrivilege: %v", err)
	}

	if err := m.scanExistingProcesses(); err != nil {
		logger.Error("Ошибка сканирования существующих процессов: %v", err)
	}

	go m.startPeriodicScanning()

	m.isRunning = true
	logger.Info("Мониторинг процессов запущен (режим периодического сканирования)")
	return nil
}

func (m *WindowsMonitor) Stop() {
	if !m.isRunning {
		return
	}

	logger.Info("Остановка мониторинга процессов...")

	if m.cancel != nil {
		m.cancel()
	}

	if m.scanTicker != nil {
		m.scanTicker.Stop()
	}

	m.isRunning = false
	logger.Info("Мониторинг процессов остановлен")
}

func (m *WindowsMonitor) startPeriodicScanning() {
	m.scanTicker = time.NewTicker(1 * time.Second)
	defer m.scanTicker.Stop()

	logger.Info("Запущено периодическое сканирование процессов (интервал: 1 секунда)")

	for {
		select {
		case <-m.ctx.Done():
			logger.Info("Периодическое сканирование остановлено")
			return
		case <-m.scanTicker.C:
			m.scanForNewProcesses()
		}
	}
}

func (m *WindowsMonitor) scanExistingProcesses() error {
	var processes []Win32_Process
	query := "SELECT ProcessId, Name, ExecutablePath FROM Win32_Process"

	if err := wmi.Query(query, &processes); err != nil {
		return fmt.Errorf("ошибка получения списка процессов: %v", err)
	}

	logger.Info("Найдено %d существующих процессов", len(processes))

	for _, process := range processes {
		m.knownPIDs[process.ProcessId] = true
	}

	return nil
}

func (m *WindowsMonitor) scanForNewProcesses() {
	var processes []Win32_Process
	query := "SELECT ProcessId, Name, ExecutablePath FROM Win32_Process"

	if err := wmi.Query(query, &processes); err != nil {
		logger.Error("Ошибка сканирования процессов: %v", err)
		return
	}

	newProcessCount := 0
	for _, process := range processes {
		if !m.knownPIDs[process.ProcessId] {
			m.knownPIDs[process.ProcessId] = true
			m.handleProcessCreation(&process)
			newProcessCount++
		}
	}

	if newProcessCount > 0 {
		logger.Debug("Обнаружено %d новых процессов", newProcessCount)
	}

	m.cleanupTerminatedProcesses(processes)
}

func (m *WindowsMonitor) cleanupTerminatedProcesses(currentProcesses []Win32_Process) {
	currentPIDs := make(map[uint32]bool)
	for _, process := range currentProcesses {
		currentPIDs[process.ProcessId] = true
	}

	for pid := range m.knownPIDs {
		if !currentPIDs[pid] {
			delete(m.knownPIDs, pid)
		}
	}
}

func (m *WindowsMonitor) handleProcessCreation(process *Win32_Process) {
	path := m.getProcessPath(process)
	if path == "" {
		logger.Debug("Не удалось получить путь для процесса PID=%d, Name=%s",
			process.ProcessId, process.Name)
		return
	}

	if m.shouldIgnoreProcess(path, process.Name) {
		logger.Debug("Игнорируется системный процесс: PID=%d, Name=%s",
			process.ProcessId, process.Name)
		return
	}

	logger.Info("Обнаружен новый процесс: PID=%d, Name=%s, Path=%s",
		process.ProcessId, process.Name, path)

	if m.callback != nil {
		m.callback(process.ProcessId, path)
	}
}

func (m *WindowsMonitor) getProcessPath(process *Win32_Process) string {
	if process.ExecutablePath != nil && *process.ExecutablePath != "" {
		return *process.ExecutablePath
	}

	return m.queryProcessImageName(process.ProcessId)
}

func (m *WindowsMonitor) queryProcessImageName(pid uint32) string {
	handle, err := windows.OpenProcess(
		windows.PROCESS_QUERY_LIMITED_INFORMATION,
		false,
		pid,
	)
	if err != nil {
		return ""
	}
	defer windows.CloseHandle(handle)

	var pathBuffer [windows.MAX_PATH]uint16
	bufferSize := uint32(len(pathBuffer))

	err = windows.QueryFullProcessImageName(
		handle,
		0,
		&pathBuffer[0],
		&bufferSize,
	)
	if err != nil {
		return ""
	}

	return windows.UTF16ToString(pathBuffer[:bufferSize])
}

func (m *WindowsMonitor) shouldIgnoreProcess(path, name string) bool {
	if path == "" {
		return true
	}

	systemProcesses := []string{
		"System",
		"Registry",
		"smss.exe",
		"csrss.exe",
		"wininit.exe",
		"winlogon.exe",
		"services.exe",
		"lsass.exe",
		"svchost.exe",
		"dwm.exe",
		"explorer.exe",
		"RuntimeBroker.exe",
		"WmiPrvSE.exe",
		"dllhost.exe",
		"conhost.exe",
		"cmd.exe",
	}

	for _, sysProc := range systemProcesses {
		if name == sysProc {
			return true
		}
	}

	if name == "school-agent.exe" || name == "agent.exe" || name == "main.exe" {
		return true
	}

	return false
}

func (m *WindowsMonitor) KillProcess(pid uint32) error {
	if m.dryRun {
		logger.Info("DRY RUN: Процесс PID=%d НЕ был завершен (режим тестирования)", pid)
		return nil
	}

	logger.Info("Завершение процесса PID=%d", pid)

	handle, err := windows.OpenProcess(
		windows.PROCESS_TERMINATE,
		false,
		pid,
	)
	if err != nil {
		return fmt.Errorf("не удалось открыть процесс PID=%d: %v", pid, err)
	}
	defer windows.CloseHandle(handle)

	err = windows.TerminateProcess(handle, 1)
	if err != nil {
		return fmt.Errorf("не удалось завершить процесс PID=%d: %v", pid, err)
	}

	delete(m.knownPIDs, pid)

	logger.Info("Процесс PID=%d успешно завершен", pid)
	return nil
}

func (m *WindowsMonitor) enableDebugPrivilege() error {
	var token windows.Token

	err := windows.OpenProcessToken(
		windows.CurrentProcess(),
		windows.TOKEN_ADJUST_PRIVILEGES|windows.TOKEN_QUERY,
		&token,
	)
	if err != nil {
		return fmt.Errorf("не удалось получить токен процесса: %v", err)
	}
	defer token.Close()

	var luid windows.LUID
	err = windows.LookupPrivilegeValue(nil, windows.StringToUTF16Ptr("SeDebugPrivilege"), &luid)
	if err != nil {
		return fmt.Errorf("не удалось найти SeDebugPrivilege: %v", err)
	}

	privileges := windows.Tokenprivileges{
		PrivilegeCount: 1,
		Privileges: [1]windows.LUIDAndAttributes{
			{
				Luid:       luid,
				Attributes: windows.SE_PRIVILEGE_ENABLED,
			},
		},
	}

	err = windows.AdjustTokenPrivileges(
		token,
		false,
		&privileges,
		0,
		nil,
		nil,
	)
	if err != nil {
		return fmt.Errorf("не удалось включить SeDebugPrivileg: %v", err)
	}

	logger.Info("SeDebugPrivilege успешно включена")
	return nil
}

func isDryRunMode() bool {
	return false
}

func (m *WindowsMonitor) GetRunningProcesses() ([]ProcessInfo, error) {
	var processes []Win32_Process
	query := "SELECT ProcessId, Name, ExecutablePath FROM Win32_Process"

	if err := wmi.Query(query, &processes); err != nil {
		return nil, fmt.Errorf("ошибка получения списка процессов: %v", err)
	}

	result := make([]ProcessInfo, 0, len(processes))
	for _, process := range processes {
		path := m.getProcessPath(&process)
		result = append(result, ProcessInfo{
			PID:  process.ProcessId,
			Name: process.Name,
			Path: path,
		})
	}

	return result, nil
}

type ProcessInfo struct {
	PID  uint32
	Name string
	Path string
	PPID uint32
}
