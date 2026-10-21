package httpdelivery

import (
	"errors"
	"net/http"

	"solv-backend/internal/core/domain"
)

func (h *TeacherHandler) GetCourseAnalytics(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role == "student" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo docentes y administradores pueden acceder a este recurso")
		return
	}

	tenantID := getTenantFromCtx(r)

	subjectID := r.PathValue("id")
	if subjectID == "" {
		subjectID = r.PathValue("courseId")
	}
	if subjectID == "" {
		SendError(w, http.StatusBadRequest, "Missing course ID", "El identificador del curso es requerido")
		return
	}

	teacherID := r.Header.Get("X-User-Id")

	analytics, err := h.service.GetCourseAnalytics(r.Context(), tenantID, teacherID, subjectID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			SendError(w, http.StatusNotFound, "Course not found", "El curso solicitado no existe")
			return
		}
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al calcular analitica del curso")
		return
	}

	SendJSON(w, http.StatusOK, analytics, "Analitica del curso obtenida exitosamente")
}
