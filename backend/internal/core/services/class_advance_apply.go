package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/google/uuid"

	"solv-backend/internal/core/domain"
)

const (
	// maxAdvanceApplyFiles acota los archivos que se escriben al aplicar un avance.
	maxAdvanceApplyFiles = 1024
	// maxAdvanceBackupBytes acota lo que se lee del volumen para respaldar el
	// estado previo de los archivos que se van a sobrescribir.
	maxAdvanceBackupBytes = 8 << 20
	// advanceBackupRoot es la carpeta del volumen del alumno donde queda el
	// estado previo de cada aplicación.
	advanceBackupRoot = ".solv_backups"
	// maxAdvanceBackups es cuántos respaldos se conservan por volumen. El
	// respaldo es una red de seguridad, no un historial de versiones.
	maxAdvanceBackups = 5
)

// ErrAdvanceStorageUnavailable indica que el servicio no tiene el almacén de
// snapshots ni el volumen de los entornos cableados.
var ErrAdvanceStorageUnavailable = errors.New("advance snapshot storage is not configured")

// AdvanceApplyResult resume qué cambió en el entorno del alumno. El frontend
// muestra estas listas para que el alumno vea qué se agregó y qué se reemplazó.
type AdvanceApplyResult struct {
	AdvanceID    string   `json:"advance_id"`
	FilesWritten int      `json:"files_written"`
	Overwritten  []string `json:"overwritten"`
	Added        []string `json:"added"`
	BackupPath   *string  `json:"backup_path,omitempty"`
}

// WithApplier habilita la aplicación de un avance en el entorno del alumno: el
// almacén de snapshots (A3), el repositorio de entornos y la capacidad de
// operar el volumen. Sin esto, emitir y congelar siguen funcionando.
func (s *ClassAdvanceService) WithApplier(snapshots domain.AdvanceSnapshotStore, workspaces domain.WorkspaceRepository, volumes domain.WorkspaceVolumeStore) *ClassAdvanceService {
	s.snapshots = snapshots
	s.workspaces = workspaces
	s.volumes = volumes
	return s
}

// PendingAdvances devuelve los avisos que el alumno todavía puede aplicar. Un
// aviso cuya ventana de rezago venció se cierra como expired en el acto, para
// que deje de aparecer en la consulta periódica del laboratorio en lugar de
// mostrarse y fallar al confirmarlo.
func (s *ClassAdvanceService) PendingAdvances(ctx context.Context, tenantID, studentID string) ([]domain.AdvanceDelivery, error) {
	if err := requireStudentIdentity(tenantID, studentID); err != nil {
		return nil, err
	}
	deliveries, err := s.store.PendingAdvances(ctx, tenantID, studentID)
	if err != nil {
		return nil, err
	}
	now := s.now()
	appliable := make([]domain.AdvanceDelivery, 0, len(deliveries))
	for _, delivery := range deliveries {
		if delivery.Appliable(now) {
			appliable = append(appliable, delivery)
			continue
		}
		// En examen el aviso no se marca como vencido: es una prohibición, no un
		// plazo. Se oculta y queda en manos del docente resolverlo.
		if delivery.Purpose != "exam" {
			s.decideBestEffort(ctx, tenantID, studentID, delivery.ID, domain.AdvanceRecipientExpired, nil)
		}
	}
	return appliable, nil
}

// AdvancePreview describe qué cambiaría en el entorno del alumno si aplica el
// aviso. Es la confirmación previa de la pantalla: separa los archivos que se
// agregan de los que se reemplazan, sin escribir nada.
type AdvancePreview struct {
	AdvanceID   string   `json:"advance_id"`
	ExerciseID  string   `json:"exercise_id"`
	FileCount   int      `json:"file_count"`
	Added       []string `json:"added"`
	Overwritten []string `json:"overwritten"`
	// Files trae el contenido emitido de cada archivo, para que el editor arme
	// el diff en memoria sin escribir archivos temporales en el disco del
	// alumno.
	Files map[string]string `json:"files"`
}

// Preview lee el snapshot y el volumen vigente del alumno para decir qué
// archivos se agregan y cuáles se reemplazan. No escribe: el respaldo y la
// copia ocurren al aplicar.
func (s *ClassAdvanceService) Preview(ctx context.Context, tenantID, studentID, advanceID string) (*AdvancePreview, error) {
	if err := requireStudentIdentity(tenantID, studentID); err != nil {
		return nil, err
	}
	if s.snapshots == nil || s.workspaces == nil || s.volumes == nil {
		return nil, ErrAdvanceStorageUnavailable
	}
	if _, err := uuid.Parse(strings.TrimSpace(advanceID)); err != nil {
		return nil, domain.ErrAdvanceNotFound
	}

	delivery, err := s.store.AdvanceForStudent(ctx, tenantID, studentID, advanceID)
	if err != nil {
		return nil, err
	}
	if delivery.Status != domain.AdvanceRecipientPending {
		return nil, domain.ErrAdvanceNotPending
	}
	if delivery.Purpose == "exam" {
		return nil, domain.ErrAdvanceExamBlocked
	}
	if !delivery.Appliable(s.now()) {
		s.decideBestEffort(ctx, tenantID, studentID, delivery.ID, domain.AdvanceRecipientExpired, nil)
		return nil, domain.ErrAdvanceGraceExpired
	}

	workspace, err := s.studentWorkspace(ctx, studentID, delivery)
	if err != nil || workspace == nil {
		return nil, domain.ErrAdvanceWorkspaceStopped
	}

	// Tampoco aquí se exige el contenedor en ejecución: leer el volumen no lo
	// necesita, y el rechazo de la copia se explica al aplicar con su motivo.
	incoming, err := s.readSnapshot(ctx, delivery)
	if err != nil {
		return nil, err
	}
	current, err := s.volumeFiles(ctx, workspaceVolumeName(workspace))
	if err != nil {
		return nil, fmt.Errorf("failed to read student volume before previewing advance: %w", err)
	}

	overwritten, added := splitAdvanceChanges(incoming, current)
	return &AdvancePreview{
		AdvanceID:   delivery.ID,
		ExerciseID:  delivery.ExerciseID,
		FileCount:   len(incoming),
		Added:       added,
		Overwritten: overwritten,
		Files:       incoming,
	}, nil
}

// Dismiss marca el aviso como descartado. Descartar dos veces no cambia nada.
func (s *ClassAdvanceService) Dismiss(ctx context.Context, tenantID, studentID, advanceID string) error {
	if err := requireStudentIdentity(tenantID, studentID); err != nil {
		return err
	}
	delivery, err := s.store.AdvanceForStudent(ctx, tenantID, studentID, advanceID)
	if err != nil {
		return err
	}
	switch delivery.Status {
	case domain.AdvanceRecipientDismissed:
		return nil
	case domain.AdvanceRecipientPending:
		if err := s.store.DecideRecipient(ctx, tenantID, studentID, advanceID, domain.AdvanceRecipientDismissed, nil); err != nil {
			return err
		}
		s.recordAudit(ctx, tenantID, studentID, domain.AuditClassAdvanceDismissed, delivery.SessionID, map[string]any{
			"advance_id":  delivery.ID,
			"exercise_id": delivery.ExerciseID,
		})
		return nil
	default:
		return domain.ErrAdvanceNotPending
	}
}

// Apply copia el avance completo al volumen del alumno. Es el camino de
// compatibilidad hacia atrás de la entrega sin selección de archivos.
func (s *ClassAdvanceService) Apply(ctx context.Context, tenantID, studentID, advanceID string) (*AdvanceApplyResult, error) {
	return s.ApplySelection(ctx, tenantID, studentID, advanceID, nil)
}

// ApplySelection copia al volumen del alumno solo los archivos elegidos, sin
// pausar ni reiniciar su contenedor. Antes de escribir respalda en
// .solv_backups/<avance_id>/ los archivos que se van a sobrescribir, y deja el
// resultado en el destinatario. Con la lista vacía aplica el avance completo,
// que es lo que hacía la ruta antes del checklist.
func (s *ClassAdvanceService) ApplySelection(ctx context.Context, tenantID, studentID, advanceID string, selectedFiles []string) (*AdvanceApplyResult, error) {
	if err := requireStudentIdentity(tenantID, studentID); err != nil {
		return nil, err
	}
	if s.snapshots == nil || s.workspaces == nil || s.volumes == nil {
		return nil, ErrAdvanceStorageUnavailable
	}
	if _, err := uuid.Parse(strings.TrimSpace(advanceID)); err != nil {
		return nil, domain.ErrAdvanceNotFound
	}

	delivery, err := s.store.AdvanceForStudent(ctx, tenantID, studentID, advanceID)
	if err != nil {
		return nil, err
	}
	if delivery.Status != domain.AdvanceRecipientPending {
		return nil, domain.ErrAdvanceNotPending
	}
	if delivery.Purpose == "exam" {
		return nil, domain.ErrAdvanceExamBlocked
	}

	now := s.now()
	if !delivery.Appliable(now) {
		// El aviso vencido se cierra en el acto: así deja de aparecer en la
		// consulta periódica del alumno.
		s.decideBestEffort(ctx, tenantID, studentID, delivery.ID, domain.AdvanceRecipientExpired, nil)
		return nil, domain.ErrAdvanceGraceExpired
	}

	workspace, err := s.studentWorkspace(ctx, studentID, delivery)
	if err != nil || workspace == nil {
		return nil, domain.ErrAdvanceWorkspaceStopped
	}
	if workspace.Status != domain.WorkspaceStatusRunning {
		return nil, domain.ErrAdvanceWorkspaceStopped
	}

	// Las aplicaciones simultáneas se acotan aquí: el resto espera su turno.
	select {
	case s.applySlots <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	defer func() { <-s.applySlots }()

	// Un fallo al leer el snapshot deja el aviso pendiente: nada se tocó y el
	// alumno puede reintentar cuando el almacén responda.
	incoming, err := s.readSnapshot(ctx, delivery)
	if err != nil {
		return nil, err
	}

	// El checklist del editor llega como una lista de rutas. Se acota antes de
	// leer el volumen: una ruta que no está en el avance se rechaza en vez de
	// aplicarse en silencio.
	if len(selectedFiles) > 0 {
		incoming, err = selectAdvanceFiles(incoming, selectedFiles)
		if err != nil {
			return nil, err
		}
	}

	volumeName := workspaceVolumeName(workspace)

	// El respaldo se arma antes de tocar el volumen: si no se puede leer el
	// estado previo, no se escribe nada.
	current, err := s.volumeFiles(ctx, volumeName)
	if err != nil {
		return nil, fmt.Errorf("failed to read student volume before applying advance: %w", err)
	}

	overwritten, added := splitAdvanceChanges(incoming, current)

	var backupPath *string
	if len(overwritten) > 0 {
		root := advanceBackupRoot + "/" + delivery.ID
		backupFiles := make(map[string]string, len(overwritten))
		for _, path := range overwritten {
			// Se guarda el contenido vigente del alumno, no el del snapshot.
			backupFiles[root+"/"+path] = current[path]
		}
		// Si el respaldo no se escribe, el código del alumno sigue intacto: el
		// aviso queda pendiente en lugar de fallar de forma definitiva.
		if err := s.volumes.WriteWorkspaceVolumeFiles(ctx, volumeName, backupFiles); err != nil {
			return nil, fmt.Errorf("failed to back up student files before applying advance: %w", err)
		}
		backupPath = &root
	}

	if err := s.volumes.WriteWorkspaceVolumeFiles(ctx, volumeName, incoming); err != nil {
		s.decideBestEffort(ctx, tenantID, studentID, delivery.ID, domain.AdvanceRecipientFailed, backupPath)
		return nil, fmt.Errorf("failed to write advance into student volume: %w", err)
	}

	if err := s.store.DecideRecipient(ctx, tenantID, studentID, delivery.ID, domain.AdvanceRecipientApplied, backupPath); err != nil {
		return nil, err
	}

	// La poda es best-effort: si falla, la aplicación ya quedó bien y los
	// respaldos viejos se reintentan en la próxima. Nunca borra el recién creado.
	s.pruneAdvanceBackups(ctx, volumeName)

	s.recordAudit(ctx, tenantID, studentID, domain.AuditClassAdvanceApplied, delivery.SessionID, map[string]any{
		"advance_id":    delivery.ID,
		"exercise_id":   delivery.ExerciseID,
		"files_written": len(incoming),
		"overwritten":   len(overwritten),
		"added":         len(added),
		"backup_path":   backupPath,
	})

	return &AdvanceApplyResult{
		AdvanceID:    delivery.ID,
		FilesWritten: len(incoming),
		Overwritten:  overwritten,
		Added:        added,
		BackupPath:   backupPath,
	}, nil
}

// AdvanceRollbackResult resume qué se restauró al deshacer un avance.
type AdvanceRollbackResult struct {
	AdvanceID     string   `json:"advance_id"`
	FilesRestored int      `json:"files_restored"`
	Restored      []string `json:"restored"`
	BackupPath    string   `json:"backup_path"`
}

// Rollback deshace el último avance aplicado: devuelve al volumen del alumno el
// contenido que quedó en .solv_backups/<avance_id>/ y deja el destinatario en
// pendiente para que pueda volver a aplicarlo. El respaldo se borra recién
// cuando el estado ya quedó actualizado.
func (s *ClassAdvanceService) Rollback(ctx context.Context, tenantID, studentID, advanceID string) (*AdvanceRollbackResult, error) {
	if err := requireStudentIdentity(tenantID, studentID); err != nil {
		return nil, err
	}
	if s.snapshots == nil || s.workspaces == nil || s.volumes == nil {
		return nil, ErrAdvanceStorageUnavailable
	}
	if _, err := uuid.Parse(strings.TrimSpace(advanceID)); err != nil {
		return nil, domain.ErrAdvanceNotFound
	}

	delivery, err := s.store.AdvanceForStudent(ctx, tenantID, studentID, advanceID)
	if err != nil {
		return nil, err
	}
	if delivery.Status != domain.AdvanceRecipientApplied {
		return nil, domain.ErrAdvanceRollbackNotApplied
	}

	workspace, err := s.studentWorkspace(ctx, studentID, delivery)
	if err != nil || workspace == nil {
		return nil, domain.ErrAdvanceWorkspaceStopped
	}
	if workspace.Status != domain.WorkspaceStatusRunning {
		return nil, domain.ErrAdvanceWorkspaceStopped
	}

	select {
	case s.applySlots <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	defer func() { <-s.applySlots }()

	volumeName := workspaceVolumeName(workspace)
	backupRoot := advanceBackupRoot + "/" + delivery.ID
	current, err := s.volumeFiles(ctx, volumeName)
	if err != nil {
		return nil, fmt.Errorf("failed to read student volume before rolling back advance: %w", err)
	}

	restored, err := advanceBackupFiles(current, backupRoot)
	if err != nil {
		return nil, err
	}

	if err := s.volumes.WriteWorkspaceVolumeFiles(ctx, volumeName, restored); err != nil {
		return nil, fmt.Errorf("failed to restore advance backup into student volume: %w", err)
	}

	// Reabrir el destinatario es lo que permite volver a aplicar. Si falla, el
	// respaldo se conserva y el alumno puede reintentar deshacer.
	if err := s.store.ReopenRecipient(ctx, tenantID, studentID, delivery.ID); err != nil {
		return nil, err
	}

	s.dropAdvanceBackup(ctx, volumeName, backupRoot)

	paths := make([]string, 0, len(restored))
	for path := range restored {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	s.recordAudit(ctx, tenantID, studentID, domain.AuditClassAdvanceRollback, delivery.SessionID, map[string]any{
		"advance_id":     delivery.ID,
		"exercise_id":    delivery.ExerciseID,
		"files_restored": len(paths),
		"backup_path":    backupRoot,
	})

	return &AdvanceRollbackResult{
		AdvanceID:     delivery.ID,
		FilesRestored: len(paths),
		Restored:      paths,
		BackupPath:    backupRoot,
	}, nil
}

// advanceBackupFiles extrae del volumen los archivos del respaldo y los
// devuelve con su ruta original, lista para volver a escribirlos. Sin respaldo
// no hay nada que deshacer: el avance solo agregó archivos nuevos.
func advanceBackupFiles(current map[string]string, backupRoot string) (map[string]string, error) {
	prefix := backupRoot + "/"
	restored := make(map[string]string)
	for path, content := range current {
		if !strings.HasPrefix(path, prefix) {
			continue
		}
		original, err := normalizeAdvancePath(strings.TrimPrefix(path, prefix))
		if err != nil {
			continue
		}
		restored[original] = content
	}
	if len(restored) == 0 {
		return nil, domain.ErrAdvanceBackupMissing
	}
	return restored, nil
}

// dropAdvanceBackup borra la carpeta del respaldo ya restaurado. Es
// best-effort: si el almacén no ofrece la poda, el respaldo queda en disco y no
// afecta al rollback que ya terminó.
func (s *ClassAdvanceService) dropAdvanceBackup(ctx context.Context, volumeName, backupRoot string) {
	pruner, ok := s.volumes.(domain.WorkspaceBackupPruner)
	if !ok {
		return
	}
	if err := pruner.DeleteWorkspaceVolumePaths(ctx, volumeName, []string{backupRoot}); err != nil {
		log.Printf("[Advance Rollback] no se pudo borrar el respaldo %s de %s: %v", backupRoot, volumeName, err)
	}
}

// selectAdvanceFiles deja solo los archivos elegidos del snapshot. Una ruta
// fuera del avance invalida la selección completa: aplicar la mitad de lo
// pedido es peor que rechazar el checklist.
func selectAdvanceFiles(incoming map[string]string, selected []string) (map[string]string, error) {
	chosen := make(map[string]string, len(selected))
	for _, raw := range selected {
		path, err := normalizeAdvancePath(raw)
		if err != nil {
			return nil, domain.ErrAdvanceSelectionInvalid
		}
		content, ok := incoming[path]
		if !ok {
			return nil, domain.ErrAdvanceSelectionInvalid
		}
		chosen[path] = content
	}
	if len(chosen) == 0 {
		return nil, domain.ErrAdvanceSelectionInvalid
	}
	return chosen, nil
}

// normalizeAdvancePath acepta la ruta relativa que muestra el editor y
// descarta rutas absolutas, retrocesos y nombres con unidad de disco.
func normalizeAdvancePath(raw string) (string, error) {
	path := strings.ReplaceAll(strings.TrimSpace(raw), "\\", "/")
	for strings.HasPrefix(path, "./") {
		path = strings.TrimPrefix(path, "./")
	}
	if path == "" || strings.HasPrefix(path, "/") {
		return "", domain.ErrAdvanceSelectionInvalid
	}
	if len(path) >= 2 && path[1] == ':' {
		return "", domain.ErrAdvanceSelectionInvalid
	}
	for _, segment := range strings.Split(path, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return "", domain.ErrAdvanceSelectionInvalid
		}
	}
	return path, nil
}

// pruneAdvanceBackups conserva los respaldos más recientes del volumen y borra
// los que exceden el tope. Si el almacén no ofrece la capacidad de poda, no se
// hace nada: la aplicación sigue funcionando.
func (s *ClassAdvanceService) pruneAdvanceBackups(ctx context.Context, volumeName string) {
	pruner, ok := s.volumes.(domain.WorkspaceBackupPruner)
	if !ok {
		return
	}
	dirs, err := pruner.ListWorkspaceBackupDirs(ctx, volumeName, advanceBackupRoot)
	if err != nil {
		log.Printf("[Advance Backup] no se pudieron listar los respaldos de %s: %v", volumeName, err)
		return
	}
	if len(dirs) <= maxAdvanceBackups {
		return
	}
	stale := dirs[maxAdvanceBackups:]
	if err := pruner.DeleteWorkspaceVolumePaths(ctx, volumeName, stale); err != nil {
		log.Printf("[Advance Backup] no se pudieron podar %d respaldos de %s: %v", len(stale), volumeName, err)
	}
}

// workspaceVolumeName arma el nombre del volumen del alumno en la materia, que
// es el mismo que usa el ciclo de vida de los entornos.
func workspaceVolumeName(workspace *domain.WorkspaceInstance) string {
	return workspace.VolumeName()
}

// studentWorkspace elige el entorno del alumno para el avance. Con práctica se
// prefiere el de esa práctica (ADR-059), que es el que el alumno abre hoy; el
// entorno por materia queda como respaldo de laboratorios anteriores.
func (s *ClassAdvanceService) studentWorkspace(ctx context.Context, studentID string, delivery *domain.AdvanceDelivery) (*domain.WorkspaceInstance, error) {
	if exerciseID := strings.TrimSpace(delivery.ExerciseID); exerciseID != "" {
		if repo, ok := s.workspaces.(workspaceExerciseRepository); ok {
			workspace, err := repo.GetByStudentSubjectExercise(ctx, studentID, delivery.SubjectID, exerciseID)
			if err == nil && workspace != nil {
				return workspace, nil
			}
		}
	}
	return s.workspaces.GetByStudentAndSubject(ctx, studentID, delivery.SubjectID)
}

// volumeFiles lee el estado vigente del volumen del alumno, acotado al mismo
// tope que el respaldo. Es el lado contra el que se compara el snapshot.
func (s *ClassAdvanceService) volumeFiles(ctx context.Context, volumeName string) (map[string]string, error) {
	existing, err := s.volumes.SnapshotWorkspaceVolume(ctx, volumeName, maxAdvanceBackupBytes)
	if err != nil {
		return nil, err
	}
	current := make(map[string]string, len(existing))
	for _, file := range existing {
		current[file.Path] = string(file.Content)
	}
	return current, nil
}

// splitAdvanceChanges separa el snapshot en archivos que se agregan y archivos
// que reemplazan algo que el alumno ya tenía. El orden es estable para que la
// confirmación y el resultado se lean igual.
func splitAdvanceChanges(incoming, current map[string]string) (overwritten, added []string) {
	overwritten = make([]string, 0, len(incoming))
	added = make([]string, 0, len(incoming))
	for path := range incoming {
		if _, ok := current[path]; ok {
			overwritten = append(overwritten, path)
		} else {
			added = append(added, path)
		}
	}
	sort.Strings(overwritten)
	sort.Strings(added)
	return overwritten, added
}

// readSnapshot entrega el código emitido, verificado contra su hash. Sin
// archivos no hay nada que copiar, y copiar un snapshot incompleto sería peor
// que rechazarlo.
func (s *ClassAdvanceService) readSnapshot(ctx context.Context, delivery *domain.AdvanceDelivery) (map[string]string, error) {
	files, err := s.snapshots.Files(ctx, delivery.SnapshotKey, delivery.SnapshotSHA256, maxAdvanceApplyFiles)
	if err != nil {
		if errors.Is(err, domain.ErrAdvanceSnapshotTooLarge) {
			return nil, err
		}
		return nil, fmt.Errorf("%w: %v", domain.ErrAdvanceSnapshotInvalid, err)
	}
	if len(files) == 0 {
		return nil, domain.ErrAdvanceSnapshotInvalid
	}
	incoming := make(map[string]string, len(files))
	for _, file := range files {
		incoming[file.Path] = string(file.Content)
	}
	return incoming, nil
}

// decideBestEffort deja constancia de un fallo de copia sin tapar el error
// original: si el registro falla, el aviso sigue pendiente y se puede
// reintentar.
func (s *ClassAdvanceService) decideBestEffort(ctx context.Context, tenantID, studentID, advanceID, status string, backupPath *string) {
	_ = s.store.DecideRecipient(ctx, tenantID, studentID, advanceID, status, backupPath)
}

// requireStudentIdentity exige institución y alumno con identificador válido.
func requireStudentIdentity(tenantID, studentID string) error {
	if strings.TrimSpace(tenantID) == "" {
		return domain.ErrInvalidTenant
	}
	if _, err := uuid.Parse(strings.TrimSpace(studentID)); err != nil {
		return domain.ErrAdvanceNotFound
	}
	return nil
}
