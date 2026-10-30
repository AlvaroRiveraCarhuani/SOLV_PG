package domain_test

import (
	"errors"
	"testing"

	"solv-backend/internal/core/domain"
)

func TestParseReviewStatus(t *testing.T) {
	cases := []struct {
		raw      string
		expected domain.ReviewStatus
		wantErr  bool
	}{
		{"pending", domain.ReviewPending, false},
		{"  IN_REVIEW ", domain.ReviewInReview, false},
		{"graded", domain.ReviewGraded, false},
		{"returned", domain.ReviewReturned, false},
		{"not_required", domain.ReviewNotRequired, false},
		{"aprobado", "", true},
		{"", "", true},
	}

	for _, tc := range cases {
		got, err := domain.ParseReviewStatus(tc.raw)
		if tc.wantErr {
			if !errors.Is(err, domain.ErrInvalidReviewStatus) {
				t.Errorf("ParseReviewStatus(%q): expected ErrInvalidReviewStatus, got %v", tc.raw, err)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseReviewStatus(%q): unexpected error %v", tc.raw, err)
			continue
		}
		if got != tc.expected {
			t.Errorf("ParseReviewStatus(%q) = %q, want %q", tc.raw, got, tc.expected)
		}
	}
}

func TestValidateReviewTransition(t *testing.T) {
	cases := []struct {
		from, to domain.ReviewStatus
		allowed  bool
	}{
		{domain.ReviewPending, domain.ReviewInReview, true},
		{domain.ReviewPending, domain.ReviewGraded, true},
		{domain.ReviewInReview, domain.ReviewPending, true},
		{domain.ReviewInReview, domain.ReviewGraded, true},
		{domain.ReviewGraded, domain.ReviewReturned, true},
		{domain.ReviewGraded, domain.ReviewInReview, true},
		{domain.ReviewReturned, domain.ReviewGraded, true},
		// Idempotencia: reintentar el mismo estado siempre se acepta.
		{domain.ReviewGraded, domain.ReviewGraded, true},
		{domain.ReviewPending, domain.ReviewPending, true},
		// Saltos inválidos.
		{domain.ReviewPending, domain.ReviewReturned, false},
		{domain.ReviewNotRequired, domain.ReviewGraded, false},
		{domain.ReviewReturned, domain.ReviewPending, false},
	}

	for _, tc := range cases {
		err := domain.ValidateReviewTransition(tc.from, tc.to)
		if tc.allowed && err != nil {
			t.Errorf("%s -> %s: expected allowed, got %v", tc.from, tc.to, err)
		}
		if !tc.allowed && !errors.Is(err, domain.ErrInvalidReviewTransition) {
			t.Errorf("%s -> %s: expected ErrInvalidReviewTransition, got %v", tc.from, tc.to, err)
		}
	}
}

func TestResolvedSubmissionScore(t *testing.T) {
	thirty := 30
	if got := domain.ResolvedSubmissionScore(&thirty, "WA", false); got == nil || *got != 30 {
		t.Errorf("explicit score must win, got %v", got)
	}
	if got := domain.ResolvedSubmissionScore(nil, "AC", false); got == nil || *got != 100 {
		t.Errorf("AC without score must resolve to 100, got %v", got)
	}
	if got := domain.ResolvedSubmissionScore(nil, "WA", true); got == nil || *got != 100 {
		t.Errorf("manual override without score must resolve to 100, got %v", got)
	}
	if got := domain.ResolvedSubmissionScore(nil, "WA", false); got != nil {
		t.Errorf("ungraded submission must resolve to nil, got %v", *got)
	}
}

func TestNeedsGradeReason(t *testing.T) {
	base := 90
	other := 70
	if domain.NeedsGradeReason(&base, nil) {
		t.Error("choosing without changing the score must not require a reason")
	}
	if domain.NeedsGradeReason(&base, &base) {
		t.Error("same score must not require a reason")
	}
	if !domain.NeedsGradeReason(&base, &other) {
		t.Error("changing the score must require a reason")
	}
	if !domain.NeedsGradeReason(nil, &other) {
		t.Error("grading an ungraded cell must require a reason")
	}
}
