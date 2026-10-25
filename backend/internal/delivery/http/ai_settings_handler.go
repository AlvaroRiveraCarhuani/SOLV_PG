package httpdelivery

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

// AISettingsHandler expone la configuración de Inteligencia Artificial Híbrida
// (Nube + Servidor Local LAN) por institución: GET/PUT de settings, prueba de
// conexión real y estado liviano para el resto de los roles.
type AISettingsHandler struct {
	aiService *services.AISettingsService
}

func NewAISettingsHandler(aiService *services.AISettingsService) *AISettingsHandler {
	return &AISettingsHandler{aiService: aiService}
}

// GetSettings devuelve la configuración pública (sin la API Key cifrada).
func (h *AISettingsHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	if !requireAdminRole(w, r) {
		return
	}
	tenantID := getTenantFromCtx(r)

	stored, err := h.aiService.Get(r.Context(), tenantID)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al obtener la configuración de IA")
		return
	}

	resp := domain.NewAISettingsResponseDTO(*stored)
	SendJSON(w, http.StatusOK, resp, "Configuración de IA obtenida exitosamente")
}

// UpdateSettings valida, cifra y persiste la configuración del instituto.
func (h *AISettingsHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	if !requireAdminRole(w, r) {
		return
	}
	tenantID := getTenantFromCtx(r)

	var dto domain.UpdateAISettingsDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		SendError(w, http.StatusBadRequest, "invalid_payload", "payload JSON inválido")
		return
	}

	resp, err := h.aiService.Update(r.Context(), tenantID, dto)
	if err != nil {
		var vErr *services.AISettingsValidationError
		if errors.As(err, &vErr) {
			SendError(w, http.StatusBadRequest, vErr.Code, vErr.Message)
			return
		}
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al actualizar la configuración de IA")
		return
	}

	SendJSON(w, http.StatusOK, *resp, "Configuración de IA actualizada")
}

// TestConnection realiza una verificación real contra el motor configurado.
func (h *AISettingsHandler) TestConnection(w http.ResponseWriter, r *http.Request) {
	if !requireAdminRole(w, r) {
		return
	}
	tenantID := getTenantFromCtx(r)

	var dto domain.TestAIConnectionRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		SendError(w, http.StatusBadRequest, "invalid_payload", "payload JSON inválido")
		return
	}

	resp, err := h.aiService.TestConnection(r.Context(), tenantID, dto)
	if err != nil {
		var vErr *services.AISettingsValidationError
		if errors.As(err, &vErr) {
			SendError(w, http.StatusBadRequest, vErr.Code, vErr.Message)
			return
		}
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al probar la conexión con el motor de IA")
		return
	}

	SendJSON(w, http.StatusOK, *resp, "Prueba de conexión completada")
}

// ListModels descubre dinámicamente los modelos ofrecidos por el proveedor.
func (h *AISettingsHandler) ListModels(w http.ResponseWriter, r *http.Request) {
	if !requireAdminRole(w, r) {
		return
	}
	tenantID := getTenantFromCtx(r)

	var dto domain.ListAIModelsRequestDTO
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&dto); err != nil && !errors.Is(err, io.EOF) {
			SendError(w, http.StatusBadRequest, "invalid_payload", "payload JSON inválido")
			return
		}
	}

	resp, err := h.aiService.ListModels(r.Context(), tenantID, dto)
	if err != nil {
		var vErr *services.AISettingsValidationError
		if errors.As(err, &vErr) {
			SendError(w, http.StatusBadRequest, vErr.Code, vErr.Message)
			return
		}
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al consultar los modelos de IA")
		return
	}

	SendJSON(w, http.StatusOK, *resp, "Modelos de IA obtenidos")
}

// Status devuelve el estado liviano (enabled/provider/model + perfil de motor
// resuelto) para roles autenticados: admin, teacher y el resto del tenant. El
// userID resuelve la cascada BYOK personal del usuario.
func (h *AISettingsHandler) Status(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantFromCtx(r)
	userID := getUserIDFromCtx(r)

	resp, err := h.aiService.GetEffectiveStatus(r.Context(), tenantID, userID)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al obtener el estado de IA")
		return
	}

	SendJSON(w, http.StatusOK, *resp, "Estado de IA obtenido")
}

// requireStaffRole admite solo teacher y admin. student recibe 403.
func requireStaffRole(w http.ResponseWriter, r *http.Request) bool {
	role := r.Header.Get("X-User-Role")
	if role == "" {
		role = domain.GetUserRole(r.Context())
	}
	if role != "teacher" && role != "admin" {
		SendError(w, http.StatusForbidden, "forbidden", "Acceso denegado: la configuración personal de IA está disponible para docentes y administradores")
		return false
	}
	return true
}

// GetMySettings devuelve la API Key personal (BYOK) del usuario autenticado.
func (h *AISettingsHandler) GetMySettings(w http.ResponseWriter, r *http.Request) {
	if !requireStaffRole(w, r) {
		return
	}
	resp, err := h.aiService.GetUserOverride(r.Context(), getTenantFromCtx(r), getUserIDFromCtx(r))
	if err != nil {
		var vErr *services.AISettingsValidationError
		if errors.As(err, &vErr) {
			SendError(w, http.StatusBadRequest, vErr.Code, vErr.Message)
			return
		}
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al obtener tu configuración personal de IA")
		return
	}
	SendJSON(w, http.StatusOK, *resp, "Configuración personal de IA obtenida")
}

// UpdateMySettings valida, cifra y persiste la API Key personal (BYOK).
func (h *AISettingsHandler) UpdateMySettings(w http.ResponseWriter, r *http.Request) {
	if !requireStaffRole(w, r) {
		return
	}
	var dto domain.UpdateAIUserOverrideDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		SendError(w, http.StatusBadRequest, "invalid_payload", "payload JSON inválido")
		return
	}

	resp, err := h.aiService.UpdateUserOverride(r.Context(), getTenantFromCtx(r), getUserIDFromCtx(r), dto)
	if err != nil {
		var vErr *services.AISettingsValidationError
		if errors.As(err, &vErr) {
			SendError(w, http.StatusBadRequest, vErr.Code, vErr.Message)
			return
		}
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al guardar tu configuración personal de IA")
		return
	}
	SendJSON(w, http.StatusOK, *resp, "Configuración personal de IA actualizada")
}

// TestMyConnection prueba la conexión con la API Key personal del usuario.
func (h *AISettingsHandler) TestMyConnection(w http.ResponseWriter, r *http.Request) {
	if !requireStaffRole(w, r) {
		return
	}
	var dto domain.TestAIConnectionRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		SendError(w, http.StatusBadRequest, "invalid_payload", "payload JSON inválido")
		return
	}

	resp, err := h.aiService.TestUserConnection(r.Context(), getTenantFromCtx(r), getUserIDFromCtx(r), dto)
	if err != nil {
		var vErr *services.AISettingsValidationError
		if errors.As(err, &vErr) {
			SendError(w, http.StatusBadRequest, vErr.Code, vErr.Message)
			return
		}
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al probar la conexión con tu motor de IA")
		return
	}
	SendJSON(w, http.StatusOK, *resp, "Prueba de conexión completada")
}

// ListMyModels descubre los modelos disponibles con la API Key personal.
func (h *AISettingsHandler) ListMyModels(w http.ResponseWriter, r *http.Request) {
	if !requireStaffRole(w, r) {
		return
	}
	var dto domain.ListAIModelsRequestDTO
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&dto); err != nil && !errors.Is(err, io.EOF) {
			SendError(w, http.StatusBadRequest, "invalid_payload", "payload JSON inválido")
			return
		}
	}

	resp, err := h.aiService.ListUserModels(r.Context(), getTenantFromCtx(r), getUserIDFromCtx(r), dto)
	if err != nil {
		var vErr *services.AISettingsValidationError
		if errors.As(err, &vErr) {
			SendError(w, http.StatusBadRequest, vErr.Code, vErr.Message)
			return
		}
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al consultar los modelos de IA")
		return
	}
	SendJSON(w, http.StatusOK, *resp, "Modelos de IA obtenidos")
}
