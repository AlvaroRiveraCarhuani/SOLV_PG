package services

import (
	"context"
	"strings"

	"solv-backend/internal/core/domain"
)

// FindPracticeEnvironmentForExercise devuelve el entorno que la persona ya creó
// para una práctica, sin arrancar nada. Es la lectura que permite ofrecer la
// descarga del proyecto desde la primera pantalla, con el contenedor detenido,
// hibernado o corriendo. Que el entorno no exista no es un error: la pantalla
// explica que primero hay que abrirlo una vez.
func (s *PracticeEnvironmentService) FindPracticeEnvironmentForExercise(ctx context.Context, studentID, subjectID, exerciseID string) (*domain.PracticeEnvironment, error) {
	studentID = strings.TrimSpace(studentID)
	subjectID = strings.TrimSpace(subjectID)
	exerciseID = strings.TrimSpace(exerciseID)
	if studentID == "" || subjectID == "" || exerciseID == "" {
		return nil, nil
	}
	repo, ok := s.repo.(environmentExerciseRepository)
	if !ok {
		return nil, nil
	}
	// El repositorio filtra por tenant y por la terna completa: un id de otra
	// práctica o de otro alumno no devuelve nada en vez de un entorno ajeno.
	environment, err := repo.GetByStudentSubjectExercise(ctx, studentID, subjectID, exerciseID)
	if err != nil || environment == nil {
		return nil, nil
	}
	if environment.StudentID != studentID {
		return nil, nil
	}
	return environment, nil
}
