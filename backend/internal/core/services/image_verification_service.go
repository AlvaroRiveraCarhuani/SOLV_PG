package services

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"

	"solv-backend/internal/core/domain"
)

var (
	dockerImageVerificationRegex = regexp.MustCompile(`^[a-z0-9]+(?:[._-][a-z0-9]+)*(?:\/[a-z0-9]+(?:[._-][a-z0-9]+)*)*:[a-zA-Z0-9_.-]+$`)
	ErrImageFormatInvalid        = errors.New("formato de imagen inválido: debe ser repositorio:tag sin espacios (ej: python:3.12-slim-bookworm)")
	ErrLatestTagForbiddenVerif   = errors.New("el tag :latest no está permitido por reproducibilidad y gobernanza")
)

type cacheEntry struct {
	result    *domain.ImageVerificationResult
	expiresAt time.Time
}

type ImageVerificationService struct {
	dockerCli    *client.Client
	cache        map[string]cacheEntry
	mu           sync.RWMutex
	dockerRoot   string
	customStatfs func(path string) (totalGB float64, freeGB float64, err error)
}

func NewImageVerificationService(dockerCli *client.Client) *ImageVerificationService {
	return &ImageVerificationService{
		dockerCli:  dockerCli,
		cache:      make(map[string]cacheEntry),
		dockerRoot: "/var/lib/docker",
	}
}

// SetCustomStatfs permite inyectar cálculo de disco en tests unitarios
func (s *ImageVerificationService) SetCustomStatfs(fn func(path string) (float64, float64, error)) {
	s.customStatfs = fn
}

// ListLocalImages lista imágenes residentes en el host para acelerar la selección
func (s *ImageVerificationService) ListLocalImages(ctx context.Context) ([]domain.LocalImageItem, error) {
	if s.dockerCli == nil {
		return []domain.LocalImageItem{}, nil
	}

	rawImages, err := s.dockerCli.ImageList(ctx, image.ListOptions{All: false})
	if err != nil {
		return nil, fmt.Errorf("error al listar imágenes del host: %w", err)
	}

	var items []domain.LocalImageItem
	seenTags := make(map[string]bool)

	for _, img := range rawImages {
		for _, tag := range img.RepoTags {
			if tag == "<none>:<none>" || tag == "" || seenTags[tag] {
				continue
			}
			seenTags[tag] = true

			sizeMB := img.Size / (1024 * 1024)
			hasLatest := strings.HasSuffix(tag, ":latest")
			isOfficial := !strings.Contains(tag, "/") || strings.HasPrefix(tag, "library/")

			items = append(items, domain.LocalImageItem{
				RepoTag:      tag,
				SizeMB:       sizeMB,
				CreatedAt:    time.Unix(img.Created, 0),
				IsOfficial:   isOfficial,
				HasLatestTag: hasLatest,
			})
		}
	}

	return items, nil
}

// VerifyImage ejecuta la verificación de compatibilidad, existencia, almacenamiento y smoke test
func (s *ImageVerificationService) VerifyImage(ctx context.Context, imageRef string, force bool) (*domain.ImageVerificationResult, error) {
	imageRef = strings.TrimSpace(imageRef)
	if imageRef == "" {
		return nil, errors.New("la referencia de imagen no puede estar vacía")
	}

	if !dockerImageVerificationRegex.MatchString(imageRef) {
		return nil, ErrImageFormatInvalid
	}

	if strings.HasSuffix(strings.ToLower(imageRef), ":latest") {
		return nil, ErrLatestTagForbiddenVerif
	}

	// 1. Verificación de Caché asimétrico (24h acierto, 5min fallo)
	if !force {
		s.mu.RLock()
		entry, found := s.cache[imageRef]
		s.mu.RUnlock()

		if found && time.Now().Before(entry.expiresAt) {
			resultCopy := *entry.result
			resultCopy.Cached = true
			return &resultCopy, nil
		}
	}

	hostArch := runtime.GOARCH
	totalGB, freeGB, err := s.getHostDiskStats()
	if err != nil {
		totalGB = 100.0
		freeGB = 50.0
	}

	result := &domain.ImageVerificationResult{
		ImageRef:        imageRef,
		HostArch:        hostArch,
		HostDiskTotalGB: round2(totalGB),
		HostDiskFreeGB:  round2(freeGB),
		VerifiedAt:      time.Now(),
		Cached:          false,
	}

	if s.dockerCli == nil {
		result.Exists = true
		result.ArchitectureCompatible = true
		result.StorageStatus = domain.StorageStatusOK
		result.StorageMessage = "Docker daemon mock: verificación omitida"
		s.storeCache(imageRef, result, 24*time.Hour)
		return result, nil
	}

	// 2. Comprobar si la imagen ya reside localmente en el host
	inspect, _, inspectErr := s.dockerCli.ImageInspectWithRaw(ctx, imageRef)
	if inspectErr == nil {
		// La imagen existe localmente
		result.IsLocal = true
		result.Exists = true
		result.SizeBytes = inspect.Size
		result.SizeFormatted = formatBytes(inspect.Size)
		result.EstimatedUncompressedMB = inspect.Size / (1024 * 1024)

		if inspect.Architecture == hostArch || (hostArch == "amd64" && inspect.Architecture == "x86_64") {
			result.ArchitectureCompatible = true
		} else {
			result.ArchitectureCompatible = false
			result.BuildxSuggestion = fmt.Sprintf("docker buildx build --platform linux/%s -t %s .", hostArch, imageRef)
		}

		result.SupportedPlatforms = []string{fmt.Sprintf("%s/%s", inspect.Os, inspect.Architecture)}

		// Evaluación de disco
		s.evaluateStorageStatus(result, result.EstimatedUncompressedMB, freeGB, totalGB)

		// 3. Smoke Test / Capability probe si la arquitectura es compatible
		if result.ArchitectureCompatible {
			probeCtx, probeCancel := context.WithTimeout(ctx, 4*time.Second)
			probeResult := s.runSafeCapabilityProbe(probeCtx, imageRef)
			probeCancel()
			result.CapabilitiesProbe = probeResult
		}

		s.storeCache(imageRef, result, 24*time.Hour)
		return result, nil
	}

	// 4. Si no es local, consultar manifiesto remoto mediante DistributionInspect
	dist, distErr := s.dockerCli.DistributionInspect(ctx, imageRef, "")
	if distErr != nil {
		result.Exists = false
		result.ErrorMessage = fmt.Sprintf("Imagen no encontrada en el registro o inaccesible: %v", distErr)
		result.StorageStatus = domain.StorageStatusWarning
		result.StorageMessage = "No se pudo verificar el tamaño en el registro público"
		s.storeCache(imageRef, result, 5*time.Minute)
		return result, nil
	}

	result.Exists = true
	result.IsLocal = false
	result.SizeBytes = dist.Descriptor.Size
	result.SizeFormatted = formatBytes(dist.Descriptor.Size)
	// Factor de expansión decompresión estimado: 2.5x del tamaño comprimido
	if dist.Descriptor.Size > 0 {
		result.EstimatedUncompressedMB = int64(float64(dist.Descriptor.Size) * 2.5 / (1024 * 1024))
		if result.EstimatedUncompressedMB < 100 {
			result.EstimatedUncompressedMB = 100
		}
	} else {
		result.EstimatedUncompressedMB = 350 // Estimación promedio de capas base
	}

	var platforms []string
	archMatch := false
	for _, p := range dist.Platforms {
		platStr := fmt.Sprintf("%s/%s", p.OS, p.Architecture)
		platforms = append(platforms, platStr)
		if (p.OS == "linux" || p.OS == "") && (p.Architecture == hostArch || (hostArch == "amd64" && p.Architecture == "x86_64")) {
			archMatch = true
		}
	}

	result.SupportedPlatforms = platforms
	result.ArchitectureCompatible = archMatch
	if !archMatch && len(platforms) > 0 {
		result.BuildxSuggestion = fmt.Sprintf("docker buildx build --platform linux/%s -t %s .", hostArch, imageRef)
	}

	s.evaluateStorageStatus(result, result.EstimatedUncompressedMB, freeGB, totalGB)
	s.storeCache(imageRef, result, 24*time.Hour)
	return result, nil
}

// evaluateStorageStatus calcula el riesgo de almacenamiento proyectando capas y espacios de trabajo
func (s *ImageVerificationService) evaluateStorageStatus(result *domain.ImageVerificationResult, uncompressedMB int64, freeGB, totalGB float64) {
	// Proyección de 30 estudiantes con 512MB de capa escribible + tamaño base
	projectedHeadroomGB := float64(uncompressedMB)/1024.0 + (30.0 * 0.5)
	remainingFreeGB := freeGB - projectedHeadroomGB

	if remainingFreeGB < (totalGB*0.10) || remainingFreeGB < 5.0 {
		result.StorageStatus = domain.StorageStatusCriticalBlocked
		result.StorageMessage = fmt.Sprintf("Espacio en disco insuficiente: la imagen y proyección demandan %.1f GB, restando solo %.1f GB libres (<10%%)", projectedHeadroomGB, remainingFreeGB)
	} else if remainingFreeGB < (totalGB * 0.25) {
		result.StorageStatus = domain.StorageStatusWarning
		result.StorageMessage = fmt.Sprintf("Advertencia de almacenamiento: tras desplegar laboratorios restarán %.1f GB libres", remainingFreeGB)
	} else {
		result.StorageStatus = domain.StorageStatusOK
		result.StorageMessage = fmt.Sprintf("Capacidad de almacenamiento óptima: %.1f GB disponibles en el host", freeGB)
	}
}

// runSafeCapabilityProbe ejecuta un contenedor efímero aislado sin red ni privilegios para detectar runtime
func (s *ImageVerificationService) runSafeCapabilityProbe(ctx context.Context, imageRef string) *domain.CapabilitiesProbeResult {
	probeCmd := []string{
		"sh", "-c",
		"echo SOLV_PROBE_START; " +
			"for cmd in python3 node gcc g++ java rustc go bash; do " +
			"  if command -v $cmd >/dev/null 2>&1; then " +
			"    echo RUNTIME:$cmd; " +
			"  fi; " +
			"done",
	}

	hostConfig := &container.HostConfig{
		NetworkMode: "none",
		Resources: container.Resources{
			Memory: 256 * 1024 * 1024,
		},
		ReadonlyRootfs: true,
		SecurityOpt:    []string{"no-new-privileges:true"},
	}

	containerConfig := &container.Config{
		Image: imageRef,
		Cmd:   probeCmd,
	}

	resp, err := s.dockerCli.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, "")
	if err != nil {
		return &domain.CapabilitiesProbeResult{
			Success: false,
			Output:  fmt.Sprintf("Probe create error: %v", err),
		}
	}
	containerID := resp.ID
	defer func() {
		_ = s.dockerCli.ContainerRemove(context.Background(), containerID, container.RemoveOptions{Force: true})
	}()

	if err := s.dockerCli.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return &domain.CapabilitiesProbeResult{
			Success: false,
			Output:  fmt.Sprintf("Probe start error: %v", err),
		}
	}

	statusCh, errCh := s.dockerCli.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return &domain.CapabilitiesProbeResult{Success: false, Output: err.Error()}
		}
	case <-statusCh:
	case <-ctx.Done():
		return &domain.CapabilitiesProbeResult{Success: false, Output: "Probe timed out"}
	}

	logsReader, err := s.dockerCli.ContainerLogs(ctx, containerID, container.LogsOptions{ShowStdout: true, ShowStderr: false})
	if err != nil {
		return &domain.CapabilitiesProbeResult{Success: true}
	}
	defer logsReader.Close()

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, logsReader)
	rawOut := buf.String()

	var detected []string
	lines := strings.Split(rawOut, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if idx := strings.Index(trimmed, "RUNTIME:"); idx != -1 {
			runtimeName := strings.TrimSpace(trimmed[idx+8:])
			if runtimeName != "" {
				detected = append(detected, runtimeName)
			}
		}
	}

	return &domain.CapabilitiesProbeResult{
		Success:          true,
		DetectedRuntimes: detected,
		Output:           strings.Join(detected, ", "),
	}
}

func (s *ImageVerificationService) getHostDiskStats() (float64, float64, error) {
	if s.customStatfs != nil {
		return s.customStatfs(s.dockerRoot)
	}

	var stat syscall.Statfs_t
	paths := []string{s.dockerRoot, "/", "."}
	var err error

	for _, p := range paths {
		err = syscall.Statfs(p, &stat)
		if err == nil {
			totalGB := float64(stat.Blocks*uint64(stat.Bsize)) / (1024 * 1024 * 1024)
			freeGB := float64(stat.Bavail*uint64(stat.Bsize)) / (1024 * 1024 * 1024)
			return totalGB, freeGB, nil
		}
	}

	return 0, 0, err
}

func (s *ImageVerificationService) storeCache(key string, result *domain.ImageVerificationResult, ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cache[key] = cacheEntry{
		result:    result,
		expiresAt: time.Now().Add(ttl),
	}
}

func formatBytes(b int64) string {
	if b <= 0 {
		return "0 B"
	}
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func round2(val float64) float64 {
	return float64(int(val*100)) / 100.0
}
