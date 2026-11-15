package services

import (
	"context"
	"errors"
	"testing"

	"solv-backend/internal/core/domain"
)

// mockRosterSyncRepo embebe la interfaz completa para no reimplementar el
// contrato entero: solo interesa la capacidad de sincronización.
type mockRosterSyncRepo struct {
	domain.AdminGovernanceRepository
	calls   int
	lastReq domain.RosterSyncRequest
	outcome *domain.RosterSyncOutcome
	err     error
}

func (m *mockRosterSyncRepo) SyncSubjectRoster(ctx context.Context, tenantID, subjectID string, req domain.RosterSyncRequest) (*domain.RosterSyncOutcome, error) {
	m.calls++
	m.lastReq = req
	if m.err != nil {
		return nil, m.err
	}
	return m.outcome, nil
}

func TestRosterEnrollFromSource_NormalizaYDelega(t *testing.T) {
	repo := &mockRosterSyncRepo{
		outcome: &domain.RosterSyncOutcome{
			CreatedStudents:     2,
			EnrolledStudents:    3,
			RemovedStudents:     1,
			SkippedNotFound:     1,
			MatchedByExternalID: 2,
		},
	}
	svc := NewAdminGovernanceService(nil, repo)

	result, err := svc.RosterEnrollFromSource(context.Background(), "tenant-1", "subject-9", domain.RosterSyncRequest{
		Entries: []domain.RosterEntry{
			{Email: "  Student1@UAB.edu.bo ", ExternalID: " 202012345 ", FirstName: "Ana", LastName: "Quispe"},
			{Email: "student1@uab.edu.bo", ExternalID: "otro"}, // correo repetido
			{Email: "student2@uab.edu.bo", ExternalID: "202012346"},
			{Email: "202012345", ExternalID: "202012347"}, // sin correo válido
		},
		Reconcile: true,
	})
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}

	if repo.calls != 1 {
		t.Fatalf("se esperaba una sola sincronización, hubo %d", repo.calls)
	}
	if repo.lastReq.SourceSystem != rosterDefaultSourceSystem {
		t.Errorf("origen por defecto esperado %q, obtenido %q", rosterDefaultSourceSystem, repo.lastReq.SourceSystem)
	}
	if !repo.lastReq.Reconcile {
		t.Error("la reconciliación pedida no llegó al repositorio")
	}
	if len(repo.lastReq.Entries) != 2 {
		t.Fatalf("se esperaban 2 filas válidas, llegaron %d: %+v", len(repo.lastReq.Entries), repo.lastReq.Entries)
	}
	if first := repo.lastReq.Entries[0]; first.Email != "student1@uab.edu.bo" || first.ExternalID != "202012345" || first.FirstName != "Ana" {
		t.Errorf("fila normalizada inesperada: %+v", first)
	}

	if result.EnrolledStudents != 3 || result.CreatedStudents != 2 || result.RemovedStudents != 1 {
		t.Errorf("resultado mal mapeado: %+v", result)
	}
	if result.MatchedByExternalID != 2 || result.SkippedNotFound != 1 {
		t.Errorf("conteos de identificación mal mapeados: %+v", result)
	}
	if result.TotalDetected != 2 {
		t.Errorf("total detectado debe contar filas válidas: %d", result.TotalDetected)
	}
}

func TestRosterEnrollFromSource_RechazaReconciliarOrigenManual(t *testing.T) {
	repo := &mockRosterSyncRepo{outcome: &domain.RosterSyncOutcome{}}
	svc := NewAdminGovernanceService(nil, repo)

	_, err := svc.RosterEnrollFromSource(context.Background(), "tenant-1", "subject-9", domain.RosterSyncRequest{
		Entries:      []domain.RosterEntry{{Email: "student1@uab.edu.bo"}},
		SourceSystem: "manual",
		Reconcile:    true,
	})
	if err == nil {
		t.Fatal("se esperaba error al reconciliar una nómina de origen manual")
	}
	if repo.calls != 0 {
		t.Error("no debe llegar al repositorio una reconciliación manual")
	}
}

func TestRosterEnrollFromSource_SinFilasValidas(t *testing.T) {
	repo := &mockRosterSyncRepo{outcome: &domain.RosterSyncOutcome{}}
	svc := NewAdminGovernanceService(nil, repo)

	if _, err := svc.RosterEnrollFromSource(context.Background(), "tenant-1", "subject-9", domain.RosterSyncRequest{
		Entries: []domain.RosterEntry{{Email: "sin-arroba"}},
	}); err == nil {
		t.Fatal("se esperaba error con una nómina sin correos válidos")
	}
}

func TestRosterEnrollFromSource_PropagaErrorDelRepositorio(t *testing.T) {
	repo := &mockRosterSyncRepo{err: errors.New("materia inexistente")}
	svc := NewAdminGovernanceService(nil, repo)

	if _, err := svc.RosterEnrollFromSource(context.Background(), "tenant-1", "subject-9", domain.RosterSyncRequest{
		Entries: []domain.RosterEntry{{Email: "student1@uab.edu.bo"}},
	}); err == nil {
		t.Fatal("se esperaba que el error del repositorio subiera")
	}
}

// Un repositorio sin la capacidad de sincronización conserva el camino
// anterior: alta y matrícula, sin bajas.
func TestRosterEnrollFromSource_SinCapacidadDeSincronizacion(t *testing.T) {
	repo := &mockStudentBatchGovRepo{createResult: 1, enrollResult: 1}
	svc := NewAdminGovernanceService(nil, repo)

	result, err := svc.RosterEnrollFromSource(context.Background(), "tenant-1", "subject-9", domain.RosterSyncRequest{
		Entries: []domain.RosterEntry{{Email: "Student1@UAB.edu.bo"}},
	})
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if repo.enrolledCalled != 1 {
		t.Fatal("el camino de compatibilidad debe matricular")
	}
	if len(repo.lastEnrollEmails) != 1 || repo.lastEnrollEmails[0] != "student1@uab.edu.bo" {
		t.Errorf("correo normalizado inesperado: %v", repo.lastEnrollEmails)
	}
	if result.RemovedStudents != 0 {
		t.Errorf("sin capacidad de sincronización no hay bajas: %d", result.RemovedStudents)
	}
}
