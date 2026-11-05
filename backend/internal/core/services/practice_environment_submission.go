package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/google/uuid"

	"solv-backend/internal/core/domain"
)

const (
	// maxProjectSnapshotBytes acota el snapshot entregado a SpeedGrader.
	maxProjectSnapshotBytes = 512 * 1024
)

var (
	// ErrPracticeEnvironmentForbidden indica que el environment no pertenece al alumno autenticado.
	ErrPracticeEnvironmentForbidden = errors.New("environment does not belong to the authenticated student")
	// ErrSubmissionNotConfigured indica que el servicio no tiene repositorio de submissions.
	ErrSubmissionNotConfigured = errors.New("submission persistence is not configured")
	// ErrPracticeEnvironmentVolumeUnavailable indica que el orquestador no puede operar el volumen.
	ErrPracticeEnvironmentVolumeUnavailable = errors.New("environment volume access is not available")
	// ErrInvalidExerciseID indica que el exercise_id recibido no es un UUID válido.
	ErrInvalidExerciseID = errors.New("invalid exercise_id")
)

// SubmitPracticeEnvironmentProject recolecta de forma determinista los archivos de código
// del volumen del environment y crea la submission que consume SpeedGrader.
func (s *PracticeEnvironmentService) SubmitPracticeEnvironmentProject(ctx context.Context, environmentID, studentID, exerciseID string, files map[string]string) (*domain.Submission, error) {
	environmentID = strings.TrimSpace(environmentID)
	exerciseID = strings.TrimSpace(exerciseID)
	if environmentID == "" || exerciseID == "" {
		return nil, fmt.Errorf("environment_id and exercise_id are required")
	}
	if _, err := uuid.Parse(exerciseID); err != nil {
		return nil, ErrInvalidExerciseID
	}
	if s.submissionRepo == nil {
		return nil, ErrSubmissionNotConfigured
	}

	environment, err := s.repo.GetByID(ctx, environmentID)
	if err != nil || environment == nil {
		// Un environment ausente es un 404 del transporte, no un fallo interno:
		// el repositorio ya distingue "no existe" de "id mal formado".
		return nil, fmt.Errorf("%w: environment %s", domain.ErrNotFound, environmentID)
	}
	if studentID != "" && environment.StudentID != studentID {
		return nil, ErrPracticeEnvironmentForbidden
	}

	// Una entrega no puede escribir en un volumen congelado: sería la forma de
	// seguir editando después del cierre (ADR-005 §5).
	if environment.FrozenAt != nil || environment.Status == domain.PracticeEnvironmentStatusFrozen {
		return nil, domain.ErrPracticeEnvironmentFrozen
	}

	tenantID := domain.GetTenantID(ctx)
	if tenantID == "" {
		tenantID = environment.TenantID
	}
	if tenantID == "" {
		tenantID = domain.DefaultTenantID
	}
	isExam := s.isExamExercise(ctx, exerciseID, tenantID, environment.SubjectID)

	volumeStore, ok := s.docker.(domain.PracticeEnvironmentVolumeStore)
	if !ok {
		return nil, ErrPracticeEnvironmentVolumeUnavailable
	}

	volumeName := environment.VolumeName()
	if len(files) > 0 {
		if err := volumeStore.WritePracticeEnvironmentVolumeFiles(ctx, volumeName, files); err != nil {
			return nil, fmt.Errorf("failed to write project files: %w", err)
		}
	}

	volumeFiles, err := volumeStore.SnapshotPracticeEnvironmentVolume(ctx, volumeName, maxProjectSnapshotBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to snapshot environment volume: %w", err)
	}

	snapshot := buildProjectSnapshot(volumeFiles)
	if strings.TrimSpace(snapshot) == "" {
		return nil, fmt.Errorf("environment %s has no code files to submit", environmentID)
	}

	submission := &domain.Submission{
		ID:            uuid.NewString(),
		TenantID:      tenantID,
		ExerciseID:    exerciseID,
		StudentID:     environment.StudentID,
		EnvironmentID: &environment.ID, Code: snapshot,
		// Un proyecto de IDE no lo evalúa el juez: no tiene veredicto. La
		// ausencia se guarda como ausencia y el trabajo queda en revisión.
		Verdict:            "",
		ReviewStatus:       string(domain.ReviewPending),
		ASTResult:          json.RawMessage("{}"),
		GeneratedCases:     json.RawMessage("{}"),
		ComplexityAnalysis: json.RawMessage("{}"),
	}

	if err := s.submissionRepo.Create(ctx, submission); err != nil {
		return nil, fmt.Errorf("failed to create project submission: %w", err)
	}

	// Entrega de examen: el alumno termina y su volumen queda en solo lectura.
	// La clase sigue abierta para los demás. Si el cierre falla, la entrega ya
	// quedó registrada y el barrido del aula lo reintenta.
	if isExam && s.examFreezer != nil {
		if freezeErr := s.examFreezer.FreezePracticeEnvironment(ctx, tenantID, environment.StudentID, environment.ID); freezeErr != nil {
			log.Printf("[Exam Freeze] la entrega %s quedó registrada pero no se pudo congelar el volumen %s: %v", submission.ID, environment.ID, freezeErr)
		}
	}

	return submission, nil
}

// isExamExercise dice si la entrega pertenece a un examen de la materia del
// laboratorio. Sin repositorio de ejercicios, o si no se puede leer, se trata
// como clase: un volumen nunca se congela por una duda.
func (s *PracticeEnvironmentService) isExamExercise(ctx context.Context, exerciseID, tenantID, subjectID string) bool {
	if s.exerciseRepo == nil || strings.TrimSpace(exerciseID) == "" {
		return false
	}
	exercise, err := s.exerciseRepo.GetByIDAndTenant(ctx, exerciseID, tenantID)
	if err != nil || exercise == nil {
		return false
	}
	if exercise.Purpose != string(domain.ExercisePurposeExam) {
		return false
	}
	// La entrega congela el volumen del laboratorio: el examen tiene que ser de
	// la misma materia para no cerrar un entorno ajeno.
	if exercise.SubjectID != nil && strings.TrimSpace(*exercise.SubjectID) != "" {
		return *exercise.SubjectID == subjectID
	}
	return true
}

// buildProjectSnapshot arma el snapshot estructurado con cabeceras
// "# === FILE: <ruta> ===" en orden alfabético por ruta relativa.
func buildProjectSnapshot(files []domain.PracticeEnvironmentVolumeFile) string {
	if len(files) == 0 {
		return ""
	}

	sorted := make([]domain.PracticeEnvironmentVolumeFile, len(files))
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
