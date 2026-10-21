package postgres

import (
	"context"
	"fmt"

	"github.com/lib/pq"
	"solv-backend/internal/core/domain"
)

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
