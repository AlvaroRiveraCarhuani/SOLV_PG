package services

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"solv-backend/internal/core/domain"
)

// tenantSlugPattern acota el identificador a lo que puede viajar en un
// subdominio y en una ruta sin escapes.
var tenantSlugPattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)

// TenantProvisioningService valida y delega el alta de una institución.
type TenantProvisioningService struct {
	repo domain.TenantProvisioningRepository
}

func NewTenantProvisioningService(repo domain.TenantProvisioningRepository) *TenantProvisioningService {
	return &TenantProvisioningService{repo: repo}
}

// Provision crea la institución, su primer administrador y el catálogo base.
//
// El catálogo base es deliberadamente pequeño: un periodo académico vigente y
// una materia ancla. Todo lo demás lo arma el administrador desde el panel. Un
// catálogo precargado con materias que la institución no dicta es peor que un
// catálogo vacío, porque hay que borrarlo antes de empezar.
func (s *TenantProvisioningService) Provision(ctx context.Context, req domain.TenantProvisioningRequest) (*domain.TenantProvisioningResult, error) {
	req.Name = strings.TrimSpace(req.Name)
	req.Slug = strings.ToLower(strings.TrimSpace(req.Slug))
	req.AdminEmail = strings.ToLower(strings.TrimSpace(req.AdminEmail))
	req.AdminFirstName = strings.TrimSpace(req.AdminFirstName)
	req.AdminLastName = strings.TrimSpace(req.AdminLastName)

	if len(req.Name) < 2 {
		return nil, errors.New("el nombre de la institución es obligatorio")
	}
	if !tenantSlugPattern.MatchString(req.Slug) {
		return nil, fmt.Errorf("identificador inválido %q: usá minúsculas, números y guiones", req.Slug)
	}
	if !isValidEmailAddress(req.AdminEmail) {
		return nil, errors.New("el correo del administrador no es válido")
	}
	if req.AdminFirstName == "" || req.AdminLastName == "" {
		return nil, errors.New("el nombre y el apellido del administrador son obligatorios")
	}

	req.AllowedDomains = normalizeAllowedDomains(req.AllowedDomains)

	if req.PeriodCode == "" {
		req.PeriodCode = fmt.Sprintf("%d-I", time.Now().Year())
	}
	if req.PeriodName == "" {
		req.PeriodName = fmt.Sprintf("Primer Semestre %d", time.Now().Year())
	}
	if req.PeriodStart.IsZero() || req.PeriodEnd.IsZero() {
		return nil, errors.New("el periodo académico necesita fecha de inicio y de fin")
	}
	if !req.PeriodEnd.After(req.PeriodStart) {
		return nil, errors.New("el periodo académico termina antes de empezar")
	}

	return s.repo.ProvisionTenant(ctx, req)
}

// normalizeAllowedDomains deja cada dominio con su arroba inicial y sin
// duplicados. Un dominio vacío significa "sin filtro de dominio": la institución
// autentica por su propia lista y las cuentas se crean por invitación.
func normalizeAllowedDomains(domains []string) []string {
	cleaned := make([]string, 0, len(domains))
	seen := make(map[string]bool, len(domains))
	for _, raw := range domains {
		d := strings.ToLower(strings.TrimSpace(raw))
		d = strings.TrimPrefix(d, "@")
		if d == "" || seen[d] {
			continue
		}
		seen[d] = true
		cleaned = append(cleaned, "@"+d)
	}
	return cleaned
}
