package httpdelivery_test

import (
	"context"
	"fmt"
	"io"
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
	review    *domain.TeacherSubmissionReviewDTO
	added     *domain.SubmissionComment
}

func (r *teacherReadRepo) GetTeacherSubmissionReview(_ context.Context, _, _, _ string) (*domain.TeacherSubmissionReviewDTO, error) {
	if r.review != nil {
		return r.review, nil
	}
	return nil, r.reviewErr
}

func (r *teacherReadRepo) AddComment(_ context.Context, comment *domain.SubmissionComment) error {
	r.added = comment
	return nil
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

// El runner efímero es exclusivo del juez: una entrega IDE responde 409.
func TestTeacherRunEphemeral_IDEReturnsConflict(t *testing.T) {
	repo := &teacherReadRepo{review: &domain.TeacherSubmissionReviewDTO{
		ID:              readSubmissionID,
		EnvironmentType: domain.PracticeEnvironmentTypeIDEPersistente,
	}}
	handler := newTeacherReadHandler(repo)

	rec := httptest.NewRecorder()
	handler.RunEphemeral(rec, teacherReadRequest(http.MethodPost, "/api/v1/teacher/submissions/"+readSubmissionID+"/run-ephemeral", readSubmissionID))

	if rec.Code != http.StatusConflict {
		t.Fatalf("esperado 409, obtenido %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "ephemeral_not_supported_for_ide") {
		t.Fatalf("respuesta sin el código de error esperado: %s", rec.Body.String())
	}
}

// Comentar una línea que no existe en la entrega responde 422, no 500.
func TestTeacherAddComment_LineOutOfRangeReturnsUnprocessable(t *testing.T) {
	repo := &teacherReadRepo{review: &domain.TeacherSubmissionReviewDTO{
		ID:   readSubmissionID,
		Code: "primera linea\nsegunda linea\n",
	}}
	handler := newTeacherReadHandler(repo)

	req := teacherReadRequest(
		http.MethodPost,
		"/api/v1/teacher/submissions/"+readSubmissionID+"/comments",
		readSubmissionID,
	)
	req.Body = io.NopCloser(strings.NewReader(`{"line_number": 9, "comment": "fuera de rango"}`))
	rec := httptest.NewRecorder()

	handler.AddSubmissionComment(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("esperado 422, obtenido %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "line_out_of_range") {
		t.Fatalf("respuesta sin el código de error esperado: %s", rec.Body.String())
	}
}

// Una línea dentro del snapshot se acepta sin traducir nada en el backend.
func TestTeacherAddComment_LineInsideSnapshotReturnsCreated(t *testing.T) {
	repo := &teacherReadRepo{review: &domain.TeacherSubmissionReviewDTO{
		ID:   readSubmissionID,
		Code: "primera linea\nsegunda linea\n",
	}}
	handler := newTeacherReadHandler(repo)

	req := teacherReadRequest(
		http.MethodPost,
		"/api/v1/teacher/submissions/"+readSubmissionID+"/comments",
		readSubmissionID,
	)
	req.Body = io.NopCloser(strings.NewReader(`{"line_number": 2, "comment": "en rango"}`))
	rec := httptest.NewRecorder()

	handler.AddSubmissionComment(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("esperado 201, obtenido %d: %s", rec.Code, rec.Body.String())
	}
}

// Un comentario de una línea en una cabecera sigue siendo válido: es la nota
// general de la rúbrica. La restricción de archivo aplica solo a los bloques.
func TestTeacherAddComment_HeaderLineReturnsCreated(t *testing.T) {
	repo := &teacherReadRepo{review: &domain.TeacherSubmissionReviewDTO{
		ID:              readSubmissionID,
		EnvironmentType: domain.PracticeEnvironmentTypeIDEPersistente,
		Code:            "# === FILE: a.py ===\nx\ny\n",
		FilesAvailable:  true,
		Files: []domain.ProjectFile{
			{Path: "a.py", StartLine: 2, LineCount: 2},
		},
	}}
	handler := newTeacherReadHandler(repo)

	req := teacherReadRequest(
		http.MethodPost,
		"/api/v1/teacher/submissions/"+readSubmissionID+"/comments",
		readSubmissionID,
	)
	req.Body = io.NopCloser(strings.NewReader(`{"line_number": 1, "comment": "nota general"}`))
	rec := httptest.NewRecorder()

	handler.AddSubmissionComment(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("esperado 201, obtenido %d: %s", rec.Code, rec.Body.String())
	}
	if repo.added == nil || repo.added.LineNumber != 1 {
		t.Fatal("el comentario de cabecera debe llegar al repositorio con su línea global")
	}
}

// Un bloque IDE debe vivir dentro de un solo archivo: cruzar cabeceras o
// archivos responde 422 y no se guarda el comentario.
func TestTeacherAddComment_RangeAcrossFilesReturnsUnprocessable(t *testing.T) {
	repo := &teacherReadRepo{review: &domain.TeacherSubmissionReviewDTO{
		ID:              readSubmissionID,
		EnvironmentType: domain.PracticeEnvironmentTypeIDEPersistente,
		Code:            "# === FILE: a.py ===\nx\ny\n\n# === FILE: b.py ===\nz\nw\n",
		FilesAvailable:  true,
		Files: []domain.ProjectFile{
			{Path: "a.py", StartLine: 2, LineCount: 2},
			{Path: "b.py", StartLine: 5, LineCount: 2},
		},
	}}
	handler := newTeacherReadHandler(repo)

	req := teacherReadRequest(
		http.MethodPost,
		"/api/v1/teacher/submissions/"+readSubmissionID+"/comments",
		readSubmissionID,
	)
	req.Body = io.NopCloser(strings.NewReader(`{"line_number": 2, "end_line_number": 6, "comment": "bloque"}`))
	rec := httptest.NewRecorder()

	handler.AddSubmissionComment(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("esperado 422, obtenido %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "line_range_spans_files") {
		t.Fatalf("respuesta sin el código de error esperado: %s", rec.Body.String())
	}
	if repo.added != nil {
		t.Fatal("un rango inválido no debe llegar al repositorio")
	}
}

// Un rango invertido es validación: 422 line_range_invalid.
func TestTeacherAddComment_ReversedRangeReturnsUnprocessable(t *testing.T) {
	repo := &teacherReadRepo{review: &domain.TeacherSubmissionReviewDTO{
		ID:   readSubmissionID,
		Code: "primera linea\nsegunda linea\ntercera linea\n",
	}}
	handler := newTeacherReadHandler(repo)

	req := teacherReadRequest(
		http.MethodPost,
		"/api/v1/teacher/submissions/"+readSubmissionID+"/comments",
		readSubmissionID,
	)
	req.Body = io.NopCloser(strings.NewReader(`{"line_number": 3, "end_line_number": 2, "comment": "al revés"}`))
	rec := httptest.NewRecorder()

	handler.AddSubmissionComment(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("esperado 422, obtenido %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "line_range_invalid") {
		t.Fatalf("respuesta sin el código de error esperado: %s", rec.Body.String())
	}
}

// Un bloque dentro del mismo archivo conserva la línea inicial y el cierre.
func TestTeacherAddComment_BlockInsideOneFileIsPersisted(t *testing.T) {
	repo := &teacherReadRepo{review: &domain.TeacherSubmissionReviewDTO{
		ID:              readSubmissionID,
		EnvironmentType: domain.PracticeEnvironmentTypeIDEPersistente,
		Code:            "# === FILE: a.py ===\nx\ny\nz\n",
		FilesAvailable:  true,
		Files: []domain.ProjectFile{
			{Path: "a.py", StartLine: 2, LineCount: 3},
		},
	}}
	handler := newTeacherReadHandler(repo)

	req := teacherReadRequest(
		http.MethodPost,
		"/api/v1/teacher/submissions/"+readSubmissionID+"/comments",
		readSubmissionID,
	)
	req.Body = io.NopCloser(strings.NewReader(`{"line_number": 2, "end_line_number": 4, "comment": "bloque"}`))
	rec := httptest.NewRecorder()

	handler.AddSubmissionComment(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("esperado 201, obtenido %d: %s", rec.Code, rec.Body.String())
	}
	if repo.added == nil || repo.added.EndLineNumber == nil || *repo.added.EndLineNumber != 4 {
		t.Fatalf("el rango no llegó al repositorio: %+v", repo.added)
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
