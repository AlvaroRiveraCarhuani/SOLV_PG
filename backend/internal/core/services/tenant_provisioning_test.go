package services

import (
	"context"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
)

type mockTenantProvisioningRepo struct {
	calls   int
	lastReq domain.TenantProvisioningRequest
	result  *domain.TenantProvisioningResult
	err     error
}

func (m *mockTenantProvisioningRepo) ProvisionTenant(ctx context.Context, req domain.TenantProvisioningRequest) (*domain.TenantProvisioningResult, error) {
	m.calls++
	m.lastReq = req
	if m.err != nil {
		return nil, m.err
	}
	if m.result != nil {
		return m.result, nil
	}
	return &domain.TenantProvisioningResult{TenantID: "t-1", Slug: req.Slug}, nil
}

func (m *mockTenantProvisioningRepo) ListTenants(ctx context.Context) ([]*domain.TenantSummary, error) {
	return nil, m.err
}

func validProvisioningRequest() domain.TenantProvisioningRequest {
	return domain.TenantProvisioningRequest{
		Name:           "Universidad Ejemplo",
		Slug:           "uejemplo",
		AllowedDomains: []string{"UEjemplo.edu", "@uejemplo.edu.bo", " "},
		AdminEmail:     "Admin@UEjemplo.edu",
		AdminFirstName: "Ada",
		AdminLastName:  "Lovelace",
		PeriodCode:     "2026-I",
		PeriodName:     "Primer Semestre 2026",
		PeriodStart:    time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		PeriodEnd:      time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC),
	}
}

func TestProvisionTenant_NormalizaYDelega(t *testing.T) {
	repo := &mockTenantProvisioningRepo{}
	svc := NewTenantProvisioningService(repo)

	result, err := svc.Provision(context.Background(), validProvisioningRequest())
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if repo.calls != 1 {
		t.Fatalf("se esperaba una llamada al repositorio, hubo %d", repo.calls)
	}
	if repo.lastReq.AdminEmail != "admin@uejemplo.edu" {
		t.Errorf("correo sin normalizar: %q", repo.lastReq.AdminEmail)
	}
	if repo.lastReq.Slug != "uejemplo" {
		t.Errorf("slug inesperado: %q", repo.lastReq.Slug)
	}
	domains := repo.lastReq.AllowedDomains
	if len(domains) != 2 || domains[0] != "@uejemplo.edu" || domains[1] != "@uejemplo.edu.bo" {
		t.Errorf("dominios normalizados inesperados: %v", domains)
	}
	if result.TenantID != "t-1" {
		t.Errorf("resultado no propagado: %+v", result)
	}
}

func TestProvisionTenant_RechazaSlugInvalido(t *testing.T) {
	repo := &mockTenantProvisioningRepo{}
	svc := NewTenantProvisioningService(repo)

	for _, slug := range []string{"", "con espacio", "guion-final-", "-inicio", "ñandu", "con_guion_bajo"} {
		req := validProvisioningRequest()
		req.Slug = slug
		if _, err := svc.Provision(context.Background(), req); err == nil {
			t.Errorf("slug %q debería rechazarse", slug)
		}
	}
	if repo.calls != 0 {
		t.Error("un slug inválido no debe llegar al repositorio")
	}
}

// El slug se normaliza a minúsculas en vez de rechazarse: quien escribe
// "UEjemplo" está nombrando la misma institución.
func TestProvisionTenant_SlugSeNormalizaAMinusculas(t *testing.T) {
	repo := &mockTenantProvisioningRepo{}
	svc := NewTenantProvisioningService(repo)

	req := validProvisioningRequest()
	req.Slug = "  UEjemplo  "
	if _, err := svc.Provision(context.Background(), req); err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if repo.lastReq.Slug != "uejemplo" {
		t.Errorf("slug sin normalizar: %q", repo.lastReq.Slug)
	}
}

func TestProvisionTenant_ExigeAdministradorConNombre(t *testing.T) {
	repo := &mockTenantProvisioningRepo{}
	svc := NewTenantProvisioningService(repo)

	req := validProvisioningRequest()
	req.AdminLastName = "   "
	if _, err := svc.Provision(context.Background(), req); err == nil {
		t.Fatal("se esperaba error sin apellido del administrador")
	}

	req = validProvisioningRequest()
	req.AdminEmail = "no-es-correo"
	if _, err := svc.Provision(context.Background(), req); err == nil {
		t.Fatal("se esperaba error con un correo inválido")
	}
}

func TestProvisionTenant_PeriodoCoherente(t *testing.T) {
	repo := &mockTenantProvisioningRepo{}
	svc := NewTenantProvisioningService(repo)

	req := validProvisioningRequest()
	req.PeriodEnd = req.PeriodStart.Add(-24 * time.Hour)
	if _, err := svc.Provision(context.Background(), req); err == nil {
		t.Fatal("se esperaba error con un periodo que termina antes de empezar")
	}

	req = validProvisioningRequest()
	req.PeriodStart = time.Time{}
	if _, err := svc.Provision(context.Background(), req); err == nil {
		t.Fatal("se esperaba error sin fecha de inicio")
	}
}

func TestProvisionTenant_PropagaConflictoDeSlug(t *testing.T) {
	repo := &mockTenantProvisioningRepo{err: domain.ErrTenantSlugExists}
	svc := NewTenantProvisioningService(repo)

	if _, err := svc.Provision(context.Background(), validProvisioningRequest()); err != domain.ErrTenantSlugExists {
		t.Fatalf("se esperaba ErrTenantSlugExists, se obtuvo %v", err)
	}
}
