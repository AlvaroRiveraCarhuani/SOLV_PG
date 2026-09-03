package httpdelivery

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

type BackupHandler struct {
	service *services.BackupService
}

func NewBackupHandler(service *services.BackupService) *BackupHandler {
	return &BackupHandler{service: service}
}

func requireAdminRole(w http.ResponseWriter, r *http.Request) bool {
	role := r.Header.Get("X-User-Role")
	if role != "" && role != "admin" {
		SendError(w, http.StatusForbidden, "forbidden", "Acceso denegado: se requiere rol de administrador")
		return false
	}
	return true
}

func (h *BackupHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	if !requireAdminRole(w, r) {
		return
	}
	tenantID := getTenantFromCtx(r)

	cfg, err := h.service.GetConfig(r.Context(), tenantID)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al obtener configuración de respaldos")
		return
	}

	SendJSON(w, http.StatusOK, cfg, "Configuración de respaldos obtenida exitosamente")
}

func (h *BackupHandler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	if !requireAdminRole(w, r) {
		return
	}
	tenantID := getTenantFromCtx(r)

	var dto domain.UpdateBackupConfigDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		SendError(w, http.StatusBadRequest, "invalid_json", "Cuerpo JSON inválido")
		return
	}

	cfg, err := h.service.UpdateConfig(r.Context(), tenantID, dto)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al actualizar configuración de respaldos")
		return
	}

	SendJSON(w, http.StatusOK, cfg, "Configuración de respaldos actualizada exitosamente")
}

func (h *BackupHandler) List(w http.ResponseWriter, r *http.Request) {
	if !requireAdminRole(w, r) {
		return
	}
	tenantID := getTenantFromCtx(r)

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}

	items, total, err := h.service.ListExecutions(r.Context(), tenantID, page, limit)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al obtener historial de respaldos")
		return
	}

	w.Header().Set("X-Total-Count", strconv.FormatInt(total, 10))
	SendJSON(w, http.StatusOK, items, "Historial de respaldos obtenido exitosamente")
}

func (h *BackupHandler) Trigger(w http.ResponseWriter, r *http.Request) {
	if !requireAdminRole(w, r) {
		return
	}
	tenantID := getTenantFromCtx(r)
	adminUserID := getUserIDFromCtx(r)

	exec, err := h.service.TriggerBackup(r.Context(), tenantID, adminUserID)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al ejecutar copia de seguridad")
		return
	}

	SendJSON(w, http.StatusCreated, exec, "Copia de seguridad ejecutada exitosamente")
}

func (h *BackupHandler) Verify(w http.ResponseWriter, r *http.Request) {
	if !requireAdminRole(w, r) {
		return
	}
	tenantID := getTenantFromCtx(r)
	id := r.PathValue("id")
	if id == "" {
		SendError(w, http.StatusBadRequest, "missing_id", "id de respaldo requerido")
		return
	}

	res, err := h.service.VerifyBackup(r.Context(), tenantID, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			SendError(w, http.StatusNotFound, "not_found", "Respaldo no encontrado")
			return
		}
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al verificar integridad del respaldo")
		return
	}

	SendJSON(w, http.StatusOK, res, "Verificación de integridad completada")
}

func (h *BackupHandler) Download(w http.ResponseWriter, r *http.Request) {
	if !requireAdminRole(w, r) {
		return
	}
	tenantID := getTenantFromCtx(r)
	id := r.PathValue("id")
	if id == "" {
		SendError(w, http.StatusBadRequest, "missing_id", "id de respaldo requerido")
		return
	}

	filePath, fileName, err := h.service.GetBackupFilePath(r.Context(), tenantID, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			SendError(w, http.StatusNotFound, "not_found", "Respaldo no encontrado")
			return
		}
		SendError(w, http.StatusNotFound, "file_not_found", err.Error())
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", fileName))
	w.Header().Set("Content-Type", "application/gzip")
	http.ServeFile(w, r, filePath)
}
