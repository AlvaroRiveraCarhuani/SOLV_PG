package httpdelivery

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-playground/validator/v10"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

type authzPracticeEnvironmentRepo struct {
	domain.PracticeEnvironmentRepository
	environment *domain.PracticeEnvironment
	statuses    []string
	heartbeats  int
}

func (r *authzPracticeEnvironmentRepo) GetByID(context.Context, string) (*domain.PracticeEnvironment, error) {
	if r.environment == nil {
		return nil, domain.ErrNotFound
	}
	return r.environment, nil
}

func (r *authzPracticeEnvironmentRepo) UpdateStatus(_ context.Context, _ string, status string) error {
	r.statuses = append(r.statuses, status)
	return nil
}

func (r *authzPracticeEnvironmentRepo) RecordHeartbeat(context.Context, string) error {
	r.heartbeats++
	return nil
}

func (r *authzPracticeEnvironmentRepo) GetAllRunningPracticeEnvironments(context.Context) ([]*domain.PracticeEnvironment, error) {
	return nil, nil
}

type authzOrchestrator struct {
	domain.PracticeEnvironmentOrchestrator
	stops int
}

func (o *authzOrchestrator) StopAndRemoveContainer(context.Context, string) error {
	o.stops++
	return nil
}

type authzHostMonitor struct {
	domain.HostMonitor
}

func newPracticeEnvironmentAuthzFixture(studentID string) (*PracticeEnvironmentHandler, *authzPracticeEnvironmentRepo, *authzOrchestrator) {
	repo := &authzPracticeEnvironmentRepo{environment: &domain.PracticeEnvironment{
		ID:           "environment-1",
		TenantID:     "tenant-1",
		StudentID:    studentID,
		SubjectID:    "subject-1",
		Status:       domain.PracticeEnvironmentStatusRunning,
		SemgrepAudit: json.RawMessage(`{"findings":[]}`),
	}}
	orch := &authzOrchestrator{}
	svc := services.NewPracticeEnvironmentService(repo, orch, &authzHostMonitor{})
	return NewPracticeEnvironmentHandler(svc, validator.New()), repo, orch
}

func practiceEnvironmentRequestContext(userID, role string) context.Context {
	ctx := context.WithValue(context.Background(), domain.UserIDKey, userID)
	ctx = context.WithValue(ctx, domain.TenantIDKey, "tenant-1")
	return context.WithValue(ctx, domain.UserRoleKey, role)
}

func practiceEnvironmentMux(handler *PracticeEnvironmentHandler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /api/v1/environments/{id}", handler.TerminatePracticeEnvironment)
	mux.HandleFunc("GET /api/v1/environments/{id}/audit", handler.GetSemgrepAudit)
	mux.HandleFunc("POST /api/v1/environments/{id}/heartbeat", handler.Heartbeat)
	mux.HandleFunc("POST /api/v1/environments/{id}/restart", handler.RestartPracticeEnvironment)
	mux.HandleFunc("POST /api/v1/environments/{id}/pause", handler.PausePracticeEnvironment)
	return mux
}

func TestPracticeEnvironmentOperationsRejectOtherStudents(t *testing.T) {
	operations := []struct {
		name   string
		method string
		path   string
	}{
		{name: "terminate", method: http.MethodDelete, path: "/api/v1/environments/environment-1"},
		{name: "audit", method: http.MethodGet, path: "/api/v1/environments/environment-1/audit"},
		{name: "heartbeat", method: http.MethodPost, path: "/api/v1/environments/environment-1/heartbeat"},
		{name: "restart", method: http.MethodPost, path: "/api/v1/environments/environment-1/restart"},
		{name: "pause", method: http.MethodPost, path: "/api/v1/environments/environment-1/pause"},
	}

	for _, operation := range operations {
		t.Run(operation.name, func(t *testing.T) {
			handler, repo, orch := newPracticeEnvironmentAuthzFixture("student-owner")
			mux := practiceEnvironmentMux(handler)
			req := httptest.NewRequestWithContext(
				practiceEnvironmentRequestContext("student-intruder", "student"),
				operation.method,
				operation.path,
				nil,
			)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if rec.Code != http.StatusForbidden {
				t.Fatalf("esperado 403 para un entorno ajeno, obtenido %d: %s", rec.Code, rec.Body.String())
			}
			if len(repo.statuses) != 0 || repo.heartbeats != 0 || orch.stops != 0 {
				t.Fatalf("la operación ajena no debe tener efectos: statuses=%v heartbeats=%d stops=%d", repo.statuses, repo.heartbeats, orch.stops)
			}
		})
	}
}

func TestPracticeEnvironmentOwnerKeepsTerminateAuditHeartbeatAndPause(t *testing.T) {
	handler, repo, _ := newPracticeEnvironmentAuthzFixture("student-owner")
	mux := practiceEnvironmentMux(handler)
	ctx := practiceEnvironmentRequestContext("student-owner", "student")

	heartbeat := httptest.NewRequestWithContext(ctx, http.MethodPost, "/api/v1/environments/environment-1/heartbeat", nil)
	heartbeatRec := httptest.NewRecorder()
	mux.ServeHTTP(heartbeatRec, heartbeat)
	if heartbeatRec.Code != http.StatusOK || repo.heartbeats != 1 {
		t.Fatalf("heartbeat del propietario: status=%d heartbeats=%d", heartbeatRec.Code, repo.heartbeats)
	}

	audit := httptest.NewRequestWithContext(ctx, http.MethodGet, "/api/v1/environments/environment-1/audit", nil)
	auditRec := httptest.NewRecorder()
	mux.ServeHTTP(auditRec, audit)
	if auditRec.Code != http.StatusOK {
		t.Fatalf("auditoría del propietario: status=%d body=%s", auditRec.Code, auditRec.Body.String())
	}

	pause := httptest.NewRequestWithContext(ctx, http.MethodPost, "/api/v1/environments/environment-1/pause", nil)
	pauseRec := httptest.NewRecorder()
	mux.ServeHTTP(pauseRec, pause)
	if pauseRec.Code != http.StatusOK {
		t.Fatalf("pause del propietario: status=%d body=%s", pauseRec.Code, pauseRec.Body.String())
	}
	if len(repo.statuses) != 1 || repo.statuses[0] != domain.PracticeEnvironmentStatusHibernated {
		t.Fatalf("pause debía marcar el entorno como hibernated: %v", repo.statuses)
	}

	terminate := httptest.NewRequestWithContext(ctx, http.MethodDelete, "/api/v1/environments/environment-1", nil)
	terminateRec := httptest.NewRecorder()
	mux.ServeHTTP(terminateRec, terminate)
	if terminateRec.Code != http.StatusOK {
		t.Fatalf("terminate del propietario: status=%d body=%s", terminateRec.Code, terminateRec.Body.String())
	}
	if len(repo.statuses) != 2 || repo.statuses[1] != domain.PracticeEnvironmentStatusFailed {
		t.Fatalf("terminate debía marcar el entorno como failed: %v", repo.statuses)
	}
}
