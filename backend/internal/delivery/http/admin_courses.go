package httpdelivery

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/delivery/http/middleware"
)

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

	// Identidades reales primero: el panel jamás expone UUIDs de estudiantes
	// ni correos sintéticos construidos con el UUID.
	studentIDs := make([]string, 0)
	for _, ws := range allWorkspaces {
		if ws.TenantID == tenantID && ws.SubjectID == courseID {
			studentIDs = append(studentIDs, ws.StudentID)
		}
	}
	identities := h.studentIdentities(r.Context(), tenantID, studentIDs)

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

			identity := identities[ws.StudentID]
			studentDisplay := identity.Name
			emailDisplay := identity.Email

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

			memUsedMB, metricsAvailable := h.containerUsageMB(r.Context(), ws)

			containers = append(containers, DockerContainerDTO{
				ID:                  ws.ID,
				StudentName:         studentDisplay,
				StudentEmail:        emailDisplay,
				CourseName:          courseName,
				ImageTag:            "",
				MemoryUsedMB:        memUsedMB,
				MetricsAvailable:    metricsAvailable,
				MemoryLimitMB:       ws.MemoryLimitMB,
				TTLRemainingSeconds: ttl,
				Status:              wsStatus,
				StartedAt:           ws.CreatedAt.Format(time.RFC3339),
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(containers)
}

type CourseLoadDTO struct {
	ID                 string `json:"id"`
	CourseName         string `json:"course_name"`
	TeacherName        string `json:"teacher_name"`
	ActiveStudents     int    `json:"active_students"`
	HibernatedStudents int    `json:"hibernated_students"`
	RAMUsedMB          int64  `json:"ram_used_mb"`
}

// studentProfile es la identidad visible de un estudiante en el panel
// admin: nombre real y correo real, jamás el UUID interno.
type studentProfile struct {
	Name  string
	Email string
}

// studentIdentities resuelve nombre y correo desde el directorio del tenant.
// Si el estudiante no está en el directorio (o no hay directorio), devuelve
// un rótulo neutro sin datos internos: nunca un UUID ni un correo sintético.
func (h *AdminHandler) studentIdentities(ctx context.Context, tenantID string, studentIDs []string) map[string]studentProfile {
	identities := make(map[string]studentProfile, len(studentIDs))
	for _, id := range studentIDs {
		identities[id] = studentProfile{Name: "Estudiante"}
	}
	if h.studentDir == nil {
		return identities
	}
	items, err := h.studentDir.ListStudentsDirectory(ctx, tenantID, "", "", "all", "all")
	if err != nil {
		return identities
	}
	for _, item := range items {
		name := strings.TrimSpace(item.FirstName + " " + item.LastName)
		if name == "" {
			continue
		}
		identities[item.ID] = studentProfile{Name: name, Email: strings.TrimSpace(item.Email)}
	}
	return identities
}

// containerUsageMB lee el consumo real del contenedor desde Docker. Sin
// métricas disponibles devuelve (0, false) para que la UI muestre
// «Sin métricas» en vez de un valor inventado.
func (h *AdminHandler) containerUsageMB(ctx context.Context, ws *domain.WorkspaceInstance) (int64, bool) {
	if h.orchestrator == nil || ws.ContainerID == nil || *ws.ContainerID == "" {
		return 0, false
	}
	metrics, err := h.orchestrator.GetContainerMetrics(ctx, *ws.ContainerID)
	if err != nil || metrics == nil || !metrics.IsRunning || metrics.MemoryUsageBytes <= 0 {
		return 0, false
	}
	return metrics.MemoryUsageBytes / (1024 * 1024), true
}

// courseNames resuelve nombres reales de materias para no exponer UUIDs.
func (h *AdminHandler) courseNames(ctx context.Context, tenantID string, subjectIDs []string) map[string]string {
	names := make(map[string]string, len(subjectIDs))
	for _, id := range subjectIDs {
		names[id] = "Materia"
	}
	if h.subjectRepo == nil {
		return names
	}
	for _, id := range subjectIDs {
		if sub, err := h.subjectRepo.GetByID(ctx, tenantID, id); err == nil && sub != nil {
			names[id] = sub.Name
		}
	}
	return names
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
				switch ws.Status {
				case "running":
					activeCount++
					ramTotalMB += ws.MemoryLimitMB
				case "hibernated":
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
	_ = json.NewEncoder(w).Encode(res)
}
