package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strings"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/infrastructure/mailer"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type DefaultRubricService struct {
	rubricRepo   domain.RubricRepository
	exerciseRepo domain.ExerciseRepository
	db           *sqlx.DB
	mailer       domain.Mailer
	tenantRepo   domain.TenantRepository
	notifier     NotificationEmitter
}

func NewDefaultRubricService(
	rubricRepo domain.RubricRepository,
	exerciseRepo domain.ExerciseRepository,
	db *sqlx.DB,
) *DefaultRubricService {
	return &DefaultRubricService{
		rubricRepo:   rubricRepo,
		exerciseRepo: exerciseRepo,
		db:           db,
	}
}

func (s *DefaultRubricService) SetMailer(m domain.Mailer) {
	s.mailer = m
}

func (s *DefaultRubricService) SetTenantRepository(tr domain.TenantRepository) {
	s.tenantRepo = tr
}

// SetNotificationService publica el aviso in-app de calificación publicada,
// junto al correo que ya existía. Sin notifier, el correo sigue funcionando.
func (s *DefaultRubricService) SetNotificationService(notifier NotificationEmitter) {
	s.notifier = notifier
}

func (s *DefaultRubricService) GetByExerciseID(ctx context.Context, exerciseID uuid.UUID) (*domain.Rubric, error) {
	return s.rubricRepo.GetByExerciseID(ctx, exerciseID)
}

func (s *DefaultRubricService) CreateOrUpdate(ctx context.Context, rubric *domain.Rubric) error {
	exercise, err := s.exerciseRepo.GetByID(ctx, rubric.ExerciseID.String())
	if err != nil {
		return fmt.Errorf("failed to verify exercise: %w", err)
	}

	if exercise.EnvironmentType == "JUEZ_EFIMERO" {
		return domain.ErrRubricNotAllowedForJudge
	}

	for i := range rubric.Criteria {
		if rubric.Criteria[i].ID == "" {
			rubric.Criteria[i].ID = uuid.New().String()
		}
	}

	if err := rubric.Validate(); err != nil {
		return err
	}

	return s.rubricRepo.CreateOrUpdate(ctx, rubric)
}

func (s *DefaultRubricService) CalculateScore(
	criteria []domain.RubricCriterion,
	selections map[string]string,
) (float64, []domain.CriterionBreakdown, error) {
	if len(criteria) == 0 {
		return 0, nil, nil
	}

	var totalScore float64
	var breakdown []domain.CriterionBreakdown

	for _, criterion := range criteria {
		selectedLevelName, ok := selections[criterion.ID]
		if !ok {
			return 0, nil, fmt.Errorf("%w: criterion '%s'", domain.ErrRubricSelectionMissing, criterion.Name)
		}

		var matchedLevel *domain.RubricLevel
		for _, lvl := range criterion.Levels {
			if lvl.Name == selectedLevelName {
				matchedLevel = &lvl
				break
			}
		}

		if matchedLevel == nil {
			return 0, nil, fmt.Errorf("invalid level '%s' for criterion '%s'", selectedLevelName, criterion.Name)
		}

		weighted := (criterion.Weight / 100.0) * matchedLevel.Score
		totalScore += weighted

		breakdown = append(breakdown, domain.CriterionBreakdown{
			Criterion: criterion.Name,
			Level:     matchedLevel.Name,
			Score:     matchedLevel.Score,
			Weighted:  math.Round(weighted*100) / 100,
		})
	}

	finalScore := math.Round(totalScore*100) / 100
	return finalScore, breakdown, nil
}

func (s *DefaultRubricService) EvaluateSubmission(
	ctx context.Context,
	submissionID uuid.UUID,
	req domain.RubricEvaluationRequest,
	actor domain.RubricActor,
) (*domain.RubricEvaluationResponse, error) {
	var subMeta struct {
		ExerciseID uuid.UUID      `db:"exercise_id"`
		TenantID   uuid.UUID      `db:"tenant_id"`
		SubjectID  sql.NullString `db:"subject_id"`
	}
	// La entrega se busca dentro de la institución del docente y de una materia
	// que tenga asignada. Un administrador conserva el alcance de su tenant.
	tenantID := domain.GetTenantID(ctx)
	err := s.db.GetContext(ctx, &subMeta, `
		SELECT sub.exercise_id, sub.tenant_id, ex.subject_id::text AS subject_id
		FROM submissions sub
		JOIN exercises ex ON ex.id = sub.exercise_id
		LEFT JOIN subjects s ON s.id = ex.subject_id
		WHERE sub.id = $1
		  AND ($2 = '' OR sub.tenant_id::text = $2)
		  AND ($3 OR (s.teacher_id IS NOT NULL AND s.teacher_id::text = $4))
	`, submissionID, tenantID, actor.IsAdmin, actor.UserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: la entrega %s no existe en esta institución", domain.ErrNotFound, submissionID)
		}
		return nil, fmt.Errorf("failed to load submission: %w", err)
	}
	if !subMeta.SubjectID.Valid || strings.TrimSpace(subMeta.SubjectID.String) == "" {
		return nil, fmt.Errorf("la entrega %s no tiene materia para escribir la libreta", submissionID)
	}

	rubric, err := s.rubricRepo.GetByExerciseID(ctx, subMeta.ExerciseID)
	if err != nil {
		return nil, fmt.Errorf("rubric not found for exercise: %w", err)
	}

	score, breakdown, err := s.CalculateScore(rubric.Criteria, req.Selections)
	if err != nil {
		return nil, err
	}

	var authorID any
	if userID := strings.TrimSpace(domain.GetUserID(ctx)); userID != "" {
		authorID = userID
	}

	roundedScoreInt := int(math.Round(score))
	feedback := strings.TrimSpace(req.Comments)

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin rubric transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Una nota manual de otro docente no se reemplaza sin justificación. La
	// celda se bloquea para que dos docentes no la cambien a la vez, y el
	// motivo queda escrito cuando la recorrección la hace otra persona.
	justification := ""
	var existingCell struct {
		Source       string         `db:"source"`
		OverriddenBy sql.NullString `db:"overridden_by"`
		Score        sql.NullInt64  `db:"score"`
	}
	cellErr := tx.GetContext(ctx, &existingCell, `
		SELECT source, overridden_by::text, score
		FROM exercise_grades
		WHERE selected_submission_id = $1 AND tenant_id = $2
		FOR UPDATE
	`, submissionID, subMeta.TenantID)
	if cellErr != nil && !errors.Is(cellErr, sql.ErrNoRows) {
		return nil, fmt.Errorf("failed to load grade cell: %w", cellErr)
	}
	if cellErr == nil && existingCell.Source == "manual" && existingCell.OverriddenBy.Valid &&
		existingCell.OverriddenBy.String != actor.UserID &&
		(!existingCell.Score.Valid || int(existingCell.Score.Int64) != roundedScoreInt) {
		if len(feedback) < 10 {
			return nil, domain.ErrGradeReasonRequired
		}
		justification = feedback
	}

	// La rúbrica es una decisión académica del docente: escribe la nota y el
	// estado de revisión, nunca el veredicto. El veredicto pertenece al juez y
	// una entrega de IDE no tiene ninguno.
	result, err := tx.ExecContext(ctx, `
		UPDATE submissions
		SET score = $1,
		    graded_by = COALESCE($3, graded_by),
		    review_status = 'graded',
		    reviewed_at = NOW(),
		    reviewed_by = COALESCE($3, reviewed_by),
		    feedback_summary = $4
		WHERE id = $2
	`, roundedScoreInt, submissionID, authorID, feedback)
	if err != nil {
		return nil, fmt.Errorf("failed to update submission score: %w", err)
	}
	if affected, err := result.RowsAffected(); err == nil && affected == 0 {
		return nil, fmt.Errorf("%w: la entrega %s no existe en esta institución", domain.ErrNotFound, submissionID)
	}

	if feedback != "" && authorID != nil {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO submission_comments (tenant_id, submission_id, author_id, line_number, comment, created_at)
			VALUES ($1, $2, $3, 1, $4, NOW())
		`, subMeta.TenantID, submissionID, authorID, feedback); err != nil {
			return nil, fmt.Errorf("failed to insert rubric comment: %w", err)
		}
	}

	// La nota de la entrega y la celda de la libreta se escriben en la misma
	// transacción: si la libreta falla, la entrega no queda como calificada.
	// La celda declara procedencia humana: la nota la puso un docente con
	// criterios, no el sistema. La recorrección del mismo docente vuelve a
	// escribir la celda, para que corregir no quede bloqueado por sí mismo.
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO exercise_grades (
		    student_id, exercise_id, subject_id, tenant_id,
		    selected_submission_id, score, source, overridden_by, reason, updated_at
		)
		SELECT sub.student_id,
		       sub.exercise_id,
		       ex.subject_id,
		       sub.tenant_id,
		       sub.id,
		       $1,
		       'manual',
		       $4,
		       CASE WHEN $5 <> '' THEN $5 ELSE 'Evaluado por rúbrica' END,
		       NOW()
		FROM submissions sub
		JOIN exercises ex ON ex.id = sub.exercise_id
		WHERE sub.id = $2 AND sub.tenant_id = $3 AND ex.subject_id IS NOT NULL
		ON CONFLICT (student_id, exercise_id) DO UPDATE
		SET selected_submission_id = EXCLUDED.selected_submission_id,
		    score = EXCLUDED.score,
		    source = EXCLUDED.source,
		    overridden_by = EXCLUDED.overridden_by,
		    reason = EXCLUDED.reason,
		    updated_at = NOW()
		WHERE exercise_grades.source <> 'manual'
		   OR exercise_grades.selected_submission_id = EXCLUDED.selected_submission_id
	`, roundedScoreInt, submissionID, subMeta.TenantID, authorID, justification); err != nil {
		return nil, fmt.Errorf("failed to update gradebook: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit rubric evaluation: %w", err)
	}

	// Despacho asíncrono desacoplado de aviso al estudiante
	s.notifyRubricEvaluated(ctx, subMeta.TenantID.String(), submissionID.String(), subMeta.ExerciseID.String())

	return &domain.RubricEvaluationResponse{
		Score:     score,
		Breakdown: breakdown,
		Comments:  req.Comments,
	}, nil
}

func (s *DefaultRubricService) notifyRubricEvaluated(ctx context.Context, tenantID, submissionID, exerciseID string) {
	if s.db == nil {
		return
	}

	var meta struct {
		StudentID     string  `db:"student_id"`
		StudentEmail  string  `db:"student_email"`
		StudentName   string  `db:"student_name"`
		ExerciseTitle string  `db:"exercise_title"`
		SubjectName   *string `db:"subject_name"`
		SubjectID     *string `db:"subject_id"`
	}

	query := `
		SELECT 
			sub.student_id::text AS student_id,
			u.email AS student_email,
			TRIM(u.first_name || ' ' || u.last_name) AS student_name,
			e.title AS exercise_title,
			s.name AS subject_name,
			e.subject_id::text AS subject_id
		FROM submissions sub
		JOIN users u ON u.id = sub.student_id
		JOIN exercises e ON e.id = sub.exercise_id
		LEFT JOIN subjects s ON s.id = e.subject_id
		WHERE sub.id = $1
	`
	if err := s.db.GetContext(ctx, &meta, query, submissionID); err != nil {
		return
	}

	branding := domain.EmailBranding{
		InstitutionName: "Plataforma Académica",
	}
	if s.tenantRepo != nil {
		if tenant, err := s.tenantRepo.GetByID(ctx, tenantID); err == nil && tenant != nil {
			branding = tenant.ResolveBranding()
		}
	}

	subjectName := "Asignatura"
	if meta.SubjectName != nil && strings.TrimSpace(*meta.SubjectName) != "" {
		subjectName = strings.TrimSpace(*meta.SubjectName)
	}
	subjectID := ""
	if meta.SubjectID != nil {
		subjectID = *meta.SubjectID
	}

	domainName := strings.TrimSpace(branding.BaseDomain)
	if domainName == "" {
		domainName = "localhost:4200"
	}
	reviewURL := fmt.Sprintf("https://%s/student/courses/%s/assignments/%s", domainName, subjectID, exerciseID)

	if s.mailer != nil && strings.TrimSpace(meta.StudentEmail) != "" {
		htmlBody, textBody := mailer.BuildGradeFeedbackEmail(
			branding,
			meta.StudentName,
			meta.ExerciseTitle,
			subjectName,
			reviewURL,
		)

		s.mailer.SendAsync(&domain.EmailMessage{
			To:       meta.StudentEmail,
			Subject:  fmt.Sprintf("[%s] Evaluación publicada: %s — %s", branding.InstitutionName, meta.ExerciseTitle, subjectName),
			HTMLBody: htmlBody,
			TextBody: textBody,
		})
	}

	if s.notifier != nil && strings.TrimSpace(meta.StudentID) != "" {
		s.notifier.NotifyAsync(tenantID, domain.CreateNotificationDTO{
			RecipientUserID: meta.StudentID,
			Title:           fmt.Sprintf("Evaluación publicada: %s", meta.ExerciseTitle),
			Message:         fmt.Sprintf("%s ya tiene nota y retroalimentación disponibles.", subjectName),
			Severity:        domain.NotificationSeverityInfo,
			EventType:       "grade_published",
			Link:            fmt.Sprintf("/student/history/%s", submissionID),
		})
	}
}
