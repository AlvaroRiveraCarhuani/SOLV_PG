package httpdelivery

import (
	"errors"
	"net/http"
	"strings"

	"solv-backend/internal/core/domain"
)

// StartReviewEnvironment crea (o reabre) el preview de una entrega IDE. El
// docente recibe una URL para abrir en una pestaña nueva; el QoS Worker apaga el
// contenedor cuando la app deja de tener tráfico.
func (h *TeacherHandler) StartReviewEnvironment(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-User-Role") == "student" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo docentes pueden probar la aplicación de una entrega")
		return
	}
	if h.reviewEnvironmentService == nil {
		SendError(w, http.StatusServiceUnavailable, "review_environment_unavailable", "El nodo no tiene orquestador de previews configurado")
		return
	}

	submissionID := strings.TrimSpace(r.PathValue("id"))
	if submissionID == "" {
		SendError(w, http.StatusBadRequest, "missing_submission_id", "El identificador de la entrega es requerido")
		return
	}
	teacherID := strings.TrimSpace(r.Header.Get("X-User-Id"))
	if teacherID == "" {
		SendError(w, http.StatusUnauthorized, "missing_user", "La sesión no identifica al docente que abre el entorno")
		return
	}

	access, err := h.reviewEnvironmentService.Start(
		r.Context(),
		getTenantFromCtx(r),
		teacherID,
		submissionID,
	)
	if err != nil {
		writeReviewEnvironmentError(w, err)
		return
	}
	SendJSON(w, http.StatusOK, access, "Entorno de prueba listo")
}

func writeReviewEnvironmentError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidResourceID):
		SendError(w, http.StatusBadRequest, "invalid_submission_id", "El identificador de la entrega no tiene un formato válido")
	case errors.Is(err, domain.ErrInvalidTenant):
		SendError(w, http.StatusBadRequest, "invalid_tenant", "El tenant de la sesión no es válido")
	case errors.Is(err, domain.ErrNotFound):
		SendError(w, http.StatusNotFound, "submission_not_found", "La entrega solicitada no existe")
	case errors.Is(err, domain.ErrReviewEnvironmentRequiresIDE):
		SendError(w, http.StatusConflict, "review_environment_requires_ide", "Solo los laboratorios IDE ejecutan una aplicación que se pueda abrir en el navegador")
	case errors.Is(err, domain.ErrReviewEnvironmentCommandMissing):
		SendError(w, http.StatusConflict, "app_run_command_missing", "La práctica no define cómo arrancar su aplicación. Agregá el comando y el puerto en el editor del laboratorio")
	case errors.Is(err, domain.ErrReviewEnvironmentSnapshotUnavailable):
		SendError(w, http.StatusConflict, "project_snapshot_unavailable", "La entrega no tiene un proyecto multiarchivo que se pueda ejecutar")
	case errors.Is(err, domain.ErrReviewEnvironmentTemplateMissing):
		SendError(w, http.StatusConflict, "template_image_missing", "La práctica no resuelve una imagen de contenedor para ejecutar la aplicación")
	case errors.Is(err, domain.ErrInvalidAppPort):
		SendError(w, http.StatusConflict, "invalid_app_port", "El puerto de la aplicación debe estar entre 1 y 65535")
	case errors.Is(err, domain.ErrReviewEnvironmentStarting):
		SendError(w, http.StatusConflict, "review_environment_starting", "El entorno de prueba se está iniciando; esperá unos segundos y reintentá")
	case errors.Is(err, domain.ErrReviewEnvironmentHostMemory):
		SendError(w, http.StatusServiceUnavailable, "oom_guard", "El nodo no tiene memoria disponible para abrir el entorno de prueba. Cerrá otros laboratorios e intentá de nuevo")
	case errors.Is(err, domain.ErrReviewEnvironmentNotReady):
		SendError(w, http.StatusBadGateway, "app_not_ready", "El contenedor arrancó pero la aplicación no quedó escuchando en el puerto declarado. Revisá el comando de arranque")
	case errors.Is(err, domain.ErrReviewEnvironmentStartFailed):
		SendError(w, http.StatusBadGateway, "app_start_failed", "No se pudo iniciar el contenedor de la aplicación. Revisá la imagen y los logs del laboratorio")
	case errors.Is(err, domain.ErrReviewEnvironmentUnavailable):
		SendError(w, http.StatusServiceUnavailable, "review_environment_unavailable", "El nodo no tiene orquestador de previews configurado")
	default:
		SendError(w, http.StatusInternalServerError, "review_environment_failed", "No se pudo abrir el entorno de prueba")
	}
}
