package services

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"solv-backend/internal/core/domain"
)

type auditUpdateCall struct {
	templateID      string
	smokeStatus     string
	smokeOutput     string
	secStatus       string
	cveCritical     int
	cveHigh         int
	secReportJSON   []byte
	finalStatus     string
	rejectionReason string
}

type recordingAuditRepo struct {
	domain.AdminGovernanceRepository
	calls []auditUpdateCall
}

func (r *recordingAuditRepo) UpdateAuditResults(
	_ context.Context,
	templateID string,
	smokeStatus, smokeOutput, secStatus string,
	cveCritical, cveHigh int,
	secReportJSON []byte,
	finalStatus, rejectionReason string,
) error {
	r.calls = append(r.calls, auditUpdateCall{
		templateID:      templateID,
		smokeStatus:     smokeStatus,
		smokeOutput:     smokeOutput,
		secStatus:       secStatus,
		cveCritical:     cveCritical,
		cveHigh:         cveHigh,
		secReportJSON:   secReportJSON,
		finalStatus:     finalStatus,
		rejectionReason: rejectionReason,
	})
	return nil
}

func (r *recordingAuditRepo) UpdateEOLStatus(_ context.Context, _, _, _, _ string) error {
	return nil
}

// AuditTemplate decide y persiste el estado final. La prueba reemplaza los
// runners de Docker y Trivy y el repositorio, de modo que verifica la llamada a
// UpdateAuditResults sin daemon, red ni SQL.
func TestAuditTemplatePersistsAuditOutcome(t *testing.T) {
	cases := []struct {
		name            string
		current         string
		smokeStatus     string
		smokeOutput     string
		secStatus       string
		cveCritical     int
		cveHigh         int
		report          []byte
		wantStatus      string
		wantReason      string
		wantReasonEmpty bool
		wantSmoke       string
		wantSec         string
		wantCritical    int
		wantHigh        int
	}{
		{
			name:            "publica una plantilla pendiente cuando todo pasa",
			current:         "PENDIENTE_AUDITORIA",
			smokeStatus:     "passed",
			secStatus:       "passed",
			report:          []byte(`{"ok":true}`),
			wantStatus:      "APROBADA",
			wantReasonEmpty: true,
			wantSmoke:       "passed",
			wantSec:         "passed",
		},
		{
			name:        "rechaza en admisión por un smoke fallido",
			current:     "PENDIENTE_AUDITORIA",
			smokeStatus: "failed",
			smokeOutput: "Falta la herramienta: ripgrep",
			secStatus:   "passed",
			report:      []byte(`{"ok":true}`),
			wantStatus:  "RECHAZADA",
			wantReason:  "Falta la herramienta",
			wantSmoke:   "failed",
			wantSec:     "passed",
		},
		{
			name:         "rechaza en admisión por un CVE crítico",
			current:      "PENDIENTE_AUDITORIA",
			smokeStatus:  "passed",
			secStatus:    "failed",
			cveCritical:  3,
			cveHigh:      1,
			report:       []byte(`{"Critical":3,"High":1}`),
			wantStatus:   "RECHAZADA",
			wantReason:   "3 vulnerabilidades críticas",
			wantSmoke:    "passed",
			wantSec:      "failed",
			wantCritical: 3,
			wantHigh:     1,
		},
		{
			name:            "un re-escaneo no retira una plantilla publicada",
			current:         "APROBADA",
			smokeStatus:     "passed",
			secStatus:       "failed",
			cveCritical:     2,
			cveHigh:         478,
			report:          []byte(`{"Critical":2,"High":478}`),
			wantStatus:      "APROBADA",
			wantReasonEmpty: true,
			wantSmoke:       "passed",
			wantSec:         "failed",
			wantCritical:    2,
			wantHigh:        478,
		},
		{
			name:        "conserva pendiente cuando el escaneo no produce informe",
			current:     "PENDIENTE_AUDITORIA",
			smokeStatus: "passed",
			secStatus:   "failed",
			report:      []byte(`{"error":"scan"}`),
			wantStatus:  "PENDIENTE_AUDITORIA",
			wantReason:  "Auditoría no completada",
			wantSmoke:   "passed",
			wantSec:     "failed",
		},
		{
			name:            "conserva aprobada sin pisar el motivo si Trivy no se pudo ejecutar",
			current:         "APROBADA",
			smokeStatus:     "passed",
			secStatus:       "error",
			report:          []byte(`{"error":"trivy_unavailable"}`),
			wantStatus:      "APROBADA",
			wantReasonEmpty: true,
			wantSmoke:       "passed",
			wantSec:         "error",
		},
		{
			name:        "conserva pendiente si el smoke no se pudo ejecutar",
			current:     "PENDIENTE_AUDITORIA",
			smokeStatus: "error",
			smokeOutput: "Docker no responde",
			secStatus:   "passed",
			report:      []byte(`{"error":"smoke_infrastructure"}`),
			wantStatus:  "PENDIENTE_AUDITORIA",
			wantReason:  "Auditoría no completada",
			wantSmoke:   "error",
			wantSec:     "skipped",
		},
		{
			name:        "rechaza por smoke aunque el escaneo no se complete",
			current:     "PENDIENTE_AUDITORIA",
			smokeStatus: "failed",
			smokeOutput: "Falta la herramienta: ripgrep",
			secStatus:   "error",
			report:      []byte(`{"error":"trivy_unavailable"}`),
			wantStatus:  "RECHAZADA",
			wantReason:  "Falta la herramienta",
			wantSmoke:   "failed",
			wantSec:     "error",
		},
		{
			name:        "conserva pausada aunque falle el smoke",
			current:     "PAUSADA",
			smokeStatus: "failed",
			smokeOutput: "Falta la herramienta: ripgrep",
			secStatus:   "passed",
			report:      []byte(`{"ok":true}`),
			wantStatus:  "PAUSADA",
			wantReason:  "Falta la herramienta",
			wantSmoke:   "failed",
			wantSec:     "passed",
		},
		{
			name:            "marca warning cuando hay hallazgos altos",
			current:         "PENDIENTE_AUDITORIA",
			smokeStatus:     "passed",
			secStatus:       "passed",
			cveHigh:         2,
			report:          []byte(`{"High":2}`),
			wantStatus:      "APROBADA",
			wantReasonEmpty: true,
			wantSmoke:       "passed",
			wantSec:         "warning",
			wantHigh:        2,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &recordingAuditRepo{}
			worker := NewTemplateAuditWorker(nil, repo)
			worker.smokeRunner = func(context.Context, *domain.AdminTemplateReviewItem) (string, string) {
				return tc.smokeStatus, tc.smokeOutput
			}
			worker.trivyRunner = func(context.Context, string) (string, int, int, []byte) {
				return tc.secStatus, tc.cveCritical, tc.cveHigh, tc.report
			}

			item := &domain.AdminTemplateReviewItem{
				ID:          "tpl-audit-1",
				Name:        "Plantilla de auditoría",
				DockerImage: "local/audit-test",
				BaseRamMB:   0,
				Status:      tc.current,
			}
			if err := worker.AuditTemplate(context.Background(), item); err != nil && !errors.Is(err, errAuditInfrastructure) {
				t.Fatalf("AuditTemplate devolvió error: %v", err)
			}
			if len(repo.calls) != 1 {
				t.Fatalf("esperaba 1 llamada a UpdateAuditResults, obtuve %d", len(repo.calls))
			}
			call := repo.calls[0]
			if call.templateID != item.ID {
				t.Fatalf("templateID = %q, quiero %q", call.templateID, item.ID)
			}
			if call.finalStatus != tc.wantStatus {
				t.Fatalf("estado final = %q, quiero %q (motivo: %q)", call.finalStatus, tc.wantStatus, call.rejectionReason)
			}
			if tc.wantReasonEmpty {
				if call.rejectionReason != "" {
					t.Fatalf("no esperaba motivo, obtuve %q", call.rejectionReason)
				}
			} else if !strings.Contains(call.rejectionReason, tc.wantReason) {
				t.Fatalf("motivo = %q, quiero que contenga %q", call.rejectionReason, tc.wantReason)
			}
			if call.smokeStatus != tc.wantSmoke {
				t.Fatalf("smokeStatus = %q, quiero %q", call.smokeStatus, tc.wantSmoke)
			}
			if call.secStatus != tc.wantSec {
				t.Fatalf("secStatus = %q, quiero %q", call.secStatus, tc.wantSec)
			}
			if call.cveCritical != tc.wantCritical || call.cveHigh != tc.wantHigh {
				t.Fatalf("CVEs = (%d, %d), quiero (%d, %d)", call.cveCritical, call.cveHigh, tc.wantCritical, tc.wantHigh)
			}
			if !bytes.Equal(call.secReportJSON, tc.report) {
				t.Fatalf("reporte = %s, quiero %s", call.secReportJSON, tc.report)
			}
		})
	}
}

// Un re-escaneo que solo espera Trivy no vuelve a lanzar el contenedor de
// smoke: conserva el estado aprobado y su salida.
func TestAuditTemplateReusesPassedSmoke(t *testing.T) {
	repo := &recordingAuditRepo{}
	worker := NewTemplateAuditWorker(nil, repo)
	smokeCalls := 0
	worker.smokeRunner = func(context.Context, *domain.AdminTemplateReviewItem) (string, string) {
		smokeCalls++
		return "failed", "no debía ejecutarse"
	}
	worker.trivyRunner = func(context.Context, string) (string, int, int, []byte) {
		return "passed", 0, 0, []byte(`{"ok":true}`)
	}

	item := &domain.AdminTemplateReviewItem{
		ID:                  "tpl-reaudit",
		Name:                "Plantilla reauditada",
		DockerImage:         "local/audit-test",
		Status:              "APROBADA",
		SmokeTestStatus:     "passed",
		SmokeTestOutput:     "Verificado durante la compilación en el Constructor",
		SecurityAuditStatus: "pending",
	}
	if err := worker.AuditTemplate(context.Background(), item); err != nil {
		t.Fatalf("AuditTemplate devolvió error: %v", err)
	}
	if smokeCalls != 0 {
		t.Fatalf("el smoke se ejecutó %d veces; debía reaprovecharse el aprobado", smokeCalls)
	}
	call := repo.calls[0]
	if call.smokeStatus != "passed" || call.smokeOutput != item.SmokeTestOutput {
		t.Fatalf("smoke persistido = (%q, %q), quiero el aprobado original", call.smokeStatus, call.smokeOutput)
	}
	if call.finalStatus != "APROBADA" {
		t.Fatalf("estado = %q, quiero APROBADA", call.finalStatus)
	}
}

type pendingAuditRepo struct {
	*recordingAuditRepo
	pending []*domain.AdminTemplateReviewItem
}

func (r *pendingAuditRepo) ListPendingAuditTemplates(context.Context) ([]*domain.AdminTemplateReviewItem, error) {
	return r.pending, nil
}

// La cola puede devolver una plantilla pausada, suspendida o rechazada porque
// su smoke o su escaneo quedaron pendientes. El reconciliador no debe
// auditarlas: el estado lo decide una persona y, al reactivarlas, la cola las
// vuelve a tomar.
func TestReconcilePendingSkipsHumanControlledStatuses(t *testing.T) {
	repo := &pendingAuditRepo{
		recordingAuditRepo: &recordingAuditRepo{},
		pending: []*domain.AdminTemplateReviewItem{
			{ID: "tpl-paused", Name: "Pausada", DockerImage: "local/audit-test", Status: "PAUSADA"},
			{ID: "tpl-suspended", Name: "Suspendida", DockerImage: "local/audit-test", Status: "SUSPENDIDA"},
			{ID: "tpl-rejected", Name: "Rechazada", DockerImage: "local/audit-test", Status: "RECHAZADA"},
			{ID: "tpl-pending", Name: "Pendiente", DockerImage: "local/audit-test", Status: "PENDIENTE_AUDITORIA"},
		},
	}
	worker := NewTemplateAuditWorker(nil, repo)
	worker.cacheDir = t.TempDir()
	writeTrivyDB(t, worker.cacheDir)
	worker.smokeRunner = func(context.Context, *domain.AdminTemplateReviewItem) (string, string) {
		return "passed", ""
	}
	worker.trivyRunner = func(context.Context, string) (string, int, int, []byte) {
		return "passed", 0, 0, []byte(`{"ok":true}`)
	}

	worker.reconcilePending(context.Background())

	if len(repo.calls) != 1 {
		t.Fatalf("esperaba una sola auditoría, obtuve %d", len(repo.calls))
	}
	if repo.calls[0].templateID != "tpl-pending" {
		t.Fatalf("se auditó %q; pausada, suspendida y rechazada debían omitirse", repo.calls[0].templateID)
	}
}
