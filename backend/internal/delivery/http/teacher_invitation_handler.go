package httpdelivery

import (
	"encoding/json"
	"net/http"

	"solv-backend/internal/core/services"
	"solv-backend/internal/delivery/http/middleware"
)

type TeacherInvitationHandler struct {
	service *services.TeacherInvitationService
}

func NewTeacherInvitationHandler(service *services.TeacherInvitationService) *TeacherInvitationHandler {
	return &TeacherInvitationHandler{service: service}
}

func (h *TeacherInvitationHandler) CreateInvitation(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantIDFromContext(r.Context())
	if err != nil || tenantID == "" {
		http.Error(w, `{"error":"Tenant ID missing in context"}`, http.StatusUnauthorized)
		return
	}

	var req struct {
		Email         string `json:"email"`
		DurationHours int    `json:"duration_hours"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request payload"}`, http.StatusBadRequest)
		return
	}

	inv, err := h.service.CreateInvitation(r.Context(), tenantID, req.Email, req.DurationHours)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(inv)
}

func (h *TeacherInvitationHandler) AcceptInvitation(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantIDFromContext(r.Context())
	if err != nil || tenantID == "" {
		http.Error(w, `{"error":"Tenant ID missing in context"}`, http.StatusUnauthorized)
		return
	}

	userID := r.Header.Get("X-User-Id")
	userEmail := r.Header.Get("X-User-Email")

	var req struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request payload"}`, http.StatusBadRequest)
		return
	}

	if err := h.service.AcceptInvitation(r.Context(), tenantID, req.Token, userID, userEmail); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "accepted",
		"message": "User role updated to teacher successfully",
	})
}
