package httpdelivery

import (
	"errors"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/google/uuid"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

type TeacherHandler struct {
	service              *services.TeacherService
	metricsSnapshotService *services.MetricsSnapshotService
	authService          *services.AuthService
	upgrader             websocket.Upgrader
}

func NewTeacherHandler(service *services.TeacherService, metricsSnapshotService *services.MetricsSnapshotService) *TeacherHandler {
	return &TeacherHandler{
		service:              service,
		metricsSnapshotService: metricsSnapshotService,
		upgrader:             defaultUpgrader,
	}
}

func (h *TeacherHandler) SetAuthService(authService *services.AuthService) {
	h.authService = authService
}

func (h *TeacherHandler) GetCourses(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role == "student" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo docentes y administradores pueden acceder a este recurso")
		return
	}

	tenantID := getTenantFromCtx(r)

	teacherID := r.Header.Get("X-User-Id")

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

func (h *TeacherHandler) GetAttention(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role == "student" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo docentes y administradores pueden acceder a este recurso")
		return
	}

	tenantID := getTenantFromCtx(r)

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

	tenantID := getTenantFromCtx(r)

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

func (h *TeacherHandler) GetLiveSessionsMetrics(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role == "student" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo docentes y administradores pueden acceder a este recurso")
		return
	}

	tenantID := getTenantFromCtx(r)
	teacherID := r.Header.Get("X-User-Id")
	
	metrics, err := h.metricsSnapshotService.GetWorkspaceMetrics(r.Context(), tenantID, teacherID)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al obtener métricas de sesiones en vivo")
		return
	}

	SendJSON(w, http.StatusOK, metrics, "Métricas de sesiones en vivo obtenidas exitosamente")
}
