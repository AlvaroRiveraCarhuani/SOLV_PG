package httpdelivery

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

type keystrokeHandlerSubmissionRepo struct {
	domain.SubmissionRepository
	submission *domain.Submission
}

func (r *keystrokeHandlerSubmissionRepo) GetByID(context.Context, string, string) (*domain.Submission, error) {
	return r.submission, nil
}

type keystrokeHandlerStore struct {
	domain.KeystrokeRepository
	saved int
}

func (s *keystrokeHandlerStore) SaveBatch(_ context.Context, _ string, events []domain.SubmissionKeystrokeEvent) error {
	s.saved = len(events)
	return nil
}

func newKeystrokeHandlerFixture() (*SubmissionHandler, *keystrokeHandlerStore) {
	repo := &keystrokeHandlerSubmissionRepo{submission: &domain.Submission{
		ID:        "submission-1",
		TenantID:  "tenant-1",
		StudentID: "student-owner",
	}}
	store := &keystrokeHandlerStore{}
	service := services.NewSubmissionService(repo)
	service.SetKeystrokeRepository(store)
	return NewSubmissionHandler(service), store
}

func keystrokeRequest(role, userID string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/submissions/submission-1/keystroke-events",
		strings.NewReader(`{"events":[{}]}`))
	req.SetPathValue("id", "submission-1")
	ctx := context.WithValue(req.Context(), domain.TenantIDKey, "tenant-1")
	ctx = context.WithValue(ctx, domain.UserIDKey, userID)
	ctx = context.WithValue(ctx, domain.UserRoleKey, role)
	return req.WithContext(ctx)
}

func TestSaveKeystrokeEventsRejectsOtherStudents(t *testing.T) {
	handler, store := newKeystrokeHandlerFixture()

	rec := httptest.NewRecorder()
	handler.SaveKeystrokeEvents(rec, keystrokeRequest("student", "student-intruder"))

	if rec.Code != http.StatusForbidden {
		t.Fatalf("se esperaba 403, se obtuvo %d: %s", rec.Code, rec.Body.String())
	}
	if store.saved != 0 {
		t.Fatalf("no debía guardarse nada, se guardaron %d eventos", store.saved)
	}
}

func TestSaveKeystrokeEventsAllowsOwner(t *testing.T) {
	handler, store := newKeystrokeHandlerFixture()

	rec := httptest.NewRecorder()
	handler.SaveKeystrokeEvents(rec, keystrokeRequest("student", "student-owner"))

	if rec.Code != http.StatusOK {
		t.Fatalf("el dueño debe poder guardar: %d %s", rec.Code, rec.Body.String())
	}
	if store.saved != 1 {
		t.Fatalf("debía guardarse un evento, se guardaron %d", store.saved)
	}
}
