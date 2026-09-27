package httpdelivery

import (
	"encoding/json"
	"errors"
	"net/http"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

// ServerPoliciesHandler expone GET/PUT /api/v1/admin/server/policies
// (submódulo 14.6 / ADR-014). El admin institucional ajusta aquí los límites
// QoS que el worker QoS recarga por ciclo.
type ServerPoliciesHandler struct {
	policiesService *services.ServerPoliciesService
}

func NewServerPoliciesHandler(policiesService *services.ServerPoliciesService) *ServerPoliciesHandler {
	return &ServerPoliciesHandler{policiesService: policiesService}
}

func (h *ServerPoliciesHandler) GetPolicies(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantFromCtx(r)

	policies, err := h.policiesService.Get(r.Context(), tenantID)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al obtener políticas del servidor")
		return
	}

	SendJSON(w, http.StatusOK, policies, "Políticas del servidor obtenidas exitosamente")
}

func (h *ServerPoliciesHandler) UpdatePolicies(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantFromCtx(r)

	var dto domain.UpdateServerPoliciesDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		SendError(w, http.StatusBadRequest, "invalid_payload", "payload JSON inválido")
		return
	}

	policies, err := h.policiesService.Update(r.Context(), tenantID, dto)
	if err != nil {
		var vErr *services.PoliciesValidationError
		if errors.As(err, &vErr) {
			SendError(w, http.StatusUnprocessableEntity, vErr.Code, vErr.Message)
			return
		}
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al actualizar políticas del servidor")
		return
	}

	SendJSON(w, http.StatusOK, policies, "Políticas del servidor actualizadas. El worker QoS las aplicará en su próximo ciclo")
}
