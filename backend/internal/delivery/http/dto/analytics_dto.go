package dto

type DifficultyMetricDTO struct {
	Count       int     `json:"count"`
	SuccessRate float64 `json:"success_rate"`
}

type TagMetricDTO struct {
	Tag         string  `json:"tag"`
	Count       int     `json:"count"`
	SuccessRate float64 `json:"success_rate"`
}

type FailedCaseMetricDTO struct {
	ExerciseTitle string `json:"exercise_title"`
	CaseIndex     int    `json:"case_index"`
	FailCount     int    `json:"fail_count"`
}

type TimelineMetricDTO struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

type CourseAnalyticsResponseDTO struct {
	DifficultyDistribution         map[string]DifficultyMetricDTO `json:"difficulty_distribution"`
	TopTags                        []TagMetricDTO                 `json:"top_tags"`
	MostFailedCases                []FailedCaseMetricDTO          `json:"most_failed_cases"`
	AvgResolutionTimeByDifficulty map[string]int                 `json:"avg_resolution_time_by_difficulty"`
	SubmissionsTimeline            []TimelineMetricDTO            `json:"submissions_timeline"`
}
