package httpdelivery

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"solv-backend/internal/core/domain"
)

func (h *TeacherHandler) GetCourseSubmissions(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role == "student" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo docentes y administradores pueden acceder a este recurso")
		return
	}

	tenantID := getTenantFromCtx(r)

	subjectID := r.PathValue("id")
	if subjectID == "" {
		SendError(w, http.StatusBadRequest, "Missing subject ID", "El identificador de la materia es requerido")
		return
	}

	teacherID := r.Header.Get("X-User-Id")
	exerciseID := r.URL.Query().Get("exercise_id")
	verdict := r.URL.Query().Get("verdict")

	items, err := h.service.ListCourseSubmissions(r.Context(), tenantID, teacherID, subjectID, exerciseID, verdict)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			SendError(w, http.StatusNotFound, "Subject not found", "La materia solicitada no existe o no pertenece al docente")
			return
		}
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al obtener cola de entregas")
		return
	}

	SendJSON(w, http.StatusOK, items, "Cola de entregas obtenida exitosamente")
}

func (h *TeacherHandler) GetSubmissionReview(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role == "student" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: los estudiantes no pueden acceder a la vista de revisión de casos privados")
		return
	}

	tenantID := getTenantFromCtx(r)

	submissionID := r.PathValue("id")
	if submissionID == "" {
		SendError(w, http.StatusBadRequest, "Missing submission ID", "El identificador de la entrega es requerido")
		return
	}

	teacherID := r.Header.Get("X-User-Id")

	review, err := h.service.GetTeacherSubmissionReview(r.Context(), tenantID, teacherID, submissionID)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidResourceID) {
			SendError(w, http.StatusBadRequest, "invalid_submission_id", "El identificador de la entrega no tiene un formato válido")
			return
		}
		if errors.Is(err, domain.ErrNotFound) {
			SendError(w, http.StatusNotFound, "Submission not found", "La entrega solicitada no existe o no pertenece a una materia del docente")
			return
		}
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al obtener detalle de revision de entrega")
		return
	}

	sanitizeReviewPayload(review)
	SendJSON(w, http.StatusOK, review, "Detalle de revisión SpeedGrader obtenido exitosamente")
}

// sanitizeReviewPayload evita duplicar el snapshot en modalidad IDE: el visor
// ya recibe `files` y `base_files`, así que `code` y `boilerplate` se vacían
// recién en la respuesta HTTP. No se toca el repositorio porque AddComment
// valida el rango contra `code`. Si el snapshot no se pudo separar, se conserva
// el texto crudo para que el docente nunca pierda la entrega.
func sanitizeReviewPayload(review *domain.TeacherSubmissionReviewDTO) {
	if review == nil {
		return
	}
	if review.EnvironmentType == domain.WorkspaceTypeIDEPersistente && review.FilesAvailable {
		review.Code = ""
		review.Boilerplate = ""
	}
}

func (h *TeacherHandler) AddSubmissionComment(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role == "student" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo docentes pueden agregar comentarios de feedback")
		return
	}

	tenantID := getTenantFromCtx(r)

	submissionID := r.PathValue("id")
	if submissionID == "" {
		SendError(w, http.StatusBadRequest, "Missing submission ID", "El identificador de la entrega es requerido")
		return
	}

	var dto domain.AddCommentRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		SendError(w, http.StatusBadRequest, "Invalid JSON", "Cuerpo de solicitud inválido")
		return
	}

	if strings.TrimSpace(dto.Comment) == "" {
		SendError(w, http.StatusBadRequest, "Comment required", "El contenido del comentario es requerido")
		return
	}

	authorID := r.Header.Get("X-User-Id")
	if authorID == "" {
		authorID = domain.DefaultTenantID
	}

	comment := &domain.SubmissionComment{
		TenantID:           tenantID,
		SubmissionID:       submissionID,
		AuthorID:           authorID,
		LineNumber:         dto.LineNumber,
		EndLineNumber:      dto.EndLineNumber,
		FilePath:           dto.FilePath,
		LocalLineNumber:    dto.LocalLineNumber,
		LocalEndLineNumber: dto.LocalEndLineNumber,
		Comment:            dto.Comment,
	}

	if err := h.service.AddComment(r.Context(), comment); err != nil {
		if errors.Is(err, domain.ErrLineRangeInvalid) {
			SendError(w, http.StatusUnprocessableEntity, "line_range_invalid", "El rango del comentario debe empezar en una línea válida y terminar en una línea igual o posterior")
			return
		}
		if errors.Is(err, domain.ErrLineRangeSpansFiles) {
			SendError(w, http.StatusUnprocessableEntity, "line_range_spans_files", "El bloque comentado debe pertenecer a un solo archivo")
			return
		}
		if errors.Is(err, domain.ErrLineOutOfRange) {
			SendError(w, http.StatusUnprocessableEntity, "line_out_of_range", "La línea del comentario no existe en la entrega")
			return
		}
		if errors.Is(err, domain.ErrCommentFileUnknown) {
			SendError(w, http.StatusUnprocessableEntity, "comment_file_unknown", "El archivo del comentario no existe en la entrega")
			return
		}
		if errors.Is(err, domain.ErrInvalidResourceID) {
			SendError(w, http.StatusBadRequest, "invalid_submission_id", "El identificador de la entrega no tiene un formato válido")
			return
		}
		if errors.Is(err, domain.ErrNotFound) {
			SendError(w, http.StatusNotFound, "submission_not_found", "La entrega no existe en esta institución")
			return
		}
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al guardar comentario")
		return
	}

	SendJSON(w, http.StatusCreated, comment, "Comentario registrado exitosamente")
}

func (h *TeacherHandler) GetSubmissionComments(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantFromCtx(r)

	submissionID := r.PathValue("id")
	if submissionID == "" {
		SendError(w, http.StatusBadRequest, "Missing submission ID", "El identificador de la entrega es requerido")
		return
	}

	comments, err := h.service.GetCommentsBySubmission(r.Context(), tenantID, submissionID)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al obtener comentarios")
		return
	}

	SendJSON(w, http.StatusOK, comments, "Comentarios obtenidos exitosamente")
}
