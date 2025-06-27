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

// LogLevel определяет уровень логирования
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

// Initialize инициализирует систему логирования
func Initialize(logPath, level string) error {
	mu.Lock()
	
	// Устанавливаем уровень логирования
	currentLevel = parseLogLevel(level)

	// Создаем директорию для логов
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		mu.Unlock()
		return fmt.Errorf("ошибка создания директории для логов: %v", err)
	}

	// Открываем файл для записи логов
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		mu.Unlock()
		return fmt.Errorf("ошибка открытия файла логов: %v", err)
	}

	// Создаем multiwriter для записи в файл и консоль
	var writers []io.Writer
	writers = append(writers, logFile)
	
	// Если это не служба Windows, добавляем вывод в консоль
	if !isService() {
		writers = append(writers, os.Stdout)
	}

	multiWriter := io.MultiWriter(writers...)
	fileLogger = log.New(multiWriter, "", 0)

	// Пытаемся инициализировать Windows Event Log
	eventLogger, err = eventlog.Open("SchoolAgent")
	if err != nil {
		// Если не получилось открыть, пытаемся установить
		err = eventlog.InstallAsEventCreate("SchoolAgent", eventlog.Error|eventlog.Warning|eventlog.Info)
		if err != nil {
			// Если не получилось установить, продолжаем без Event Log
			// Освобождаем мьютекс перед вызовом Warn
			mu.Unlock()
			Warn("Не удалось инициализировать Windows Event Log: %v", err)
			mu.Lock()
		} else {
			eventLogger, err = eventlog.Open("SchoolAgent")
			if err != nil {
				// Освобождаем мьютекс перед вызовом Warn
				mu.Unlock()
				Warn("Не удалось открыть Windows Event Log после установки: %v", err)
				mu.Lock()
			}
		}
	}

	// Освобождаем мьютекс перед вызовом Info
	mu.Unlock()
	
	Info("Система логирования инициализирована (уровень: %s)", level)
	return nil
}

// Debug записывает сообщение уровня DEBUG
func Debug(format string, args ...interface{}) {
	logMessage(DEBUG, format, args...)
}

// Info записывает сообщение уровня INFO
func Info(format string, args ...interface{}) {
	logMessage(INFO, format, args...)
}

// Warn записывает сообщение уровня WARN
func Warn(format string, args ...interface{}) {
	logMessage(WARN, format, args...)
}

// Error записывает сообщение уровня ERROR
func Error(format string, args ...interface{}) {
	logMessage(ERROR, format, args...)
}

// logMessage записывает сообщение с указанным уровнем
func logMessage(level LogLevel, format string, args ...interface{}) {
	mu.RLock()
	defer mu.RUnlock()

	// Проверяем уровень логирования
	if level < currentLevel {
		return
	}

	// Форматируем сообщение
	message := fmt.Sprintf(format, args...)
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	levelStr := levelToString(level)
	
	logEntry := fmt.Sprintf("[%s] [%s] %s", timestamp, levelStr, message)

	// Записываем в файл
	if fileLogger != nil {
		fileLogger.Println(logEntry)
	}

	// Записываем в Windows Event Log
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

// parseLogLevel преобразует строку в LogLevel
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

// levelToString преобразует LogLevel в строку
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

// isService проверяет, запущена ли программа как служба Windows
func isService() bool {
	// Простая проверка - если нет консоли, значит это служба
	return os.Getenv("TERM") == ""
}

// Close закрывает систему логирования
func Close() {
	mu.Lock()
	defer mu.Unlock()

	if eventLogger != nil {
		eventLogger.Close()
		eventLogger = nil
	}
}

// LogProcessKill логирует завершение процесса
func LogProcessKill(pid uint32, path string, reason string) {
	Info("KILL_PROCESS: PID=%d, Path=%s, Reason=%s", pid, path, reason)
}

// LogProcessAllow логирует разрешение процесса
func LogProcessAllow(pid uint32, path string) {
	Debug("ALLOW_PROCESS: PID=%d, Path=%s", pid, path)
}

// LogWhitelistUpdate логирует обновление whitelist
func LogWhitelistUpdate(version string, itemCount int) {
	Info("WHITELIST_UPDATE: Version=%s, Items=%d", version, itemCount)
}

// LogError логирует ошибку с дополнительным контекстом
func LogError(operation string, err error) {
	Error("OPERATION_ERROR: Operation=%s, Error=%v", operation, err)
}

// LogStartup логирует запуск агента
func LogStartup() {
	Info("=== SCHOOL AGENT STARTUP ===")
}

// LogShutdown логирует остановку агента
func LogShutdown() {
	Info("=== SCHOOL AGENT SHUTDOWN ===")
}