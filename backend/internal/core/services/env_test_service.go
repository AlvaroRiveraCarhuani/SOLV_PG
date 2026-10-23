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
	ErrJobNotFound        = errors.New("job de prueba de entorno no encontrado")
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
	builder  domain.ImageBuilderPort
	config   EnvTestConfig

	sem         chan struct{}
	cancelMu    sync.Mutex
	cancelFuncs map[string]context.CancelFunc
}

func (s *EnvTestService) SetImageBuilder(builder domain.ImageBuilderPort) {
	s.builder = builder
}

func (s *EnvTestService) PruneOrphanStackImages(ctx context.Context, activeTags map[string]bool) ([]string, error) {
	if s.builder == nil {
		return nil, nil
	}
	return s.builder.PruneOrphanStackImages(ctx, activeTags)
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

	targetEnv := strings.TrimSpace(req.TargetEnvironment)
	if targetEnv == "" {
		targetEnv = "IDE_PERSISTENTE"
	}
	timeoutMS := req.TimeoutMS
	if timeoutMS <= 0 {
		timeoutMS = 5000
	}

	jobID := uuid.New().String()
	job := &domain.EnvTestJob{
		ID:                jobID,
		Image:             imageRef,
		Tools:             cleanTools,
		TargetEnvironment: targetEnv,
		Entrypoint:        strings.TrimSpace(req.Entrypoint),
		TimeoutMS:         timeoutMS,
		SampleInput:       req.SampleInput,
		Status:            domain.EnvTestStatusPending,
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
	// Ejecucion asincrona del job con timeout propio, desacoplada del request HTTP.
	//nolint:gosec
	go s.executeJob(jobCtx, jobID, imageRef, strings.TrimSpace(req.DockerfileContent), cleanTools, targetEnv, strings.TrimSpace(req.Entrypoint), req.SampleInput, timeoutMS, req.BaseRamMB)

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
