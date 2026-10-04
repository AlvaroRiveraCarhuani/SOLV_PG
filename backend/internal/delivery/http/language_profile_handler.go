package httpdelivery

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

type LanguageProfileHandler struct {
	service *services.LanguageProfileService
}

func NewLanguageProfileHandler(service *services.LanguageProfileService) *LanguageProfileHandler {
	return &LanguageProfileHandler{
		service: service,
	}
}

// ListProfiles maneja GET /api/v1/judge/profiles (acceso público/autenticado)
func (h *LanguageProfileHandler) ListProfiles(w http.ResponseWriter, r *http.Request) {
	profiles, err := h.service.ListProfiles(r.Context())
	if err != nil {
		SendError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Error al listar los perfiles de lenguaje")
		return
	}
	SendJSON(w, http.StatusOK, profiles, "Perfiles de lenguaje obtenidos exitosamente")
}

// GetProfile maneja GET /api/v1/judge/profiles/{language}
func (h *LanguageProfileHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	lang := r.PathValue("language")
	if lang == "" {
		SendError(w, http.StatusBadRequest, "MISSING_LANGUAGE", "El parámetro de lenguaje es requerido")
		return
	}

	profile, err := h.service.GetProfile(r.Context(), lang)
	if err != nil {
		if errors.Is(err, domain.ErrUnknownLanguage) {
			SendError(w, http.StatusBadRequest, "UNKNOWN_LANGUAGE", "Lenguaje no soportado por el juez")
			return
		}
		SendError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Error al obtener el perfil de lenguaje")
		return
	}

	SendJSON(w, http.StatusOK, profile, "Perfil de lenguaje obtenido exitosamente")
}

// UpdateProfile maneja PUT /api/v1/admin/judge/profiles/{language} (restringido a admin, auditado)
func (h *LanguageProfileHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userRole := r.Header.Get("X-User-Role")
	if userRole != "admin" {
		SendError(w, http.StatusForbidden, "FORBIDDEN", "Se requiere rol de administrador para modificar perfiles")
		return
	}

	lang := r.PathValue("language")
	if lang == "" {
		SendError(w, http.StatusBadRequest, "MISSING_LANGUAGE", "El parámetro de lenguaje es requerido")
		return
	}

	var dto domain.UpdateLanguageProfileDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		SendError(w, http.StatusBadRequest, "INVALID_JSON", "Cuerpo de la petición JSON inválido")
		return
	}

	adminID := getUserIDFromCtx(r)
	updated, err := h.service.UpdateProfile(r.Context(), adminID, lang, dto)
	if err != nil {
		if errors.Is(err, domain.ErrUnknownLanguage) {
			SendError(w, http.StatusBadRequest, "UNKNOWN_LANGUAGE", "Lenguaje no soportado por el juez")
			return
		}
		if errors.Is(err, domain.ErrReasonRequired) {
			SendError(w, http.StatusBadRequest, "REASON_REQUIRED", "El motivo del cambio es obligatorio para auditoría")
			return
		}
		if errors.Is(err, domain.ErrLanguageProfileTimeoutRange) {
			SendError(w, http.StatusBadRequest, "INVALID_TIMEOUT", "El tiempo límite debe estar en el rango permitido")
			return
		}
		if errors.Is(err, domain.ErrLanguageProfileMemoryRange) {
			SendError(w, http.StatusBadRequest, "INVALID_MEMORY", "El límite de memoria debe estar en el rango permitido")
			return
		}
		if errors.Is(err, domain.ErrLanguageProfileImageUnpinned) {
			SendError(w, http.StatusBadRequest, "IMAGE_UNPINNED", "La imagen debe estar fijada por digest sha256 y no usar :latest")
			return
		}
		if errors.Is(err, domain.ErrRamExceedsHostCapacity) || strings.Contains(err.Error(), "supera max_allowed_ram_mb") {
			SendError(w, http.StatusBadRequest, "RAM_EXCEEDS_HOST", "La memoria solicitada supera la capacidad estructural del host")
			return
		}

		SendError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	SendJSON(w, http.StatusOK, updated, "Perfil de lenguaje actualizado exitosamente")
}
