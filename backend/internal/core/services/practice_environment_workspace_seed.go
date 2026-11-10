package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"

	"solv-backend/internal/core/domain"
)

// workspaceInitStatusUpdater es una capacidad opcional del repositorio de
// entornos. Persiste solo la máquina de estados de la siembra, sin mezclarla
// con el estado operativo del contenedor.
type workspaceInitStatusUpdater interface {
	UpdateWorkspaceInitStatus(ctx context.Context, id, status string) error
}

// lockWorkspaceStart serializa las aperturas concurrentes del mismo entorno
// dentro de este proceso. La clave incluye tenant, persona y práctica; dos
// prácticas distintas no se bloquean entre sí.
func (s *PracticeEnvironmentService) lockWorkspaceStart(tenantID, userID, subjectID, exerciseID string) func() {
	if s == nil {
		return func() {}
	}
	key := strings.Join([]string{
		strings.TrimSpace(tenantID),
		strings.TrimSpace(userID),
		strings.TrimSpace(subjectID),
		strings.TrimSpace(exerciseID),
	}, "|")

	s.workspaceSeedMu.Lock()
	if s.workspaceSeedLocks == nil {
		s.workspaceSeedLocks = make(map[string]*sync.Mutex)
	}
	lock, ok := s.workspaceSeedLocks[key]
	if !ok {
		lock = &sync.Mutex{}
		s.workspaceSeedLocks[key] = lock
	}
	s.workspaceSeedMu.Unlock()

	lock.Lock()
	return lock.Unlock
}

// resolveWorkspaceSeed lee la versión publicada exacta que corresponde al
// ejercicio en este momento. La decisión se toma una sola vez, antes de crear
// el entorno; una publicación posterior no cambia la asignación.
func (s *PracticeEnvironmentService) resolveWorkspaceSeed(ctx context.Context, tenantID string, exercise *domain.Exercise) (*domain.WorkspaceProjectVersion, error) {
	if s == nil || s.workspaceProjects == nil || exercise == nil {
		return nil, nil
	}
	if exercise.EnvironmentType != string(domain.EnvironmentTypeIDEPersistente) {
		return nil, nil
	}
	project, err := s.workspaceProjects.GetProjectByExercise(ctx, tenantID, exercise.ID)
	if err != nil {
		if errors.Is(err, domain.ErrWorkspaceProjectNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to resolve workspace project: %w", err)
	}
	if project == nil || project.CurrentPublishedVersionID == nil || strings.TrimSpace(*project.CurrentPublishedVersionID) == "" {
		return nil, nil
	}
	version, err := s.workspaceProjects.GetVersion(ctx, tenantID, *project.CurrentPublishedVersionID)
	if err != nil {
		return nil, fmt.Errorf("%w: no se pudo leer la versión publicada", domain.ErrWorkspaceSeedVersionInvalid)
	}
	if version == nil || version.TenantID != tenantID || version.ProjectID != project.ID || version.ExerciseID != exercise.ID {
		return nil, fmt.Errorf("%w: la versión no pertenece al proyecto del ejercicio", domain.ErrWorkspaceSeedVersionInvalid)
	}
	return version, nil
}

// ensureExistingWorkspaceSeed reanuda solo una inicialización que la base
// identifica como pendiente o fallida. Un entorno ya verificado, legacy o sin
// proyecto no vuelve a sembrarse ni consulta la publicación actual.
func (s *PracticeEnvironmentService) ensureExistingWorkspaceSeed(ctx context.Context, instance *domain.PracticeEnvironment) error {
	if s == nil || instance == nil {
		return nil
	}
	switch instance.WorkspaceSeedMode {
	case "", domain.WorkspaceSeedModeNone:
		return nil
	case domain.WorkspaceSeedModeLegacy:
		return nil
	case domain.WorkspaceSeedModeProject:
	default:
		_ = s.setWorkspaceInitStatus(ctx, instance, domain.WorkspaceInitStatusQuarantined)
		return fmt.Errorf("%w: modo de siembra desconocido %q", domain.ErrWorkspaceSeedVolumeAmbiguous, instance.WorkspaceSeedMode)
	}

	switch instance.WorkspaceInitStatus {
	case domain.WorkspaceInitStatusSetupDone:
		seed, err := s.loadAssignedWorkspaceSeed(ctx, instance)
		if err != nil {
			return err
		}
		// El estado de la base no es suficiente: un volumen recreado o perdido
		// debe detectarse antes de entregar un IDE vacío. Solo se comprueba el
		// marcador; el contenido puede haber sido modificado por el alumno.
		if verifier, ok := s.docker.(domain.WorkspaceSeedVolumeVerifier); ok {
			if instance.VolumeName() == "" {
				_ = s.setWorkspaceInitStatus(ctx, instance, domain.WorkspaceInitStatusQuarantined)
				return fmt.Errorf("%w: el entorno no tiene volumen", domain.ErrWorkspaceSeedVolumeUnavailable)
			}
			if err := verifier.VerifyWorkspaceVolume(ctx, instance.VolumeName(), seed); err != nil {
				status := domain.WorkspaceInitStatusFailed
				if errors.Is(err, domain.ErrWorkspaceSeedVolumeConflict) || errors.Is(err, domain.ErrWorkspaceSeedVolumeAmbiguous) {
					status = domain.WorkspaceInitStatusQuarantined
				}
				_ = s.setWorkspaceInitStatus(ctx, instance, status)
				return err
			}
		}
		return nil
	case domain.WorkspaceInitStatusQuarantined:
		return fmt.Errorf("%w: el entorno requiere revisión", domain.ErrWorkspaceSeedVolumeAmbiguous)
	case domain.WorkspaceInitStatusPending, domain.WorkspaceInitStatusVolumeCreated, domain.WorkspaceInitStatusFailed:
		// Estados transitorios reintentables.
	default:
		_ = s.setWorkspaceInitStatus(ctx, instance, domain.WorkspaceInitStatusQuarantined)
		return fmt.Errorf("%w: estado de inicialización desconocido %q", domain.ErrWorkspaceSeedVolumeAmbiguous, instance.WorkspaceInitStatus)
	}

	seed, err := s.loadAssignedWorkspaceSeed(ctx, instance)
	if err != nil {
		return err
	}
	if instance.VolumeName() == "" {
		_ = s.setWorkspaceInitStatus(ctx, instance, domain.WorkspaceInitStatusQuarantined)
		return fmt.Errorf("%w: el entorno no tiene volumen", domain.ErrWorkspaceSeedVolumeUnavailable)
	}
	if s.docker == nil {
		_ = s.setWorkspaceInitStatus(ctx, instance, domain.WorkspaceInitStatusFailed)
		return domain.ErrWorkspaceSeedVolumeUnavailable
	}
	if err := s.docker.EnsureVolumeExists(ctx, instance.VolumeName()); err != nil {
		_ = s.setWorkspaceInitStatus(ctx, instance, domain.WorkspaceInitStatusFailed)
		return fmt.Errorf("failed to ensure workspace volume: %w", err)
	}
	return s.seedWorkspaceVolume(ctx, instance, seed)
}

// loadAssignedWorkspaceSeed recupera exclusivamente la versión persistida en el
// entorno. Nunca consulta current_published_version_id, de modo que una
// publicación posterior no cambia lo que recibe un entorno existente.
func (s *PracticeEnvironmentService) loadAssignedWorkspaceSeed(ctx context.Context, instance *domain.PracticeEnvironment) (domain.WorkspaceSeed, error) {
	if instance == nil {
		return domain.WorkspaceSeed{}, fmt.Errorf("%w: entorno ausente", domain.ErrWorkspaceSeedVolumeUnavailable)
	}
	if instance.WorkspaceProjectVersionID == nil || strings.TrimSpace(*instance.WorkspaceProjectVersionID) == "" {
		_ = s.setWorkspaceInitStatus(ctx, instance, domain.WorkspaceInitStatusQuarantined)
		return domain.WorkspaceSeed{}, fmt.Errorf("%w: el entorno no conserva la referencia de versión", domain.ErrWorkspaceSeedVersionInvalid)
	}
	if s.workspaceProjects == nil {
		_ = s.setWorkspaceInitStatus(ctx, instance, domain.WorkspaceInitStatusFailed)
		return domain.WorkspaceSeed{}, fmt.Errorf("%w: no hay repositorio de versiones configurado", domain.ErrWorkspaceSeedVolumeUnavailable)
	}
	version, err := s.workspaceProjects.GetVersion(ctx, instance.TenantID, *instance.WorkspaceProjectVersionID)
	if err != nil || version == nil {
		_ = s.setWorkspaceInitStatus(ctx, instance, domain.WorkspaceInitStatusQuarantined)
		return domain.WorkspaceSeed{}, fmt.Errorf("%w: no se pudo recuperar la versión asignada", domain.ErrWorkspaceSeedVersionInvalid)
	}
	if version.TenantID != instance.TenantID || version.ExerciseID != valueOrEmpty(instance.ExerciseID) {
		_ = s.setWorkspaceInitStatus(ctx, instance, domain.WorkspaceInitStatusQuarantined)
		return domain.WorkspaceSeed{}, fmt.Errorf("%w: la versión asignada no pertenece al entorno", domain.ErrWorkspaceSeedVersionInvalid)
	}
	project, err := s.workspaceProjects.GetProjectByExercise(ctx, instance.TenantID, valueOrEmpty(instance.ExerciseID))
	if err != nil || project == nil || project.ID != version.ProjectID {
		_ = s.setWorkspaceInitStatus(ctx, instance, domain.WorkspaceInitStatusQuarantined)
		return domain.WorkspaceSeed{}, fmt.Errorf("%w: la versión asignada no pertenece al proyecto del ejercicio", domain.ErrWorkspaceSeedVersionInvalid)
	}
	seed, err := workspaceSeedFromVersion(version)
	if err != nil {
		// Una versión inmutable con hash o snapshot inválido no se arregla
		// reintentando: requiere revisión y no se escribe nada.
		_ = s.setWorkspaceInitStatus(ctx, instance, domain.WorkspaceInitStatusQuarantined)
		return domain.WorkspaceSeed{}, err
	}
	return seed, nil
}

// seedWorkspaceVolume delega en la capacidad conservadora del orquestador y
// traduce su resultado a la máquina de estados. Nunca marca setup_done si la
// verificación no terminó.
func (s *PracticeEnvironmentService) seedWorkspaceVolume(ctx context.Context, instance *domain.PracticeEnvironment, seed domain.WorkspaceSeed) error {
	if instance == nil {
		return fmt.Errorf("%w: entorno ausente", domain.ErrWorkspaceSeedVolumeUnavailable)
	}
	store, ok := s.docker.(domain.WorkspaceSeedVolumeStore)
	if !ok {
		_ = s.setWorkspaceInitStatus(ctx, instance, domain.WorkspaceInitStatusFailed)
		return domain.ErrWorkspaceSeedVolumeUnavailable
	}
	if instance.VolumeName() == "" {
		_ = s.setWorkspaceInitStatus(ctx, instance, domain.WorkspaceInitStatusQuarantined)
		return fmt.Errorf("%w: el entorno no tiene volumen", domain.ErrWorkspaceSeedVolumeUnavailable)
	}
	if err := s.setWorkspaceInitStatus(ctx, instance, domain.WorkspaceInitStatusVolumeCreated); err != nil {
		return err
	}
	if err := store.PrepareWorkspaceVolume(ctx, instance.VolumeName(), seed); err != nil {
		status := domain.WorkspaceInitStatusFailed
		if errors.Is(err, domain.ErrWorkspaceSeedVolumeNotEmpty) ||
			errors.Is(err, domain.ErrWorkspaceSeedVolumeConflict) ||
			errors.Is(err, domain.ErrWorkspaceSeedVolumeAmbiguous) ||
			errors.Is(err, domain.ErrWorkspaceSeedVersionInvalid) {
			status = domain.WorkspaceInitStatusQuarantined
		}
		_ = s.setWorkspaceInitStatus(ctx, instance, status)
		return err
	}
	if err := s.setWorkspaceInitStatus(ctx, instance, domain.WorkspaceInitStatusSetupDone); err != nil {
		return err
	}
	return nil
}

func (s *PracticeEnvironmentService) setWorkspaceInitStatus(ctx context.Context, instance *domain.PracticeEnvironment, status string) error {
	if instance == nil {
		return fmt.Errorf("%w: entorno ausente", domain.ErrWorkspaceSeedVolumeUnavailable)
	}
	updater, ok := s.repo.(workspaceInitStatusUpdater)
	if !ok {
		return fmt.Errorf("%w: no hay persistencia para el estado de inicialización", domain.ErrWorkspaceSeedVolumeUnavailable)
	}
	if err := updater.UpdateWorkspaceInitStatus(ctx, instance.ID, status); err != nil {
		return err
	}
	instance.WorkspaceInitStatus = status
	return nil
}

// workspaceSeedFromVersion valida y convierte la versión inmutable en archivos.
// La fuente es text_snapshot, nunca exercises.boilerplate ni el borrador actual.
func workspaceSeedFromVersion(version *domain.WorkspaceProjectVersion) (domain.WorkspaceSeed, error) {
	if version == nil || strings.TrimSpace(version.TextSnapshot) == "" {
		return domain.WorkspaceSeed{}, fmt.Errorf("%w: la versión está vacía", domain.ErrWorkspaceSeedVersionInvalid)
	}
	files, ok := domain.ParseProjectSnapshot(version.TextSnapshot, "")
	if !ok || len(files) == 0 {
		return domain.WorkspaceSeed{}, fmt.Errorf("%w: el snapshot no es reconocible", domain.ErrWorkspaceSeedVersionInvalid)
	}
	sum := sha256.Sum256([]byte(version.TextSnapshot))
	hash := hex.EncodeToString(sum[:])
	if strings.TrimSpace(version.ContentSHA256) != "" && !strings.EqualFold(hash, version.ContentSHA256) {
		return domain.WorkspaceSeed{}, fmt.Errorf("%w: el hash de la versión no coincide", domain.ErrWorkspaceSeedVersionInvalid)
	}
	seed := domain.WorkspaceSeed{
		VersionID:     version.ID,
		ContentSHA256: hash,
		Files:         make([]domain.WorkspaceSeedFile, 0, len(files)),
	}
	totalBytes := 0
	for _, file := range files {
		totalBytes += len(file.Content)
		seed.Files = append(seed.Files, domain.WorkspaceSeedFile{Path: file.Path, Content: file.Content})
	}
	if version.FileCount > 0 && version.FileCount != len(files) {
		return domain.WorkspaceSeed{}, fmt.Errorf("%w: la cantidad de archivos no coincide", domain.ErrWorkspaceSeedVersionInvalid)
	}
	if version.TotalBytes > 0 && version.TotalBytes != totalBytes {
		return domain.WorkspaceSeed{}, fmt.Errorf("%w: el tamaño del proyecto no coincide", domain.ErrWorkspaceSeedVersionInvalid)
	}
	return seed, nil
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}
