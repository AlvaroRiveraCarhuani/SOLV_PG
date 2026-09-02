package httpdelivery

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512 * 1024
)

type WebSocketMessage struct {
	Event        string      `json:"event"`
	SubmissionID string      `json:"submission_id,omitempty"`
	Stage        string      `json:"stage"`
	Data         interface{} `json:"data,omitempty"`
	Timestamp    time.Time   `json:"timestamp"`
}

type WebSocketClient struct {
	hub      *WebSocketHub
	conn     *websocket.Conn
	send     chan []byte
	userID   string
	tenantID string
}

func (c *WebSocketClient) readPump() {
	defer func() {
		c.hub.Unregister(c)
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[WebSocketHub] Unexpected close for user %s: %v", c.userID, err)
			}
			break
		}

		var incoming map[string]interface{}
		if err := json.Unmarshal(message, &incoming); err == nil {
			if incoming["action"] == "ping" {
				c.hub.EmitToUser(c.userID, WebSocketMessage{
					Event:     "PONG",
					Stage:     "PONG",
					Timestamp: time.Now(),
				})
			}
		}
	}
}

func (c *WebSocketClient) writePump() {
	ticker := time.NewTicker(25 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			_, _ = w.Write(message)

			n := len(c.send)
			for i := 0; i < n; i++ {
				_, _ = w.Write([]byte{'\n'})
				_, _ = w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

type WebSocketHub struct {
	clients     map[*WebSocketClient]bool
	userClients map[string]map[*WebSocketClient]bool
	broadcast   chan WebSocketMessage
	mu          sync.RWMutex
}

func NewWebSocketHub() *WebSocketHub {
	return &WebSocketHub{
		clients:     make(map[*WebSocketClient]bool),
		userClients: make(map[string]map[*WebSocketClient]bool),
		broadcast:   make(chan WebSocketMessage, 256),
	}
}

func (h *WebSocketHub) Run() {
	for msg := range h.broadcast {
		data, err := json.Marshal(msg)
		if err != nil {
			continue
		}

		h.mu.RLock()
		for client := range h.clients {
			select {
			case client.send <- data:
			default:
				// Canal saturado
			}
		}
		h.mu.RUnlock()
	}
}

func (h *WebSocketHub) Register(client *WebSocketClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[client] = true
	if _, exists := h.userClients[client.userID]; !exists {
		h.userClients[client.userID] = make(map[*WebSocketClient]bool)
	}
	h.userClients[client.userID][client] = true
}

func (h *WebSocketHub) Unregister(client *WebSocketClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)
		close(client.send)
	}
	if userMap, exists := h.userClients[client.userID]; exists {
		delete(userMap, client)
		if len(userMap) == 0 {
			delete(h.userClients, client.userID)
		}
	}
}

func (h *WebSocketHub) EmitToUser(userID string, msg WebSocketMessage) {
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now()
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	if userMap, exists := h.userClients[userID]; exists {
		for client := range userMap {
			select {
			case client.send <- data:
			default:
			}
		}
	}
}

func (h *WebSocketHub) Broadcast(msg WebSocketMessage) {
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now()
	}
	h.broadcast <- msg
}
