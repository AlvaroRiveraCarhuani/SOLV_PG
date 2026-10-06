package domain

import (
	"errors"
	"time"
)

var (
	ErrCyclicPrerequisite      = errors.New("CYCLIC_PREREQUISITE: no se pueden crear ciclos de prerrequisitos entre módulos")
	ErrPrerequisiteCrossCourse = errors.New("CROSS_COURSE_PREREQUISITE: el módulo prerrequisito debe pertenecer al mismo curso")
	ErrSelfPrerequisite        = errors.New("SELF_PREREQUISITE: un módulo no puede ser prerrequisito de sí mismo")
	ErrModuleNotFound          = errors.New("MODULE_NOT_FOUND: módulo no encontrado")
)

type ModuleState string

const (
	ModuleStateLocked     ModuleState = "locked"
	ModuleStateUnlocked   ModuleState = "unlocked"
	ModuleStateInProgress ModuleState = "in_progress"
	ModuleStateCompleted  ModuleState = "completed"
)

// CourseModule represents a curricular module within a subject/course
type CourseModule struct {
	ID          string    `json:"id" db:"id"`
	SubjectID   string    `json:"subject_id" db:"subject_id"`
	Title       string    `json:"title" db:"title"`
	Description string    `json:"description" db:"description"`
	OrderIndex  int       `json:"order_index" db:"order_index"`
	PassScore   int       `json:"pass_score" db:"pass_score"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// ModulePrerequisite defines a dependency between modules
type ModulePrerequisite struct {
	ModuleID             string `json:"module_id" db:"module_id"`
	PrerequisiteModuleID string `json:"prerequisite_module_id" db:"prerequisite_module_id"`
}

// CurricularExercise represents an exercise item within the curricular map
type CurricularExercise struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Difficulty  *string `json:"difficulty,omitempty"`
	Purpose     string  `json:"purpose"`
	BestScore   *int    `json:"best_score"`
	Attempts    int     `json:"attempts"`
	Submittable bool    `json:"submittable"`
}

// CurricularModuleMap represents a module node in the student's curricular map
type CurricularModuleMap struct {
	ID                    string               `json:"id"`
	Title                 string               `json:"title"`
	Description           string               `json:"description"`
	OrderIndex            int                  `json:"order_index"`
	PassScore             int                  `json:"pass_score"`
	State                 ModuleState          `json:"state"`
	LockReason            string               `json:"lock_reason,omitempty"`
	PrerequisiteModuleIDs []string             `json:"prerequisite_module_ids"`
	Exercises             []CurricularExercise `json:"exercises"`
}

// CourseCurricularMap represents the full map for a student
type CourseCurricularMap struct {
	CourseID            string                `json:"course_id"`
	Modules             []CurricularModuleMap `json:"modules"`
	UnassignedExercises []CurricularExercise  `json:"unassigned_exercises"`
}

// TeacherModuleDetails represents the module with prerequisites and exercise IDs for teacher view
type TeacherModuleDetails struct {
	CourseModule
	Prerequisites []string `json:"prerequisites"`
	ExerciseIDs   []string `json:"exercise_ids"`
}
