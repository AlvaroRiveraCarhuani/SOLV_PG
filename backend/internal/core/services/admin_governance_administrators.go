package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/mail"
	"solv-backend/internal/core/domain"
	"strings"
)

// ListAdministrators devuelve todos los usuarios con rol admin pertenecientes al tenant.
func (s *AdminGovernanceService) ListAdministrators(
	ctx context.Context,
	tenantID string,
) ([]*domain.AdminUserItem, error) {
	if s.govRepo == nil {
		return []*domain.AdminUserItem{}, nil
	}
	return s.govRepo.ListAdministrators(ctx, tenantID)
}

// CreateAdministrator valida y registra o promueve a un usuario al rol de administrador.
func (s *AdminGovernanceService) CreateAdministrator(
	ctx context.Context,
	tenantID, requestingUserID string,
	dto domain.CreateAdminDTO,
) (*domain.AdminUserItem, error) {
	normalizedEmail := strings.ToLower(strings.TrimSpace(dto.Email))
	trimmedFirst := strings.TrimSpace(dto.FirstName)
	trimmedLast := strings.TrimSpace(dto.LastName)

	if normalizedEmail == "" || len(normalizedEmail) > 255 {
		return nil, domain.ErrInvalidAdminEmail
	}
	if _, err := mail.ParseAddress(normalizedEmail); err != nil {
		return nil, domain.ErrInvalidAdminEmail
	}
	if trimmedFirst == "" || trimmedLast == "" {
		return nil, domain.ErrAdminNamesRequired
	}

	created, err := s.govRepo.CreateAdministrator(ctx, tenantID, normalizedEmail, trimmedFirst, trimmedLast)
	if err != nil {
		return nil, err
	}

	if s.auditRepo != nil && requestingUserID != "" {
		meta, _ := json.Marshal(map[string]interface{}{
			"target_email": created.Email,
			"target_name":  fmt.Sprintf("%s %s", created.FirstName, created.LastName),
			"role":         "admin",
		})
		_ = s.auditRepo.Create(ctx, &domain.AuditLog{
			TenantID:     tenantID,
			ActorID:      requestingUserID,
			Action:       "ADMINISTRATOR_CREATED",
			ResourceType: "user",
			ResourceID:   &created.ID,
			StatusCode:   201,
			Metadata:     meta,
		})
	}

	return created, nil
}

// RevokeAdministrator valida invariantes de seguridad y revoca los privilegios de un administrador.
func (s *AdminGovernanceService) RevokeAdministrator(
	ctx context.Context,
	tenantID, targetUserID, requestingUserID string,
) error {
	trimmedTarget := strings.TrimSpace(targetUserID)
	trimmedRequesting := strings.TrimSpace(requestingUserID)

	if trimmedTarget == "" {
		return domain.ErrAdminNotFound
	}

	// Invariante 1: No es posible revocar los propios privilegios
	if trimmedTarget == trimmedRequesting {
		return domain.ErrCannotRevokeSelf
	}

	// Invariante 2: Garantía de no-orfandad (mínimo 1 administrador activo)
	activeCount, err := s.govRepo.CountActiveAdministrators(ctx, tenantID)
	if err != nil {
		return err
	}
	if activeCount <= 1 {
		return domain.ErrLastAdministrator
	}

	if err := s.govRepo.RevokeAdministrator(ctx, tenantID, trimmedTarget); err != nil {
		return err
	}

	if s.auditRepo != nil && trimmedRequesting != "" {
		meta, _ := json.Marshal(map[string]interface{}{
			"target_user_id": trimmedTarget,
			"action":         "revoke_admin_role",
		})
		_ = s.auditRepo.Create(ctx, &domain.AuditLog{
			TenantID:     tenantID,
			ActorID:      trimmedRequesting,
			Action:       "ADMINISTRATOR_REVOKED",
			ResourceType: "user",
			ResourceID:   &trimmedTarget,
			StatusCode:   200,
			Metadata:     meta,
		})
	}

	return nil
}
