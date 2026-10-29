package httpdelivery

import (
	"encoding/json"
	"errors"
	"net/http"
	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
	"strings"
)

// -----------------------------------------------------------------------------
// Catálogo Extensible de Componentes Tecnológicos y Constructor de Entornos
// -----------------------------------------------------------------------------

func (h *AdminAcademicHandler) ListTemplateComponents(w http.ResponseWriter, r *http.Request) {
	if h.govService == nil {
		SendError(w, http.StatusInternalServerError, "service_unavailable", "Servicio no configurado")
		return
	}
	tenantID := getTenantFromCtx(r)
	layer := r.URL.Query().Get("layer")
	includeInactive := r.URL.Query().Get("include_inactive") == "true"

	items, err := h.govService.ListTemplateComponents(r.Context(), tenantID, layer, includeInactive)
	if err != nil {
		SendError(w, http.StatusInternalServerError, "fetch_failed", err.Error())
		return
	}
	SendJSON(w, http.StatusOK, items, "Componentes del catálogo obtenidos")
}

func (h *AdminAcademicHandler) CreateTemplateComponent(w http.ResponseWriter, r *http.Request) {
	if h.govService == nil {
		SendError(w, http.StatusInternalServerError, "service_unavailable", "Servicio no configurado")
		return
	}
	tenantID := getTenantFromCtx(r)
	adminID := r.Header.Get("X-User-Id")
	if adminID == "" {
		if uid, ok := r.Context().Value(domain.UserIDKey).(string); ok && uid != "" {
			adminID = uid
		}
	}

	var dto domain.CreateTemplateComponentDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		SendError(w, http.StatusBadRequest, "invalid_payload", "JSON inválido")
		return
	}

	created, err := h.govService.CreateTemplateComponent(r.Context(), tenantID, adminID, dto)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidPackageName) {
			SendError(w, http.StatusUnprocessableEntity, "invalid_package_name", err.Error())
			return
		}
		if errors.Is(err, domain.ErrComponentSlugConflict) {
			SendError(w, http.StatusConflict, "component_exists", "Ya existe una tecnología con este nombre o identificador")
			return
		}
		SendError(w, http.StatusUnprocessableEntity, "validation_failed", err.Error())
		return
	}
	SendJSON(w, http.StatusCreated, created, "Tecnología registrada en el catálogo")
}

func (h *AdminAcademicHandler) UpdateTemplateComponent(w http.ResponseWriter, r *http.Request) {
	if h.govService == nil {
		SendError(w, http.StatusInternalServerError, "service_unavailable", "Servicio no configurado")
		return
	}
	tenantID := getTenantFromCtx(r)
	id := r.PathValue("id")
	if id == "" {
		SendError(w, http.StatusBadRequest, "missing_id", "ID de componente requerido")
		return
	}
	adminID := r.Header.Get("X-User-Id")

	var dto domain.UpdateTemplateComponentDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		SendError(w, http.StatusBadRequest, "invalid_payload", "JSON inválido")
		return
	}

	updated, err := h.govService.UpdateTemplateComponent(r.Context(), tenantID, id, adminID, dto)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidPackageName) {
			SendError(w, http.StatusUnprocessableEntity, "invalid_package_name", err.Error())
			return
		}
		if errors.Is(err, domain.ErrComponentNotFound) {
			SendError(w, http.StatusNotFound, "not_found", "Componente tecnológico no encontrado")
			return
		}
		SendError(w, http.StatusUnprocessableEntity, "update_failed", err.Error())
		return
	}
	SendJSON(w, http.StatusOK, updated, "Componente tecnológico actualizado")
}

func (h *AdminAcademicHandler) DeleteTemplateComponent(w http.ResponseWriter, r *http.Request) {
	if h.govService == nil {
		SendError(w, http.StatusInternalServerError, "service_unavailable", "Servicio no configurado")
		return
	}
	tenantID := getTenantFromCtx(r)
	id := r.PathValue("id")
	if id == "" {
		SendError(w, http.StatusBadRequest, "missing_id", "ID de componente requerido")
		return
	}
	if err := h.govService.DeleteTemplateComponent(r.Context(), tenantID, id); err != nil {
		if errors.Is(err, domain.ErrComponentSystemLocked) {
			SendError(w, http.StatusConflict, "system_component_locked", "Las tecnologías base del sistema no se pueden eliminar; puede desactivarlas")
			return
		}
		if errors.Is(err, domain.ErrComponentNotFound) {
			SendError(w, http.StatusNotFound, "not_found", "Componente tecnológico no encontrado")
			return
		}
		SendError(w, http.StatusInternalServerError, "delete_failed", err.Error())
		return
	}
	SendJSON(w, http.StatusOK, map[string]string{"id": id, "status": "deleted"}, "Componente tecnológico eliminado")
}

func (h *AdminAcademicHandler) ComposeTemplatePreview(w http.ResponseWriter, r *http.Request) {
	if h.govService == nil {
		SendError(w, http.StatusInternalServerError, "service_unavailable", "Servicio no configurado")
		return
	}
	tenantID := getTenantFromCtx(r)
	var req struct {
		TargetEnvironment string             `json:"target_environment"`
		StackConfig       domain.StackConfig `json:"stack_config"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "invalid_payload", "JSON inválido")
		return
	}
	plan, err := h.govService.ComposeTemplatePreview(r.Context(), tenantID, req.TargetEnvironment, req.StackConfig)
	if err != nil {
		SendError(w, http.StatusUnprocessableEntity, "compose_failed", err.Error())
		return
	}
	SendJSON(w, http.StatusOK, plan, "Plan de construcción de plantilla compilado")
}

func (h *AdminAcademicHandler) UpdateOfficialTemplate(w http.ResponseWriter, r *http.Request) {
	if h.govService == nil {
		SendError(w, http.StatusInternalServerError, "service_unavailable", "Servicio no configurado")
		return
	}
	tenantID := getTenantFromCtx(r)
	templateID := r.PathValue("id")
	if templateID == "" {
		SendError(w, http.StatusBadRequest, "missing_id", "ID de plantilla requerido")
		return
	}
	adminID := r.Header.Get("X-User-Id")
	if adminID == "" {
		if uid, ok := r.Context().Value(domain.UserIDKey).(string); ok && uid != "" {
			adminID = uid
		}
	}

	var dto domain.UpdateOfficialTemplateDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		SendError(w, http.StatusBadRequest, "invalid_payload", "JSON inválido")
		return
	}
	if strings.TrimSpace(dto.Name) == "" || strings.TrimSpace(dto.DockerImage) == "" {
		SendError(w, http.StatusUnprocessableEntity, "validation_failed", "Nombre e imagen son obligatorios")
		return
	}

	updated, err := h.govService.UpdateOfficialTemplate(r.Context(), tenantID, templateID, adminID, dto)
	if err != nil {
		if errors.Is(err, domain.ErrRamExceedsHostCapacity) {
			SendError(w, http.StatusUnprocessableEntity, "ram_exceeds_host", err.Error())
			return
		}
		if errors.Is(err, services.ErrTemplateVerificationRequired) {
			SendError(w, http.StatusUnprocessableEntity, "verification_required", err.Error())
			return
		}
		SendError(w, http.StatusUnprocessableEntity, "update_failed", err.Error())
		return
	}
	SendJSON(w, http.StatusOK, updated, "Plantilla actualizada sin interrumpir contenedores en ejecución")
}
