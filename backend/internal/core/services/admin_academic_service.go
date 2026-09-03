package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"solv-backend/internal/core/domain"
)

var (
	ErrInvalidDateRange = errors.New("end_date must be equal to or after start_date")
	ErrConflict         = errors.New("conflict: resource has dependencies")
)

type AcademicPeriodService struct {
	repo domain.AcademicPeriodRepository
}

func NewAcademicPeriodService(repo domain.AcademicPeriodRepository) *AcademicPeriodService {
	return &AcademicPeriodService{repo: repo}
}

func parseDateFlexible(s string) (time.Time, error) {
	// Intentar YYYY-MM-DD primero
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t, nil
	}
	// Intentar RFC3339
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("invalid date format, expected YYYY-MM-DD or RFC3339: %s", s)
}

func (s *AcademicPeriodService) CreatePeriod(ctx context.Context, tenantID string, dto domain.CreateAcademicPeriodDTO) (*domain.AcademicPeriod, error) {
	startDate, err := parseDateFlexible(dto.StartDate)
	if err != nil {
		return nil, err
	}
	endDate, err := parseDateFlexible(dto.EndDate)
	if err != nil {
		return nil, err
	}

	if endDate.Before(startDate) {
		return nil, ErrInvalidDateRange
	}

	isActive := true
	if dto.IsActive != nil {
		isActive = *dto.IsActive
	}

	period := &domain.AcademicPeriod{
		ID:        uuid.NewString(),
		TenantID:  tenantID,
		Name:      dto.Name,
		Code:      dto.Code,
		StartDate: startDate,
		EndDate:   endDate,
		IsActive:  isActive,
	}

	if err := s.repo.Create(ctx, period); err != nil {
		return nil, err
	}

	return period, nil
}

func (s *AcademicPeriodService) GetPeriod(ctx context.Context, tenantID, id string) (*domain.AcademicPeriod, error) {
	return s.repo.GetByID(ctx, tenantID, id)
}

func (s *AcademicPeriodService) ListPeriods(ctx context.Context, tenantID string) ([]*domain.AcademicPeriod, error) {
	return s.repo.ListByTenant(ctx, tenantID)
}

func (s *AcademicPeriodService) UpdatePeriod(ctx context.Context, tenantID, id string, dto domain.UpdateAcademicPeriodDTO) (*domain.AcademicPeriod, error) {
	period, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	startDate, err := parseDateFlexible(dto.StartDate)
	if err != nil {
		return nil, err
	}
	endDate, err := parseDateFlexible(dto.EndDate)
	if err != nil {
		return nil, err
	}

	if endDate.Before(startDate) {
		return nil, ErrInvalidDateRange
	}

	period.Name = dto.Name
	period.Code = dto.Code
	period.StartDate = startDate
	period.EndDate = endDate
	if dto.IsActive != nil {
		period.IsActive = *dto.IsActive
	}

	if err := s.repo.Update(ctx, period); err != nil {
		return nil, err
	}

	return period, nil
}

func (s *AcademicPeriodService) DeletePeriod(ctx context.Context, tenantID, id string) error {
	return s.repo.Delete(ctx, tenantID, id)
}

func (s *AcademicPeriodService) ArchiveExpiredPeriods(ctx context.Context) (int64, error) {
	return s.repo.ArchiveExpiredPeriods(ctx)
}

// -----------------------------------------------------------------------------
// MaintenanceService (ADR-031)
// -----------------------------------------------------------------------------

type MaintenanceService struct {
	tenantRepo domain.TenantRepository
}

func NewMaintenanceService(tenantRepo domain.TenantRepository) *MaintenanceService {
	return &MaintenanceService{tenantRepo: tenantRepo}
}

func (s *MaintenanceService) EnableMaintenance(ctx context.Context, tenantID string, dto domain.EnableMaintenanceDTO) error {
	var until *time.Time
	if dto.Until != "" {
		t, err := time.Parse(time.RFC3339, dto.Until)
		if err != nil {
			// Intentar formato sin zona o simple
			t2, err2 := time.Parse("2006-01-02T15:04:05", dto.Until)
			if err2 != nil {
				return fmt.Errorf("invalid until format, expected RFC3339: %w", err)
			}
			t = t2
		}
		until = &t
	}

	return s.tenantRepo.SetMaintenance(ctx, tenantID, true, until, dto.Reason)
}

func (s *MaintenanceService) DisableMaintenance(ctx context.Context, tenantID string) error {
	return s.tenantRepo.SetMaintenance(ctx, tenantID, false, nil, "")
}

func (s *MaintenanceService) GetStatus(ctx context.Context, tenantID string) (*domain.MaintenanceStatus, error) {
	return s.tenantRepo.GetMaintenance(ctx, tenantID)
}
