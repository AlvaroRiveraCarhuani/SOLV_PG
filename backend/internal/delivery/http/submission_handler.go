package httpdelivery

import (
	"encoding/json"
	"fmt"
	"net/http"

	"solv-backend/internal/core/services"
	"solv-backend/internal/delivery/http/middleware"
)

type SubmissionHandler struct {
	service *services.SubmissionService
	wsHub   *WebSocketHub
}

func NewSubmissionHandler(service *services.SubmissionService) *SubmissionHandler {
	return &SubmissionHandler{service: service}
}

func (h *SubmissionHandler) SetWebSocketHub(hub *WebSocketHub) {
	h.wsHub = hub
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

	if h.wsHub != nil && dto.StudentID != "" {
		h.wsHub.EmitToUser(dto.StudentID, WebSocketMessage{
			Event:        "EVALUATION_COMPLETED",
			SubmissionID: sub.ID,
			Stage:        "COMPLETED",
			Data:         sub,
		})
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

func (h *SubmissionHandler) GetSubmissionByID(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantIDFromContext(r.Context())
	if err != nil || tenantID == "" {
		http.Error(w, `{"error":"Tenant ID missing in context"}`, http.StatusUnauthorized)
		return
	}

	submissionID := r.PathValue("id")
	if submissionID == "" {
		http.Error(w, `{"error":"Submission ID missing"}`, http.StatusBadRequest)
		return
	}

	sub, err := h.service.GetSubmissionByID(r.Context(), tenantID, submissionID)
	if err != nil {
		http.Error(w, `{"error":"Submission not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sub)
}

type OverrideSubmissionDTO struct {
	Verdict        string  `json:"verdict"`
	OverrideReason string  `json:"override_reason"`
	Score          *int    `json:"score,omitempty"`
}

func (h *SubmissionHandler) OverrideSubmission(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantIDFromContext(r.Context())
	if err != nil || tenantID == "" {
		http.Error(w, `{"error":"Tenant ID missing in context"}`, http.StatusUnauthorized)
		return
	}

	submissionID := r.PathValue("id")
	if submissionID == "" {
		http.Error(w, `{"error":"Submission ID missing"}`, http.StatusBadRequest)
		return
	}

	var dto OverrideSubmissionDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
		return
	}

	if dto.Verdict == "" || dto.OverrideReason == "" {
		http.Error(w, `{"error":"verdict and override_reason are required"}`, http.StatusBadRequest)
		return
	}

	var gradedBy *string
	userID := r.Header.Get("X-User-Id")
	if userID != "" {
		gradedBy = &userID
	}

	err = h.service.OverrideSubmission(r.Context(), tenantID, submissionID, dto.Verdict, dto.OverrideReason, dto.Score, gradedBy)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "overridden",
		"message": "Submission verdict updated successfully",
	})
}

