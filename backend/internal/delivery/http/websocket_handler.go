package httpdelivery

import (
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
	"solv-backend/internal/core/services"
)

var defaultUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Permite conexiones desde cualquier origen / subdominio de tenant
	},
}

type WebSocketHandler struct {
	hub         *WebSocketHub
	authService *services.AuthService
	upgrader    websocket.Upgrader
}

func NewWebSocketHandler(hub *WebSocketHub, authService *services.AuthService) *WebSocketHandler {
	return &WebSocketHandler{
		hub:         hub,
		authService: authService,
		upgrader:    defaultUpgrader,
	}
}

func (h *WebSocketHandler) HandleEvaluationWS(w http.ResponseWriter, r *http.Request) {
	var tokenStr string
	// 1. Extraer token de Query param, Header o Cookie
	if qToken := r.URL.Query().Get("token"); qToken != "" {
		tokenStr = qToken
	} else if authHeader := r.Header.Get("Authorization"); strings.HasPrefix(authHeader, "Bearer ") {
		tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
	} else if cookie, err := r.Cookie("solv_session"); err == nil && cookie.Value != "" {
		tokenStr = cookie.Value
	}

	userID := "anonymous"
	tenantID := "00000000-0000-0000-0000-000000000001"

	if tokenStr != "" && h.authService != nil {
		claims, err := h.authService.ValidateSessionToken(tokenStr)
		if err != nil {
			SendError(w, http.StatusUnauthorized, "invalid session token", "Token de sesión inválido para WebSocket")
			return
		}
		if uID, ok := claims["user_id"].(string); ok && uID != "" {
			userID = uID
		}
		if tID, ok := claims["tenant_id"].(string); ok && tID != "" {
			tenantID = tID
		}
	} else {
		// Fallback para headers o testing
		if hUID := r.Header.Get("X-User-Id"); hUID != "" {
			userID = hUID
		}
		if hTID := r.Header.Get("X-Tenant-Id"); hTID != "" {
			tenantID = hTID
		}
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WebSocketHandler] Failed to upgrade connection: %v", err)
		return
	}

	client := &WebSocketClient{
		hub:      h.hub,
		conn:     conn,
		send:     make(chan []byte, 256),
		userID:   userID,
		tenantID: tenantID,
	}

	h.hub.Register(client)

	// Emitir confirmación de conexión inicial
	h.hub.EmitToUser(userID, WebSocketMessage{
		Event: "CONNECTION_ESTABLISHED",
		Stage: "CONNECTED",
		Data: map[string]string{
			"user_id":   userID,
			"tenant_id": tenantID,
			"status":    "ready",
		},
	})

	go client.writePump()
	go client.readPump()
}
