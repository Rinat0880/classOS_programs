package ipc

import (
	"encoding/json"
	"net"
	"school_agent/internal/config"
	"school_agent/internal/models"

	"github.com/Microsoft/go-winio"
)

type Server struct {
	msgChan chan models.IPCMessage
}

func New(msgChan chan models.IPCMessage) *Server {
	return &Server{msgChan: msgChan}
}

func (s *Server) Start() {
	go func() {
		l, err := winio.ListenPipe(config.PipeName, nil)
		if err != nil {
			return
		}
		defer l.Close()
		
		for {
			conn, err := l.Accept()
			if err != nil {
				continue
			}
			go s.handleConn(conn)
		}
	}()
}

func (s *Server) handleConn(c net.Conn) {
	defer c.Close()
	decoder := json.NewDecoder(c)
	var msg models.IPCMessage
	if err := decoder.Decode(&msg); err == nil {
		s.msgChan <- msg
	}
}