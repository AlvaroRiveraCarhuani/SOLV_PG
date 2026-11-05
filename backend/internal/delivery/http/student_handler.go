package httpdelivery

import (
	"encoding/json"
	"math"
	"net/http"
	"time"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/delivery/http/middleware"
)

type StudentHandler struct {
	subjectRepo    domain.SubjectRepository
	environmentRepo  domain.PracticeEnvironmentRepository
	submissionRepo domain.SubmissionRepository
	exerciseRepo   domain.ExerciseRepository
	commentReader  domain.SubmissionCommentReader
}

func NewStudentHandler(
	subjectRepo domain.SubjectRepository,
	environmentRepo domain.PracticeEnvironmentRepository,
	submissionRepo domain.SubmissionRepository,
	exerciseRepo domain.ExerciseRepository,
	commentReaders ...domain.SubmissionCommentReader,
) *StudentHandler {
	handler := &StudentHandler{
		subjectRepo:    subjectRepo,
		environmentRepo:  environmentRepo,
		submissionRepo: submissionRepo,
		exerciseRepo:   exerciseRepo,
	}
	if len(commentReaders) > 0 {
		handler.commentReader = commentReaders[0]
	}
	return handler
}

// StudentSubjectItem resume una materia para el inicio del alumno. No lleva
// entorno: el entorno es de una práctica y el inicio no elige práctica; se
// abre desde la práctica o desde el aula (ADR-059).
type StudentSubjectItem struct {
	Subject            *domain.Subject `json:"subject"`
	TotalExercises     int             `json:"total_exercises"`
	CompletedExercises int             `json:"completed_exercises"`
	ProgressPercent    int             `json:"progress_percent"`
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

// studentIdentity resuelve tenant y usuario con la misma precedencia que
// GetDashboard: contexto de middleware, cabeceras y por ultimo el contexto
// de dominio.
func studentIdentity(r *http.Request) (string, string) {
	tenantID, err := middleware.GetTenantIDFromContext(r.Context())
	if err != nil || tenantID == "" {
		tenantID = r.Header.Get("X-Tenant-Id")
	}
	if tenantID == "" {
		if ctxTenantID, ok := r.Context().Value(domain.TenantIDKey).(string); ok && ctxTenantID != "" {
			tenantID = ctxTenantID
		}
	}

	userID := r.Header.Get("X-User-Id")
	if userID == "" {
		if ctxUserID, ok := r.Context().Value(domain.UserIDKey).(string); ok && ctxUserID != "" {
			userID = ctxUserID
		}
	}

	return tenantID, userID
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

		subjectItems = append(subjectItems, StudentSubjectItem{
			Subject:            subj,
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

// StudentEvaluationItem resume el estado academico de un ejercicio para el
// estudiante: intentos, mejor nota, ultima entrega y estado canonico.
type StudentEvaluationItem struct {
	ExerciseID      string     `json:"exercise_id"`
	Title           string     `json:"title"`
	Description     string     `json:"description"`
	Type            string     `json:"type"`
	EnvironmentType string     `json:"environment_type"`
	Purpose         string     `json:"purpose"`
	Difficulty      *string    `json:"difficulty,omitempty"`
	DueDate         *time.Time `json:"due_date,omitempty"`
	SubjectID       string     `json:"subject_id"`
	SubjectName     string     `json:"subject_name"`
	SubjectCode     string     `json:"subject_code"`
	AttemptsCount   int        `json:"attempts_count"`
	BestScore       *int       `json:"best_score,omitempty"`
	LastVerdict     string     `json:"last_verdict,omitempty"`
	// LastVerdictOrigin acompaña al último resultado: sin él, un proyecto
	// corregido con rúbrica se leería como una evaluación del juez.
	LastVerdictOrigin string   `json:"last_verdict_origin,omitempty"`
	LastSubmittedAt *time.Time `json:"last_submitted_at,omitempty"`
	AcademicStatus  string     `json:"academic_status"`
}

// StudentSubmissionHistoryItem enriquece una entrega con los datos del
// ejercicio y de la materia para el historial completo del estudiante.
type StudentSubmissionHistoryItem struct {
	*domain.Submission
	ExerciseTitle   string               `json:"exercise_title,omitempty"`
	ExerciseType    string               `json:"exercise_type,omitempty"`
	ExercisePurpose string               `json:"exercise_purpose,omitempty"`
	EnvironmentType string               `json:"environment_type,omitempty"`
	Language        string               `json:"language,omitempty"`
	Files           []domain.ProjectFile `json:"files,omitempty"`
	FilesAvailable  bool                 `json:"files_available"`
	SubjectID       string               `json:"subject_id,omitempty"`
	SubjectCode     string               `json:"subject_code,omitempty"`
	SubjectName     string               `json:"subject_name,omitempty"`
}

// submissionAggregation acumula las entregas de un alumno por ejercicio.
type submissionAggregation struct {
	AttemptsCount   int
	MaxScore        *int
	HasACUngraded   bool
	LastVerdict     string
	// LastVerdictOrigin viaja con LastVerdict: ambos son de la misma entrega.
	LastVerdictOrigin string
	LastSubmittedAt *time.Time
	IsCompleted     bool
}

const (
	academicStatusCompleted  = "completed"
	academicStatusOverdue    = "overdue"
	academicStatusInProgress = "in_progress"
	academicStatusPending    = "pending"
)

// GetEvaluations devuelve todos los ejercicios publicados de las materias
// inscritas, cada uno con su resumen de entregas y estado academico.
func (h *StudentHandler) GetEvaluations(w http.ResponseWriter, r *http.Request) {
	tenantID, userID := studentIdentity(r)
	if tenantID == "" {
		SendError(w, http.StatusUnauthorized, "Tenant ID missing in context", "Tenant no identificado")
		return
	}
	if userID == "" {
		SendError(w, http.StatusUnauthorized, "User ID missing in request", "Usuario no autenticado")
		return
	}

	var subjects []*domain.Subject
	if h.subjectRepo != nil {
		if listed, err := h.subjectRepo.ListByStudent(r.Context(), tenantID, userID); err == nil {
			subjects = listed
		}
	}

	var submissions []*domain.Submission
	if h.submissionRepo != nil {
		if listed, err := h.submissionRepo.ListByStudent(r.Context(), tenantID, userID); err == nil {
			submissions = listed
		}
	}

	aggregates := make(map[string]*submissionAggregation, len(submissions))
	for _, sub := range submissions {
		if sub == nil {
			continue
		}
		agg := aggregates[sub.ExerciseID]
		if agg == nil {
			agg = &submissionAggregation{}
			aggregates[sub.ExerciseID] = agg
		}

		agg.AttemptsCount++
		if sub.Score != nil {
			if agg.MaxScore == nil || *sub.Score > *agg.MaxScore {
				score := *sub.Score
				agg.MaxScore = &score
			}
		} else if sub.Verdict == "AC" {
			agg.HasACUngraded = true
		}
		if agg.LastSubmittedAt == nil || sub.SubmittedAt.After(*agg.LastSubmittedAt) {
			agg.LastVerdict = sub.Verdict
			agg.LastVerdictOrigin = sub.VerdictOrigin
			submittedAt := sub.SubmittedAt
			agg.LastSubmittedAt = &submittedAt
		}
		if sub.Verdict == "AC" || (sub.Score != nil && *sub.Score >= 51) {
			agg.IsCompleted = true
		}
	}

	items := make([]StudentEvaluationItem, 0)
	for _, subj := range subjects {
		if subj == nil {
			continue
		}

		var exercises []*domain.Exercise
		if h.exerciseRepo != nil {
			if listed, err := h.exerciseRepo.ListBySubject(r.Context(), tenantID, subj.ID); err == nil {
				exercises = listed
			}
		}

		for _, ex := range exercises {
			if ex == nil || (ex.Status != "" && ex.Status != "published") {
				continue
			}

			agg := aggregates[ex.ID]
			if agg == nil {
				agg = &submissionAggregation{}
			}

			var bestScore *int
			if agg.MaxScore != nil {
				score := *agg.MaxScore
				bestScore = &score
			}
			if agg.HasACUngraded && (bestScore == nil || *bestScore < 100) {
				score := 100
				bestScore = &score
			}

			items = append(items, StudentEvaluationItem{
				ExerciseID:      ex.ID,
				Title:           ex.Title,
				Description:     ex.Description,
				Type:            string(ex.Type),
				EnvironmentType: ex.EnvironmentType,
				Purpose:         ex.Purpose,
				Difficulty:      ex.Difficulty,
				DueDate:         ex.DueDate,
				SubjectID:       subj.ID,
				SubjectName:     subj.Name,
				SubjectCode:     subj.Code,
				AttemptsCount:   agg.AttemptsCount,
				BestScore:         bestScore,
				LastVerdict:       agg.LastVerdict,
				LastVerdictOrigin: agg.LastVerdictOrigin,
				LastSubmittedAt:   agg.LastSubmittedAt,
				AcademicStatus:  academicStatus(agg, ex.DueDate),
			})
		}
	}

	SendJSON(w, http.StatusOK, items, "Evaluaciones del estudiante obtenidas exitosamente")
}

// academicStatus aplica la precedencia cerrada: completado, vencido, en
// curso o pendiente.
func academicStatus(agg *submissionAggregation, dueDate *time.Time) string {
	if agg.IsCompleted {
		return academicStatusCompleted
	}
	if dueDate != nil && dueDate.Before(time.Now()) {
		return academicStatusOverdue
	}
	if agg.AttemptsCount > 0 {
		return academicStatusInProgress
	}
	return academicStatusPending
}

// GetSubmissions devuelve el historial completo de entregas del estudiante,
// sin truncar, enriquecido con ejercicio y materia.
func (h *StudentHandler) GetSubmissions(w http.ResponseWriter, r *http.Request) {
	tenantID, userID := studentIdentity(r)
	if tenantID == "" {
		SendError(w, http.StatusUnauthorized, "Tenant ID missing in context", "Tenant no identificado")
		return
	}
	if userID == "" {
		SendError(w, http.StatusUnauthorized, "User ID missing in request", "Usuario no autenticado")
		return
	}

	var submissions []*domain.Submission
	if h.submissionRepo != nil {
		if listed, err := h.submissionRepo.ListByStudent(r.Context(), tenantID, userID); err == nil {
			submissions = listed
		}
	}

	subjectByID := make(map[string]*domain.Subject)
	if h.subjectRepo != nil {
		if subjects, err := h.subjectRepo.ListByStudent(r.Context(), tenantID, userID); err == nil {
			for _, subj := range subjects {
				if subj != nil {
					subjectByID[subj.ID] = subj
				}
			}
		}
	}

	exerciseByID := make(map[string]*domain.Exercise)
	historyItems := make([]StudentSubmissionHistoryItem, 0, len(submissions))
	for _, sub := range submissions {
		if sub == nil {
			continue
		}

		item := StudentSubmissionHistoryItem{Submission: sub}

		exercise, cached := exerciseByID[sub.ExerciseID]
		if !cached && h.exerciseRepo != nil {
			if fetched, err := h.exerciseRepo.GetByIDAndTenant(r.Context(), sub.ExerciseID, tenantID); err == nil && fetched != nil {
				exercise = fetched
			}
			exerciseByID[sub.ExerciseID] = exercise
		}
		if exercise != nil {
			item.ExerciseTitle = exercise.Title
			item.ExerciseType = string(exercise.Type)
			item.ExercisePurpose = exercise.Purpose
			item.EnvironmentType = exercise.EnvironmentType
			item.Language = exercise.Language
			if item.EnvironmentType == "" {
				item.EnvironmentType = string(domain.EnvironmentTypeJuezEfimero)
			}
			if item.EnvironmentType == string(domain.EnvironmentTypeIDEPersistente) && sub.Code != "" {
				if files, ok := domain.ParseProjectSnapshot(sub.Code, exercise.Language); ok {
					item.Files = files
					item.FilesAvailable = true
				}
			}
			if exercise.SubjectID != nil {
				if subj, found := subjectByID[*exercise.SubjectID]; found && subj != nil {
					item.SubjectID = subj.ID
					item.SubjectCode = subj.Code
					item.SubjectName = subj.Name
				}
			}
		}

		historyItems = append(historyItems, item)
	}

	SendJSON(w, http.StatusOK, historyItems, "Historial de entregas obtenido exitosamente")
}

// GetSubmissionComments entrega al estudiante los comentarios de una entrega
// propia. La ruta docente no se reutiliza: aquí el dueño se valida contra la
// entrega y un identificador ajeno responde igual que uno inexistente.
func (h *StudentHandler) GetSubmissionComments(w http.ResponseWriter, r *http.Request) {
	tenantID, userID := studentIdentity(r)
	if tenantID == "" || userID == "" {
		SendError(w, http.StatusUnauthorized, "student_identity_missing", "No se pudo identificar al estudiante de la sesión")
		return
	}

	submissionID := r.PathValue("id")
	if submissionID == "" {
		SendError(w, http.StatusBadRequest, "missing_submission_id", "El identificador de la entrega es requerido")
		return
	}
	if h.submissionRepo == nil {
		SendError(w, http.StatusInternalServerError, "submission_repo_unavailable", "No se pudo consultar la entrega")
		return
	}

	submission, err := h.submissionRepo.GetByID(r.Context(), tenantID, submissionID)
	if err != nil || submission == nil || submission.StudentID != userID {
		SendError(w, http.StatusNotFound, "submission_not_found", "La entrega no existe o no pertenece al estudiante")
		return
	}

	comments := make([]*domain.SubmissionComment, 0)
	if h.commentReader != nil {
		listed, err := h.commentReader.GetCommentsBySubmission(r.Context(), tenantID, submissionID)
		if err != nil {
			SendError(w, http.StatusInternalServerError, "comments_unavailable", "No se pudieron obtener los comentarios")
			return
		}
		if listed != nil {
			comments = listed
		}
	}

	SendJSON(w, http.StatusOK, comments, "Comentarios de la entrega obtenidos exitosamente")
}
