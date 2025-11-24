package ws

import (
	"log"
	"school_agent/internal/models"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	url      string
	token    string
	hostname string
	conn     *websocket.Conn
	mu       sync.Mutex
	
	// Канал для передачи команд в Core
	CommandChan chan models.WSCommand
}

func New(url, token, hostname string) *Client {
	return &Client{
		url:         url,
		token:       token,
		hostname:    hostname,
		CommandChan: make(chan models.WSCommand, 10),
	}
}

func (c *Client) Start(stopChan chan struct{}) {
	go c.connectLoop(stopChan)
}

func (c *Client) connectLoop(stopChan chan struct{}) {
	for {
		select {
		case <-stopChan:
			return
		default:
			conn, _, err := websocket.DefaultDialer.Dial(c.url, nil)
			if err != nil {
				time.Sleep(10 * time.Second)
				continue
			}

			c.mu.Lock()
			c.conn = conn
			c.mu.Unlock()

			// Auth
			c.SendJSON(map[string]string{"type": "auth", "token": c.token})
			
			log.Println("WS Connected")

			// Чтение сообщений
			for {
				var cmd models.WSCommand
				err := conn.ReadJSON(&cmd)
				if err != nil {
					break // Разрыв
				}
				c.CommandChan <- cmd
			}
			
			c.mu.Lock()
			c.conn = nil
			c.mu.Unlock()
		}
	}
}

func (c *Client) SendHeartbeat(user string) {
	payload := map[string]interface{}{
		"type":      "heartbeat",
		"device":    c.hostname,
		"user":      user,
		"timestamp": time.Now(),
	}
	c.SendJSON(payload)
}

func (c *Client) SendJSON(v interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return nil
	}
	return c.conn.WriteJSON(v)
}