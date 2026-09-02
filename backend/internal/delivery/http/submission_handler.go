package httpdelivery

import (
	"encoding/json"
	"net/http"
	"strings"

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
	role := r.Header.Get("X-User-Role")
	if role == "student" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo docentes y administradores pueden modificar calificaciones")
		return
	}

	tenantID, err := middleware.GetTenantIDFromContext(r.Context())
	if err != nil || tenantID == "" {
		tenantID = r.Header.Get("X-Tenant-Id")
	}
	if tenantID == "" {
		tenantID = "00000000-0000-0000-0000-000000000001"
	}

	submissionID := r.PathValue("id")
	if submissionID == "" {
		SendError(w, http.StatusBadRequest, "Missing submission ID", "El identificador de la entrega es requerido")
		return
	}

	var dto OverrideSubmissionDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		SendError(w, http.StatusBadRequest, "Invalid JSON", "Cuerpo de solicitud inválido")
		return
	}

	if dto.Verdict == "" || dto.OverrideReason == "" {
		SendError(w, http.StatusBadRequest, "Validation Error", "verdict y override_reason son obligatorios")
		return
	}

	if len(strings.TrimSpace(dto.OverrideReason)) < 10 {
		SendError(w, http.StatusUnprocessableEntity, "Validation Error", "La justificación debe tener al menos 10 caracteres")
		return
	}

	var gradedBy *string
	userID := r.Header.Get("X-User-Id")
	if userID != "" {
		gradedBy = &userID
	}

	err = h.service.OverrideSubmission(r.Context(), tenantID, submissionID, dto.Verdict, dto.OverrideReason, dto.Score, gradedBy)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al registrar override de calificacion")
		return
	}

	SendJSON(w, http.StatusOK, map[string]string{
		"status":  "overridden",
		"verdict": dto.Verdict,
	}, "Calificación actualizada exitosamente")
}

