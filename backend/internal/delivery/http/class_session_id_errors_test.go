package httpdelivery_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
	httpdelivery "solv-backend/internal/delivery/http"
)

// aulaStore responde solo a lo que el cierre necesita. closeErr permite simular
// un aula que el almacén no encuentra.
type aulaStore struct {
	domain.ClassSessionStore
	closeErr error
}

func (s *aulaStore) CloseSession(_ context.Context, _, _, _ string) (*domain.ClassSession, error) {
	return nil, s.closeErr
}

func closeAulaRequest(sessionID, role string) *http.Request {
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/teacher/sessions/"+sessionID+"/close", nil)
	ctx := context.WithValue(req.Context(), domain.TenantIDKey, domain.DefaultTenantID)
	ctx = context.WithValue(ctx, domain.UserIDKey, "teacher-aula")
	ctx = context.WithValue(ctx, domain.UserRoleKey, role)
	req = req.WithContext(ctx)
	req.Header.Set("X-User-Id", "teacher-aula")
	req.Header.Set("X-User-Role", role)
	return req
}

// Un aula que el almacén no encuentra responde 404: no es una petición mal
// armada ni un fallo interno.
func TestCloseAula_MissingSessionReturnsNotFound(t *testing.T) {
	handler := httpdelivery.NewClassSessionHandler(
		services.NewClassSessionService(&aulaStore{closeErr: domain.ErrNotFound}),
	)

	rec := httptest.NewRecorder()
	handler.CloseAula(rec, closeAulaRequest("0a4f0b52-3d1c-4f8e-9a4a-3c1d2b5e6f70", "teacher"))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("esperado 404, obtenido %d: %s", rec.Code, rec.Body.String())
	}
}

// Un aula cuya ausencia informa el propio dominio también es 404, para que el
// panel distinga "no existe" de "error del servidor".
func TestCloseAula_SessionNotFoundErrorReturnsNotFound(t *testing.T) {
	handler := httpdelivery.NewClassSessionHandler(
		services.NewClassSessionService(&aulaStore{closeErr: domain.ErrSessionNotFound}),
	)

	rec := httptest.NewRecorder()
	handler.CloseAula(rec, closeAulaRequest("0a4f0b52-3d1c-4f8e-9a4a-3c1d2b5e6f70", "teacher"))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("esperado 404, obtenido %d: %s", rec.Code, rec.Body.String())
	}
}

// Un identificador con formato inválido en el cuerpo sí es una validación: 400.
func TestOpenAula_MalformedSubjectReturnsBadRequest(t *testing.T) {
	handler := httpdelivery.NewClassSessionHandler(
		services.NewClassSessionService(&aulaStore{}),
	)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/teacher/sessions", nil)
	ctx := context.WithValue(req.Context(), domain.TenantIDKey, domain.DefaultTenantID)
	ctx = context.WithValue(ctx, domain.UserIDKey, "teacher-aula")
	ctx = context.WithValue(ctx, domain.UserRoleKey, "teacher")
	req = req.WithContext(ctx)
	req.Header.Set("X-User-Id", "teacher-aula")
	req.Header.Set("X-User-Role", "teacher")
	req.Header.Set("Content-Type", "application/json")
	req.Body = io.NopCloser(strings.NewReader(`{"subject_id":"no-es-un-uuid"}`))

	rec := httptest.NewRecorder()
	handler.OpenAula(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("esperado 400, obtenido %d: %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("respuesta no es JSON: %s", rec.Body.String())
	}
	if body.Error != domain.ErrInvalidResourceID.Error() {
		t.Fatalf("esperado el error de validación %q, obtenido %q", domain.ErrInvalidResourceID.Error(), body.Error)
	}
}
