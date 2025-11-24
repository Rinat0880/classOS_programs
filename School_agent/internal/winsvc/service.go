package winsvc

import (
	"log"
	"school_agent/internal/core"

	"github.com/kardianos/service"
)

type ServiceProgram struct {
	Agent *core.Agent
}

func (p *ServiceProgram) Start(s service.Service) error {
	log.Println("Service Starting...")
	p.Agent = core.New()
	log.Println("Service Starting2...")
	go p.Agent.Run()
	log.Println("Service Starting3...")
	// блокируем Windows, пока агент не получит Stop()
	
	log.Println("waiting...")
	p.Agent.Wait()

	
	log.Println("No not waiting...")
	return nil
}

func (p *ServiceProgram) Stop(s service.Service) error {
	log.Println("Service Stopping...")
	if p.Agent != nil {
		p.Agent.Stop()
	}
	return nil
}
