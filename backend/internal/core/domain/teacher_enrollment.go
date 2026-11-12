package domain

import "context"

// AvailableStudentItem representa a un estudiante registrado en el tenant
// elegible para ser matriculado por el docente en una materia.
type AvailableStudentItem struct {
	ID        string `db:"id" json:"id"`
	FirstName string `db:"first_name" json:"first_name"`
	LastName  string `db:"last_name" json:"last_name"`
	Email     string `db:"email" json:"email"`
}

// TeacherEnrollmentStore agrupa las operaciones de búsqueda y matrícula de
// estudiantes por parte del docente en sus materias asignadas.
type TeacherEnrollmentStore interface {
	SearchAvailableStudentsForCourse(ctx context.Context, tenantID, teacherID, subjectID, query string, limit int) ([]*AvailableStudentItem, error)
	EnrollStudentInCourse(ctx context.Context, tenantID, teacherID, subjectID, studentID string) error
}
