package services_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
	"solv-backend/internal/infrastructure/storage/memory"
)

// Mock Registry
type mockRegistry struct {
	mu           sync.Mutex
	isLocal      bool
	localSize    int64
	localDigest  string
	inspectErr   error
	remoteDigest string
	remoteErr    error
	pullErr      error
	pullCalled   bool
}

func (m *mockRegistry) InspectLocal(ctx context.Context, imageRef string) (bool, int64, string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.isLocal, m.localSize, m.localDigest, m.inspectErr
}

func (m *mockRegistry) InspectRemoteDigest(ctx context.Context, imageRef string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.remoteDigest, m.remoteErr
}

func (m *mockRegistry) PullImage(ctx context.Context, imageRef string, onProgress func(doneBytes, totalBytes int64, currentLayer, totalLayers int, action string)) error {
	m.mu.Lock()
	m.pullCalled = true
	err := m.pullErr
	m.mu.Unlock()

	if err != nil {
		return err
	}
	if onProgress != nil {
		onProgress(100, 100, 1, 1, "done")
	}
	return nil
}

// Mock Runner
type mockRunner struct {
	mu        sync.Mutex
	results   []domain.ToolResult
	exitCode  int
	err       error
	runCalled bool
	delay     time.Duration
	lastTools []string
}

func (m *mockRunner) RunSmokeTest(ctx context.Context, imageRef string, tools []string, memoryLimitMB int64) ([]domain.ToolResult, int, error) {
	m.mu.Lock()
	m.runCalled = true
	m.lastTools = append([]string{}, tools...)
	res := m.results
	code := m.exitCode
	err := m.err
	delay := m.delay
	m.mu.Unlock()

	if delay > 0 {
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return nil, 124, ctx.Err()
		}
	}

	return res, code, err
}

func (m *mockRunner) RunJudgeSmokeTest(ctx context.Context, imageRef string, entrypoint string, sampleInput string, timeoutMS int, memoryLimitMB int64) (string, int64, int, error) {
	m.mu.Lock()
	m.runCalled = true
	code := m.exitCode
	err := m.err
	delay := m.delay
	m.mu.Unlock()

	if delay > 0 {
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return "TLE", 124, 124, ctx.Err()
		}
	}

	if code != 0 {
		return "execution failed", 50, code, err
	}
	return "PASS: output matches", 25, 0, nil
}

func TestEnvTestService_Validation(t *testing.T) {
	repo := memory.NewEnvTestJobMemoryRepository(time.Hour)
	reg := &mockRegistry{}
	run := &mockRunner{}
	svc := services.NewEnvTestService(repo, reg, run, services.EnvTestConfig{})

	ctx := context.Background()

	// Empty image
	_, err := svc.StartJob(ctx, domain.StartEnvTestRequest{Image: ""})
	if err == nil {
		t.Errorf("expected error for empty image")
	}

	// Latest tag
	_, err = svc.StartJob(ctx, domain.StartEnvTestRequest{Image: "python:latest"})
	if err == nil {
		t.Errorf("expected error for latest tag")
	}

	// Invalid tool
	_, err = svc.StartJob(ctx, domain.StartEnvTestRequest{Image: "python:3.12", Tools: []string{"gcc;rm -rf /"}})
	if err == nil {
		t.Errorf("expected error for invalid tool name")
	}
}

func TestEnvTestService_HappyPathLocal(t *testing.T) {
	repo := memory.NewEnvTestJobMemoryRepository(time.Hour)
	reg := &mockRegistry{
		isLocal:      true,
		localDigest:  "sha256:abc",
		remoteDigest: "sha256:abc",
	}
	run := &mockRunner{
		results: []domain.ToolResult{
			{Name: "gcc", Present: true, Path: "/usr/bin/gcc", Version: "gcc 13.2"},
		},
		exitCode: 0,
	}
	svc := services.NewEnvTestService(repo, reg, run, services.EnvTestConfig{})

	job, err := svc.StartJob(context.Background(), domain.StartEnvTestRequest{
		Image: "gcc:13.2",
		Tools: []string{"gcc"},
	})
	if err != nil {
		t.Fatalf("unexpected start error: %v", err)
	}

	// Wait for background execution to finish
	var finishedJob *domain.EnvTestJob
	for i := 0; i < 20; i++ {
		time.Sleep(20 * time.Millisecond)
		finishedJob, _ = svc.GetJob(context.Background(), job.ID)
		if finishedJob != nil && finishedJob.IsTerminal() {
			break
		}
	}

	if finishedJob == nil || finishedJob.Status != domain.EnvTestStatusSuccess {
		t.Fatalf("expected job success, got: %+v", finishedJob)
	}
	if reg.pullCalled {
		t.Errorf("did not expect pull when image is local with matching digest")
	}
	if finishedJob.Result == nil || len(finishedJob.Result.Tools) != 1 {
		t.Errorf("expected 1 tool result in report")
	}
}

func TestEnvTestService_DigestDegradation_OfflineLocal(t *testing.T) {
	repo := memory.NewEnvTestJobMemoryRepository(time.Hour)
	reg := &mockRegistry{
		isLocal:   true,
		remoteErr: errors.New("connection timeout to docker.io"),
	}
	run := &mockRunner{
		results: []domain.ToolResult{
			{Name: "python", Present: true},
		},
		exitCode: 0,
	}
	svc := services.NewEnvTestService(repo, reg, run, services.EnvTestConfig{})

	job, err := svc.StartJob(context.Background(), domain.StartEnvTestRequest{
		Image: "python:3.12-slim",
		Tools: []string{"python"},
	})
	if err != nil {
		t.Fatalf("unexpected start error: %v", err)
	}

	var finishedJob *domain.EnvTestJob
	for i := 0; i < 20; i++ {
		time.Sleep(20 * time.Millisecond)
		finishedJob, _ = svc.GetJob(context.Background(), job.ID)
		if finishedJob != nil && finishedJob.IsTerminal() {
			break
		}
	}

	if finishedJob == nil || finishedJob.Status != domain.EnvTestStatusSuccess {
		t.Fatalf("expected job success in degraded mode, got: %+v", finishedJob)
	}
	if !finishedJob.DigestUnverified {
		t.Errorf("expected DigestUnverified = true for offline local image")
	}
}

func TestEnvTestService_DigestDegradation_OfflineRemoteFail(t *testing.T) {
	repo := memory.NewEnvTestJobMemoryRepository(time.Hour)
	reg := &mockRegistry{
		isLocal:   false,
		remoteErr: errors.New("dns resolution error"),
	}
	run := &mockRunner{}
	svc := services.NewEnvTestService(repo, reg, run, services.EnvTestConfig{})

	job, _ := svc.StartJob(context.Background(), domain.StartEnvTestRequest{
		Image: "nonexistent:1.0",
		Tools: []string{"test"},
	})

	var finishedJob *domain.EnvTestJob
	for i := 0; i < 20; i++ {
		time.Sleep(20 * time.Millisecond)
		finishedJob, _ = svc.GetJob(context.Background(), job.ID)
		if finishedJob != nil && finishedJob.IsTerminal() {
			break
		}
	}

	if finishedJob == nil || finishedJob.Status != domain.EnvTestStatusFailed {
		t.Fatalf("expected failed status, got: %+v", finishedJob)
	}
	if finishedJob.ErrorCode != domain.EnvTestErrRegistryUnreach {
		t.Errorf("expected error code %q, got %q", domain.EnvTestErrRegistryUnreach, finishedJob.ErrorCode)
	}
}

func TestEnvTestService_NotFoundRemote(t *testing.T) {
	repo := memory.NewEnvTestJobMemoryRepository(time.Hour)
	reg := &mockRegistry{
		isLocal:   false,
		remoteErr: errors.New("Error response from daemon: manifest unknown: not found"),
	}
	run := &mockRunner{}
	svc := services.NewEnvTestService(repo, reg, run, services.EnvTestConfig{})

	job, _ := svc.StartJob(context.Background(), domain.StartEnvTestRequest{
		Image: "eclipse-temurin:21-jdk-bookworm",
		Tools: []string{"java"},
	})

	var finishedJob *domain.EnvTestJob
	for i := 0; i < 20; i++ {
		time.Sleep(20 * time.Millisecond)
		finishedJob, _ = svc.GetJob(context.Background(), job.ID)
		if finishedJob != nil && finishedJob.IsTerminal() {
			break
		}
	}

	if finishedJob == nil || finishedJob.Status != domain.EnvTestStatusFailed {
		t.Fatalf("expected failed status, got: %+v", finishedJob)
	}
	if finishedJob.ErrorCode != domain.EnvTestErrNotFoundRemote {
		t.Errorf("expected error code %q, got %q", domain.EnvTestErrNotFoundRemote, finishedJob.ErrorCode)
	}
	if finishedJob.IsLocalImage {
		t.Errorf("expected IsLocalImage to be false")
	}
}

func TestEnvTestService_NetworkFailure(t *testing.T) {
	repo := memory.NewEnvTestJobMemoryRepository(time.Hour)
	reg := &mockRegistry{
		isLocal:   false,
		remoteErr: errors.New("dial tcp: lookup registry-1.docker.io: no such host"),
	}
	run := &mockRunner{}
	svc := services.NewEnvTestService(repo, reg, run, services.EnvTestConfig{})

	job, _ := svc.StartJob(context.Background(), domain.StartEnvTestRequest{
		Image: "python:3.12-slim",
		Tools: []string{"python3"},
	})

	var finishedJob *domain.EnvTestJob
	for i := 0; i < 20; i++ {
		time.Sleep(20 * time.Millisecond)
		finishedJob, _ = svc.GetJob(context.Background(), job.ID)
		if finishedJob != nil && finishedJob.IsTerminal() {
			break
		}
	}

	if finishedJob == nil || finishedJob.Status != domain.EnvTestStatusFailed {
		t.Fatalf("expected failed status, got: %+v", finishedJob)
	}
	if finishedJob.ErrorCode != domain.EnvTestErrNetworkFailure {
		t.Errorf("expected error code %q, got %q", domain.EnvTestErrNetworkFailure, finishedJob.ErrorCode)
	}
}

func TestEnvTestService_OOM_Detection(t *testing.T) {
	repo := memory.NewEnvTestJobMemoryRepository(time.Hour)
	reg := &mockRegistry{isLocal: true}
	run := &mockRunner{
		exitCode: 137,
		err:      errors.New("container killed by OOM"),
	}
	svc := services.NewEnvTestService(repo, reg, run, services.EnvTestConfig{})

	job, _ := svc.StartJob(context.Background(), domain.StartEnvTestRequest{
		Image: "heavy:1.0",
		Tools: []string{"bigtool"},
	})

	var finishedJob *domain.EnvTestJob
	for i := 0; i < 20; i++ {
		time.Sleep(20 * time.Millisecond)
		finishedJob, _ = svc.GetJob(context.Background(), job.ID)
		if finishedJob != nil && finishedJob.IsTerminal() {
			break
		}
	}

	if finishedJob == nil || finishedJob.Status != domain.EnvTestStatusFailed {
		t.Fatalf("expected failed status on OOM, got: %+v", finishedJob)
	}
	if finishedJob.ErrorCode != domain.EnvTestErrTestOOM {
		t.Errorf("expected error code %q, got %q", domain.EnvTestErrTestOOM, finishedJob.ErrorCode)
	}
}

func TestEnvTestService_Cancellation(t *testing.T) {
	repo := memory.NewEnvTestJobMemoryRepository(time.Hour)
	reg := &mockRegistry{isLocal: true}
	run := &mockRunner{
		delay: 500 * time.Millisecond,
	}
	svc := services.NewEnvTestService(repo, reg, run, services.EnvTestConfig{})

	job, _ := svc.StartJob(context.Background(), domain.StartEnvTestRequest{
		Image: "slow:1.0",
		Tools: []string{"test"},
	})

	time.Sleep(30 * time.Millisecond)
	if err := svc.CancelJob(context.Background(), job.ID); err != nil {
		t.Fatalf("unexpected cancel error: %v", err)
	}

	time.Sleep(50 * time.Millisecond)
	got, _ := svc.GetJob(context.Background(), job.ID)
	if got.Status != domain.EnvTestStatusCanceled {
		t.Errorf("expected status canceled, got: %v", got.Status)
	}
}

func TestEnvTestService_IDEPersistente_VerifiesOpenVSCodeBinary(t *testing.T) {
	repo := memory.NewEnvTestJobMemoryRepository(time.Hour)
	reg := &mockRegistry{isLocal: true}
	openVSCodePath := "/home/.openvscode-server/bin/openvscode-server"
	run := &mockRunner{
		results: []domain.ToolResult{
			{Name: "python3", Present: true, Path: "/usr/bin/python3"},
			{Name: openVSCodePath, Present: true, Path: openVSCodePath, Version: "1.96.0"},
		},
		exitCode: 0,
	}
	svc := services.NewEnvTestService(repo, reg, run, services.EnvTestConfig{})

	job, err := svc.StartJob(context.Background(), domain.StartEnvTestRequest{
		Image:             "python:3.12-slim",
		Tools:             []string{"python3"},
		TargetEnvironment: "IDE_PERSISTENTE",
	})
	if err != nil {
		t.Fatalf("unexpected start error: %v", err)
	}

	var finishedJob *domain.EnvTestJob
	for i := 0; i < 20; i++ {
		time.Sleep(20 * time.Millisecond)
		finishedJob, _ = svc.GetJob(context.Background(), job.ID)
		if finishedJob != nil && finishedJob.IsTerminal() {
			break
		}
	}

	if finishedJob == nil || finishedJob.Status != domain.EnvTestStatusSuccess {
		t.Fatalf("expected job success, got: %+v", finishedJob)
	}

	run.mu.Lock()
	requestedTools := run.lastTools
	run.mu.Unlock()
	foundEditor := false
	for _, tool := range requestedTools {
		if tool == openVSCodePath {
			foundEditor = true
			break
		}
	}
	if !foundEditor {
		t.Errorf("esperado que el smoke test verifique %s, herramientas pedidas: %v", openVSCodePath, requestedTools)
	}

	var editorResult *domain.ToolResult
	for i := range finishedJob.Result.Tools {
		if finishedJob.Result.Tools[i].Name == "openvscode-server" {
			editorResult = &finishedJob.Result.Tools[i]
			break
		}
	}
	if editorResult == nil {
		t.Fatalf("esperado resultado openvscode-server en el reporte, obtenido: %+v", finishedJob.Result.Tools)
	}
	if !editorResult.Present {
		t.Errorf("esperado openvscode-server presente = true")
	}
}

func TestEnvTestService_IDEPersistente_FailsWhenOpenVSCodeMissing(t *testing.T) {
	repo := memory.NewEnvTestJobMemoryRepository(time.Hour)
	reg := &mockRegistry{isLocal: true}
	openVSCodePath := "/home/.openvscode-server/bin/openvscode-server"
	run := &mockRunner{
		results: []domain.ToolResult{
			{Name: "python3", Present: true, Path: "/usr/bin/python3"},
			{Name: openVSCodePath, Present: false},
		},
		exitCode: 0,
	}
	svc := services.NewEnvTestService(repo, reg, run, services.EnvTestConfig{})

	job, err := svc.StartJob(context.Background(), domain.StartEnvTestRequest{
		Image:             "python:3.12-slim",
		Tools:             []string{"python3"},
		TargetEnvironment: "IDE_PERSISTENTE",
	})
	if err != nil {
		t.Fatalf("unexpected start error: %v", err)
	}

	var finishedJob *domain.EnvTestJob
	for i := 0; i < 20; i++ {
		time.Sleep(20 * time.Millisecond)
		finishedJob, _ = svc.GetJob(context.Background(), job.ID)
		if finishedJob != nil && finishedJob.IsTerminal() {
			break
		}
	}

	if finishedJob == nil || finishedJob.Status != domain.EnvTestStatusFailed {
		t.Fatalf("expected failed status when editor missing, got: %+v", finishedJob)
	}
	if finishedJob.ErrorCode != domain.EnvTestErrTestCrash {
		t.Errorf("expected error code %q, got %q", domain.EnvTestErrTestCrash, finishedJob.ErrorCode)
	}
}

func TestEnvTestService_JuezEfimero_DoesNotRequireOpenVSCode(t *testing.T) {
	repo := memory.NewEnvTestJobMemoryRepository(time.Hour)
	reg := &mockRegistry{isLocal: true}
	run := &mockRunner{
		results: []domain.ToolResult{
			{Name: "gcc", Present: true, Path: "/usr/bin/gcc"},
		},
		exitCode: 0,
	}
	svc := services.NewEnvTestService(repo, reg, run, services.EnvTestConfig{})

	job, err := svc.StartJob(context.Background(), domain.StartEnvTestRequest{
		Image:             "gcc:13.2",
		Tools:             []string{"gcc"},
		TargetEnvironment: "JUEZ_EFIMERO",
		Entrypoint:        "gcc solution.c -o solution && ./solution",
		TimeoutMS:         2000,
	})
	if err != nil {
		t.Fatalf("unexpected start error: %v", err)
	}

	var finishedJob *domain.EnvTestJob
	for i := 0; i < 20; i++ {
		time.Sleep(20 * time.Millisecond)
		finishedJob, _ = svc.GetJob(context.Background(), job.ID)
		if finishedJob != nil && finishedJob.IsTerminal() {
			break
		}
	}

	if finishedJob == nil || finishedJob.Status != domain.EnvTestStatusSuccess {
		t.Fatalf("expected job success, got: %+v", finishedJob)
	}

	run.mu.Lock()
	requestedTools := run.lastTools
	run.mu.Unlock()
	for _, tool := range requestedTools {
		if tool == "/home/.openvscode-server/bin/openvscode-server" {
			t.Errorf("el entorno JUEZ_EFIMERO no debe verificar el binario del editor")
		}
	}
}

func TestEnvTestService_JudgeExecutionSuccess(t *testing.T) {
	repo := memory.NewEnvTestJobMemoryRepository(time.Hour)
	reg := &mockRegistry{isLocal: true}
	run := &mockRunner{exitCode: 0}
	svc := services.NewEnvTestService(repo, reg, run, services.EnvTestConfig{})

	job, err := svc.StartJob(context.Background(), domain.StartEnvTestRequest{
		Image:             "gcc:13.2",
		TargetEnvironment: "JUEZ_EFIMERO",
		Entrypoint:        "gcc solution.c -o solution && ./solution",
		TimeoutMS:         2000,
		SampleInput:       "42\n",
	})
	if err != nil {
		t.Fatalf("unexpected error starting judge job: %v", err)
	}

	time.Sleep(50 * time.Millisecond)
	got, _ := svc.GetJob(context.Background(), job.ID)
	if got.Status != domain.EnvTestStatusSuccess {
		t.Fatalf("expected judge test status success, got: %v (%s)", got.Status, got.ErrorMessage)
	}
	if got.Result == nil || got.Result.ExitCode != 0 {
		t.Errorf("expected exit code 0 in result, got: %+v", got.Result)
	}
}

func TestEnvTestService_StartJob_RamExceedsHost(t *testing.T) {
	repo := memory.NewEnvTestJobMemoryRepository(time.Hour)
	reg := &mockRegistry{isLocal: true}
	run := &mockRunner{exitCode: 0}
	svc := services.NewEnvTestService(repo, reg, run, services.EnvTestConfig{})

	job, err := svc.StartJob(context.Background(), domain.StartEnvTestRequest{
		Image:     "python:3.12-slim",
		Tools:     []string{"python3"},
		BaseRamMB: 9999999, // Supera con creces el total del host
	})
	if err != nil {
		t.Fatalf("unexpected error starting job: %v", err)
	}

	time.Sleep(50 * time.Millisecond)
	got, _ := svc.GetJob(context.Background(), job.ID)
	if got.Status != domain.EnvTestStatusFailed {
		t.Fatalf("expected job status failed due to ram_exceeds_host, got: %v", got.Status)
	}
	if got.ErrorCode != "ram_exceeds_host" {
		t.Errorf("expected error code ram_exceeds_host, got: %s", got.ErrorCode)
	}
}

func TestEnvTestService_StartJob_DualExecution(t *testing.T) {
	repo := memory.NewEnvTestJobMemoryRepository(time.Hour)
	reg := &mockRegistry{isLocal: true}
	run := &mockRunner{
		results: []domain.ToolResult{
			{Name: "python3", Present: true, Path: "/usr/bin/python3", Version: "Python 3.12.0"},
		},
		exitCode: 0,
	}
	svc := services.NewEnvTestService(repo, reg, run, services.EnvTestConfig{})

	job, err := svc.StartJob(context.Background(), domain.StartEnvTestRequest{
		Image:             "python:3.12-slim",
		Tools:             []string{"python3"},
		TargetEnvironment: domain.TargetEnvironmentAmbos,
		Entrypoint:        "python3 solution.py",
		TimeoutMS:         2000,
	})
	if err != nil {
		t.Fatalf("unexpected error starting dual job: %v", err)
	}

	time.Sleep(50 * time.Millisecond)
	got, _ := svc.GetJob(context.Background(), job.ID)
	if got.Status != domain.EnvTestStatusSuccess {
		t.Fatalf("expected dual test status success, got: %v (%s)", got.Status, got.ErrorMessage)
	}
	if got.Result == nil || len(got.Result.Tools) < 2 {
		t.Fatalf("expected at least 2 tool results (tools + entrypoint), got: %+v", got.Result)
	}

	foundEntrypoint := false
	foundTool := false
	for _, tool := range got.Result.Tools {
		if tool.Name == "entrypoint" && tool.Present {
			foundEntrypoint = true
		}
		if tool.Name == "python3" && tool.Present {
			foundTool = true
		}
	}
	if !foundEntrypoint {
		t.Errorf("expected entrypoint smoke test result in dual test, got: %+v", got.Result.Tools)
	}
	if !foundTool {
		t.Errorf("expected python3 tool result in dual test, got: %+v", got.Result.Tools)
	}
}
