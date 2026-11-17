package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/errdefs"
	"github.com/docker/docker/pkg/stdcopy"
)

// ensureTrivyImage deja la imagen del escáner disponible antes de crear un
// contenedor. La descarga de la base CVE también la necesita: sin esto, un
// arranque en frío no puede primar la caché.
func (w *TemplateAuditWorker) ensureTrivyImage(ctx context.Context) error {
	_, _, err := w.dockerCli.ImageInspectWithRaw(ctx, TrivyDockerImage)
	if err == nil {
		return nil
	}
	if !errdefs.IsNotFound(err) {
		return err
	}
	log.Printf("[TemplateAuditWorker] Descargando imagen de auditoría %s...", TrivyDockerImage)
	out, pullErr := w.dockerCli.ImagePull(ctx, TrivyDockerImage, image.PullOptions{})
	if pullErr != nil {
		return pullErr
	}
	defer func() { _ = out.Close() }()
	if _, copyErr := io.Copy(io.Discard, out); copyErr != nil {
		return copyErr
	}
	return nil
}

// trivyResources fija el techo de CPU y memoria del escáner y de su descarga.
// Ninguno de los dos puede competir con el host.
func trivyResources() container.Resources {
	return container.Resources{
		NanoCPUs: trivyCPULimit,
		Memory:   trivyMemoryLimit,
	}
}

// trivyScanArgs arma el comando del escaneo. `--skip-db-update` evita que cada
// auditoría vuelva a descargar la base CVE: Trivy pide refresco cada 6 h y el
// worker la sincroniza antes de mirar la cola, así que la base ya está fresca.
// Sin este flag, cada plantilla auditada cuesta 133 MB de red y 1,7 GB de
// escritura, y el resultado del escaneo es el mismo.
func trivyScanArgs(dockerImage string) []string {
	return []string{
		"image",
		"--severity", "CRITICAL,HIGH",
		"--format", "json",
		"--quiet",
		"--skip-db-update",
		// El escaneo corre sin red: si intentara actualizar el índice Java
		// abortaría con "network is unreachable" y la plantilla quedaría
		// reintentándose para siempre. El índice lo deja el worker en caché.
		"--skip-java-db-update",
		dockerImage,
	}
}

// runTrivyScan ejecuta Trivy escaneando la imagen por CVEs de severidad CRITICAL y HIGH
func (w *TemplateAuditWorker) runTrivyScan(ctx context.Context, dockerImage string) (string, int, int, []byte) {
	if err := w.ensureTrivyImage(ctx); err != nil {
		log.Printf("[TemplateAuditWorker] Error preparando Trivy: %v", err)
		return auditStatusError, 0, 0, trivyErrorReport("trivy_pull", err)
	}

	scanCtx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()

	hostConfig := &container.HostConfig{
		NetworkMode: "none",
		Binds: []string{
			"/var/run/docker.sock:/var/run/docker.sock:ro",
			fmt.Sprintf("%s:/root/.cache", w.cacheDir),
		},
		Resources:  trivyResources(),
		AutoRemove: false,
	}

	containerConfig := &container.Config{
		Image:        TrivyDockerImage,
		Cmd:          trivyScanArgs(dockerImage),
		AttachStdout: true,
		AttachStderr: true,
	}

	resp, err := w.dockerCli.ContainerCreate(scanCtx, containerConfig, hostConfig, nil, nil, "")
	if err != nil {
		log.Printf("[TemplateAuditWorker] Error creando contenedor Trivy: %v", err)
		return auditStatusError, 0, 0, trivyErrorReport("trivy_create", err)
	}
	containerID := resp.ID
	defer func() {
		_ = w.dockerCli.ContainerRemove(context.Background(), containerID, container.RemoveOptions{Force: true})
	}()

	if err := w.dockerCli.ContainerStart(scanCtx, containerID, container.StartOptions{}); err != nil {
		log.Printf("[TemplateAuditWorker] Error iniciando contenedor Trivy: %v", err)
		return auditStatusError, 0, 0, trivyErrorReport("trivy_start", err)
	}

	statusCh, errCh := w.dockerCli.ContainerWait(scanCtx, containerID, container.WaitConditionNotRunning)
	var exitCode int64
	select {
	case waitErr := <-errCh:
		if waitErr != nil {
			log.Printf("[TemplateAuditWorker] Error esperando escaneo Trivy: %v", waitErr)
			return auditStatusError, 0, 0, trivyErrorReport("trivy_wait", waitErr)
		}
	case res, ok := <-statusCh:
		if !ok {
			return auditStatusError, 0, 0, trivyErrorReport("trivy_wait", fmt.Errorf("el contenedor cerró sin código de salida"))
		}
		exitCode = res.StatusCode
	case <-scanCtx.Done():
		log.Printf("[TemplateAuditWorker] Timeout en escaneo Trivy para imagen %s", dockerImage)
		return auditStatusError, 0, 0, trivyErrorReport("trivy_timeout", fmt.Errorf("tiempo límite de escaneo agotado (3 min)"))
	}
	if exitCode != 0 {
		return auditStatusError, 0, 0, trivyErrorReport("trivy_exit", fmt.Errorf("el escáner terminó con código %d", exitCode))
	}

	logsReader, err := w.dockerCli.ContainerLogs(scanCtx, containerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: false,
	})
	if err != nil {
		return auditStatusError, 0, 0, trivyErrorReport("trivy_logs", err)
	}
	defer func() { _ = logsReader.Close() }()

	var stdoutBuf, stderrBuf bytes.Buffer
	_, _ = stdcopy.StdCopy(&stdoutBuf, &stderrBuf, logsReader)
	rawJSON := stdoutBuf.Bytes()

	if len(strings.TrimSpace(string(rawJSON))) == 0 {
		return auditStatusError, 0, 0, trivyErrorReport("trivy_empty", fmt.Errorf("el escáner no produjo un informe"))
	}

	var report trivyReport
	if err := json.Unmarshal(rawJSON, &report); err != nil {
		log.Printf("[TemplateAuditWorker] Error parseando JSON de Trivy: %v", err)
		// Sin un informe legible no se puede afirmar que la imagen esté limpia.
		return auditStatusError, 0, 0, trivyErrorReport("trivy_report", err)
	}

	criticalCount := 0
	highCount := 0
	for _, res := range report.Results {
		for _, vuln := range res.Vulnerabilities {
			sev := strings.ToUpper(vuln.Severity)
			switch sev {
			case "CRITICAL":
				criticalCount++
			case "HIGH":
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

// UpdateTrivyDB actualiza la base de vulnerabilidades de Trivy descargando el paquete a cache.
// La base comprimida ronda los 130 MB y se expande a más de 1 GB: cinco minutos
// solo alcanzan en redes rápidas. Quince dan margen sin dejar el worker colgado.
func (w *TemplateAuditWorker) UpdateTrivyDB(ctx context.Context) (err error) {
	syncCtx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()

	// Una descarga interrumpida deja trivy.db a medias; se descarta para no
	// ocupar espacio ni simular una base válida en el siguiente arranque. Se
	// borra trivy/db, que el worker dejó escribible, y se vuelve a preparar para
	// que el siguiente intento también pueda purgarse.
	defer func() {
		if err == nil || w.hasTrivyDB() || strings.TrimSpace(w.cacheDir) == "" {
			return
		}
		if purgeErr := os.RemoveAll(filepath.Join(w.cacheDir, "trivy", "db")); purgeErr != nil {
			log.Printf("[TemplateAuditWorker] No se pudo limpiar la base CVE incompleta: %v", purgeErr)
		}
		prepareTrivyCacheDir(w.cacheDir)
	}()

	if err := w.ensureTrivyImage(syncCtx); err != nil {
		return fmt.Errorf("error preparando la imagen de Trivy: %w", err)
	}

	hostConfig := &container.HostConfig{
		Binds: []string{
			fmt.Sprintf("%s:/root/.cache", w.cacheDir),
		},
		Resources:  trivyResources(),
		AutoRemove: false,
	}

	containerConfig := &container.Config{
		Image: TrivyDockerImage,
		Cmd: []string{
			"image",
			"--download-db-only",
		},
		Labels: map[string]string{auditContainerLabel: trivyDBContainerRole},
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

	// ContainerWait no falla si el proceso sale con error: hay que leer el código.
	inspect, inspectErr := w.dockerCli.ContainerInspect(context.Background(), resp.ID)
	if inspectErr != nil {
		return fmt.Errorf("no se pudo verificar la descarga de base CVE: %w", inspectErr)
	}
	if inspect.State != nil && inspect.State.ExitCode != 0 {
		return fmt.Errorf("la descarga de base CVE terminó con código %d", inspect.State.ExitCode)
	}
	if !w.hasTrivyDB() {
		return fmt.Errorf("la descarga de base CVE no dejó una base completa")
	}

	log.Println("[TemplateAuditWorker] Base de datos CVE actualizada correctamente en cache")

	// El índice Java viaja aparte y se descarga acá porque el escaneo corre sin
	// red. Si falla no se bloquea la cola: solo quedan sin analizar las imágenes
	// que traen JAR.
	switch {
	case w.javaDBIsFresh():
		log.Println("[TemplateAuditWorker] Índice Java vigente en cache; no se descarga")
	default:
		if javaErr := w.javaDBSyncer(ctx); javaErr != nil {
			log.Printf("[TemplateAuditWorker] No se pudo actualizar el índice Java: %v. Las imágenes con JAR quedarán sin analizar hasta que haya red", javaErr)
		}
	}
	return nil
}

// trivyJavaDBMaxAge espacia la descarga del índice Java. Son 954 MiB
// comprimidos y se expanden a varios GB: bajarlo a diario no se justifica en un
// host de desarrollo, porque la cobertura de JAR cambia mucho más despacio que
// la base CVE.
const trivyJavaDBMaxAge = 7 * 24 * time.Hour

// javaDBIsFresh indica si el índice Java de la caché se descargó dentro del
// margen permitido, para no repetir una descarga de casi 1 GB.
func (w *TemplateAuditWorker) javaDBIsFresh() bool {
	raw, err := os.ReadFile(filepath.Join(w.cacheDir, "trivy", "java-db", "metadata.json"))
	if err != nil {
		return false
	}
	var meta struct {
		UpdatedAt    string `json:"UpdatedAt"`
		DownloadedAt string `json:"DownloadedAt"`
	}
	if err := json.Unmarshal(raw, &meta); err != nil {
		return false
	}
	stamp := strings.TrimSpace(meta.UpdatedAt)
	if stamp == "" {
		stamp = strings.TrimSpace(meta.DownloadedAt)
	}
	updated, err := time.Parse(time.RFC3339, stamp)
	if err != nil {
		return false
	}
	return w.now().Sub(updated) < trivyJavaDBMaxAge
}

// updateJavaDB descarga el índice de dependencias Java en la misma caché que usa
// el escaneo. Sin él, Trivy aborta cualquier imagen con JAR: el escaneo corre con
// la red apagada y no puede bajar el índice por su cuenta.
func (w *TemplateAuditWorker) updateJavaDB(ctx context.Context) error {
	// Casi 1 GB comprimido tarda diez minutos en una conexión doméstica y luego
	// se expande: quince minutos no alcanzan para completar descarga y escritura.
	syncCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()

	if err := w.ensureTrivyImage(syncCtx); err != nil {
		return fmt.Errorf("error preparando la imagen de Trivy: %w", err)
	}

	hostConfig := &container.HostConfig{
		Binds: []string{
			fmt.Sprintf("%s:/root/.cache", w.cacheDir),
		},
		Resources:  trivyResources(),
		AutoRemove: false,
	}
	containerConfig := &container.Config{
		Image:  TrivyDockerImage,
		Cmd:    []string{"image", "--download-java-db-only"},
		Labels: map[string]string{auditContainerLabel: trivyDBContainerRole},
	}

	resp, err := w.dockerCli.ContainerCreate(syncCtx, containerConfig, hostConfig, nil, nil, "")
	if err != nil {
		return fmt.Errorf("error creando contenedor para el índice Java: %w", err)
	}
	defer func() {
		_ = w.dockerCli.ContainerRemove(context.Background(), resp.ID, container.RemoveOptions{Force: true})
	}()

	if err := w.dockerCli.ContainerStart(syncCtx, resp.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("error iniciando la descarga del índice Java: %w", err)
	}

	statusCh, errCh := w.dockerCli.ContainerWait(syncCtx, resp.ID, container.WaitConditionNotRunning)
	select {
	case waitErr := <-errCh:
		if waitErr != nil {
			return fmt.Errorf("error esperando el índice Java: %w", waitErr)
		}
	case <-statusCh:
	case <-syncCtx.Done():
		return fmt.Errorf("timeout en la descarga del índice Java")
	}

	inspect, inspectErr := w.dockerCli.ContainerInspect(context.Background(), resp.ID)
	if inspectErr != nil {
		return fmt.Errorf("no se pudo verificar el índice Java: %w", inspectErr)
	}
	if inspect.State != nil && inspect.State.ExitCode != 0 {
		return fmt.Errorf("la descarga del índice Java terminó con código %d", inspect.State.ExitCode)
	}

	log.Println("[TemplateAuditWorker] Índice Java actualizado en cache")
	return nil
}

// pruneStaleDBDownloads elimina los contenedores de descarga de la base CVE que
// quedaron vivos de un arranque anterior. El worker los crea sin AutoRemove
// porque necesita leer el código de salida, y hasta ahora solo los borraba un
// defer del propio proceso: si el backend se reiniciaba durante la ventana de
// 15 minutos, el contenedor seguía descargando sin dueño, con un núcleo
// ocupado, compitiendo por la misma caché. Se ejecuta una vez al arrancar,
// antes de primar la base.
func (w *TemplateAuditWorker) pruneStaleDBDownloads(ctx context.Context) {
	if w.dockerCli == nil {
		return
	}
	filter := filters.NewArgs(filters.Arg("label", auditContainerLabel+"="+trivyDBContainerRole))
	stale, err := w.dockerCli.ContainerList(ctx, container.ListOptions{All: true, Filters: filter})
	if err != nil {
		log.Printf("[TemplateAuditWorker] No se pudo revisar descargas de base CVE huérfanas: %v", err)
		return
	}
	for _, c := range stale {
		if removeErr := w.dockerCli.ContainerRemove(ctx, c.ID, container.RemoveOptions{Force: true}); removeErr != nil {
			log.Printf("[TemplateAuditWorker] No se pudo eliminar el contenedor de descarga huérfano %.12s: %v", c.ID, removeErr)
			continue
		}
		log.Printf("[TemplateAuditWorker] Contenedor de descarga huérfano %.12s eliminado (estado %s)", c.ID, c.State)
	}
}

func trivyErrorReport(kind string, err error) []byte {
	detail := ""
	if err != nil {
		detail = err.Error()
	}
	payload, _ := json.Marshal(map[string]string{"error": kind, "detail": detail})
	return payload
}
