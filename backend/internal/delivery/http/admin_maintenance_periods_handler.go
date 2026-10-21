package httpdelivery

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
	"strings"
)

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
		var vErr *services.MaintenanceValidationError
		if errors.As(err, &vErr) {
			SendError(w, http.StatusUnprocessableEntity, vErr.Code, vErr.Message)
			return
		}
		SendError(w, http.StatusBadRequest, err.Error(), "Error al activar modo mantenimiento")
		return
	}

	h.writeMaintenanceAudit(r, tenantID, "MAINTENANCE_ENABLED", dto.Reason, dto.Until, http.StatusOK)

	SendJSON(w, http.StatusOK, map[string]string{"status": "maintenance_enabled"}, "Modo mantenimiento activado exitosamente")
}

func (h *AdminAcademicHandler) DisableMaintenance(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantFromCtx(r)

	if err := h.maintenanceService.DisableMaintenance(r.Context(), tenantID); err != nil {
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al desactivar modo mantenimiento")
		return
	}

	h.writeMaintenanceAudit(r, tenantID, "MAINTENANCE_DISABLED", "", "", http.StatusOK)

	SendJSON(w, http.StatusOK, map[string]string{"status": "maintenance_disabled"}, "Modo mantenimiento desactivado exitosamente")
}

func (h *AdminAcademicHandler) GetMaintenanceStatus(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantFromCtx(r)

	if cleared, _ := h.maintenanceService.ClearExpiredMaintenance(r.Context(), tenantID); cleared {
		h.writeMaintenanceAudit(r, tenantID, "MAINTENANCE_AUTO_DISABLED", "vigencia vencida", "", http.StatusOK)
	}

	status, err := h.maintenanceService.GetStatus(r.Context(), tenantID)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al obtener estado de mantenimiento")
		return
	}

	SendJSON(w, http.StatusOK, status, "Estado de mantenimiento obtenido exitosamente")
}

// writeMaintenanceAudit records MAINTENANCE_ENABLED / MAINTENANCE_DISABLED /
// MAINTENANCE_AUTO_DISABLED events with motive, vigencia and actor metadata,
// mirroring the TEMPLATE_REVIEWED pattern. Best-effort: never fails the request.
func (h *AdminAcademicHandler) writeMaintenanceAudit(r *http.Request, tenantID, action, reason, until string, statusCode int) {
	if h.auditLogRepo == nil {
		return
	}
	adminID := getUserIDFromCtx(r)
	if adminID == "" {
		adminID = r.Header.Get("X-User-Id")
	}
	if adminID == "" {
		adminID = domain.DefaultTenantID
	}
	meta, _ := json.Marshal(map[string]any{
		"reason": reason,
		"until":  until,
		"actor":  adminID,
	})
	ipAddr := r.RemoteAddr
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		ipAddr = host
	}
	_ = h.auditLogRepo.Create(r.Context(), &domain.AuditLog{
		TenantID:     tenantID,
		ActorID:      adminID,
		Action:       action,
		ResourceType: "maintenance",
		StatusCode:   statusCode,
		Metadata:     meta,
		IPAddress:    ipAddr,
		UserAgent:    r.UserAgent(),
	})
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
		if errors.Is(err, services.ErrPeriodExpired) {
			SendError(w, http.StatusUnprocessableEntity, "period_expired", "No se puede registrar como activo un periodo académico cuya fecha ya finalizó")
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
		if errors.Is(err, services.ErrPeriodExpired) {
			SendError(w, http.StatusUnprocessableEntity, "period_expired", "No se puede activar un periodo académico cuya fecha ya finalizó")
			return
		}
		if errors.Is(err, services.ErrPeriodArchived) {
			SendError(w, http.StatusConflict, "period_archived", "El periodo académico está formalmente archivado y es inmutable (ADR-029)")
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

// ArchivePeriod congela formalmente un periodo académico con confirmación fuerte
// por tipeo del código (ADR-029). Es irreversible desde la API.
func (h *AdminAcademicHandler) ArchivePeriod(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantFromCtx(r)
	id := r.PathValue("id")
	if id == "" {
		SendError(w, http.StatusBadRequest, "missing_id", "id de periodo requerido")
		return
	}

	var dto domain.ArchiveAcademicPeriodDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		SendError(w, http.StatusBadRequest, "invalid_payload", "payload JSON inválido")
		return
	}
	if dto.ConfirmationCode == "" {
		SendError(w, http.StatusUnprocessableEntity, "validation_failed", "confirmation_code es obligatorio para archivar un periodo")
		return
	}

	archivedBy := getUserIDFromCtx(r)
	if err := h.periodService.ArchivePeriod(r.Context(), tenantID, id, archivedBy, dto.ConfirmationCode); err != nil {
		if errors.Is(err, services.ErrConfirmationFailed) {
			SendError(w, http.StatusUnprocessableEntity, "confirmation_failed", "El código de confirmación no coincide con el código del período")
			return
		}
		if errors.Is(err, services.ErrPeriodArchived) {
			SendError(w, http.StatusConflict, "period_archived", "El periodo académico ya está archivado")
			return
		}
		if strings.Contains(err.Error(), "not found") {
			SendError(w, http.StatusNotFound, "not_found", "periodo académico no encontrado")
			return
		}
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al archivar periodo académico")
		return
	}

	SendJSON(w, http.StatusOK, map[string]string{"status": "archived"}, "Periodo académico archivado. Sus materias quedaron en modo solo lectura (:ro)")
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
		if errors.Is(err, services.ErrPeriodArchived) {
			SendError(w, http.StatusConflict, "period_archived", "El periodo académico está formalmente archivado y no puede eliminarse (ADR-029)")
			return
		}
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

// -----------------------------------------------------------------------------
// Course Reassignment (ADR-036)
// -----------------------------------------------------------------------------

func (h *AdminAcademicHandler) ReassignCourse(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role == "student" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo administradores pueden reasignar cursos")
		return
	}

	tenantID := getTenantFromCtx(r)
	id := r.PathValue("id")
	if id == "" {
		SendError(w, http.StatusBadRequest, "missing_id", "id de materia requerido")
		return
	}

	var dto domain.ReassignCourseDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		SendError(w, http.StatusBadRequest, "invalid_payload", "payload JSON inválido")
		return
	}

	if dto.NewTeacherID == "" {
		SendError(w, http.StatusUnprocessableEntity, "validation_failed", "new_teacher_id es obligatorio")
		return
	}

	if h.govService == nil {
		SendError(w, http.StatusInternalServerError, "service_unavailable", "Servicio de gobernanza no configurado")
		return
	}

	subject, err := h.govService.ReassignCourse(r.Context(), tenantID, id, dto)
	if err != nil {
		if errors.Is(err, services.ErrTeacherNotFoundOrRole) {
			SendError(w, http.StatusUnprocessableEntity, "invalid_teacher", "El usuario asignado no existe o no tiene rol de docente")
			return
		}
		if strings.Contains(err.Error(), "not found") {
			SendError(w, http.StatusNotFound, "not_found", "Materia no encontrada")
			return
		}
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al reasignar docente a la materia")
		return
	}

	SendJSON(w, http.StatusOK, subject, "Docente titular reasignado exitosamente")
}
