	package main

	import (
		"context"
		"fmt"
		"log"
		"os"
		"os/signal"
		"syscall"
		"time"

		"school_agent/internal/config"
		"school_agent/internal/logger"
		"school_agent/internal/monitor"
		"school_agent/internal/whitelist"

		"golang.org/x/sys/windows/svc"
	)

	type service struct {
		ctx    context.Context
		cancel context.CancelFunc
		cfg    *config.Config
		wlm    *whitelist.Manager
		mon    *monitor.WindowsMonitor
	}

	func main() {
		inService, err := svc.IsWindowsService()
		if err != nil {
			log.Fatalf("Ошибка проверки режима службы: %v", err)
		}

		if inService {
			runService()
		} else {
			runConsole()
		}
	}

	func runService() {
		err := svc.Run("SchoolAgent", &service{})
		if err != nil {
			log.Fatalf("Ошибка запуска службы: %v", err)
		}
	}

	func runConsole() {
		fmt.Println("Запуск School Agent в консольном режиме...")
		
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		agent := &service{
			ctx:    ctx,
			cancel: cancel,
		}

		if err := agent.start(); err != nil {
			log.Fatalf("Ошибка запуска агента: %v", err)
		}

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		fmt.Println("Получен сигнал остановки...")
		agent.stop()
	}

	func (s *service) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (bool, uint32) {
		const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown | svc.AcceptPauseAndContinue
		changes <- svc.Status{State: svc.StartPending}

		s.ctx, s.cancel = context.WithCancel(context.Background())

		if err := s.start(); err != nil {
			changes <- svc.Status{State: svc.Stopped}
			return true, 1
		}

		changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

	loop:
		for {
			select {
			case c := <-r:
				switch c.Cmd {
				case svc.Interrogate:
					changes <- c.CurrentStatus
				case svc.Stop, svc.Shutdown:
					break loop
				case svc.Pause:
					changes <- svc.Status{State: svc.Paused, Accepts: cmdsAccepted}
				case svc.Continue:
					changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
				default:
					log.Printf("Неожиданная команда службы: %d", c.Cmd)
				}
			case <-s.ctx.Done():
				break loop
			}
		}

		changes <- svc.Status{State: svc.StopPending}
		s.stop()
		changes <- svc.Status{State: svc.Stopped}
		return false, 0
	}

	func (s *service) start() error {
		cfg, err := config.LoadConfig()
		if err != nil {
			return fmt.Errorf("ошибка загрузки конфигурации: %v", err)
		}
		s.cfg = cfg

		if err := logger.Initialize(cfg.LogPath, cfg.LogLevel); err != nil {
			return fmt.Errorf("ошибка инициализации логгера: %v", err)
		}

		logger.Info("Запуск School Agent...")

		s.wlm = whitelist.NewManager(cfg.WhitelistPath, cfg.WhitelistURL)
		if err := s.wlm.Initialize(); err != nil {
			logger.Error("Ошибка инициализации whitelist: %v", err)
			return err
		}

		s.mon = monitor.NewWindowsMonitor(s.processCallback)
		
		if err := s.mon.Start(s.ctx); err != nil {
			logger.Error("Ошибка запуска мониторинга: %v", err)
			return err
		}

		go s.whitelistUpdateLoop()

		logger.Info("School Agent успешно запущен")
		return nil
	}

	func (s *service) stop() {
		logger.Info("Остановка School Agent...")
		
		if s.cancel != nil {
			s.cancel()
		}

		if s.mon != nil {
			s.mon.Stop()
		}

		logger.Info("School Agent остановлен")
	}

	func (s *service) processCallback(pid uint32, path string) {
		logger.Debug("Обнаружен новый процесс: PID=%d, Path=%s", pid, path)

		if s.wlm.IsAllowed(path) {
			logger.Debug("Процесс разрешен: %s", path)
			return
		}

		logger.Warn("Процесс НЕ в whitelist, завершаем: PID=%d, Path=%s", pid, path)
		
		if err := s.mon.KillProcess(pid); err != nil {
			logger.Error("Ошибка завершения процесса PID=%d: %v", pid, err)
		} else {
			logger.Info("Процесс успешно завершен: PID=%d, Path=%s", pid, path)
		}
	}

	func (s *service) whitelistUpdateLoop() {
		ticker := time.NewTicker(time.Duration(s.cfg.UpdateInterval) * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				logger.Debug("Обновление whitelist...")
				if err := s.wlm.Update(); err != nil {
					logger.Error("Ошибка обновления whitelist: %v", err)
				} else {
					logger.Info("Whitelist успешно обновлен")
				}
			case <-s.ctx.Done():
				return
			}
		}
	}