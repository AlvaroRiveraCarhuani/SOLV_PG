package httpdelivery_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
	httpdelivery "solv-backend/internal/delivery/http"
)

// subjectRepo simula la lectura del entorno de una materia. subjectErr permite
// representar una materia que el almacén no encuentra.
type subjectRepo struct {
	domain.WorkspaceRepository
	subjectErr error
}

func (r *subjectRepo) GetByStudentAndSubject(_ context.Context, _, _ string) (*domain.WorkspaceInstance, error) {
	return nil, r.subjectErr
}

func workspaceForSubjectRequest(rawQuery string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/current"+rawQuery, nil)
	ctx := context.WithValue(req.Context(), domain.TenantIDKey, domain.DefaultTenantID)
	ctx = context.WithValue(ctx, domain.UserIDKey, "teacher-lab")
	ctx = context.WithValue(ctx, domain.UserRoleKey, "teacher")
	req = req.WithContext(ctx)
	req.Header.Set("X-User-Id", "teacher-lab")
	return req
}

func newWorkspaceForSubjectHandler(repo domain.WorkspaceRepository) *httpdelivery.WorkspaceHandler {
	return httpdelivery.NewWorkspaceHandler(services.NewWorkspaceService(repo, nil, nil), nil)
}

// Una materia con formato inválido es una petición mal armada: 400 con el
// error de validación, distinto del de un laboratorio que no existe.
func TestGetWorkspaceForSubject_MalformedSubjectReturnsBadRequest(t *testing.T) {
	handler := newWorkspaceForSubjectHandler(&subjectRepo{})

	rec := httptest.NewRecorder()
	handler.GetWorkspaceForSubject(rec, workspaceForSubjectRequest("?subject_id=programacion-1"))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("esperado 400, obtenido %d: %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("respuesta no es JSON: %s", rec.Body.String())
	}
	if !strings.HasPrefix(body.Error, domain.ErrInvalidResourceID.Error()) {
		t.Fatalf("esperado el error de validación %q, obtenido %q", domain.ErrInvalidResourceID.Error(), body.Error)
	}
}

// Una materia que el almacén no encuentra es un recurso ausente: 404, no 400.
func TestGetWorkspaceForSubject_MissingSubjectReturnsNotFound(t *testing.T) {
	handler := newWorkspaceForSubjectHandler(&subjectRepo{
		subjectErr: fmt.Errorf("materia: %w", domain.ErrNotFound),
	})

	rec := httptest.NewRecorder()
	handler.GetWorkspaceForSubject(rec, workspaceForSubjectRequest("?subject_id=0a4f0b52-3d1c-4f8e-9a4a-3c1d2b5e6f70"))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("esperado 404, obtenido %d: %s", rec.Code, rec.Body.String())
	}
}
