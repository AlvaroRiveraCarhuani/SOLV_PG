package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/client"
	"github.com/shirou/gopsutil/v3/mem"

	"solv-backend/internal/core/domain"
)

const (
	TrivyDockerImage     = "aquasec/trivy:0.55.0"
	DefaultTrivyCacheDir = "/tmp/solv-trivy-cache"
)

const (
	auditStatusPassed  = "passed"
	auditStatusFailed  = "failed"
	auditStatusSkipped = "skipped"
	// auditStatusError marca que no se pudo completar la comprobación por una
	// causa de infraestructura (Docker, red o el propio escáner). No es un
	// veredicto sobre la imagen: la plantilla conserva su estado y se reintenta.
	auditStatusError = "error"
)

// trivyReport representa la estructura resumida devuelta por Trivy en formato JSON
type trivyReport struct {
	Results []struct {
		Target          string `json:"Target"`
		Class           string `json:"Class"`
		Type            string `json:"Type"`
		Vulnerabilities []struct {
			VulnerabilityID  string `json:"VulnerabilityID"`
			PkgName          string `json:"PkgName"`
			InstalledVersion string `json:"InstalledVersion"`
			FixedVersion     string `json:"FixedVersion"`
			Severity         string `json:"Severity"`
			Title            string `json:"Title"`
			PrimaryURL       string `json:"PrimaryURL"`
		} `json:"Vulnerabilities"`
	} `json:"Results"`
}

type TemplateAuditWorker struct {
	dockerCli *client.Client
	govRepo   domain.AdminGovernanceRepository
	cacheDir  string
	mu        sync.Mutex
	stopCh    chan struct{}
	// smokeRunner y trivyRunner permiten probar AuditTemplate sin daemon ni
	// escáner reales. El constructor los enlaza a los métodos del worker.
	smokeRunner func(context.Context, *domain.AdminTemplateReviewItem) (string, string)
	trivyRunner func(context.Context, string) (string, int, int, []byte)
}

func NewTemplateAuditWorker(dockerCli *client.Client, govRepo domain.AdminGovernanceRepository) *TemplateAuditWorker {
	cacheDir := os.Getenv("TRIVY_CACHE_DIR")
	if cacheDir == "" {
		cacheDir = DefaultTrivyCacheDir
	}
	// Permisos 0755 requeridos para montaje como /root/.cache en contenedor Trivy (datos publicos de CVE).
	//nolint:gosec
	_ = os.MkdirAll(cacheDir, 0755)

	worker := &TemplateAuditWorker{
		dockerCli: dockerCli,
		govRepo:   govRepo,
		cacheDir:  cacheDir,
		stopCh:    make(chan struct{}),
	}
	worker.smokeRunner = worker.runSmokeTest
	worker.trivyRunner = worker.runTrivyScan
	return worker
}

// Start arranca los bucles en segundo plano: reconciliador periódico y sincronización diaria de base CVE
func (w *TemplateAuditWorker) Start(ctx context.Context) {
	log.Printf("[TemplateAuditWorker] Iniciando worker de auditoría (cache: %s)...", w.cacheDir)

	// 1. Reconciliador de plantillas pendientes
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		// Ejecución inicial inmediata
		w.reconcilePending(ctx)

		for {
			select {
			case <-ctx.Done():
				return
			case <-w.stopCh:
				return
			case <-ticker.C:
				w.reconcilePending(ctx)
			}
		}
	}()

	// 2. Cron diario de descarga de base de datos CVE (B-04)
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-w.stopCh:
				return
			case <-ticker.C:
				log.Println("[TemplateAuditWorker] Ejecutando sincronización diaria de base CVE...")
				if err := w.UpdateTrivyDB(ctx); err != nil {
					log.Printf("[TemplateAuditWorker] Error actualizando base CVE: %v", err)
				}
			}
		}
	}()

	// 3. Tarea programada semanal (7 días) de ciclo de vida EOL (C-01)
	go func() {
		ticker := time.NewTicker(7 * 24 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-w.stopCh:
				return
			case <-ticker.C:
				log.Println("[TemplateAuditWorker] Ejecutando sincronización periódica de ciclo de vida EOL...")
				w.recheckAllEOL(ctx)
			}
		}
	}()
}

func (w *TemplateAuditWorker) Stop() {
	close(w.stopCh)
}

func (w *TemplateAuditWorker) reconcilePending(ctx context.Context) {
	w.mu.Lock()
	defer w.mu.Unlock()

	templates, err := w.govRepo.ListPendingAuditTemplates(ctx)
	if err != nil {
		log.Printf("[TemplateAuditWorker] Error consultando plantillas pendientes: %v", err)
		return
	}

	for _, item := range templates {
		// Pausada, suspendida y rechazada las decide una persona. No se gastan
		// contenedores ni escaneos mientras la plantilla esté fuera del catálogo
		// por decisión humana; al reactivarla, la cola la vuelve a tomar.
		switch strings.ToUpper(strings.TrimSpace(item.Status)) {
		case "PAUSADA", "PAUSED", "SUSPENDIDA", "SUSPENDED", "RECHAZADA", "REJECTED":
			log.Printf("[TemplateAuditWorker] Plantilla %s en estado %s: auditoría omitida", item.ID, item.Status)
			continue
		}
		select {
		case <-ctx.Done():
			return
		case <-w.stopCh:
			return
		default:
			log.Printf("[TemplateAuditWorker] Auditando plantilla %s (imagen: %s)...", item.Name, item.DockerImage)
			if err := w.AuditTemplate(ctx, item); err != nil {
				log.Printf("[TemplateAuditWorker] Error auditando plantilla %s: %v", item.ID, err)
			}
		}
	}
}

// AuditTemplate ejecuta el smoke test de herramientas y el escaneo de vulnerabilidades Trivy
func (w *TemplateAuditWorker) AuditTemplate(ctx context.Context, item *domain.AdminTemplateReviewItem) error {
	// Pre-chequeo: Techo estructural de RAM contra el host (falla rápido antes del contenedor efímero)
	if item.BaseRamMB > 0 {
		totalHostMB := 8192
		if v, err := mem.VirtualMemoryWithContext(ctx); err == nil && v != nil {
			totalHostMB = domain.Uint64ToInt(v.Total / (1024 * 1024))
		}
		maxAllowedRAM := domain.CalculateHostMaxAllowedRAM(totalHostMB)
		if err := domain.ValidateRamAgainstHost(item.BaseRamMB, maxAllowedRAM); err != nil {
			smokeStatus := "failed"
			smokeOutput := fmt.Sprintf("ram_exceeds_host: memoria declarada (%d MB) supera el techo permitido por el host (%d MB)", item.BaseRamMB, maxAllowedRAM)
			secStatus := "skipped"
			reportJSON := json.RawMessage(`{"error": "ram_exceeds_host"}`)
			finalStatus := "RECHAZADA"
			_ = w.govRepo.UpdateAuditResults(
				ctx,
				item.ID,
				smokeStatus,
				smokeOutput,
				secStatus,
				0,
				0,
				reportJSON,
				finalStatus,
				smokeOutput,
			)
			return err
		}
	}

	// Paso 1: Smoke Test efímero de herramientas declaradas (B-02)
	if w.smokeRunner == nil {
		w.smokeRunner = w.runSmokeTest
	}
	smokeStatus, smokeOutput := w.smokeRunner(ctx, item)

	// Paso 2: Escaneo de vulnerabilidades CVE con Trivy (B-03). Si el smoke no
	// pudo ni preparar la imagen, no se lanza el escáner: no hay nada que
	// escanear y el fallo es de infraestructura, no de la imagen.
	var (
		secStatus   string
		cveCritical int
		cveHigh     int
		reportJSON  []byte
	)
	if strings.EqualFold(smokeStatus, auditStatusError) {
		secStatus = auditStatusSkipped
		reportJSON = []byte(`{"error":"smoke_infrastructure"}`)
	} else {
		if w.trivyRunner == nil {
			w.trivyRunner = w.runTrivyScan
		}
		secStatus, cveCritical, cveHigh, reportJSON = w.trivyRunner(ctx, item.DockerImage)
	}

	// Paso 3: Resolución de máquina de estados. Un smoke fallido o CVE críticos
	// retiran la plantilla del catálogo, incluso si ya estaba aprobada. Un fallo
	// de infraestructura conserva el estado y se reintenta. Los estados que
	// decide una persona (pausada, suspendida, rechazada) no se auto-reactivan
	// con una auditoría.
	finalStatus, failureReason := resolveAuditOutcome(item.Status, smokeStatus, smokeOutput, secStatus, cveCritical)
	if secStatus == "passed" && cveHigh > 0 {
		secStatus = "warning"
	}

	// Paso 4: Persistir resultados en la base de datos
	err := w.govRepo.UpdateAuditResults(
		ctx,
		item.ID,
		smokeStatus,
		smokeOutput,
		secStatus,
		cveCritical,
		cveHigh,
		reportJSON,
		finalStatus,
		failureReason,
	)
	if err != nil {
		return fmt.Errorf("error persistiendo auditoría: %w", err)
	}

	log.Printf("[TemplateAuditWorker] Plantilla %s auditada: Estado=%s, Smoke=%s, CVE_Crit=%d, CVE_High=%d",
		item.Name, finalStatus, smokeStatus, cveCritical, cveHigh)

	// Paso 5: Chequeo de ciclo de vida EOL (C-01, C-02)
	eolStatus, eolDate, eolMessage := checkEOL(item.DockerImage)
	if err := w.govRepo.UpdateEOLStatus(ctx, item.ID, eolStatus, eolDate, eolMessage); err != nil {
		log.Printf("[TemplateAuditWorker] Error actualizando estado EOL para plantilla %s: %v", item.ID, err)
	}

	return nil
}

// resolveAuditOutcome decide el estado final de una auditoría. Devuelve el
// estado que se persiste y el motivo que verá el administrador.
//
// Un fallo de infraestructura no es un fallo de la imagen: se conserva el
// estado actual y la cola reintenta. Solo un smoke fallido o CVE críticos
// retiran la plantilla del catálogo.
func resolveAuditOutcome(currentStatus, smokeStatus, smokeOutput, secStatus string, cveCritical int) (string, string) {
	current := strings.ToUpper(strings.TrimSpace(currentStatus))
	smokeFailed := strings.EqualFold(smokeStatus, auditStatusFailed)
	smokeError := strings.EqualFold(smokeStatus, auditStatusError)
	securityFailed := strings.EqualFold(secStatus, auditStatusFailed) && cveCritical > 0
	securityError := strings.EqualFold(secStatus, auditStatusError) ||
		(strings.EqualFold(secStatus, auditStatusFailed) && cveCritical == 0)

	if smokeError || (securityError && !smokeFailed) {
		return currentStatus, infrastructureAuditReason(smokeStatus, smokeOutput, secStatus)
	}

	if !smokeFailed && !securityFailed {
		switch current {
		case "PENDIENTE_AUDITORIA", "PENDING":
			return "APROBADA", ""
		default:
			// Aprobada sigue aprobada; pausada, suspendida o rechazada espera
			// una decisión humana.
			return currentStatus, ""
		}
	}

	reasons := make([]string, 0, 2)
	if smokeFailed {
		reason := strings.TrimSpace(smokeOutput)
		if reason == "" {
			reason = "La comprobación de arranque no superó las herramientas declaradas"
		}
		reasons = append(reasons, reason)
	}
	if securityFailed {
		label := "vulnerabilidades críticas"
		if cveCritical == 1 {
			label = "vulnerabilidad crítica"
		}
		reasons = append(reasons, fmt.Sprintf("Trivy detectó %d %s", cveCritical, label))
	}
	reason := strings.Join(reasons, ". ")

	switch current {
	case "PAUSADA", "PAUSED", "SUSPENDIDA", "SUSPENDED", "RECHAZADA", "REJECTED":
		return currentStatus, reason
	default:
		return "RECHAZADA", reason
	}
}

func infrastructureAuditReason(smokeStatus, smokeOutput, secStatus string) string {
	if strings.EqualFold(smokeStatus, auditStatusError) {
		detail := strings.TrimSpace(smokeOutput)
		if detail == "" {
			detail = "la comprobación de arranque no se pudo ejecutar"
		}
		return "Auditoría no completada: " + detail + ". Se reintentará."
	}
	if strings.EqualFold(secStatus, auditStatusError) || strings.EqualFold(secStatus, auditStatusFailed) {
		return "Auditoría no completada: el escaneo de seguridad no produjo un informe válido. Se reintentará."
	}
	return "Auditoría no completada por un fallo de infraestructura. Se reintentará."
}

type EOLResult struct {
	Status  string `json:"status"`
	Date    string `json:"date"`
	Message string `json:"message"`
}
