package domain

// WeakTag represents a topic tag where a student has low performance
type WeakTag struct {
	Tag         string  `json:"tag" db:"tag"`
	SuccessRate float64 `json:"success_rate" db:"success_rate"`
	Attempts    int     `json:"attempts" db:"attempts"`
}

// RecommendationItem represents a suggested exercise for reinforcement
type RecommendationItem struct {
	ExerciseID string `json:"exercise_id" db:"exercise_id"`
	Title      string `json:"title" db:"title"`
	Difficulty string `json:"difficulty" db:"difficulty"`
	MatchedTag string `json:"matched_tag" db:"matched_tag"`
	Reason     string `json:"reason" db:"reason"`
}

// StudentRecommendations holds the computed recommendation payload for a student in a course
type StudentRecommendations struct {
	HasEnoughData   bool                 `json:"has_enough_data"`
	WeakTags        []WeakTag            `json:"weak_tags"`
	Recommendations []RecommendationItem `json:"recommendations"`
	Message         string               `json:"message,omitempty"`
}
