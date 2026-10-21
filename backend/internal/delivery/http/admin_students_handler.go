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
