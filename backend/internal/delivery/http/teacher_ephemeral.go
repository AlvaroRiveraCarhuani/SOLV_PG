package httpdelivery

import (
	"context"
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

func (h *TeacherHandler) authorizeTerminal(ctx context.Context, tenantID, userID, role, containerID string) error {
	if role != "teacher" && role != "admin" {
		return domain.ErrNotFound
	}
	authorizer := h.terminalAuthorizer
	if authorizer == nil && h.service != nil {
		authorizer = h.service
	}
	if authorizer == nil {
		return domain.ErrNotFound
	}
	return authorizer.AuthorizeTerminal(ctx, tenantID, userID, containerID)
}

func (h *TeacherHandler) RunEphemeral(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "teacher", "admin") {
		return
	}

	tenantID := getTenantFromCtx(r)
	if tenantID == "" {
		SendError(w, http.StatusUnauthorized, "invalid session", "La sesión no identifica una institución")
		return
	}

	submissionID := r.PathValue("id")
	if submissionID == "" {
		SendError(w, http.StatusBadRequest, "Missing submission ID", "El identificador de la entrega es requerido")
		return
	}

	var dto domain.EphemeralRunRequestDTO
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&dto)
	}

	teacherID := getUserIDFromCtx(r)
	if teacherID == "" {
		SendError(w, http.StatusUnauthorized, "invalid session", "La sesión no identifica al docente")
		return
	}

	result, err := h.service.RunEphemeral(r.Context(), tenantID, teacherID, submissionID, dto.Code, dto.Language)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidResourceID) {
			SendError(w, http.StatusBadRequest, "invalid_submission_id", "El identificador de la entrega no tiene un formato válido")
			return
		}
		if errors.Is(err, domain.ErrNotFound) {
			SendError(w, http.StatusNotFound, "Submission not found", "La entrega solicitada no existe")
			return
		}
		if errors.Is(err, domain.ErrEphemeralNotSupportedForIDE) {
			SendError(w, http.StatusConflict, "ephemeral_not_supported_for_ide", "La ejecución efímera solo aplica al Evaluador Automático; los proyectos se corrigen con rúbrica")
			return
		}
		SendError(w, http.StatusInternalServerError, err.Error(), "Error en ejecución efímera")
		return
	}

	SendJSON(w, http.StatusOK, result, "Ejecución efímera completada exitosamente")
}

func (h *TeacherHandler) PostTutorExec(w http.ResponseWriter, r *http.Request) {
	role, _ := r.Context().Value(domain.UserRoleKey).(string)
	if role != "teacher" && role != "admin" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo docentes y administradores pueden acceder a este recurso")
		return
	}
	if h.authService == nil {
		SendError(w, http.StatusServiceUnavailable, "session_revocation_unavailable", "No se pudo comprobar la revocación de la sesión")
		return
	}
	token, _ := r.Context().Value(domain.UserTokenKey).(string)
	if strings.TrimSpace(token) == "" {
		SendError(w, http.StatusUnauthorized, "invalid session", "La sesión no tiene un token verificable")
		return
	}
	revoked, revErr := h.authService.IsSessionRevokedContext(r.Context(), token)
	if revErr != nil {
		SendError(w, http.StatusServiceUnavailable, "session_revocation_unavailable", "No se pudo comprobar la revocación de la sesión")
		return
	}
	if revoked {
		SendError(w, http.StatusUnauthorized, "invalid session", "La sesión fue revocada")
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

	tenantID, _ := r.Context().Value(domain.TenantIDKey).(string)
	teacherID, _ := r.Context().Value(domain.UserIDKey).(string)
	if tenantID == "" || teacherID == "" {
		SendError(w, http.StatusUnauthorized, "invalid session", "Sesión sin tenant o usuario")
		return
	}
	if err := h.authorizeTerminal(r.Context(), tenantID, teacherID, role, containerID); err != nil {
		SendError(w, http.StatusForbidden, "forbidden", "El docente no tiene acceso a esta sesión")
		return
	}

	resp, err := h.service.ExecuteTutorCommand(r.Context(), tenantID, teacherID, containerID, req.Command)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al ejecutar comando tutor en el contenedor")
		return
	}

	SendJSON(w, http.StatusOK, resp, "Comando ejecutado exitosamente en el contenedor")
}

func (h *TeacherHandler) HandleTerminalWebSocket(w http.ResponseWriter, r *http.Request) {
	if !websocketOriginAllowed(r) {
		SendError(w, http.StatusForbidden, "forbidden origin", "Origen no permitido para WebSocket")
		return
	}

	principal, err := websocketSession(r, h.authService)
	if err != nil {
		SendError(w, http.StatusUnauthorized, "invalid session token", "Token de sesión inválido para WebSocket")
		return
	}

	containerID := r.PathValue("id")
	if containerID == "" {
		containerID = r.URL.Query().Get("container_id")
	}
	if err := h.authorizeTerminal(r.Context(), principal.tenantID, principal.userID, principal.role, containerID); err != nil {
		SendError(w, http.StatusForbidden, "forbidden", "El docente no tiene acceso a esta sesión")
		return
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[TerminalMirror] Upgrade failed: %v", err)
		return
	}
	defer func() { _ = conn.Close() }()
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
	if h.authService != nil {
		unregister := h.authService.RegisterTokenRevocationListener(principal.sessionID, func() {
			_ = conn.WriteControl(
				websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "session revoked"),
				time.Now().Add(writeWait),
			)
			_ = conn.Close()
		})
		defer unregister()
		revoked, revErr := h.authService.IsSessionIDRevokedContext(r.Context(), principal.sessionID)
		if revErr != nil || revoked {
			_ = conn.WriteControl(
				websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "session revoked"),
				time.Now().Add(writeWait),
			)
			return
		}
		h.authService.WatchSessionRevocation(r.Context(), principal.sessionID, func() {
			_ = conn.WriteControl(
				websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "session revoked"),
				time.Now().Add(writeWait),
			)
			_ = conn.Close()
		})
	}

	// 1. Enviar frame inicial de bienvenida y buffer reciente
	initBuffer, _ := h.service.GetInitialTerminalBuffer(r.Context(), containerID, 100)

	welcomePayload := map[string]interface{}{
		"type":         "init",
		"container_id": containerID,
		"user_id":      principal.userID,
		"tenant_id":    principal.tenantID,
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
					if h.authService != nil {
						revoked, revErr := h.authService.IsSessionIDRevokedContext(r.Context(), principal.sessionID)
						if revErr != nil || revoked {
							_ = conn.WriteJSON(map[string]interface{}{"type": "error", "error": "session revoked"})
							_ = conn.Close()
							return
						}
					}
					if err := h.authorizeTerminal(r.Context(), principal.tenantID, principal.userID, principal.role, containerID); err != nil {
						_ = conn.WriteJSON(map[string]interface{}{"type": "error", "error": "forbidden"})
						_ = conn.Close()
						return
					}
					resp, execErr := h.service.ExecuteTutorCommand(r.Context(), principal.tenantID, principal.userID, containerID, incoming.Command)
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
