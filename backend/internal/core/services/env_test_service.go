package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"solv-backend/internal/core/domain"
)

var (
	ErrJobNotFound = errors.New("job de prueba de entorno no encontrado")
	ErrJobAlreadyFinished = errors.New("el job ya se encuentra en un estado terminal")
)

// EnvTestConfig parámetros inyectables de ejecución para la prueba de entorno (DA-06)
type EnvTestConfig struct {
	MaxConcurrentJobs int
	TotalJobTimeout   time.Duration
	MemoryLimitMB     int64
}

// EnvTestService coordina la ejecución asíncrona de pruebas de entorno sin acoplarse a Docker ni HTTP
type EnvTestService struct {
	repo     domain.EnvTestJobRepository
	registry domain.ImageRegistryPort
	runner   domain.ContainerRunnerPort
	config   EnvTestConfig

	sem         chan struct{}
	cancelMu    sync.Mutex
	cancelFuncs map[string]context.CancelFunc
}

// NewEnvTestService crea una nueva instancia del servicio de prueba de entorno
func NewEnvTestService(
	repo domain.EnvTestJobRepository,
	registry domain.ImageRegistryPort,
	runner domain.ContainerRunnerPort,
	config EnvTestConfig,
) *EnvTestService {
	if config.MaxConcurrentJobs <= 0 {
		config.MaxConcurrentJobs = 2
	}
	if config.TotalJobTimeout <= 0 {
		config.TotalJobTimeout = 15 * time.Minute
	}
	if config.MemoryLimitMB <= 0 {
		config.MemoryLimitMB = 256
	}

	return &EnvTestService{
		repo:        repo,
		registry:    registry,
		runner:      runner,
		config:      config,
		sem:         make(chan struct{}, config.MaxConcurrentJobs),
		cancelFuncs: make(map[string]context.CancelFunc),
	}
}

// StartJob valida la solicitud, persiste el job como pending y despacha la ejecución en background
func (s *EnvTestService) StartJob(ctx context.Context, req domain.StartEnvTestRequest) (*domain.EnvTestJob, error) {
	imageRef := strings.TrimSpace(req.Image)
	if imageRef == "" {
		return nil, errors.New("la imagen docker es requerida")
	}

	// Validar que no tenga tag :latest
	if strings.HasSuffix(imageRef, ":latest") {
		return nil, errors.New("el tag :latest no está permitido por reproducibilidad y gobernanza")
	}

	// Validar y sanear herramientas requeridas
	var cleanTools []string
	for _, tool := range req.Tools {
		trimmed := strings.TrimSpace(tool)
		if trimmed == "" {
			continue
		}
		if err := domain.ValidateToolName(trimmed); err != nil {
			return nil, fmt.Errorf("herramienta %q inválida: %w", trimmed, err)
		}
		cleanTools = append(cleanTools, trimmed)
	}

	jobID := uuid.New().String()
	job := &domain.EnvTestJob{
		ID:     jobID,
		Image:  imageRef,
		Tools:  cleanTools,
		Status: domain.EnvTestStatusPending,
		Progress: domain.EnvTestProgress{
			CurrentAction: "Encolado, esperando slot de ejecución...",
		},
	}

	if err := s.repo.Save(ctx, job); err != nil {
		return nil, fmt.Errorf("error al registrar job: %w", err)
	}

	// Contexto de ciclo de vida completo del job con timeout global (DA-02)
	jobCtx, cancel := context.WithTimeout(context.Background(), s.config.TotalJobTimeout)

	s.cancelMu.Lock()
	s.cancelFuncs[jobID] = cancel
	s.cancelMu.Unlock()

	// Lanzar ejecución asíncrona
	go s.executeJob(jobCtx, jobID, imageRef, cleanTools)

	return job, nil
}

// GetJob obtiene el estado actual y progreso del job por su identificador
func (s *EnvTestService) GetJob(ctx context.Context, jobID string) (*domain.EnvTestJob, error) {
	return s.repo.GetByID(ctx, jobID)
}

// CancelJob solicita la cancelación inmediata del job en ejecución
func (s *EnvTestService) CancelJob(ctx context.Context, jobID string) error {
	s.cancelMu.Lock()
	cancel, exists := s.cancelFuncs[jobID]
	if exists {
		cancel()
		delete(s.cancelFuncs, jobID)
	}
	s.cancelMu.Unlock()

	return s.repo.Cancel(ctx, jobID)
}

// executeJob maneja el ciclo de vida, semáforo y lógica de degradación de la prueba
func (s *EnvTestService) executeJob(ctx context.Context, jobID, imageRef string, tools []string) {
	// Limpieza de función cancel al terminar
	defer func() {
		s.cancelMu.Lock()
		delete(s.cancelFuncs, jobID)
		s.cancelMu.Unlock()
	}()

	// Adquisición de semáforo de concurrencia
	select {
	case s.sem <- struct{}{}:
		// Slot adquirido
		defer func() {
			<-s.sem // Invariante: liberación garantizada en cualquier camino terminal
		}()
	case <-ctx.Done():
		_ = s.repo.Cancel(context.Background(), jobID)
		return
	}

	// 1. Degradación de Digest en 3 niveles (DA-03)
	isLocal, _, localDigest, err := s.registry.InspectLocal(ctx, imageRef)
	if err != nil && ctx.Err() != nil {
		_ = s.repo.Cancel(context.Background(), jobID)
		return
	}

	remoteDigest, distErr := s.registry.InspectRemoteDigest(ctx, imageRef)

	digestUnverified := false
	needPull := false

	if distErr != nil {
		if isLocal {
			// Nivel B: Fallo de red/registro pero existe imagen local -> proceder con advertencia
			digestUnverified = true
			needPull = false
		} else {
			// Nivel C: Fallo de red/registro y no está local -> Error terminal
			_ = s.repo.Fail(context.Background(), jobID, domain.EnvTestErrRegistryUnreach, "No es posible acceder al registro de imágenes para verificar el manifiesto")
			return
		}
	} else {
		// Nivel A: Registro accesible
		if !isLocal {
			needPull = true
		} else if localDigest != "" && remoteDigest != "" && localDigest != remoteDigest {
			needPull = true
		}
	}

	// 2. Descarga si es requerida
	if needPull {
		_ = s.repo.UpdateProgress(context.Background(), jobID, domain.EnvTestProgress{
			CurrentAction: "Iniciando descarga de capas...",
		})

		pullErr := s.registry.PullImage(ctx, imageRef, func(doneBytes, totalBytes int64, currentLayer, totalLayers int, action string) {
			var pct float64
			if totalBytes > 0 {
				pct = float64(doneBytes) / float64(totalBytes) * 100
				if pct > 100 {
					pct = 100
				}
			}
			_ = s.repo.UpdateProgress(context.Background(), jobID, domain.EnvTestProgress{
				BytesDone:     doneBytes,
				BytesTotal:    totalBytes,
				LayerCurrent:  currentLayer,
				LayersTotal:   totalLayers,
				Percent:       pct,
				CurrentAction: action,
			})
		})

		if pullErr != nil {
			if errors.Is(pullErr, context.Canceled) {
				_ = s.repo.Cancel(context.Background(), jobID)
				return
			}
			if strings.Contains(pullErr.Error(), domain.EnvTestErrPullStalled) {
				_ = s.repo.Fail(context.Background(), jobID, domain.EnvTestErrPullStalled, "Descarga detenida: sin actividad de red por más de 60 segundos")
				return
			}
			_ = s.repo.Fail(context.Background(), jobID, domain.EnvTestErrPullTimeout, fmt.Sprintf("Error durante la descarga de la imagen: %v", pullErr))
			return
		}
	}

	// 3. Ejecución de Smoke Test en Contenedor Efímero Aislado
	_ = s.repo.UpdateProgress(context.Background(), jobID, domain.EnvTestProgress{
		CurrentAction: "Ejecutando smoke test en contenedor efímero...",
	})

	startTime := time.Now()
	results, exitCode, runnerErr := s.runner.RunSmokeTest(ctx, imageRef, tools, s.config.MemoryLimitMB)
	durationMs := time.Since(startTime).Milliseconds()

	if runnerErr != nil {
		if errors.Is(runnerErr, context.Canceled) {
			_ = s.repo.Cancel(context.Background(), jobID)
			return
		}
		if strings.Contains(runnerErr.Error(), domain.EnvTestErrTestOOM) || exitCode == 137 {
			_ = s.repo.Fail(context.Background(), jobID, domain.EnvTestErrTestOOM, "El contenedor superó el límite de memoria asignado durante la prueba")
			return
		}
		_ = s.repo.Fail(context.Background(), jobID, domain.EnvTestErrTestCrash, fmt.Sprintf("Fallo en la ejecución del contenedor de prueba: %v", runnerErr))
		return
	}

	testResult := &domain.EnvTestResult{
		Tools:      results,
		ExitCode:   exitCode,
		DurationMs: durationMs,
	}

	// Si alguna herramienta requerida faltó o el exit code fue distinto de cero, marcamos el job como fallido
	hasMissing := false
	for _, res := range results {
		if !res.Present {
			hasMissing = true
			break
		}
	}

	if hasMissing || exitCode != 0 {
		_ = s.repo.Fail(context.Background(), jobID, domain.EnvTestErrTestCrash, "Una o más herramientas requeridas no se encuentran instaladas en la imagen")
		// Adosar el resultado para que el frontend pueda mostrar el desglose por herramienta
		job, _ := s.repo.GetByID(context.Background(), jobID)
		if job != nil {
			job.Result = testResult
			job.DigestUnverified = digestUnverified
			_ = s.repo.Save(context.Background(), job)
		}
		return
	}

	// Éxito completo
	_ = s.repo.Complete(context.Background(), jobID, testResult)
	if digestUnverified {
		job, _ := s.repo.GetByID(context.Background(), jobID)
		if job != nil {
			job.DigestUnverified = true
			_ = s.repo.Save(context.Background(), job)
		}
	}
}
