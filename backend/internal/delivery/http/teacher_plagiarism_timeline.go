package httpdelivery

import (
	"errors"
	"net/http"

	"solv-backend/internal/core/domain"
)

func (h *TeacherHandler) AnalyzePlagiarism(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role == "student" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo docentes y administradores pueden acceder a este recurso")
		return
	}

	tenantID := getTenantFromCtx(r)
	teacherID := r.Header.Get("X-User-Id")

	subjectID := r.URL.Query().Get("subject_id")
	if subjectID == "" {
		subjectID = r.PathValue("id")
	}
	if subjectID == "" {
		SendError(w, http.StatusBadRequest, "Missing subject ID", "El identificador de la materia es requerido")
		return
	}

	exerciseID := r.URL.Query().Get("exercise_id")

	report, err := h.service.AnalyzePlagiarism(r.Context(), tenantID, teacherID, subjectID, exerciseID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			SendError(w, http.StatusNotFound, "Subject not found", "La materia solicitada no existe")
			return
		}
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al ejecutar análisis de plagio")
		return
	}

	SendJSON(w, http.StatusOK, report, "Análisis de similitud estructural AST completado exitosamente")
}

func (h *TeacherHandler) GetTimeline(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role == "student" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo docentes y administradores pueden acceder a este recurso")
		return
	}

	tenantID := getTenantFromCtx(r)
	teacherID := r.Header.Get("X-User-Id")
	submissionID := r.PathValue("id")
	if submissionID == "" {
		SendError(w, http.StatusBadRequest, "Missing submission ID", "El identificador de la entrega es requerido")
		return
	}

	timeline, err := h.service.GetSubmissionTimeline(r.Context(), tenantID, teacherID, submissionID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			SendError(w, http.StatusNotFound, "Submission not found", "La entrega solicitada no existe")
			return
		}
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al obtener timeline de la entrega")
		return
	}

	SendJSON(w, http.StatusOK, timeline, "Telemetría de Time-Travel Replay obtenida exitosamente")
}

func (h *TeacherHandler) GetKeystrokeEvents(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role == "student" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo docentes y administradores pueden acceder a este recurso")
		return
	}

	tenantID := getTenantFromCtx(r)
	teacherID := r.Header.Get("X-User-Id")
	submissionID := r.PathValue("id")
	if submissionID == "" {
		SendError(w, http.StatusBadRequest, "Missing submission ID", "El identificador de la entrega es requerido")
		return
	}

	report, err := h.service.GetKeystrokeEvents(r.Context(), tenantID, teacherID, submissionID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			SendError(w, http.StatusNotFound, "Submission not found", "La entrega solicitada no existe")
			return
		}
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al obtener eventos de escritura")
		return
	}

	SendJSON(w, http.StatusOK, report, "Eventos de escritura obtenidos exitosamente")
}
