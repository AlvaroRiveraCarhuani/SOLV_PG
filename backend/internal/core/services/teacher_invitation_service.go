package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"solv-backend/internal/core/domain"

	"github.com/google/uuid"
)

type TeacherInvitationService struct {
	repo domain.TeacherInvitationRepository
}

func NewTeacherInvitationService(repo domain.TeacherInvitationRepository) *TeacherInvitationService {
	return &TeacherInvitationService{repo: repo}
}

func (s *TeacherInvitationService) CreateInvitation(ctx context.Context, tenantID, email, origin, roleType string, durationHours int) (*domain.TeacherInvitation, error) {
	if email == "" {
		return nil, errors.New("email is required")
	}
	if durationHours <= 0 {
		durationHours = 72 // ADR-025: TTL 72h
	}
	if origin == "" {
		origin = "manual"
	}
	if roleType == "" {
		roleType = "titular"
	}

	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return nil, fmt.Errorf("failed to generate random token: %w", err)
	}
	token := hex.EncodeToString(bytes)

	inv := &domain.TeacherInvitation{
		ID:        uuid.New().String(),
		TenantID:  tenantID,
		Token:     token,
		Email:     email,
		Origin:    origin,
		RoleType:  roleType,
		Used:      false,
		ExpiresAt: time.Now().Add(time.Duration(durationHours) * time.Hour),
	}

	if err := s.repo.Create(ctx, inv); err != nil {
		return nil, fmt.Errorf("failed to create teacher invitation: %w", err)
	}
	return inv, nil
}

func (s *TeacherInvitationService) ListTeachers(ctx context.Context, tenantID, search, status, origin string) ([]*domain.TeacherListItem, error) {
	return s.repo.ListTeachers(ctx, tenantID, search, status, origin)
}

func (s *TeacherInvitationService) ResendInvitation(ctx context.Context, tenantID, invitationID string) (*domain.TeacherInvitation, error) {
	inv, err := s.repo.GetByID(ctx, tenantID, invitationID)
	if err != nil {
		return nil, err
	}
	if inv.Used {
		return nil, errors.New("la invitación ya fue aceptada previamente")
	}
	// Si está expirada, renovarla automáticamente por 72h
	if time.Now().After(inv.ExpiresAt) {
		return s.RenewInvitation(ctx, tenantID, invitationID)
	}
	return inv, nil
}

func (s *TeacherInvitationService) RenewInvitation(ctx context.Context, tenantID, invitationID string) (*domain.TeacherInvitation, error) {
	inv, err := s.repo.GetByID(ctx, tenantID, invitationID)
	if err != nil {
		return nil, err
	}
	if inv.Used {
		return nil, errors.New("la invitación ya fue aceptada y no puede renovarse")
	}

	// Emitir nuevo token sin duplicar fila (ADR-025)
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return nil, fmt.Errorf("failed to generate random token: %w", err)
	}
	inv.Token = hex.EncodeToString(bytes)
	inv.ExpiresAt = time.Now().Add(72 * time.Hour)

	if err := s.repo.Update(ctx, inv); err != nil {
		return nil, err
	}
	return inv, nil
}

func (s *TeacherInvitationService) AcceptInvitation(ctx context.Context, tenantID, token, userID, userEmail string) error {
	if token == "" || userID == "" || userEmail == "" {
		return errors.New("token, userID and userEmail are required")
	}
	return s.repo.AcceptInvitationTx(ctx, tenantID, token, userID, userEmail)
}

func (s *TeacherInvitationService) GetTeacherCourses(ctx context.Context, tenantID, teacherID string) ([]*domain.TeacherCourseItem, error) {
	return s.repo.GetTeacherCourses(ctx, tenantID, teacherID)
}

func (s *TeacherInvitationService) DeleteInvitation(ctx context.Context, tenantID, id string) error {
	return s.repo.DeleteInvitation(ctx, tenantID, id)
}
