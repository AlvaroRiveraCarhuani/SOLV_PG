package services

import (
	"context"
	"strings"

	"solv-backend/internal/core/domain"
)

// assertActorWritesSubject aplica el alcance docente a una escritura sobre una
// materia. Un administrador conserva el alcance de su institución; un docente
// solo escribe donde dicta; y el id de una materia ajena responde "no existe"
// (notFound) en vez de confirmar que existe.
//
// Sin rol en el contexto se deja pasar: los llamadores internos (workers, jobs)
// ya resuelven su alcance por institución y no pasan por un actor humano.
func assertActorWritesSubject(
	ctx context.Context,
	reader domain.SubjectOwnershipReader,
	subjectID string,
	notFound error,
) error {
	teacherID, ok := domain.TeacherWriteScope(ctx)
	if !ok {
		return notFound
	}
	if teacherID == "" {
		return nil
	}
	if reader == nil || strings.TrimSpace(subjectID) == "" {
		// Sin forma de comprobar la pertenencia, un docente no escribe.
		return notFound
	}

	owns, err := reader.TeacherOwnsSubject(ctx, domain.GetTenantID(ctx), subjectID, teacherID)
	if err != nil {
		return err
	}
	if !owns {
		return notFound
	}
	return nil
}
