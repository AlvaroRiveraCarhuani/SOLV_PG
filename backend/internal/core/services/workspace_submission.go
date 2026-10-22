package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"

	"solv-backend/internal/core/domain"
)

const (
	// maxProjectSnapshotBytes acota el snapshot entregado a SpeedGrader.
	maxProjectSnapshotBytes = 512 * 1024

	projectSubmissionVerdict = "PENDING"
)

var (
	// ErrWorkspaceForbidden indica que el workspace no pertenece al alumno autenticado.
	ErrWorkspaceForbidden = errors.New("workspace does not belong to the authenticated student")
	// ErrSubmissionNotConfigured indica que el servicio no tiene repositorio de submissions.
	ErrSubmissionNotConfigured = errors.New("submission persistence is not configured")
	// ErrWorkspaceVolumeUnavailable indica que el orquestador no puede operar el volumen.
	ErrWorkspaceVolumeUnavailable = errors.New("workspace volume access is not available")
	// ErrInvalidExerciseID indica que el exercise_id recibido no es un UUID válido.
	ErrInvalidExerciseID = errors.New("invalid exercise_id")
)

// SubmitWorkspaceProject recolecta de forma determinista los archivos de código
// del volumen del workspace y crea la submission que consume SpeedGrader.
func (s *WorkspaceService) SubmitWorkspaceProject(ctx context.Context, workspaceID, studentID, exerciseID string, files map[string]string) (*domain.Submission, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	exerciseID = strings.TrimSpace(exerciseID)
	if workspaceID == "" || exerciseID == "" {
		return nil, fmt.Errorf("workspace_id and exercise_id are required")
	}
	if _, err := uuid.Parse(exerciseID); err != nil {
		return nil, ErrInvalidExerciseID
	}
	if s.submissionRepo == nil {
		return nil, ErrSubmissionNotConfigured
	}

	workspace, err := s.repo.GetByID(ctx, workspaceID)
	if err != nil || workspace == nil {
		return nil, fmt.Errorf("workspace %s not found", workspaceID)
	}
	if studentID != "" && workspace.StudentID != studentID {
		return nil, ErrWorkspaceForbidden
	}

	volumeStore, ok := s.docker.(domain.WorkspaceVolumeStore)
	if !ok {
		return nil, ErrWorkspaceVolumeUnavailable
	}

	volumeName := fmt.Sprintf("solv_workspace_%s_%s", workspace.StudentID, workspace.SubjectID)
	if len(files) > 0 {
		if err := volumeStore.WriteWorkspaceVolumeFiles(ctx, volumeName, files); err != nil {
			return nil, fmt.Errorf("failed to write project files: %w", err)
		}
	}

	volumeFiles, err := volumeStore.SnapshotWorkspaceVolume(ctx, volumeName, maxProjectSnapshotBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to snapshot workspace volume: %w", err)
	}

	snapshot := buildProjectSnapshot(volumeFiles)
	if strings.TrimSpace(snapshot) == "" {
		return nil, fmt.Errorf("workspace %s has no code files to submit", workspaceID)
	}

	tenantID := domain.GetTenantID(ctx)
	if tenantID == "" {
		tenantID = workspace.TenantID
	}
	if tenantID == "" {
		tenantID = domain.DefaultTenantID
	}

	zeroScore := 0
	submission := &domain.Submission{
		ID:                 uuid.NewString(),
		TenantID:           tenantID,
		ExerciseID:         exerciseID,
		StudentID:          workspace.StudentID,
		WorkspaceID:        &workspace.ID,
		Code:               snapshot,
		Verdict:            projectSubmissionVerdict,
		ASTResult:          json.RawMessage("{}"),
		GeneratedCases:     json.RawMessage("{}"),
		ComplexityAnalysis: json.RawMessage("{}"),
		Score:              &zeroScore,
	}

	if err := s.submissionRepo.Create(ctx, submission); err != nil {
		return nil, fmt.Errorf("failed to create project submission: %w", err)
	}

	return submission, nil
}

// buildProjectSnapshot arma el snapshot estructurado con cabeceras
// "# === FILE: <ruta> ===" en orden alfabético por ruta relativa.
func buildProjectSnapshot(files []domain.WorkspaceVolumeFile) string {
	if len(files) == 0 {
		return ""
	}

	sorted := make([]domain.WorkspaceVolumeFile, len(files))
	copy(sorted, files)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Path < sorted[j].Path })

	var builder strings.Builder
	for _, file := range sorted {
		content := string(file.Content)
		if builder.Len() > 0 {
			builder.WriteString("\n")
		}
		builder.WriteString(fmt.Sprintf("# === FILE: %s ===\n", file.Path))
		builder.WriteString(content)
		if !strings.HasSuffix(content, "\n") {
			builder.WriteString("\n")
		}
	}
	return builder.String()
}
