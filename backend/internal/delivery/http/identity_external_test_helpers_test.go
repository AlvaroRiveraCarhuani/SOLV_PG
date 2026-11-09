package httpdelivery_test

import (
	"context"
	"net/http"

	"solv-backend/internal/core/domain"
)

// withTestIdentity simula lo que hacen WithAuth/WithTenant después de validar
// la sesión. Los tests que invocan handlers directamente deben usarlo en lugar
// de cabeceras de identidad.
func withTestIdentity(r *http.Request, userID, tenantID, role string) *http.Request {
	ctx := r.Context()
	if userID != "" {
		ctx = context.WithValue(ctx, domain.UserIDKey, userID)
	}
	if tenantID != "" {
		ctx = context.WithValue(ctx, domain.TenantIDKey, tenantID)
	}
	if role != "" {
		ctx = context.WithValue(ctx, domain.UserRoleKey, role)
	}
	return r.WithContext(ctx)
}

func withTestEmail(r *http.Request, email string) *http.Request {
	if email == "" {
		return r
	}
	return r.WithContext(context.WithValue(r.Context(), domain.UserEmailKey, email))
}
