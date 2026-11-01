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

// overrideRepo responde solo al UPDATE del override. overrideErr permite
// simular la entrega ausente tal como la informa el repositorio.
type overrideRepo struct {
	domain.SubmissionRepository
	overrideErr error
	calls       int
}

func (r *overrideRepo) UpdateOverride(_ context.Context, _, _, _, _ string, _ *int, _ *string) error {
	r.calls++
	return r.overrideErr
}

func overrideRequest(submissionID string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/submissions/"+submissionID+"/override",
		strings.NewReader(`{"verdict":"AC","override_reason":"justificacion suficientemente larga","score":90}`))
	req.SetPathValue("id", submissionID)
	ctx := context.WithValue(req.Context(), domain.TenantIDKey, domain.DefaultTenantID)
	req = req.WithContext(ctx)
	req.Header.Set("X-User-Role", "teacher")
	req.Header.Set("X-User-Id", "teacher-grades")
	req.Header.Set("Content-Type", "application/json")
	return req
}

// Una entrega que el repositorio no encuentra no se responde como guardada:
// el 200 queda reservado para cuando la fila se actualizó.
func TestOverrideSubmission_MissingSubmissionReturnsNotFound(t *testing.T) {
	repo := &overrideRepo{overrideErr: fmt.Errorf("%w: no existe", domain.ErrNotFound)}
	handler := httpdelivery.NewSubmissionHandler(services.NewSubmissionService(repo))

	rec := httptest.NewRecorder()
	handler.OverrideSubmission(rec, overrideRequest("0a4f0b52-3d1c-4f8e-9a4a-3c1d2b5e6f70"))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("esperado 404, obtenido %d: %s", rec.Code, rec.Body.String())
	}
	if repo.calls != 1 {
		t.Fatalf("el repositorio debía intentar el override una vez, se llamó %d", repo.calls)
	}
}

// Un identificador con formato inválido es validación: 400, no 500.
func TestOverrideSubmission_MalformedIDReturnsBadRequest(t *testing.T) {
	repo := &overrideRepo{overrideErr: fmt.Errorf("%w: no-es-uuid", domain.ErrInvalidResourceID)}
	handler := httpdelivery.NewSubmissionHandler(services.NewSubmissionService(repo))

	rec := httptest.NewRecorder()
	handler.OverrideSubmission(rec, overrideRequest("no-es-uuid"))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("esperado 400, obtenido %d: %s", rec.Code, rec.Body.String())
	}
}

// El caso feliz sigue respondiendo 200 con el veredicto aplicado.
func TestOverrideSubmission_UpdatedSubmissionReturnsOK(t *testing.T) {
	repo := &overrideRepo{}
	handler := httpdelivery.NewSubmissionHandler(services.NewSubmissionService(repo))

	rec := httptest.NewRecorder()
	handler.OverrideSubmission(rec, overrideRequest("0a4f0b52-3d1c-4f8e-9a4a-3c1d2b5e6f70"))

	if rec.Code != http.StatusOK {
		t.Fatalf("esperado 200, obtenido %d: %s", rec.Code, rec.Body.String())
	}
}
