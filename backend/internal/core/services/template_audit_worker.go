package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/client"
	"github.com/shirou/gopsutil/v3/mem"

	"solv-backend/internal/core/domain"
)

const (
	TrivyDockerImage = "aquasec/trivy:0.55.0"
	// La base CVE se expande a más de 1 GB. Guardarla en un directorio temporal
	// la deja en RAM cuando /tmp es tmpfs y empuja al host a swap, así que este
	// es solo el respaldo para procesos sin directorio de caché de usuario
	// (servicios sin HOME). El valor normal sale de resolveTrivyCacheDir.
	DefaultTrivyCacheDir = "/var/lib/solv/trivy-cache"
	// El escáner y su descarga no pueden quedarse con el host. Medio núcleo
	// alcanza para el trabajo y calienta la mitad que un núcleo entero, que en
	// una laptop se nota. El smoke ya usa el mismo techo.
	trivyCPULimit    int64 = 500_000_000
	trivyMemoryLimit int64 = 1 << 30
)

const (
	// Etiquetas de los contenedores auxiliares del worker. El de la descarga de
	// la base CVE no usa AutoRemove porque hay que leer su código de salida; si
	// el proceso muere durante la descarga, la etiqueta permite barrerlo.
	auditContainerLabel  = "io.solv.template-audit"
	trivyDBContainerRole = "trivy-db-download"
)

const (
	// Espera del worker cuando el escáner no está listo (sin base CVE). Un
	// reintento cada minuto gasta diez intentos en la primera hora cuando la red
	// falla; cinco minutos por intento son suficientes para recuperarse.
	trivyDBRetryBase = 5 * time.Minute
	trivyDBRetryMax  = time.Hour
	// Espera por plantilla cuando el fallo es de una imagen concreta.
	templateRetryBase = 10 * time.Minute
	templateRetryMax  = time.Hour
	trivyDBRefresh    = 24 * time.Hour
	// Cadencia del reconciliador. Una cola de auditoría se atiende en minutos:
	// revisarla cada 10 s son 8.640 consultas al día sin ganar nada.
	auditReconcileInterval = 30 * time.Second
)

// errAuditInfrastructure indica que la auditoría se persistió, pero no pudo
// completarse por una causa ajena a la plantilla. El reconciliador la espacia
// en vez de reintentarla en cada ciclo.
var errAuditInfrastructure = errors.New("auditoría no completada por infraestructura")

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
	// retryMu protege solo los mapas de espaciado. EnqueueTemplate lo toma en
	// el camino HTTP y no puede esperar a que termine un escaneo (w.mu).
	retryMu sync.Mutex
	// wakeCh despierta al reconciliador cuando un endpoint pide una auditoría:
	// la capacidad 1 descarta avisos repetidos sin bloquear al emisor.
	wakeCh chan struct{}
	stopCh chan struct{}
	// smokeRunner y trivyRunner permiten probar AuditTemplate sin daemon ni
	// escáner reales. El constructor los enlaza a los métodos del worker.
	smokeRunner func(context.Context, *domain.AdminTemplateReviewItem) (string, string)
	trivyRunner func(context.Context, string) (string, int, int, []byte)

	// dbSyncer, dbReady y now permiten probar el primado y el backoff sin
	// Docker ni reloj real.
	dbSyncer func(context.Context) error
	dbReady  func() bool
	now      func() time.Time
	// javaDBSyncer descarga el índice Java en la misma caché del escaneo.
	javaDBSyncer func(context.Context) error

	nextDBRetry     time.Time
	dbBackoff       time.Duration
	nextDBRefresh   time.Time
	templateRetry   map[string]time.Time
	templateBackoff map[string]time.Duration
}

// resolveTrivyCacheDir decide dónde vive la base CVE. Nunca elige un directorio
// temporal: ahí la base queda en RAM cuando /tmp es tmpfs. La ruta configurada
// manda, luego el directorio de caché del usuario y, si no hay ninguno, el
// respaldo en /var/lib.
func resolveTrivyCacheDir() string {
	if configured := strings.TrimSpace(os.Getenv("TRIVY_CACHE_DIR")); configured != "" {
		return configured
	}
	if userCache, err := os.UserCacheDir(); err == nil && userCache != "" && !isInsideDir(userCache, os.TempDir()) {
		return filepath.Join(userCache, "solv", "trivy")
	}
	return DefaultTrivyCacheDir
}

// isInsideDir informa si path es dir o está dentro de dir.
func isInsideDir(path, dir string) bool {
	cleanPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	cleanDir, err := filepath.Abs(dir)
	if err != nil {
		return false
	}
	if cleanPath == cleanDir {
		return true
	}
	return strings.HasPrefix(cleanPath, cleanDir+string(os.PathSeparator))
}

// prepareTrivyCacheDir deja los directorios de la base CVE escribibles por el
// contenedor (que escribe como root) y por el usuario del servicio (que borra
// una descarga a medias). Sin esto la purga falla con permission denied y el
// espacio queda ocupado hasta el siguiente arranque. Dentro solo hay datos
// públicos de CVE.
func prepareTrivyCacheDir(cacheDir string) {
	//nolint:gosec
	_ = os.MkdirAll(filepath.Join(cacheDir, "trivy", "db"), 0777)
	//nolint:gosec
	_ = os.MkdirAll(filepath.Join(cacheDir, "trivy", "java-db"), 0777)
	//nolint:gosec
	_ = os.Chmod(filepath.Join(cacheDir, "trivy"), 0777)
	//nolint:gosec
	_ = os.Chmod(filepath.Join(cacheDir, "trivy", "db"), 0777)
	//nolint:gosec
	_ = os.Chmod(filepath.Join(cacheDir, "trivy", "java-db"), 0777)
}

func NewTemplateAuditWorker(dockerCli *client.Client, govRepo domain.AdminGovernanceRepository) *TemplateAuditWorker {
	cacheDir := resolveTrivyCacheDir()
	if isInsideDir(cacheDir, os.TempDir()) {
		log.Printf("[TemplateAuditWorker] Aviso: la caché de Trivy está en %s, dentro del directorio temporal; con /tmp en RAM la base CVE ocupa memoria del host", cacheDir)
	}
	// Permisos 0755 requeridos para montaje como /root/.cache en contenedor Trivy (datos publicos de CVE).
	//nolint:gosec
	_ = os.MkdirAll(cacheDir, 0755)
	prepareTrivyCacheDir(cacheDir)

	worker := &TemplateAuditWorker{
		dockerCli: dockerCli,
		govRepo:   govRepo,
		cacheDir:  cacheDir,
		wakeCh:    make(chan struct{}, 1),
		stopCh:    make(chan struct{}),
	}
	worker.smokeRunner = worker.runSmokeTest
	worker.trivyRunner = worker.runTrivyScan
	worker.dbSyncer = worker.UpdateTrivyDB
	worker.dbReady = worker.hasTrivyDB
	worker.javaDBSyncer = worker.updateJavaDB
	worker.now = time.Now
	worker.templateRetry = make(map[string]time.Time)
	worker.templateBackoff = make(map[string]time.Duration)
	return worker
}

// Start arranca los bucles en segundo plano: reconciliador periódico y sincronización diaria de base CVE
func (w *TemplateAuditWorker) Start(ctx context.Context) {
	log.Printf("[TemplateAuditWorker] Iniciando worker de auditoría (cache: %s)...", w.cacheDir)

	// 1. Reconciliador de plantillas pendientes. El escáner se asegura antes de
	// tocar la cola: sin base CVE no hay nada que auditar y las plantillas no
	// deben pagar el reintento de una falla de infraestructura.
	go func() {
		ticker := time.NewTicker(auditReconcileInterval)
		defer ticker.Stop()

		w.pruneStaleDBDownloads(ctx)
		w.ensureScanner(ctx)
		w.reconcilePending(ctx)

		for {
			select {
			case <-ctx.Done():
				return
			case <-w.stopCh:
				return
			case <-w.wakeCh:
				w.ensureScanner(ctx)
				w.reconcilePending(ctx)
			case <-ticker.C:
				w.ensureScanner(ctx)
				w.reconcilePending(ctx)
			}
		}
	}()

	// 2. Tarea programada semanal (7 días) de ciclo de vida EOL (C-01)
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

// EnqueueTemplate adelanta la auditoría de una plantilla: borra el espaciado
// por id acumulado por una falla de infraestructura y despierta al
// reconciliador sin esperar el tick periódico. No bloquea y admite llamadas
// repetidas: el canal descarta el aviso sobrante si ya había uno pendiente.
func (w *TemplateAuditWorker) EnqueueTemplate(templateID string) {
	id := strings.TrimSpace(templateID)
	if id == "" {
		return
	}

	w.retryMu.Lock()
	delete(w.templateRetry, id)
	delete(w.templateBackoff, id)
	w.retryMu.Unlock()

	select {
	case w.wakeCh <- struct{}{}:
	default:
	}
}

// ensureScanner mantiene disponible la base CVE. Si falta, la descarga antes
// de la primera auditoría y reintenta con espera creciente; nunca deja que el
// reconciliador escanee contra un escáner incompleto.
func (w *TemplateAuditWorker) ensureScanner(ctx context.Context) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.ensureScannerLocked(ctx)
}

func (w *TemplateAuditWorker) ensureScannerLocked(ctx context.Context) {
	now := w.now()
	ready := w.dbReady()
	if ready {
		if w.nextDBRefresh.IsZero() {
			w.nextDBRefresh = now.Add(trivyDBRefresh)
			return
		}
		if now.Before(w.nextDBRefresh) {
			return
		}
	} else if now.Before(w.nextDBRetry) {
		return
	}

	log.Println("[TemplateAuditWorker] Asegurando base CVE de Trivy...")
	if err := w.dbSyncer(ctx); err != nil {
		if w.dbBackoff == 0 {
			w.dbBackoff = trivyDBRetryBase
		} else if w.dbBackoff < trivyDBRetryMax {
			w.dbBackoff *= 2
			if w.dbBackoff > trivyDBRetryMax {
				w.dbBackoff = trivyDBRetryMax
			}
		}
		w.nextDBRetry = now.Add(w.dbBackoff)
		w.nextDBRefresh = w.nextDBRetry
		log.Printf("[TemplateAuditWorker] Escáner no disponible: %v. Próximo intento en %s", err, w.dbBackoff)
		return
	}

	w.dbBackoff = 0
	w.nextDBRetry = time.Time{}
	w.nextDBRefresh = now.Add(trivyDBRefresh)
	log.Println("[TemplateAuditWorker] Base CVE lista para auditar")
}

// hasTrivyDB comprueba que la caché contiene una base real y completa. Trivy
// escribe trivy.db mientras descarga, así que el archivo por sí solo no prueba
// nada: metadata.json solo aparece cuando la descarga terminó bien.
func (w *TemplateAuditWorker) hasTrivyDB() bool {
	found := false
	_ = filepath.WalkDir(w.cacheDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if entry.IsDir() || entry.Name() != "trivy.db" {
			return nil
		}
		info, statErr := entry.Info()
		if statErr != nil || info.Size() == 0 {
			return nil
		}
		if isCompleteTrivyDB(filepath.Dir(path)) {
			found = true
			return fs.SkipAll
		}
		return nil
	})
	return found
}

// isCompleteTrivyDB valida el metadata.json que Trivy escribe al finalizar.
func isCompleteTrivyDB(dbDir string) bool {
	raw, err := os.ReadFile(filepath.Join(dbDir, "metadata.json"))
	if err != nil {
		return false
	}
	var meta struct {
		Version int `json:"Version"`
	}
	if err := json.Unmarshal(raw, &meta); err != nil {
		return false
	}
	return meta.Version > 0
}

func (w *TemplateAuditWorker) reconcilePending(ctx context.Context) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.dbReady() {
		return
	}

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
		if w.retryScheduled(item.ID) {
			continue
		}
		select {
		case <-ctx.Done():
			return
		case <-w.stopCh:
			return
		default:
			log.Printf("[TemplateAuditWorker] Auditando plantilla %s (imagen: %s)...", item.Name, item.DockerImage)
			err := w.AuditTemplate(ctx, item)
			switch {
			case errors.Is(err, errAuditInfrastructure):
				delay := w.scheduleRetry(item.ID)
				log.Printf("[TemplateAuditWorker] Plantilla %s sin auditoría por infraestructura; próximo intento en %s", item.ID, delay)
			case err != nil:
				log.Printf("[TemplateAuditWorker] Error auditando plantilla %s: %v", item.ID, err)
			default:
				w.clearRetry(item.ID)
			}
		}
	}
}

// retryScheduled informa si la plantilla todavía espera su próximo intento.
func (w *TemplateAuditWorker) retryScheduled(id string) bool {
	w.retryMu.Lock()
	defer w.retryMu.Unlock()
	next, ok := w.templateRetry[id]
	return ok && w.now().Before(next)
}

// scheduleRetry espacia el próximo intento de una plantilla con falla de
// infraestructura y devuelve la espera aplicada.
func (w *TemplateAuditWorker) scheduleRetry(id string) time.Duration {
	w.retryMu.Lock()
	defer w.retryMu.Unlock()
	delay := w.templateBackoff[id]
	if delay == 0 {
		delay = templateRetryBase
	} else if delay < templateRetryMax {
		delay *= 2
		if delay > templateRetryMax {
			delay = templateRetryMax
		}
	}
	w.templateBackoff[id] = delay
	w.templateRetry[id] = w.now().Add(delay)
	return delay
}

// clearRetry olvida el espaciado de una plantilla que ya se auditó.
func (w *TemplateAuditWorker) clearRetry(id string) {
	w.retryMu.Lock()
	defer w.retryMu.Unlock()
	delete(w.templateBackoff, id)
	delete(w.templateRetry, id)
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
			// Una plantilla publicada conserva su estado aunque la cuota declarada
			// ya no quepa: retirarla dejaría sin laboratorio a los cursos en
			// curso. El resultado queda registrado para que una persona decida.
			finalStatus := "RECHAZADA"
			if isApprovedTemplateStatus(item.Status) {
				finalStatus = item.Status
			}
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
				rejectionReasonFor(finalStatus, smokeOutput),
			)
			return err
		}
	}

	// Paso 1: Smoke Test efímero de herramientas declaradas (B-02). Un
	// re-escaneo que solo espera Trivy reaprovecha el smoke ya aprobado: el
	// contenedor efímero no aporta información nueva y cuesta arranque y RAM.
	if w.smokeRunner == nil {
		w.smokeRunner = w.runSmokeTest
	}
	smokeStatus, smokeOutput := "", ""
	if keepsPassedSmoke(item) {
		smokeStatus, smokeOutput = item.SmokeTestStatus, item.SmokeTestOutput
	} else {
		smokeStatus, smokeOutput = w.smokeRunner(ctx, item)
	}

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
		rejectionReasonFor(finalStatus, failureReason),
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

	infrastructureFailure := strings.EqualFold(smokeStatus, auditStatusError) ||
		(!strings.EqualFold(smokeStatus, auditStatusFailed) &&
			(strings.EqualFold(secStatus, auditStatusError) ||
				(strings.EqualFold(secStatus, auditStatusFailed) && cveCritical == 0)))
	if infrastructureFailure {
		return errAuditInfrastructure
	}
	return nil
}

// resolveAuditOutcome decide el estado final de una auditoría. Devuelve el
// estado que se persiste y el motivo que verá el administrador.
//
// La auditoría es una puerta de admisión, no un interruptor del catálogo. Un
// fallo de infraestructura no es un fallo de la imagen: se conserva el estado
// y la cola reintenta. Un smoke fallido o CVE críticos rechazan solo lo que
// todavía no se publicó; una plantilla ya aprobada conserva su estado para no
// tumbar laboratorios de cursos en curso, y el hallazgo queda registrado en las
// columnas de auditoría para que una persona decida retirarla.
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
		if isAwaitingAuditStatus(current) {
			return "APROBADA", ""
		}
		// Aprobada sigue aprobada; pausada, suspendida o rechazada espera una
		// decisión humana.
		return currentStatus, ""
	}

	reason := auditFailureReason(smokeFailed, smokeOutput, securityFailed, cveCritical)

	// Publicada y en revisión manual: el estado lo decide una persona.
	if isApprovedTemplateStatus(current) || isHumanReviewStatus(current) {
		return currentStatus, reason
	}
	return "RECHAZADA", reason
}

// auditFailureReason arma el motivo legible de un resultado en rojo.
func auditFailureReason(smokeFailed bool, smokeOutput string, securityFailed bool, cveCritical int) string {
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
	return strings.Join(reasons, ". ")
}

// isAwaitingAuditStatus indica que la plantilla aún no se publicó y la
// auditoría decide su admisión.
func isAwaitingAuditStatus(status string) bool {
	current := strings.ToUpper(strings.TrimSpace(status))
	return current == "PENDIENTE_AUDITORIA" || current == "PENDING"
}

// isHumanReviewStatus agrupa los estados que solo cambia una persona.
func isHumanReviewStatus(status string) bool {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "PAUSADA", "PAUSED", "SUSPENDIDA", "SUSPENDED", "RECHAZADA", "REJECTED":
		return true
	default:
		return false
	}
}

// keepsPassedSmoke indica que el smoke ya está aprobado y solo falta el
// escaneo. Repetir el contenedor efímero en ese caso no aporta información:
// se conserva el resultado guardado y se ejecuta Trivy directamente.
func keepsPassedSmoke(item *domain.AdminTemplateReviewItem) bool {
	if item == nil || !strings.EqualFold(item.SmokeTestStatus, auditStatusPassed) {
		return false
	}
	security := strings.ToLower(strings.TrimSpace(item.SecurityAuditStatus))
	return security == "pending" || security == auditStatusError
}

// rejectionReasonFor no pisa el motivo de una plantilla publicada: un
// re-escaneo que encuentra hallazgos se registra en sus propias columnas y el
// motivo de rechazo queda para cuando una persona retira la plantilla.
func rejectionReasonFor(finalStatus, reason string) string {
	if isApprovedTemplateStatus(finalStatus) {
		return ""
	}
	return reason
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
