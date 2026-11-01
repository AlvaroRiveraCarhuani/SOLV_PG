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
type EmitAdvanceInput struct {
	TenantID   string
	TeacherID  string
	Role       string
	SessionID  string
	ExerciseID string
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
	if _, err := uuid.Parse(strings.TrimSpace(in.ExerciseID)); err != nil {
		return nil, nil, domain.ErrNotFound
	}
	if s.snapshots == nil || s.workspaces == nil || s.volumes == nil {
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

	source, err := s.teacherEnvironment(ctx, in, session.SubjectID)
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
		ExerciseID:     in.ExerciseID,
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
func (s *ClassAdvanceService) teacherEnvironment(ctx context.Context, in EmitAdvanceInput, subjectID string) (map[string]string, error) {
	teacherID := strings.TrimSpace(in.TeacherID)
	workspace, err := s.workspaces.GetByStudentAndSubject(ctx, teacherID, subjectID)
	if err != nil || workspace == nil {
		return nil, domain.ErrAdvanceSourceMissing
	}

	volumeName := fmt.Sprintf("solv_workspace_%s_%s", workspace.StudentID, workspace.SubjectID)
	files, err := s.volumes.SnapshotWorkspaceVolume(ctx, volumeName, maxAdvanceSourceBytes)
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
