package httpdelivery

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
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
	hub       *WebSocketHub
	conn      *websocket.Conn
	send      chan []byte
	userID    string
	tenantID  string
	sessionID string
}

func (c *WebSocketClient) readPump() {
	defer func() {
		c.hub.Unregister(c)
		_ = c.conn.Close()
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
				c.hub.EmitToUserInTenant(c.tenantID, c.userID, WebSocketMessage{
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
		_ = c.conn.Close()
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
	clients        map[*WebSocketClient]bool
	userClients    map[string]map[*WebSocketClient]bool
	sessionClients map[string]map[*WebSocketClient]bool
	broadcast      chan WebSocketMessage
	mu             sync.RWMutex
}

func NewWebSocketHub() *WebSocketHub {
	return &WebSocketHub{
		clients:        make(map[*WebSocketClient]bool),
		userClients:    make(map[string]map[*WebSocketClient]bool),
		sessionClients: make(map[string]map[*WebSocketClient]bool),
		broadcast:      make(chan WebSocketMessage, 256),
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
	h.RegisterSession(client, "")
}

// RegisterSession indexa al cliente por tenant y usuario y recuerda el hash de
// su token para poder cerrar exactamente esa conexión cuando se revoque.
func (h *WebSocketHub) RegisterSession(client *WebSocketClient, sessionID string) {
	client.sessionID = sessionID
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[client] = true
	key := webSocketRecipientKey(client.tenantID, client.userID)
	if _, exists := h.userClients[key]; !exists {
		h.userClients[key] = make(map[*WebSocketClient]bool)
	}
	h.userClients[key][client] = true
	if sessionID != "" {
		if _, exists := h.sessionClients[sessionID]; !exists {
			h.sessionClients[sessionID] = make(map[*WebSocketClient]bool)
		}
		h.sessionClients[sessionID][client] = true
	}
}

func (h *WebSocketHub) Unregister(client *WebSocketClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)
		close(client.send)
	}
	key := webSocketRecipientKey(client.tenantID, client.userID)
	if userMap, exists := h.userClients[key]; exists {
		delete(userMap, client)
		if len(userMap) == 0 {
			delete(h.userClients, key)
		}
	}
	if client.sessionID != "" {
		if sessionMap, exists := h.sessionClients[client.sessionID]; exists {
			delete(sessionMap, client)
			if len(sessionMap) == 0 {
				delete(h.sessionClients, client.sessionID)
			}
		}
	}
}

func webSocketRecipientKey(tenantID, userID string) string {
	return tenantID + "\x00" + userID
}

func (h *WebSocketHub) EmitToUserInTenant(tenantID, userID string, msg WebSocketMessage) {
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now()
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	if userMap, exists := h.userClients[webSocketRecipientKey(tenantID, userID)]; exists {
		for client := range userMap {
			select {
			case client.send <- data:
			default:
			}
		}
	}
}

// CloseSession cierra las conexiones emitidas con el token revocado. El cierre
// de la conexión dispara la desuscripción normal de cada readPump.
func (h *WebSocketHub) CloseSession(sessionID string) {
	h.mu.Lock()
	clients := make([]*WebSocketClient, 0, len(h.sessionClients[sessionID]))
	for client := range h.sessionClients[sessionID] {
		clients = append(clients, client)
	}
	delete(h.sessionClients, sessionID)
	h.mu.Unlock()

	h.closeClients(clients)
}

func (h *WebSocketHub) closeClients(clients []*WebSocketClient) {
	for _, client := range clients {
		h.Unregister(client)
		if client.conn == nil {
			continue
		}
		_ = client.conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "session revoked"),
			time.Now().Add(writeWait),
		)
		_ = client.conn.Close()
	}
}

// CloseAll se usa cuando no es posible comprobar el estado de revocación. Es
// preferible cortar las conexiones locales a seguir entregando eventos a
// sesiones no verificables.
func (h *WebSocketHub) CloseAll() {
	h.mu.RLock()
	clients := make([]*WebSocketClient, 0, len(h.clients))
	for client := range h.clients {
		clients = append(clients, client)
	}
	h.mu.RUnlock()
	h.closeClients(clients)
}

func (h *WebSocketHub) CloseSessions(sessionIDs []string) {
	for _, sessionID := range sessionIDs {
		h.CloseSession(sessionID)
	}
}

// StartRevocationSync propaga las revocaciones hechas por otras instancias.
// La base de datos es la fuente compartida; este sondeo acota el retraso de
// cierre en las conexiones ya abiertas. La autorización posterior siempre
// consulta la misma base y no depende de este sondeo.
func (h *WebSocketHub) StartRevocationSync(ctx context.Context, store domain.SessionRevocationRepository, interval time.Duration) {
	if store == nil {
		h.CloseAll()
		return
	}
	if interval <= 0 {
		interval = services.SessionRevocationPollInterval
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				sessionIDs, err := store.ListRevokedSessions(ctx)
				if err != nil {
					log.Printf("[WebSocketHub] No se pudo consultar revocaciones; se cierran conexiones locales: %v", err)
					h.CloseAll()
					continue
				}
				h.CloseSessions(sessionIDs)
			}
		}
	}()
}

// BindSessionRevocations conecta el registro de sesiones revocadas del
// servicio de auth con el cierre de las conexiones del hub.
func (h *WebSocketHub) BindSessionRevocations(authService *services.AuthService) func() {
	if authService == nil {
		return func() {}
	}
	return authService.RegisterRevocationListener(h.CloseSession)
}

func (h *WebSocketHub) Broadcast(msg WebSocketMessage) {
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now()
	}
	h.broadcast <- msg
}
