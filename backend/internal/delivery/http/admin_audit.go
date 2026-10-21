package httpdelivery

import (
	"encoding/json"
	"net/http"
	"strconv"

	"solv-backend/internal/delivery/http/middleware"
)

func (h *AdminHandler) ListAuditLogs(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantIDFromContext(r.Context())
	if err != nil || tenantID == "" {
		http.Error(w, `{"error":"Tenant ID missing in context"}`, http.StatusUnauthorized)
		return
	}

	search := r.URL.Query().Get("search")
	if search == "" {
		search = r.URL.Query().Get("actor_id")
	}
	action := r.URL.Query().Get("action")
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}
	offset := 0
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	logs, err := h.auditRepo.ListFiltered(r.Context(), tenantID, search, action, limit, offset)
	if err != nil {
		http.Error(w, `{"error":"Failed to retrieve audit logs"}`, http.StatusInternalServerError)
		return
	}
	total, err := h.auditRepo.CountFiltered(r.Context(), tenantID, search, action)
	if err != nil {
		http.Error(w, `{"error":"Failed to count audit logs"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"tenant_id": tenantID,
		"limit":     limit,
		"offset":    offset,
		"total":     total,
		"data":      logs,
	})
}

// GetActorTimeline sirve la cronología aislada de un actor para el drawer
// off-canvas del wireframe AUDIT_LOGS.md (identidad resuelta por el repo).
func (h *AdminHandler) GetActorTimeline(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantIDFromContext(r.Context())
	if err != nil || tenantID == "" {
		http.Error(w, `{"error":"Tenant ID missing in context"}`, http.StatusUnauthorized)
		return
	}

	actorID := r.PathValue("actorId")
	if actorID == "" {
		http.Error(w, `{"error":"missing_actor"}`, http.StatusBadRequest)
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 200
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 200 {
		limit = l
	}

	logs, err := h.auditRepo.ListByActorTimeline(r.Context(), tenantID, actorID, limit)
	if err != nil {
		http.Error(w, `{"error":"Failed to retrieve actor timeline"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"tenant_id": tenantID,
		"actor_id":  actorID,
		"count":     len(logs),
		"data":      logs,
	})
}
