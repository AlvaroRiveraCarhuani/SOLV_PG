package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/errdefs"
	"github.com/docker/docker/pkg/stdcopy"

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
	// Paso 1: Smoke Test efímero de herramientas declaradas (B-02)
	smokeStatus, smokeOutput := w.runSmokeTest(ctx, item)

	// Paso 2: Escaneo de vulnerabilidades CVE con Trivy (B-03)
	secStatus, cveCritical, cveHigh, reportJSON := w.runTrivyScan(ctx, item.DockerImage)

	// Paso 3: Resolución de máquina de estados
	// Si hay vulnerabilidades críticas o falló el smoke test -> RECHAZADA
	finalStatus := "APROBADA"
	if cveCritical > 0 || smokeStatus == "failed" || secStatus == "failed" {
		finalStatus = "RECHAZADA"
		if secStatus != "failed" && cveCritical > 0 {
			secStatus = "failed"
		}
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

// runSmokeTest ejecuta un contenedor efímero aislado sin red ni permisos para chequear binarios declarados
func (w *TemplateAuditWorker) runSmokeTest(ctx context.Context, item *domain.AdminTemplateReviewItem) (string, string) {
	if len(item.ToolsDeclared) == 0 {
		return "skipped", "No se declararon herramientas requeridas (prueba no aplicable)"
	}

	// Construir script de verificación para /bin/sh o PATH
	var checks []string
	for _, tool := range item.ToolsDeclared {
		cleanTool := strings.TrimSpace(tool)
		if cleanTool == "" {
			continue
		}
		// Sanitizar nombre de herramienta para evitar inyección en el script del contenedor
		cleanTool = strings.ReplaceAll(cleanTool, ";", "")
		cleanTool = strings.ReplaceAll(cleanTool, "&", "")
		cleanTool = strings.ReplaceAll(cleanTool, "|", "")
		cleanTool = strings.ReplaceAll(cleanTool, "`", "")
		cleanTool = strings.ReplaceAll(cleanTool, "$", "")
		cleanTool = strings.TrimSpace(cleanTool)
		if cleanTool != "" {
			checks = append(checks, fmt.Sprintf(`command -v "%s" >/dev/null 2>&1 || which "%s" >/dev/null 2>&1 || missing="$missing %s"`, cleanTool, cleanTool, cleanTool))
		}
	}

	if len(checks) == 0 {
		return "skipped", "No se declararon herramientas requeridas válidas (prueba no aplicable)"
	}

	script := fmt.Sprintf(`missing=""; %s; if [ -n "$missing" ]; then echo "MISSING:$missing"; exit 1; else echo "ALL_FOUND"; exit 0; fi`, strings.Join(checks, "; "))

	testCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	hostConfig := &container.HostConfig{
		NetworkMode:    "none",
		ReadonlyRootfs: true,
		Resources: container.Resources{
			Memory:   256 * 1024 * 1024, // 256 MB
			NanoCPUs: 500000000,          // 0.5 CPU
		},
		AutoRemove: false,
	}

	containerConfig := &container.Config{
		Image:        item.DockerImage,
		Cmd:          []string{"/bin/sh", "-c", script},
		AttachStdout: true,
		AttachStderr: true,
	}

	resp, err := w.dockerCli.ContainerCreate(testCtx, containerConfig, hostConfig, nil, nil, "")
	if err != nil {
		return "failed", fmt.Sprintf("Error creando contenedor efímero: %v", err)
	}
	containerID := resp.ID
	defer func() {
		_ = w.dockerCli.ContainerRemove(context.Background(), containerID, container.RemoveOptions{Force: true})
	}()

	if err := w.dockerCli.ContainerStart(testCtx, containerID, container.StartOptions{}); err != nil {
		return "failed", fmt.Sprintf("Error arrancando contenedor efímero: %v", err)
	}

	statusCh, errCh := w.dockerCli.ContainerWait(testCtx, containerID, container.WaitConditionNotRunning)
	var exitCode int64
	select {
	case waitErr := <-errCh:
		if waitErr != nil {
			return "failed", fmt.Sprintf("Error esperando contenedor efímero: %v", waitErr)
		}
	case res := <-statusCh:
		exitCode = res.StatusCode
	case <-testCtx.Done():
		return "failed", "Tiempo de espera agotado al verificar herramientas (timeout 20s)"
	}

	// Leer salida
	logsReader, err := w.dockerCli.ContainerLogs(testCtx, containerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	})
	if err != nil {
		if exitCode == 0 {
			return "passed", "Todas las herramientas declaradas están instaladas"
		}
		return "failed", "Fallo al verificar herramientas declaradas"
	}
	defer logsReader.Close()

	var stdoutBuf, stderrBuf bytes.Buffer
	_, _ = stdcopy.StdCopy(&stdoutBuf, &stderrBuf, logsReader)
	outputStr := strings.TrimSpace(stdoutBuf.String())

	if exitCode == 0 {
		return "passed", "Todas las herramientas declaradas están disponibles y ejecutables en el PATH"
	}

	if strings.Contains(outputStr, "MISSING:") {
		parts := strings.Split(outputStr, "MISSING:")
		missingList := strings.TrimSpace(parts[1])
		return "failed", fmt.Sprintf("Faltan las siguientes herramientas declaradas en la imagen: %s", missingList)
	}

	return "failed", fmt.Sprintf("La imagen no contiene las herramientas requeridas o no soporta /bin/sh (exit %d)", exitCode)
}

// runTrivyScan ejecuta Trivy escaneando la imagen por CVEs de severidad CRITICAL y HIGH
func (w *TemplateAuditWorker) runTrivyScan(ctx context.Context, dockerImage string) (string, int, int, []byte) {
	// Verificar o descargar imagen de Trivy si no existe
	_, _, err := w.dockerCli.ImageInspectWithRaw(ctx, TrivyDockerImage)
	if err != nil && errdefs.IsNotFound(err) {
		log.Printf("[TemplateAuditWorker] Descargando imagen de auditoría %s...", TrivyDockerImage)
		out, pullErr := w.dockerCli.ImagePull(ctx, TrivyDockerImage, image.PullOptions{})
		if pullErr != nil {
			log.Printf("[TemplateAuditWorker] Error descargando Trivy: %v", pullErr)
			return "failed", 0, 0, []byte(fmt.Sprintf(`{"error": "no se pudo descargar imagen de auditoría: %s"}`, pullErr.Error()))
		}
		defer out.Close()
		_, _ = io.Copy(io.Discard, out)
	}

	scanCtx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()

	hostConfig := &container.HostConfig{
		Binds: []string{
			"/var/run/docker.sock:/var/run/docker.sock:ro",
			fmt.Sprintf("%s:/root/.cache", w.cacheDir),
		},
		AutoRemove: false,
	}

	containerConfig := &container.Config{
		Image: TrivyDockerImage,
		Cmd: []string{
			"image",
			"--severity", "CRITICAL,HIGH",
			"--format", "json",
			"--quiet",
			dockerImage,
		},
		AttachStdout: true,
		AttachStderr: true,
	}

	resp, err := w.dockerCli.ContainerCreate(scanCtx, containerConfig, hostConfig, nil, nil, "")
	if err != nil {
		log.Printf("[TemplateAuditWorker] Error creando contenedor Trivy: %v", err)
		return "failed", 0, 0, []byte(fmt.Sprintf(`{"error": "creación de escáner falló: %s"}`, err.Error()))
	}
	containerID := resp.ID
	defer func() {
		_ = w.dockerCli.ContainerRemove(context.Background(), containerID, container.RemoveOptions{Force: true})
	}()

	if err := w.dockerCli.ContainerStart(scanCtx, containerID, container.StartOptions{}); err != nil {
		log.Printf("[TemplateAuditWorker] Error iniciando contenedor Trivy: %v", err)
		return "failed", 0, 0, []byte(fmt.Sprintf(`{"error": "inicio de escáner falló: %s"}`, err.Error()))
	}

	statusCh, errCh := w.dockerCli.ContainerWait(scanCtx, containerID, container.WaitConditionNotRunning)
	select {
	case waitErr := <-errCh:
		if waitErr != nil {
			log.Printf("[TemplateAuditWorker] Error esperando escaneo Trivy: %v", waitErr)
			return "failed", 0, 0, []byte(fmt.Sprintf(`{"error": "espera de escáner falló: %s"}`, waitErr.Error()))
		}
	case <-statusCh:
	case <-scanCtx.Done():
		log.Printf("[TemplateAuditWorker] Timeout en escaneo Trivy para imagen %s", dockerImage)
		return "failed", 0, 0, []byte(`{"error": "tiempo límite de escaneo agotado (3 min)"}`)
	}

	logsReader, err := w.dockerCli.ContainerLogs(scanCtx, containerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: false,
	})
	if err != nil {
		return "failed", 0, 0, []byte(fmt.Sprintf(`{"error": "no se pudo leer salida de auditoría: %s"}`, err.Error()))
	}
	defer logsReader.Close()

	var stdoutBuf, stderrBuf bytes.Buffer
	_, _ = stdcopy.StdCopy(&stdoutBuf, &stderrBuf, logsReader)
	rawJSON := stdoutBuf.Bytes()

	if len(rawJSON) == 0 {
		return "passed", 0, 0, []byte(`{"Results": []}`)
	}

	var report trivyReport
	if err := json.Unmarshal(rawJSON, &report); err != nil {
		log.Printf("[TemplateAuditWorker] Error parseando JSON de Trivy: %v", err)
		return "passed", 0, 0, rawJSON
	}

	criticalCount := 0
	highCount := 0
	for _, res := range report.Results {
		for _, vuln := range res.Vulnerabilities {
			sev := strings.ToUpper(vuln.Severity)
			if sev == "CRITICAL" {
				criticalCount++
			} else if sev == "HIGH" {
				highCount++
			}
		}
	}

	status := "passed"
	if criticalCount > 0 {
		status = "failed"
	}

	return status, criticalCount, highCount, rawJSON
}

// UpdateTrivyDB actualiza la base de vulnerabilidades de Trivy descargando el paquete a cache
func (w *TemplateAuditWorker) UpdateTrivyDB(ctx context.Context) error {
	syncCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	hostConfig := &container.HostConfig{
		Binds: []string{
			fmt.Sprintf("%s:/root/.cache", w.cacheDir),
		},
		AutoRemove: false,
	}

	containerConfig := &container.Config{
		Image: TrivyDockerImage,
		Cmd: []string{
			"image",
			"--download-db-only",
		},
	}

	resp, err := w.dockerCli.ContainerCreate(syncCtx, containerConfig, hostConfig, nil, nil, "")
	if err != nil {
		return fmt.Errorf("error creando contenedor para descarga de base CVE: %w", err)
	}
	defer func() {
		_ = w.dockerCli.ContainerRemove(context.Background(), resp.ID, container.RemoveOptions{Force: true})
	}()

	if err := w.dockerCli.ContainerStart(syncCtx, resp.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("error iniciando descarga de base CVE: %w", err)
	}

	statusCh, errCh := w.dockerCli.ContainerWait(syncCtx, resp.ID, container.WaitConditionNotRunning)
	select {
	case waitErr := <-errCh:
		if waitErr != nil {
			return fmt.Errorf("error esperando descarga de base CVE: %w", waitErr)
		}
	case <-statusCh:
	case <-syncCtx.Done():
		return fmt.Errorf("timeout en descarga de base CVE")
	}

	log.Println("[TemplateAuditWorker] Base de datos CVE actualizada correctamente en cache")
	return nil
}

func (w *TemplateAuditWorker) recheckAllEOL(ctx context.Context) {
	templates, err := w.govRepo.ListTemplates(ctx, "", "", "")
	if err != nil {
		log.Printf("[TemplateAuditWorker] Error listando plantillas para chequeo EOL: %v", err)
		return
	}
	for _, item := range templates {
		status, date, msg := checkEOL(item.DockerImage)
		if err := w.govRepo.UpdateEOLStatus(ctx, item.ID, status, date, msg); err != nil {
			log.Printf("[TemplateAuditWorker] Error actualizando EOL de plantilla %s: %v", item.ID, err)
		}
	}
}

type EOLResult struct {
	Status  string `json:"status"`
	Date    string `json:"date"`
	Message string `json:"message"`
}

var (
	eolCycleRegex = regexp.MustCompile(`^v?(\d+\.\d+|\d+)`)
	eolHTTPClient = &http.Client{Timeout: 5 * time.Second}
	eolCacheMu    sync.RWMutex
	eolCache      = make(map[string]EOLResult)
)

func parseProductAndCycle(imageRef string) (product string, cycle string, ok bool) {
	parts := strings.Split(imageRef, ":")
	if len(parts) != 2 {
		return "", "", false
	}
	repo := parts[0]
	tag := parts[1]

	if slashIdx := strings.LastIndex(repo, "/"); slashIdx != -1 {
		repo = repo[slashIdx+1:]
	}

	repo = strings.ToLower(repo)
	switch repo {
	case "python":
		product = "python"
	case "node", "nodejs":
		product = "nodejs"
	case "ubuntu":
		product = "ubuntu"
	case "debian":
		product = "debian"
	case "golang", "go":
		product = "go"
	case "postgres", "postgresql":
		product = "postgresql"
	case "mysql":
		product = "mysql"
	case "redis":
		product = "redis"
	case "alpine":
		product = "alpine"
	default:
		return "", "", false
	}

	match := eolCycleRegex.FindStringSubmatch(tag)
	if len(match) < 2 {
		return "", "", false
	}
	cycle = match[1]
	return product, cycle, true
}

func checkEOL(imageRef string) (string, string, string) {
	eolCacheMu.RLock()
	cached, found := eolCache[imageRef]
	eolCacheMu.RUnlock()
	if found {
		return cached.Status, cached.Date, cached.Message
	}

	product, cycle, ok := parseProductAndCycle(imageRef)
	if !ok {
		res := EOLResult{
			Status:  "supported",
			Date:    "",
			Message: "Ciclo de vida no indexado o gestionado por la comunidad",
		}
		eolCacheMu.Lock()
		eolCache[imageRef] = res
		eolCacheMu.Unlock()
		return res.Status, res.Date, res.Message
	}

	url := fmt.Sprintf("https://endoflife.date/api/%s/%s.json", product, cycle)
	resp, err := eolHTTPClient.Get(url)
	if err != nil || resp.StatusCode != http.StatusOK {
		if resp != nil {
			_ = resp.Body.Close()
		}
		res := EOLResult{
			Status:  "supported",
			Date:    "",
			Message: "Ciclo de vida activo o no indexado",
		}
		return res.Status, res.Date, res.Message
	}
	defer resp.Body.Close()

	var payload struct {
		Cycle string          `json:"cycle"`
		EOL   json.RawMessage `json:"eol"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		res := EOLResult{Status: "supported", Date: "", Message: "Soporte activo"}
		return res.Status, res.Date, res.Message
	}

	rawEOL := strings.Trim(string(payload.EOL), `"`)
	now := time.Now()
	var res EOLResult

	if rawEOL == "true" {
		res = EOLResult{
			Status:  "eol",
			Date:    "",
			Message: fmt.Sprintf("Versión %s %s sin soporte oficial (fin de ciclo de vida)", product, cycle),
		}
	} else if rawEOL == "false" || rawEOL == "" {
		res = EOLResult{
			Status:  "supported",
			Date:    "",
			Message: fmt.Sprintf("Versión %s %s con soporte oficial activo", product, cycle),
		}
	} else {
		parsedDate, parseErr := time.Parse("2006-01-02", rawEOL)
		if parseErr != nil {
			res = EOLResult{
				Status:  "supported",
				Date:    rawEOL,
				Message: fmt.Sprintf("Soporte oficial hasta %s", rawEOL),
			}
		} else {
			if parsedDate.Before(now) {
				res = EOLResult{
					Status:  "eol",
					Date:    rawEOL,
					Message: fmt.Sprintf("Versión sin soporte oficial (finalizó el %s)", rawEOL),
				}
			} else if parsedDate.Before(now.AddDate(0, 6, 0)) {
				res = EOLResult{
					Status:  "warning",
					Date:    rawEOL,
					Message: fmt.Sprintf("Próximo a fin de soporte oficial (finaliza el %s)", rawEOL),
				}
			} else {
				res = EOLResult{
					Status:  "supported",
					Date:    rawEOL,
					Message: fmt.Sprintf("Soporte oficial activo hasta %s", rawEOL),
				}
			}
		}
	}

	eolCacheMu.Lock()
	eolCache[imageRef] = res
	eolCacheMu.Unlock()

	return res.Status, res.Date, res.Message
}
