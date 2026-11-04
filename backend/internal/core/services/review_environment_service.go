package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"solv-backend/internal/core/domain"
)

// ReviewEnvironmentService abre un preview inmutable de una entrega IDE. Cada
// clic crea un volumen propio con el snapshot y un contenedor que solo ejecuta la
// app; nunca monta ni reinicia el environment vivo del alumno.
type ReviewEnvironmentService struct {
	repo         domain.ReviewEnvironmentRepository
	source       domain.ReviewEnvironmentSource
	orchestrator domain.ReviewEnvironmentOrchestrator
	hostMonitor  domain.HostMonitor

	idleTimeout  time.Duration
	startTimeout time.Duration
	waitForPort  func(ctx context.Context, hostPort string, timeout time.Duration) error

	locksMu sync.Mutex
	locks   map[string]*sync.Mutex
	reapMu  sync.Mutex
}

func NewReviewEnvironmentService(
	repo domain.ReviewEnvironmentRepository,
	source domain.ReviewEnvironmentSource,
	orchestrator domain.ReviewEnvironmentOrchestrator,
	hostMonitor domain.HostMonitor,
) *ReviewEnvironmentService {
	if repo == nil || source == nil || orchestrator == nil || hostMonitor == nil {
		return nil
	}
	return &ReviewEnvironmentService{
		repo:         repo,
		source:       source,
		orchestrator: orchestrator,
		hostMonitor:  hostMonitor,
		idleTimeout:  domain.ReviewEnvironmentIdleTimeout,
		startTimeout: domain.ReviewEnvironmentStartTimeout,
		waitForPort:  waitForTCPPort,
		locks:        make(map[string]*sync.Mutex),
	}
}

// SetTimeouts ajusta la ventana de inactividad y la espera de arranque. Se usa en
// tests y en nodos con instalaciones más lentas.
func (s *ReviewEnvironmentService) SetTimeouts(idle, start time.Duration) {
	if idle > 0 {
		s.idleTimeout = idle
	}
	if start > 0 {
		s.startTimeout = start
	}
}

// Start devuelve el preview activo de la entrega o crea uno nuevo. Si la app no
// llega a escuchar en el puerto declarado, el contenedor y su volumen se retiran:
// un preview roto no debe consumir RAM hasta que venza el TTL.
func (s *ReviewEnvironmentService) Start(ctx context.Context, tenantID, teacherID, submissionID string) (*domain.ReviewEnvironmentAccess, error) {
	if s == nil {
		return nil, domain.ErrReviewEnvironmentUnavailable
	}
	if strings.TrimSpace(tenantID) == "" {
		return nil, domain.ErrInvalidTenant
	}
	if strings.TrimSpace(submissionID) == "" {
		return nil, domain.ErrNotFound
	}

	unlock := s.lockFor(submissionID)
	defer unlock()

	if existing, err := s.repo.GetActiveBySubmission(ctx, tenantID, submissionID); err == nil && existing != nil {
		// Un arranque que quedó colgado (proceso caído) no debe bloquear el
		// botón para siempre: pasado el margen, el pedido actual lo reemplaza.
		if existing.Status == domain.ReviewEnvironmentStatusStarting && time.Since(existing.CreatedAt) <= s.startTimeout+30*time.Second {
			return nil, domain.ErrReviewEnvironmentStarting
		}
		if existing.Status == domain.ReviewEnvironmentStatusRunning && strings.TrimSpace(existing.AccessURL) != "" && s.containerAlive(ctx, existing) {
			expiresAt := time.Now().UTC().Add(s.idleTimeout)
			_ = s.repo.Touch(ctx, existing.ID, time.Now().UTC(), expiresAt, existing.LastRxBytes, existing.LastTxBytes)
			return &domain.ReviewEnvironmentAccess{URL: existing.AccessURL, ExpiresAt: expiresAt}, nil
		}
		// Un registro running sin URL, sin contenedor o con el contenedor ya
		// fuera de marcha quedó a medias: se libera para crear uno limpio.
		if !s.release(ctx, existing) {
			return nil, fmt.Errorf("%w: no se pudo liberar el preview anterior", domain.ErrReviewEnvironmentStartFailed)
		}
		_ = s.repo.MarkExpired(ctx, existing.ID)
	} else if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}

	// A partir de acá la creación es una operación de infraestructura: se
	// desacopla de la cancelación del navegador para no dejar volúmenes huérfanos.
	workCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Minute)
	defer cancel()

	reviewCtx, err := s.source.GetReviewEnvironmentContext(workCtx, tenantID, teacherID, submissionID)
	if err != nil {
		return nil, err
	}
	if reviewCtx.EnvironmentType != domain.PracticeEnvironmentTypeIDEPersistente {
		return nil, domain.ErrReviewEnvironmentRequiresIDE
	}
	if strings.TrimSpace(reviewCtx.TemplateImage) == "" {
		return nil, domain.ErrReviewEnvironmentTemplateMissing
	}
	command := strings.TrimSpace(reviewCtx.AppRunCommand)
	if command == "" {
		return nil, domain.ErrReviewEnvironmentCommandMissing
	}
	if len(command) > domain.ReviewEnvironmentMaxCommandLength {
		return nil, fmt.Errorf("%w: app_run_command exceeds %d characters", domain.ErrReviewEnvironmentCommandMissing, domain.ReviewEnvironmentMaxCommandLength)
	}
	appPort := reviewCtx.AppPort
	if appPort == 0 {
		appPort = domain.ReviewEnvironmentDefaultPort
	}
	if appPort < 1 || appPort > 65535 {
		return nil, domain.ErrInvalidAppPort
	}
	files, ok := domain.ParseProjectSnapshot(reviewCtx.Code, reviewCtx.Language)
	if !ok || len(files) == 0 {
		return nil, domain.ErrReviewEnvironmentSnapshotUnavailable
	}

	memoryMB := reviewCtx.TemplateMemoryMB
	if memoryMB < domain.ReviewEnvironmentMinMemoryMB {
		memoryMB = domain.ReviewEnvironmentMinMemoryMB
	}
	if !s.hostMonitor.CanAllocateMemory(memoryMB) {
		return nil, domain.ErrReviewEnvironmentHostMemory
	}

	envID := uuid.NewString()
	compactID := shortReviewID(strings.ReplaceAll(envID, "-", ""))
	baseDomain := getBaseDomain()
	hostPort, portErr := reservePracticeEnvironmentHostPort()
	if portErr != nil {
		if baseDomain == "" {
			return nil, fmt.Errorf("%w: %v", domain.ErrReviewEnvironmentStartFailed, portErr)
		}
		log.Printf("[Review Environment] WARNING: no se pudo reservar puerto local para %s: %v", envID, portErr)
		hostPort = ""
	}
	accessURL := reviewEnvironmentAccessURL(compactID, hostPort)

	volumeName := fmt.Sprintf("solv-review-env-%s", compactID)
	containerName := fmt.Sprintf("solv-review-env-%s", compactID)
	now := time.Now().UTC()
	env := &domain.ReviewEnvironment{
		ID:             envID,
		TenantID:       tenantID,
		SubmissionID:   submissionID,
		TeacherID:      teacherID,
		VolumeName:     volumeName,
		AccessURL:      accessURL,
		Status:         domain.ReviewEnvironmentStatusStarting,
		LastActivityAt: now,
		ExpiresAt:      now.Add(s.idleTimeout),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := s.repo.Create(workCtx, env); err != nil {
		return nil, err
	}

	if err := s.orchestrator.EnsureVolumeExists(workCtx, volumeName); err != nil {
		return nil, s.fail(workCtx, env, "volume_create_failed", err)
	}
	fileMap := make(map[string]string, len(files))
	for _, file := range files {
		fileMap[file.Path] = file.Content
	}
	if err := s.orchestrator.WritePracticeEnvironmentVolumeFiles(workCtx, volumeName, fileMap); err != nil {
		return nil, s.fail(workCtx, env, "snapshot_write_failed", err)
	}

	networkName := "solv-traefik-net"
	if err := s.orchestrator.EnsureICCDisabledNetworkExists(workCtx, networkName); err != nil {
		return nil, s.fail(workCtx, env, "network_create_failed", err)
	}

	labels := map[string]string{
		"solv.managed":               "true",
		"solv.review_environment":    "true",
		"solv.review_environment_id": envID,
	}
	if baseDomain != "" {
		router := "review-" + compactID
		labels["traefik.enable"] = "true"
		labels["traefik.docker.network"] = networkName
		labels[fmt.Sprintf("traefik.http.routers.%s.rule", router)] = fmt.Sprintf("Host(`review-%s.%s`)", compactID, baseDomain)
		labels[fmt.Sprintf("traefik.http.services.%s.loadbalancer.server.port", router)] = strconv.Itoa(appPort)
	}

	containerID, err := s.orchestrator.StartReviewEnvironmentContainer(workCtx, domain.ReviewEnvironmentContainerConfig{
		Image:         reviewCtx.TemplateImage,
		ContainerName: containerName,
		VolumeName:    volumeName,
		NetworkName:   networkName,
		AppPort:       appPort,
		Command:       command,
		SetupScript:   reviewCtx.TemplateSetup,
		Labels:        labels,
		MemoryLimitMB: memoryMB,
		HostPort:      hostPort,
	})
	if err != nil {
		return nil, s.fail(workCtx, env, "container_start_failed", err)
	}
	env.ContainerID = &containerID

	metrics, _ := s.orchestrator.GetContainerMetrics(workCtx, containerID)
	var rxBytes, txBytes int64
	if metrics != nil {
		rxBytes = uint64ToInt64(metrics.RxBytes)
		txBytes = uint64ToInt64(metrics.TxBytes)
	}
	expiresAt := time.Now().UTC().Add(s.idleTimeout)
	if err := s.repo.SetRunning(workCtx, env.ID, containerID, accessURL, expiresAt, rxBytes, txBytes); err != nil {
		return nil, s.fail(workCtx, env, "state_update_failed", err)
	}
	env.Status = domain.ReviewEnvironmentStatusRunning
	env.ExpiresAt = expiresAt

	if hostPort != "" && s.startTimeout > 0 {
		if waitErr := s.waitForPort(workCtx, hostPort, s.startTimeout); waitErr != nil {
			logs, _ := s.orchestrator.GetContainerLogs(workCtx, containerID, 40)
			log.Printf("[Review Environment] app %s no escuchó en %d s: %v\n%s", envID, int(s.startTimeout.Seconds()), waitErr, truncateLog(logs))
			return nil, s.fail(workCtx, env, "app_not_ready", fmt.Errorf("%w: la app no escuchó en el puerto %d", domain.ErrReviewEnvironmentNotReady, appPort))
		}
	}

	return &domain.ReviewEnvironmentAccess{URL: accessURL, ExpiresAt: expiresAt}, nil
}

// ReapExpired apaga los previews sin tráfico. El activo se renueva cuando el
// contador de red crece; sin tráfico durante la ventana de inactividad, el
// contenedor y su volumen se eliminan. Lo llama el QoS Worker.
func (s *ReviewEnvironmentService) ReapExpired(ctx context.Context) {
	if s == nil || !s.reapMu.TryLock() {
		return
	}
	defer s.reapMu.Unlock()

	active, err := s.repo.ListActive(ctx)
	if err != nil {
		log.Printf("[Review Environment] Error listing active previews: %v", err)
		return
	}
	now := time.Now().UTC()
	for _, env := range active {
		if env == nil {
			continue
		}
		if env.Status == domain.ReviewEnvironmentStatusStarting {
			if now.Sub(env.CreatedAt) < s.startTimeout+30*time.Second {
				continue
			}
			if s.release(ctx, env) {
				_ = s.repo.MarkFailed(ctx, env.ID, "app_start_timeout", "el preview no terminó de arrancar")
			}
			continue
		}
		if env.ContainerID == nil || strings.TrimSpace(*env.ContainerID) == "" {
			s.expire(ctx, env)
			continue
		}

		metrics, mErr := s.orchestrator.GetContainerMetrics(ctx, *env.ContainerID)
		if mErr != nil {
			log.Printf("[Review Environment] No se pudo leer métricas de %s: %v", env.ID, mErr)
			continue
		}
		if metrics != nil && !metrics.IsRunning {
			s.expire(ctx, env)
			continue
		}
		if metrics == nil {
			continue
		}

		rxBytes := uint64ToInt64(metrics.RxBytes)
		txBytes := uint64ToInt64(metrics.TxBytes)
		if rxBytes > env.LastRxBytes || txBytes > env.LastTxBytes {
			expiresAt := now.Add(s.idleTimeout)
			if err := s.repo.Touch(ctx, env.ID, now, expiresAt, rxBytes, txBytes); err != nil {
				log.Printf("[Review Environment] No se pudo renovar %s: %v", env.ID, err)
			}
			continue
		}
		if now.Before(env.ExpiresAt) {
			continue
		}
		s.expire(ctx, env)
	}
}

func (s *ReviewEnvironmentService) expire(ctx context.Context, env *domain.ReviewEnvironment) {
	if !s.release(ctx, env) {
		return
	}
	if err := s.repo.MarkExpired(ctx, env.ID); err != nil {
		log.Printf("[Review Environment] No se pudo marcar %s como expirado: %v", env.ID, err)
	}
}

// release retira el contenedor y el volumen del preview. Devuelve false si el
// contenedor sigue vivo, para no marcar expirado un recurso que aún consume RAM.
func (s *ReviewEnvironmentService) release(ctx context.Context, env *domain.ReviewEnvironment) bool {
	if env == nil {
		return true
	}
	if env.ContainerID != nil && strings.TrimSpace(*env.ContainerID) != "" {
		if err := s.orchestrator.StopAndRemoveContainer(ctx, *env.ContainerID); err != nil {
			log.Printf("[Review Environment] No se pudo detener %s: %v", env.ID, err)
			return false
		}
	}
	if strings.TrimSpace(env.VolumeName) != "" {
		if err := s.orchestrator.RemoveVolume(ctx, env.VolumeName); err != nil {
			log.Printf("[Review Environment] No se pudo borrar el volumen %s: %v", env.VolumeName, err)
			return false
		}
	}
	return true
}

func (s *ReviewEnvironmentService) fail(ctx context.Context, env *domain.ReviewEnvironment, code string, cause error) error {
	// Un fallo de volumen, snapshot o red también es un fallo de arranque: el
	// handler responde 502 y no un 500 genérico.
	if !errors.Is(cause, domain.ErrReviewEnvironmentNotReady) && !errors.Is(cause, domain.ErrReviewEnvironmentStartFailed) {
		cause = fmt.Errorf("%w: %v", domain.ErrReviewEnvironmentStartFailed, cause)
	}
	if !s.release(ctx, env) {
		// El registro queda activo para que el QoS reintente la limpieza; no se
		// marca failed un contenedor que todavía puede estar consumiendo RAM.
		return cause
	}
	message := truncateLog(cause.Error())
	if err := s.repo.MarkFailed(ctx, env.ID, code, message); err != nil {
		log.Printf("[Review Environment] No se pudo registrar el fallo de %s: %v", env.ID, err)
	}
	return cause
}

// containerAlive evita entregar una URL muerta. Ante un error del daemon se
// conserva el preview: no se recrea por una caída temporal.
func (s *ReviewEnvironmentService) containerAlive(ctx context.Context, env *domain.ReviewEnvironment) bool {
	if env == nil || env.ContainerID == nil || strings.TrimSpace(*env.ContainerID) == "" {
		return false
	}
	metrics, err := s.orchestrator.GetContainerMetrics(ctx, *env.ContainerID)
	if err != nil {
		log.Printf("[Review Environment] No se pudo verificar el contenedor %s: %v", env.ID, err)
		return true
	}
	if metrics == nil {
		return true
	}
	return metrics.IsRunning
}

func (s *ReviewEnvironmentService) lockFor(submissionID string) func() {
	s.locksMu.Lock()
	lock := s.locks[submissionID]
	if lock == nil {
		lock = &sync.Mutex{}
		s.locks[submissionID] = lock
	}
	s.locksMu.Unlock()
	lock.Lock()
	return lock.Unlock
}

func shortReviewID(compactID string) string {
	if len(compactID) > 12 {
		return compactID[:12]
	}
	return compactID
}

func reviewEnvironmentAccessURL(compactID, hostPort string) string {
	if baseDomain := getBaseDomain(); baseDomain != "" {
		return fmt.Sprintf("https://review-%s.%s", compactID, baseDomain)
	}
	if hostPort != "" {
		return fmt.Sprintf("http://localhost:%s", hostPort)
	}
	return ""
}

func waitForTCPPort(ctx context.Context, hostPort string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	address := net.JoinHostPort("127.0.0.1", hostPort)
	var lastErr error
	for {
		conn, err := net.DialTimeout("tcp", address, time.Second)
		if err == nil {
			_ = conn.Close()
			return nil
		}
		lastErr = err
		if time.Now().After(deadline) {
			return lastErr
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(400 * time.Millisecond):
		}
	}
}

func uint64ToInt64(value uint64) int64 {
	const maxInt64 = uint64(^uint64(0) >> 1)
	if value > maxInt64 {
		return int64(maxInt64)
	}
	return int64(value)
}

func truncateLog(value string) string {
	value = strings.TrimSpace(value)
	if len(value) <= 2000 {
		return value
	}
	return value[:2000]
}
