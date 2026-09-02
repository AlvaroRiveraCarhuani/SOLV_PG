package httpdelivery

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

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

func (h *TeacherHandler) GetCourseSubmissions(w http.ResponseWriter, r *http.Request) {
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

	tenantID, _ := r.Context().Value(domain.TenantIDKey).(string)
	if tenantID == "" {
		tenantID = r.Header.Get("X-Tenant-Id")
	}
	if tenantID == "" {
		tenantID = "00000000-0000-0000-0000-000000000001"
	}

	submissionID := r.PathValue("id")
	if submissionID == "" {
		SendError(w, http.StatusBadRequest, "Missing submission ID", "El identificador de la entrega es requerido")
		return
	}

	teacherID := r.Header.Get("X-User-Id")

	review, err := h.service.GetTeacherSubmissionReview(r.Context(), tenantID, teacherID, submissionID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			SendError(w, http.StatusNotFound, "Submission not found", "La entrega solicitada no existe o no pertenece a una materia del docente")
			return
		}
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al obtener detalle de revision de entrega")
		return
	}

	SendJSON(w, http.StatusOK, review, "Detalle de revisión SpeedGrader obtenido exitosamente")
}

func (h *TeacherHandler) AddSubmissionComment(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role == "student" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo docentes pueden agregar comentarios de feedback")
		return
	}

	tenantID, _ := r.Context().Value(domain.TenantIDKey).(string)
	if tenantID == "" {
		tenantID = r.Header.Get("X-Tenant-Id")
	}
	if tenantID == "" {
		tenantID = "00000000-0000-0000-0000-000000000001"
	}

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
		authorID = "00000000-0000-0000-0000-000000000001"
	}

	comment := &domain.SubmissionComment{
		TenantID:     tenantID,
		SubmissionID: submissionID,
		AuthorID:     authorID,
		LineNumber:   dto.LineNumber,
		Comment:      dto.Comment,
	}

	if err := h.service.AddComment(r.Context(), comment); err != nil {
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al guardar comentario")
		return
	}

	SendJSON(w, http.StatusCreated, comment, "Comentario registrado exitosamente")
}

func (h *TeacherHandler) GetSubmissionComments(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := r.Context().Value(domain.TenantIDKey).(string)
	if tenantID == "" {
		tenantID = r.Header.Get("X-Tenant-Id")
	}
	if tenantID == "" {
		tenantID = "00000000-0000-0000-0000-000000000001"
	}

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

func (h *TeacherHandler) RunEphemeral(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role == "student" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo docentes pueden ejecutar el runner efímero")
		return
	}

	tenantID, _ := r.Context().Value(domain.TenantIDKey).(string)
	if tenantID == "" {
		tenantID = r.Header.Get("X-Tenant-Id")
	}
	if tenantID == "" {
		tenantID = "00000000-0000-0000-0000-000000000001"
	}

	submissionID := r.PathValue("id")
	if submissionID == "" {
		SendError(w, http.StatusBadRequest, "Missing submission ID", "El identificador de la entrega es requerido")
		return
	}

	var dto domain.EphemeralRunRequestDTO
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&dto)
	}

	teacherID := r.Header.Get("X-User-Id")

	result, err := h.service.RunEphemeral(r.Context(), tenantID, teacherID, submissionID, dto.Code, dto.Language)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			SendError(w, http.StatusNotFound, "Submission not found", "La entrega solicitada no existe")
			return
		}
		SendError(w, http.StatusInternalServerError, err.Error(), "Error en ejecución efímera")
		return
	}

	SendJSON(w, http.StatusOK, result, "Ejecución efímera completada exitosamente")
}

func (h *TeacherHandler) ExportGrades(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role == "student" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo docentes pueden exportar calificaciones")
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

	csvData, filename, err := h.service.ExportCourseGradesCSV(r.Context(), tenantID, teacherID, subjectID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			SendError(w, http.StatusNotFound, "Subject not found", "La materia solicitada no existe o no pertenece al docente")
			return
		}
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al exportar calificaciones")
		return
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(csvData)
}
