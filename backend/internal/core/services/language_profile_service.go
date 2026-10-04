package services

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/shirou/gopsutil/v3/mem"
	"solv-backend/internal/core/domain"
)

var (
	ErrNonCanonicalLanguage = errors.New("la clave de lenguaje debe ser canónica (ej: cpp en lugar de c++, csharp en lugar de c#)")
)

type LanguageProfileService struct {
	repo domain.LanguageProfileRepository
}

func NewLanguageProfileService(repo domain.LanguageProfileRepository) *LanguageProfileService {
	return &LanguageProfileService{
		repo: repo,
	}
}

// IsCanonicalJudgeLanguage verifica si el lenguaje pertenece a las 6 claves canónicas del juez (D-EJ-04).
func IsCanonicalJudgeLanguage(lang string) bool {
	return domain.SupportedCanonicalLanguages[strings.TrimSpace(lang)]
}

func (s *LanguageProfileService) getHostTotalRAM(ctx context.Context) int {
	totalMB := 8192
	if v, err := mem.VirtualMemoryWithContext(ctx); err == nil && v != nil && v.Total > 0 {
		totalMB = int(v.Total / (1024 * 1024))
	}
	return totalMB
}

// ListProfiles devuelve la lista de todos los perfiles de lenguaje configurados.
func (s *LanguageProfileService) ListProfiles(ctx context.Context) ([]*domain.LanguageProfile, error) {
	return s.repo.ListProfiles(ctx)
}

// GetProfile devuelve el perfil para una clave canónica. Rechaza alias no canónicos con error.
func (s *LanguageProfileService) GetProfile(ctx context.Context, language string) (*domain.LanguageProfile, error) {
	cleaned := strings.TrimSpace(language)
	if !IsCanonicalJudgeLanguage(cleaned) {
		return nil, domain.ErrUnknownLanguage
	}
	return s.repo.GetProfileByLanguage(ctx, cleaned)
}

// UpdateProfile actualiza un perfil existente con validación estructural y registro de auditoría.
func (s *LanguageProfileService) UpdateProfile(
	ctx context.Context,
	adminID string,
	language string,
	dto domain.UpdateLanguageProfileDTO,
) (*domain.LanguageProfile, error) {
	cleaned := strings.TrimSpace(language)
	if !IsCanonicalJudgeLanguage(cleaned) {
		return nil, domain.ErrUnknownLanguage
	}

	if strings.TrimSpace(dto.Reason) == "" {
		return nil, domain.ErrReasonRequired
	}

	current, err := s.repo.GetProfileByLanguage(ctx, cleaned)
	if err != nil {
		return nil, err
	}

	oldJSON, _ := json.Marshal(current)
	updated := *current

	if dto.DefaultTimeoutMS != nil {
		if *dto.DefaultTimeoutMS < 100 || *dto.DefaultTimeoutMS > 10000 {
			return nil, domain.ErrLanguageProfileTimeoutRange
		}
		updated.DefaultTimeoutMS = *dto.DefaultTimeoutMS
	}

	if dto.DefaultMemoryMB != nil {
		if *dto.DefaultMemoryMB < 64 || *dto.DefaultMemoryMB > 1024 {
			return nil, domain.ErrLanguageProfileMemoryRange
		}
		updated.DefaultMemoryMB = *dto.DefaultMemoryMB
	}

	// 1. Frontera ValidateRamAgainstHost para perfil de lenguaje
	totalHostMB := s.getHostTotalRAM(ctx)
	maxAllowedRAM := domain.CalculateHostMaxAllowedRAM(totalHostMB)
	if err := domain.ValidateRamAgainstHost(updated.DefaultMemoryMB, maxAllowedRAM); err != nil {
		return nil, err
	}

	if dto.BuildMemoryMB != nil {
		if *dto.BuildMemoryMB < 64 || *dto.BuildMemoryMB > 2048 {
			return nil, domain.ErrLanguageProfileMemoryRange
		}
		if err := domain.ValidateRamAgainstHost(*dto.BuildMemoryMB, maxAllowedRAM); err != nil {
			return nil, err
		}
		updated.BuildMemoryMB = *dto.BuildMemoryMB
	}

	if dto.BuildTimeoutMS != nil {
		if *dto.BuildTimeoutMS < 100 || *dto.BuildTimeoutMS > 60000 {
			return nil, domain.ErrLanguageProfileTimeoutRange
		}
		updated.BuildTimeoutMS = *dto.BuildTimeoutMS
	}

	if dto.Image != nil {
		img := strings.TrimSpace(*dto.Image)
		if !strings.Contains(img, "@sha256:") || strings.HasSuffix(strings.ToLower(img), ":latest") {
			return nil, domain.ErrLanguageProfileImageUnpinned
		}
		updated.Image = img
	}

	if dto.CheckerSidecarImage != nil {
		img := strings.TrimSpace(*dto.CheckerSidecarImage)
		if !strings.Contains(img, "@sha256:") || strings.HasSuffix(strings.ToLower(img), ":latest") {
			return nil, domain.ErrLanguageProfileImageUnpinned
		}
		updated.CheckerSidecarImage = img
	}

	if dto.BuildCommand != nil {
		updated.BuildCommand = strings.TrimSpace(*dto.BuildCommand)
	}

	if dto.P95WindowDays != nil {
		if *dto.P95WindowDays < 1 || *dto.P95WindowDays > 365 {
			return nil, errors.New("la ventana p95_window_days debe estar entre 1 y 365 días")
		}
		updated.P95WindowDays = *dto.P95WindowDays
	}

	newJSON, _ := json.Marshal(updated)

	audit := &domain.LanguageProfileAudit{
		Language:  cleaned,
		Author:    adminID,
		OldValues: oldJSON,
		NewValues: newJSON,
		Reason:    strings.TrimSpace(dto.Reason),
	}

	if err := s.repo.UpdateProfile(ctx, &updated, audit); err != nil {
		return nil, err
	}

	return &updated, nil
}
