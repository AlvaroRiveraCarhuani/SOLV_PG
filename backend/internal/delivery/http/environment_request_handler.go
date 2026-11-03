package httpdelivery

import (
	"encoding/json"
	"errors"
	"net/http"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

// EnvironmentRequestHandler expone el pedido de entorno del docente y la cola
// donde el administrador lo responde. El docente nunca envía una imagen ni una
// cuota de memoria: envía una intención, y el backend arma la receta.
type EnvironmentRequestHandler struct {
	service *services.EnvironmentRequestService
}

func NewEnvironmentRequestHandler(service *services.EnvironmentRequestService) *EnvironmentRequestHandler {
	return &EnvironmentRequestHandler{service: service}
}

func (h *EnvironmentRequestHandler) requireTeacherRole(w http.ResponseWriter, r *http.Request) bool {
	role := r.Header.Get("X-User-Role")
	if role == "" {
		role = domain.GetUserRole(r.Context())
	}
	if role == "student" {
		SendError(w, http.StatusForbidden, "forbidden", "Acceso denegado: solo docentes y administradores pueden solicitar entornos")
		return false
	}
	return true
}

// Preview devuelve la receta armada sin guardarla, junto con la plantilla
// existente o la solicitud gemela si las hay.
func (h *EnvironmentRequestHandler) Preview(w http.ResponseWriter, r *http.Request) {
	if !h.requireTeacherRole(w, r) {
		return
	}
	var dto domain.EnvironmentRequestPreviewDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		SendError(w, http.StatusBadRequest, "invalid_payload", "payload JSON inválido")
		return
	}
	preview, err := h.service.Preview(r.Context(), getTenantFromCtx(r), getUserIDFromCtx(r), dto)
	if err != nil {
		SendError(w, http.StatusUnprocessableEntity, "preview_failed", err.Error())
		return
	}
	SendJSON(w, http.StatusOK, preview, "Receta armada")
}

// Create registra la solicitud. Si la receta ya está en revisión, suma la
// materia del docente a esa ficha.
func (h *EnvironmentRequestHandler) Create(w http.ResponseWriter, r *http.Request) {
	if !h.requireTeacherRole(w, r) {
		return
	}
	var dto domain.CreateEnvironmentRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		SendError(w, http.StatusBadRequest, "invalid_payload", "payload JSON inválido")
		return
	}
	created, err := h.service.Create(r.Context(), getTenantFromCtx(r), getUserIDFromCtx(r), dto)
	if err != nil {
		if errors.Is(err, domain.ErrEnvironmentRequestInvalid) {
			SendError(w, http.StatusUnprocessableEntity, "invalid_request", err.Error())
			return
		}
		SendError(w, http.StatusInternalServerError, "create_failed", err.Error())
		return
	}
	status := http.StatusCreated
	if created.Joined {
		status = http.StatusOK
	}
	SendJSON(w, status, created, created.Message)
}

// ListMine devuelve solo las solicitudes del docente autenticado.
func (h *EnvironmentRequestHandler) ListMine(w http.ResponseWriter, r *http.Request) {
	if !h.requireTeacherRole(w, r) {
		return
	}
	items, err := h.service.ListMine(r.Context(), getTenantFromCtx(r), getUserIDFromCtx(r))
	if err != nil {
		SendError(w, http.StatusInternalServerError, "fetch_failed", err.Error())
		return
	}
	SendJSON(w, http.StatusOK, items, "Mis solicitudes de entorno")
}

// GetByID devuelve una solicitud propia.
func (h *EnvironmentRequestHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	if !h.requireTeacherRole(w, r) {
		return
	}
	id := r.PathValue("id")
	item, err := h.service.ListMine(r.Context(), getTenantFromCtx(r), getUserIDFromCtx(r))
	if err != nil {
		SendError(w, http.StatusInternalServerError, "fetch_failed", err.Error())
		return
	}
	for _, request := range item {
		if request.ID == id {
			SendJSON(w, http.StatusOK, request, "Solicitud de entorno")
			return
		}
	}
	SendError(w, http.StatusNotFound, "not_found", "Solicitud de entorno no encontrada")
}

// ListForReview es la cola del administrador.
func (h *EnvironmentRequestHandler) ListForReview(w http.ResponseWriter, r *http.Request) {
	if !requireAdminRole(w, r) {
		return
	}
	items, err := h.service.ListForReview(r.Context(), getTenantFromCtx(r), r.URL.Query().Get("status"))
	if err != nil {
		if errors.Is(err, domain.ErrEnvironmentRequestInvalid) {
			SendError(w, http.StatusBadRequest, "invalid_status", err.Error())
			return
		}
		SendError(w, http.StatusInternalServerError, "fetch_failed", err.Error())
		return
	}
	SendJSON(w, http.StatusOK, items, "Solicitudes de entorno por revisar")
}

// Review registra la respuesta del administrador.
func (h *EnvironmentRequestHandler) Review(w http.ResponseWriter, r *http.Request) {
	if !requireAdminRole(w, r) {
		return
	}
	id := r.PathValue("id")
	if id == "" {
		SendError(w, http.StatusBadRequest, "missing_id", "Falta el identificador de la solicitud")
		return
	}
	var dto domain.ReviewEnvironmentRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		SendError(w, http.StatusBadRequest, "invalid_payload", "payload JSON inválido")
		return
	}
	reviewed, err := h.service.Review(r.Context(), getTenantFromCtx(r), id, getUserIDFromCtx(r), dto)
	if err != nil {
		if errors.Is(err, domain.ErrEnvironmentRequestInvalid) {
			SendError(w, http.StatusUnprocessableEntity, "invalid_review", err.Error())
			return
		}
		if errors.Is(err, domain.ErrEnvironmentRequestNotFound) {
			SendError(w, http.StatusNotFound, "not_found", "Solicitud de entorno no encontrada")
			return
		}
		SendError(w, http.StatusInternalServerError, "review_failed", err.Error())
		return
	}
	SendJSON(w, http.StatusOK, reviewed, "Solicitud revisada")
}
