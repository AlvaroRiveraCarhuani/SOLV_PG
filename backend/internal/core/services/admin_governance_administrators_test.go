package services_test

import (
	"context"
	"errors"
	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
	"testing"
	"time"
)

type mockAdminGovRepoForAdmins struct {
	mockAdminGovernanceRepo
	admins           []*domain.AdminUserItem
	activeCount      int
	createCalled     bool
	revokeCalled     bool
	lastCreatedEmail string
	lastRevokedID    string
}

func (m *mockAdminGovRepoForAdmins) ListAdministrators(ctx context.Context, tenantID string) ([]*domain.AdminUserItem, error) {
	return m.admins, nil
}

func (m *mockAdminGovRepoForAdmins) CountActiveAdministrators(ctx context.Context, tenantID string) (int, error) {
	return m.activeCount, nil
}

func (m *mockAdminGovRepoForAdmins) CreateAdministrator(ctx context.Context, tenantID, email, firstName, lastName string) (*domain.AdminUserItem, error) {
	m.createCalled = true
	m.lastCreatedEmail = email
	item := &domain.AdminUserItem{
		ID:        "admin-new-uuid",
		Email:     email,
		FirstName: firstName,
		LastName:  lastName,
		Role:      "admin",
		Status:    "active",
		Origin:    "manual",
		CreatedAt: time.Now(),
	}
	m.admins = append(m.admins, item)
	return item, nil
}

func (m *mockAdminGovRepoForAdmins) RevokeAdministrator(ctx context.Context, tenantID, targetUserID string) error {
	m.revokeCalled = true
	m.lastRevokedID = targetUserID
	return nil
}

type mockAuditRepoForAdmins struct {
	logs []*domain.AuditLog
}

func (m *mockAuditRepoForAdmins) Create(ctx context.Context, log *domain.AuditLog) error {
	m.logs = append(m.logs, log)
	return nil
}

func (m *mockAuditRepoForAdmins) ListByTenant(ctx context.Context, tenantID string, limit int) ([]*domain.AuditLog, error) {
	return nil, nil
}
func (m *mockAuditRepoForAdmins) ListFiltered(ctx context.Context, tenantID, search, action string, limit, offset int) ([]*domain.AuditLog, error) {
	return nil, nil
}
func (m *mockAuditRepoForAdmins) CountFiltered(ctx context.Context, tenantID, search, action string) (int, error) {
	return 0, nil
}
func (m *mockAuditRepoForAdmins) ListByActorTimeline(ctx context.Context, tenantID, actorID string, limit int) ([]*domain.AuditLog, error) {
	return nil, nil
}

func TestAdminGovernanceService_ListAdministrators(t *testing.T) {
	ctx := context.Background()
	repo := &mockAdminGovRepoForAdmins{
		admins: []*domain.AdminUserItem{
			{ID: "admin-1", Email: "admin1@uab.edu.bo", Role: "admin"},
			{ID: "admin-2", Email: "admin2@uab.edu.bo", Role: "admin"},
		},
	}
	svc := services.NewAdminGovernanceService(nil, repo)

	list, err := svc.ListAdministrators(ctx, "tenant-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 admins, got %d", len(list))
	}
}

func TestAdminGovernanceService_CreateAdministrator_Validations(t *testing.T) {
	ctx := context.Background()
	repo := &mockAdminGovRepoForAdmins{}
	svc := services.NewAdminGovernanceService(nil, repo)

	// Email vacío
	_, err := svc.CreateAdministrator(ctx, "tenant-1", "admin-req", domain.CreateAdminDTO{
		FirstName: "Juan",
		LastName:  "Perez",
		Email:     "",
	})
	if !errors.Is(err, domain.ErrInvalidAdminEmail) {
		t.Errorf("expected ErrInvalidAdminEmail for empty email, got %v", err)
	}

	// Email malformado
	_, err = svc.CreateAdministrator(ctx, "tenant-1", "admin-req", domain.CreateAdminDTO{
		FirstName: "Juan",
		LastName:  "Perez",
		Email:     "no-es-correo",
	})
	if !errors.Is(err, domain.ErrInvalidAdminEmail) {
		t.Errorf("expected ErrInvalidAdminEmail for malformed email, got %v", err)
	}

	// Nombre vacío
	_, err = svc.CreateAdministrator(ctx, "tenant-1", "admin-req", domain.CreateAdminDTO{
		FirstName: "",
		LastName:  "Perez",
		Email:     "juan@uab.edu.bo",
	})
	if !errors.Is(err, domain.ErrAdminNamesRequired) {
		t.Errorf("expected ErrAdminNamesRequired for empty first name, got %v", err)
	}

	// Apellido vacío
	_, err = svc.CreateAdministrator(ctx, "tenant-1", "admin-req", domain.CreateAdminDTO{
		FirstName: "Juan",
		LastName:  "   ",
		Email:     "juan@uab.edu.bo",
	})
	if !errors.Is(err, domain.ErrAdminNamesRequired) {
		t.Errorf("expected ErrAdminNamesRequired for empty last name, got %v", err)
	}
}

func TestAdminGovernanceService_CreateAdministrator_SuccessAndAudit(t *testing.T) {
	ctx := context.Background()
	repo := &mockAdminGovRepoForAdmins{}
	audit := &mockAuditRepoForAdmins{}
	svc := services.NewAdminGovernanceService(nil, repo)
	svc.SetAuditRepo(audit)

	admin, err := svc.CreateAdministrator(ctx, "tenant-1", "admin-requester", domain.CreateAdminDTO{
		FirstName: "Carlos",
		LastName:  "Gomez",
		Email:     "  Carlos.Gomez@UAB.edu.bo  ",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if admin.Email != "carlos.gomez@uab.edu.bo" {
		t.Errorf("expected email to be normalized, got %q", admin.Email)
	}
	if !repo.createCalled {
		t.Errorf("expected repo.CreateAdministrator to be called")
	}

	if len(audit.logs) != 1 {
		t.Fatalf("expected 1 audit log, got %d", len(audit.logs))
	}
	if audit.logs[0].Action != "ADMINISTRATOR_CREATED" {
		t.Errorf("expected action ADMINISTRATOR_CREATED, got %q", audit.logs[0].Action)
	}
	if audit.logs[0].ActorID != "admin-requester" {
		t.Errorf("expected actor admin-requester, got %q", audit.logs[0].ActorID)
	}
}

func TestAdminGovernanceService_RevokeAdministrator_CannotRevokeSelf(t *testing.T) {
	ctx := context.Background()
	repo := &mockAdminGovRepoForAdmins{activeCount: 3}
	svc := services.NewAdminGovernanceService(nil, repo)

	err := svc.RevokeAdministrator(ctx, "tenant-1", "admin-123", "admin-123")
	if !errors.Is(err, domain.ErrCannotRevokeSelf) {
		t.Errorf("expected ErrCannotRevokeSelf when target equals requester, got %v", err)
	}
	if repo.revokeCalled {
		t.Errorf("repo.RevokeAdministrator should not have been called")
	}
}

func TestAdminGovernanceService_RevokeAdministrator_LastAdministratorBlocked(t *testing.T) {
	ctx := context.Background()
	repo := &mockAdminGovRepoForAdmins{activeCount: 1}
	svc := services.NewAdminGovernanceService(nil, repo)

	err := svc.RevokeAdministrator(ctx, "tenant-1", "admin-target", "admin-requester")
	if !errors.Is(err, domain.ErrLastAdministrator) {
		t.Errorf("expected ErrLastAdministrator when activeCount <= 1, got %v", err)
	}
	if repo.revokeCalled {
		t.Errorf("repo.RevokeAdministrator should not have been called")
	}
}

func TestAdminGovernanceService_RevokeAdministrator_Success(t *testing.T) {
	ctx := context.Background()
	repo := &mockAdminGovRepoForAdmins{activeCount: 2}
	audit := &mockAuditRepoForAdmins{}
	svc := services.NewAdminGovernanceService(nil, repo)
	svc.SetAuditRepo(audit)

	err := svc.RevokeAdministrator(ctx, "tenant-1", "admin-target", "admin-requester")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.revokeCalled {
		t.Errorf("expected repo.RevokeAdministrator to be called")
	}
	if repo.lastRevokedID != "admin-target" {
		t.Errorf("expected lastRevokedID 'admin-target', got %q", repo.lastRevokedID)
	}

	if len(audit.logs) != 1 {
		t.Fatalf("expected 1 audit log, got %d", len(audit.logs))
	}
	if audit.logs[0].Action != "ADMINISTRATOR_REVOKED" {
		t.Errorf("expected action ADMINISTRATOR_REVOKED, got %q", audit.logs[0].Action)
	}
}
