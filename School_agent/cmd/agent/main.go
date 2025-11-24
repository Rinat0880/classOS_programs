package main

import (
	"log"
	"os"
	"school_agent/internal/sysuser" 
	"school_agent/internal/winsvc"
	"time"

	"github.com/kardianos/service"
)

func main() {
	// 1. Логгер сервиса (service.log)
	logFile, err := os.OpenFile("C:\\ProgramData\\SchoolAgent\\service.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		log.SetOutput(logFile)
	}

	// === НОВАЯ ЛОГИКА: ОПРЕДЕЛЕНИЕ ROOT USER ===
	log.Println("------------------------------------------------")
	log.Println("Service Starting...")
	
	// Пытаемся узнать реального пользователя (повторяем попытки, т.к. при старте ПК пользователь может еще не войти)
	go monitorActiveUser()
	// ===========================================

	svcConfig := &service.Config{
		Name:        "SchoolAgent",
		DisplayName: "School System Agent",
		Description: "Monitoring Agent",
	}

	prg := &winsvc.ServiceProgram{}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		log.Fatal(err)
	}

	if len(os.Args) > 1 {
		err = service.Control(s, os.Args[1])
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	err = s.Run()
	if err != nil {
		log.Fatal(err)
	}
}

// Функция мониторинга (запускается в горутине)
func monitorActiveUser() {
	// Даем системе прогрузиться
	time.Sleep(2 * time.Second)

	user, err := sysuser.GetActiveUser()
	if err != nil {
		log.Printf("[SYSTEM CHECK] Could not detect active user yet: %v", err)
	} else if user == "" {
		log.Printf("[SYSTEM CHECK] No user currently logged in on console.")
	} else {
		log.Printf("[SYSTEM CHECK] ACTIVE ROOT USER DETECTED: %s", user)
	}
}