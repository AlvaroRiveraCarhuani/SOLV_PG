package httpdelivery_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
	httpdelivery "solv-backend/internal/delivery/http"
)

// teacherReadRepo responde con el resultado de la lectura de la entrega que
// comparten el timeline, los keystrokes y el runner efímero.
type teacherReadRepo struct {
	domain.TeacherRepository
	reviewErr error
}

func (r *teacherReadRepo) GetTeacherSubmissionReview(_ context.Context, _, _, _ string) (*domain.TeacherSubmissionReviewDTO, error) {
	return nil, r.reviewErr
}

func teacherReadRequest(method, path, submissionID string) *http.Request {
	req := httptest.NewRequest(method, path, strings.NewReader(`{}`))
	req.SetPathValue("id", submissionID)
	ctx := context.WithValue(req.Context(), domain.TenantIDKey, domain.DefaultTenantID)
	req = req.WithContext(ctx)
	req.Header.Set("X-User-Role", "teacher")
	req.Header.Set("X-User-Id", "teacher-read")
	req.Header.Set("Content-Type", "application/json")
	return req
}

func newTeacherReadHandler(repo *teacherReadRepo) *httpdelivery.TeacherHandler {
	return httpdelivery.NewTeacherHandler(services.NewTeacherService(repo, nil), nil)
}

const readSubmissionID = "0a4f0b52-3d1c-4f8e-9a4a-3c1d2b5e6f70"

// Una entrega ausente responde 404 en los tres lectores del docente.
func TestTeacherSubmissionReaders_MissingReturnsNotFound(t *testing.T) {
	repo := &teacherReadRepo{reviewErr: fmt.Errorf("%w: no existe", domain.ErrNotFound)}
	handler := newTeacherReadHandler(repo)

	cases := []struct {
		name   string
		method string
		path   string
		call   func(http.ResponseWriter, *http.Request)
	}{
		{"timeline", http.MethodGet, "/api/v1/teacher/submissions/" + readSubmissionID + "/timeline", handler.GetTimeline},
		{"keystrokes", http.MethodGet, "/api/v1/teacher/submissions/" + readSubmissionID + "/keystroke-events", handler.GetKeystrokeEvents},
		{"runner", http.MethodPost, "/api/v1/teacher/submissions/" + readSubmissionID + "/run-ephemeral", handler.RunEphemeral},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			tc.call(rec, teacherReadRequest(tc.method, tc.path, readSubmissionID))
			if rec.Code != http.StatusNotFound {
				t.Fatalf("esperado 404, obtenido %d: %s", rec.Code, rec.Body.String())
			}
		})
	}
}

// Un id con formato inválido es validación: 400, no 500.
func TestTeacherSubmissionReaders_MalformedIDReturnsBadRequest(t *testing.T) {
	repo := &teacherReadRepo{reviewErr: fmt.Errorf("%w: no-es-uuid", domain.ErrInvalidResourceID)}
	handler := newTeacherReadHandler(repo)

	cases := []struct {
		name   string
		method string
		path   string
		call   func(http.ResponseWriter, *http.Request)
	}{
		{"timeline", http.MethodGet, "/api/v1/teacher/submissions/no-es-uuid/timeline", handler.GetTimeline},
		{"keystrokes", http.MethodGet, "/api/v1/teacher/submissions/no-es-uuid/keystroke-events", handler.GetKeystrokeEvents},
		{"runner", http.MethodPost, "/api/v1/teacher/submissions/no-es-uuid/run-ephemeral", handler.RunEphemeral},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			tc.call(rec, teacherReadRequest(tc.method, tc.path, "no-es-uuid"))
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("esperado 400, obtenido %d: %s", rec.Code, rec.Body.String())
			}
		})
	}
}
