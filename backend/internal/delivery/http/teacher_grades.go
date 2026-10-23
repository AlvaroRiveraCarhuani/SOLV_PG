package httpdelivery

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"solv-backend/internal/core/domain"
)

func (h *TeacherHandler) GetCourseGradesMatrix(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role == "student" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo docentes pueden consultar la matriz de calificaciones")
		return
	}

	tenantID := getTenantFromCtx(r)

	subjectID := r.PathValue("id")
	if subjectID == "" {
		SendError(w, http.StatusBadRequest, "Missing subject ID", "El identificador de la materia es requerido")
		return
	}

	teacherID := r.Header.Get("X-User-Id")

	matrix, err := h.service.GetCourseGradesMatrix(r.Context(), tenantID, teacherID, subjectID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			SendError(w, http.StatusNotFound, "Subject not found", "La materia solicitada no existe o no pertenece al docente")
			return
		}
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al obtener matriz de calificaciones")
		return
	}

	SendJSON(w, http.StatusOK, matrix, "Matriz de calificaciones obtenida exitosamente")
}

func (h *TeacherHandler) ExportGrades(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role == "student" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo docentes pueden exportar calificaciones")
		return
	}

	tenantID := getTenantFromCtx(r)

	subjectID := r.PathValue("id")
	if subjectID == "" {
		SendError(w, http.StatusBadRequest, "Missing subject ID", "El identificador de la materia es requerido")
		return
	}

	teacherID := r.Header.Get("X-User-Id")

	csvData, filename, err := h.service.ExportCourseGradesCSV(r.Context(), tenantID, teacherID, subjectID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			SendError(w, http.StatusNotFound, "Subject not found", "La materia solicitada no existe o no pertenece al docente")
			return
		}
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al exportar calificaciones")
		return
	}

	// El nombre viene del servicio; se eliminan comillas y CR/LF para impedir header injection.
	safeName := strings.NewReplacer(`"`, "", "\r", "", "\n", "").Replace(filename)
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", safeName))
	w.WriteHeader(http.StatusOK)
	// El CSV se sirve como descarga (Content-Type text/csv y Content-Disposition
	// attachment), no como HTML: el nombre ya está saneado y el contenido no se
	// interpreta como markup.
	//nolint:gosec
	_, _ = w.Write(csvData)
}
