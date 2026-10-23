package httpdelivery

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

type MetricsHandler struct {
	workspaceRepo   domain.WorkspaceRepository
	hostMonitor     domain.HostMonitor
	zombieCollector *services.ZombieCollectorWorker
}

func NewMetricsHandler(workspaceRepo domain.WorkspaceRepository, hostMonitor domain.HostMonitor, zombieCollector *services.ZombieCollectorWorker) *MetricsHandler {
	return &MetricsHandler{
		workspaceRepo:   workspaceRepo,
		hostMonitor:     hostMonitor,
		zombieCollector: zombieCollector,
	}
}

func (h *MetricsHandler) HandleMetrics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Obtener workspaces activos
	activeWorkspaces, err := h.workspaceRepo.GetActiveWorkspaces(ctx)
	activeCount := 0
	if err == nil {
		activeCount = len(activeWorkspaces)
	}

	// 2. Obtener métricas del Host
	_, availMB, err := h.hostMonitor.GetHostMemoryStats()
	availBytes := uint64(0)
	if err == nil {
		availBytes = availMB * 1024 * 1024
	}

	// 3. Obtener OOM_GUARD_MB
	oomGuardMB := uint64(1408)
	if envGuard := os.Getenv("OOM_GUARD_MB"); envGuard != "" {
		if val, parseErr := strconv.ParseUint(envGuard, 10, 64); parseErr == nil {
			oomGuardMB = val
		}
	}
	oomGuardBytes := oomGuardMB * 1024 * 1024

	// 4. Obtener conteo de contenedores huérfanos reclamados
	reclaimedCount := uint64(0)
	if h.zombieCollector != nil {
		reclaimedCount = h.zombieCollector.GetReclaimedCount()
	}

	// 5. Obtener días hasta la expiración del certificado TLS (vía TLS dial o fallback)
	certExpiryDays := getCertExpiryDays()

	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	_, _ = fmt.Fprintf(w, "# HELP solv_active_workspaces_total Total number of active running/pending student workspaces\n")
	_, _ = fmt.Fprintf(w, "# TYPE solv_active_workspaces_total gauge\n")
	_, _ = fmt.Fprintf(w, "solv_active_workspaces_total %d\n\n", activeCount)

	_, _ = fmt.Fprintf(w, "# HELP solv_host_available_memory_bytes Physical host available RAM in bytes\n")
	_, _ = fmt.Fprintf(w, "# TYPE solv_host_available_memory_bytes gauge\n")
	_, _ = fmt.Fprintf(w, "solv_host_available_memory_bytes %d\n\n", availBytes)

	_, _ = fmt.Fprintf(w, "# HELP solv_host_oom_guard_bytes Calibrated Node Allocatable OOM Guard threshold in bytes\n")
	_, _ = fmt.Fprintf(w, "# TYPE solv_host_oom_guard_bytes gauge\n")
	_, _ = fmt.Fprintf(w, "solv_host_oom_guard_bytes %d\n\n", oomGuardBytes)

	_, _ = fmt.Fprintf(w, "# HELP solv_orphan_containers_reclaimed_total Total number of orphan/zombie containers reclaimed by collector\n")
	_, _ = fmt.Fprintf(w, "# TYPE solv_orphan_containers_reclaimed_total counter\n")
	_, _ = fmt.Fprintf(w, "solv_orphan_containers_reclaimed_total %d\n\n", reclaimedCount)

	_, _ = fmt.Fprintf(w, "# HELP solv_cert_expiry_days Remaining validity of wildcard TLS certificate in days\n")
	_, _ = fmt.Fprintf(w, "# TYPE solv_cert_expiry_days gauge\n")
	_, _ = fmt.Fprintf(w, "solv_cert_expiry_days %.2f\n", certExpiryDays)
}

func getCertExpiryDays() float64 {
	dialer := &net.Dialer{Timeout: 2 * time.Second}
	tlsDialer := &tls.Dialer{
		NetDialer: dialer,
		Config: &tls.Config{
			// Conexion exclusiva a 127.0.0.1:443 para inspeccionar vigencia de certificado local sin trafico de datos.
			//nolint:gosec
			InsecureSkipVerify: true,
		},
	}
	conn, err := tlsDialer.DialContext(context.Background(), "tcp", "127.0.0.1:443")
	if err == nil {
		defer func() { _ = conn.Close() }()
		if tlsConn, ok := conn.(*tls.Conn); ok {
			certs := tlsConn.ConnectionState().PeerCertificates
			if len(certs) > 0 {
				days := time.Until(certs[0].NotAfter).Hours() / 24.0
				if days > 0 {
					return days
				}
			}
		}
	}
	// Fallback por defecto si aún no hay cert montado
	return 90.0
}
