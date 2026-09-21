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
	govService         *services.AdminGovernanceService
	imageService       *services.ImageVerificationService
	auditLogRepo       domain.AuditLogRepository
}

func NewAdminAcademicHandler(
	periodService *services.AcademicPeriodService,
	maintenanceService *services.MaintenanceService,
	govService *services.AdminGovernanceService,
) *AdminAcademicHandler {
	return &AdminAcademicHandler{
		periodService:      periodService,
		maintenanceService: maintenanceService,
		govService:         govService,
	}
}

func (h *AdminAcademicHandler) WithImageService(imageService *services.ImageVerificationService) *AdminAcademicHandler {
	h.imageService = imageService
	return h
}

func (h *AdminAcademicHandler) WithAuditLogRepo(auditLogRepo domain.AuditLogRepository) *AdminAcademicHandler {
	h.auditLogRepo = auditLogRepo
	return h
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

// -----------------------------------------------------------------------------
// Student Directory & Reset OOM (ADR-033)
// -----------------------------------------------------------------------------

func (h *AdminAcademicHandler) ListStudents(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role == "student" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: rol student no autorizado")
		return
	}

	tenantID := getTenantFromCtx(r)
	search := r.URL.Query().Get("search")
	subjectID := r.URL.Query().Get("subject_id")
	status := r.URL.Query().Get("status")
	periodID := r.URL.Query().Get("period_id")

	if h.govService == nil {
		SendError(w, http.StatusInternalServerError, "service_unavailable", "Servicio de gobernanza no configurado")
		return
	}

	students, err := h.govService.ListStudents(r.Context(), tenantID, search, subjectID, status, periodID)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al obtener directorio de estudiantes")
		return
	}

	if students == nil {
		students = []*domain.AdminStudentDirectoryItem{}
	}

	SendJSON(w, http.StatusOK, students, "Directorio de estudiantes obtenido exitosamente")
}

func (h *AdminAcademicHandler) GetStudentCourses(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role == "student" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: rol student no autorizado")
		return
	}

	tenantID := getTenantFromCtx(r)
	studentID := r.PathValue("id")
	if studentID == "" {
		SendError(w, http.StatusBadRequest, "missing_id", "id de estudiante requerido")
		return
	}

	if h.govService == nil {
		SendError(w, http.StatusInternalServerError, "service_unavailable", "Servicio de gobernanza no configurado")
		return
	}

	courses, err := h.govService.GetStudentCourses(r.Context(), tenantID, studentID)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al obtener materias del estudiante")
		return
	}

	if courses == nil {
		courses = []*domain.AdminStudentCourseItem{}
	}

	SendJSON(w, http.StatusOK, courses, "Materias del estudiante obtenidas exitosamente")
}

func (h *AdminAcademicHandler) CreateStudent(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role == "student" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo administradores pueden registrar estudiantes")
		return
	}

	tenantID := getTenantFromCtx(r)
	var dto domain.CreateStudentDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		SendError(w, http.StatusBadRequest, "invalid_payload", "payload JSON inválido")
		return
	}

	if h.govService == nil {
		SendError(w, http.StatusInternalServerError, "service_unavailable", "Servicio de gobernanza no configurado")
		return
	}

	student, err := h.govService.CreateStudent(r.Context(), tenantID, dto)
	if err != nil {
		if strings.Contains(err.Error(), "already_exists") {
			SendError(w, http.StatusConflict, "already_exists", "Ya existe un estudiante registrado con ese correo institucional")
			return
		}
		SendError(w, http.StatusBadRequest, err.Error(), err.Error())
		return
	}

	SendJSON(w, http.StatusCreated, student, "Estudiante registrado exitosamente")
}

func (h *AdminAcademicHandler) UpdateStudentStatus(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role == "student" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo administradores pueden suspender o reactivar cuentas")
		return
	}

	tenantID := getTenantFromCtx(r)
	studentID := r.PathValue("id")
	if studentID == "" {
		SendError(w, http.StatusBadRequest, "missing_id", "id de estudiante requerido")
		return
	}

	var dto domain.UpdateStudentStatusDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		SendError(w, http.StatusBadRequest, "invalid_payload", "payload JSON inválido")
		return
	}

	if h.govService == nil {
		SendError(w, http.StatusInternalServerError, "service_unavailable", "Servicio de gobernanza no configurado")
		return
	}

	if err := h.govService.UpdateStudentStatus(r.Context(), tenantID, studentID, dto); err != nil {
		SendError(w, http.StatusBadRequest, err.Error(), "Error al actualizar estado del estudiante")
		return
	}

	SendJSON(w, http.StatusOK, map[string]string{"id": studentID, "status": dto.Status}, "Estado del estudiante actualizado exitosamente")
}

func (h *AdminAcademicHandler) ResetStudentOOM(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role == "student" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo administradores pueden resetear penalizaciones OOM")
		return
	}

	tenantID := getTenantFromCtx(r)
	studentID := r.PathValue("id")
	if studentID == "" {
		SendError(w, http.StatusBadRequest, "missing_id", "id de estudiante requerido")
		return
	}

	var dto domain.ResetOOMDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		SendError(w, http.StatusBadRequest, "invalid_payload", "payload JSON inválido")
		return
	}

	if len(dto.Reason) < 10 {
		SendError(w, http.StatusUnprocessableEntity, "reason_too_short", "El motivo de justificación debe contener al menos 10 caracteres")
		return
	}

	if h.govService == nil {
		SendError(w, http.StatusInternalServerError, "service_unavailable", "Servicio de gobernanza no configurado")
		return
	}

	result, err := h.govService.ResetStudentOOM(r.Context(), tenantID, studentID, dto)
	if err != nil {
		if errors.Is(err, services.ErrReasonTooShort) {
			SendError(w, http.StatusUnprocessableEntity, "reason_too_short", "El motivo de justificación debe contener al menos 10 caracteres")
			return
		}
		if strings.Contains(err.Error(), "not found") {
			SendError(w, http.StatusNotFound, "not_found", "Estudiante no encontrado en el tenant")
			return
		}
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al resetear penalizaciones OOM")
		return
	}

	SendJSON(w, http.StatusOK, result, "Penalizaciones OOM reseteadas exitosamente")
}

// -----------------------------------------------------------------------------
// Docker Templates Governance (ADR-030)
// -----------------------------------------------------------------------------

func (h *AdminAcademicHandler) ListTemplates(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role == "student" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo administradores y docentes pueden consultar plantillas")
		return
	}

	tenantID := getTenantFromCtx(r)
	status := r.URL.Query().Get("status")
	search := r.URL.Query().Get("search")

	if h.govService == nil {
		SendError(w, http.StatusInternalServerError, "service_unavailable", "Servicio de gobernanza no configurado")
		return
	}

	templates, err := h.govService.ListTemplates(r.Context(), tenantID, status, search)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al obtener listado de plantillas")
		return
	}

	SendJSON(w, http.StatusOK, templates, "Listado de plantillas obtenido exitosamente")
}

func (h *AdminAcademicHandler) ReviewTemplate(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role != "admin" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo administradores pueden revisar plantillas")
		return
	}

	tenantID := getTenantFromCtx(r)
	templateID := r.PathValue("id")
	if templateID == "" {
		SendError(w, http.StatusBadRequest, "missing_id", "id de plantilla requerido")
		return
	}

	adminID := r.Header.Get("X-User-Id")
	if adminID == "" {
		adminID = "00000000-0000-0000-0000-000000000001"
	}

	var dto domain.ReviewTemplateDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		SendError(w, http.StatusBadRequest, "invalid_payload", "payload JSON inválido")
		return
	}

	if h.govService == nil {
		SendError(w, http.StatusInternalServerError, "service_unavailable", "Servicio de gobernanza no configurado")
		return
	}

	item, err := h.govService.ReviewTemplate(r.Context(), tenantID, templateID, adminID, dto)
	if err != nil {
		if errors.Is(err, services.ErrInvalidReviewStatus) || errors.Is(err, services.ErrRejectionReasonRequired) {
			SendError(w, http.StatusUnprocessableEntity, "validation_failed", err.Error())
			return
		}
		if strings.Contains(err.Error(), "not found") {
			SendError(w, http.StatusNotFound, "not_found", "Plantilla no encontrada")
			return
		}
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al revisar la plantilla")
		return
	}

	if h.auditLogRepo != nil {
		meta, _ := json.Marshal(map[string]any{
			"template_id":      item.ID,
			"status":           dto.Status,
			"rejection_reason": dto.RejectionReason,
		})
		_ = h.auditLogRepo.Create(r.Context(), &domain.AuditLog{
			TenantID:     tenantID,
			ActorID:      adminID,
			Action:       "TEMPLATE_REVIEWED",
			ResourceType: "lab_template",
			ResourceID:   &item.ID,
			StatusCode:   http.StatusOK,
			Metadata:     meta,
			IPAddress:    r.RemoteAddr,
			UserAgent:    r.UserAgent(),
		})
	}

	SendJSON(w, http.StatusOK, item, "Plantilla revisada exitosamente")
}

func (h *AdminAcademicHandler) CreateTemplate(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role != "admin" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo administradores pueden registrar plantillas oficiales")
		return
	}

	tenantID := getTenantFromCtx(r)
	adminID := r.Header.Get("X-User-Id")
	if adminID == "" {
		adminID = "00000000-0000-0000-0000-000000000001"
	}

	var dto domain.CreateOfficialTemplateDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		SendError(w, http.StatusBadRequest, "invalid_payload", "payload JSON inválido")
		return
	}

	if dto.Name == "" || dto.DockerImage == "" {
		SendError(w, http.StatusUnprocessableEntity, "validation_failed", "Nombre e imagen Docker son requeridos")
		return
	}

	if h.govService == nil {
		SendError(w, http.StatusInternalServerError, "service_unavailable", "Servicio de gobernanza no configurado")
		return
	}

	item, err := h.govService.CreateOfficialTemplate(r.Context(), tenantID, adminID, dto)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, services.ErrInvalidDockerImage) || errors.Is(err, services.ErrLatestTagForbidden) || strings.Contains(err.Error(), "requerido") {
			status = http.StatusUnprocessableEntity
		}
		SendError(w, status, err.Error(), err.Error())
		return
	}

	if h.auditLogRepo != nil {
		meta, _ := json.Marshal(map[string]any{
			"template_id": item.ID,
			"name":        item.Name,
			"image":       item.DockerImage,
		})
		_ = h.auditLogRepo.Create(r.Context(), &domain.AuditLog{
			TenantID:     tenantID,
			ActorID:      adminID,
			Action:       "TEMPLATE_REGISTERED",
			ResourceType: "lab_template",
			ResourceID:   &item.ID,
			StatusCode:   http.StatusCreated,
			Metadata:     meta,
			IPAddress:    r.RemoteAddr,
			UserAgent:    r.UserAgent(),
		})
	}

	SendJSON(w, http.StatusCreated, item, "Plantilla oficial registrada exitosamente")
}

func (h *AdminAcademicHandler) DuplicateTemplate(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role != "admin" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo administradores pueden duplicar plantillas")
		return
	}

	tenantID := getTenantFromCtx(r)
	templateID := r.PathValue("id")
	if templateID == "" {
		SendError(w, http.StatusBadRequest, "missing_id", "id de plantilla requerido")
		return
	}

	adminID := r.Header.Get("X-User-Id")
	if adminID == "" {
		adminID = "00000000-0000-0000-0000-000000000001"
	}

	if h.govService == nil {
		SendError(w, http.StatusInternalServerError, "service_unavailable", "Servicio de gobernanza no configurado")
		return
	}

	item, err := h.govService.DuplicateTemplate(r.Context(), tenantID, templateID, adminID)
	if err != nil {
		if strings.Contains(err.Error(), "no encontrada") {
			SendError(w, http.StatusNotFound, "not_found", err.Error())
			return
		}
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al duplicar la plantilla")
		return
	}

	if h.auditLogRepo != nil {
		meta, _ := json.Marshal(map[string]any{
			"source_template_id": templateID,
			"duplicated_id":      item.ID,
			"name":               item.Name,
		})
		_ = h.auditLogRepo.Create(r.Context(), &domain.AuditLog{
			TenantID:     tenantID,
			ActorID:      adminID,
			Action:       "TEMPLATE_DUPLICATED",
			ResourceType: "lab_template",
			ResourceID:   &item.ID,
			StatusCode:   http.StatusCreated,
			Metadata:     meta,
			IPAddress:    r.RemoteAddr,
			UserAgent:    r.UserAgent(),
		})
	}

	SendJSON(w, http.StatusCreated, item, "Plantilla duplicada exitosamente en cola de auditoría")
}

// -----------------------------------------------------------------------------
// Emergency Actions (ADR-032)
// -----------------------------------------------------------------------------

func (h *AdminAcademicHandler) ExecuteEmergencyAction(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role != "admin" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo administradores pueden ejecutar acciones de emergencia")
		return
	}

	tenantID := getTenantFromCtx(r)
	action := r.PathValue("action")
	if action == "" {
		SendError(w, http.StatusBadRequest, "missing_action", "Acción de emergencia requerida")
		return
	}

	adminID := r.Header.Get("X-User-Id")
	if adminID == "" {
		adminID = "00000000-0000-0000-0000-000000000001"
	}

	var req domain.EmergencyActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "invalid_payload", "payload JSON inválido")
		return
	}

	if h.govService == nil {
		SendError(w, http.StatusInternalServerError, "service_unavailable", "Servicio de gobernanza no configurado")
		return
	}

	result, err := h.govService.ExecuteEmergencyAction(r.Context(), tenantID, adminID, action, req)
	if err != nil {
		if errors.Is(err, services.ErrInvalidConfirmationPhrase) {
			SendError(w, http.StatusUnprocessableEntity, "invalid_confirmation_phrase", "La frase de confirmación no coincide exactamente")
			return
		}
		if errors.Is(err, services.ErrUnknownEmergencyAction) {
			SendError(w, http.StatusUnprocessableEntity, "unknown_action", "Acción de emergencia no reconocida")
			return
		}
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al ejecutar acción de emergencia")
		return
	}

	SendJSON(w, http.StatusOK, result, result.Message)
}

// -----------------------------------------------------------------------------
// Image Verification & Local Images (ADR-030)
// -----------------------------------------------------------------------------

func (h *AdminAcademicHandler) ListLocalImages(w http.ResponseWriter, r *http.Request) {
	if h.imageService == nil {
		SendError(w, http.StatusServiceUnavailable, "service_unavailable", "Servicio de verificación de imágenes no disponible")
		return
	}

	images, err := h.imageService.ListLocalImages(r.Context())
	if err != nil {
		SendError(w, http.StatusInternalServerError, "docker_error", err.Error())
		return
	}

	SendJSON(w, http.StatusOK, images, "Imágenes locales recuperadas exitosamente")
}

func (h *AdminAcademicHandler) VerifyImage(w http.ResponseWriter, r *http.Request) {
	if h.imageService == nil {
		SendError(w, http.StatusServiceUnavailable, "service_unavailable", "Servicio de verificación de imágenes no disponible")
		return
	}

	var req domain.VerifyImageRequest
	if r.Method == http.MethodGet {
		req.Image = r.URL.Query().Get("image")
		req.Force = r.URL.Query().Get("force") == "true"
	} else {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			SendError(w, http.StatusBadRequest, "invalid_payload", "payload JSON inválido")
			return
		}
	}

	if strings.TrimSpace(req.Image) == "" {
		SendError(w, http.StatusUnprocessableEntity, "validation_failed", "El campo image es obligatorio")
		return
	}

	result, err := h.imageService.VerifyImage(r.Context(), req.Image, req.Force)
	if err != nil {
		if errors.Is(err, services.ErrImageFormatInvalid) || errors.Is(err, services.ErrLatestTagForbiddenVerif) || errors.Is(err, services.ErrRegistryNotAllowed) {
			SendError(w, http.StatusUnprocessableEntity, "validation_failed", err.Error())
			return
		}
		SendError(w, http.StatusInternalServerError, "verification_failed", err.Error())
		return
	}

	// Registro de auditoría si se utilizó bypass manual force=true (SEC-04)
	if req.Force && h.auditLogRepo != nil {
		meta, _ := json.Marshal(map[string]any{
			"image_ref": req.Image,
			"bypass":    "force_verification",
			"reason":    "Bypass manual de verificación de imagen Docker",
		})
		adminID := r.Header.Get("X-User-Id")
		if adminID == "" {
			adminID = "00000000-0000-0000-0000-000000000001"
		}
		_ = h.auditLogRepo.Create(r.Context(), &domain.AuditLog{
			TenantID:     getTenantFromCtx(r),
			ActorID:      adminID,
			Action:       "IMAGE_VERIFICATION_FORCE_BYPASS",
			ResourceType: "docker_image",
			StatusCode:   http.StatusOK,
			Metadata:     meta,
			IPAddress:    r.RemoteAddr,
			UserAgent:    r.UserAgent(),
		})
	}

	SendJSON(w, http.StatusOK, result, "Verificación de imagen completada")
}
