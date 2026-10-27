package httpdelivery

import (
	"context"
	"log"
	"strings"
	"time"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

type LoadSnapshot struct {
	Timestamp        string  `json:"timestamp"`
	RAMPercent       float64 `json:"ram_percent"`
	RAMUsedGB        float64 `json:"ram_used_gb"`
	CPUPercent       float64 `json:"cpu_percent"`
	ActiveContainers int     `json:"active_containers"`
}

type AdminHandler struct {
	auditRepo     domain.AuditLogRepository
	tenantRepo    domain.TenantRepository
	workspaceRepo domain.WorkspaceRepository
	subjectRepo   domain.SubjectRepository
	hostMonitor   domain.HostMonitor
	orchestrator  domain.WorkspaceOrchestrator
	studentDir    StudentDirectorySource

	policiesService *services.ServerPoliciesService
	telemetry       domain.HostTelemetry
	metricsSamples  domain.HostMetricsSampleStore
}

// SetPoliciesService inyecta la fuente unica de politicas del tenant para que
// el panel de salud no reimplemente el parseo y muestre valores distintos de
// los que aplica la admision de laboratorios.
func (h *AdminHandler) SetPoliciesService(svc *services.ServerPoliciesService) {
	h.policiesService = svc
}

// SetTelemetry inyecta la fuente unica de telemetria del host. Sin ella, el
// handler degrada: usa el monitor de admision como fallback de capacidad y
// deja los datos de hardware y Docker nulos.
func (h *AdminHandler) SetTelemetry(t domain.HostTelemetry) {
	h.telemetry = t
}

// SetMetricsSampleStore inyecta el historial persistido del host. Sin el, el
// panel degrada a lista vacia (el grafico muestra su estado "Recopilando
// telemetria") en vez de inventar puntos: D1 prohibe el sembrado sintetico.
func (h *AdminHandler) SetMetricsSampleStore(store domain.HostMetricsSampleStore) {
	h.metricsSamples = store
}

// StudentDirectorySource resuelve identidades reales de estudiantes (nombre y
// correo) para que las vistas de administración no expongan UUIDs internos.
type StudentDirectorySource interface {
	ListStudentsDirectory(ctx context.Context, tenantID, search, subjectID, status, periodID string) ([]*domain.AdminStudentDirectoryItem, error)
}

// workspaceStatusCounter cuenta workspaces por estado sin depender del listado
// de activos. Se declara en el consumidor a propósito: solo el panel de salud
// necesita el desglose completo por estado.
type workspaceStatusCounter interface {
	CountWorkspacesByStatus(ctx context.Context, tenantID string) (map[string]int, error)
}

func (h *AdminHandler) SetStudentDirectory(src StudentDirectorySource) {
	h.studentDir = src
}

func (h *AdminHandler) SetOrchestrator(orch domain.WorkspaceOrchestrator) {
	h.orchestrator = orch
}

func NewAdminHandler(
	auditRepo domain.AuditLogRepository,
	tenantRepo domain.TenantRepository,
	workspaceRepo domain.WorkspaceRepository,
	subjectRepo domain.SubjectRepository,
	hostMonitor domain.HostMonitor,
) *AdminHandler {
	return &AdminHandler{
		auditRepo:     auditRepo,
		tenantRepo:    tenantRepo,
		workspaceRepo: workspaceRepo,
		subjectRepo:   subjectRepo,
		hostMonitor:   hostMonitor,
	}
}

// getHistorySnapshots lee los ultimos samples reales persistidos por el worker
// de telemetria (D1). El parametro window acepta 60m | 24h (default 60m) y se
// acota a la retencion de 7 dias: preguntar por mas historia que la guardada no
// tiene sentido. Las marcas viajan en RFC3339 para que el grafico calcule los
// deltas reales en vez de inventar un "59 - index".
func (h *AdminHandler) getHistorySnapshots(ctx context.Context, windowParam, tenantID string) []LoadSnapshot {
	if tenantID == "" {
		tenantID = domain.DefaultTenantID
	}
	window, limit := parseHistoryWindow(windowParam)

	if h.metricsSamples == nil {
		return []LoadSnapshot{}
	}

	samples, err := h.metricsSamples.ListSince(ctx, tenantID, time.Now().Add(-window), limit)
	if err != nil {
		log.Printf("[AdminHealth] Warning: no se pudieron leer los samples de telemetria: %v", err)
		return []LoadSnapshot{}
	}

	snapshots := make([]LoadSnapshot, 0, len(samples))
	for _, s := range samples {
		snapshots = append(snapshots, LoadSnapshot{
			Timestamp:        s.SampledAt.Format(time.RFC3339),
			RAMPercent:       s.RAMPercent,
			RAMUsedGB:        s.RAMUsedGB,
			CPUPercent:       s.CPUPercent,
			ActiveContainers: s.ActiveContainers,
		})
	}
	return snapshots
}

// parseHistoryWindow interpreta el parametro ?window= del panel. Acepta
// duraciones Go (60m, 24h); ante valores invalidos o fuera de rango nunca
// devuelve una ventana mayor a la retencion de 7 dias.
func parseHistoryWindow(param string) (time.Duration, int) {
	defaultWindow := 60 * time.Minute
	defaultLimit := 61

	w := strings.TrimSpace(param)
	if w == "" {
		return defaultWindow, defaultLimit
	}

	d, err := parseWindowDuration(w)
	if err != nil || d <= 0 {
		return defaultWindow, defaultLimit
	}
	if d > domain.HostMetricsHistoryWindowMax {
		d = domain.HostMetricsHistoryWindowMax
	}

	// Tope generoso: un sample por minuto mas la fila del limite exacto. 24h de
	// muestreo real son 1440 filas; el tope nunca se alcanza con la frecuencia
	// del worker.
	limit := int(d.Minutes()) + 1
	return d, limit
}

// parseWindowDuration interpreta la ventana del panel. Acepta las duraciones de
// time.ParseDuration (60m, 24h, 720h) y además el sufijo de días (30d, 7d) que
// se usa en operaciones manuales sin entrar en conflicto con el Go standard.
func parseWindowDuration(raw string) (time.Duration, error) {
	lower := strings.ToLower(strings.TrimSpace(raw))
	if strings.HasSuffix(lower, "d") {
		daysStr := strings.TrimSuffix(lower, "d")
		days, err := time.ParseDuration(daysStr + "h")
		if err != nil {
			return 0, err
		}
		return days * 24, nil
	}
	return time.ParseDuration(lower)
}
