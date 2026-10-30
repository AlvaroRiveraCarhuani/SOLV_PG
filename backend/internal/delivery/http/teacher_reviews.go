package httpdelivery

import (
	"encoding/json"
	"errors"
	"net/http"

	"solv-backend/internal/core/domain"
)

// GetReviewQueue devuelve las entregas que esperan corrección agrupadas por
// práctica. Es la cola real: se apoya en review_status, no en el veredicto.
func (h *TeacherHandler) GetReviewQueue(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-User-Role") == "student" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo docentes y administradores pueden acceder a la cola de revisión")
		return
	}

	tenantID := getTenantFromCtx(r)
	teacherID := r.Header.Get("X-User-Id")
	subjectID := r.URL.Query().Get("subject_id")

	groups, err := h.service.ListPendingReviewGroups(r.Context(), tenantID, teacherID, subjectID)
	if err != nil {
		sendReviewError(w, err, "Error al obtener la cola de revisión")
		return
	}

	SendJSON(w, http.StatusOK, groups, "Cola de revisión obtenida exitosamente")
}

// UpdateSubmissionReview cambia el estado de revisión de una entrega.
func (h *TeacherHandler) UpdateSubmissionReview(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-User-Role") == "student" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo docentes pueden revisar entregas")
		return
	}

	tenantID := getTenantFromCtx(r)
	teacherID := r.Header.Get("X-User-Id")
	submissionID := r.PathValue("id")
	if submissionID == "" {
		SendError(w, http.StatusBadRequest, "Missing submission ID", "El identificador de la entrega es requerido")
		return
	}

	var req domain.UpdateReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "Invalid JSON", "Cuerpo de solicitud inválido")
		return
	}

	cell, err := h.service.UpdateSubmissionReview(r.Context(), tenantID, teacherID, submissionID, req)
	if err != nil {
		sendReviewError(w, err, "Error al actualizar el estado de revisión")
		return
	}

	SendJSON(w, http.StatusOK, map[string]interface{}{
		"submission_id": submissionID,
		"review_status": req.Status,
		"cell":          cell,
	}, "Estado de revisión actualizado")
}

// SetExerciseGrade fija qué entrega cuenta para la nota de la práctica.
func (h *TeacherHandler) SetExerciseGrade(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-User-Role") == "student" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo docentes pueden cambiar la nota de una práctica")
		return
	}

	tenantID := getTenantFromCtx(r)
	teacherID := r.Header.Get("X-User-Id")
	studentID := r.PathValue("studentId")
	exerciseID := r.PathValue("exerciseId")
	if studentID == "" || exerciseID == "" {
		SendError(w, http.StatusBadRequest, "Missing identifiers", "El alumno y la práctica son requeridos")
		return
	}

	var req domain.SetExerciseGradeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "Invalid JSON", "Cuerpo de solicitud inválido")
		return
	}

	cell, err := h.service.SetExerciseGrade(r.Context(), tenantID, teacherID, studentID, exerciseID, req)
	if err != nil {
		sendReviewError(w, err, "Error al fijar la nota de la práctica")
		return
	}

	SendJSON(w, http.StatusOK, cell, "Nota de la práctica actualizada")
}

func sendReviewError(w http.ResponseWriter, err error, fallback string) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		SendError(w, http.StatusNotFound, "Not found", "El recurso solicitado no existe o no pertenece al docente")
	case errors.Is(err, domain.ErrSubmissionNotInSubject):
		SendError(w, http.StatusBadRequest, "Invalid submission", "La entrega elegida no pertenece a ese alumno y esa práctica")
	case errors.Is(err, domain.ErrInvalidReviewStatus):
		SendError(w, http.StatusUnprocessableEntity, "Invalid review status", "El estado de revisión no es válido")
	case errors.Is(err, domain.ErrInvalidReviewTransition):
		SendError(w, http.StatusConflict, "Invalid transition", "Ese cambio de estado no está permitido para la entrega")
	case errors.Is(err, domain.ErrGradeReasonRequired):
		SendError(w, http.StatusUnprocessableEntity, "Reason required", "Para cambiar la nota debes indicar un motivo de al menos 10 caracteres")
	case errors.Is(err, domain.ErrInvalidScoreRange):
		SendError(w, http.StatusUnprocessableEntity, "Invalid score", "La nota debe estar entre 0 y 100")
	case errors.Is(err, domain.ErrInvalidTenant):
		SendError(w, http.StatusBadRequest, "Invalid tenant", "La institución de la solicitud no es válida")
	case errors.Is(err, domain.ErrReviewStoreUnavailable):
		SendError(w, http.StatusServiceUnavailable, "Review unavailable", "La revisión docente no está disponible en este momento")
	default:
		SendError(w, http.StatusInternalServerError, err.Error(), fallback)
	}
}
