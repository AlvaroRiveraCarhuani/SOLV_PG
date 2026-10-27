package domain

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

type Tenant struct {
	ID             string    `db:"id" json:"id"`
	Name           string    `db:"name" json:"name"`
	Slug           string    `db:"slug" json:"slug"`
	AllowedDomains []byte    `db:"allowed_domains" json:"allowed_domains"` // JSONB array (e.g. ["@uab.edu.bo"])
	Config         []byte    `db:"config" json:"config"`                   // JSONB object
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time `db:"updated_at" json:"updated_at"`
}

// EmailBranding encapsula los metadatos institucionales requeridos para el renderizado
// de correos transaccionales multi-tenant y marca blanca.
type EmailBranding struct {
	InstitutionName string `json:"institution_name"`
	LogoURL         string `json:"logo_url"`
	BaseDomain      string `json:"base_domain"`
	SupportEmail    string `json:"support_email"`
}

// AbsoluteLogoURL retorna la URL pública canónica del imagotipo institucional.
func (b EmailBranding) AbsoluteLogoURL() string {
	if b.LogoURL == "" {
		return ""
	}
	if strings.HasPrefix(b.LogoURL, "http://") || strings.HasPrefix(b.LogoURL, "https://") {
		return b.LogoURL
	}
	domain := strings.TrimSpace(b.BaseDomain)
	if domain == "" {
		return ""
	}
	logoPath := b.LogoURL
	if !strings.HasPrefix(logoPath, "/") {
		logoPath = "/" + logoPath
	}
	return fmt.Sprintf("https://%s%s", domain, logoPath)
}

// ResolveBranding extrae y normaliza la configuración de marca del tenant.
func (t *Tenant) ResolveBranding() EmailBranding {
	branding := EmailBranding{
		InstitutionName: t.Name,
		BaseDomain:      t.Slug,
	}
	if len(t.Config) > 0 {
		var cfg struct {
			InstitutionName string `json:"institution_name"`
			LogoURL         string `json:"logo_url"`
			BaseDomain      string `json:"base_domain"`
			SupportEmail    string `json:"support_email"`
		}
		if err := json.Unmarshal(t.Config, &cfg); err == nil {
			if strings.TrimSpace(cfg.InstitutionName) != "" {
				branding.InstitutionName = strings.TrimSpace(cfg.InstitutionName)
			}
			if strings.TrimSpace(cfg.LogoURL) != "" {
				branding.LogoURL = strings.TrimSpace(cfg.LogoURL)
			}
			if strings.TrimSpace(cfg.BaseDomain) != "" {
				branding.BaseDomain = strings.TrimSpace(cfg.BaseDomain)
			}
			if strings.TrimSpace(cfg.SupportEmail) != "" {
				branding.SupportEmail = strings.TrimSpace(cfg.SupportEmail)
			}
		}
	}
	if strings.TrimSpace(branding.InstitutionName) == "" {
		branding.InstitutionName = "Plataforma Académica"
	}
	return branding
}

type contextKey string

const (
	TenantIDKey contextKey = "tenant_id"
	UserIDKey   contextKey = "user_id"
	UserRoleKey contextKey = "user_role"

	DefaultTenantID = "00000000-0000-0000-0000-000000000001"
)

func GetTenantID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if val, ok := ctx.Value(TenantIDKey).(string); ok {
		return val
	}
	return ""
}

func GetUserID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if val, ok := ctx.Value(UserIDKey).(string); ok {
		return val
	}
	return ""
}

func GetUserRole(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if val, ok := ctx.Value(UserRoleKey).(string); ok {
		return val
	}
	return ""
}

var ErrTenantIDMissing = errors.New("missing tenant_id in context")
var ErrInvalidTenant = errors.New("invalid tenant")
var ErrNotFound = errors.New("resource not found")
