package httpdelivery_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
	httpdelivery "solv-backend/internal/delivery/http"
)

// readRepo responde con el resultado de la lectura por id y permite simular
// una entrega ausente, un id mal formado o una falla del almacén.
type readRepo struct {
	domain.SubmissionRepository
	getErr error
}

func (r *readRepo) GetByID(_ context.Context, _, _ string) (*domain.Submission, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	return &domain.Submission{ID: "0a4f0b52-3d1c-4f8e-9a4a-3c1d2b5e6f70", StudentID: "student-1"}, nil
}

func readSubmissionRequest(submissionID string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/submissions/"+submissionID, nil)
	req.SetPathValue("id", submissionID)
	ctx := context.WithValue(req.Context(), domain.TenantIDKey, domain.DefaultTenantID)
	req = req.WithContext(ctx)
	req.Header.Set("X-User-Role", "teacher")
	req.Header.Set("X-User-Id", "teacher-grades")
	return req
}

// Una entrega inexistente es un 404, igual que en el override.
func TestGetSubmissionByID_MissingReturnsNotFound(t *testing.T) {
	repo := &readRepo{getErr: fmt.Errorf("%w: no existe", domain.ErrNotFound)}
	handler := httpdelivery.NewSubmissionHandler(services.NewSubmissionService(repo))

	rec := httptest.NewRecorder()
	handler.GetSubmissionByID(rec, readSubmissionRequest("0a4f0b52-3d1c-4f8e-9a4a-3c1d2b5e6f70"))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("esperado 404, obtenido %d: %s", rec.Code, rec.Body.String())
	}
}

// Un id con formato inválido es validación: 400.
func TestGetSubmissionByID_MalformedIDReturnsBadRequest(t *testing.T) {
	repo := &readRepo{getErr: fmt.Errorf("%w: no-es-uuid", domain.ErrInvalidResourceID)}
	handler := httpdelivery.NewSubmissionHandler(services.NewSubmissionService(repo))

	rec := httptest.NewRecorder()
	handler.GetSubmissionByID(rec, readSubmissionRequest("no-es-uuid"))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("esperado 400, obtenido %d: %s", rec.Code, rec.Body.String())
	}
}

// Un fallo real del almacén sigue siendo 500: no se disfraza de ausencia.
func TestGetSubmissionByID_StoreFailureReturnsInternalError(t *testing.T) {
	repo := &readRepo{getErr: errors.New("connection refused")}
	handler := httpdelivery.NewSubmissionHandler(services.NewSubmissionService(repo))

	rec := httptest.NewRecorder()
	handler.GetSubmissionByID(rec, readSubmissionRequest("0a4f0b52-3d1c-4f8e-9a4a-3c1d2b5e6f70"))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("esperado 500, obtenido %d: %s", rec.Code, rec.Body.String())
	}
}

// keystrokeRepo acepta cualquier lote: la prueba mira la lectura previa de la
// entrega, no la persistencia de los eventos.
type keystrokeRepo struct{}

func (keystrokeRepo) SaveBatch(_ context.Context, _ string, _ []domain.SubmissionKeystrokeEvent) error {
	return nil
}

func (keystrokeRepo) GetBySubmission(_ context.Context, _ string) ([]domain.SubmissionKeystrokeEvent, error) {
	return nil, nil
}

// Guardar keystrokes sobre una entrega que no existe también es 404.
func TestSaveKeystrokeEvents_MissingSubmissionReturnsNotFound(t *testing.T) {
	repo := &readRepo{getErr: fmt.Errorf("%w: no existe", domain.ErrNotFound)}
	service := services.NewSubmissionService(repo)
	service.SetKeystrokeRepository(keystrokeRepo{})
	handler := httpdelivery.NewSubmissionHandler(service)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/submissions/0a4f0b52-3d1c-4f8e-9a4a-3c1d2b5e6f70/keystroke-events",
		strings.NewReader(`{"events":[{"key":"a","at":1,"line":1,"column":1}]}`))
	req.SetPathValue("id", "0a4f0b52-3d1c-4f8e-9a4a-3c1d2b5e6f70")
	ctx := context.WithValue(req.Context(), domain.TenantIDKey, domain.DefaultTenantID)
	req = req.WithContext(ctx)
	req.Header.Set("X-User-Role", "student")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	handler.SaveKeystrokeEvents(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("esperado 404, obtenido %d: %s", rec.Code, rec.Body.String())
	}
}
