package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
)

type queueJobRepo struct {
	job *domain.DependencyPreparationJob
}

func (r *queueJobRepo) Enqueue(_ context.Context, job *domain.DependencyPreparationJob) (*domain.DependencyPreparationJob, error) {
	stored := *job
	if stored.ID == "" {
		stored.ID = "job-1"
	}
	r.job = &stored
	return r.job, nil
}

func (r *queueJobRepo) ClaimNext(_ context.Context) (*domain.DependencyPreparationJob, error) {
	if r.job == nil || r.job.Status != domain.DependencyPreparationJobPending {
		return nil, nil
	}
	claimed := *r.job
	claimed.Status = domain.DependencyPreparationJobRunning
	r.job = &claimed
	return &claimed, nil
}

func (r *queueJobRepo) UpdateResult(
	_ context.Context,
	_ string,
	status domain.DependencyPreparationJobStatus,
	result domain.DependencyPreparationResult,
) error {
	if r.job == nil {
		return domain.ErrDependencyPreparationJobNotFound
	}
	r.job.Status = status
	r.job.Message = result.Message
	if result.Key != "" {
		r.job.Key = result.Key
	}
	r.job.TreePath = result.Dir
	r.job.Bytes = result.Bytes
	if result.Attempts > 0 {
		r.job.Attempts = result.Attempts
	}
	return nil
}

func (r *queueJobRepo) GetByVersion(_ context.Context, _, _ string) (*domain.DependencyPreparationJob, error) {
	if r.job == nil {
		return nil, domain.ErrDependencyPreparationJobNotFound
	}
	return r.job, nil
}

func (r *queueJobRepo) ListActiveKeys(_ context.Context) (map[string]bool, error) {
	return map[string]bool{}, nil
}

func (r *queueJobRepo) ListVersionsWithoutJob(_ context.Context, _ int) ([]domain.DependencyPreparationJobCandidate, error) {
	return nil, nil
}

func (r *queueJobRepo) ReclaimStale(_ context.Context, _ time.Duration) (int, error) {
	return 0, nil
}

func (r *queueJobRepo) ResetForRetry(_ context.Context, id, key string) error {
	if r.job == nil || r.job.ID != id {
		return domain.ErrDependencyPreparationJobNotFound
	}
	switch r.job.Status {
	case domain.DependencyPreparationJobFailed,
		domain.DependencyPreparationJobSkipped,
		domain.DependencyPreparationJobPrepared:
	default:
		return domain.ErrDependencyPreparationNotRetryable
	}
	r.job.Status = domain.DependencyPreparationJobPending
	r.job.Key = key
	r.job.Message = ""
	r.job.TreePath = ""
	r.job.Bytes = 0
	r.job.Attempts = 0
	return nil
}

type queueTargetResolver struct {
	target *domain.DependencyPreparationTarget
	err    error
}

func (r *queueTargetResolver) ResolveDependencyPreparationTarget(
	_ context.Context, _, _ string,
) (*domain.DependencyPreparationTarget, error) {
	return r.target, r.err
}

// queueVersionRepo implementa el contrato completo para no tener que simular la
// persistencia: la cola solo usa GetVersion.
type queueVersionRepo struct {
	version *domain.WorkspaceProjectVersion
}

func (r *queueVersionRepo) GetProjectByExercise(_ context.Context, _, _ string) (*domain.WorkspaceProject, error) {
	return nil, domain.ErrWorkspaceProjectNotFound
}

func (r *queueVersionRepo) CreateProject(_ context.Context, _ *domain.WorkspaceProject) (*domain.WorkspaceProject, error) {
	return nil, errors.New("no implementado")
}

func (r *queueVersionRepo) GetDraftByProject(_ context.Context, _, _ string) (*domain.WorkspaceProjectDraft, error) {
	return nil, domain.ErrWorkspaceProjectNotFound
}

func (r *queueVersionRepo) SaveDraft(_ context.Context, _ *domain.WorkspaceProjectDraft, _ int64) (*domain.WorkspaceProjectDraft, error) {
	return nil, errors.New("no implementado")
}

func (r *queueVersionRepo) Publish(_ context.Context, _, _, _ string) (*domain.WorkspaceProjectVersion, error) {
	return nil, errors.New("no implementado")
}

func (r *queueVersionRepo) ListVersions(_ context.Context, _, _ string) ([]*domain.WorkspaceProjectVersion, error) {
	return nil, nil
}

func (r *queueVersionRepo) GetVersion(_ context.Context, _, _ string) (*domain.WorkspaceProjectVersion, error) {
	if r.version == nil {
		return nil, domain.ErrWorkspaceProjectNotFound
	}
	return r.version, nil
}

func (r *queueVersionRepo) GetLegacyReference(_ context.Context, _, _ string) (*domain.WorkspaceLegacyProjectReference, error) {
	return nil, domain.ErrWorkspaceProjectNotFound
}

// queueRunner comprueba que los archivos del proyecto estén escritos antes de
// instalar: es la condición para que el gestor encuentre el manifiesto.
type queueRunner struct {
	installs   int
	verifies   int
	installErr error
	seenFile   bool
}

func (r *queueRunner) RunInstall(_ context.Context, _ domain.DependencyPreparationRequest, dir string, _ io.Writer) error {
	r.installs++
	if info, err := os.Stat(filepath.Join(dir, "requirements.txt")); err == nil && !info.IsDir() {
		r.seenFile = true
	}
	return r.installErr
}

func (r *queueRunner) RunVerify(_ context.Context, _ domain.DependencyPreparationRequest, _ string, _ io.Writer) error {
	r.verifies++
	return nil
}

func queueTestVersion() *domain.WorkspaceProjectVersion {
	files := []domain.ProjectFile{{Path: "requirements.txt", Content: "flask==3.0.0\n"}}
	return &domain.WorkspaceProjectVersion{
		ID:           "ver-1",
		TenantID:     "tenant-1",
		ProjectID:    "proj-1",
		ExerciseID:   "ex-1",
		TextSnapshot: domain.EncodeProjectSnapshot(files),
	}
}

func queueTestTarget() *domain.DependencyPreparationTarget {
	return &domain.DependencyPreparationTarget{
		ComponentID:   "comp-1",
		ComponentSlug: "python",
		BaseImage:     "python:3.12-slim-bookworm",
		Profile: &domain.DependencyProfile{
			Version:         1,
			Detect:          []string{"requirements.txt"},
			Lockfiles:       []string{"requirements.txt"},
			Install:         "pip install -r requirements.txt",
			Materialize:     domain.DependencyMaterializeTree,
			DependencyPaths: []string{".venv"},
			Verify:          "python -c 'import flask'",
			Network:         "build",
		},
	}
}

func newTestQueue(
	t *testing.T,
	runner domain.DependencyPreparationRunnerPort,
	target *domain.DependencyPreparationTarget,
) (*DependencyPreparationQueue, *queueJobRepo) {
	t.Helper()
	worker, err := NewDependencyPreparationWorker(runner, t.TempDir(), domain.DependencyPreparationLimits{
		RetryBackoff: time.Millisecond,
		// El techo real es de 10 GB; acá solo se comprueba el flujo.
		MinFreeBytes: 1,
	})
	if err != nil {
		t.Fatalf("no se pudo crear el worker: %v", err)
	}
	jobs := &queueJobRepo{}
	queue, err := NewDependencyPreparationQueue(
		jobs,
		&queueTargetResolver{target: target},
		&queueVersionRepo{version: queueTestVersion()},
		worker,
	)
	if err != nil {
		t.Fatalf("no se pudo crear la cola: %v", err)
	}
	return queue, jobs
}

func TestDependencyPreparationQueueEnqueuesConfiguredProfile(t *testing.T) {
	queue, jobs := newTestQueue(t, &queueRunner{}, queueTestTarget())

	job, err := queue.Enqueue(context.Background(), "tenant-1", "ex-1", "ver-1")
	if err != nil {
		t.Fatalf("el encolado no debía fallar: %v", err)
	}
	if job.Status != domain.DependencyPreparationJobPending {
		t.Fatalf("el trabajo debía quedar pendiente, obtuvo %s", job.Status)
	}
	if len(job.Key) != 64 {
		t.Fatalf("la clave debía ser un sha256 de 64 caracteres, obtuvo %q", job.Key)
	}
	if jobs.job == nil || jobs.job.ComponentID != "comp-1" {
		t.Fatalf("el trabajo debía guardar el componente del catálogo, obtuvo %+v", jobs.job)
	}
}

func TestDependencyPreparationQueueSkipsVersionWithoutProfile(t *testing.T) {
	queue, _ := newTestQueue(t, &queueRunner{}, nil)

	job, err := queue.Enqueue(context.Background(), "tenant-1", "ex-1", "ver-1")
	if err != nil {
		t.Fatalf("sin perfil el encolado no es un error: %v", err)
	}
	if job.Status != domain.DependencyPreparationJobSkipped {
		t.Fatalf("sin perfil el trabajo debía quedar omitido, obtuvo %s", job.Status)
	}
	if job.Message == "" {
		t.Fatal("el trabajo omitido debe explicar por qué no se prepara")
	}
}

func TestDependencyPreparationQueuePreparesVersionAndKeepsProjectFiles(t *testing.T) {
	runner := &queueRunner{}
	queue, jobs := newTestQueue(t, runner, queueTestTarget())

	if _, err := queue.Enqueue(context.Background(), "tenant-1", "ex-1", "ver-1"); err != nil {
		t.Fatalf("el encolado no debía fallar: %v", err)
	}
	job, processed, err := queue.ProcessNext(context.Background())
	if err != nil {
		t.Fatalf("la preparación no debía fallar: %v", err)
	}
	if !processed || job == nil {
		t.Fatal("debía procesar el trabajo encolado")
	}
	if job.Status != domain.DependencyPreparationJobPrepared {
		t.Fatalf("el trabajo debía quedar preparado, obtuvo %s", job.Status)
	}
	if !runner.seenFile {
		t.Fatal("el manifiesto no estaba en el árbol cuando corrió la instalación")
	}
	if runner.installs != 1 || runner.verifies != 1 {
		t.Fatalf("esperaba una instalación y una comprobación, obtuvo %d y %d", runner.installs, runner.verifies)
	}
	if jobs.job == nil || jobs.job.TreePath == "" {
		t.Fatal("el trabajo debía guardar la ruta del árbol preparado")
	}
	if _, err := os.Stat(filepath.Join(jobs.job.TreePath, "requirements.txt")); err != nil {
		t.Fatalf("el árbol preparado debía contener el proyecto: %v", err)
	}
}

func TestDependencyPreparationQueuePersistsFailure(t *testing.T) {
	queue, jobs := newTestQueue(t, &queueRunner{installErr: errors.New("comando inválido")}, queueTestTarget())

	if _, err := queue.Enqueue(context.Background(), "tenant-1", "ex-1", "ver-1"); err != nil {
		t.Fatalf("el encolado no debía fallar: %v", err)
	}
	job, processed, err := queue.ProcessNext(context.Background())
	if err == nil {
		t.Fatal("un fallo de instalación debe devolver error")
	}
	if !processed || job == nil {
		t.Fatal("el trabajo se procesó y debe informarse")
	}
	if job.Status != domain.DependencyPreparationJobFailed {
		t.Fatalf("el trabajo debía quedar fallido, obtuvo %s", job.Status)
	}
	if jobs.job == nil || jobs.job.Status != domain.DependencyPreparationJobFailed {
		t.Fatalf("el estado fallido debía persistirse, obtuvo %+v", jobs.job)
	}
}

func TestPreparedDependenciesReturnsPreparedTree(t *testing.T) {
	queue, jobs := newTestQueue(t, &queueRunner{}, queueTestTarget())
	treePath := t.TempDir()
	jobs.job = &domain.DependencyPreparationJob{
		ID:        "job-1",
		TenantID:  "tenant-1",
		VersionID: "ver-1",
		Status:    domain.DependencyPreparationJobPrepared,
		Key:       strings.Repeat("a", 64),
		TreePath:  treePath,
	}

	tree, found, err := queue.PreparedDependencies(context.Background(), "tenant-1", "ver-1")
	if err != nil {
		t.Fatalf("el árbol preparado debía resolverse: %v", err)
	}
	if !found || tree == nil {
		t.Fatal("una versión preparada debía informar su árbol")
	}
	if tree.RootPath != treePath || tree.Key != strings.Repeat("a", 64) {
		t.Fatalf("el árbol debía apuntar a la preparación, obtuvo %+v", tree)
	}
	if len(tree.Paths) != 1 || tree.Paths[0] != ".venv" {
		t.Fatalf("las rutas debían salir del perfil, obtuvo %v", tree.Paths)
	}
}

func TestPreparedDependenciesIgnoresUnpreparedJob(t *testing.T) {
	queue, jobs := newTestQueue(t, &queueRunner{}, queueTestTarget())
	jobs.job = &domain.DependencyPreparationJob{
		ID:        "job-1",
		TenantID:  "tenant-1",
		VersionID: "ver-1",
		Status:    domain.DependencyPreparationJobFailed,
		Key:       strings.Repeat("a", 64),
		TreePath:  t.TempDir(),
	}

	tree, found, err := queue.PreparedDependencies(context.Background(), "tenant-1", "ver-1")
	if err != nil {
		t.Fatalf("un trabajo fallido no es un error de resolución: %v", err)
	}
	if found || tree != nil {
		t.Fatal("un trabajo fallido no tiene árbol que enlazar")
	}
}

func TestPrepareTreeRootAssignsDeclaredOwner(t *testing.T) {
	owner, err := preparationOwnerFromUser(fmt.Sprintf("%d:%d", os.Getuid(), os.Getgid()))
	if err != nil {
		t.Fatalf("el usuario del proceso debe ser válido: %v", err)
	}
	dir := filepath.Join(t.TempDir(), "tree")
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatalf("no se pudo crear el directorio: %v", err)
	}
	if err := prepareTreeRoot(dir, owner); err != nil {
		t.Fatalf("no se pudo asignar el dueño: %v", err)
	}
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("no se pudo leer el directorio: %v", err)
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		t.Fatal("no se pudo leer el dueño del directorio")
	}
	if int(stat.Uid) != os.Getuid() || int(stat.Gid) != os.Getgid() {
		t.Fatalf("dueño inesperado: %d:%d", stat.Uid, stat.Gid)
	}
	if info.Mode().Perm()&0o020 == 0 {
		t.Fatalf("el grupo debe poder escribir en el árbol, obtuvo %o", info.Mode().Perm())
	}
}

func TestWritePreparationFilesRejectsEscapingPaths(t *testing.T) {
	root := t.TempDir()
	err := writePreparationFiles(root, []domain.WorkspaceSeedFile{
		{Path: "../fuera.txt", Content: "x"},
	}, preparationOwner{})
	if err == nil {
		t.Fatal("una ruta que sale del árbol debe rechazarse")
	}
	if _, statErr := os.Stat(filepath.Join(root, "..", "fuera.txt")); statErr == nil {
		t.Fatal("no debía escribirse nada fuera del árbol")
	}
}

func TestDependencyPreparationQueueRetryRequeuesFailedVersion(t *testing.T) {
	queue, jobs := newTestQueue(t, &queueRunner{installErr: errors.New("comando inválido")}, queueTestTarget())
	ctx := context.Background()
	if _, err := queue.Enqueue(ctx, "tenant-1", "ex-1", "ver-1"); err != nil {
		t.Fatalf("el encolado no debía fallar: %v", err)
	}
	if _, _, err := queue.ProcessNext(ctx); err == nil {
		t.Fatal("la instalación debía fallar")
	}
	if jobs.job == nil || jobs.job.Status != domain.DependencyPreparationJobFailed {
		t.Fatalf("el trabajo debía quedar fallido, obtuvo %+v", jobs.job)
	}
	previousKey := jobs.job.Key

	// El administrador corrige el perfil y sube su versión: la clave tiene que
	// reflejarlo, o el reintento reusaría lo que falló con el comando viejo.
	corrected := queueTestTarget()
	corrected.Profile.Version = 2
	corrected.Profile.Install = "pip install --no-cache-dir -r requirements.txt"
	queue.targets = &queueTargetResolver{target: corrected}

	job, err := queue.Retry(ctx, "tenant-1", "ex-1", "ver-1")
	if err != nil {
		t.Fatalf("el reintento no debía fallar: %v", err)
	}
	if job.Status != domain.DependencyPreparationJobPending {
		t.Fatalf("el reintento debía dejar el trabajo pendiente, obtuvo %s", job.Status)
	}
	if job.Key == previousKey || job.Key == "" {
		t.Fatalf("la clave debía recalcularse con el perfil corregido: %q", job.Key)
	}
	if job.Message != "" || job.Attempts != 0 {
		t.Fatalf("el reintento no arrastra el desenlace anterior: %+v", job)
	}
}

func TestDependencyPreparationQueueRetryKeepsSkippedVersionSkipped(t *testing.T) {
	queue, _ := newTestQueue(t, &queueRunner{}, nil)
	ctx := context.Background()
	if _, err := queue.Enqueue(ctx, "tenant-1", "ex-1", "ver-1"); err != nil {
		t.Fatalf("sin perfil el encolado no es un error: %v", err)
	}

	job, err := queue.Retry(ctx, "tenant-1", "ex-1", "ver-1")
	if err != nil {
		t.Fatalf("el reintento sin perfil no es un error: %v", err)
	}
	if job.Status != domain.DependencyPreparationJobSkipped {
		t.Fatalf("sin perfil la versión sigue omitida, obtuvo %s", job.Status)
	}
	if job.Message == "" {
		t.Fatal("el reintento omitido debe conservar el motivo")
	}
}

func TestDependencyPreparationQueueRetryRejectsJobInProgress(t *testing.T) {
	queue, jobs := newTestQueue(t, &queueRunner{}, queueTestTarget())
	jobs.job = &domain.DependencyPreparationJob{
		ID:        "job-1",
		TenantID:  "tenant-1",
		VersionID: "ver-1",
		ProjectID: "proj-1",
		Status:    domain.DependencyPreparationJobPending,
	}

	_, err := queue.Retry(context.Background(), "tenant-1", "ex-1", "ver-1")
	if !errors.Is(err, domain.ErrDependencyPreparationNotRetryable) {
		t.Fatalf("un trabajo en cola no se reintenta, obtuvo %v", err)
	}
}

func TestDependencyPreparationQueueRetryRepreparesWhenTreeIsGone(t *testing.T) {
	queue, jobs := newTestQueue(t, &queueRunner{}, queueTestTarget())
	ctx := context.Background()
	if _, err := queue.Enqueue(ctx, "tenant-1", "ex-1", "ver-1"); err != nil {
		t.Fatalf("el encolado no debía fallar: %v", err)
	}
	if _, _, err := queue.ProcessNext(ctx); err != nil {
		t.Fatalf("la preparación no debía fallar: %v", err)
	}
	if jobs.job == nil || jobs.job.Status != domain.DependencyPreparationJobPrepared {
		t.Fatalf("el trabajo debía quedar preparado, obtuvo %+v", jobs.job)
	}

	preparation, err := queue.Preparation(ctx, "tenant-1", "ver-1")
	if err != nil || preparation == nil {
		t.Fatalf("la vista debía existir: %v", err)
	}
	if preparation.CanRetry {
		t.Fatal("con el árbol en disco no hay nada que reintentar")
	}

	// El árbol desapareció del disco: el estado dice preparado y el enlace ya no
	// tendría de dónde copiar.
	if err := os.RemoveAll(jobs.job.TreePath); err != nil {
		t.Fatalf("no se pudo borrar el árbol: %v", err)
	}
	preparation, err = queue.Preparation(ctx, "tenant-1", "ver-1")
	if err != nil || preparation == nil {
		t.Fatalf("la vista debía seguir existiendo: %v", err)
	}
	if !preparation.CanRetry || preparation.Status != domain.DependencyPreparationJobPrepared {
		t.Fatalf("sin árbol la vista debía ofrecer reintentar y conservar el estado: %+v", preparation)
	}

	job, err := queue.Retry(ctx, "tenant-1", "ex-1", "ver-1")
	if err != nil {
		t.Fatalf("sin árbol el reintento debía aceptarse: %v", err)
	}
	if job.Status != domain.DependencyPreparationJobPending {
		t.Fatalf("el reintento debía dejar el trabajo pendiente, obtuvo %s", job.Status)
	}
}

func TestDependencyPreparationQueueRuntimeEnvPointsToThePreparedStore(t *testing.T) {
	target := queueTestTarget()
	target.Profile.RuntimeEnv = map[string]string{
		"NUGET_PACKAGES": "/home/workspace/.solv-cache/nuget",
		"GOMODCACHE":     "/home/workspace/.solv-cache/go-mod",
	}
	queue, _ := newTestQueue(t, &queueRunner{}, target)

	env, err := queue.RuntimeEnv(context.Background(), "tenant-1", "ex-1")
	if err != nil {
		t.Fatalf("el entorno del aula no debía fallar: %v", err)
	}
	want := []string{
		"GOMODCACHE=/home/workspace/.solv-cache/go-mod",
		"NUGET_PACKAGES=/home/workspace/.solv-cache/nuget",
	}
	if len(env) != len(want) || env[0] != want[0] || env[1] != want[1] {
		t.Fatalf("el entorno debía apuntar al almacén entregado y en orden: %v", env)
	}
}

func TestDependencyPreparationQueueRuntimeEnvIgnoresCacheDelivery(t *testing.T) {
	target := queueTestTarget()
	target.Profile.Materialize = domain.DependencyMaterializeCache
	target.Profile.RuntimeEnv = map[string]string{"GOMODCACHE": "/opt/solv/cache/go-mod"}
	queue, _ := newTestQueue(t, &queueRunner{}, target)

	env, err := queue.RuntimeEnv(context.Background(), "tenant-1", "ex-1")
	if err != nil {
		t.Fatalf("el entorno del aula no debía fallar: %v", err)
	}
	if len(env) != 0 {
		t.Fatalf("sin entrega de árbol no hay ruta que apuntar: %v", env)
	}
}

func TestDependencyPreparationPreparationViewIsAbsentWithoutRecord(t *testing.T) {
	queue, _ := newTestQueue(t, &queueRunner{}, queueTestTarget())

	preparation, err := queue.Preparation(context.Background(), "tenant-1", "ver-1")
	if err != nil {
		t.Fatalf("la falta de trabajo no es un error: %v", err)
	}
	if preparation != nil {
		t.Fatalf("una versión sin trabajo no tiene estado que mostrar, obtuvo %+v", preparation)
	}
}
