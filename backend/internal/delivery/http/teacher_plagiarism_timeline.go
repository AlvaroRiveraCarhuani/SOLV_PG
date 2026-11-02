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

// GetPlagiarismPair entrega el documento del visor: las dos entregas, sus
// tramos compartidos y la telemetría resumida. No expone casos de prueba,
// rúbrica ni comentarios.
func (h *TeacherHandler) GetPlagiarismPair(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role == "student" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo docentes y administradores pueden acceder a este recurso")
		return
	}

	tenantID := getTenantFromCtx(r)
	teacherID := r.Header.Get("X-User-Id")
	if role == "admin" {
		teacherID = ""
	} else if teacherID == "" {
		SendError(w, http.StatusForbidden, "plagiarism_pair_forbidden", "No se pudo identificar al docente de la sesión")
		return
	}

	submissionA := r.PathValue("submission_a")
	submissionB := r.PathValue("submission_b")
	if submissionA == "" || submissionB == "" {
		SendError(w, http.StatusBadRequest, "missing_submissions", "Las dos entregas del par son requeridas")
		return
	}

	pair, err := h.service.GetPlagiarismPair(r.Context(), tenantID, teacherID, submissionA, submissionB)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrPlagiarismPairForbidden):
			SendError(w, http.StatusForbidden, "plagiarism_pair_forbidden", "Alguna de las entregas no pertenece a tus materias")
		case errors.Is(err, domain.ErrPlagiarismPairDifferentExercise):
			SendError(w, http.StatusUnprocessableEntity, "plagiarism_pair_different_exercise", "Las entregas pertenecen a prácticas distintas")
		case errors.Is(err, domain.ErrPlagiarismPairSameSubmission):
			SendError(w, http.StatusBadRequest, "plagiarism_pair_same_submission", "El par necesita dos entregas distintas")
		case errors.Is(err, domain.ErrInvalidResourceID):
			SendError(w, http.StatusBadRequest, "invalid_submission_id", "El identificador de la entrega no es válido")
		case errors.Is(err, domain.ErrNotFound):
			SendError(w, http.StatusNotFound, "plagiarism_pair_not_found", "El par de entregas no existe")
		default:
			SendError(w, http.StatusInternalServerError, "plagiarism_pair_failed", "No se pudo cargar el par de entregas")
		}
		return
	}

	SendJSON(w, http.StatusOK, pair, "Par de entregas listo")
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
		if errors.Is(err, domain.ErrInvalidResourceID) {
			SendError(w, http.StatusBadRequest, "invalid_submission_id", "El identificador de la entrega no tiene un formato válido")
			return
		}
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
		if errors.Is(err, domain.ErrInvalidResourceID) {
			SendError(w, http.StatusBadRequest, "invalid_submission_id", "El identificador de la entrega no tiene un formato válido")
			return
		}
		if errors.Is(err, domain.ErrNotFound) {
			SendError(w, http.StatusNotFound, "Submission not found", "La entrega solicitada no existe")
			return
		}
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al obtener eventos de escritura")
		return
	}

	SendJSON(w, http.StatusOK, report, "Eventos de escritura obtenidos exitosamente")
}
