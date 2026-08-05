package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"solv-backend/internal/core/domain"
	"time"

	"github.com/google/uuid"
)

type TeacherInvitationService struct {
	repo domain.TeacherInvitationRepository
}

func NewTeacherInvitationService(repo domain.TeacherInvitationRepository) *TeacherInvitationService {
	return &TeacherInvitationService{repo: repo}
}

func (s *TeacherInvitationService) CreateInvitation(ctx context.Context, tenantID, email string, durationHours int) (*domain.TeacherInvitation, error) {
	if email == "" {
		return nil, errors.New("email is required")
	}
	if durationHours <= 0 {
		durationHours = 48
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
		Used:      false,
		ExpiresAt: time.Now().Add(time.Duration(durationHours) * time.Hour),
	}

	if err := s.repo.Create(ctx, inv); err != nil {
		return nil, fmt.Errorf("failed to create teacher invitation: %w", err)
	}
	return inv, nil
}

func (s *TeacherInvitationService) AcceptInvitation(ctx context.Context, tenantID, token, userID, userEmail string) error {
	if token == "" || userID == "" || userEmail == "" {
		return errors.New("token, userID and userEmail are required")
	}
	return s.repo.AcceptInvitationTx(ctx, tenantID, token, userID, userEmail)
}
