package services

import (
	"context"
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
) (*domain.RubricEvaluationResponse, error) {
	var subMeta struct {
		ExerciseID uuid.UUID `db:"exercise_id"`
		TenantID   uuid.UUID `db:"tenant_id"`
	}
	err := s.db.GetContext(ctx, &subMeta, "SELECT exercise_id, tenant_id FROM submissions WHERE id = $1", submissionID)
	if err != nil {
		return nil, fmt.Errorf("submission not found: %w", err)
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
	_, err = s.db.ExecContext(ctx, `
		UPDATE submissions
		SET score = $1, verdict = 'AC', manual_override = TRUE, override_reason = 'Evaluado por rúbrica', graded_by = COALESCE($3, graded_by)
		WHERE id = $2
	`, roundedScoreInt, submissionID, authorID)
	if err != nil {
		return nil, fmt.Errorf("failed to update submission score: %w", err)
	}

	if strings.TrimSpace(req.Comments) != "" && authorID != nil {
		if _, err := s.db.ExecContext(ctx, `
			INSERT INTO submission_comments (tenant_id, submission_id, author_id, line_number, comment, created_at)
			VALUES ($1, $2, $3, 1, $4, NOW())
		`, subMeta.TenantID, submissionID, authorID, req.Comments); err != nil {
			return nil, fmt.Errorf("failed to insert rubric comment: %w", err)
		}
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
	if s.mailer == nil || s.db == nil {
		return
	}

	var meta struct {
		StudentEmail  string  `db:"student_email"`
		StudentName   string  `db:"student_name"`
		ExerciseTitle string  `db:"exercise_title"`
		SubjectName   *string `db:"subject_name"`
		SubjectID     *string `db:"subject_id"`
	}

	query := `
		SELECT 
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
	if err := s.db.GetContext(ctx, &meta, query, submissionID); err != nil || strings.TrimSpace(meta.StudentEmail) == "" {
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
