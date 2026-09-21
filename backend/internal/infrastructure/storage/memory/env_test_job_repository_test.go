package memory_test

import (
	"context"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/infrastructure/storage/memory"
)

func TestEnvTestJobMemoryRepository_Lifecycle(t *testing.T) {
	ctx := context.Background()
	repo := memory.NewEnvTestJobMemoryRepository(time.Hour)

	job := &domain.EnvTestJob{
		ID:     "job-101",
		Image:  "python:3.12-slim",
		Tools:  []string{"python", "pip"},
		Status: domain.EnvTestStatusPending,
	}

	if err := repo.Save(ctx, job); err != nil {
		t.Fatalf("unexpected save error: %v", err)
	}

	got, err := repo.GetByID(ctx, "job-101")
	if err != nil {
		t.Fatalf("unexpected get error: %v", err)
	}
	if got.Status != domain.EnvTestStatusPending {
		t.Errorf("expected pending, got %v", got.Status)
	}

	// Update Progress
	prog := domain.EnvTestProgress{
		BytesDone:  500,
		BytesTotal: 1000,
		Percent:    50.0,
	}
	if err := repo.UpdateProgress(ctx, "job-101", prog); err != nil {
		t.Fatalf("unexpected update progress error: %v", err)
	}

	got, _ = repo.GetByID(ctx, "job-101")
	if got.Status != domain.EnvTestStatusPulling || got.Progress.Percent != 50.0 {
		t.Errorf("expected pulling with 50%%, got status %v, pct %v", got.Status, got.Progress.Percent)
	}

	// List Active
	active, err := repo.ListActive(ctx)
	if err != nil || len(active) != 1 {
		t.Errorf("expected 1 active job, got %d (err: %v)", len(active), err)
	}

	// Complete
	res := &domain.EnvTestResult{
		Tools: []domain.ToolResult{
			{Name: "python", Present: true, Version: "Python 3.12.2"},
			{Name: "pip", Present: true, Version: "pip 24.0"},
		},
		ExitCode:   0,
		DurationMs: 1200,
	}
	if err := repo.Complete(ctx, "job-101", res); err != nil {
		t.Fatalf("unexpected complete error: %v", err)
	}

	got, _ = repo.GetByID(ctx, "job-101")
	if got.Status != domain.EnvTestStatusSuccess || got.Result == nil || len(got.Result.Tools) != 2 {
		t.Errorf("expected success with 2 tools, got %v", got.Status)
	}

	// List Active should now be 0
	active, _ = repo.ListActive(ctx)
	if len(active) != 0 {
		t.Errorf("expected 0 active jobs after completion, got %d", len(active))
	}
}

func TestEnvTestJobMemoryRepository_CancelAndFail(t *testing.T) {
	ctx := context.Background()
	repo := memory.NewEnvTestJobMemoryRepository(time.Hour)

	j1 := &domain.EnvTestJob{ID: "j1", Image: "alpine:3.19", Status: domain.EnvTestStatusPending}
	j2 := &domain.EnvTestJob{ID: "j2", Image: "ubuntu:24.04", Status: domain.EnvTestStatusPending}

	_ = repo.Save(ctx, j1)
	_ = repo.Save(ctx, j2)

	_ = repo.Cancel(ctx, "j1")
	got1, _ := repo.GetByID(ctx, "j1")
	if got1.Status != domain.EnvTestStatusCanceled {
		t.Errorf("expected canceled, got %v", got1.Status)
	}

	_ = repo.Fail(ctx, "j2", domain.EnvTestErrPullStalled, "descarga detenida")
	got2, _ := repo.GetByID(ctx, "j2")
	if got2.Status != domain.EnvTestStatusFailed || got2.ErrorCode != domain.EnvTestErrPullStalled {
		t.Errorf("expected failed with pull_stalled, got %v / %v", got2.Status, got2.ErrorCode)
	}
}
