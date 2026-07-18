package handlers

import (
	"encoding/json"
	"net/http"

	"solv-backend/internal/core/services"
)

type LabHandler struct {
	service *services.LabService
}

func NewLabHandler(service *services.LabService) *LabHandler {
	return &LabHandler{service: service}
}

type StartLabRequest struct {
	UserID     string `json:"user_id"`
	TemplateID string `json:"template_id"`
	RAMLimitMB int    `json:"ram_limit_mb"`
	UserEmail  string `json:"user_email"`
}

func (h *LabHandler) HandleStartLab(w http.ResponseWriter, r *http.Request) {
	var req StartLabRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	instance, err := h.service.StartLab(r.Context(), req.UserID, req.TemplateID, req.RAMLimitMB, req.UserEmail)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // Use 201 for strict REST, but returning 200 is fine as well.
	json.NewEncoder(w).Encode(instance)
}
