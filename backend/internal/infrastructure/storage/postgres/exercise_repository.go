package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"solv-backend/internal/core/domain"
)

type PostgresExerciseRepository struct {
	db *sqlx.DB
}

func NewPostgresExerciseRepository(db *sqlx.DB) domain.ExerciseRepository {
	return &PostgresExerciseRepository{db: db}
}

func (r *PostgresExerciseRepository) GetByID(ctx context.Context, id string) (*domain.Exercise, error) {
	tenantID := domain.GetTenantID(ctx)
	return r.GetByIDAndTenant(ctx, id, tenantID)
}

func (r *PostgresExerciseRepository) GetByIDAndTenant(ctx context.Context, id, tenantID string) (*domain.Exercise, error) {
	var query string
	var args []interface{}
	if tenantID != "" {
		query = `
			SELECT id, subject_id, title, description, type, COALESCE(environment_type, 'JUEZ_EFIMERO') AS environment_type,
			       template_id, difficulty, 
			       COALESCE(tags, '{}') AS tags,
			       COALESCE(purpose, 'class') AS purpose,
			       COALESCE(per_student_seed, false) AS per_student_seed,
			       due_date, 
			       COALESCE(boilerplate, '') AS boilerplate, 
			       COALESCE(status, 'draft') AS status, 
			       COALESCE(language, 'python') AS language, 
			       COALESCE(time_limit_ms, 1000) AS time_limit_ms, 
			       COALESCE(memory_limit_mb, 128) AS memory_limit_mb, 
			       COALESCE(reference_solution, '') AS reference_solution,
			       COALESCE(stale, false) AS stale,
			       last_valid_dry_run_at, expected_complexity,
			       config, tenant_id, created_at
			FROM exercises
			WHERE id = $1 AND tenant_id = $2
		`
		args = []interface{}{id, tenantID}
	} else {
		query = `
			SELECT id, subject_id, title, description, type, COALESCE(environment_type, 'JUEZ_EFIMERO') AS environment_type,
			       template_id, difficulty, 
			       COALESCE(tags, '{}') AS tags,
			       COALESCE(purpose, 'class') AS purpose,
			       COALESCE(per_student_seed, false) AS per_student_seed,
			       due_date, 
			       COALESCE(boilerplate, '') AS boilerplate, 
			       COALESCE(status, 'draft') AS status, 
			       COALESCE(language, 'python') AS language, 
			       COALESCE(time_limit_ms, 1000) AS time_limit_ms, 
			       COALESCE(memory_limit_mb, 128) AS memory_limit_mb, 
			       COALESCE(reference_solution, '') AS reference_solution,
			       COALESCE(stale, false) AS stale,
			       last_valid_dry_run_at, expected_complexity,
			       config, tenant_id, created_at
			FROM exercises
			WHERE id = $1
		`
		args = []interface{}{id}
	}
	var exercise domain.Exercise
	err := r.db.GetContext(ctx, &exercise, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get exercise by id %s: %w", id, err)
	}

	// Cargar casos de prueba desde exercise_test_cases con aislamiento tenant
	var testCases []domain.TestCase
	if tenantID != "" {
		tcQuery := `
			SELECT tc.id, tc.exercise_id, tc.order_index, tc.input, tc.expected_output, tc.visibility, tc.weight, tc.created_at
			FROM exercise_test_cases tc
			JOIN exercises e ON e.id = tc.exercise_id
			WHERE tc.exercise_id = $1 AND e.tenant_id = $2
			ORDER BY tc.order_index ASC
		`
		_ = r.db.SelectContext(ctx, &testCases, tcQuery, id, tenantID)
	} else {
		tcQuery := `
			SELECT id, exercise_id, order_index, input, expected_output, visibility, weight, created_at
			FROM exercise_test_cases
			WHERE exercise_id = $1
			ORDER BY order_index ASC
		`
		_ = r.db.SelectContext(ctx, &testCases, tcQuery, id)
	}

	if len(testCases) > 0 {
		for i := range testCases {
			testCases[i].Normalize()
		}
		if exercise.Config.Algorithm == nil {
			exercise.Config.Algorithm = &domain.AlgorithmConfig{
				TimeLimitMS:   exercise.TimeLimitMS,
				MemoryLimitMB: exercise.MemoryLimitMB,
			}
		}
		exercise.Config.Algorithm.TestCases = testCases
	}

	return &exercise, nil
}

func (r *PostgresExerciseRepository) Create(ctx context.Context, exercise *domain.Exercise) error {
	tenantID := domain.GetTenantID(ctx)
	if exercise.TenantID == "" {
		exercise.TenantID = tenantID
	}
	if exercise.Status == "" {
		exercise.Status = "draft"
	}
	if exercise.Language == "" {
		exercise.Language = "python"
	}
	if exercise.TimeLimitMS == 0 {
		exercise.TimeLimitMS = 1000
	}
	if exercise.MemoryLimitMB == 0 {
		exercise.MemoryLimitMB = 128
	}
	if exercise.Purpose == "" {
		exercise.Purpose = string(domain.ExercisePurposeClass)
	}
	if exercise.Tags == nil {
		exercise.Tags = pq.StringArray{}
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback()

	if exercise.EnvironmentType == "" {
		exercise.EnvironmentType = string(domain.EnvironmentTypeJuezEfimero)
	}

	query := `
		INSERT INTO exercises (
			id, subject_id, title, description, type, environment_type, template_id, difficulty, tags, purpose, per_student_seed,
			due_date, boilerplate, status, language, time_limit_ms, memory_limit_mb,
			reference_solution, stale, expected_complexity, config, tenant_id
		)
		VALUES (
			:id, :subject_id, :title, :description, :type, :environment_type, :template_id, :difficulty, :tags, :purpose, :per_student_seed,
			:due_date, :boilerplate, :status, :language, :time_limit_ms, :memory_limit_mb,
			:reference_solution, :stale, :expected_complexity, :config, :tenant_id
		)
	`
	_, err = tx.NamedExecContext(ctx, query, exercise)
	if err != nil {
		return fmt.Errorf("failed to create exercise: %w", err)
	}

	if exercise.Config.Algorithm != nil && len(exercise.Config.Algorithm.TestCases) > 0 {
		for idx, tc := range exercise.Config.Algorithm.TestCases {
			tc.Normalize()
			if err := tc.Validate(); err != nil {
				return err
			}
			_, err = tx.ExecContext(ctx, `
				INSERT INTO exercise_test_cases (
					exercise_id, order_index, input, expected_output, visibility, weight
				) VALUES ($1, $2, $3, $4, $5, $6)
			`, exercise.ID, idx, tc.Input, tc.ExpectedOutput, string(tc.Visibility), tc.Weight)
			if err != nil {
				return fmt.Errorf("failed to insert test case %d: %w", idx, err)
			}
		}
	}

	return tx.Commit()
}

func (r *PostgresExerciseRepository) Update(ctx context.Context, exercise *domain.Exercise) error {
	tenantID := domain.GetTenantID(ctx)
	if exercise.TenantID == "" {
		exercise.TenantID = tenantID
	}
	if exercise.Purpose == "" {
		exercise.Purpose = string(domain.ExercisePurposeClass)
	}
	if exercise.Tags == nil {
		exercise.Tags = pq.StringArray{}
	}
	if exercise.EnvironmentType == "" {
		exercise.EnvironmentType = string(domain.EnvironmentTypeJuezEfimero)
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback()

	query := `
		UPDATE exercises
		SET title = :title,
		    description = :description,
		    subject_id = :subject_id,
		    type = :type,
		    environment_type = :environment_type,
		    template_id = :template_id,
		    difficulty = :difficulty,
		    tags = :tags,
		    purpose = :purpose,
		    per_student_seed = :per_student_seed,
		    due_date = :due_date,
		    boilerplate = :boilerplate,
		    language = :language,
		    time_limit_ms = :time_limit_ms,
		    memory_limit_mb = :memory_limit_mb,
		    reference_solution = :reference_solution,
		    stale = :stale,
		    expected_complexity = :expected_complexity,
		    config = :config
		WHERE id = :id AND tenant_id = :tenant_id
	`
	res, err := tx.NamedExecContext(ctx, query, exercise)
	if err != nil {
		return fmt.Errorf("failed to update exercise %s: %w", exercise.ID, err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("exercise %s not found in tenant", exercise.ID)
	}

	if exercise.Config.Algorithm != nil && len(exercise.Config.Algorithm.TestCases) > 0 {
		_, err = tx.ExecContext(ctx, `DELETE FROM exercise_test_cases WHERE exercise_id = $1`, exercise.ID)
		if err != nil {
			return fmt.Errorf("failed to delete old test cases for %s: %w", exercise.ID, err)
		}
		for idx, tc := range exercise.Config.Algorithm.TestCases {
			tc.Normalize()
			if err := tc.Validate(); err != nil {
				return err
			}
			_, err = tx.ExecContext(ctx, `
				INSERT INTO exercise_test_cases (
					exercise_id, order_index, input, expected_output, visibility, weight
				) VALUES ($1, $2, $3, $4, $5, $6)
			`, exercise.ID, idx, tc.Input, tc.ExpectedOutput, string(tc.Visibility), tc.Weight)
			if err != nil {
				return fmt.Errorf("failed to insert test case %d: %w", idx, err)
			}
		}
	}

	return tx.Commit()
}

func (r *PostgresExerciseRepository) UpdateStatus(ctx context.Context, id, tenantID, status string) error {
	query := `UPDATE exercises SET status = $1 WHERE id = $2 AND tenant_id = $3`
	res, err := r.db.ExecContext(ctx, query, status, id, tenantID)
	if err != nil {
		return fmt.Errorf("failed to update status for exercise %s: %w", id, err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("exercise %s not found in tenant", id)
	}
	return nil
}

func (r *PostgresExerciseRepository) UpdateConfig(ctx context.Context, id, tenantID string, config domain.ExerciseConfig) error {
	query := `UPDATE exercises SET config = $1, stale = TRUE WHERE id = $2 AND tenant_id = $3`
	res, err := r.db.ExecContext(ctx, query, config, id, tenantID)
	if err != nil {
		return fmt.Errorf("failed to update config for exercise %s: %w", id, err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("exercise %s not found in tenant", id)
	}
	return nil
}

func (r *PostgresExerciseRepository) UpdateExpectedJSON(ctx context.Context, id string, expectedJSON string) error {
	tenantID := domain.GetTenantID(ctx)
	query := `
		UPDATE exercises
		SET config = jsonb_set(config, '{database,expected_json}', to_jsonb($2::text))
		WHERE id = $1 AND tenant_id = $3
	`
	_, err := r.db.ExecContext(ctx, query, id, expectedJSON, tenantID)
	if err != nil {
		return fmt.Errorf("failed to update expected_json for exercise %s: %w", id, err)
	}
	return nil
}

func (r *PostgresExerciseRepository) MarkExerciseStale(ctx context.Context, exerciseID, tenantID string, stale bool) error {
	query := `UPDATE exercises SET stale = $1 WHERE id = $2 AND tenant_id = $3`
	_, err := r.db.ExecContext(ctx, query, stale, exerciseID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to mark exercise stale: %w", err)
	}
	return nil
}

func (r *PostgresExerciseRepository) UpdateExerciseLastValidDryRun(ctx context.Context, exerciseID, tenantID string, dryRunAt time.Time) error {
	query := `UPDATE exercises SET stale = FALSE, last_valid_dry_run_at = $1 WHERE id = $2 AND tenant_id = $3`
	_, err := r.db.ExecContext(ctx, query, dryRunAt, exerciseID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to update last valid dry run for exercise: %w", err)
	}
	return nil
}

type dryRunJobDB struct {
	ID              string          `db:"id"`
	ExerciseID      string          `db:"exercise_id"`
	Status          string          `db:"status"`
	ProgressCurrent int             `db:"progress_current"`
	ProgressTotal   int             `db:"progress_total"`
	Result          json.RawMessage `db:"result"`
	Error           string          `db:"error"`
	CreatedAt       time.Time       `db:"created_at"`
	UpdatedAt       time.Time       `db:"updated_at"`
}

func (r *PostgresExerciseRepository) CreateDryRunJob(ctx context.Context, job *domain.DryRunJob) error {
	resultBytes := []byte("{}")
	if job.Result != nil {
		if b, err := json.Marshal(job.Result); err == nil {
			resultBytes = b
		}
	}
	query := `
		INSERT INTO dry_run_jobs (id, exercise_id, status, progress_current, progress_total, result, error, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
	`
	_, err := r.db.ExecContext(ctx, query,
		job.ID,
		job.ExerciseID,
		string(job.Status),
		job.ProgressCurrent,
		job.ProgressTotal,
		resultBytes,
		job.Error,
	)
	if err != nil {
		return fmt.Errorf("failed to create dry_run_job: %w", err)
	}
	return nil
}

func (r *PostgresExerciseRepository) GetDryRunJob(ctx context.Context, jobID string) (*domain.DryRunJob, error) {
	query := `
		SELECT id, exercise_id, status, progress_current, progress_total, result, error, created_at, updated_at
		FROM dry_run_jobs
		WHERE id = $1
	`
	var dbJob dryRunJobDB
	err := r.db.GetContext(ctx, &dbJob, query, jobID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("dry_run_job not found")
		}
		return nil, fmt.Errorf("failed to get dry_run_job: %w", err)
	}

	job := &domain.DryRunJob{
		ID:              dbJob.ID,
		ExerciseID:      dbJob.ExerciseID,
		Status:          domain.DryRunJobStatus(dbJob.Status),
		ProgressCurrent: dbJob.ProgressCurrent,
		ProgressTotal:   dbJob.ProgressTotal,
		Error:           dbJob.Error,
		CreatedAt:       dbJob.CreatedAt,
		UpdatedAt:       dbJob.UpdatedAt,
	}

	if len(dbJob.Result) > 0 && string(dbJob.Result) != "{}" {
		var res domain.EvaluationResult
		if err := json.Unmarshal(dbJob.Result, &res); err == nil {
			job.Result = &res
		}
	}

	return job, nil
}

func (r *PostgresExerciseRepository) UpdateDryRunJobProgress(
	ctx context.Context,
	jobID string,
	status domain.DryRunJobStatus,
	current, total int,
	result *domain.EvaluationResult,
	errMsg string,
) error {
	resultBytes := []byte("{}")
	if result != nil {
		if b, err := json.Marshal(result); err == nil {
			resultBytes = b
		}
	}
	query := `
		UPDATE dry_run_jobs
		SET status = $1,
		    progress_current = $2,
		    progress_total = $3,
		    result = $4,
		    error = $5,
		    updated_at = NOW()
		WHERE id = $6
	`
	_, err := r.db.ExecContext(ctx, query,
		string(status),
		current,
		total,
		resultBytes,
		errMsg,
		jobID,
	)
	if err != nil {
		return fmt.Errorf("failed to update dry_run_job progress: %w", err)
	}
	return nil
}

func (r *PostgresExerciseRepository) ListDueByStudent(ctx context.Context, tenantID, studentID string) ([]*domain.DueAssignment, error) {
	query := `
		SELECT 
			e.id AS exercise_id,
			e.title,
			COALESCE(e.description, '') AS description,
			s.id AS subject_id,
			s.name AS subject_name,
			s.code AS subject_code,
			e.due_date,
			e.type
		FROM exercises e
		JOIN subjects s ON s.id = e.subject_id AND s.tenant_id = e.tenant_id
		JOIN enrollments en ON en.subject_id = s.id AND en.student_id = $2 AND en.tenant_id = $1
		WHERE e.tenant_id = $1
		  AND (e.due_date IS NULL OR e.due_date > NOW())
		ORDER BY e.due_date ASC NULLS LAST, e.created_at DESC
	`
	var assignments []*domain.DueAssignment
	err := r.db.SelectContext(ctx, &assignments, query, tenantID, studentID)
	if err != nil {
		return []*domain.DueAssignment{}, nil
	}
	if assignments == nil {
		assignments = []*domain.DueAssignment{}
	}
	return assignments, nil
}

func (r *PostgresExerciseRepository) ListBySubject(ctx context.Context, tenantID, subjectID string) ([]*domain.Exercise, error) {
	query := `
		SELECT id, subject_id, title, description, type, COALESCE(environment_type, 'JUEZ_EFIMERO') AS environment_type,
		       template_id, difficulty,
		       COALESCE(tags, '{}') AS tags,
		       COALESCE(purpose, 'class') AS purpose,
		       COALESCE(per_student_seed, false) AS per_student_seed,
		       due_date, 
		       COALESCE(boilerplate, '') AS boilerplate, 
		       COALESCE(status, 'draft') AS status, 
		       COALESCE(language, 'python') AS language, 
		       COALESCE(time_limit_ms, 1000) AS time_limit_ms, 
		       COALESCE(memory_limit_mb, 128) AS memory_limit_mb, 
		       COALESCE(reference_solution, '') AS reference_solution,
		       COALESCE(stale, false) AS stale,
		       last_valid_dry_run_at,
		       config, tenant_id, created_at
		FROM exercises
		WHERE tenant_id = $1 AND subject_id = $2
		ORDER BY created_at DESC
	`
	var exercises []*domain.Exercise
	err := r.db.SelectContext(ctx, &exercises, query, tenantID, subjectID)
	if err != nil {
		return []*domain.Exercise{}, nil
	}
	if exercises == nil {
		exercises = []*domain.Exercise{}
	}
	return exercises, nil
}

func (r *PostgresExerciseRepository) GetStudentRecommendations(ctx context.Context, tenantID, subjectID, studentID string) (*domain.StudentRecommendations, error) {
	result := &domain.StudentRecommendations{
		HasEnoughData:   false,
		WeakTags:        []domain.WeakTag{},
		Recommendations: []domain.RecommendationItem{},
	}

	// 1. Verificar si el estudiante tiene al menos 5 submissions en el curso
	countQuery := `
		SELECT COUNT(*)
		FROM submissions s
		JOIN exercises e ON e.id = s.exercise_id
		WHERE e.subject_id = $1 AND s.student_id = $2
		  AND ($3 = '' OR s.tenant_id = $3)
	`
	var totalSubmissions int
	if err := r.db.GetContext(ctx, &totalSubmissions, countQuery, subjectID, studentID, tenantID); err != nil {
		totalSubmissions = 0
	}

	if totalSubmissions < 5 {
		result.HasEnoughData = false
		result.Message = "Completa algunos ejercicios para recibir sugerencias personalizadas"
		return result, nil
	}

	result.HasEnoughData = true

	// 2. Obtener tags debiles (< 60% de exito y al menos 3 intentos)
	weakTagsQuery := `
		SELECT
			tag,
			COUNT(*)::int as attempts,
			AVG(CASE WHEN s.verdict = 'AC' OR s.score = 100 THEN 1.0 ELSE 0.0 END)::float8 as success_rate
		FROM exercises e
		CROSS JOIN LATERAL unnest(e.tags) as tag
		JOIN submissions s ON s.exercise_id = e.id
		WHERE e.subject_id = $1
		  AND s.student_id = $2
		  AND ($3 = '' OR s.tenant_id = $3)
		GROUP BY tag
		HAVING COUNT(*) >= 3
		   AND AVG(CASE WHEN s.verdict = 'AC' OR s.score = 100 THEN 1.0 ELSE 0.0 END) < 0.6
		ORDER BY success_rate ASC
		LIMIT 3
	`
	var weakTags []domain.WeakTag
	if err := r.db.SelectContext(ctx, &weakTags, weakTagsQuery, subjectID, studentID, tenantID); err != nil {
		weakTags = []domain.WeakTag{}
	}
	if weakTags == nil {
		weakTags = []domain.WeakTag{}
	}
	result.WeakTags = weakTags

	if len(weakTags) == 0 {
		result.Message = "¡Vas al día! No hay refuerzos sugeridos."
		return result, nil
	}

	// 3. Extraer los nombres de los tags debiles
	tagNames := make([]string, len(weakTags))
	for i, wt := range weakTags {
		tagNames[i] = wt.Tag
	}

	// 4. Buscar ejercicios sugeridos (no intentados por el alumno, publicados, del mismo curso, con tags afines)
	recQuery := `
		SELECT DISTINCT e.id, e.title, COALESCE(e.difficulty, 'easy') AS difficulty, e.tags
		FROM exercises e
		WHERE e.subject_id = $1
		  AND e.status = 'published'
		  AND ($2 = '' OR e.tenant_id = $2)
		  AND e.id NOT IN (
			  SELECT exercise_id FROM submissions WHERE student_id = $3
		  )
		  AND e.tags && $4
		ORDER BY
		  CASE COALESCE(e.difficulty, 'easy')
			WHEN 'easy' THEN 1
			WHEN 'medium' THEN 2
			WHEN 'hard' THEN 3
			ELSE 4
		  END, e.title ASC
		LIMIT 5
	`
	type recRow struct {
		ID         string         `db:"id"`
		Title      string         `db:"title"`
		Difficulty string         `db:"difficulty"`
		Tags       pq.StringArray `db:"tags"`
	}

	var rows []recRow
	if err := r.db.SelectContext(ctx, &rows, recQuery, subjectID, tenantID, studentID, pq.Array(tagNames)); err != nil {
		rows = []recRow{}
	}

	if len(rows) == 0 {
		result.Message = "¡Vas al día! No hay refuerzos sugeridos."
		return result, nil
	}

	weakTagSet := make(map[string]bool)
	for _, t := range tagNames {
		weakTagSet[t] = true
	}

	for _, row := range rows {
		matchedTag := ""
		for _, t := range row.Tags {
			if weakTagSet[t] {
				matchedTag = t
				break
			}
		}
		if matchedTag == "" && len(row.Tags) > 0 {
			matchedTag = row.Tags[0]
		}

		var reason string
		switch row.Difficulty {
		case "easy":
			reason = fmt.Sprintf("Practica %s con un ejercicio más sencillo", matchedTag)
		case "medium":
			reason = fmt.Sprintf("Refuerza %s con práctica intermedia", matchedTag)
		case "hard":
			reason = fmt.Sprintf("Ponte a prueba en %s con un ejercicio avanzado", matchedTag)
		default:
			reason = fmt.Sprintf("Practica %s para reforzar conceptos", matchedTag)
		}

		result.Recommendations = append(result.Recommendations, domain.RecommendationItem{
			ExerciseID: row.ID,
			Title:      row.Title,
			Difficulty: row.Difficulty,
			MatchedTag: matchedTag,
			Reason:     reason,
		})
	}

	return result, nil
}
