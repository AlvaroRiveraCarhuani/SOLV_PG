package domain

// DifficultyMetric consolida el total de ejercicios y la tasa de aprobacion por nivel de dificultad.
type DifficultyMetric struct {
	Count       int     `json:"count"`
	SuccessRate float64 `json:"success_rate"`
}

// TagMetric consolida el total de ejercicios asociados y la tasa de aprobacion por etiqueta tematica.
type TagMetric struct {
	Tag         string  `json:"tag"`
	Count       int     `json:"count"`
	SuccessRate float64 `json:"success_rate"`
}

// FailedCaseMetric representa un caso de prueba con alto indice de fallos en el curso.
type FailedCaseMetric struct {
	ExerciseTitle string `json:"exercise_title"`
	CaseIndex     int    `json:"case_index"`
	FailCount     int    `json:"fail_count"`
}

// TimelineMetric representa el volumen de envios por fecha.
type TimelineMetric struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

// CourseAnalytics reune las metricas academicas agregadas a nivel de curso/materia.
type CourseAnalytics struct {
	DifficultyDistribution         map[string]DifficultyMetric `json:"difficulty_distribution"`
	TopTags                        []TagMetric                 `json:"top_tags"`
	MostFailedCases                []FailedCaseMetric          `json:"most_failed_cases"`
	AvgResolutionTimeByDifficulty map[string]int              `json:"avg_resolution_time_by_difficulty"`
	SubmissionsTimeline            []TimelineMetric            `json:"submissions_timeline"`
}
