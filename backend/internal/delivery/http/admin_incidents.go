package httpdelivery

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/delivery/http/middleware"
)

type TechnicalIncidentDTO struct {
	ID            string `json:"id"`
	Type          string `json:"type"`
	WorkspaceID   string `json:"workspace_id"`
	StudentName   string `json:"student_name"`
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

	// Identidades y nombres reales: nada de UUIDs de estudiantes ni de materias.
	studentIDs := make([]string, 0)
	subjectIDs := make([]string, 0)
	for _, ws := range allWorkspaces {
		if ws.TenantID == tenantID && (ws.Status == "failed" || ws.Status == "oom_killed" || ws.OOMStrikeCount > 0) {
			studentIDs = append(studentIDs, ws.StudentID)
			subjectIDs = append(subjectIDs, ws.SubjectID)
		}
	}
	identities := h.studentIdentities(r.Context(), tenantID, studentIDs)
	courseNames := h.courseNames(r.Context(), tenantID, subjectIDs)

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
				StudentName:   identities[ws.StudentID].Name,
				CourseName:    courseNames[ws.SubjectID],
				Description:   desc,
				MemoryLimitMB: ws.MemoryLimitMB,
				Timestamp:     ts,
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(incidents)
}

type DockerContainerDTO struct {
	ID                  string `json:"id"`
	StudentName         string `json:"student_name"`
	StudentEmail        string `json:"student_email"`
	CourseName          string `json:"course_name"`
	ImageTag            string `json:"image_tag"`
	MemoryUsedMB        int64  `json:"memory_used_mb"`
	MetricsAvailable    bool   `json:"metrics_available"`
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

	// Identidades y nombres reales para el listado del clúster.
	studentIDs := make([]string, 0)
	subjectIDs := make([]string, 0)
	for _, ws := range allWorkspaces {
		if ws.TenantID == tenantID {
			studentIDs = append(studentIDs, ws.StudentID)
			subjectIDs = append(subjectIDs, ws.SubjectID)
		}
	}
	identities := h.studentIdentities(r.Context(), tenantID, studentIDs)
	courseNames := h.courseNames(r.Context(), tenantID, subjectIDs)

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
			memUsedMB, metricsAvailable := h.containerUsageMB(r.Context(), ws)
			containers = append(containers, DockerContainerDTO{
				ID:                  ws.ID,
				StudentName:         identities[ws.StudentID].Name,
				StudentEmail:        identities[ws.StudentID].Email,
				CourseName:          courseNames[ws.SubjectID],
				ImageTag:            "",
				MemoryUsedMB:        memUsedMB,
				MetricsAvailable:    metricsAvailable,
				MemoryLimitMB:       ws.MemoryLimitMB,
				TTLRemainingSeconds: ttl,
				Status:              status,
				StartedAt:           ws.CreatedAt.Format(time.RFC3339),
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(containers)
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
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
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
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
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
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":       "resolved",
		"workspace_id": workspaceID,
		"message":      "Incidencia técnica resuelta y contadores OOM normalizados.",
	})
}
