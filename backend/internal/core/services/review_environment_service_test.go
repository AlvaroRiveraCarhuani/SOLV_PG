package services

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
)

type fakeReviewEnvironmentRepository struct {
	env          *domain.ReviewEnvironment
	created      int
	touched      bool
	failedCode   string
	expired      bool
	activeErr    error
	getActiveErr error
}

func (f *fakeReviewEnvironmentRepository) Create(_ context.Context, env *domain.ReviewEnvironment) error {
	f.created++
	env.CreatedAt = time.Now().UTC()
	env.UpdatedAt = env.CreatedAt
	f.env = env
	return nil
}

func (f *fakeReviewEnvironmentRepository) GetActiveBySubmission(_ context.Context, _, _ string) (*domain.ReviewEnvironment, error) {
	if f.getActiveErr != nil {
		return nil, f.getActiveErr
	}
	if f.activeErr != nil {
		return nil, f.activeErr
	}
	if f.env == nil || !f.env.IsActive() {
		return nil, domain.ErrNotFound
	}
	return f.env, nil
}

func (f *fakeReviewEnvironmentRepository) ListActive(_ context.Context) ([]*domain.ReviewEnvironment, error) {
	if f.env == nil || !f.env.IsActive() {
		return nil, nil
	}
	return []*domain.ReviewEnvironment{f.env}, nil
}

func (f *fakeReviewEnvironmentRepository) SetRunning(_ context.Context, id, containerID, accessURL string, expiresAt time.Time, rxBytes, txBytes int64) error {
	f.env.ID = id
	f.env.ContainerID = &containerID
	f.env.AccessURL = accessURL
	f.env.Status = domain.ReviewEnvironmentStatusRunning
	f.env.ExpiresAt = expiresAt
	f.env.LastRxBytes = rxBytes
	f.env.LastTxBytes = txBytes
	return nil
}

func (f *fakeReviewEnvironmentRepository) Touch(_ context.Context, id string, lastActivityAt, expiresAt time.Time, rxBytes, txBytes int64) error {
	f.touched = true
	f.env.ID = id
	f.env.LastActivityAt = lastActivityAt
	f.env.ExpiresAt = expiresAt
	f.env.LastRxBytes = rxBytes
	f.env.LastTxBytes = txBytes
	return nil
}

func (f *fakeReviewEnvironmentRepository) MarkFailed(_ context.Context, id, errorCode, _ string) error {
	f.failedCode = errorCode
	f.env.ID = id
	f.env.Status = domain.ReviewEnvironmentStatusFailed
	return nil
}

func (f *fakeReviewEnvironmentRepository) MarkExpired(_ context.Context, id string) error {
	f.expired = true
	f.env.ID = id
	f.env.Status = domain.ReviewEnvironmentStatusExpired
	return nil
}

type fakeReviewEnvironmentSource struct {
	context *domain.ReviewEnvironmentContext
	err     error
	calls   int
}

func (f *fakeReviewEnvironmentSource) GetReviewEnvironmentContext(_ context.Context, _, _, _ string) (*domain.ReviewEnvironmentContext, error) {
	f.calls++
	return f.context, f.err
}

type fakeReviewEnvironmentOrchestrator struct {
	volume         string
	writtenFiles   map[string]string
	network        string
	startConfig    domain.ReviewEnvironmentContainerConfig
	startCalls     int
	startErr       error
	containerID    string
	metrics        *domain.ContainerMetrics
	logs           string
	stopped        []string
	removedVolumes []string
	removeErr      error
}

func (f *fakeReviewEnvironmentOrchestrator) EnsureVolumeExists(_ context.Context, volumeName string) error {
	f.volume = volumeName
	return nil
}

func (f *fakeReviewEnvironmentOrchestrator) EnsureICCDisabledNetworkExists(_ context.Context, networkName string) error {
	f.network = networkName
	return nil
}

func (f *fakeReviewEnvironmentOrchestrator) WritePracticeEnvironmentVolumeFiles(_ context.Context, _ string, files map[string]string) error {
	f.writtenFiles = files
	return nil
}

func (f *fakeReviewEnvironmentOrchestrator) StartReviewEnvironmentContainer(_ context.Context, config domain.ReviewEnvironmentContainerConfig) (string, error) {
	f.startCalls++
	f.startConfig = config
	if f.startErr != nil {
		return "", f.startErr
	}
	if f.containerID == "" {
		f.containerID = "container-123"
	}
	return f.containerID, nil
}

func (f *fakeReviewEnvironmentOrchestrator) GetContainerMetrics(_ context.Context, _ string) (*domain.ContainerMetrics, error) {
	return f.metrics, nil
}

func (f *fakeReviewEnvironmentOrchestrator) GetContainerLogs(_ context.Context, _ string, _ int) (string, error) {
	return f.logs, nil
}

func (f *fakeReviewEnvironmentOrchestrator) StopAndRemoveContainer(_ context.Context, containerID string) error {
	if f.removeErr != nil {
		return f.removeErr
	}
	f.stopped = append(f.stopped, containerID)
	return nil
}

func (f *fakeReviewEnvironmentOrchestrator) RemoveVolume(_ context.Context, volumeName string) error {
	if f.removeErr != nil {
		return f.removeErr
	}
	f.removedVolumes = append(f.removedVolumes, volumeName)
	return nil
}

type fakeReviewEnvironmentHostMonitor struct {
	allow bool
}

func (f *fakeReviewEnvironmentHostMonitor) GetHostMemoryStats() (float64, uint64, error) {
	return 50, 8192, nil
}

func (f *fakeReviewEnvironmentHostMonitor) CanAllocateMemory(int64) bool {
	return f.allow
}

func newReviewEnvironmentFixture(t *testing.T) (*ReviewEnvironmentService, *fakeReviewEnvironmentRepository, *fakeReviewEnvironmentSource, *fakeReviewEnvironmentOrchestrator) {
	t.Helper()
	repo := &fakeReviewEnvironmentRepository{}
	source := &fakeReviewEnvironmentSource{
		context: &domain.ReviewEnvironmentContext{
			SubmissionID:     "11111111-1111-1111-1111-111111111111",
			ExerciseID:       "22222222-2222-2222-2222-222222222222",
			EnvironmentType:  domain.PracticeEnvironmentTypeIDEPersistente,
			Code:             "# === FILE: main.js ===\nconsole.log('hola')\n\n# === FILE: lib/util.js ===\nexport const x = 1\n",
			Language:         "javascript",
			AppRunCommand:    "node main.js",
			AppPort:          3000,
			TemplateImage:    "node:22-alpine",
			TemplateMemoryMB: 512,
		},
	}
	orchestrator := &fakeReviewEnvironmentOrchestrator{
		containerID: "container-123",
		metrics:     &domain.ContainerMetrics{IsRunning: true, RxBytes: 10, TxBytes: 20},
	}
	t.Setenv("BASE_DOMAIN", "")
	svc := NewReviewEnvironmentService(repo, source, orchestrator, &fakeReviewEnvironmentHostMonitor{allow: true})
	svc.SetTimeouts(time.Minute, time.Minute)
	svc.waitForPort = func(context.Context, string, time.Duration) error { return nil }
	return svc, repo, source, orchestrator
}

func TestReviewEnvironmentStartRequiresRunCommand(t *testing.T) {
	svc, repo, source, orchestrator := newReviewEnvironmentFixture(t)
	source.context.AppRunCommand = "  "

	_, err := svc.Start(context.Background(), "tenant-1", "teacher-1", source.context.SubmissionID)
	if !errors.Is(err, domain.ErrReviewEnvironmentCommandMissing) {
		t.Fatalf("expected ErrReviewEnvironmentCommandMissing, got %v", err)
	}
	if repo.created != 0 || orchestrator.startCalls != 0 {
		t.Fatalf("no debía crear recursos sin comando: created=%d start=%d", repo.created, orchestrator.startCalls)
	}
}

func TestReviewEnvironmentStartRejectsJudgeModality(t *testing.T) {
	svc, _, source, orchestrator := newReviewEnvironmentFixture(t)
	source.context.EnvironmentType = domain.PracticeEnvironmentTypeJuezEfimero

	_, err := svc.Start(context.Background(), "tenant-1", "teacher-1", source.context.SubmissionID)
	if !errors.Is(err, domain.ErrReviewEnvironmentRequiresIDE) {
		t.Fatalf("expected ErrReviewEnvironmentRequiresIDE, got %v", err)
	}
	if orchestrator.startCalls != 0 {
		t.Fatal("un juez no debe iniciar un contenedor de app")
	}
}

func TestReviewEnvironmentStartRejectsHostMemoryExhausted(t *testing.T) {
	svc, repo, source, _ := newReviewEnvironmentFixture(t)
	svc.hostMonitor = &fakeReviewEnvironmentHostMonitor{allow: false}

	_, err := svc.Start(context.Background(), "tenant-1", "teacher-1", source.context.SubmissionID)
	if !errors.Is(err, domain.ErrReviewEnvironmentHostMemory) {
		t.Fatalf("expected ErrReviewEnvironmentHostMemory, got %v", err)
	}
	if repo.created != 0 {
		t.Fatal("OOM_GUARD debe rechazar antes de crear el volumen")
	}
}

func TestReviewEnvironmentStartCreatesIsolatedPreview(t *testing.T) {
	svc, repo, source, orchestrator := newReviewEnvironmentFixture(t)

	access, err := svc.Start(context.Background(), "tenant-1", "teacher-1", source.context.SubmissionID)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if !strings.HasPrefix(access.URL, "http://localhost:") {
		t.Fatalf("expected localhost URL, got %q", access.URL)
	}
	if access.ExpiresAt.Before(time.Now().UTC().Add(30 * time.Second)) {
		t.Fatalf("expected live expiry, got %v", access.ExpiresAt)
	}
	if repo.created != 1 || repo.env.Status != domain.ReviewEnvironmentStatusRunning {
		t.Fatalf("expected one running row, got created=%d status=%s", repo.created, repo.env.Status)
	}
	if !strings.HasPrefix(repo.env.VolumeName, "solv-review-env-") {
		t.Fatalf("expected review volume, got %q", repo.env.VolumeName)
	}
	if strings.Contains(repo.env.VolumeName, "solv-environment") {
		t.Fatalf("el preview nunca debe usar el volumen del alumno: %q", repo.env.VolumeName)
	}
	if len(orchestrator.writtenFiles) != 2 || strings.TrimSpace(orchestrator.writtenFiles["main.js"]) != "console.log('hola')" {
		t.Fatalf("snapshot incompleto: %#v", orchestrator.writtenFiles)
	}
	if orchestrator.startConfig.Image != "node:22-alpine" || orchestrator.startConfig.Command != "node main.js" || orchestrator.startConfig.AppPort != 3000 {
		t.Fatalf("configuración inesperada: %#v", orchestrator.startConfig)
	}
	if orchestrator.startConfig.Labels["solv.review_environment"] != "true" {
		t.Fatalf("el preview debe llevar su etiqueta de ciclo de vida: %#v", orchestrator.startConfig.Labels)
	}
	if orchestrator.startConfig.MemoryLimitMB != 512 {
		t.Fatalf("expected 512 MB floor, got %d", orchestrator.startConfig.MemoryLimitMB)
	}
	if repo.env.ContainerID == nil || *repo.env.ContainerID != "container-123" {
		t.Fatal("expected running container id persisted")
	}
}

func TestReviewEnvironmentStartReturnsExistingPreview(t *testing.T) {
	svc, repo, source, orchestrator := newReviewEnvironmentFixture(t)
	expires := time.Now().UTC().Add(10 * time.Minute)
	containerID := "container-existing"
	repo.env = &domain.ReviewEnvironment{
		ID:           "env-existing",
		SubmissionID: source.context.SubmissionID,
		Status:       domain.ReviewEnvironmentStatusRunning,
		AccessURL:    "https://review-abc.example.com",
		ContainerID:  &containerID,
		VolumeName:   "solv-review-env-abc",
		ExpiresAt:    expires,
	}

	access, err := svc.Start(context.Background(), "tenant-1", "teacher-1", source.context.SubmissionID)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if access.URL != "https://review-abc.example.com" {
		t.Fatalf("expected existing URL, got %q", access.URL)
	}
	if !repo.touched {
		t.Fatal("reabrir el preview debe renovar su ventana de inactividad")
	}
	if source.calls != 0 || orchestrator.startCalls != 0 {
		t.Fatalf("no debía recrear el preview: source=%d start=%d", source.calls, orchestrator.startCalls)
	}
}

func TestReviewEnvironmentStartReplacesDeadContainer(t *testing.T) {
	svc, repo, source, orchestrator := newReviewEnvironmentFixture(t)
	oldContainer := "container-dead"
	repo.env = &domain.ReviewEnvironment{
		ID:           "env-old",
		SubmissionID: source.context.SubmissionID,
		Status:       domain.ReviewEnvironmentStatusRunning,
		AccessURL:    "http://localhost:4100",
		ContainerID:  &oldContainer,
		VolumeName:   "solv-review-env-old",
		ExpiresAt:    time.Now().UTC().Add(10 * time.Minute),
		CreatedAt:    time.Now().UTC(),
	}
	orchestrator.metrics = &domain.ContainerMetrics{IsRunning: false}

	access, err := svc.Start(context.Background(), "tenant-1", "teacher-1", source.context.SubmissionID)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if access.URL == "http://localhost:4100" {
		t.Fatal("no debe devolver la URL de un contenedor muerto")
	}
	if len(orchestrator.stopped) != 1 || orchestrator.stopped[0] != oldContainer {
		t.Fatalf("expected old container removed, got %#v", orchestrator.stopped)
	}
	if orchestrator.startCalls != 1 {
		t.Fatalf("expected a fresh container, got %d starts", orchestrator.startCalls)
	}
}

func TestReviewEnvironmentStartCleansUpWhenAppIsNotReady(t *testing.T) {
	svc, repo, source, orchestrator := newReviewEnvironmentFixture(t)
	orchestrator.logs = "Error: Cannot find module 'main.js'"
	svc.waitForPort = func(context.Context, string, time.Duration) error {
		return errors.New("connection refused")
	}

	_, err := svc.Start(context.Background(), "tenant-1", "teacher-1", source.context.SubmissionID)
	if !errors.Is(err, domain.ErrReviewEnvironmentNotReady) {
		t.Fatalf("expected ErrReviewEnvironmentNotReady, got %v", err)
	}
	if len(orchestrator.stopped) != 1 || len(orchestrator.removedVolumes) != 1 {
		t.Fatalf("un preview que no escucha debe liberar contenedor y volumen: %#v %#v", orchestrator.stopped, orchestrator.removedVolumes)
	}
	if repo.failedCode != "app_not_ready" {
		t.Fatalf("expected app_not_ready, got %q", repo.failedCode)
	}
}

func TestReviewEnvironmentReapRenewsActivePreview(t *testing.T) {
	svc, repo, _, orchestrator := newReviewEnvironmentFixture(t)
	containerID := "container-123"
	repo.env = &domain.ReviewEnvironment{
		ID:             "env-1",
		ContainerID:    &containerID,
		VolumeName:     "solv-review-env-1",
		Status:         domain.ReviewEnvironmentStatusRunning,
		ExpiresAt:      time.Now().UTC().Add(-time.Minute),
		LastRxBytes:    100,
		LastTxBytes:    200,
		LastActivityAt: time.Now().UTC().Add(-2 * time.Minute),
	}
	orchestrator.metrics = &domain.ContainerMetrics{IsRunning: true, RxBytes: 500, TxBytes: 200}

	svc.ReapExpired(context.Background())

	if !repo.touched {
		t.Fatal("el tráfico de red debe renovar el TTL")
	}
	if len(orchestrator.stopped) != 0 {
		t.Fatal("no debe apagar un preview con actividad")
	}
	if !repo.env.ExpiresAt.After(time.Now().UTC()) {
		t.Fatalf("expected future expiry, got %v", repo.env.ExpiresAt)
	}
}

func TestReviewEnvironmentReapExpiresIdlePreview(t *testing.T) {
	svc, repo, _, orchestrator := newReviewEnvironmentFixture(t)
	containerID := "container-123"
	repo.env = &domain.ReviewEnvironment{
		ID:          "env-1",
		ContainerID: &containerID,
		VolumeName:  "solv-review-env-1",
		Status:      domain.ReviewEnvironmentStatusRunning,
		ExpiresAt:   time.Now().UTC().Add(-time.Minute),
		LastRxBytes: 500,
		LastTxBytes: 200,
	}
	orchestrator.metrics = &domain.ContainerMetrics{IsRunning: true, RxBytes: 500, TxBytes: 200}

	svc.ReapExpired(context.Background())

	if len(orchestrator.stopped) != 1 || orchestrator.stopped[0] != containerID {
		t.Fatalf("expected container stopped, got %#v", orchestrator.stopped)
	}
	if len(orchestrator.removedVolumes) != 1 {
		t.Fatalf("expected volume removed, got %#v", orchestrator.removedVolumes)
	}
	if !repo.expired {
		t.Fatal("expected preview marked expired")
	}
}

func TestReviewEnvironmentReapKeepsPreviewWhenDockerFails(t *testing.T) {
	svc, repo, _, orchestrator := newReviewEnvironmentFixture(t)
	containerID := "container-123"
	repo.env = &domain.ReviewEnvironment{
		ID:          "env-1",
		ContainerID: &containerID,
		VolumeName:  "solv-review-env-1",
		Status:      domain.ReviewEnvironmentStatusRunning,
		ExpiresAt:   time.Now().UTC().Add(-time.Minute),
	}
	orchestrator.removeErr = errors.New("daemon caído")

	svc.ReapExpired(context.Background())

	if repo.expired {
		t.Fatal("un fallo de Docker no debe marcar el preview como expirado")
	}
}
