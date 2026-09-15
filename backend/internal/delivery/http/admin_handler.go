package httpdelivery

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/delivery/http/middleware"
)

type AdminHandler struct {
	auditRepo     domain.AuditLogRepository
	tenantRepo    domain.TenantRepository
	workspaceRepo domain.WorkspaceRepository
	subjectRepo   domain.SubjectRepository
	hostMonitor   domain.HostMonitor
}

func NewAdminHandler(
	auditRepo domain.AuditLogRepository,
	tenantRepo domain.TenantRepository,
	workspaceRepo domain.WorkspaceRepository,
	subjectRepo domain.SubjectRepository,
	hostMonitor domain.HostMonitor,
) *AdminHandler {
	return &AdminHandler{
		auditRepo:     auditRepo,
		tenantRepo:    tenantRepo,
		workspaceRepo: workspaceRepo,
		subjectRepo:   subjectRepo,
		hostMonitor:   hostMonitor,
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
	if err != nil || tenant == nil {
		http.Error(w, `{"error":"Tenant not found"}`, http.StatusNotFound)
		return
	}

	var currentConfig map[string]interface{}
	if len(tenant.Config) > 0 {
		if err := json.Unmarshal(tenant.Config, &currentConfig); err != nil {
			currentConfig = make(map[string]interface{})
		}
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

type HostHardwareMetricsDTO struct {
	RAMUsedBytes   uint64  `json:"ram_used_bytes"`
	RAMTotalBytes  uint64  `json:"ram_total_bytes"`
	RAMPercent     float64 `json:"ram_percent"`
	CPUCores       int     `json:"cpu_cores"`
	CPUPercent     float64 `json:"cpu_percent"`
	DiskUsedBytes  uint64  `json:"disk_used_bytes"`
	DiskTotalBytes uint64  `json:"disk_total_bytes"`
	DiskPercent    float64 `json:"disk_percent"`
}

type HealthMetricsResponse struct {
	TenantID        string                  `json:"tenant_id"`
	RunningLabs     int                     `json:"running_labs"`
	HibernatedLabs  int                     `json:"hibernated_labs"`
	OOMKilledLabs   int                     `json:"oom_killed_labs"`
	TotalRAMAllocMB int64                   `json:"total_ram_alloc_mb"`
	HealthStatus    string                  `json:"health_status"`
	HostHardware    *HostHardwareMetricsDTO `json:"host_hardware,omitempty"`
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
	hibernatedCount := 0
	oomCount := 0
	var totalRAM int64 = 0

	for _, ws := range allRunning {
		if ws.TenantID == tenantID {
			if ws.Status == "running" {
				runningCount++
				totalRAM += ws.MemoryLimitMB
			} else if ws.Status == "hibernated" {
				hibernatedCount++
			}
			if ws.Status == "failed" || ws.Status == "oom_killed" || ws.OOMStrikeCount > 0 {
				oomCount++
			}
		}
	}

	var hostHardware *HostHardwareMetricsDTO
	if v, memErr := mem.VirtualMemory(); memErr == nil {
		cores, _ := cpu.Counts(true)
		cpuPercents, _ := cpu.Percent(0, false)
		cpuVal := 0.0
		if len(cpuPercents) > 0 {
			cpuVal = cpuPercents[0]
		}
		diskUsage, _ := disk.Usage("/")
		diskUsed := uint64(0)
		diskTotal := uint64(0)
		diskPct := 0.0
		if diskUsage != nil {
			diskUsed = diskUsage.Used
			diskTotal = diskUsage.Total
			diskPct = diskUsage.UsedPercent
		}

		hostHardware = &HostHardwareMetricsDTO{
			RAMUsedBytes:   v.Used,
			RAMTotalBytes:  v.Total,
			RAMPercent:     v.UsedPercent,
			CPUCores:       cores,
			CPUPercent:     cpuVal,
			DiskUsedBytes:  diskUsed,
			DiskTotalBytes: diskTotal,
			DiskPercent:    diskPct,
		}
	}

	resp := HealthMetricsResponse{
		TenantID:        tenantID,
		RunningLabs:     runningCount,
		HibernatedLabs:  hibernatedCount,
		OOMKilledLabs:   oomCount,
		TotalRAMAllocMB: totalRAM,
		HealthStatus:    "healthy",
		HostHardware:    hostHardware,
	}

	if oomCount > 5 {
		resp.HealthStatus = "warning"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

type CourseLoadDTO struct {
	ID                 string `json:"id"`
	CourseName         string `json:"course_name"`
	TeacherName        string `json:"teacher_name"`
	ActiveStudents     int    `json:"active_students"`
	HibernatedStudents int    `json:"hibernated_students"`
	RAMUsedMB          int64  `json:"ram_used_mb"`
}

func (h *AdminHandler) GetCoursesLoad(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantIDFromContext(r.Context())
	if err != nil || tenantID == "" {
		http.Error(w, `{"error":"Tenant ID missing in context"}`, http.StatusUnauthorized)
		return
	}

	subjects, err := h.subjectRepo.ListByTenant(r.Context(), tenantID)
	if err != nil {
		subjects = []*domain.Subject{}
	}

	allWorkspaces, err := h.workspaceRepo.GetAllRunningWorkspaces(r.Context())
	if err != nil {
		allWorkspaces = []*domain.WorkspaceInstance{}
	}

	res := make([]CourseLoadDTO, 0, len(subjects))
	for _, sub := range subjects {
		activeCount := 0
		hibernatedCount := 0
		var ramTotalMB int64 = 0

		for _, ws := range allWorkspaces {
			if ws.SubjectID == sub.ID {
				if ws.Status == "running" {
					activeCount++
					ramTotalMB += ws.MemoryLimitMB
				} else if ws.Status == "hibernated" {
					hibernatedCount++
				}
			}
		}

		teacherName := "Cátedra Asignada"
		if sub.TeacherID != nil && *sub.TeacherID != "" {
			teacherID := *sub.TeacherID
			if len(teacherID) > 8 {
				teacherID = teacherID[:8]
			}
			teacherName = "Docente ID: " + teacherID
		}

		res = append(res, CourseLoadDTO{
			ID:                 sub.ID,
			CourseName:         sub.Name,
			TeacherName:        teacherName,
			ActiveStudents:     activeCount,
			HibernatedStudents: hibernatedCount,
			RAMUsedMB:          ramTotalMB,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

type TechnicalIncidentDTO struct {
	ID            string `json:"id"`
	Type          string `json:"type"`
	WorkspaceID   string `json:"workspace_id"`
	StudentID     string `json:"student_name"`
	CourseName    string `json:"course_name"`
	Description   string `json:"description"`
	MemoryLimitMB int64  `json:"memory_limit_mb"`
	Timestamp     string `json:"timestamp"`
}

func (h *AdminHandler) GetIncidents(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantIDFromContext(r.Context())
	if err != nil || tenantID == "" {
		http.Error(w, `{"error":"Tenant ID missing in context"}`, http.StatusUnauthorized)
		return
	}

	allWorkspaces, err := h.workspaceRepo.GetAllRunningWorkspaces(r.Context())
	if err != nil {
		allWorkspaces = []*domain.WorkspaceInstance{}
	}

	incidents := make([]TechnicalIncidentDTO, 0)
	for _, ws := range allWorkspaces {
		if ws.TenantID == tenantID && (ws.Status == "failed" || ws.Status == "oom_killed" || ws.OOMStrikeCount > 0) {
			desc := "Excedió cuota de memoria configurada (Exit code 137)"
			ts := ws.UpdatedAt.Format(time.RFC3339)
			if ws.LastOOMKilledAt != nil {
				ts = ws.LastOOMKilledAt.Format(time.RFC3339)
			}
			wsID := ws.ID
			if len(wsID) > 8 {
				wsID = wsID[:8]
			}
			incidents = append(incidents, TechnicalIncidentDTO{
				ID:            "inc-" + wsID,
				Type:          "oom_killed",
				WorkspaceID:   ws.ID,
				StudentID:     ws.StudentID,
				CourseName:    "Materia ID: " + ws.SubjectID,
				Description:   desc,
				MemoryLimitMB: ws.MemoryLimitMB,
				Timestamp:     ts,
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(incidents)
}
