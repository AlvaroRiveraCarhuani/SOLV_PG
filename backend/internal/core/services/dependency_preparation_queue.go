package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"
	"time"

	"solv-backend/internal/core/domain"
)

// dependencyPreparationProjectPath es la ruta que ve el contenedor y tiene que
// ser la misma que ve el alumno: el árbol se monta acá como único punto de
// escritura, y el volumen del entorno se monta en esa misma ruta al abrirlo. Si
// no coinciden, todo lo que un gestor hornee en rutas absolutas —los scripts del
// venv, el archivo de activos de .NET, la caché de CMake— apunta a un directorio
// que no existe en el aula y falla sin explicar por qué.
const dependencyPreparationProjectPath = "/home/workspace"

// DependencyPreparationQueue convierte una versión publicada en un trabajo y lo
// ejecuta con el worker del nodo. La cola vive en la base: un reinicio no pierde
// lo que todavía no se preparó.
type DependencyPreparationQueue struct {
	jobs     domain.DependencyPreparationJobRepository
	targets  domain.DependencyPreparationTargetResolver
	versions domain.WorkspaceProjectRepository
	worker   *DependencyPreparationWorker
}

func NewDependencyPreparationQueue(
	jobs domain.DependencyPreparationJobRepository,
	targets domain.DependencyPreparationTargetResolver,
	versions domain.WorkspaceProjectRepository,
	worker *DependencyPreparationWorker,
) (*DependencyPreparationQueue, error) {
	if jobs == nil {
		return nil, errors.New("la cola de preparación necesita el repositorio de trabajos")
	}
	if targets == nil {
		return nil, errors.New("la cola de preparación necesita el resolvedor del catálogo")
	}
	if versions == nil {
		return nil, errors.New("la cola de preparación necesita el repositorio de versiones")
	}
	if worker == nil {
		return nil, errors.New("la cola de preparación necesita el worker")
	}
	return &DependencyPreparationQueue{jobs: jobs, targets: targets, versions: versions, worker: worker},
		nil
}

// Enqueue registra la preparación de una versión publicada. Sin perfil la fila
// nace `skipped`: el docente ve que la versión no prepara nada en lugar de
// esperar algo que nunca iba a ocurrir. Es idempotente por versión.
func (q *DependencyPreparationQueue) Enqueue(
	ctx context.Context,
	tenantID, exerciseID, versionID string,
) (*domain.DependencyPreparationJob, error) {
	version, err := q.versions.GetVersion(ctx, tenantID, versionID)
	if err != nil {
		return nil, err
	}
	job := &domain.DependencyPreparationJob{
		TenantID:  tenantID,
		VersionID: version.ID,
		ProjectID: version.ProjectID,
		Status:    domain.DependencyPreparationJobPending,
	}

	target, err := q.targets.ResolveDependencyPreparationTarget(ctx, tenantID, exerciseID)
	if err != nil {
		return nil, err
	}
	if target == nil || target.Profile == nil || target.Profile.IsEmpty() {
		job.Status = domain.DependencyPreparationJobSkipped
		job.Message = "el entorno de la versión no declara perfil de dependencias"
		return q.jobs.Enqueue(ctx, job)
	}
	if target.Profile.Materialize == domain.DependencyMaterializeNone {
		job.Status = domain.DependencyPreparationJobSkipped
		job.Message = "el perfil declara que la familia no resuelve dependencias de proyecto"
		return q.jobs.Enqueue(ctx, job)
	}

	request, err := q.buildRequest(version, target)
	if err != nil {
		return nil, err
	}
	job.ComponentID = target.ComponentID
	job.Key = request.Key()
	return q.jobs.Enqueue(ctx, job)
}

// EnqueueMissing encola las versiones publicadas que no tienen trabajo. Se corre
// al arrancar: cubre la publicación cuyo encolado no llegó a guardarse.
func (q *DependencyPreparationQueue) EnqueueMissing(ctx context.Context, limit int) (int, error) {
	candidates, err := q.jobs.ListVersionsWithoutJob(ctx, limit)
	if err != nil {
		return 0, err
	}
	created := 0
	for _, candidate := range candidates {
		if _, err := q.Enqueue(ctx, candidate.TenantID, candidate.ExerciseID, candidate.VersionID); err != nil {
			return created, err
		}
		created++
	}
	return created, nil
}

// ProcessNext toma un trabajo pendiente y lo lleva hasta su estado final. El
// desenlace siempre se persiste, incluso cuando la preparación falla: un
// trabajo que queda en `running` para siempre no se distingue de uno colgado.
func (q *DependencyPreparationQueue) ProcessNext(
	ctx context.Context,
) (*domain.DependencyPreparationJob, bool, error) {
	job, err := q.jobs.ClaimNext(ctx)
	if err != nil {
		return nil, false, err
	}
	if job == nil {
		return nil, false, nil
	}

	fail := func(cause error) (*domain.DependencyPreparationJob, bool, error) {
		result := domain.DependencyPreparationResult{Message: cause.Error()}
		if updateErr := q.jobs.UpdateResult(ctx, job.ID, domain.DependencyPreparationJobFailed, result); updateErr != nil {
			return job, true, errors.Join(cause, updateErr)
		}
		job.Status = domain.DependencyPreparationJobFailed
		job.Message = cause.Error()
		return job, true, cause
	}

	version, err := q.versions.GetVersion(ctx, job.TenantID, job.VersionID)
	if err != nil {
		return fail(err)
	}
	target, err := q.targets.ResolveDependencyPreparationTarget(ctx, job.TenantID, version.ExerciseID)
	if err != nil {
		return fail(err)
	}
	if target == nil || target.Profile == nil || target.Profile.IsEmpty() ||
		target.Profile.Materialize == domain.DependencyMaterializeNone {
		reason := "el entorno de la versión ya no declara perfil de dependencias"
		result := domain.DependencyPreparationResult{Message: reason}
		if updateErr := q.jobs.UpdateResult(ctx, job.ID, domain.DependencyPreparationJobSkipped, result); updateErr != nil {
			return job, true, updateErr
		}
		job.Status = domain.DependencyPreparationJobSkipped
		job.Message = reason
		return job, true, nil
	}

	request, err := q.buildRequest(version, target)
	if err != nil {
		return fail(err)
	}

	result, prepareErr := q.worker.Prepare(ctx, request)
	status := domain.DependencyPreparationJobPrepared
	if prepareErr != nil {
		status = domain.DependencyPreparationJobFailed
	}
	if updateErr := q.jobs.UpdateResult(ctx, job.ID, status, result); updateErr != nil {
		return job, true, errors.Join(prepareErr, updateErr)
	}
	job.Status = status
	job.Message = result.Message
	job.Key = result.Key
	job.TreePath = result.Dir
	job.Bytes = result.Bytes
	job.LogPath = result.LogPath
	job.Attempts = result.Attempts
	return job, true, prepareErr
}

// PreparedDependencies entrega el árbol de una versión ya preparada para el
// enlace al volumen. Solo cuentan los trabajos en `prepared`: uno fallido o
// pendiente no tiene árbol que prometer.
func (q *DependencyPreparationQueue) PreparedDependencies(
	ctx context.Context,
	tenantID, versionID string,
) (*domain.WorkspaceDependencyTree, bool, error) {
	job, err := q.jobs.GetByVersion(ctx, tenantID, versionID)
	if errors.Is(err, domain.ErrDependencyPreparationJobNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if job.Status != domain.DependencyPreparationJobPrepared || job.TreePath == "" || job.Key == "" {
		return nil, false, nil
	}
	version, err := q.versions.GetVersion(ctx, tenantID, versionID)
	if err != nil {
		return nil, false, err
	}
	target, err := q.targets.ResolveDependencyPreparationTarget(ctx, tenantID, version.ExerciseID)
	if err != nil {
		return nil, false, err
	}
	if target == nil || target.Profile == nil || len(target.Profile.DependencyPaths) == 0 {
		return nil, false, nil
	}
	tree := &domain.WorkspaceDependencyTree{
		RootPath: job.TreePath,
		Key:      job.Key,
		Paths:    target.Profile.DependencyPaths,
	}
	if err := tree.Validate(); err != nil {
		return nil, false, err
	}
	return tree, true, nil
}

// WorkspacePreparationView es el estado de la preparación que ve el docente.
// No incluye la ruta del árbol ni el log: son datos del nodo, no de la pantalla.
type WorkspacePreparationView struct {
	Status    domain.DependencyPreparationJobStatus `json:"status"`
	Attempts  int                                   `json:"attempts"`
	Message   string                                `json:"message,omitempty"`
	Bytes     int64                                 `json:"bytes"`
	UpdatedAt time.Time                             `json:"updated_at"`
	CanRetry  bool                                  `json:"can_retry"`
}

// Job devuelve el trabajo de una versión, o nil cuando nunca se encoló.
func (q *DependencyPreparationQueue) Job(
	ctx context.Context,
	tenantID, versionID string,
) (*domain.DependencyPreparationJob, error) {
	job, err := q.jobs.GetByVersion(ctx, tenantID, versionID)
	if errors.Is(err, domain.ErrDependencyPreparationJobNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return job, nil
}

// CanRetry dice si la versión admite un reintento ahora. Un trabajo preparado
// cuyo árbol ya no está también: el estado dice preparado y el enlace no
// tendría de dónde copiar. Un pendiente o en curso no se toca: ya hay trabajo.
func (q *DependencyPreparationQueue) CanRetry(job *domain.DependencyPreparationJob) bool {
	if job == nil {
		return false
	}
	if job.Retryable() {
		return true
	}
	return job.Status == domain.DependencyPreparationJobPrepared && !q.worker.TreePresent(job.TreePath)
}

// Preparation arma la vista del estado. Sin trabajo devuelve nil: la versión se
// publicó antes de que la cola existiera, y eso no es un error.
func (q *DependencyPreparationQueue) Preparation(
	ctx context.Context,
	tenantID, versionID string,
) (*WorkspacePreparationView, error) {
	job, err := q.Job(ctx, tenantID, versionID)
	if err != nil || job == nil {
		return nil, err
	}
	return &WorkspacePreparationView{
		Status:    job.Status,
		Attempts:  job.Attempts,
		Message:   job.Message,
		Bytes:     job.Bytes,
		UpdatedAt: job.UpdatedAt,
		CanRetry:  q.CanRetry(job),
	}, nil
}

// RuntimeEnv devuelve las variables que el aula necesita para encontrar el
// almacén en la misma ruta donde se preparó. Solo aplica a las familias que
// entregan un árbol: sin entrega no hay ruta que apuntar. El orden es estable
// para que la configuración del contenedor no cambie entre aperturas.
func (q *DependencyPreparationQueue) RuntimeEnv(
	ctx context.Context,
	tenantID, exerciseID string,
) ([]string, error) {
	target, err := q.targets.ResolveDependencyPreparationTarget(ctx, tenantID, exerciseID)
	if err != nil {
		return nil, err
	}
	if target == nil || target.Profile == nil ||
		target.Profile.Materialize != domain.DependencyMaterializeTree {
		return nil, nil
	}
	env := make([]string, 0, len(target.Profile.RuntimeEnv))
	for name, value := range target.Profile.RuntimeEnv {
		env = append(env, name+"="+value)
	}
	sort.Strings(env)
	return env, nil
}

// Retry devuelve a la cola una versión cuya preparación no terminó bien. Vuelve
// a leer el perfil y recalcula la clave: corregir el perfil en el catálogo es
// justamente lo que un reintento tiene que aprovechar. Si el perfil sigue
// ausente, la versión queda `skipped` con el motivo, sin encolar un trabajo que
// nacería omitido otra vez.
func (q *DependencyPreparationQueue) Retry(
	ctx context.Context,
	tenantID, exerciseID, versionID string,
) (*domain.DependencyPreparationJob, error) {
	version, err := q.versions.GetVersion(ctx, tenantID, versionID)
	if err != nil {
		return nil, err
	}
	if version.ExerciseID != exerciseID {
		return nil, domain.ErrDependencyPreparationJobNotFound
	}
	job, err := q.Job(ctx, tenantID, versionID)
	if err != nil {
		return nil, err
	}
	if !q.CanRetry(job) {
		return nil, fmt.Errorf("%w: la versión está en cola o ya tiene su árbol",
			domain.ErrDependencyPreparationNotRetryable)
	}

	target, err := q.targets.ResolveDependencyPreparationTarget(ctx, tenantID, exerciseID)
	if err != nil {
		return nil, err
	}
	if target == nil || target.Profile == nil || target.Profile.IsEmpty() ||
		target.Profile.Materialize == domain.DependencyMaterializeNone {
		reason := "el entorno de la versión no declara perfil de dependencias"
		if err := q.jobs.UpdateResult(ctx, job.ID, domain.DependencyPreparationJobSkipped,
			domain.DependencyPreparationResult{Message: reason}); err != nil {
			return nil, err
		}
		return q.Job(ctx, tenantID, versionID)
	}

	request, err := q.buildRequest(version, target)
	if err != nil {
		return nil, err
	}
	if err := q.jobs.ResetForRetry(ctx, job.ID, request.Key()); err != nil {
		return nil, err
	}
	return q.Job(ctx, tenantID, versionID)
}

// Usage informa cuántos árboles hay y cuánto ocupan. Se usa para el registro del
// nodo y para sostener la cuota cuando la decisión 15 fije el número.
func (q *DependencyPreparationQueue) Usage() (int, int64, error) {
	return q.worker.Usage()
}

// ReclaimStale devuelve a la cola los trabajos que un nodo tomó y no terminó.
// Se llama desde el bucle del nodo: un trabajo vencido vuelve a `pending` y el
// reconciliador no tiene que adivinar qué pasó.
func (q *DependencyPreparationQueue) ReclaimStale(ctx context.Context, olderThan time.Duration) (int, error) {
	return q.jobs.ReclaimStale(ctx, olderThan)
}

// PruneOrphans borra los árboles que ninguna versión preparada referencia. La
// lista sale de los trabajos: si una clave no está, no hay versión que la use.
func (q *DependencyPreparationQueue) PruneOrphans(ctx context.Context) ([]string, error) {
	keys, err := q.jobs.ListActiveKeys(ctx)
	if err != nil {
		return nil, err
	}
	return q.worker.PruneOrphanPreparedTrees(ctx, keys)
}

// buildRequest traduce el perfil del catálogo a una solicitud concreta. Los
// archivos salen del snapshot publicado, nunca del borrador: la preparación
// corresponde a la versión que el alumno va a recibir.
func (q *DependencyPreparationQueue) buildRequest(
	version *domain.WorkspaceProjectVersion,
	target *domain.DependencyPreparationTarget,
) (domain.DependencyPreparationRequest, error) {
	if version == nil || strings.TrimSpace(version.TextSnapshot) == "" {
		return domain.DependencyPreparationRequest{}, fmt.Errorf(
			"%w: la versión no tiene contenido", domain.ErrInvalidDependencyPreparation)
	}
	parsed, ok := domain.ParseProjectSnapshot(version.TextSnapshot, "")
	if !ok || len(parsed) == 0 {
		return domain.DependencyPreparationRequest{}, fmt.Errorf(
			"%w: el snapshot de la versión no es reconocible", domain.ErrInvalidDependencyPreparation)
	}
	files := make([]domain.WorkspaceSeedFile, 0, len(parsed))
	for _, file := range parsed {
		files = append(files, domain.WorkspaceSeedFile{Path: file.Path, Content: file.Content})
	}
	hasLockfile, lockfileHash := lockfileFingerprint(files, target.Profile.Lockfiles)

	return domain.DependencyPreparationRequest{
		TenantID:           version.TenantID,
		VersionID:          version.ID,
		ComponentSlug:      target.ComponentSlug,
		BaseImage:          target.BaseImage,
		ProfileVersion:     target.Profile.Version,
		ProjectPath:        dependencyPreparationProjectPath,
		User:               target.User,
		Home:               target.Home,
		Install:            target.Profile.Install,
		InstallWithoutLock: target.Profile.InstallWithoutLock,
		Verify:             target.Profile.Verify,
		HasLockfile:        hasLockfile,
		OfflineFlags:       target.Profile.OfflineFlags,
		CacheEnv:           target.Profile.CacheEnv,
		Network:            target.Profile.Network,
		LockfileHash:       lockfileHash,
		Files:              files,
	}, nil
}

// lockfileFingerprint devuelve si la versión trae alguno de los archivos de
// versión fija y un hash de su contenido. El hash entra en la clave: cambiar el
// lockfile cambia el árbol, aunque la versión del proyecto sea la misma.
func lockfileFingerprint(files []domain.WorkspaceSeedFile, patterns []string) (bool, string) {
	if len(patterns) == 0 {
		return false, ""
	}
	matched := make([]domain.WorkspaceSeedFile, 0, len(files))
	for _, file := range files {
		for _, pattern := range patterns {
			if ok, err := path.Match(pattern, file.Path); err == nil && ok {
				matched = append(matched, file)
				break
			}
			if ok, err := path.Match(pattern, path.Base(file.Path)); err == nil && ok {
				matched = append(matched, file)
				break
			}
		}
	}
	if len(matched) == 0 {
		return false, ""
	}
	sort.Slice(matched, func(i, j int) bool { return matched[i].Path < matched[j].Path })
	digest := sha256.New()
	for _, file := range matched {
		fmt.Fprintf(digest, "%s\n%d\n%s\n", file.Path, len(file.Content), file.Content)
	}
	return true, hex.EncodeToString(digest.Sum(nil))
}
