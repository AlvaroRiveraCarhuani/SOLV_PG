package services

import (
	"context"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/infrastructure/storage/memory"
)

type fakeRegistryAndBuilder struct {
	localImages     map[string]bool
	builtDockerfile map[string]string
	buildCalls      int
	prunedTags      []string
}

func newFakeRegistryAndBuilder() *fakeRegistryAndBuilder {
	return &fakeRegistryAndBuilder{
		localImages:     make(map[string]bool),
		builtDockerfile: make(map[string]string),
	}
}

func (f *fakeRegistryAndBuilder) InspectLocal(ctx context.Context, imageRef string) (bool, int64, string, error) {
	if f.localImages[imageRef] {
		return true, 150 * 1024 * 1024, "sha256:local123", nil
	}
	return false, 0, "", nil
}

func (f *fakeRegistryAndBuilder) InspectRemoteDigest(ctx context.Context, imageRef string) (string, error) {
	return "sha256:remote123", nil
}

func (f *fakeRegistryAndBuilder) PullImage(ctx context.Context, imageRef string, onProgress func(doneBytes, totalBytes int64, currentLayer, totalLayers int, action string)) error {
	f.localImages[imageRef] = true
	return nil
}

func (f *fakeRegistryAndBuilder) BuildLocalStackImage(ctx context.Context, tag string, dockerfileContent string, onProgress func(action string)) error {
	f.buildCalls++
	if onProgress != nil {
		onProgress("Integrando tecnologías seleccionadas...")
	}
	f.localImages[tag] = true
	f.builtDockerfile[tag] = dockerfileContent
	return nil
}

func (f *fakeRegistryAndBuilder) PruneOrphanStackImages(ctx context.Context, activeTags map[string]bool) ([]string, error) {
	var pruned []string
	for tag := range f.localImages {
		if domain.IsBuiltStackImageTag(tag) && !activeTags[tag] {
			delete(f.localImages, tag)
			pruned = append(pruned, tag)
		}
	}
	f.prunedTags = pruned
	return pruned, nil
}

type fakeSmokeRunner struct{}

func (r *fakeSmokeRunner) RunSmokeTest(ctx context.Context, imageRef string, tools []string, memoryLimitMB int64) ([]domain.ToolResult, int, error) {
	var res []domain.ToolResult
	for _, t := range tools {
		res = append(res, domain.ToolResult{Name: t, Present: true, Version: "1.0.0"})
	}
	return res, 0, nil
}

func (r *fakeSmokeRunner) RunJudgeSmokeTest(ctx context.Context, imageRef string, entrypoint string, sampleInput string, timeoutMS int, memoryLimitMB int64) (string, int64, int, error) {
	return "OK", 12, 0, nil
}

func waitForJobTerminal(t *testing.T, svc *EnvTestService, jobID string) *domain.EnvTestJob {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		job, err := svc.GetJob(context.Background(), jobID)
		if err == nil && job != nil && job.IsTerminal() {
			return job
		}
		time.Sleep(15 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for job %s to finish", jobID)
	return nil
}

func TestEnvTestService_BuildCacheMissAndCacheHit(t *testing.T) {
	repo := memory.NewEnvTestJobMemoryRepository(1 * time.Hour)
	builder := newFakeRegistryAndBuilder()
	runner := &fakeSmokeRunner{}

	svc := NewEnvTestService(repo, builder, runner, EnvTestConfig{MaxConcurrentJobs: 2, TotalJobTimeout: 5 * time.Second, MemoryLimitMB: 512})
	svc.SetImageBuilder(builder)

	tag := "solv-local/stack:a9f4c21e8b10"
	df := "FROM python:3.12-slim-bookworm\nRUN pip install --no-cache-dir fastapi"

	// 1. Primera vez (Cache Miss) -> Debe construir la imagen local
	job1, err := svc.StartJob(context.Background(), domain.StartEnvTestRequest{
		Image:             tag,
		DockerfileContent: df,
		Tools:             []string{"python3", "pip"},
		TargetEnvironment: "IDE_PERSISTENTE",
		BaseRamMB:         512,
	})
	if err != nil {
		t.Fatalf("unexpected error starting job1: %v", err)
	}
	final1 := waitForJobTerminal(t, svc, job1.ID)
	if final1.Status != domain.EnvTestStatusSuccess {
		t.Fatalf("expected job1 success, got %s (%s)", final1.Status, final1.ErrorMessage)
	}
	if builder.buildCalls != 1 {
		t.Fatalf("expected 1 build call on cache miss, got %d", builder.buildCalls)
	}

	// 2. Segunda vez con el mismo tag (Cache Hit) -> NO debe reconstruir la imagen
	job2, err := svc.StartJob(context.Background(), domain.StartEnvTestRequest{
		Image:             tag,
		DockerfileContent: df,
		Tools:             []string{"python3", "pip"},
		TargetEnvironment: "IDE_PERSISTENTE",
		BaseRamMB:         512,
	})
	if err != nil {
		t.Fatalf("unexpected error starting job2: %v", err)
	}
	final2 := waitForJobTerminal(t, svc, job2.ID)
	if final2.Status != domain.EnvTestStatusSuccess {
		t.Fatalf("expected job2 success, got %s", final2.Status)
	}
	if builder.buildCalls != 1 {
		t.Fatalf("expected buildCalls to remain 1 on cache hit, got %d", builder.buildCalls)
	}

	// 3. Limpieza de imágenes construidas localmente que quedaron huérfanas
	builder.localImages["solv-local/stack:orphan999999"] = true
	pruned, err := svc.PruneOrphanStackImages(context.Background(), map[string]bool{tag: true})
	if err != nil {
		t.Fatalf("unexpected prune error: %v", err)
	}
	if len(pruned) != 1 || pruned[0] != "solv-local/stack:orphan999999" {
		t.Fatalf("expected orphan999999 to be pruned, got %v", pruned)
	}
	if !builder.localImages[tag] {
		t.Fatalf("expected active tag %s to be preserved", tag)
	}
}
