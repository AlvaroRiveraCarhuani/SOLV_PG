package services_test

import (
	"context"
	"errors"
	"testing"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

type mockAcademicPeriodRepo struct {
	periods     map[string]*domain.AcademicPeriod
	archivedIDs map[string]string // id -> archivedBy
	deletedIDs  []string
	updated     map[string]*domain.AcademicPeriod
}

func newMockAcademicPeriodRepo() *mockAcademicPeriodRepo {
	return &mockAcademicPeriodRepo{
		periods:     make(map[string]*domain.AcademicPeriod),
		archivedIDs: make(map[string]string),
		updated:     make(map[string]*domain.AcademicPeriod),
	}
}

func (m *mockAcademicPeriodRepo) Create(ctx context.Context, p *domain.AcademicPeriod) error {
	m.periods[p.ID] = p
	return nil
}

func (m *mockAcademicPeriodRepo) GetByID(ctx context.Context, tenantID, id string) (*domain.AcademicPeriod, error) {
	p, ok := m.periods[id]
	if !ok {
		return nil, errors.New("academic period not found")
	}
	return p, nil
}

func (m *mockAcademicPeriodRepo) ListByTenant(ctx context.Context, tenantID string) ([]*domain.AcademicPeriod, error) {
	return nil, nil
}

func (m *mockAcademicPeriodRepo) Update(ctx context.Context, p *domain.AcademicPeriod) error {
	if p.IsArchived {
		return errors.New("academic period not found or not updated")
	}
	m.periods[p.ID] = p
	m.updated[p.ID] = p
	return nil
}

func (m *mockAcademicPeriodRepo) Delete(ctx context.Context, tenantID, id string) error {
	m.deletedIDs = append(m.deletedIDs, id)
	delete(m.periods, id)
	return nil
}

func (m *mockAcademicPeriodRepo) ArchiveExpiredPeriods(ctx context.Context) (int64, error) {
	return 0, nil
}

func (m *mockAcademicPeriodRepo) Archive(ctx context.Context, tenantID, id, archivedBy string) error {
	p, ok := m.periods[id]
	if !ok {
		return errors.New("academic period not found")
	}
	p.IsArchived = true
	p.IsActive = false
	m.archivedIDs[id] = archivedBy
	return nil
}

func TestAcademicPeriodService_ArchivePeriod_ConfirmationStrong(t *testing.T) {
	repo := newMockAcademicPeriodRepo()
	svc := services.NewAcademicPeriodService(repo)

	_ = repo.Create(context.Background(), &domain.AcademicPeriod{
		ID:       "p-1",
		TenantID: "t-1",
		Name:     "Semestre II / 2026",
		Code:     "2026-2",
		IsActive: true,
	})

	// Codigo incorrecto es rechazado y NO archiva
	err := svc.ArchivePeriod(context.Background(), "t-1", "p-1", "admin-1", "codigo-malo")
	if !errors.Is(err, services.ErrConfirmationFailed) {
		t.Fatalf("expected ErrConfirmationFailed, got %v", err)
	}
	if len(repo.archivedIDs) != 0 {
		t.Fatalf("period must NOT be archived on confirmation failure")
	}

	// Codigo exacto archiva formalmente con trazabilidad
	if err := svc.ArchivePeriod(context.Background(), "t-1", "p-1", "admin-1", "2026-2"); err != nil {
		t.Fatalf("expected successful archive, got %v", err)
	}
	if repo.archivedIDs["p-1"] != "admin-1" {
		t.Fatalf("expected archived_by=admin-1, got %q", repo.archivedIDs["p-1"])
	}

	// Un periodo archivado es inmutable: no reactiva ni edita
	_, err = svc.UpdatePeriod(context.Background(), "t-1", "p-1", domain.UpdateAcademicPeriodDTO{IsActive: boolPtr(true)})
	if !errors.Is(err, services.ErrPeriodArchived) {
		t.Fatalf("expected ErrPeriodArchived on update of archived period, got %v", err)
	}

	// Un periodo archivado no se elimina
	err = svc.DeletePeriod(context.Background(), "t-1", "p-1")
	if !errors.Is(err, services.ErrPeriodArchived) {
		t.Fatalf("expected ErrPeriodArchived on delete of archived period, got %v", err)
	}

	// Archivar dos veces es rechazado
	err = svc.ArchivePeriod(context.Background(), "t-1", "p-1", "admin-1", "2026-2")
	if !errors.Is(err, services.ErrPeriodArchived) {
		t.Fatalf("expected ErrPeriodArchived on double archive, got %v", err)
	}
}

func boolPtr(b bool) *bool {
	return &b
}
