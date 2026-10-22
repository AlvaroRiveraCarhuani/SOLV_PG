package httpdelivery

import (
	"encoding/json"
	"math"
	"net/http"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/delivery/http/middleware"
)

type StudentHandler struct {
	subjectRepo    domain.SubjectRepository
	workspaceRepo  domain.WorkspaceRepository
	submissionRepo domain.SubmissionRepository
	exerciseRepo   domain.ExerciseRepository
}

func NewStudentHandler(
	subjectRepo domain.SubjectRepository,
	workspaceRepo domain.WorkspaceRepository,
	submissionRepo domain.SubmissionRepository,
	exerciseRepo domain.ExerciseRepository,
) *StudentHandler {
	return &StudentHandler{
		subjectRepo:    subjectRepo,
		workspaceRepo:  workspaceRepo,
		submissionRepo: submissionRepo,
		exerciseRepo:   exerciseRepo,
	}
}

type StudentSubjectItem struct {
	Subject            *domain.Subject           `json:"subject"`
	ActiveWorkspace    *domain.WorkspaceInstance `json:"active_workspace,omitempty"`
	TotalExercises     int                       `json:"total_exercises"`
	CompletedExercises int                       `json:"completed_exercises"`
	ProgressPercent    int                       `json:"progress_percent"`
}

// StudentRecentSubmissionItem enriquece una entrega con el titulo del
// ejercicio y el codigo de la materia para que el dashboard no tenga que
// resolverlos por su cuenta.
type StudentRecentSubmissionItem struct {
	*domain.Submission
	ExerciseTitle string `json:"exercise_title,omitempty"`
	SubjectCode   string `json:"subject_code,omitempty"`
}

type StudentDashboardResponse struct {
	StudentID         string                        `json:"student_id"`
	TenantID          string                        `json:"tenant_id"`
	Subjects          []StudentSubjectItem          `json:"subjects"`
	RecentSubmissions []StudentRecentSubmissionItem `json:"recent_submissions"`
}

func (h *StudentHandler) GetDashboard(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantIDFromContext(r.Context())
	if err != nil || tenantID == "" {
		tenantID = r.Header.Get("X-Tenant-Id")
	}
	if tenantID == "" {
		if ctxTenantID, ok := r.Context().Value(domain.TenantIDKey).(string); ok && ctxTenantID != "" {
			tenantID = ctxTenantID
		}
	}
	if tenantID == "" {
		SendError(w, http.StatusUnauthorized, "Tenant ID missing in context", "Tenant no identificado")
		return
	}

	userID := r.Header.Get("X-User-Id")
	if userID == "" {
		if ctxUserID, ok := r.Context().Value(domain.UserIDKey).(string); ok && ctxUserID != "" {
			userID = ctxUserID
		}
	}
	if userID == "" {
		SendError(w, http.StatusUnauthorized, "User ID missing in request", "Usuario no autenticado")
		return
	}

	var subjects []*domain.Subject
	if h.subjectRepo != nil {
		subjects, err = h.subjectRepo.ListByStudent(r.Context(), tenantID, userID)
		if err != nil {
			subjects = []*domain.Subject{}
		}
	}

	var submissions []*domain.Submission
	if h.submissionRepo != nil {
		submissions, err = h.submissionRepo.ListByStudent(r.Context(), tenantID, userID)
		if err != nil {
			submissions = []*domain.Submission{}
		}
	}

	solvedExercises := make(map[string]bool, len(submissions))
	for _, sub := range submissions {
		if sub == nil {
			continue
		}
		if sub.Verdict == "AC" || (sub.Score != nil && *sub.Score >= 51) {
			solvedExercises[sub.ExerciseID] = true
		}
	}

	exerciseByID := make(map[string]*domain.Exercise)
	subjectByID := make(map[string]*domain.Subject, len(subjects))
	subjectItems := make([]StudentSubjectItem, 0, len(subjects))
	for _, subj := range subjects {
		if subj == nil {
			continue
		}
		subjectByID[subj.ID] = subj

		totalExercises := 0
		completedExercises := 0
		if h.exerciseRepo != nil {
			exercises, exErr := h.exerciseRepo.ListBySubject(r.Context(), tenantID, subj.ID)
			if exErr != nil {
				exercises = nil
			}
			for _, ex := range exercises {
				if ex == nil {
					continue
				}
				exerciseByID[ex.ID] = ex
				if ex.Status == "" || ex.Status == "published" {
					totalExercises++
					if solvedExercises[ex.ID] {
						completedExercises++
					}
				}
			}
		}

		progressPercent := 0
		if totalExercises > 0 {
			progressPercent = int(math.Round(float64(completedExercises) * 100.0 / float64(totalExercises)))
		}

		var ws *domain.WorkspaceInstance
		if h.workspaceRepo != nil {
			ws, _ = h.workspaceRepo.GetByStudentAndSubject(r.Context(), userID, subj.ID)
		}
		subjectItems = append(subjectItems, StudentSubjectItem{
			Subject:            subj,
			ActiveWorkspace:    ws,
			TotalExercises:     totalExercises,
			CompletedExercises: completedExercises,
			ProgressPercent:    progressPercent,
		})
	}

	const recentLimit = 5
	if len(submissions) > recentLimit {
		submissions = submissions[:recentLimit]
	}
	recentSubmissions := make([]StudentRecentSubmissionItem, 0, len(submissions))
	for _, sub := range submissions {
		if sub == nil {
			continue
		}
		item := StudentRecentSubmissionItem{Submission: sub}
		exercise, ok := exerciseByID[sub.ExerciseID]
		if !ok && h.exerciseRepo != nil {
			if fetched, fetchErr := h.exerciseRepo.GetByIDAndTenant(r.Context(), sub.ExerciseID, tenantID); fetchErr == nil && fetched != nil {
				exercise = fetched
				exerciseByID[fetched.ID] = fetched
			}
		}
		if exercise != nil {
			item.ExerciseTitle = exercise.Title
			if exercise.SubjectID != nil {
				if subject, found := subjectByID[*exercise.SubjectID]; found && subject != nil {
					item.SubjectCode = subject.Code
				}
			}
		}
		recentSubmissions = append(recentSubmissions, item)
	}

	resp := StudentDashboardResponse{
		StudentID:         userID,
		TenantID:          tenantID,
		Subjects:          subjectItems,
		RecentSubmissions: recentSubmissions,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *StudentHandler) GetDueAssignments(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantIDFromContext(r.Context())
	if err != nil || tenantID == "" {
		tenantID = r.Header.Get("X-Tenant-Id")
	}
	if tenantID == "" {
		if ctxTenantID, ok := r.Context().Value(domain.TenantIDKey).(string); ok && ctxTenantID != "" {
			tenantID = ctxTenantID
		}
	}
	if tenantID == "" {
		SendError(w, http.StatusUnauthorized, "Tenant ID missing in context", "Tenant no identificado")
		return
	}

	userID := r.Header.Get("X-User-Id")
	if userID == "" {
		if ctxUserID, ok := r.Context().Value(domain.UserIDKey).(string); ok && ctxUserID != "" {
			userID = ctxUserID
		}
	}
	if userID == "" {
		SendError(w, http.StatusUnauthorized, "User ID missing in request", "Usuario no autenticado")
		return
	}

	var assignments []*domain.DueAssignment
	if h.exerciseRepo != nil {
		assignments, err = h.exerciseRepo.ListDueByStudent(r.Context(), tenantID, userID)
		if err != nil {
			assignments = []*domain.DueAssignment{}
		}
	} else {
		assignments = []*domain.DueAssignment{}
	}

	SendJSON(w, http.StatusOK, assignments, "Entregas pendientes obtenidas exitosamente")
}

func (h *StudentHandler) GetRecommendations(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantIDFromContext(r.Context())
	if err != nil || tenantID == "" {
		tenantID = r.Header.Get("X-Tenant-Id")
	}
	if tenantID == "" {
		if ctxTenantID, ok := r.Context().Value(domain.TenantIDKey).(string); ok && ctxTenantID != "" {
			tenantID = ctxTenantID
		}
	}
	if tenantID == "" {
		SendError(w, http.StatusUnauthorized, "Tenant ID missing in context", "Tenant no identificado")
		return
	}

	userID := r.Header.Get("X-User-Id")
	if userID == "" {
		if ctxUserID, ok := r.Context().Value(domain.UserIDKey).(string); ok && ctxUserID != "" {
			userID = ctxUserID
		}
	}
	if userID == "" {
		SendError(w, http.StatusUnauthorized, "User ID missing in request", "Usuario no autenticado")
		return
	}

	courseID := r.PathValue("id")
	if courseID == "" {
		courseID = r.PathValue("courseId")
	}
	if courseID == "" {
		SendError(w, http.StatusBadRequest, "Course ID is required", "ID del curso requerido")
		return
	}

	if h.exerciseRepo == nil {
		SendJSON(w, http.StatusOK, &domain.StudentRecommendations{
			HasEnoughData:   false,
			WeakTags:        []domain.WeakTag{},
			Recommendations: []domain.RecommendationItem{},
			Message:         "Servicio de recomendaciones no disponible",
		}, "Recomendaciones obtenidas")
		return
	}

	recommendations, err := h.exerciseRepo.GetStudentRecommendations(r.Context(), tenantID, courseID, userID)
	if err != nil {
		SendError(w, http.StatusInternalServerError, "Failed to fetch student recommendations", "Error al obtener recomendaciones de refuerzo")
		return
	}

	SendJSON(w, http.StatusOK, recommendations, "Recomendaciones obtenidas exitosamente")
}
