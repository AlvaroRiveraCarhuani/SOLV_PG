package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"solv-backend/internal/core/domain"
)

// maxAdvanceSourceBytes acota lo que se lee del entorno del docente al emitir.
const maxAdvanceSourceBytes = 8 << 20

// EmitAdvanceInput agrupa una emisión vista desde el aula: quién emite, en qué
// sesión y a quién.
// EmitAdvanceInput agrupa una emisión vista desde el aula. No incluye la
// práctica: sale de la sesión, que es la única fuente de verdad, para que el
// snapshot y el avance registrado no puedan hablar de prácticas distintas.
type EmitAdvanceInput struct {
	TenantID   string
	TeacherID  string
	Role       string
	SessionID  string
	Scope      string
	StudentIDs []string
}

// Emit guarda el código actual del entorno de clase del docente como snapshot y
// emite el avance. El alumno recibe la versión emitida aunque el docente siga
// editando después; si la emisión falla, el objeto se borra.
func (s *ClassAdvanceService) Emit(ctx context.Context, in EmitAdvanceInput) (*domain.ClassAdvance, []string, error) {
	if strings.TrimSpace(in.TenantID) == "" {
		return nil, nil, domain.ErrInvalidTenant
	}
	if _, err := uuid.Parse(strings.TrimSpace(in.SessionID)); err != nil {
		return nil, nil, domain.ErrSessionNotFound
	}
	if s.snapshots == nil || s.environments == nil || s.volumes == nil {
		return nil, nil, ErrAdvanceStorageUnavailable
	}

	// La sesión se valida antes de empaquetar: emitir en examen o con el aula
	// cerrada no debe gastar una subida al almacén.
	session, err := s.sessions.GetSession(ctx, in.TenantID, teacherScope(in.Role, in.TeacherID), in.SessionID)
	if err != nil {
		return nil, nil, err
	}
	if !session.IsOpen() {
		return nil, nil, domain.ErrAdvanceSessionClosed
	}
	if session.FrozenAt != nil {
		return nil, nil, domain.ErrAdvanceFrozen
	}

	// La práctica la fija el aula. El snapshot se lee de ese entorno y el avance
	// se registra con el mismo identificador.
	exerciseID := ""
	if session.ExerciseID != nil {
		exerciseID = strings.TrimSpace(*session.ExerciseID)
	}
	if exerciseID == "" {
		return nil, nil, domain.ErrAdvancePracticeRequired
	}

	source, err := s.teacherEnvironment(ctx, in, session)
	if err != nil {
		return nil, nil, err
	}

	advanceID := uuid.NewString()
	key := fmt.Sprintf("class-advances/%s/%s.tar.gz", in.TenantID, advanceID)
	hash, size, files, err := s.snapshots.PutFiles(ctx, key, source)
	if err != nil {
		return nil, nil, err
	}

	advance, recipients, err := s.Issue(ctx, IssueAdvanceInput{
		ID:             advanceID,
		TenantID:       in.TenantID,
		TeacherID:      in.TeacherID,
		Role:           in.Role,
		SessionID:      in.SessionID,
		ExerciseID:     exerciseID,
		Scope:          in.Scope,
		StudentIDs:     in.StudentIDs,
		SnapshotKey:    key,
		SnapshotSHA256: hash,
		SnapshotSize:   size,
		FileCount:      files,
	})
	if err != nil {
		// Sin avance no hay razón para conservar el objeto.
		_ = s.snapshots.Delete(ctx, key)
		return nil, nil, err
	}
	return advance, recipients, nil
}

// teacherEnvironment lee el código del entorno de clase del docente. El avance
// sale de lo que el docente tiene abierto, no de una copia aparte, así que hace
// falta que ese entorno exista en la materia del aula.
func (s *ClassAdvanceService) teacherEnvironment(ctx context.Context, in EmitAdvanceInput, session *domain.ClassSession) (map[string]string, error) {
	teacherID := strings.TrimSpace(in.TeacherID)
	environment, err := s.teacherPracticeEnvironment(ctx, teacherID, session)
	if err != nil {
		return nil, err
	}
	if environment == nil {
		return nil, domain.ErrAdvanceSourceMissing
	}

	volumeName := environment.VolumeName()
	files, err := s.volumes.SnapshotPracticeEnvironmentVolume(ctx, volumeName, maxAdvanceSourceBytes)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrAdvanceSourceMissing, err)
	}
	if len(files) == 0 {
		return nil, domain.ErrAdvanceSourceMissing
	}

	source := make(map[string]string, len(files))
	for _, file := range files {
		source[file.Path] = string(file.Content)
	}
	return source, nil
}

// teacherPracticeEnvironment devuelve el entorno de clase del docente en la práctica de la
// sesión. No hay respaldo por materia: sin práctica no hay entorno que emitir.
func (s *ClassAdvanceService) teacherPracticeEnvironment(ctx context.Context, teacherID string, session *domain.ClassSession) (*domain.PracticeEnvironment, error) {
	if session.ExerciseID == nil {
		return nil, domain.ErrAdvancePracticeRequired
	}
	exerciseID := strings.TrimSpace(*session.ExerciseID)
	if exerciseID == "" {
		return nil, domain.ErrAdvancePracticeRequired
	}
	repo, ok := s.environments.(environmentExerciseRepository)
	if !ok {
		return nil, ErrAdvanceStorageUnavailable
	}
	environment, err := repo.GetByStudentSubjectExercise(ctx, teacherID, session.SubjectID, exerciseID)
	if err != nil {
		// Un entorno ausente no es un fallo de almacenamiento: el emisor informa
		// que no hay código que copiar.
		return nil, nil
	}
	return environment, nil
}
