package services

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"solv-backend/internal/core/domain"
)

// strictVolumeSnapshotter es una capacidad opcional del orquestador: la lectura
// que falla en vez de recortar. Con ella la entrega rechaza un proyecto que no
// entra en el tope; sin ella se queda con lo que entra, como antes.
type strictVolumeSnapshotter interface {
	SnapshotPracticeEnvironmentVolumeStrict(ctx context.Context, volumeName string, maxBytes int64) ([]domain.PracticeEnvironmentVolumeFile, error)
}

func snapshotForSubmission(ctx context.Context, store domain.PracticeEnvironmentVolumeStore, volumeName string, maxBytes int64) ([]domain.PracticeEnvironmentVolumeFile, error) {
	if strict, ok := store.(strictVolumeSnapshotter); ok {
		return strict.SnapshotPracticeEnvironmentVolumeStrict(ctx, volumeName, maxBytes)
	}
	return store.SnapshotPracticeEnvironmentVolume(ctx, volumeName, maxBytes)
}

// volumeUsageMeasurer es una capacidad opcional del orquestador: medir el
// volumen sin depender de que el adaptador de pruebas la implemente.
type volumeUsageMeasurer interface {
	MeasurePracticeEnvironmentVolumeMB(ctx context.Context, volumeName string) (int64, error)
}

// annotateDiskLimit deja en el entorno la cuota vigente para que la pantalla
// muestre "usado de total". La medición la escribe el barrido de disco: acá no se
// consulta el daemon.
func (s *PracticeEnvironmentService) annotateDiskLimit(ctx context.Context, environment *domain.PracticeEnvironment) {
	if environment == nil {
		return
	}
	tenantID := environment.TenantID
	if tenantID == "" {
		tenantID = domain.DefaultTenantID
	}
	environment.DiskLimitMB = s.effectivePolicies(ctx, tenantID).EffectiveDiskQuotaMB()
}

// diskQuotaVerdict mide el volumen y lo compara con la política del nodo. Un
// adaptador sin medición deja pasar la entrega y lo deja dicho en el log: nunca
// se bloquea a un alumno por una medición que no se pudo hacer.
func (s *PracticeEnvironmentService) diskQuotaVerdict(ctx context.Context, tenantID, environmentID, volumeName string) *domain.DiskQuotaVerdict {
	measurer, ok := s.docker.(volumeUsageMeasurer)
	if !ok {
		return nil
	}
	policies := s.effectivePolicies(ctx, tenantID)
	if policies.EffectiveDiskQuotaMB() <= 0 {
		return nil
	}
	usedMB, err := measurer.MeasurePracticeEnvironmentVolumeMB(ctx, volumeName)
	if err != nil {
		log.Printf("[Disk Quota] entorno %s: no se pudo medir el volumen: %v", environmentID, err)
		return nil
	}
	verdict := domain.EvaluateDiskQuota(usedMB*1024*1024, policies.EffectiveDiskQuotaMB(), policies.EffectiveDiskWarningPct(), policies.EffectiveDiskBlockPct())
	if verdict.Status == domain.DiskQuotaWarning {
		log.Printf("[Disk Quota] entorno %s: %s", environmentID, verdict.Message)
	}
	return &verdict
}

// submissionLimits agrupa los topes de una entrega. Un proyecto grande admite
// el mismo árbol que la carga del docente: si la entrega aplicara los topes del
// camino liviano, el alumno no podría entregar lo que el docente sembró.
type submissionLimits struct {
	totalBytes int64
	fileBytes  int
	files      int
	artifact   bool
}

var (
	submissionLimitsText = submissionLimits{
		totalBytes: 4 * 1024 * 1024,
		fileBytes:  1024 * 1024,
		files:      800,
	}
	submissionLimitsArtifact = submissionLimits{
		totalBytes: int64(maxWorkspaceProjectArchiveExpanded),
		fileBytes:  maxWorkspaceProjectArchiveFile,
		files:      maxWorkspaceProjectArchiveFiles,
		// El árbol grande no se persiste como texto: va al depósito.
		artifact: true,
	}
)

// submissionLimitsFor decide los topes según la versión sembrada en el entorno.
// Si no se puede leer la versión se conservan los topes livianos: es el
// comportamiento anterior y no agranda una lectura por un dato dudoso.
func (s *PracticeEnvironmentService) submissionLimitsFor(ctx context.Context, environment *domain.PracticeEnvironment) submissionLimits {
	if environment == nil || environment.WorkspaceProjectVersionID == nil || s.workspaceProjects == nil {
		return submissionLimitsText
	}
	version, err := s.workspaceProjects.GetVersion(ctx, environment.TenantID, *environment.WorkspaceProjectVersionID)
	if err != nil || version == nil {
		log.Printf("aviso: entrega del entorno %s sin versión legible, se usan los topes livianos: %v", environment.ID, err)
		return submissionLimitsText
	}
	if version.Artifact() != nil {
		return submissionLimitsArtifact
	}
	return submissionLimitsText
}

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
		return nil, s.frozenErrorFor(ctx, environment)
	}

	tenantID := domain.GetTenantID(ctx)
	if tenantID == "" {
		tenantID = environment.TenantID
	}
	if tenantID == "" {
		tenantID = domain.DefaultTenantID
	}

	// La fecha límite y la política de reentregas se verifican sobre el ejercicio.
	if s.exerciseRepo != nil {
		exercise, err := s.exerciseRepo.GetByIDAndTenant(ctx, exerciseID, tenantID)
		if err == nil && exercise != nil {
			if exercise.DueDate != nil && time.Now().After(*exercise.DueDate) {
				return nil, domain.ErrExerciseDeadlineExpired
			}
			if !exercise.AllowsResubmit() && s.submissionRepo != nil {
				existing, err := s.submissionRepo.ListByExerciseAndStudent(ctx, tenantID, exerciseID, environment.StudentID)
				if err == nil && len(existing) > 0 {
					return nil, domain.ErrResubmissionNotAllowed
				}
			}
		}
	}

	volumeStore, ok := s.docker.(domain.PracticeEnvironmentVolumeStore)
	if !ok {
		return nil, ErrPracticeEnvironmentVolumeUnavailable
	}

	volumeName := environment.VolumeName()
	// Cuota blanda de disco: se mide el volumen antes de escribir la entrega y
	// un entorno lleno se frena con un mensaje accionable en vez de crecer hasta
	// quedarse sin nodo.
	if verdict := s.diskQuotaVerdict(ctx, tenantID, environmentID, volumeName); verdict != nil {
		if verdict.Status == domain.DiskQuotaFull {
			return nil, fmt.Errorf("%w: %s; borrá archivos generados o cachés y volvé a entregar", domain.ErrDiskQuotaExceeded, verdict.Message)
		}
	}
	if len(files) > 0 {
		if err := volumeStore.WritePracticeEnvironmentVolumeFiles(ctx, volumeName, files); err != nil {
			return nil, fmt.Errorf("failed to write project files: %w", err)
		}
	}

	limits := s.submissionLimitsFor(ctx, environment)
	volumeFiles, err := snapshotForSubmission(ctx, volumeStore, volumeName, limits.totalBytes)
	if err != nil {
		if errors.Is(err, domain.ErrPracticeEnvironmentSnapshotTooLarge) {
			// El detalle del adaptador trae la ruta y el tamaño.
			return nil, fmt.Errorf("the project to submit exceeds the total limit of %d MB: %w", limits.totalBytes/(1024*1024), err)
		}
		return nil, fmt.Errorf("failed to snapshot environment volume: %w", err)
	}

	if len(volumeFiles) > limits.files {
		return nil, fmt.Errorf("environment %s has %d files to submit; the limit is %d", environmentID, len(volumeFiles), limits.files)
	}
	for _, file := range volumeFiles {
		if len(file.Content) > limits.fileBytes {
			return nil, fmt.Errorf("environment %s has a file larger than %d bytes (%d bytes): %s", environmentID, limits.fileBytes, len(file.Content), file.Path)
		}
	}

	snapshot := buildProjectSnapshot(volumeFiles)
	if strings.TrimSpace(snapshot) == "" {
		return nil, fmt.Errorf("environment %s has no code files to submit", environmentID)
	}

	baseStatus := classifyIDEBaseStatus(environment)

	submission := &domain.Submission{
		ID:            uuid.NewString(),
		TenantID:      tenantID,
		ExerciseID:    exerciseID,
		StudentID:     environment.StudentID,
		EnvironmentID: &environment.ID,
		// Un proyecto de IDE no lo evalúa el juez: no tiene veredicto. La
		// ausencia se guarda como ausencia y el trabajo queda en revisión.
		Verdict:                   "",
		ReviewStatus:              string(domain.ReviewPending),
		ASTResult:                 json.RawMessage("{}"),
		GeneratedCases:            json.RawMessage("{}"),
		// Un entorno IDE no ejecuta análisis asintótico; se deja nil (NULL en DB) para no crear {} vacío.
		ComplexityAnalysis:        nil,
		BaseStatus:                baseStatus,
		WorkspaceProjectVersionID: environment.WorkspaceProjectVersionID,
	}

	// Un proyecto grande viaja al depósito: la fila guarda la referencia y el
	// texto queda vacío. El camino liviano sigue persistiendo el snapshot.
	if limits.artifact {
		if err := s.storeProjectSubmissionArchive(ctx, submission, volumeFiles); err != nil {
			return nil, err
		}
	} else {
		submission.Code = snapshot
	}

	if err := s.submissionRepo.Create(ctx, submission); err != nil {
		return nil, fmt.Errorf("failed to create project submission: %w", err)
	}

	// Entrega de examen: se registra el checkpoint de entrega con su código y timestamp.
	// El volumen permanece en lectura/escritura (:rw) para permitir correcciones y
	// re-entregas mientras el examen siga en curso. El congelamiento atómico en solo
	// lectura (:ro) se realiza al cerrar el aula de examen.

	return submission, nil
}

// storeProjectSubmissionArchive empaqueta el árbol entregado y lo deposita. La
// fila de la entrega solo conserva la referencia, la huella y los tamaños; el
// objeto queda para la retención del depósito.
func (s *PracticeEnvironmentService) storeProjectSubmissionArchive(
	ctx context.Context,
	submission *domain.Submission,
	volumeFiles []domain.PracticeEnvironmentVolumeFile,
) error {
	if s == nil || s.workspaceArtifacts == nil {
		return domain.ErrWorkspaceProjectArchiveStoreUnavailable
	}
	entries := make([]WorkspaceProjectImportFile, 0, len(volumeFiles))
	for _, file := range volumeFiles {
		entries = append(entries, WorkspaceProjectImportFile{Path: file.Path, Content: file.Content})
	}
	archive, err := packWorkspaceProjectArchive(entries)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(archive)
	hash := hex.EncodeToString(sum[:])
	key := submissionArtifactKey(submission.TenantID, submission.ExerciseID, submission.ID)
	if _, _, err := s.workspaceArtifacts.Put(ctx, key, bytes.NewReader(archive)); err != nil {
		return fmt.Errorf("no se pudo guardar la entrega en el depósito: %w", err)
	}
	artifactBytes := int64(len(archive))
	artifactFiles := len(entries)
	submission.ArtifactKey = &key
	submission.ArtifactSHA256 = &hash
	submission.ArtifactBytes = &artifactBytes
	submission.ArtifactFiles = &artifactFiles
	return nil
}

// classifyIDEBaseStatus traduce el estado de siembra del entorno a la
// clasificación persistida de la entrega. Un entorno legacy o sin datos de
// siembra no puede pretender una versión exacta.
func classifyIDEBaseStatus(environment *domain.PracticeEnvironment) string {
	if environment == nil {
		return domain.BaseStatusLegacyUnknown
	}
	switch environment.WorkspaceSeedMode {
	case domain.WorkspaceSeedModeNone:
		return domain.BaseStatusNoBase
	case domain.WorkspaceSeedModeProject:
		if environment.WorkspaceProjectVersionID != nil {
			return domain.BaseStatusVersion
		}
		return domain.BaseStatusLegacyUnknown
	default:
		return domain.BaseStatusLegacyUnknown
	}
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
