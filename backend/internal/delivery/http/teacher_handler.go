package httpdelivery

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

type TeacherHandler struct {
	service                  *services.TeacherService
	metricsSnapshotService   *services.MetricsSnapshotService
	reviewEnvironmentService *services.ReviewEnvironmentService
	authService              *services.AuthService
	terminalAuthorizer       TerminalSessionAuthorizer
	upgrader                 websocket.Upgrader
}

// TerminalSessionAuthorizer valida en el servidor que el docente pueda
// observar y ejecutar comandos en un contenedor concreto. No se alimenta de
// cabeceras del cliente.
type TerminalSessionAuthorizer interface {
	AuthorizeTerminal(ctx context.Context, tenantID, teacherID, containerID string) error
}

func NewTeacherHandler(service *services.TeacherService, metricsSnapshotService *services.MetricsSnapshotService) *TeacherHandler {
	return &TeacherHandler{
		service:                service,
		metricsSnapshotService: metricsSnapshotService,
		upgrader:               defaultUpgrader,
	}
}

func (h *TeacherHandler) SetAuthService(authService *services.AuthService) {
	h.authService = authService
}

func (h *TeacherHandler) SetTerminalAuthorizer(authorizer TerminalSessionAuthorizer) {
	h.terminalAuthorizer = authorizer
}

// SetReviewEnvironmentService habilita el preview efímero de una entrega IDE.
// Sin el servicio, la ruta responde 503 en lugar de fallar con un nil pointer.
func (h *TeacherHandler) SetReviewEnvironmentService(service *services.ReviewEnvironmentService) {
	h.reviewEnvironmentService = service
}

func (h *TeacherHandler) GetCourses(w http.ResponseWriter, r *http.Request) {
	if !hasAnyRole(r, "teacher", "admin") {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo docentes y administradores pueden acceder a este recurso")
		return
	}

	tenantID := getTenantFromCtx(r)

	teacherID := requestUserID(r)

	periodID := r.URL.Query().Get("period_id")

	// Validar period_id si viene no vacío
	if periodID != "" {
		if _, err := uuid.Parse(periodID); err != nil {
			SendError(w, http.StatusBadRequest, "invalid period_id", "period_id debe ser un UUID válido")
			return
		}
	}

	courses, err := h.service.GetCoursesSummary(r.Context(), tenantID, teacherID, periodID)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al obtener cursos del docente")
		return
	}

	SendJSON(w, http.StatusOK, courses, "Cursos del docente obtenidos exitosamente")
}

// GetOverview devuelve el resumen del panel docente: aula, cola de corrección,
// entornos abiertos y materias con sus conteos, en una sola respuesta.
func (h *TeacherHandler) GetOverview(w http.ResponseWriter, r *http.Request) {
	if !hasAnyRole(r, "teacher", "admin") {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo docentes y administradores pueden acceder a este recurso")
		return
	}

	tenantID := getTenantFromCtx(r)
	teacherID := requestUserID(r)
	periodID := r.URL.Query().Get("period_id")

	if periodID != "" {
		if _, err := uuid.Parse(periodID); err != nil {
			SendError(w, http.StatusBadRequest, "invalid period_id", "period_id debe ser un UUID válido")
			return
		}
	}

	overview, err := h.service.GetTeacherOverview(r.Context(), tenantID, teacherID, periodID)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al obtener el resumen del panel docente")
		return
	}

	SendJSON(w, http.StatusOK, overview, "Resumen del panel docente obtenido exitosamente")
}

// GetExerciseBank devuelve el read model del banco de ejercicios del docente
// con semántica de revisión por entorno (juez vs IDE).
func (h *TeacherHandler) GetExerciseBank(w http.ResponseWriter, r *http.Request) {
	if !hasAnyRole(r, "teacher", "admin") {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo docentes y administradores pueden acceder a este recurso")
		return
	}

	tenantID := getTenantFromCtx(r)
	teacherID := requestUserID(r)

	items, err := h.service.GetTeacherExerciseBank(r.Context(), tenantID, teacherID)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al obtener el banco de ejercicios")
		return
	}

	SendJSON(w, http.StatusOK, items, "Banco de ejercicios obtenido exitosamente")
}

func (h *TeacherHandler) GetCourseLabs(w http.ResponseWriter, r *http.Request) {
	if !hasAnyRole(r, "teacher", "admin") {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo docentes y administradores pueden acceder a este recurso")
		return
	}

	tenantID := getTenantFromCtx(r)

	subjectID := r.PathValue("id")
	if subjectID == "" {
		SendError(w, http.StatusBadRequest, "Missing subject ID", "El identificador de la materia es requerido")
		return
	}

	teacherID := requestUserID(r)

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

// DuplicateExercise copia un laboratorio del docente a otra de sus materias.
func (h *TeacherHandler) DuplicateExercise(w http.ResponseWriter, r *http.Request) {
	if !hasAnyRole(r, "teacher", "admin") {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo docentes y administradores pueden acceder a este recurso")
		return
	}

	tenantID := getTenantFromCtx(r)
	teacherID := requestUserID(r)
	exerciseID := r.PathValue("id")
	if exerciseID == "" {
		SendError(w, http.StatusBadRequest, "Missing exercise ID", "El identificador del laboratorio es requerido")
		return
	}

	var dto struct {
		SubjectID string `json:"subject_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		SendError(w, http.StatusBadRequest, "Invalid JSON", "Cuerpo de solicitud inválido")
		return
	}
	if _, err := uuid.Parse(dto.SubjectID); err != nil {
		SendError(w, http.StatusBadRequest, "Invalid subject ID", "La materia destino no es válida")
		return
	}

	copied, err := h.service.DuplicateExercise(r.Context(), tenantID, teacherID, exerciseID, dto.SubjectID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			SendError(w, http.StatusNotFound, "Not found", "El laboratorio o la materia no existen o no pertenecen al docente")
			return
		}
		SendError(w, http.StatusBadRequest, err.Error(), "No se pudo duplicar el laboratorio")
		return
	}

	SendJSON(w, http.StatusCreated, copied, "Laboratorio duplicado exitosamente")
}

func (h *TeacherHandler) GetLiveSessionsMetrics(w http.ResponseWriter, r *http.Request) {
	if !hasAnyRole(r, "teacher", "admin") {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo docentes y administradores pueden acceder a este recurso")
		return
	}

	tenantID := getTenantFromCtx(r)
	teacherID := requestUserID(r)

	metrics, err := h.metricsSnapshotService.GetPracticeEnvironmentMetrics(r.Context(), tenantID, teacherID)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al obtener métricas de sesiones en vivo")
		return
	}

	SendJSON(w, http.StatusOK, metrics, "Métricas de sesiones en vivo obtenidas exitosamente")
}

func (h *TeacherHandler) SearchAvailableStudents(w http.ResponseWriter, r *http.Request) {
	if !hasAnyRole(r, "teacher", "admin") {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo docentes y administradores pueden acceder a este recurso")
		return
	}

	tenantID := getTenantFromCtx(r)
	subjectID := r.PathValue("id")
	if subjectID == "" {
		SendError(w, http.StatusBadRequest, "Missing subject ID", "El identificador de la materia es requerido")
		return
	}

	teacherID := requestUserID(r)
	if hasAnyRole(r, "admin") {
		teacherID = ""
	}

	query := r.URL.Query().Get("q")

	students, err := h.service.SearchAvailableStudents(r.Context(), tenantID, teacherID, subjectID, query)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			SendError(w, http.StatusNotFound, "Subject not found", "La materia solicitada no existe o no pertenece al docente")
			return
		}
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al buscar estudiantes disponibles")
		return
	}

	SendJSON(w, http.StatusOK, students, "Estudiantes disponibles obtenidos exitosamente")
}

func (h *TeacherHandler) EnrollStudent(w http.ResponseWriter, r *http.Request) {
	if !hasAnyRole(r, "teacher", "admin") {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo docentes y administradores pueden acceder a este recurso")
		return
	}

	tenantID := getTenantFromCtx(r)
	subjectID := r.PathValue("id")
	if subjectID == "" {
		SendError(w, http.StatusBadRequest, "Missing subject ID", "El identificador de la materia es requerido")
		return
	}

	teacherID := requestUserID(r)
	if hasAnyRole(r, "admin") {
		teacherID = ""
	}

	var req struct {
		StudentID string `json:"student_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "Invalid payload", "Cuerpo de solicitud invalido")
		return
	}
	req.StudentID = strings.TrimSpace(req.StudentID)
	if req.StudentID == "" {
		SendError(w, http.StatusBadRequest, "Missing student ID", "El identificador del estudiante es requerido")
		return
	}

	if err := h.service.EnrollStudentInCourse(r.Context(), tenantID, teacherID, subjectID, req.StudentID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			SendError(w, http.StatusNotFound, "Not found", "La materia o el estudiante no existen en la institucion")
			return
		}
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al matricular al estudiante")
		return
	}

	SendJSON(w, http.StatusCreated, map[string]string{
		"student_id": req.StudentID,
		"subject_id": subjectID,
	}, "Estudiante matriculado exitosamente")
}

