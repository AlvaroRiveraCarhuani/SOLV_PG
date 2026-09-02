package httpdelivery

import (
	"encoding/json"
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
	Subject         *domain.Subject           `json:"subject"`
	ActiveWorkspace *domain.WorkspaceInstance `json:"active_workspace,omitempty"`
}

type StudentDashboardResponse struct {
	StudentID         string               `json:"student_id"`
	TenantID          string               `json:"tenant_id"`
	Subjects          []StudentSubjectItem `json:"subjects"`
	RecentSubmissions []*domain.Submission `json:"recent_submissions"`
}

func (h *StudentHandler) GetDashboard(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantIDFromContext(r.Context())
	if err != nil || tenantID == "" {
		SendError(w, http.StatusUnauthorized, "Tenant ID missing in context", "Tenant no identificado")
		return
	}

	userID := r.Header.Get("X-User-Id")
	if userID == "" {
		SendError(w, http.StatusUnauthorized, "User ID missing in request", "Usuario no autenticado")
		return
	}

	subjects, err := h.subjectRepo.ListByStudent(r.Context(), tenantID, userID)
	if err != nil {
		subjects = []*domain.Subject{}
	}

	subjectItems := make([]StudentSubjectItem, 0, len(subjects))
	for _, subj := range subjects {
		ws, _ := h.workspaceRepo.GetByStudentAndSubject(r.Context(), userID, subj.ID)
		subjectItems = append(subjectItems, StudentSubjectItem{
			Subject:         subj,
			ActiveWorkspace: ws,
		})
	}

	submissions, err := h.submissionRepo.ListByStudent(r.Context(), tenantID, userID)
	if err != nil {
		submissions = []*domain.Submission{}
	}

	resp := StudentDashboardResponse{
		StudentID:         userID,
		TenantID:          tenantID,
		Subjects:          subjectItems,
		RecentSubmissions: submissions,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *StudentHandler) GetDueAssignments(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantIDFromContext(r.Context())
	if err != nil || tenantID == "" {
		SendError(w, http.StatusUnauthorized, "Tenant ID missing in context", "Tenant no identificado")
		return
	}

	userID := r.Header.Get("X-User-Id")
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
