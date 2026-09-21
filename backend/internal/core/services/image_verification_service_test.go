package services

import (
	"context"
	"testing"

	"solv-backend/internal/core/domain"
)

func TestImageVerification_ValidationErrors(t *testing.T) {
	svc := NewImageVerificationService(nil)

	tests := []struct {
		name      string
		imageRef  string
		expectErr error
	}{
		{
			name:      "Empty string",
			imageRef:  "",
			expectErr: ErrImageFormatInvalid,
		},
		{
			name:      "No tag",
			imageRef:  "python",
			expectErr: ErrImageFormatInvalid,
		},
		{
			name:      "Contains spaces",
			imageRef:  "python :3.12",
			expectErr: ErrImageFormatInvalid,
		},
		{
			name:      "Contains uppercase repo",
			imageRef:  "MyRepo/Image:1.0",
			expectErr: ErrImageFormatInvalid,
		},
		{
			name:      "Tag is latest",
			imageRef:  "python:latest",
			expectErr: ErrLatestTagForbiddenVerif,
		},
		{
			name:      "Namespaced tag is latest",
			imageRef:  "custom/my-lab:latest",
			expectErr: ErrLatestTagForbiddenVerif,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := svc.VerifyImage(context.Background(), tt.imageRef, false)
			if err == nil {
				t.Fatalf("expected error for %q, got nil result: %+v", tt.imageRef, res)
			}
			if tt.expectErr == ErrLatestTagForbiddenVerif && err != ErrLatestTagForbiddenVerif {
				t.Fatalf("expected %v, got %v", ErrLatestTagForbiddenVerif, err)
			}
		})
	}
}

func TestImageVerification_CacheHitAndForceBypass(t *testing.T) {
	svc := NewImageVerificationService(nil)
	svc.SetCustomStatfs(func(path string) (float64, float64, error) {
		return 500.0, 300.0, nil
	})

	image := "python:3.11-slim"
	res1, err := svc.VerifyImage(context.Background(), image, false)
	if err != nil {
		t.Fatalf("unexpected error on first call: %v", err)
	}
	if res1.Cached {
		t.Errorf("first call should not be cached")
	}

	// Second call should hit cache
	res2, err := svc.VerifyImage(context.Background(), image, false)
	if err != nil {
		t.Fatalf("unexpected error on second call: %v", err)
	}
	if !res2.Cached {
		t.Errorf("second call should be returned from cache")
	}

	// Force call should bypass cache
	res3, err := svc.VerifyImage(context.Background(), image, true)
	if err != nil {
		t.Fatalf("unexpected error on forced call: %v", err)
	}
	if res3.Cached {
		t.Errorf("forced call should have bypassed cache")
	}
}

func TestImageVerification_StorageEvaluation(t *testing.T) {
	svc := NewImageVerificationService(nil)

	t.Run("Optimal disk storage", func(t *testing.T) {
		res := &domain.ImageVerificationResult{}
		svc.evaluateStorageStatus(res, 500, 200.0, 500.0)
		if res.StorageStatus != domain.StorageStatusOK {
			t.Errorf("expected StorageStatusOK, got %v", res.StorageStatus)
		}
	})

	t.Run("Warning disk storage (between 10% and 25% free remaining)", func(t *testing.T) {
		res := &domain.ImageVerificationResult{}
		// 500GB total, 100GB free. Projected headroom = 0.5 + 15 = 15.5GB -> remaining 84.5GB (16.9% < 25%)
		svc.evaluateStorageStatus(res, 500, 100.0, 500.0)
		if res.StorageStatus != domain.StorageStatusWarning {
			t.Errorf("expected StorageStatusWarning, got %v", res.StorageStatus)
		}
	})

	t.Run("Critical blocked disk storage (<10% or <5GB free)", func(t *testing.T) {
		res := &domain.ImageVerificationResult{}
		// 500GB total, 20GB free. Projected headroom = 1GB + 15GB = 16GB -> remaining 4GB (< 5GB)
		svc.evaluateStorageStatus(res, 1024, 20.0, 500.0)
		if res.StorageStatus != domain.StorageStatusCriticalBlocked {
			t.Errorf("expected StorageStatusCriticalBlocked, got %v", res.StorageStatus)
		}
	})
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{500, "500 B"},
		{1024, "1.0 KB"},
		{1024 * 1024 * 150, "150.0 MB"},
		{1024 * 1024 * 1024 * 2, "2.0 GB"},
	}

	for _, tt := range tests {
		got := formatBytes(tt.bytes)
		if got != tt.expected {
			t.Errorf("formatBytes(%d) = %q, expected %q", tt.bytes, got, tt.expected)
		}
	}
}

func TestClassifyImageOrigin(t *testing.T) {
	tests := []struct {
		imageRef       string
		wantOfficial   bool
		wantOriginType string
	}{
		{"python:3.12-slim", true, "official"},
		{"library/node:20-alpine", true, "official"},
		{"postgres:16-alpine", true, "official"},
		{"ghcr.io/org/repo:1.0", false, "verified_registry"},
		{"mcr.microsoft.com/dotnet/sdk:8.0", false, "verified_registry"},
		{"pepito123/my-lab:v1", false, "community"},
	}

	for _, tt := range tests {
		official, originType, warning := classifyImageOrigin(tt.imageRef)
		if official != tt.wantOfficial {
			t.Errorf("classifyImageOrigin(%q) official = %v, want %v", tt.imageRef, official, tt.wantOfficial)
		}
		if originType != tt.wantOriginType {
			t.Errorf("classifyImageOrigin(%q) originType = %q, want %q", tt.imageRef, originType, tt.wantOriginType)
		}
		if !official && warning == "" {
			t.Errorf("classifyImageOrigin(%q) expected warning for unofficial image", tt.imageRef)
		}
	}
}

