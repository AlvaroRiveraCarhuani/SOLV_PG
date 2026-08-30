package httpdelivery

import (
	"encoding/json"
	"net/http"
	"strconv"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/delivery/http/middleware"
)

type AdminHandler struct {
	auditRepo     domain.AuditLogRepository
	tenantRepo    domain.TenantRepository
	workspaceRepo domain.WorkspaceRepository
}

func NewAdminHandler(
	auditRepo domain.AuditLogRepository,
	tenantRepo domain.TenantRepository,
	workspaceRepo domain.WorkspaceRepository,
) *AdminHandler {
	return &AdminHandler{
		auditRepo:     auditRepo,
		tenantRepo:    tenantRepo,
		workspaceRepo: workspaceRepo,
	}
}

func (h *AdminHandler) ListAuditLogs(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantIDFromContext(r.Context())
	if err != nil || tenantID == "" {
		http.Error(w, `{"error":"Tenant ID missing in context"}`, http.StatusUnauthorized)
		return
	}

	actorID := r.URL.Query().Get("actor_id")
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

	logs, err := h.auditRepo.ListFiltered(r.Context(), tenantID, actorID, action, limit, offset)
	if err != nil {
		http.Error(w, `{"error":"Failed to retrieve audit logs"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tenant_id": tenantID,
		"limit":     limit,
		"offset":    offset,
		"data":      logs,
	})
}

type UpdateBrandingDTO struct {
	LogoURL            string `json:"logo_url"`
	InstitutionName    string `json:"institution_name"`
	TenantPrimaryColor string `json:"tenant_primary_color"`
	SupportEmail       string `json:"support_email"`
}

func (h *AdminHandler) UpdateBranding(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantIDFromContext(r.Context())
	if err != nil || tenantID == "" {
		http.Error(w, `{"error":"Tenant ID missing in context"}`, http.StatusUnauthorized)
		return
	}

	var dto UpdateBrandingDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		http.Error(w, `{"error":"Invalid request payload"}`, http.StatusBadRequest)
		return
	}

	tenant, err := h.tenantRepo.GetByID(r.Context(), tenantID)
	if err != nil {
		http.Error(w, `{"error":"Tenant not found"}`, http.StatusNotFound)
		return
	}

	var currentConfig map[string]interface{}
	if len(tenant.Config) > 0 {
		_ = json.Unmarshal(tenant.Config, &currentConfig)
	}
	if currentConfig == nil {
		currentConfig = make(map[string]interface{})
	}

	if dto.LogoURL != "" {
		currentConfig["logo_url"] = dto.LogoURL
	}
	if dto.InstitutionName != "" {
		currentConfig["institution_name"] = dto.InstitutionName
	}
	if dto.TenantPrimaryColor != "" {
		currentConfig["tenant_primary_color"] = dto.TenantPrimaryColor
	}
	if dto.SupportEmail != "" {
		currentConfig["support_email"] = dto.SupportEmail
	}

	newConfigBytes, err := json.Marshal(currentConfig)
	if err != nil {
		http.Error(w, `{"error":"Failed to serialize config"}`, http.StatusInternalServerError)
		return
	}

	if err := h.tenantRepo.UpdateConfig(r.Context(), tenantID, newConfigBytes); err != nil {
		http.Error(w, `{"error":"Failed to update branding in database"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "updated",
		"message": "Branding configuration updated successfully",
		"config":  currentConfig,
	})
}

type HealthMetricsResponse struct {
	TenantID        string `json:"tenant_id"`
	RunningLabs     int    `json:"running_labs"`
	HibernatedLabs  int    `json:"hibernated_labs"`
	OOMKilledLabs   int    `json:"oom_killed_labs"`
	TotalRAMAllocMB int64  `json:"total_ram_alloc_mb"`
	HealthStatus    string `json:"health_status"`
}

func (h *AdminHandler) GetHealthMetrics(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantIDFromContext(r.Context())
	if err != nil || tenantID == "" {
		http.Error(w, `{"error":"Tenant ID missing in context"}`, http.StatusUnauthorized)
		return
	}

	allRunning, err := h.workspaceRepo.GetAllRunningWorkspaces(r.Context())
	if err != nil {
		allRunning = []*domain.WorkspaceInstance{}
	}

	runningCount := 0
	oomCount := 0
	var totalRAM int64 = 0

	for _, ws := range allRunning {
		if ws.TenantID == tenantID {
			if ws.Status == "running" {
				runningCount++
				totalRAM += ws.MemoryLimitMB
			}
			if ws.Status == "failed" || ws.OOMStrikeCount > 0 {
				oomCount++
			}
		}
	}

	resp := HealthMetricsResponse{
		TenantID:        tenantID,
		RunningLabs:     runningCount,
		HibernatedLabs:  0,
		OOMKilledLabs:   oomCount,
		TotalRAMAllocMB: totalRAM,
		HealthStatus:    "healthy",
	}

	if oomCount > 5 {
		resp.HealthStatus = "warning"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
