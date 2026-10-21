package httpdelivery

import (
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
