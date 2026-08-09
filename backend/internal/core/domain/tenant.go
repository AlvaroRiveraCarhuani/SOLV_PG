package domain

import (
	"context"
	"errors"
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

