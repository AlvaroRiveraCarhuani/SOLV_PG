package httpdelivery

import (
	"encoding/json"
	"net/http"

	"solv-backend/internal/core/services"
	"solv-backend/internal/delivery/http/middleware"
)

type SubmissionHandler struct {
	service *services.SubmissionService
}

func NewSubmissionHandler(service *services.SubmissionService) *SubmissionHandler {
	return &SubmissionHandler{service: service}
}

func (h *SubmissionHandler) CreateSubmission(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantIDFromContext(r.Context())
	if err != nil || tenantID == "" {
		http.Error(w, `{"error":"Tenant ID missing in context"}`, http.StatusUnauthorized)
		return
	}

	var dto services.CreateSubmissionDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		http.Error(w, `{"error":"Invalid request payload"}`, http.StatusBadRequest)
		return
	}

	sub, err := h.service.CreateSubmission(r.Context(), tenantID, dto)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(sub)
}

func (h *SubmissionHandler) ListSubmissionsByExercise(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantIDFromContext(r.Context())
	if err != nil || tenantID == "" {
		http.Error(w, `{"error":"Tenant ID missing in context"}`, http.StatusUnauthorized)
		return
	}

	exerciseID := r.PathValue("id")
	if exerciseID == "" {
		http.Error(w, `{"error":"Exercise ID missing"}`, http.StatusBadRequest)
		return
	}

	userID := r.Header.Get("X-User-Id")
	userRole := r.Header.Get("X-User-Role")
	if userID == "" {
		userID = "anonymous"
	}
	if userRole == "" {
		userRole = "student"
	}

	list, err := h.service.GetSubmissionsForExercise(r.Context(), tenantID, exerciseID, userID, userRole)
	if err != nil {
		http.Error(w, `{"error":"Failed to retrieve submissions"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"exercise_id": exerciseID,
		"data":        list,
	})
}
