package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/sys/windows/svc/eventlog"
)

type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

var (
	currentLevel LogLevel
	fileLogger   *log.Logger
	eventLogger  *eventlog.Log
	mu           sync.RWMutex
)

func Initialize(logPath, level string) error {
	mu.Lock()
	
	currentLevel = parseLogLevel(level)

	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		mu.Unlock()
		return fmt.Errorf("ошибка создания директории для логов: %v", err)
	}

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		mu.Unlock()
		return fmt.Errorf("ошибка открытия файла логов: %v", err)
	}

	var writers []io.Writer
	writers = append(writers, logFile)
	
	if !isService() {
		writers = append(writers, os.Stdout)
	}

	multiWriter := io.MultiWriter(writers...)
	fileLogger = log.New(multiWriter, "", 0)

	eventLogger, err = eventlog.Open("SchoolAgent")
	if err != nil {
		err = eventlog.InstallAsEventCreate("SchoolAgent", eventlog.Error|eventlog.Warning|eventlog.Info)
		if err != nil {
			mu.Unlock()
			Warn("Не удалось инициализировать Windows Event Log: %v", err)
			mu.Lock()
		} else {
			eventLogger, err = eventlog.Open("SchoolAgent")
			if err != nil {
				mu.Unlock()
				Warn("Не удалось открыть Windows Event Log после установки: %v", err)
				mu.Lock()
			}
		}
	}

	mu.Unlock()
	
	Info("Система логирования инициализирована (уровень: %s)", level)
	return nil
}

func Debug(format string, args ...interface{}) {
	logMessage(DEBUG, format, args...)
}

func Info(format string, args ...interface{}) {
	logMessage(INFO, format, args...)
}

func Warn(format string, args ...interface{}) {
	logMessage(WARN, format, args...)
}

func Error(format string, args ...interface{}) {
	logMessage(ERROR, format, args...)
}

func logMessage(level LogLevel, format string, args ...interface{}) {
	mu.RLock()
	defer mu.RUnlock()

	if level < currentLevel {
		return
	}

	message := fmt.Sprintf(format, args...)
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	levelStr := levelToString(level)
	
	logEntry := fmt.Sprintf("[%s] [%s] %s", timestamp, levelStr, message)

	if fileLogger != nil {
		fileLogger.Println(logEntry)
	}

	if eventLogger != nil {
		switch level {
		case DEBUG, INFO:
			eventLogger.Info(1, message)
		case WARN:
			eventLogger.Warning(2, message)
		case ERROR:
			eventLogger.Error(3, message)
		}
	}
}

func parseLogLevel(level string) LogLevel {
	switch strings.ToLower(level) {
	case "debug":
		return DEBUG
	case "info":
		return INFO
	case "warn", "warning":
		return WARN
	case "error":
		return ERROR
	default:
		return INFO
	}
}

func levelToString(level LogLevel) string {
	switch level {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

func isService() bool {
	return os.Getenv("TERM") == ""
}

func Close() {
	mu.Lock()
	defer mu.Unlock()

	if eventLogger != nil {
		eventLogger.Close()
		eventLogger = nil
	}
}

func LogProcessKill(pid uint32, path string, reason string) {
	Info("KILL_PROCESS: PID=%d, Path=%s, Reason=%s", pid, path, reason)
}

func LogProcessAllow(pid uint32, path string) {
	Debug("ALLOW_PROCESS: PID=%d, Path=%s", pid, path)
}

func LogWhitelistUpdate(version string, itemCount int) {
	Info("WHITELIST_UPDATE: Version=%s, Items=%d", version, itemCount)
}

func LogError(operation string, err error) {
	Error("OPERATION_ERROR: Operation=%s, Error=%v", operation, err)
}

func LogStartup() {
	Info("=== SCHOOL AGENT STARTUP ===")
}

func LogShutdown() {
	Info("=== SCHOOL AGENT SHUTDOWN ===")
}