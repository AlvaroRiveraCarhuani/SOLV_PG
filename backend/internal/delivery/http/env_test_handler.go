package httpdelivery

import (
	"encoding/json"
	"net/http"
	"strings"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

// EnvTestHandler gestiona las peticiones HTTP del ciclo de vida de pruebas de entorno asíncronas
type EnvTestHandler struct {
	svc *services.EnvTestService
}

// NewEnvTestHandler crea una nueva instancia del handler HTTP para pruebas de entorno
func NewEnvTestHandler(svc *services.EnvTestService) *EnvTestHandler {
	return &EnvTestHandler{svc: svc}
}

// StartEnvTest inicia un nuevo job asíncrono de verificación de entorno
// POST /api/v1/jobs/env-test
func (h *EnvTestHandler) StartEnvTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Método no permitido")
		return
	}

	var req domain.StartEnvTestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "invalid_json", "Cuerpo de solicitud inválido")
		return
	}

	job, err := h.svc.StartJob(r.Context(), req)
	if err != nil {
		SendError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	SendJSON(w, http.StatusAccepted, job, "Job de prueba de entorno iniciado")
}

// GetEnvTest consulta el estado y progreso en tiempo real de un job
// GET /api/v1/jobs/env-test/{id}
func (h *EnvTestHandler) GetEnvTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		SendError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Método no permitido")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		// Fallback para rutas sin path values estándar
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(parts) > 0 {
			id = parts[len(parts)-1]
		}
	}

	if id == "" {
		SendError(w, http.StatusBadRequest, "missing_id", "Identificador de job requerido")
		return
	}

	job, err := h.svc.GetJob(r.Context(), id)
	if err != nil {
		SendError(w, http.StatusNotFound, "job_not_found", "El job de prueba especificado no existe")
		return
	}

	SendJSON(w, http.StatusOK, job, "Detalle del job de prueba de entorno")
}

// CancelEnvTest cancela la ejecución de un job activo
// POST /api/v1/jobs/env-test/{id}/cancel
func (h *EnvTestHandler) CancelEnvTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Método no permitido")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(parts) >= 2 {
			id = parts[len(parts)-2]
		}
	}

	if id == "" {
		SendError(w, http.StatusBadRequest, "missing_id", "Identificador de job requerido")
		return
	}

	if err := h.svc.CancelJob(r.Context(), id); err != nil {
		SendError(w, http.StatusInternalServerError, "cancel_error", "Error al cancelar el job")
		return
	}

	SendJSON(w, http.StatusOK, map[string]string{"status": "canceled"}, "Job cancelado correctamente")
}
