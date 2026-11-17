package services_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

type mockDiskMeasurerOrchestrator struct {
	*mockOrchestrator
	measuredMB int64
	purgedMB   int64
	files      map[string]string
}

func (m *mockDiskMeasurerOrchestrator) MeasurePracticeEnvironmentVolumeMB(ctx context.Context, volumeName string) (int64, error) {
	return m.measuredMB, nil
}

func (m *mockDiskMeasurerOrchestrator) PurgePracticeEnvironmentCaches(ctx context.Context, volumeName string, paths []string) (int64, error) {
	m.measuredMB -= m.purgedMB
	if m.measuredMB < 0 {
		m.measuredMB = 0
	}
	return m.purgedMB, nil
}

func (m *mockDiskMeasurerOrchestrator) WritePracticeEnvironmentVolumeFiles(ctx context.Context, volumeName string, files map[string]string) error {
	m.files = files
	return nil
}

func (m *mockDiskMeasurerOrchestrator) SnapshotPracticeEnvironmentVolume(ctx context.Context, volumeName string, maxBytes int64) ([]domain.PracticeEnvironmentVolumeFile, error) {
	return []domain.PracticeEnvironmentVolumeFile{
		{Path: "main.py", Content: []byte("print('hello')")},
	}, nil
}

type mockQuotaRepo struct {
	*mockPracticeEnvironmentRepo
}

func (m *mockQuotaRepo) RecordDiskUsage(ctx context.Context, environmentID string, usedMB int64, state string) error {
	for _, ws := range m.environments {
		if ws != nil && ws.ID == environmentID {
			ws.DiskUsedMB = usedMB
			ws.DiskState = state
		}
	}
	return nil
}

func TestPracticeEnvironment_DiskQuotaFullCycle(t *testing.T) {
	wsRepo := &mockQuotaRepo{mockPracticeEnvironmentRepo: newMockPracticeEnvironmentRepo()}
	orch := &mockDiskMeasurerOrchestrator{
		mockOrchestrator: &mockOrchestrator{},
		measuredMB:       100, // Inicialmente 100 MB de 1024 MB (OK)
	}

	subjectID := uuid.NewString()
	templateID := "tpl-quota"
	exerciseID := uuid.NewString()

	subjRepo := &mockSubjectRepo{subjects: map[string]*domain.Subject{
		subjectID: {ID: subjectID, Name: "Sistemas Operativos"},
	}}
	tplRepo := &mockTemplateRepo{templates: map[string]*domain.Template{
		templateID: {
			ID:          templateID,
			DockerImage: "solv/ide:python",
			BaseRamMB:   512,
		},
	}}
	exRepo := &mockExerciseRepo{exercises: map[string]*domain.Exercise{
		exerciseID: {
			ID:         exerciseID,
			TemplateID: templateID,
			SubjectID:  &subjectID,
			Purpose:    string(domain.ExercisePurposeClass),
		},
	}}

	subRepo := &fakeSubmissionRepo{}

	// Políticas: cuota 1024 MB, aviso al 80% (819 MB), bloqueo al 95% (972 MB)
	quotaMB := 1024
	warningPct := 80
	blockPct := 95
	tenantRepo := &mockPoliciesTenantRepo{
		configs: map[string][]byte{
			domain.DefaultTenantID: []byte(`{"server_policies":{"disk_quota_mb":1024,"disk_warning_pct":80,"disk_block_pct":95}}`),
		},
	}
	policiesSvc := services.NewServerPoliciesService(tenantRepo)

	svc := services.NewPracticeEnvironmentService(wsRepo, orch, &mockHostMonitor{}).
		WithProvisioning(subjRepo, tplRepo).
		WithExerciseSubmission(exRepo, subRepo).
		SetPoliciesService(policiesSvc)

	ctx := context.WithValue(context.Background(), domain.TenantIDKey, domain.DefaultTenantID)

	// 1. Iniciar el entorno
	env, err := svc.StartPracticeEnvironmentForExercise(ctx, "std001", subjectID, exerciseID)
	if err != nil {
		t.Fatalf("error iniciando entorno: %v", err)
	}

	// 2. Medición inicial (100 MB -> OK)
	svc.SweepDiskUsage(ctx)
	reloaded, _ := wsRepo.GetByID(ctx, env.ID)
	if reloaded.DiskState != domain.DiskStateOK {
		t.Fatalf("esperado DiskStateOK para 100 MB, obtenido %q", reloaded.DiskState)
	}

	// 3. Simular crecimiento a 850 MB (> 80% de 1024 MB -> Warning)
	orch.measuredMB = 850
	svc.SweepDiskUsage(ctx)
	reloaded, _ = wsRepo.GetByID(ctx, env.ID)
	if reloaded.DiskState != domain.DiskStateWarning {
		t.Fatalf("esperado DiskStateWarning para 850 MB, obtenido %q", reloaded.DiskState)
	}

	// 4. Simular desborde a 1000 MB (> 95% de 1024 MB -> Full)
	orch.measuredMB = 1000
	svc.SweepDiskUsage(ctx)
	reloaded, _ = wsRepo.GetByID(ctx, env.ID)
	if reloaded.DiskState != domain.DiskStateFull {
		t.Fatalf("esperado DiskStateFull para 1000 MB, obtenido %q", reloaded.DiskState)
	}

	// 5. Intento de entrega en estado Full debe ser rechazado con ErrDiskQuotaExceeded (409)
	_, submitErr := svc.SubmitPracticeEnvironmentProject(ctx, env.ID, "std001", exerciseID, nil)
	if submitErr == nil {
		t.Fatal("se esperaba rechazo de entrega por cuota de disco excedida")
	}
	if !errors.Is(submitErr, domain.ErrDiskQuotaExceeded) {
		t.Fatalf("se esperaba ErrDiskQuotaExceeded, obtenido: %v", submitErr)
	}

	// 6. Limpieza de caché (libera 400 MB -> baja a 600 MB)
	orch.purgedMB = 400
	purgeResult, err := svc.PurgeEnvironmentCaches(ctx, env.ID, []string{"/home/workspace/.cache"})
	if err != nil {
		t.Fatalf("error al purgar caché: %v", err)
	}
	if purgeResult.FreedMB != 400 {
		t.Errorf("espacio liberado esperado 400 MB, obtenido %d", purgeResult.FreedMB)
	}
	if purgeResult.State != domain.DiskStateOK {
		t.Errorf("estado esperado tras purga DiskStateOK, obtenido %q", purgeResult.State)
	}

	// 7. Tras la limpieza, la entrega debe permitirse normalmente
	orch.measuredMB = 600
	submission, err := svc.SubmitPracticeEnvironmentProject(ctx, env.ID, "std001", exerciseID, nil)
	if err != nil {
		t.Fatalf("se esperaba entrega exitosa tras limpieza, obtenido error: %v", err)
	}
	if submission == nil {
		t.Fatal("submission no debe ser nula")
	}
	_ = quotaMB
	_ = warningPct
	_ = blockPct
}
