package services

import (
	"context"
	"errors"
	"testing"

	"solv-backend/internal/core/domain"
)

// ---------------------------------------------------------------------------
// Dobles: ninguno toca el daemon real ni la base.
// ---------------------------------------------------------------------------

// reconcilerRepo emula el repositorio con estado en memoria: expone las filas
// activas y registra las transiciones pedidas.
type reconcilerRepo struct {
	domain.WorkspaceRepository
	workspaces   []*domain.WorkspaceInstance
	updates      []statusChange
	oomStrikes   []string
	getActiveErr error
	updateStatus func(id, status string) error
}

type statusChange struct {
	id     string
	status string
}

func (r *reconcilerRepo) GetActiveWorkspaces(context.Context) ([]*domain.WorkspaceInstance, error) {
	if r.getActiveErr != nil {
		return nil, r.getActiveErr
	}
	out := make([]*domain.WorkspaceInstance, 0, len(r.workspaces))
	for _, ws := range r.workspaces {
		copy := *ws
		out = append(out, &copy)
	}
	return out, nil
}

func (r *reconcilerRepo) UpdateStatus(_ context.Context, id string, status string) error {
	if r.updateStatus != nil {
		return r.updateStatus(id, status)
	}
	r.updates = append(r.updates, statusChange{id: id, status: status})
	for _, ws := range r.workspaces {
		if ws.ID == id {
			ws.Status = status
		}
	}
	return nil
}

func (r *reconcilerRepo) IncrementOOMStrike(_ context.Context, id string) error {
	r.oomStrikes = append(r.oomStrikes, id)
	for _, ws := range r.workspaces {
		if ws.ID == id {
			ws.Status = domain.WorkspaceStatusOOMKilled
		}
	}
	return nil
}

// reconcilerOrchestrator emula el daemon: lista manejados, inspecciona y
// elimina. Los errores se inyectan por escenario.
type reconcilerOrchestrator struct {
	domain.WorkspaceOrchestrator
	managed    []string
	listErr    error
	metrics    map[string]*domain.ContainerMetrics
	metricsErr error
	removed    []string
}

func (o *reconcilerOrchestrator) ListAllManagedContainers(context.Context) ([]string, error) {
	if o.listErr != nil {
		return nil, o.listErr
	}
	return o.managed, nil
}

func (o *reconcilerOrchestrator) GetContainerMetrics(_ context.Context, id string) (*domain.ContainerMetrics, error) {
	if o.metricsErr != nil {
		return nil, o.metricsErr
	}
	if o.metrics != nil {
		if m, ok := o.metrics[id]; ok {
			return m, nil
		}
	}
	// Por defecto un contenedor que el daemon lista existe y esta en marcha.
	return &domain.ContainerMetrics{IsRunning: true}, nil
}

func (o *reconcilerOrchestrator) StopAndRemoveContainer(_ context.Context, id string) error {
	o.removed = append(o.removed, id)
	return nil
}

// reconcilerAudit registra las transiciones en auditoria.
type reconcilerAudit struct {
	domain.AuditLogRepository
	logs []*domain.AuditLog
}

func (a *reconcilerAudit) Create(_ context.Context, log *domain.AuditLog) error {
	a.logs = append(a.logs, log)
	return nil
}

func newReconcilerHarness() (*WorkspaceStateReconciler, *reconcilerRepo, *reconcilerOrchestrator, *reconcilerAudit) {
	repo := &reconcilerRepo{}
	orch := &reconcilerOrchestrator{
		metrics: make(map[string]*domain.ContainerMetrics),
	}
	audit := &reconcilerAudit{}
	r := NewWorkspaceStateReconciler(repo, orch)
	r.SetAuditRepository(audit)
	return r, repo, orch, audit
}

func ws(id, cid, status string) *domain.WorkspaceInstance {
	c := cid
	return &domain.WorkspaceInstance{
		ID:          id,
		ContainerID: &c,
		Status:      status,
		TenantID:    domain.DefaultTenantID,
	}
}

// ---------------------------------------------------------------------------
// Escenario 1: Docker responde y el contenedor existe -> nada cambia.
// ---------------------------------------------------------------------------

func TestReconcileDockerOkContainerExistsNoTransition(t *testing.T) {
	r, repo, orch, _ := newReconcilerHarness()
	repo.workspaces = []*domain.WorkspaceInstance{ws("w1", "c1", domain.WorkspaceStatusRunning)}
	orch.managed = []string{"c1"}

	r.Reconcile(context.Background())

	if len(repo.updates) != 0 {
		t.Fatalf("expected no transition, got %+v", repo.updates)
	}
	if len(repo.oomStrikes) != 0 {
		t.Fatalf("expected no oom strikes, got %v", repo.oomStrikes)
	}
}

// ---------------------------------------------------------------------------
// Escenario 2: Docker caido -> unknown, sin destruir nada; alerta al 5to ciclo.
// ---------------------------------------------------------------------------

func TestReconcileDockerDownMarksUnknownAndRetainsWorkspaces(t *testing.T) {
	r, repo, orch, audit := newReconcilerHarness()
	repo.workspaces = []*domain.WorkspaceInstance{ws("w1", "c1", domain.WorkspaceStatusRunning)}
	orch.listErr = errors.New("daemon unreachable")

	r.Reconcile(context.Background())

	if len(repo.updates) != 1 || repo.updates[0].status != domain.WorkspaceStatusUnknown {
		t.Fatalf("expected w1 -> unknown, got %+v", repo.updates)
	}
	// La fila sigue existiendo: nadie borro nada.
	if len(repo.workspaces) != 1 {
		t.Fatalf("workspace must survive a docker outage, got %d rows", len(repo.workspaces))
	}
	// 1 de 5 ciclos: todavia no hay alerta.
	if len(audit.logs) != 0 {
		t.Fatalf("expected no alert before the 5th cycle, got %d audit logs", len(audit.logs))
	}
}

func TestReconcileDockerDownAlertsOnceAtThreshold(t *testing.T) {
	r, repo, orch, audit := newReconcilerHarness()
	repo.workspaces = []*domain.WorkspaceInstance{ws("w1", "c1", domain.WorkspaceStatusUnknown)}
	orch.listErr = errors.New("daemon unreachable")

	for i := 0; i < ReconcilerUnknownAlertThreshold; i++ {
		r.Reconcile(context.Background())
	}

	alertCount := 0
	for _, log := range audit.logs {
		if log.Action == "WORKSPACE_RECONCILED_UNKNOWN_PERSISTENT" {
			alertCount++
			if log.ResourceID == nil || *log.ResourceID != "w1" {
				t.Fatalf("expected the alert to reference workspace w1")
			}
		}
	}
	if alertCount != 1 {
		t.Fatalf("expected exactly one persistent alert, got %d", alertCount)
	}

	// El ciclo 6 no vuelve a alertar (ya alertado al cruzar el umbral).
	r.Reconcile(context.Background())
	alertCount = 0
	for _, log := range audit.logs {
		if log.Action == "WORKSPACE_RECONCILED_UNKNOWN_PERSISTENT" {
			alertCount++
		}
	}
	if alertCount != 1 {
		t.Fatalf("expected the alert to fire once, got %d", alertCount)
	}
}

// ---------------------------------------------------------------------------
// Escenario 3: Docker responde y el contenedor ausente -> orphaned.
// ---------------------------------------------------------------------------

func TestReconcileMissingContainerMarksOrphaned(t *testing.T) {
	r, repo, orch, audit := newReconcilerHarness()
	repo.workspaces = []*domain.WorkspaceInstance{ws("w1", "c1", domain.WorkspaceStatusRunning)}
	orch.managed = []string{"c1", "c2"} // c1 existe, c3 no

	// Segundo workspace con contenedor desaparecido:
	repo.workspaces = append(repo.workspaces, ws("w3", "c3", domain.WorkspaceStatusRunning))

	r.Reconcile(context.Background())

	if len(repo.updates) != 1 || repo.updates[0].id != "w3" {
		t.Fatalf("expected only w3 -> orphaned, got %+v", repo.updates)
	}
	if repo.updates[0].status != domain.WorkspaceStatusOrphaned {
		t.Fatalf("expected orphaned status, got %s", repo.updates[0].status)
	}
	if repo.workspaces[1].Status != domain.WorkspaceStatusOrphaned {
		t.Fatalf("repo view of w3 must be orphaned after reconcile")
	}
	found := false
	for _, log := range audit.logs {
		if log.Action == "WORKSPACE_RECONCILED_ORPHANED" && log.ResourceID != nil && *log.ResourceID == "w3" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected an orphaned audit transition for w3, logs=%+v", audit.logs)
	}
}

// ---------------------------------------------------------------------------
// Recuperacion: daemon volvio y el contenedor de un unknown sigue existiendo.
// ---------------------------------------------------------------------------

func TestReconcileRecoversUnknownToRunning(t *testing.T) {
	r, repo, orch, _ := newReconcilerHarness()
	repo.workspaces = []*domain.WorkspaceInstance{ws("w1", "c1", domain.WorkspaceStatusUnknown)}
	orch.managed = []string{"c1"}

	r.Reconcile(context.Background())

	if len(repo.updates) != 1 || repo.updates[0].status != domain.WorkspaceStatusRunning {
		t.Fatalf("expected w1 -> running after recovery, got %+v", repo.updates)
	}
}

// ---------------------------------------------------------------------------
// Contenedor existente pero fuera de marcha.
// ---------------------------------------------------------------------------

func TestReconcileExited139MarksHibernatedAndRemoves(t *testing.T) {
	r, repo, orch, _ := newReconcilerHarness()
	repo.workspaces = []*domain.WorkspaceInstance{ws("w1", "c1", domain.WorkspaceStatusRunning)}
	orch.managed = []string{"c1"}
	orch.metrics["c1"] = &domain.ContainerMetrics{IsRunning: false, ExitCode: 0}

	r.Reconcile(context.Background())

	if len(orch.removed) != 1 || orch.removed[0] != "c1" {
		t.Fatalf("expected the exited container removed, got %v", orch.removed)
	}
	if len(repo.updates) != 1 || repo.updates[0].status != domain.WorkspaceStatusHibernated {
		t.Fatalf("expected w1 -> hibernated, got %+v", repo.updates)
	}
}

func TestReconcileExited137GoesThroughOOMStrike(t *testing.T) {
	r, repo, orch, _ := newReconcilerHarness()
	repo.workspaces = []*domain.WorkspaceInstance{ws("w1", "c1", domain.WorkspaceStatusRunning)}
	orch.managed = []string{"c1"}
	orch.metrics["c1"] = &domain.ContainerMetrics{IsRunning: false, OOMKilled: true, ExitCode: 137}

	r.Reconcile(context.Background())

	if len(repo.oomStrikes) != 1 || repo.oomStrikes[0] != "w1" {
		t.Fatalf("expected a canonical oom strike for w1, got %v", repo.oomStrikes)
	}
	if len(orch.removed) != 0 {
		t.Fatalf("the oom path must NOT remove the container (evidence stays for restart), got %v", orch.removed)
	}
}
