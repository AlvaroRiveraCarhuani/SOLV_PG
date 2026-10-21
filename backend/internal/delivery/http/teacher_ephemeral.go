package httpdelivery

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"solv-backend/internal/core/domain"
)

func (h *TeacherHandler) RunEphemeral(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role == "student" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo docentes pueden ejecutar el runner efímero")
		return
	}

	tenantID := getTenantFromCtx(r)

	submissionID := r.PathValue("id")
	if submissionID == "" {
		SendError(w, http.StatusBadRequest, "Missing submission ID", "El identificador de la entrega es requerido")
		return
	}

	var dto domain.EphemeralRunRequestDTO
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&dto)
	}

	teacherID := r.Header.Get("X-User-Id")

	result, err := h.service.RunEphemeral(r.Context(), tenantID, teacherID, submissionID, dto.Code, dto.Language)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			SendError(w, http.StatusNotFound, "Submission not found", "La entrega solicitada no existe")
			return
		}
		SendError(w, http.StatusInternalServerError, err.Error(), "Error en ejecución efímera")
		return
	}

	SendJSON(w, http.StatusOK, result, "Ejecución efímera completada exitosamente")
}

func (h *TeacherHandler) PostTutorExec(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role == "student" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo docentes y administradores pueden acceder a este recurso")
		return
	}

	containerID := r.PathValue("id")
	if containerID == "" {
		SendError(w, http.StatusBadRequest, "Missing container ID", "El identificador del contenedor es requerido")
		return
	}

	var req domain.TutorCommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "Invalid JSON body", "El cuerpo de la solicitud no es válido")
		return
	}

	tenantID := getTenantFromCtx(r)
	teacherID := r.Header.Get("X-User-Id")

	resp, err := h.service.ExecuteTutorCommand(r.Context(), tenantID, teacherID, containerID, req.Command)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al ejecutar comando tutor en el contenedor")
		return
	}

	SendJSON(w, http.StatusOK, resp, "Comando ejecutado exitosamente en el contenedor")
}

func (h *TeacherHandler) HandleTerminalWebSocket(w http.ResponseWriter, r *http.Request) {
	containerID := r.PathValue("id")
	if containerID == "" {
		containerID = r.URL.Query().Get("container_id")
	}

	var tokenStr string
	if qToken := r.URL.Query().Get("token"); qToken != "" {
		tokenStr = qToken
	} else if authHeader := r.Header.Get("Authorization"); strings.HasPrefix(authHeader, "Bearer ") {
		tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
	} else if cookie, err := r.Cookie("solv_session"); err == nil && cookie.Value != "" {
		tokenStr = cookie.Value
	}

	userID := "anonymous"
	tenantID := domain.DefaultTenantID

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
		if hUID := r.Header.Get("X-User-Id"); hUID != "" {
			userID = hUID
		}
		if hTID := r.Header.Get("X-Tenant-Id"); hTID != "" {
			tenantID = hTID
		}
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[TerminalMirror] Upgrade failed: %v", err)
		return
	}
	defer func() { _ = conn.Close() }()

	// 1. Enviar frame inicial de bienvenida y buffer reciente
	initBuffer, _ := h.service.GetInitialTerminalBuffer(r.Context(), containerID, 100)

	welcomePayload := map[string]interface{}{
		"type":         "init",
		"container_id": containerID,
		"user_id":      userID,
		"tenant_id":    tenantID,
		"buffer":       initBuffer,
		"timestamp":    time.Now().UTC().Format(time.RFC3339),
		"mode":         "SHADOW_MIRROR",
	}
	_ = conn.WriteJSON(welcomePayload)

	// 2. Loop de lectura para mensajes del docente (ping, resize, exec)
	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[TerminalMirror] Client closed: %v", err)
			}
			break
		}

		if messageType == websocket.TextMessage {
			var incoming struct {
				Type    string `json:"type"`
				Command string `json:"command"`
			}
			if err := json.Unmarshal(message, &incoming); err == nil {
				if incoming.Type == "ping" {
					_ = conn.WriteJSON(map[string]string{"type": "pong"})
				} else if incoming.Type == "exec" && incoming.Command != "" {
					resp, execErr := h.service.ExecuteTutorCommand(r.Context(), tenantID, userID, containerID, incoming.Command)
					if execErr != nil {
						_ = conn.WriteJSON(map[string]interface{}{
							"type":  "error",
							"error": execErr.Error(),
						})
					} else {
						_ = conn.WriteJSON(map[string]interface{}{
							"type":   "stdout",
							"output": fmt.Sprintf("\n%s\n", resp.Output),
						})
					}
				}
			}
		}
	}
}
