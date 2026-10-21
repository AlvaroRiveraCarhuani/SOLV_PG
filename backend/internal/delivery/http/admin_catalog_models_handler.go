package httpdelivery

import (
	_ "embed"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"solv-backend/internal/core/domain"
	"strings"
)

// -----------------------------------------------------------------------------
// Categorías y Modelos de Plantillas (ADR-030)
// -----------------------------------------------------------------------------

func (h *AdminAcademicHandler) ListTemplateCategories(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantFromCtx(r)
	if h.govService == nil {
		SendError(w, http.StatusInternalServerError, "service_unavailable", "Servicio no configurado")
		return
	}
	categories, err := h.govService.ListTemplateCategories(r.Context(), tenantID)
	if err != nil {
		SendError(w, http.StatusInternalServerError, "fetch_failed", err.Error())
		return
	}
	SendJSON(w, http.StatusOK, categories, "Categorías obtenidas")
}

func (h *AdminAcademicHandler) CreateTemplateCategory(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantFromCtx(r)
	var dto domain.CreateCategoryDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		SendError(w, http.StatusBadRequest, "invalid_payload", "JSON inválido")
		return
	}
	if strings.TrimSpace(dto.Name) == "" {
		SendError(w, http.StatusUnprocessableEntity, "validation_failed", "El nombre es obligatorio")
		return
	}
	cat, err := h.govService.CreateTemplateCategory(r.Context(), tenantID, dto)
	if err != nil {
		if errors.Is(err, domain.ErrTemplateNameConflict) {
			SendError(w, http.StatusConflict, "category_exists", "Ya existe una categoría con este nombre")
			return
		}
		SendError(w, http.StatusInternalServerError, "create_failed", err.Error())
		return
	}
	SendJSON(w, http.StatusCreated, cat, "Categoría creada")
}

func (h *AdminAcademicHandler) UpdateTemplateCategory(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantFromCtx(r)
	categoryID := r.PathValue("id")
	if categoryID == "" {
		SendError(w, http.StatusBadRequest, "missing_id", "ID de categoría requerido")
		return
	}
	var dto domain.UpdateCategoryDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		SendError(w, http.StatusBadRequest, "invalid_payload", "JSON inválido")
		return
	}
	if strings.TrimSpace(dto.Name) == "" {
		SendError(w, http.StatusUnprocessableEntity, "validation_failed", "El nombre es obligatorio")
		return
	}
	cat, err := h.govService.UpdateTemplateCategory(r.Context(), tenantID, categoryID, dto)
	if err != nil {
		if errors.Is(err, domain.ErrCategoryNotFound) {
			SendError(w, http.StatusNotFound, "category_not_found", "Categoría no encontrada")
			return
		}
		if errors.Is(err, domain.ErrTemplateNameConflict) {
			SendError(w, http.StatusConflict, "category_exists", "Ya existe una categoría con este nombre")
			return
		}
		SendError(w, http.StatusInternalServerError, "update_failed", err.Error())
		return
	}
	SendJSON(w, http.StatusOK, cat, "Categoría actualizada")
}

func (h *AdminAcademicHandler) DeleteTemplateCategory(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantFromCtx(r)
	categoryID := r.PathValue("id")
	if categoryID == "" {
		SendError(w, http.StatusBadRequest, "missing_id", "ID de categoría requerido")
		return
	}
	adminID := r.Header.Get("X-User-Id")
	if adminID == "" {
		adminID = domain.DefaultTenantID
	}
	err := h.govService.DeleteTemplateCategory(r.Context(), tenantID, adminID, categoryID)
	if err != nil {
		if errors.Is(err, domain.ErrCategoryInUse) {
			SendError(w, http.StatusConflict, "category_in_use", "No se puede eliminar la categoría porque tiene plantillas o modelos asociados")
			return
		}
		if errors.Is(err, domain.ErrCategoryNotFound) {
			SendError(w, http.StatusNotFound, "category_not_found", "Categoría no encontrada")
			return
		}
		SendError(w, http.StatusInternalServerError, "delete_failed", err.Error())
		return
	}
	SendJSON(w, http.StatusOK, map[string]string{"deleted": categoryID}, "Categoría eliminada")
}

func (h *AdminAcademicHandler) ReorderTemplateCategories(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantFromCtx(r)
	if h.govService == nil {
		SendError(w, http.StatusInternalServerError, "service_unavailable", "Servicio no configurado")
		return
	}
	var items []domain.ReorderCategoryItemDTO
	if err := json.NewDecoder(r.Body).Decode(&items); err != nil {
		SendError(w, http.StatusBadRequest, "invalid_payload", "JSON inválido")
		return
	}
	if err := h.govService.ReorderTemplateCategories(r.Context(), tenantID, items); err != nil {
		SendError(w, http.StatusInternalServerError, "reorder_failed", err.Error())
		return
	}
	SendJSON(w, http.StatusOK, map[string]string{"message": "Categorías reordenadas"}, "Categorías reordenadas exitosamente")
}

func (h *AdminAcademicHandler) ListTemplateModels(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantFromCtx(r)
	targetEnv := r.URL.Query().Get("target_environment")
	if targetEnv == "" {
		targetEnv = r.URL.Query().Get("purpose")
	}
	categoryID := r.URL.Query().Get("category")
	if categoryID == "" {
		categoryID = r.URL.Query().Get("category_id")
	}
	includeInactive := r.URL.Query().Get("include_inactive") == "true"

	if h.govService == nil {
		SendError(w, http.StatusInternalServerError, "service_unavailable", "Servicio no configurado")
		return
	}
	models, err := h.govService.ListTemplateModels(r.Context(), tenantID, targetEnv, categoryID, includeInactive)
	if err != nil {
		SendError(w, http.StatusInternalServerError, "fetch_failed", err.Error())
		return
	}
	SendJSON(w, http.StatusOK, models, "Modelos de plantilla obtenidos")
}

func (h *AdminAcademicHandler) UpdateTemplateModel(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantFromCtx(r)
	modelID := r.PathValue("id")
	if modelID == "" {
		SendError(w, http.StatusBadRequest, "missing_id", "ID de modelo requerido")
		return
	}
	var dto domain.UpdateTemplateModelDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		SendError(w, http.StatusBadRequest, "invalid_payload", "JSON inválido")
		return
	}
	if strings.TrimSpace(dto.Title) == "" {
		SendError(w, http.StatusUnprocessableEntity, "validation_failed", "El título es obligatorio")
		return
	}
	if strings.TrimSpace(dto.CategoryID) == "" {
		SendError(w, http.StatusUnprocessableEntity, "validation_failed", "La categoría es obligatoria")
		return
	}
	model, err := h.govService.UpdateTemplateModel(r.Context(), tenantID, modelID, dto)
	if err != nil {
		if errors.Is(err, domain.ErrCategoryNotFound) {
			SendError(w, http.StatusNotFound, "category_not_found", "Categoría no encontrada")
			return
		}
		if errors.Is(err, domain.ErrTemplateNameConflict) {
			SendError(w, http.StatusConflict, "model_exists", "Ya existe un modelo con este título")
			return
		}
		SendError(w, http.StatusInternalServerError, "update_failed", err.Error())
		return
	}
	SendJSON(w, http.StatusOK, model, "Modelo actualizado")
}

func (h *AdminAcademicHandler) DeactivateTemplateModel(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantFromCtx(r)
	modelID := r.PathValue("id")
	if modelID == "" {
		SendError(w, http.StatusBadRequest, "missing_id", "ID de modelo requerido")
		return
	}
	adminID := r.Header.Get("X-User-Id")
	if adminID == "" {
		if uid, ok := r.Context().Value(domain.UserIDKey).(string); ok && uid != "" {
			adminID = uid
		} else {
			adminID = domain.DefaultTenantID
		}
	}
	if err := h.govService.DeactivateTemplateModel(r.Context(), tenantID, adminID, modelID); err != nil {
		SendError(w, http.StatusInternalServerError, "deactivate_failed", err.Error())
		return
	}
	SendJSON(w, http.StatusOK, map[string]string{"id": modelID, "status": "deactivated"}, "Modelo desactivado")
}

func (h *AdminAcademicHandler) ReactivateTemplateModel(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantFromCtx(r)
	modelID := r.PathValue("id")
	if modelID == "" {
		SendError(w, http.StatusBadRequest, "missing_id", "ID de modelo requerido")
		return
	}
	adminID := r.Header.Get("X-User-Id")
	if adminID == "" {
		if uid, ok := r.Context().Value(domain.UserIDKey).(string); ok && uid != "" {
			adminID = uid
		} else {
			adminID = domain.DefaultTenantID
		}
	}
	if err := h.govService.ReactivateTemplateModel(r.Context(), tenantID, adminID, modelID); err != nil {
		SendError(w, http.StatusInternalServerError, "reactivate_failed", err.Error())
		return
	}
	SendJSON(w, http.StatusOK, map[string]string{"id": modelID, "status": "active"}, "Modelo reactivado")
}

func (h *AdminAcademicHandler) PromoteTemplateToModel(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantFromCtx(r)
	templateID := r.PathValue("id")
	if templateID == "" {
		SendError(w, http.StatusBadRequest, "missing_id", "ID de plantilla requerido")
		return
	}
	adminID := r.Header.Get("X-User-Id")
	if adminID == "" {
		adminID = domain.DefaultTenantID
	}

	var dto domain.PromoteTemplateToModelDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		SendError(w, http.StatusBadRequest, "invalid_payload", "JSON inválido")
		return
	}
	if strings.TrimSpace(dto.CategoryID) == "" {
		SendError(w, http.StatusUnprocessableEntity, "validation_failed", "La categoría es obligatoria")
		return
	}

	model, err := h.govService.PromoteTemplateToModel(r.Context(), tenantID, templateID, adminID, dto)
	if err != nil {
		if errors.Is(err, domain.ErrTemplateNotApproved) {
			SendError(w, http.StatusUnprocessableEntity, "template_not_approved", "Solo plantillas con estado APROBADA pueden ser promovidas a modelo oficial")
			return
		}
		if errors.Is(err, domain.ErrCategoryNotFound) {
			SendError(w, http.StatusNotFound, "category_not_found", "Categoría no encontrada")
			return
		}
		if errors.Is(err, domain.ErrTemplateNameConflict) {
			SendError(w, http.StatusConflict, "model_exists", "Ya existe un modelo oficial con este título")
			return
		}
		SendError(w, http.StatusInternalServerError, "promote_failed", err.Error())
		return
	}
	SendJSON(w, http.StatusCreated, model, "Plantilla promovida a modelo oficial")
}

func (h *AdminAcademicHandler) SaveDraft(w http.ResponseWriter, r *http.Request) {
	if h.govService == nil {
		SendError(w, http.StatusInternalServerError, "service_unavailable", "Servicio de gobernanza no configurado")
		return
	}

	tenantID := getTenantFromCtx(r)
	userID := r.Header.Get("X-User-Id")
	if userID == "" {
		if uid, ok := r.Context().Value(domain.UserIDKey).(string); ok && uid != "" {
			userID = uid
		}
	}
	if userID == "" {
		SendError(w, http.StatusUnauthorized, "unauthorized", "Usuario no identificado")
		return
	}

	var dto domain.CreateDraftDTO
	_ = json.NewDecoder(r.Body).Decode(&dto)

	if len(dto.FormData) == 0 || string(dto.FormData) == "null" {
		dto.FormData = json.RawMessage("{}")
	}

	draft, err := h.govService.SaveTemplateDraft(r.Context(), tenantID, userID, dto.FormData, dto.TemplateID)
	if err != nil {
		SendError(w, http.StatusInternalServerError, "save_draft_failed", err.Error())
		return
	}

	resp := domain.DraftResponse{
		ID:        draft.ID,
		UserID:    draft.UserID,
		FormData:  draft.FormData,
		UpdatedAt: draft.UpdatedAt,
	}

	SendJSON(w, http.StatusOK, resp, "Borrador guardado exitosamente")
}

func (h *AdminAcademicHandler) GetDraftByUser(w http.ResponseWriter, r *http.Request) {
	if h.govService == nil {
		SendError(w, http.StatusInternalServerError, "service_unavailable", "Servicio de gobernanza no configurado")
		return
	}

	tenantID := getTenantFromCtx(r)
	userID := r.Header.Get("X-User-Id")
	if userID == "" {
		if uid, ok := r.Context().Value(domain.UserIDKey).(string); ok && uid != "" {
			userID = uid
		}
	}
	if userID == "" {
		SendError(w, http.StatusUnauthorized, "unauthorized", "Usuario no identificado")
		return
	}

	draft, err := h.govService.GetTemplateDraft(r.Context(), tenantID, userID)
	if err != nil {
		SendError(w, http.StatusInternalServerError, "get_draft_failed", err.Error())
		return
	}
	if draft == nil {
		SendError(w, http.StatusNotFound, "draft_not_found", "No existe borrador para este usuario")
		return
	}

	resp := domain.DraftResponse{
		ID:        draft.ID,
		UserID:    draft.UserID,
		FormData:  draft.FormData,
		UpdatedAt: draft.UpdatedAt,
	}

	SendJSON(w, http.StatusOK, resp, "Borrador obtenido exitosamente")
}

func (h *AdminAcademicHandler) DeleteDraft(w http.ResponseWriter, r *http.Request) {
	if h.govService == nil {
		SendError(w, http.StatusInternalServerError, "service_unavailable", "Servicio de gobernanza no configurado")
		return
	}

	tenantID := getTenantFromCtx(r)
	userID := r.Header.Get("X-User-Id")
	if userID == "" {
		if uid, ok := r.Context().Value(domain.UserIDKey).(string); ok && uid != "" {
			userID = uid
		}
	}
	if userID == "" {
		SendError(w, http.StatusUnauthorized, "unauthorized", "Usuario no identificado")
		return
	}

	if err := h.govService.DeleteTemplateDraft(r.Context(), tenantID, userID); err != nil {
		SendError(w, http.StatusInternalServerError, "delete_draft_failed", err.Error())
		return
	}

	SendJSON(w, http.StatusOK, map[string]string{"message": "Borrador eliminado exitosamente"}, "Borrador eliminado exitosamente")
}

//go:embed docs/MANUAL_ADMIN.md
var embeddedAdminManual []byte

func (h *AdminAcademicHandler) GetAdminManual(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")

	if data, err := os.ReadFile("internal/delivery/http/docs/MANUAL_ADMIN.md"); err == nil && len(data) > 0 {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
		return
	}
	if data, err := os.ReadFile("backend/internal/delivery/http/docs/MANUAL_ADMIN.md"); err == nil && len(data) > 0 {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(embeddedAdminManual)
}
