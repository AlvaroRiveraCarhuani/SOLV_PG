package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"solv-backend/internal/core/domain"
)

var (
	ErrInvalidDateRange   = errors.New("end_date must be equal to or after start_date")
	ErrPeriodExpired      = errors.New("cannot activate an expired academic period")
	ErrConflict           = errors.New("conflict: resource has dependencies")
	ErrNotFound           = errors.New("academic period not found")
	ErrConfirmationFailed = errors.New("confirmation code does not match period code")
	ErrPeriodArchived     = errors.New("academic period is formally archived and immutable")
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
	_, _ = s.repo.ArchiveExpiredPeriods(ctx)
	return s.repo.ListByTenant(ctx, tenantID)
}

func (s *AcademicPeriodService) UpdatePeriod(ctx context.Context, tenantID, id string, dto domain.UpdateAcademicPeriodDTO) (*domain.AcademicPeriod, error) {
	period, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	// Guardia de inmutabilidad: un periodo formalmente archivado no admite
	// cambios (ni reactivación) salvo la operación inversa explícita del
	// administrador de plataforma fuera de esta API (ADR-029).
	if period.IsArchived {
		return nil, ErrPeriodArchived
	}

	if dto.Name != "" {
		period.Name = dto.Name
	}
	if dto.Code != "" {
		period.Code = dto.Code
	}

	if dto.StartDate != "" {
		startDate, err := parseDateFlexible(dto.StartDate)
		if err != nil {
			return nil, err
		}
		period.StartDate = startDate
	}

	if dto.EndDate != "" {
		endDate, err := parseDateFlexible(dto.EndDate)
		if err != nil {
			return nil, err
		}
		period.EndDate = endDate
	}

	if period.EndDate.Before(period.StartDate) {
		return nil, ErrInvalidDateRange
	}

	if dto.IsActive != nil {
		period.IsActive = *dto.IsActive
	}

	if err := s.repo.Update(ctx, period); err != nil {
		return nil, err
	}

	return period, nil
}

func (s *AcademicPeriodService) DeletePeriod(ctx context.Context, tenantID, id string) error {
	period, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if period.IsArchived {
		return ErrPeriodArchived
	}
	return s.repo.Delete(ctx, tenantID, id)
}

// ArchivePeriod congela formalmente un periodo académico (ADR-029). Exige que
// el código de confirmación coincida exactamente con el código corto del
// periodo; es la única vía (junto a la expiración automática) de establecer
// is_archived, y no tiene operación inversa en esta API.
func (s *AcademicPeriodService) ArchivePeriod(ctx context.Context, tenantID, id, archivedBy, confirmationCode string) error {
	period, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if period.IsArchived {
		return ErrPeriodArchived
	}
	if confirmationCode != period.Code {
		return ErrConfirmationFailed
	}
	return s.repo.Archive(ctx, tenantID, id, archivedBy)
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

// MaintenanceValidationError is the typed fail-closed validation error for
// maintenance activation. Code is one of maintenance_confirm_invalid |
// maintenance_reason_invalid | maintenance_until_invalid, mapping to HTTP 422.
type MaintenanceValidationError struct {
	Code    string
	Message string
}

func (e *MaintenanceValidationError) Error() string { return e.Message }

// MaintenanceConfirmPhrase is the exact type-to-confirm phrase required to
// enable maintenance mode.
const MaintenanceConfirmPhrase = "MANTENIMIENTO"

func (s *MaintenanceService) EnableMaintenance(ctx context.Context, tenantID string, dto domain.EnableMaintenanceDTO) error {
	if dto.ConfirmPhrase != MaintenanceConfirmPhrase {
		return &MaintenanceValidationError{
			Code:    "maintenance_confirm_invalid",
			Message: "confirm_phrase must match MANTENIMIENTO exactly",
		}
	}
	if len(strings.TrimSpace(dto.Reason)) < 10 {
		return &MaintenanceValidationError{
			Code:    "maintenance_reason_invalid",
			Message: "reason must contain at least 10 characters",
		}
	}

	var until *time.Time
	if strings.TrimSpace(dto.Until) != "" {
		t, err := time.Parse(time.RFC3339, dto.Until)
		if err != nil {
			// Intentar formato sin zona o simple
			t2, err2 := time.Parse("2006-01-02T15:04:05", dto.Until)
			if err2 != nil {
				return &MaintenanceValidationError{
					Code:    "maintenance_until_invalid",
					Message: fmt.Sprintf("until must be RFC3339 or empty: %s", dto.Until),
				}
			}
			t = t2
		}
		if t.Before(time.Now()) {
			return &MaintenanceValidationError{
				Code:    "maintenance_until_invalid",
				Message: "until must be in the future",
			}
		}
		until = &t
	}

	return s.tenantRepo.SetMaintenance(ctx, tenantID, true, until, strings.TrimSpace(dto.Reason))
}

func (s *MaintenanceService) DisableMaintenance(ctx context.Context, tenantID string) error {
	return s.tenantRepo.SetMaintenance(ctx, tenantID, false, nil, "")
}

func (s *MaintenanceService) GetStatus(ctx context.Context, tenantID string) (*domain.MaintenanceStatus, error) {
	_, _ = s.ClearExpiredMaintenance(ctx, tenantID)
	return s.tenantRepo.GetMaintenance(ctx, tenantID)
}

// ClearExpiredMaintenance performs the lazy auto-off: when maintenance is on
// with a past until, it persists off once and reports cleared=true so callers
// can emit the MAINTENANCE_AUTO_DISABLED audit event. No sweeper needed.
func (s *MaintenanceService) ClearExpiredMaintenance(ctx context.Context, tenantID string) (bool, error) {
	status, err := s.tenantRepo.GetMaintenance(ctx, tenantID)
	if err != nil || status == nil || !status.MaintenanceMode || status.MaintenanceUntil == nil {
		return false, err
	}
	if !time.Now().After(*status.MaintenanceUntil) {
		return false, nil
	}
	if err := s.tenantRepo.SetMaintenance(ctx, tenantID, false, nil, ""); err != nil {
		return false, err
	}
	return true, nil
}
