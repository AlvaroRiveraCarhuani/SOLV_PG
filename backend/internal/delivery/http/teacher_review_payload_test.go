package httpdelivery_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"solv-backend/internal/core/domain"
)

func encodeReview(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("respuesta no es JSON: %v (%s)", err, rec.Body.String())
	}
	return body.Data
}

func TestTeacherSubmissionReview_OmitsSnapshotInIDE(t *testing.T) {
	repo := &teacherReadRepo{review: &domain.TeacherSubmissionReviewDTO{
		ID:                 "sub-ide",
		EnvironmentType:    domain.PracticeEnvironmentTypeIDEPersistente,
		Language:           "javascript",
		FilesAvailable:     true,
		BaseFilesAvailable: true,
		Code:               "# === FILE: src/app.js ===\nconst a = 1;\n",
		Boilerplate:        "# === FILE: src/app.js ===\nconst a = 0;\n",
		Files:              []domain.ProjectFile{{Path: "src/app.js", StartLine: 2, LineCount: 1}},
		BaseFiles:          []domain.ProjectFile{{Path: "src/app.js", StartLine: 2, LineCount: 1}},
	}}
	handler := newTeacherReadHandler(repo)

	rec := httptest.NewRecorder()
	handler.GetSubmissionReview(rec, teacherReadRequest(http.MethodGet, "/api/v1/teacher/submissions/"+readSubmissionID, readSubmissionID))
	if rec.Code != http.StatusOK {
		t.Fatalf("esperado 200, obtenido %d: %s", rec.Code, rec.Body.String())
	}
	data := encodeReview(t, rec)
	if _, ok := data["code"]; ok {
		t.Fatalf("la entrega IDE no debe enviar code: %v", data["code"])
	}
	if _, ok := data["boilerplate"]; ok {
		t.Fatalf("la entrega IDE no debe enviar boilerplate")
	}
	files, ok := data["files"].([]any)
	if !ok || len(files) != 1 {
		t.Fatalf("esperado un archivo navegable, obtenido %v", data["files"])
	}
}

func TestTeacherSubmissionReview_KeepsCodeInJudge(t *testing.T) {
	repo := &teacherReadRepo{review: &domain.TeacherSubmissionReviewDTO{
		ID:              "sub-judge",
		EnvironmentType: domain.PracticeEnvironmentTypeJuezEfimero,
		Code:            "print(1)\n",
		FilesAvailable:  false,
	}}
	handler := newTeacherReadHandler(repo)

	rec := httptest.NewRecorder()
	handler.GetSubmissionReview(rec, teacherReadRequest(http.MethodGet, "/api/v1/teacher/submissions/"+readSubmissionID, readSubmissionID))
	if rec.Code != http.StatusOK {
		t.Fatalf("esperado 200, obtenido %d: %s", rec.Code, rec.Body.String())
	}
	data := encodeReview(t, rec)
	if data["code"] != "print(1)\n" {
		t.Fatalf("el juez debe conservar code, obtenido %v", data["code"])
	}
	if data["files"] != nil {
		t.Fatalf("el juez debe enviar files null, obtenido %v", data["files"])
	}
}

func TestTeacherSubmissionReview_KeepsRawCodeWhenSnapshotNotParsed(t *testing.T) {
	repo := &teacherReadRepo{review: &domain.TeacherSubmissionReviewDTO{
		ID:              "sub-fallback",
		EnvironmentType: domain.PracticeEnvironmentTypeIDEPersistente,
		Code:            "texto crudo sin cabeceras",
		FilesAvailable:  false,
	}}
	handler := newTeacherReadHandler(repo)

	rec := httptest.NewRecorder()
	handler.GetSubmissionReview(rec, teacherReadRequest(http.MethodGet, "/api/v1/teacher/submissions/"+readSubmissionID, readSubmissionID))
	data := encodeReview(t, rec)
	if data["code"] != "texto crudo sin cabeceras" {
		t.Fatalf("el fallback debe conservar el texto crudo, obtenido %v", data["code"])
	}
}

func TestTeacherSubmissionComment_UnknownFileReturns422(t *testing.T) {
	repo := &teacherReadRepo{review: &domain.TeacherSubmissionReviewDTO{
		ID:              "sub-ide",
		EnvironmentType: domain.PracticeEnvironmentTypeIDEPersistente,
		FilesAvailable:  true,
		Code:            "# === FILE: src/app.js ===\nconst a = 1;\n",
		Files:           []domain.ProjectFile{{Path: "src/app.js", StartLine: 2, LineCount: 1}},
	}}
	handler := newTeacherReadHandler(repo)

	body := `{"file_path":"src/otro.js","local_line_number":1,"comment":"Revisar"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/teacher/submissions/"+readSubmissionID+"/comments", strings.NewReader(body))
	req.SetPathValue("id", readSubmissionID)
	ctx := context.WithValue(req.Context(), domain.TenantIDKey, domain.DefaultTenantID)
	req = req.WithContext(ctx)
	req = withTestIdentity(req, "teacher-read", "", "teacher")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	handler.AddSubmissionComment(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("esperado 422, obtenido %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "comment_file_unknown") {
		t.Fatalf("esperado error comment_file_unknown: %s", rec.Body.String())
	}
}
