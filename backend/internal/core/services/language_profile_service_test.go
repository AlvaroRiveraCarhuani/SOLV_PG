package services_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

type mockLanguageProfileRepo struct {
	profiles map[string]*domain.LanguageProfile
	audits   []*domain.LanguageProfileAudit
}

func newMockLanguageProfileRepo() *mockLanguageProfileRepo {
	return &mockLanguageProfileRepo{
		profiles: map[string]*domain.LanguageProfile{
			"python": {
				Language:            "python",
				DefaultTimeoutMS:    2000,
				DefaultMemoryMB:     256,
				Image:               "python:3.11-slim@sha256:db3ff2e1800a8581e2c48a27c3995339d47bdf046da21c7627accd3d51053a93",
				BuildCommand:        "",
				BuildTimeoutMS:      10000,
				BuildMemoryMB:       512,
				P95WindowDays:       30,
				CheckerSidecarImage: "python:3.11-slim@sha256:db3ff2e1800a8581e2c48a27c3995339d47bdf046da21c7627accd3d51053a93",
				CreatedAt:           time.Now(),
				UpdatedAt:           time.Now(),
			},
			"cpp": {
				Language:            "cpp",
				DefaultTimeoutMS:    1000,
				DefaultMemoryMB:     128,
				Image:               "gcc@sha256:cde79a7114216f9a1a66509932adabdd1e8620d8c8d11be19a34ee4b22d66c91",
				BuildCommand:        "g++ -O2 /runner/solution.cpp -o /tmp/sol",
				BuildTimeoutMS:      10000,
				BuildMemoryMB:       512,
				P95WindowDays:       30,
				CheckerSidecarImage: "python:3.11-slim@sha256:db3ff2e1800a8581e2c48a27c3995339d47bdf046da21c7627accd3d51053a93",
				CreatedAt:           time.Now(),
				UpdatedAt:           time.Now(),
			},
		},
		audits: make([]*domain.LanguageProfileAudit, 0),
	}
}

func (m *mockLanguageProfileRepo) ListProfiles(ctx context.Context) ([]*domain.LanguageProfile, error) {
	list := make([]*domain.LanguageProfile, 0, len(m.profiles))
	for _, p := range m.profiles {
		list = append(list, p)
	}
	return list, nil
}

func (m *mockLanguageProfileRepo) GetProfileByLanguage(ctx context.Context, language string) (*domain.LanguageProfile, error) {
	p, ok := m.profiles[language]
	if !ok {
		return nil, domain.ErrUnknownLanguage
	}
	copied := *p
	return &copied, nil
}

func (m *mockLanguageProfileRepo) UpdateProfile(ctx context.Context, profile *domain.LanguageProfile, audit *domain.LanguageProfileAudit) error {
	if _, ok := m.profiles[profile.Language]; !ok {
		return domain.ErrUnknownLanguage
	}
	m.profiles[profile.Language] = profile
	if audit != nil {
		m.audits = append(m.audits, audit)
	}
	return nil
}

func TestLanguageProfileService_CanonicalValidation(t *testing.T) {
	repo := newMockLanguageProfileRepo()
	svc := services.NewLanguageProfileService(repo)
	ctx := context.Background()

	// 1. Claves canónicas permitidas
	canonicalList := []string{"python", "javascript", "cpp", "c", "csharp", "java"}
	for _, lang := range canonicalList {
		if !services.IsCanonicalJudgeLanguage(lang) {
			t.Errorf("expected %s to be canonical", lang)
		}
	}

	// 2. Alias o no soportados deben rechazarse en GetProfile
	invalidAliases := []string{"c++", "c#", "cs", "golang", "go", "sql", "rust", "ruby"}
	for _, alias := range invalidAliases {
		_, err := svc.GetProfile(ctx, alias)
		if !errors.Is(err, domain.ErrUnknownLanguage) {
			t.Errorf("expected ErrUnknownLanguage for alias/unsupported %s, got: %v", alias, err)
		}
	}
}

func TestLanguageProfileService_UpdateProfile_AuditingAndBounds(t *testing.T) {
	repo := newMockLanguageProfileRepo()
	svc := services.NewLanguageProfileService(repo)
	ctx := context.Background()

	// 1. Rechaza actualización sin motivo de auditoría
	newTimeout := 1500
	_, err := svc.UpdateProfile(ctx, "admin-1", "python", domain.UpdateLanguageProfileDTO{
		DefaultTimeoutMS: &newTimeout,
		Reason:           "",
	})
	if !errors.Is(err, domain.ErrReasonRequired) {
		t.Fatalf("expected ErrReasonRequired, got: %v", err)
	}

	// 2. Rechaza timeout fuera de rango (100..10000)
	invalidTimeout := 50
	_, err = svc.UpdateProfile(ctx, "admin-1", "python", domain.UpdateLanguageProfileDTO{
		DefaultTimeoutMS: &invalidTimeout,
		Reason:           "ajuste de timeout",
	})
	if !errors.Is(err, domain.ErrLanguageProfileTimeoutRange) {
		t.Fatalf("expected ErrLanguageProfileTimeoutRange, got: %v", err)
	}

	// 3. Rechaza memoria fuera de rango (64..1024)
	invalidMem := 2048
	_, err = svc.UpdateProfile(ctx, "admin-1", "python", domain.UpdateLanguageProfileDTO{
		DefaultMemoryMB: &invalidMem,
		Reason:          "ajuste de memoria",
	})
	if !errors.Is(err, domain.ErrLanguageProfileMemoryRange) {
		t.Fatalf("expected ErrLanguageProfileMemoryRange, got: %v", err)
	}

	// 4. Rechaza imagen sin digest sha256 o con :latest
	unpinnedImg := "python:latest"
	_, err = svc.UpdateProfile(ctx, "admin-1", "python", domain.UpdateLanguageProfileDTO{
		Image:  &unpinnedImg,
		Reason: "cambio de imagen",
	})
	if !errors.Is(err, domain.ErrLanguageProfileImageUnpinned) {
		t.Fatalf("expected ErrLanguageProfileImageUnpinned, got: %v", err)
	}

	// 5. Actualización exitosa con auditoría registrada
	validMem := 384
	validTimeoutVal := 2500
	validImg := "python:3.11-slim@sha256:1111111111111111111111111111111111111111111111111111111111111111"
	updated, err := svc.UpdateProfile(ctx, "admin-1", "python", domain.UpdateLanguageProfileDTO{
		DefaultTimeoutMS: &validTimeoutVal,
		DefaultMemoryMB:  &validMem,
		Image:            &validImg,
		Reason:           "Calibración p95 Q4",
	})
	if err != nil {
		t.Fatalf("unexpected error updating profile: %v", err)
	}
	if updated.DefaultMemoryMB != 384 || updated.DefaultTimeoutMS != 2500 {
		t.Errorf("updated values mismatch: memory=%d, timeout=%d", updated.DefaultMemoryMB, updated.DefaultTimeoutMS)
	}

	// Verificar asiento de auditoría
	if len(repo.audits) != 1 {
		t.Fatalf("expected 1 audit entry, got %d", len(repo.audits))
	}
	audit := repo.audits[0]
	if audit.Language != "python" || audit.Author != "admin-1" || audit.Reason != "Calibración p95 Q4" {
		t.Errorf("audit entry content mismatch: %+v", audit)
	}
}
