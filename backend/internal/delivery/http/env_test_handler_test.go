package httpdelivery_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
	httpdelivery "solv-backend/internal/delivery/http"
	"solv-backend/internal/infrastructure/storage/memory"
)

type dummyRegistry struct{}

func (d *dummyRegistry) InspectLocal(ctx context.Context, imageRef string) (bool, int64, string, error) {
	return true, 1024, "sha256:dummy", nil
}
func (d *dummyRegistry) InspectRemoteDigest(ctx context.Context, imageRef string) (string, error) {
	return "sha256:dummy", nil
}
func (d *dummyRegistry) PullImage(ctx context.Context, imageRef string, onProgress func(int64, int64, int, int, string)) error {
	return nil
}

type dummyRunner struct{}

func (d *dummyRunner) RunSmokeTest(ctx context.Context, imageRef string, tools []string, memoryLimitMB int64) ([]domain.ToolResult, int, error) {
	var res []domain.ToolResult
	for _, t := range tools {
		res = append(res, domain.ToolResult{Name: t, Present: true, Path: "/bin/" + t})
	}
	return res, 0, nil
}

func (d *dummyRunner) RunJudgeSmokeTest(ctx context.Context, imageRef string, entrypoint string, sampleInput string, timeoutMS int, memoryLimitMB int64) (string, int64, int, error) {
	return "OK: judge passed", 12, 0, nil
}

func jobCtx(userID, tenantID string) context.Context {
	ctx := context.WithValue(context.Background(), domain.UserIDKey, userID)
	return context.WithValue(ctx, domain.TenantIDKey, tenantID)
}

func TestEnvTestHandler_Endpoints(t *testing.T) {
	repo := memory.NewEnvTestJobMemoryRepository(time.Hour)
	reg := &dummyRegistry{}
	runner := &dummyRunner{}
	svc := services.NewEnvTestService(repo, reg, runner, services.EnvTestConfig{})
	handler := httpdelivery.NewEnvTestHandler(svc)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/jobs/env-test", handler.StartEnvTest)
	mux.HandleFunc("GET /api/v1/jobs/env-test/{id}", handler.GetEnvTest)
	mux.HandleFunc("POST /api/v1/jobs/env-test/{id}/cancel", handler.CancelEnvTest)

	// 1. Iniciar Job
	reqBody := []byte(`{"image":"python:3.12-slim","tools":["python","pip"]}`)
	req := httptest.NewRequestWithContext(jobCtx("admin-1", "tenant-1"), "POST", "/api/v1/jobs/env-test", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202 Accepted, got %d: %s", w.Code, w.Body.String())
	}

	var startResp struct {
		Data    domain.EnvTestJob `json:"data"`
		Message string            `json:"message"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &startResp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	jobID := startResp.Data.ID
	if jobID == "" {
		t.Fatalf("expected valid job ID in response")
	}

	// 2. Consultar Job
	getReq := httptest.NewRequestWithContext(jobCtx("admin-1", "tenant-1"), "GET", "/api/v1/jobs/env-test/"+jobID, nil)
	getW := httptest.NewRecorder()
	mux.ServeHTTP(getW, getReq)

	if getW.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", getW.Code, getW.Body.String())
	}

	// 3. Un usuario distinto no puede leer el job (404, sin filtrar existencia)
	foreignReq := httptest.NewRequestWithContext(jobCtx("admin-2", "tenant-1"), "GET", "/api/v1/jobs/env-test/"+jobID, nil)
	foreignW := httptest.NewRecorder()
	mux.ServeHTTP(foreignW, foreignReq)

	if foreignW.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for foreign owner, got %d: %s", foreignW.Code, foreignW.Body.String())
	}

	// 4. El mismo usuario de otro tenant tampoco puede leer el job
	foreignTenantReq := httptest.NewRequestWithContext(jobCtx("admin-1", "tenant-2"), "GET", "/api/v1/jobs/env-test/"+jobID, nil)
	foreignTenantW := httptest.NewRecorder()
	mux.ServeHTTP(foreignTenantW, foreignTenantReq)

	if foreignTenantW.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for foreign tenant, got %d: %s", foreignTenantW.Code, foreignTenantW.Body.String())
	}

	// 5. Cancelar Job
	cancelReq := httptest.NewRequestWithContext(jobCtx("admin-1", "tenant-1"), "POST", "/api/v1/jobs/env-test/"+jobID+"/cancel", nil)
	cancelW := httptest.NewRecorder()
	mux.ServeHTTP(cancelW, cancelReq)

	if cancelW.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on cancel, got %d: %s", cancelW.Code, cancelW.Body.String())
	}
}
