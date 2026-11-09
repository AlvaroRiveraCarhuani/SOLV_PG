package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"solv-backend/internal/core/domain"
)

type auditCaptureRepo struct {
	mu   sync.Mutex
	logs []*domain.AuditLog
}

func (r *auditCaptureRepo) Create(_ context.Context, logEntry *domain.AuditLog) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.logs = append(r.logs, logEntry)
	return nil
}

func (r *auditCaptureRepo) ListByTenant(context.Context, string, int) ([]*domain.AuditLog, error) {
	return nil, nil
}

func (r *auditCaptureRepo) ListFiltered(context.Context, string, string, string, int, int) ([]*domain.AuditLog, error) {
	return nil, nil
}

func (r *auditCaptureRepo) CountFiltered(context.Context, string, string, string) (int, error) {
	return 0, nil
}

func (r *auditCaptureRepo) ListByActorTimeline(context.Context, string, string, int) ([]*domain.AuditLog, error) {
	return nil, nil
}

func auditRequestContext(req *http.Request, role string) *http.Request {
	ctx := context.WithValue(req.Context(), domain.UserRoleKey, role)
	ctx = context.WithValue(ctx, domain.UserIDKey, "admin-1")
	ctx = context.WithValue(ctx, domain.TenantIDKey, "tenant-1")
	return req.WithContext(ctx)
}

func TestAuditMiddlewareRedactsNestedSecrets(t *testing.T) {
	repo := &auditCaptureRepo{}
	pool := NewAuditWorkerPool(repo, 8, 1)

	handler := AuditMiddleware(pool)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	body := `{
		"api_key": "sk-live-secret",
		"profile": {
			"APIKey": "sk-nested-secret",
			"encrypted_api_key": "cipher-secret",
			"access_token": "access-secret",
			"name": "perfil-publico"
		},
		"items": [
			{"password": "password-secret", "secret": "nested-secret", "value": 7}
		]
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/ai/settings", strings.NewReader(body))
	req = auditRequestContext(req, "admin")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	pool.Shutdown()

	repo.mu.Lock()
	defer repo.mu.Unlock()
	if len(repo.logs) != 1 {
		t.Fatalf("esperado un registro de auditoría, obtenido %d", len(repo.logs))
	}
	metadata := string(repo.logs[0].Metadata)
	for _, secret := range []string{"sk-live-secret", "sk-nested-secret", "cipher-secret", "access-secret", "password-secret", "nested-secret"} {
		if strings.Contains(metadata, secret) {
			t.Fatalf("el secreto %q quedó persistido en metadata: %s", secret, metadata)
		}
	}
	if !strings.Contains(metadata, "perfil-publico") || !strings.Contains(metadata, `"value":7`) {
		t.Fatalf("la metadata no sensible debía conservarse: %s", metadata)
	}
}

func TestAuditMiddlewareDiscardsNonJSONBody(t *testing.T) {
	repo := &auditCaptureRepo{}
	pool := NewAuditWorkerPool(repo, 8, 1)
	handler := AuditMiddleware(pool)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/ai/settings", strings.NewReader("api_key=sk-form-secret"))
	req = auditRequestContext(req, "admin")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	pool.Shutdown()

	repo.mu.Lock()
	defer repo.mu.Unlock()
	if len(repo.logs) != 1 {
		t.Fatalf("esperado un registro de auditoría, obtenido %d", len(repo.logs))
	}
	if strings.Contains(string(repo.logs[0].Metadata), "sk-form-secret") {
		t.Fatalf("un cuerpo no JSON no debe persistirse: %s", repo.logs[0].Metadata)
	}
}
