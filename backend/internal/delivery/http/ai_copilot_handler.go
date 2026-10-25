package httpdelivery

import (
	"encoding/json"
	"errors"
	"net/http"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

// AICopilotHandler expone los casos de uso generativos del Copiloto
// Institucional: sugerencia de componentes para el administrador y borradores
// de ejercicio con sello determinista para el docente.
type AICopilotHandler struct {
	copilotService *services.AICopilotService
}

func NewAICopilotHandler(copilotService *services.AICopilotService) *AICopilotHandler {
	return &AICopilotHandler{copilotService: copilotService}
}

// SuggestComponent genera una sugerencia de componente de catálogo (rol admin).
func (h *AICopilotHandler) SuggestComponent(w http.ResponseWriter, r *http.Request) {
	if !requireAdminRole(w, r) {
		return
	}
	tenantID := getTenantFromCtx(r)
	userID := r.Header.Get("X-User-Id")
	if userID == "" {
		userID = "admin"
	}

	var dto domain.AISuggestComponentRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		SendError(w, http.StatusBadRequest, "invalid_payload", "payload JSON inválido")
		return
	}

	resp, err := h.copilotService.SuggestComponent(r.Context(), tenantID, userID, dto)
	if err != nil {
		sendAICopilotError(w, err, "Error al generar la sugerencia de componente")
		return
	}

	SendJSON(w, http.StatusOK, *resp, "Sugerencia de componente generada")
}

// GenerateExerciseDraft genera un borrador de ejercicio sellado por el runner.
func (h *AICopilotHandler) GenerateExerciseDraft(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role == "" {
		role = domain.GetUserRole(r.Context())
	}
	if role == "student" {
		SendError(w, http.StatusForbidden, "forbidden", "Acceso denegado: el copiloto de ejercicios no está disponible para estudiantes")
		return
	}
	tenantID := getTenantFromCtx(r)
	userID := r.Header.Get("X-User-Id")
	if userID == "" {
		userID = "teacher"
	}

	var dto domain.AIGenerateExerciseRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		SendError(w, http.StatusBadRequest, "invalid_payload", "payload JSON inválido")
		return
	}

	resp, err := h.copilotService.GenerateExerciseDraft(r.Context(), tenantID, userID, dto)
	if err != nil {
		sendAICopilotError(w, err, "Error al generar el borrador de ejercicio")
		return
	}

	SendJSON(w, http.StatusOK, *resp, "Borrador de ejercicio generado")
}

// sendAICopilotError traduce los errores del módulo de IA a HTTP de forma
// uniforme: 409 IA deshabilitada, 429 cuota agotada, 400 validaciones.
func sendAICopilotError(w http.ResponseWriter, err error, fallbackMessage string) {
	switch {
	case errors.Is(err, domain.ErrAIDisabled):
		SendError(w, http.StatusConflict, "ai_disabled", err.Error())
		return
	case errors.Is(err, domain.ErrAIQuotaExceeded):
		SendError(w, http.StatusTooManyRequests, "ai_quota_exceeded", err.Error())
		return
	}
	var vErr *services.AISettingsValidationError
	if errors.As(err, &vErr) {
		SendError(w, http.StatusBadRequest, vErr.Code, vErr.Message)
		return
	}
	SendError(w, http.StatusInternalServerError, err.Error(), fallbackMessage)
}
