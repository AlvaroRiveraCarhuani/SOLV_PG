package domain

import (
	"context"
	"errors"
	"time"
)

// ErrTenantSlugExists indica que ya hay una institución con ese identificador
// corto. El slug es la clave pública de la institución (subdominio, correos y
// rutas), así que no se reutiliza.
var ErrTenantSlugExists = errors.New("ya existe una institución con ese identificador")

// ErrAdminEmailExists indica que el correo del primer administrador ya pertenece
// a una cuenta. Una persona pertenece a una institución, no a dos.
var ErrAdminEmailExists = errors.New("el correo del administrador ya tiene una cuenta")

// TenantProvisioningRequest describe el alta completa de una institución: el
// tenant, su primer administrador y el catálogo base mínimo para que el panel
// sea usable en el primer inicio de sesión.
type TenantProvisioningRequest struct {
	Name           string
	Slug           string
	AllowedDomains []string
	AdminEmail     string
	AdminFirstName string
	AdminLastName  string
	PeriodCode     string
	PeriodName     string
	PeriodStart    time.Time
	PeriodEnd      time.Time
}

// TenantProvisioningResult identifica lo que quedó creado.
type TenantProvisioningResult struct {
	TenantID   string
	Slug       string
	AdminID    string
	PeriodID   string
	SubjectID  string
	CatalogIDs []string
}

// TenantSummary es la fila que lista el comando de operación.
type TenantSummary struct {
	ID        string    `db:"id" json:"id"`
	Name      string    `db:"name" json:"name"`
	Slug      string    `db:"slug" json:"slug"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

// TenantProvisioningRepository crea una institución en una sola transacción.
// Se declara aparte de TenantRepository para no ampliar el contrato que ya usan
// los handlers y sus mocks.
type TenantProvisioningRepository interface {
	ProvisionTenant(ctx context.Context, req TenantProvisioningRequest) (*TenantProvisioningResult, error)
	ListTenants(ctx context.Context) ([]*TenantSummary, error)
}
