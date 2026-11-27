package httpdelivery

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

type exportHandlerVolume struct {
	domain.PracticeEnvironmentOrchestrator
	files []domain.PracticeEnvironmentVolumeFile
}

func (v *exportHandlerVolume) SnapshotPracticeEnvironmentVolume(ctx context.Context, volumeName string, maxBytes int64) ([]domain.PracticeEnvironmentVolumeFile, error) {
	return v.files, nil
}

func (v *exportHandlerVolume) WritePracticeEnvironmentVolumeFiles(ctx context.Context, volumeName string, files map[string]string) error {
	return nil
}

func newExportHandlerFixture(ownerID string) (*PracticeEnvironmentHandler, *exportHandlerVolume) {
	exerciseID := "11111111-1111-1111-1111-111111111111"
	repo := &authzPracticeEnvironmentRepo{environment: &domain.PracticeEnvironment{
		ID:           "environment-1",
		TenantID:     "tenant-1",
		StudentID:    ownerID,
		SubjectID:    "subject-1",
		ExerciseID:   &exerciseID,
		Status:       domain.PracticeEnvironmentStatusHibernated,
		SemgrepAudit: json.RawMessage(`{"findings":[]}`),
	}}
	orch := &exportHandlerVolume{
		files: []domain.PracticeEnvironmentVolumeFile{
			{Path: "main.py", Content: []byte("print('hola')\n")},
		},
	}
	svc := services.NewPracticeEnvironmentService(repo, orch, &authzHostMonitor{})
	return NewPracticeEnvironmentHandler(svc, validator.New()), orch
}

func TestExportPracticeEnvironmentReturnsZipAttachment(t *testing.T) {
	handler, _ := newExportHandlerFixture("student-owner")
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/student/environments/{id}/export", handler.ExportPracticeEnvironment)

	req := httptest.NewRequestWithContext(
		practiceEnvironmentRequestContext("student-owner", "student"),
		http.MethodGet,
		"/api/v1/student/environments/environment-1/export",
		nil,
	)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("esperado 200, obtenido %d: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "application/zip" {
		t.Errorf("content-type inesperado: %q", got)
	}
	disposition := rec.Header().Get("Content-Disposition")
	if !strings.Contains(disposition, "attachment; filename=") || !strings.HasSuffix(disposition, ".zip\"") {
		t.Errorf("content-disposition inesperado: %q", disposition)
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("cache-control inesperado: %q", got)
	}

	reader, err := zip.NewReader(bytes.NewReader(rec.Body.Bytes()), int64(rec.Body.Len()))
	if err != nil {
		t.Fatalf("la respuesta no es un zip válido: %v", err)
	}
	names := make(map[string]bool, len(reader.File))
	for _, file := range reader.File {
		names[file.Name] = true
	}
	for _, name := range []string{"main.py", "Dockerfile", "compose.yml", ".env.example", "README.md"} {
		if !names[name] {
			t.Errorf("falta %s en el paquete", name)
		}
	}
}

func TestExportPracticeEnvironmentRejectsAnotherStudent(t *testing.T) {
	handler, _ := newExportHandlerFixture("student-owner")
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/student/environments/{id}/export", handler.ExportPracticeEnvironment)

	req := httptest.NewRequestWithContext(
		practiceEnvironmentRequestContext("student-intruder", "student"),
		http.MethodGet,
		"/api/v1/student/environments/environment-1/export",
		nil,
	)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("esperado 403 para un entorno ajeno, obtenido %d: %s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Content-Type") == "application/zip" {
		t.Error("un rechazo no debe devolver un paquete")
	}
}
