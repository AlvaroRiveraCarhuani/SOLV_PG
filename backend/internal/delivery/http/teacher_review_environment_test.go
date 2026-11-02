package httpdelivery

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"solv-backend/internal/core/domain"
)

func TestStartReviewEnvironmentRejectsStudents(t *testing.T) {
	handler := &TeacherHandler{}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/teacher/submissions/abc/test-environment", nil)
	req.Header.Set("X-User-Role", "student")
	rec := httptest.NewRecorder()

	handler.StartReviewEnvironment(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestStartReviewEnvironmentWithoutServiceReportsUnavailable(t *testing.T) {
	handler := &TeacherHandler{}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/teacher/submissions/abc/test-environment", nil)
	req.SetPathValue("id", "abc")
	req.Header.Set("X-User-Role", "teacher")
	req.Header.Set("X-User-Id", "teacher-1")
	rec := httptest.NewRecorder()

	handler.StartReviewEnvironment(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
	if code := decodeErrorCode(t, rec); code != "review_environment_unavailable" {
		t.Fatalf("expected review_environment_unavailable, got %q", code)
	}
}

func TestWriteReviewEnvironmentErrorMapping(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{"invalid id", fmt.Errorf("%w: abc", domain.ErrInvalidResourceID), http.StatusBadRequest, "invalid_submission_id"},
		{"not found", domain.ErrNotFound, http.StatusNotFound, "submission_not_found"},
		{"judge modality", domain.ErrReviewEnvironmentRequiresIDE, http.StatusConflict, "review_environment_requires_ide"},
		{"missing command", domain.ErrReviewEnvironmentCommandMissing, http.StatusConflict, "app_run_command_missing"},
		{"missing snapshot", domain.ErrReviewEnvironmentSnapshotUnavailable, http.StatusConflict, "project_snapshot_unavailable"},
		{"missing template", domain.ErrReviewEnvironmentTemplateMissing, http.StatusConflict, "template_image_missing"},
		{"invalid port", domain.ErrInvalidAppPort, http.StatusConflict, "invalid_app_port"},
		{"already starting", domain.ErrReviewEnvironmentStarting, http.StatusConflict, "review_environment_starting"},
		{"oom guard", domain.ErrReviewEnvironmentHostMemory, http.StatusServiceUnavailable, "oom_guard"},
		{"app not ready", domain.ErrReviewEnvironmentNotReady, http.StatusBadGateway, "app_not_ready"},
		{"container failed", domain.ErrReviewEnvironmentStartFailed, http.StatusBadGateway, "app_start_failed"},
		{"unavailable", domain.ErrReviewEnvironmentUnavailable, http.StatusServiceUnavailable, "review_environment_unavailable"},
		{"unexpected", errors.New("boom"), http.StatusInternalServerError, "review_environment_failed"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			writeReviewEnvironmentError(rec, tc.err)

			if rec.Code != tc.wantStatus {
				t.Fatalf("expected %d, got %d", tc.wantStatus, rec.Code)
			}
			if code := decodeErrorCode(t, rec); code != tc.wantCode {
				t.Fatalf("expected %q, got %q", tc.wantCode, code)
			}
		})
	}
}

func decodeErrorCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON error body: %v", err)
	}
	return body.Error
}
