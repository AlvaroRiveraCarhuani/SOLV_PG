package httpdelivery

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"solv-backend/internal/core/services"
)

var defaultUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     websocketOriginAllowed,
}

type WebSocketHandler struct {
	hub         *WebSocketHub
	authService *services.AuthService
	upgrader    websocket.Upgrader
}

func NewWebSocketHandler(hub *WebSocketHub, authService *services.AuthService) *WebSocketHandler {
	if hub != nil && authService != nil {
		hub.BindSessionRevocations(authService)
	}
	return &WebSocketHandler{
		hub:         hub,
		authService: authService,
		upgrader:    defaultUpgrader,
	}
}

func (h *WebSocketHandler) HandleEvaluationWS(w http.ResponseWriter, r *http.Request) {
	if !websocketOriginAllowed(r) {
		SendError(w, http.StatusForbidden, "forbidden origin", "Origen no permitido para WebSocket")
		return
	}

	principal, err := websocketSession(r, h.authService)
	if err != nil {
		SendError(w, http.StatusUnauthorized, "invalid session token", "Token de sesión inválido para WebSocket")
		return
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WebSocketHandler] Failed to upgrade connection: %v", err)
		return
	}
	if !principal.expiresAt.IsZero() {
		timer := time.AfterFunc(time.Until(principal.expiresAt), func() {
			_ = conn.WriteControl(
				websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "session expired"),
				time.Now().Add(writeWait),
			)
			_ = conn.Close()
		})
		defer timer.Stop()
	}

	client := &WebSocketClient{
		hub:      h.hub,
		conn:     conn,
		send:     make(chan []byte, 256),
		userID:   principal.userID,
		tenantID: principal.tenantID,
	}

	h.hub.RegisterSession(client, principal.sessionID)
	if h.authService != nil {
		revoked, checkErr := h.authService.IsSessionIDRevokedContext(r.Context(), principal.sessionID)
		if checkErr != nil || revoked {
			h.hub.CloseSession(principal.sessionID)
			return
		}
	}

	// Emitir confirmación de conexión inicial
	h.hub.EmitToUserInTenant(principal.tenantID, principal.userID, WebSocketMessage{
		Event: "CONNECTION_ESTABLISHED",
		Stage: "CONNECTED",
		Data: map[string]string{
			"user_id":   principal.userID,
			"tenant_id": principal.tenantID,
			"status":    "ready",
		},
	})

	go client.writePump()
	go client.readPump()
}
