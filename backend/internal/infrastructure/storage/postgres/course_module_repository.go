package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"solv-backend/internal/core/domain"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type PostgresCourseModuleRepository struct {
	db *sqlx.DB
}

func NewPostgresCourseModuleRepository(db *sqlx.DB) *PostgresCourseModuleRepository {
	return &PostgresCourseModuleRepository{db: db}
}

// tenantIDFromContext resuelve la institución del contexto. Los módulos no tienen
// tenant_id propio: cuelgan de una materia, y la materia sí lo tiene. Toda
// consulta filtra por ese camino, porque un id de módulo ajeno alcanzaba para
// leerlo, editarlo o borrarlo desde otra institución.
func tenantIDFromContext(ctx context.Context) string {
	tenantID := domain.GetTenantID(ctx)
	if tenantID == "" {
		return domain.DefaultTenantID
	}
	return tenantID
}

func (r *PostgresCourseModuleRepository) CreateModule(ctx context.Context, module *domain.CourseModule) error {
	if module.ID == "" {
		module.ID = uuid.New().String()
	}
	if module.PassScore == 0 {
		module.PassScore = 60
	}

	var subjectInTenant bool
	if err := r.db.GetContext(ctx, &subjectInTenant,
		`SELECT EXISTS (SELECT 1 FROM subjects WHERE id = $1 AND tenant_id = $2)`,
		module.SubjectID, tenantIDFromContext(ctx),
	); err != nil {
		return fmt.Errorf("failed to check subject for module: %w", err)
	}
	if !subjectInTenant {
		return fmt.Errorf("la materia del módulo no pertenece a esta institución: %w", domain.ErrNotFound)
	}

	query := `
		INSERT INTO course_modules (id, subject_id, title, description, order_index, pass_score, created_at)
		VALUES (:id, :subject_id, :title, :description, :order_index, :pass_score, NOW())
	`
	_, err := r.db.NamedExecContext(ctx, query, module)
	if err != nil {
		return fmt.Errorf("failed to create course module: %w", err)
	}
	return nil
}

func (r *PostgresCourseModuleRepository) GetModuleByID(ctx context.Context, moduleID string) (*domain.CourseModule, error) {
	var module domain.CourseModule
	query := `
		SELECT cm.id, cm.subject_id, cm.title, cm.description, cm.order_index, cm.pass_score, cm.created_at
		FROM course_modules cm
		JOIN subjects s ON s.id = cm.subject_id
		WHERE cm.id = $1 AND s.tenant_id = $2
	`
	err := r.db.GetContext(ctx, &module, query, moduleID, tenantIDFromContext(ctx))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrModuleNotFound
		}
		return nil, fmt.Errorf("failed to get course module: %w", err)
	}
	return &module, nil
}

func (r *PostgresCourseModuleRepository) ListModulesBySubject(ctx context.Context, subjectID string) ([]*domain.CourseModule, error) {
	return r.listModulesBySubject(ctx, tenantIDFromContext(ctx), subjectID)
}

func (r *PostgresCourseModuleRepository) listModulesBySubject(ctx context.Context, tenantID, subjectID string) ([]*domain.CourseModule, error) {
	query := `
		SELECT cm.id, cm.subject_id, cm.title, cm.description, cm.order_index, cm.pass_score, cm.created_at
		FROM course_modules cm
		JOIN subjects s ON s.id = cm.subject_id
		WHERE cm.subject_id = $1 AND s.tenant_id = $2
		ORDER BY cm.order_index ASC, cm.created_at ASC
	`
	var modules []*domain.CourseModule
	err := r.db.SelectContext(ctx, &modules, query, subjectID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list modules for subject: %w", err)
	}
	if modules == nil {
		modules = []*domain.CourseModule{}
	}
	return modules, nil
}

func (r *PostgresCourseModuleRepository) UpdateModule(ctx context.Context, module *domain.CourseModule) error {
	query := `
		UPDATE course_modules cm
		SET title = :title, description = :description, order_index = :order_index, pass_score = :pass_score
		FROM subjects s
		WHERE s.id = cm.subject_id AND cm.id = :id AND s.tenant_id = :tenant_id
	`
	res, err := r.db.NamedExecContext(ctx, query, map[string]interface{}{
		"id":          module.ID,
		"title":       module.Title,
		"description": module.Description,
		"order_index": module.OrderIndex,
		"pass_score":  module.PassScore,
		"tenant_id":   tenantIDFromContext(ctx),
	})
	if err != nil {
		return fmt.Errorf("failed to update course module: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return domain.ErrModuleNotFound
	}
	return nil
}

func (r *PostgresCourseModuleRepository) DeleteModule(ctx context.Context, moduleID string) error {
	query := `
		DELETE FROM course_modules
		WHERE id = $1
		  AND subject_id IN (SELECT id FROM subjects WHERE tenant_id = $2)
	`
	res, err := r.db.ExecContext(ctx, query, moduleID, tenantIDFromContext(ctx))
	if err != nil {
		return fmt.Errorf("failed to delete course module: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return domain.ErrModuleNotFound
	}
	return nil
}

func (r *PostgresCourseModuleRepository) SetPrerequisites(ctx context.Context, moduleID string, prerequisiteModuleIDs []string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	tenantID := tenantIDFromContext(ctx)
	var moduleInTenant bool
	if err := tx.GetContext(ctx, &moduleInTenant, `
		SELECT EXISTS (
			SELECT 1 FROM course_modules cm
			JOIN subjects s ON s.id = cm.subject_id
			WHERE cm.id = $1 AND s.tenant_id = $2
		)`, moduleID, tenantID); err != nil {
		return fmt.Errorf("failed to check module for prerequisites: %w", err)
	}
	if !moduleInTenant {
		return domain.ErrModuleNotFound
	}

	// 1. Eliminar prerrequisitos actuales
	if _, err := tx.ExecContext(ctx, "DELETE FROM module_prerequisites WHERE module_id = $1", moduleID); err != nil {
		return fmt.Errorf("failed to delete old prerequisites: %w", err)
	}

	// 2. Insertar nuevos prerrequisitos
	insertQuery := `
		INSERT INTO module_prerequisites (module_id, prerequisite_module_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`
	for _, prereqID := range prerequisiteModuleIDs {
		if prereqID == moduleID {
			return domain.ErrSelfPrerequisite
		}
		var prereqInTenant bool
		if err := tx.GetContext(ctx, &prereqInTenant, `
			SELECT EXISTS (
				SELECT 1 FROM course_modules cm
				JOIN subjects s ON s.id = cm.subject_id
				WHERE cm.id = $1 AND s.tenant_id = $2
			)`, prereqID, tenantID); err != nil {
			return fmt.Errorf("failed to check prerequisite module: %w", err)
		}
		if !prereqInTenant {
			return domain.ErrModuleNotFound
		}
		if _, err := tx.ExecContext(ctx, insertQuery, moduleID, prereqID); err != nil {
			return fmt.Errorf("failed to insert prerequisite: %w", err)
		}
	}

	return tx.Commit()
}

func (r *PostgresCourseModuleRepository) GetPrerequisitesBySubject(ctx context.Context, subjectID string) ([]domain.ModulePrerequisite, error) {
	return r.listPrerequisitesBySubject(ctx, tenantIDFromContext(ctx), subjectID)
}

func (r *PostgresCourseModuleRepository) listPrerequisitesBySubject(ctx context.Context, tenantID, subjectID string) ([]domain.ModulePrerequisite, error) {
	query := `
		SELECT mp.module_id, mp.prerequisite_module_id
		FROM module_prerequisites mp
		JOIN course_modules cm ON cm.id = mp.module_id
		JOIN subjects s ON s.id = cm.subject_id
		WHERE cm.subject_id = $1 AND s.tenant_id = $2
	`
	var prereqs []domain.ModulePrerequisite
	err := r.db.SelectContext(ctx, &prereqs, query, subjectID, tenantIDFromContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to get prerequisites for subject: %w", err)
	}
	if prereqs == nil {
		prereqs = []domain.ModulePrerequisite{}
	}
	return prereqs, nil
}

func (r *PostgresCourseModuleRepository) GetPrerequisitesForModule(ctx context.Context, moduleID string) ([]string, error) {
	query := `
		SELECT mp.prerequisite_module_id
		FROM module_prerequisites mp
		JOIN course_modules cm ON cm.id = mp.module_id
		JOIN subjects s ON s.id = cm.subject_id
		WHERE mp.module_id = $1 AND s.tenant_id = $2
	`
	var prereqIDs []string
	err := r.db.SelectContext(ctx, &prereqIDs, query, moduleID, tenantIDFromContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to get prerequisites for module: %w", err)
	}
	if prereqIDs == nil {
		prereqIDs = []string{}
	}
	return prereqIDs, nil
}

func (r *PostgresCourseModuleRepository) AssignExerciseModule(ctx context.Context, exerciseID string, moduleID *string) error {
	tenantID := tenantIDFromContext(ctx)

	// Un ejercicio no puede apuntar a un módulo de otra institución: el módulo
	// desaparecería del mapa curricular y el dato quedaría cruzado.
	if moduleID != nil && *moduleID != "" {
		var moduleInTenant bool
		if err := r.db.GetContext(ctx, &moduleInTenant, `
			SELECT EXISTS (
				SELECT 1 FROM course_modules cm
				JOIN subjects s ON s.id = cm.subject_id
				WHERE cm.id = $1 AND s.tenant_id = $2
			)`, *moduleID, tenantID); err != nil {
			return fmt.Errorf("failed to check module for exercise: %w", err)
		}
		if !moduleInTenant {
			return domain.ErrModuleNotFound
		}
	}

	query := `UPDATE exercises SET module_id = $1 WHERE id = $2 AND tenant_id = $3`
	res, err := r.db.ExecContext(ctx, query, moduleID, exerciseID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to assign module to exercise: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return domain.ErrNotFound
	}
	return nil
}

type exerciseRow struct {
	ID              string     `db:"id"`
	ModuleID        *string    `db:"module_id"`
	Title           string     `db:"title"`
	Difficulty      *string    `db:"difficulty"`
	Purpose         string     `db:"purpose"`
	EnvironmentType string     `db:"environment_type"`
	DueDate         *time.Time `db:"due_date"`
}

type studentSubStat struct {
	ExerciseID string `db:"exercise_id"`
	BestScore  *int   `db:"best_score"`
	Attempts   int    `db:"attempts"`
}

func (r *PostgresCourseModuleRepository) GetCourseCurricularMapData(ctx context.Context, tenantID, subjectID, studentID string) (*domain.CourseCurricularMap, error) {
	// 1. Obtener todos los módulos del curso. Se usa el tenantID explícito de
	// este método, no el del contexto: el mapa puede pedirse desde un job.
	modules, err := r.listModulesBySubject(ctx, tenantID, subjectID)
	if err != nil {
		return nil, err
	}

	// 2. Obtener todos los prerrequisitos del curso
	prereqs, err := r.listPrerequisitesBySubject(ctx, tenantID, subjectID)
	if err != nil {
		return nil, err
	}

	prereqMap := make(map[string][]string) // module_id -> [prereq_id, ...]
	for _, p := range prereqs {
		prereqMap[p.ModuleID] = append(prereqMap[p.ModuleID], p.PrerequisiteModuleID)
	}

	// 3. Obtener ejercicios publicados del curso
	var exercises []exerciseRow
	exQuery := `
		SELECT id, module_id, title, difficulty, COALESCE(purpose, 'class') AS purpose,
		       COALESCE(environment_type, 'JUEZ_EFIMERO') AS environment_type,
		       due_date
		FROM exercises
		WHERE subject_id = $1 AND status = 'published'
		ORDER BY created_at ASC
	`
	if err := r.db.SelectContext(ctx, &exercises, exQuery, subjectID); err != nil {
		exercises = []exerciseRow{}
	}

	// 4. Obtener estadísticas de submissions del estudiante para este curso
	var stats []studentSubStat
	statQuery := `
		SELECT s.exercise_id, 
		       MAX(COALESCE(s.score, CASE WHEN s.verdict = 'AC' THEN 100 ELSE 0 END))::int as best_score,
		       COUNT(*)::int as attempts
		FROM submissions s
		JOIN exercises e ON e.id = s.exercise_id
		WHERE e.subject_id = $1 AND s.student_id = $2
		GROUP BY s.exercise_id
	`
	if studentID != "" {
		if err := r.db.SelectContext(ctx, &stats, statQuery, subjectID, studentID); err != nil {
			stats = []studentSubStat{}
		}
	}
	statMap := make(map[string]studentSubStat)
	for _, st := range stats {
		statMap[st.ExerciseID] = st
	}

	// 5. Agrupar ejercicios por module_id
	moduleExercisesMap := make(map[string][]domain.CurricularExercise)
	var unassignedExercises []domain.CurricularExercise

	for _, ex := range exercises {
		st := statMap[ex.ID]
		curEx := domain.CurricularExercise{
			ID:              ex.ID,
			Title:           ex.Title,
			Difficulty:      ex.Difficulty,
			Purpose:         ex.Purpose,
			EnvironmentType: ex.EnvironmentType,
			DueDate:         ex.DueDate,
			BestScore:       st.BestScore,
			Attempts:        st.Attempts,
			Submittable:     true, // Default temporal, se ajustará con el estado del módulo
		}
		if ex.ModuleID != nil && *ex.ModuleID != "" {
			moduleExercisesMap[*ex.ModuleID] = append(moduleExercisesMap[*ex.ModuleID], curEx)
		} else {
			unassignedExercises = append(unassignedExercises, curEx)
		}
	}

	// 6. Calcular si cada módulo está COMPLETADO
	// Un módulo está completado si todos sus ejercicios publicados tienen best_score >= pass_score
	// Módulo sin ejercicios publicados cuenta como completado.
	moduleTitleMap := make(map[string]string)
	isModuleCompletedMap := make(map[string]bool)

	for _, m := range modules {
		moduleTitleMap[m.ID] = m.Title
		mExs := moduleExercisesMap[m.ID]
		if len(mExs) == 0 {
			isModuleCompletedMap[m.ID] = true
			continue
		}

		allPassed := true
		for _, ex := range mExs {
			if ex.BestScore == nil || *ex.BestScore < m.PassScore {
				allPassed = false
				break
			}
		}
		isModuleCompletedMap[m.ID] = allPassed
	}

	// 7. Construir mapa de módulos con estados calculados
	curricularModules := make([]domain.CurricularModuleMap, 0, len(modules))

	for _, m := range modules {
		prereqIDs := prereqMap[m.ID]
		if prereqIDs == nil {
			prereqIDs = []string{}
		}

		// Determinar si está desbloqueado
		isUnlocked := true
		var lockReason string

		for _, pID := range prereqIDs {
			if !isModuleCompletedMap[pID] {
				isUnlocked = false
				pTitle := moduleTitleMap[pID]
				if pTitle == "" {
					pTitle = "módulo previo"
				}
				lockReason = fmt.Sprintf("Completa %s para desbloquear", pTitle)
				break
			}
		}

		mExs := moduleExercisesMap[m.ID]
		if mExs == nil {
			mExs = []domain.CurricularExercise{}
		}

		var state domain.ModuleState
		if !isUnlocked {
			state = domain.ModuleStateLocked
		} else if isModuleCompletedMap[m.ID] {
			state = domain.ModuleStateCompleted
		} else {
			hasAttempts := false
			for _, ex := range mExs {
				if ex.Attempts > 0 {
					hasAttempts = true
					break
				}
			}
			if hasAttempts {
				state = domain.ModuleStateInProgress
			} else {
				state = domain.ModuleStateUnlocked
			}
		}

		// Ajustar submittable para los ejercicios del módulo
		for i := range mExs {
			if state == domain.ModuleStateLocked && mExs[i].Purpose != string(domain.ExercisePurposeExam) {
				mExs[i].Submittable = false
			} else {
				mExs[i].Submittable = true
			}
		}

		curricularModules = append(curricularModules, domain.CurricularModuleMap{
			ID:                    m.ID,
			Title:                 m.Title,
			Description:           m.Description,
			OrderIndex:            m.OrderIndex,
			PassScore:             m.PassScore,
			State:                 state,
			LockReason:            lockReason,
			PrerequisiteModuleIDs: prereqIDs,
			Exercises:             mExs,
		})
	}

	if unassignedExercises == nil {
		unassignedExercises = []domain.CurricularExercise{}
	}

	return &domain.CourseCurricularMap{
		CourseID:            subjectID,
		Modules:             curricularModules,
		UnassignedExercises: unassignedExercises,
	}, nil
}

func (r *PostgresCourseModuleRepository) IsModuleLockedForStudent(ctx context.Context, tenantID, moduleID, studentID string) (bool, error) {
	if moduleID == "" {
		return false, nil
	}

	module, err := r.GetModuleByID(ctx, moduleID)
	if err != nil {
		return false, err
	}

	curMap, err := r.GetCourseCurricularMapData(ctx, tenantID, module.SubjectID, studentID)
	if err != nil {
		return false, err
	}

	for _, m := range curMap.Modules {
		if m.ID == moduleID {
			return m.State == domain.ModuleStateLocked, nil
		}
	}

	return false, nil
}
