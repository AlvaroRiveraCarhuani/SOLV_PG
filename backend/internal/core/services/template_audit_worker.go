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
}

func NewTemplateAuditWorker(dockerCli *client.Client, govRepo domain.AdminGovernanceRepository) *TemplateAuditWorker {
	cacheDir := os.Getenv("TRIVY_CACHE_DIR")
	if cacheDir == "" {
		cacheDir = DefaultTrivyCacheDir
	}
	// Permisos 0755 requeridos para montaje como /root/.cache en contenedor Trivy (datos publicos de CVE).
	//nolint:gosec
	_ = os.MkdirAll(cacheDir, 0755)

	return &TemplateAuditWorker{
		dockerCli: dockerCli,
		govRepo:   govRepo,
		cacheDir:  cacheDir,
		stopCh:    make(chan struct{}),
	}
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
			)
			return err
		}
	}

	// Paso 1: Smoke Test efímero de herramientas declaradas (B-02)
	smokeStatus, smokeOutput := w.runSmokeTest(ctx, item)

	// Paso 2: Escaneo de vulnerabilidades CVE con Trivy (B-03)
	secStatus, cveCritical, cveHigh, reportJSON := w.runTrivyScan(ctx, item.DockerImage)

	// Paso 3: Resolución de máquina de estados
	// Solo se rechaza automáticamente cuando falla el smoke test funcional (salvo que ya haya sido aprobada desde el Constructor).
	// Los hallazgos de Trivy (CVEs del sistema base) se registran como telemetría ("warning" / conteos) sin auto-rechazar imágenes oficiales.
	alreadyApproved := strings.EqualFold(item.Status, "APROBADA") || strings.EqualFold(item.Status, "approved")
	finalStatus := "APROBADA"
	if smokeStatus == "failed" && !alreadyApproved {
		finalStatus = "RECHAZADA"
	}
	if cveCritical > 0 && secStatus != "failed" {
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

type EOLResult struct {
	Status  string `json:"status"`
	Date    string `json:"date"`
	Message string `json:"message"`
}
