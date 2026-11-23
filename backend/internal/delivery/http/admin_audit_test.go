package httpdelivery

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"solv-backend/internal/core/domain"
)

type mockAuditRepo struct {
	logs        []*domain.AuditLog
	total       int
	err         error
	passedLimit int
	passedOffset int
	passedSearch string
	passedAction string
}

func (m *mockAuditRepo) Create(ctx context.Context, log *domain.AuditLog) error {
	return m.err
}

func (m *mockAuditRepo) ListByTenant(ctx context.Context, tenantID string, limit int) ([]*domain.AuditLog, error) {
	return m.logs, m.err
}

func (m *mockAuditRepo) ListFiltered(ctx context.Context, tenantID, search, action string, limit, offset int) ([]*domain.AuditLog, error) {
	m.passedSearch = search
	m.passedAction = action
	m.passedLimit = limit
	m.passedOffset = offset
	return m.logs, m.err
}

func (m *mockAuditRepo) CountFiltered(ctx context.Context, tenantID, search, action string) (int, error) {
	return m.total, m.err
}

func (m *mockAuditRepo) ListByActorTimeline(ctx context.Context, tenantID, actorID string, limit int) ([]*domain.AuditLog, error) {
	return m.logs, m.err
}

func TestListAuditLogsReturnsEmptySliceInsteadOfNull(t *testing.T) {
	repo := &mockAuditRepo{
		logs:  nil, // repo returns nil slice
		total: 0,
	}

	handler := NewAdminHandler(repo, nil, nil, nil, nil)
	req := httptest.NewRequest("GET", "/api/v1/admin/audit-logs", nil)
	ctx := context.WithValue(req.Context(), domain.TenantIDKey, "tenant-test-123")
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.ListAuditLogs(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, `"data":[]`) {
		t.Errorf("expected response to contain `\"data\":[]`, got: %s", body)
	}
	if strings.Contains(body, `"data":null`) {
		t.Errorf("response must NEVER contain `\"data\":null`, got: %s", body)
	}
}

func TestGetActorTimelineReturnsEmptySliceInsteadOfNull(t *testing.T) {
	repo := &mockAuditRepo{
		logs: nil, // repo returns nil slice
	}

	handler := NewAdminHandler(repo, nil, nil, nil, nil)
	req := httptest.NewRequest("GET", "/api/v1/admin/audit-logs/actors/actor-1/timeline", nil)
	req.SetPathValue("actorId", "actor-1")
	ctx := context.WithValue(req.Context(), domain.TenantIDKey, "tenant-test-123")
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.GetActorTimeline(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, `"data":[]`) {
		t.Errorf("expected response to contain `\"data\":[]`, got: %s", body)
	}
	if strings.Contains(body, `"data":null`) {
		t.Errorf("response must NEVER contain `\"data\":null`, got: %s", body)
	}
}
