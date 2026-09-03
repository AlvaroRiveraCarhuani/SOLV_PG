package httpdelivery

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

type AdminAcademicHandler struct {
	periodService      *services.AcademicPeriodService
	maintenanceService *services.MaintenanceService
}

func NewAdminAcademicHandler(
	periodService *services.AcademicPeriodService,
	maintenanceService *services.MaintenanceService,
) *AdminAcademicHandler {
	return &AdminAcademicHandler{
		periodService:      periodService,
		maintenanceService: maintenanceService,
	}
}

func getTenantFromCtx(r *http.Request) string {
	tenantID, _ := r.Context().Value(domain.TenantIDKey).(string)
	if tenantID == "" {
		tenantID = r.Header.Get("X-Tenant-Id")
	}
	if tenantID == "" {
		tenantID = "00000000-0000-0000-0000-000000000001"
	}
	return tenantID
}

// -----------------------------------------------------------------------------
// Maintenance Endpoints (ADR-031)
// -----------------------------------------------------------------------------

func (h *AdminAcademicHandler) EnableMaintenance(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantFromCtx(r)

	var dto domain.EnableMaintenanceDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		SendError(w, http.StatusBadRequest, "invalid_payload", "payload JSON inválido")
		return
	}

	if err := h.maintenanceService.EnableMaintenance(r.Context(), tenantID, dto); err != nil {
		SendError(w, http.StatusBadRequest, err.Error(), "Error al activar modo mantenimiento")
		return
	}

	SendJSON(w, http.StatusOK, map[string]string{"status": "maintenance_enabled"}, "Modo mantenimiento activado exitosamente")
}

func (h *AdminAcademicHandler) DisableMaintenance(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantFromCtx(r)

	if err := h.maintenanceService.DisableMaintenance(r.Context(), tenantID); err != nil {
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al desactivar modo mantenimiento")
		return
	}

	SendJSON(w, http.StatusOK, map[string]string{"status": "maintenance_disabled"}, "Modo mantenimiento desactivado exitosamente")
}

func (h *AdminAcademicHandler) GetMaintenanceStatus(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantFromCtx(r)

	status, err := h.maintenanceService.GetStatus(r.Context(), tenantID)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al obtener estado de mantenimiento")
		return
	}

	SendJSON(w, http.StatusOK, status, "Estado de mantenimiento obtenido exitosamente")
}

// -----------------------------------------------------------------------------
// Academic Periods Endpoints (ADR-029)
// -----------------------------------------------------------------------------

func (h *AdminAcademicHandler) ListPeriods(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantFromCtx(r)

	periods, err := h.periodService.ListPeriods(r.Context(), tenantID)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al listar periodos académicos")
		return
	}

	if periods == nil {
		periods = []*domain.AcademicPeriod{}
	}

	SendJSON(w, http.StatusOK, periods, "Periodos académicos obtenidos exitosamente")
}

func (h *AdminAcademicHandler) CreatePeriod(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantFromCtx(r)

	var dto domain.CreateAcademicPeriodDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		SendError(w, http.StatusBadRequest, "invalid_payload", "payload JSON inválido")
		return
	}

	if dto.Name == "" || dto.Code == "" || dto.StartDate == "" || dto.EndDate == "" {
		SendError(w, http.StatusUnprocessableEntity, "validation_failed", "todos los campos name, code, start_date y end_date son obligatorios")
		return
	}

	period, err := h.periodService.CreatePeriod(r.Context(), tenantID, dto)
	if err != nil {
		if errors.Is(err, services.ErrInvalidDateRange) {
			SendError(w, http.StatusUnprocessableEntity, "invalid_date_range", "end_date debe ser posterior o igual a start_date")
			return
		}
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "uq_tenant_period_code") {
			SendError(w, http.StatusConflict, "duplicate_code", "ya existe un periodo académico con este código")
			return
		}
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al crear periodo académico")
		return
	}

	SendJSON(w, http.StatusCreated, period, "Periodo académico creado exitosamente")
}

func (h *AdminAcademicHandler) UpdatePeriod(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantFromCtx(r)
	id := r.PathValue("id")
	if id == "" {
		SendError(w, http.StatusBadRequest, "missing_id", "id de periodo requerido")
		return
	}

	var dto domain.UpdateAcademicPeriodDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		SendError(w, http.StatusBadRequest, "invalid_payload", "payload JSON inválido")
		return
	}

	period, err := h.periodService.UpdatePeriod(r.Context(), tenantID, id, dto)
	if err != nil {
		if errors.Is(err, services.ErrInvalidDateRange) {
			SendError(w, http.StatusUnprocessableEntity, "invalid_date_range", "end_date debe ser posterior o igual a start_date")
			return
		}
		if strings.Contains(err.Error(), "not found") {
			SendError(w, http.StatusNotFound, "not_found", "periodo académico no encontrado")
			return
		}
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al actualizar periodo académico")
		return
	}

	SendJSON(w, http.StatusOK, period, "Periodo académico actualizado exitosamente")
}

func (h *AdminAcademicHandler) DeletePeriod(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantFromCtx(r)
	id := r.PathValue("id")
	if id == "" {
		SendError(w, http.StatusBadRequest, "missing_id", "id de periodo requerido")
		return
	}

	err := h.periodService.DeletePeriod(r.Context(), tenantID, id)
	if err != nil {
		if strings.Contains(err.Error(), "associated subjects") {
			SendError(w, http.StatusConflict, "conflict_associated_subjects", "No se puede eliminar un periodo académico con materias asociadas")
			return
		}
		if strings.Contains(err.Error(), "not found") {
			SendError(w, http.StatusNotFound, "not_found", "periodo académico no encontrado")
			return
		}
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al eliminar periodo académico")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
