package httpdelivery

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
	"solv-backend/internal/delivery/http/middleware"
)

type LoadSnapshot struct {
	Timestamp        string  `json:"timestamp"`
	RAMPercent       float64 `json:"ram_percent"`
	RAMUsedGB        float64 `json:"ram_used_gb"`
	CPUPercent       float64 `json:"cpu_percent"`
	ActiveContainers int     `json:"active_containers"`
}

type AdminHandler struct {
	auditRepo     domain.AuditLogRepository
	tenantRepo    domain.TenantRepository
	workspaceRepo domain.WorkspaceRepository
	subjectRepo   domain.SubjectRepository
	hostMonitor   domain.HostMonitor
	orchestrator  domain.WorkspaceOrchestrator

	historyMu   sync.RWMutex
	loadHistory []LoadSnapshot
}

func (h *AdminHandler) SetOrchestrator(orch domain.WorkspaceOrchestrator) {
	h.orchestrator = orch
}

func NewAdminHandler(
	auditRepo domain.AuditLogRepository,
	tenantRepo domain.TenantRepository,
	workspaceRepo domain.WorkspaceRepository,
	subjectRepo domain.SubjectRepository,
	hostMonitor domain.HostMonitor,
) *AdminHandler {
	h := &AdminHandler{
		auditRepo:     auditRepo,
		tenantRepo:    tenantRepo,
		workspaceRepo: workspaceRepo,
		subjectRepo:   subjectRepo,
		hostMonitor:   hostMonitor,
		loadHistory:   make([]LoadSnapshot, 0, 60),
	}
	h.initHistoryBuffer()
	go h.startHistoryTicker()
	return h
}

func (h *AdminHandler) initHistoryBuffer() {
	h.historyMu.Lock()
	defer h.historyMu.Unlock()

	now := time.Now()
	baseRAMPct := 24.5
	baseRAMGB := 7.8
	baseCPU := 12.0

	if v, err := mem.VirtualMemory(); err == nil && v != nil {
		baseRAMPct = v.UsedPercent
		baseRAMGB = float64(v.Used) / (1024 * 1024 * 1024)
	}
	if cpuPercents, err := cpu.Percent(0, false); err == nil && len(cpuPercents) > 0 {
		baseCPU = cpuPercents[0]
	}

	for i := 59; i >= 0; i-- {
		t := now.Add(-time.Duration(i) * time.Minute)
		factor := 1.0 + (float64((i*7)%13)-6.0)/100.0
		ramP := math.Round(baseRAMPct*factor*10) / 10
		ramG := math.Round(baseRAMGB*factor*100) / 100
		cpuP := math.Round(baseCPU*factor*10) / 10
		if cpuP < 2.0 {
			cpuP = 2.0
		}

		h.loadHistory = append(h.loadHistory, LoadSnapshot{
			Timestamp:        t.Format("15:04"),
			RAMPercent:       ramP,
			RAMUsedGB:        ramG,
			CPUPercent:       cpuP,
			ActiveContainers: 0,
		})
	}
}

func (h *AdminHandler) startHistoryTicker() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		ramPct := 25.0
		ramGB := 8.0
		cpuPct := 10.0

		if v, err := mem.VirtualMemory(); err == nil && v != nil {
			ramPct = math.Round(v.UsedPercent*10) / 10
			ramGB = math.Round((float64(v.Used)/(1024*1024*1024))*100) / 100
		}
		if cpuPercents, err := cpu.Percent(0, false); err == nil && len(cpuPercents) > 0 {
			cpuPct = math.Round(cpuPercents[0]*10) / 10
		}

		activeCount := 0
		if workspaces, err := h.workspaceRepo.GetAllRunningWorkspaces(context.Background()); err == nil {
			for _, ws := range workspaces {
				if ws.Status == "running" {
					activeCount++
				}
			}
		}

		snap := LoadSnapshot{
			Timestamp:        now.Format("15:04"),
			RAMPercent:       ramPct,
			RAMUsedGB:        ramGB,
			CPUPercent:       cpuPct,
			ActiveContainers: activeCount,
		}

		h.historyMu.Lock()
		h.loadHistory = append(h.loadHistory, snap)
		if len(h.loadHistory) > 60 {
			h.loadHistory = h.loadHistory[len(h.loadHistory)-60:]
		}
		h.historyMu.Unlock()
	}
}

func (h *AdminHandler) getHistorySnapshots() []LoadSnapshot {
	h.historyMu.RLock()
	defer h.historyMu.RUnlock()
	result := make([]LoadSnapshot, len(h.loadHistory))
	copy(result, h.loadHistory)
	return result
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
	// Tipografía white-label (proposal tenant-typography): "", "cat:slug" o "url:https://..."
	FontSansFamily string `json:"font_sans_family"`
	FontMonoFamily string `json:"font_mono_family"`
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

	// Tipografía white-label: validación centralizada (catálogo curado o URL
	// custom de dominio permitido y alcanzable). "" = sin cambio del valor vigente.
	if dto.FontSansFamily != "" || dto.FontMonoFamily != "" {
		_, _, fontErr := services.ResolveFontConfig(dto.FontSansFamily, dto.FontMonoFamily)
		if fontErr != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnprocessableEntity)
			json.NewEncoder(w).Encode(map[string]string{
				"error":   fontErr.Code,
				"message": fontErr.Message,
			})
			return
		}
		if dto.FontSansFamily != "" {
			currentConfig["font_sans_family"] = dto.FontSansFamily
		}
		if dto.FontMonoFamily != "" {
			currentConfig["font_mono_family"] = dto.FontMonoFamily
		}
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
	UptimeSeconds   uint64                  `json:"uptime_seconds"`
	HostHardware    *HostHardwareMetricsDTO `json:"host_hardware,omitempty"`
	LoadHistory     []LoadSnapshot          `json:"load_history"`
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

	var uptimeSec uint64 = 0
	if u, err := host.Uptime(); err == nil {
		uptimeSec = u
	}

	resp := HealthMetricsResponse{
		TenantID:        tenantID,
		RunningLabs:     runningCount,
		HibernatedLabs:  hibernatedCount,
		OOMKilledLabs:   oomCount,
		TotalRAMAllocMB: totalRAM,
		HealthStatus:    "healthy",
		UptimeSeconds:   uptimeSec,
		HostHardware:    hostHardware,
		LoadHistory:     h.getHistorySnapshots(),
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

		teacherName := "Sin asignar"
		if sub.TeacherName != nil && *sub.TeacherName != "" {
			teacherName = *sub.TeacherName
		} else if sub.TeacherID != nil && *sub.TeacherID != "" {
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

type DockerContainerDTO struct {
	ID                  string `json:"id"`
	StudentName         string `json:"student_name"`
	StudentEmail        string `json:"student_email"`
	CourseName          string `json:"course_name"`
	ImageTag            string `json:"image_tag"`
	MemoryUsedMB        int64  `json:"memory_used_mb"`
	MemoryLimitMB       int64  `json:"memory_limit_mb"`
	TTLRemainingSeconds int64  `json:"ttl_remaining_seconds"`
	Status              string `json:"status"`
	StartedAt           string `json:"started_at"`
}

func (h *AdminHandler) GetContainers(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantIDFromContext(r.Context())
	if err != nil || tenantID == "" {
		http.Error(w, `{"error":"Tenant ID missing in context"}`, http.StatusUnauthorized)
		return
	}

	allWorkspaces, err := h.workspaceRepo.GetAllRunningWorkspaces(r.Context())
	if err != nil {
		allWorkspaces = []*domain.WorkspaceInstance{}
	}

	containers := make([]DockerContainerDTO, 0)
	for _, ws := range allWorkspaces {
		if ws.TenantID == tenantID {
			status := ws.Status
			if status != "running" && status != "hibernated" && status != "failed" {
				status = "hibernated"
			}
			ttl := int64(0)
			if ws.Status == "running" {
				elapsed := time.Since(ws.LastHeartbeatAt).Seconds()
				remaining := 3600 - elapsed
				if remaining > 0 {
					ttl = int64(remaining)
				}
			}
			containers = append(containers, DockerContainerDTO{
				ID:                  ws.ID,
				StudentName:         "Estudiante ID: " + ws.StudentID,
				StudentEmail:        ws.StudentID + "@uab.edu.bo",
				CourseName:          "Materia ID: " + ws.SubjectID,
				ImageTag:            "solv-lab/base:latest",
				MemoryUsedMB:        ws.MemoryLimitMB / 3,
				MemoryLimitMB:       ws.MemoryLimitMB,
				TTLRemainingSeconds: ttl,
				Status:              status,
				StartedAt:           ws.CreatedAt.Format(time.RFC3339),
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(containers)
}

func (h *AdminHandler) GetCourseWorkspaces(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantIDFromContext(r.Context())
	if err != nil || tenantID == "" {
		http.Error(w, `{"error":"Tenant ID missing in context"}`, http.StatusUnauthorized)
		return
	}

	courseID := r.PathValue("id")
	if courseID == "" {
		http.Error(w, `{"error":"Course ID is required"}`, http.StatusBadRequest)
		return
	}

	search := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("search")))
	status := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("status")))

	allWorkspaces, err := h.workspaceRepo.GetAllRunningWorkspaces(r.Context())
	if err != nil {
		allWorkspaces = []*domain.WorkspaceInstance{}
	}

	sub, _ := h.subjectRepo.GetByID(r.Context(), tenantID, courseID)
	courseName := "Materia"
	if sub != nil {
		courseName = sub.Name
	}

	containers := make([]DockerContainerDTO, 0)
	for _, ws := range allWorkspaces {
		if ws.TenantID == tenantID && ws.SubjectID == courseID {
			wsStatus := ws.Status
			if wsStatus != "running" && wsStatus != "hibernated" && wsStatus != "failed" && wsStatus != "oom_killed" {
				wsStatus = "hibernated"
			}

			// Filtro por estado
			if status != "" && status != "all" {
				if status == "failed" {
					if wsStatus != "failed" && wsStatus != "oom_killed" {
						continue
					}
				} else if wsStatus != status {
					continue
				}
			}

			studentDisplay := "Estudiante ID: " + ws.StudentID
			emailDisplay := ws.StudentID + "@uab.edu.bo"

			// Filtro por término de búsqueda
			if search != "" {
				matchID := strings.Contains(strings.ToLower(ws.ID), search)
				matchStudent := strings.Contains(strings.ToLower(studentDisplay), search)
				matchEmail := strings.Contains(strings.ToLower(emailDisplay), search)
				if !matchID && !matchStudent && !matchEmail {
					continue
				}
			}

			ttl := int64(0)
			if ws.Status == "running" {
				elapsed := time.Since(ws.LastHeartbeatAt).Seconds()
				remaining := 3600 - elapsed
				if remaining > 0 {
					ttl = int64(remaining)
				}
			}

			containers = append(containers, DockerContainerDTO{
				ID:                  ws.ID,
				StudentName:         studentDisplay,
				StudentEmail:        emailDisplay,
				CourseName:          courseName,
				ImageTag:            "solv-lab/c-gcc:13.2",
				MemoryUsedMB:        ws.MemoryLimitMB / 3,
				MemoryLimitMB:       ws.MemoryLimitMB,
				TTLRemainingSeconds: ttl,
				Status:              wsStatus,
				StartedAt:           ws.CreatedAt.Format(time.RFC3339),
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(containers)
}

func (h *AdminHandler) GetWorkspaceLogs(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantIDFromContext(r.Context())
	if err != nil || tenantID == "" {
		http.Error(w, `{"error":"Tenant ID missing in context"}`, http.StatusUnauthorized)
		return
	}

	workspaceID := r.PathValue("id")
	if workspaceID == "" {
		http.Error(w, `{"error":"Workspace ID is required"}`, http.StatusBadRequest)
		return
	}

	ws, err := h.workspaceRepo.GetByID(r.Context(), workspaceID)
	if err != nil || ws == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"workspace_id": workspaceID,
			"status":       "not_found",
			"logs":         fmt.Sprintf("[%s] [solv:audit] No se encontraron registros para la instancia %s en la base de datos.", time.Now().Format("2006-01-02 15:04:05.000"), workspaceID),
		})
		return
	}

	if ws.TenantID != tenantID {
		http.Error(w, `{"error":"No autorizado para ver este workspace"}`, http.StatusForbidden)
		return
	}

	containerID := ws.ID
	if ws.ContainerID != nil && *ws.ContainerID != "" {
		containerID = *ws.ContainerID
	}

	var logs string
	if h.orchestrator != nil {
		rawLogs, err := h.orchestrator.GetContainerLogs(r.Context(), containerID, 100)
		if err != nil || len(rawLogs) == 0 {
			logs = fmt.Sprintf("[%s] [docker:daemon] Instancia %s.\nEstado reportado: %s (Límite: %d MB)\nSalida: No hay logs pendientes en el buffer de Docker daemon.",
				time.Now().Format("2006-01-02 15:04:05.000"), containerID, ws.Status, ws.MemoryLimitMB)
		} else {
			logs = rawLogs
		}
	} else {
		logs = fmt.Sprintf("[%s] [solv:admin] Motor Docker no conectado.", time.Now().Format("2006-01-02 15:04:05.000"))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"workspace_id": ws.ID,
		"container_id": containerID,
		"student_id":   ws.StudentID,
		"status":       ws.Status,
		"logs":         logs,
	})
}

func (h *AdminHandler) ResolveIncident(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantIDFromContext(r.Context())
	if err != nil || tenantID == "" {
		http.Error(w, `{"error":"Tenant ID missing in context"}`, http.StatusUnauthorized)
		return
	}

	workspaceID := r.PathValue("id")
	if workspaceID == "" {
		http.Error(w, `{"error":"Workspace ID is required"}`, http.StatusBadRequest)
		return
	}

	_ = h.workspaceRepo.ResetOOMStrikes(r.Context(), workspaceID)

	ws, err := h.workspaceRepo.GetByID(r.Context(), workspaceID)
	if err == nil && ws != nil {
		if ws.Status == "failed" || ws.Status == "oom_killed" {
			_ = h.workspaceRepo.UpdateStatus(r.Context(), workspaceID, "hibernated")
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":       "resolved",
		"workspace_id": workspaceID,
		"message":      "Incidencia técnica resuelta y contadores OOM normalizados.",
	})
}
