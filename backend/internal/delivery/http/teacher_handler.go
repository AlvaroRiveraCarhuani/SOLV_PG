package httpdelivery

import (
	"errors"
	"net/http"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

type TeacherHandler struct {
	service *services.TeacherService
}

func NewTeacherHandler(service *services.TeacherService) *TeacherHandler {
	return &TeacherHandler{service: service}
}

func (h *TeacherHandler) GetCourses(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role == "student" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo docentes y administradores pueden acceder a este recurso")
		return
	}

	tenantID, _ := r.Context().Value(domain.TenantIDKey).(string)
	if tenantID == "" {
		tenantID = r.Header.Get("X-Tenant-Id")
	}
	if tenantID == "" {
		tenantID = "00000000-0000-0000-0000-000000000001"
	}

	teacherID := r.Header.Get("X-User-Id")

	courses, err := h.service.GetCoursesSummary(r.Context(), tenantID, teacherID)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al obtener cursos del docente")
		return
	}

	SendJSON(w, http.StatusOK, courses, "Cursos del docente obtenidos exitosamente")
}

func (h *TeacherHandler) GetAttention(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role == "student" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo docentes y administradores pueden acceder a este recurso")
		return
	}

	tenantID, _ := r.Context().Value(domain.TenantIDKey).(string)
	if tenantID == "" {
		tenantID = r.Header.Get("X-Tenant-Id")
	}
	if tenantID == "" {
		tenantID = "00000000-0000-0000-0000-000000000001"
	}

	teacherID := r.Header.Get("X-User-Id")

	widget, err := h.service.GetAttentionWidget(r.Context(), tenantID, teacherID)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al obtener widget de atencion")
		return
	}

	SendJSON(w, http.StatusOK, widget, "Alertas de atencion obtenidas exitosamente")
}

func (h *TeacherHandler) GetCourseLabs(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role == "student" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo docentes y administradores pueden acceder a este recurso")
		return
	}

	tenantID, _ := r.Context().Value(domain.TenantIDKey).(string)
	if tenantID == "" {
		tenantID = r.Header.Get("X-Tenant-Id")
	}
	if tenantID == "" {
		tenantID = "00000000-0000-0000-0000-000000000001"
	}

	subjectID := r.PathValue("id")
	if subjectID == "" {
		SendError(w, http.StatusBadRequest, "Missing subject ID", "El identificador de la materia es requerido")
		return
	}

	teacherID := r.Header.Get("X-User-Id")

	stats, err := h.service.GetCourseLabsStats(r.Context(), tenantID, teacherID, subjectID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			SendError(w, http.StatusNotFound, "Subject not found", "La materia solicitada no existe o no pertenece al docente")
			return
		}
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al obtener estadisticas de laboratorios")
		return
	}

	SendJSON(w, http.StatusOK, stats, "Estadisticas de laboratorios obtenidas exitosamente")
}
