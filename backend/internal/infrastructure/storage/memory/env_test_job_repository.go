package memory

import (
	"context"
	"errors"
	"sync"
	"time"

	"solv-backend/internal/core/domain"
)

var (
	ErrJobNotFound = errors.New("job de prueba de entorno no encontrado")
)

// EnvTestJobMemoryRepository implementa domain.EnvTestJobRepository en memoria
type EnvTestJobMemoryRepository struct {
	mu   sync.RWMutex
	jobs map[string]*domain.EnvTestJob
	ttl  time.Duration
}

// NewEnvTestJobMemoryRepository inicializa el repositorio en memoria
func NewEnvTestJobMemoryRepository(ttl time.Duration) *EnvTestJobMemoryRepository {
	if ttl <= 0 {
		ttl = 1 * time.Hour
	}
	repo := &EnvTestJobMemoryRepository{
		jobs: make(map[string]*domain.EnvTestJob),
		ttl:  ttl,
	}
	return repo
}

// cloneJob realiza una copia defensiva para evitar carreras de datos entre goroutines
func cloneJob(j *domain.EnvTestJob) *domain.EnvTestJob {
	if j == nil {
		return nil
	}
	cp := *j
	if len(j.Tools) > 0 {
		cp.Tools = make([]string, len(j.Tools))
		copy(cp.Tools, j.Tools)
	}
	if j.Result != nil {
		resCp := *j.Result
		if len(j.Result.Tools) > 0 {
			resCp.Tools = make([]domain.ToolResult, len(j.Result.Tools))
			copy(resCp.Tools, j.Result.Tools)
		}
		cp.Result = &resCp
	}
	if j.FinishedAt != nil {
		fin := *j.FinishedAt
		cp.FinishedAt = &fin
	}
	return &cp
}

func (r *EnvTestJobMemoryRepository) Save(ctx context.Context, job *domain.EnvTestJob) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	if job.CreatedAt.IsZero() {
		job.CreatedAt = now
	}
	job.UpdatedAt = now

	r.jobs[job.ID] = cloneJob(job)
	r.purgeExpiredLocked()
	return nil
}

func (r *EnvTestJobMemoryRepository) GetByID(ctx context.Context, id string) (*domain.EnvTestJob, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	job, exists := r.jobs[id]
	if !exists {
		return nil, ErrJobNotFound
	}
	return cloneJob(job), nil
}

func (r *EnvTestJobMemoryRepository) UpdateProgress(ctx context.Context, id string, progress domain.EnvTestProgress) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	job, exists := r.jobs[id]
	if !exists {
		return ErrJobNotFound
	}
	if job.IsTerminal() {
		return nil
	}

	job.Progress = progress
	job.Status = domain.EnvTestStatusPulling
	job.UpdatedAt = time.Now()
	return nil
}

func (r *EnvTestJobMemoryRepository) Complete(ctx context.Context, id string, result *domain.EnvTestResult) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	job, exists := r.jobs[id]
	if !exists {
		return ErrJobNotFound
	}
	if job.IsTerminal() {
		return nil
	}

	now := time.Now()
	job.Status = domain.EnvTestStatusSuccess
	job.Result = result
	job.FinishedAt = &now
	job.UpdatedAt = now
	return nil
}

func (r *EnvTestJobMemoryRepository) Fail(ctx context.Context, id string, errorCode, errorMessage string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	job, exists := r.jobs[id]
	if !exists {
		return ErrJobNotFound
	}
	if job.IsTerminal() {
		return nil
	}

	now := time.Now()
	job.Status = domain.EnvTestStatusFailed
	job.ErrorCode = errorCode
	job.ErrorMessage = errorMessage
	job.FinishedAt = &now
	job.UpdatedAt = now
	return nil
}

func (r *EnvTestJobMemoryRepository) Cancel(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	job, exists := r.jobs[id]
	if !exists {
		return ErrJobNotFound
	}
	if job.IsTerminal() {
		return nil
	}

	now := time.Now()
	job.Status = domain.EnvTestStatusCanceled
	job.FinishedAt = &now
	job.UpdatedAt = now
	return nil
}

func (r *EnvTestJobMemoryRepository) ListActive(ctx context.Context) ([]*domain.EnvTestJob, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var active []*domain.EnvTestJob
	for _, job := range r.jobs {
		if !job.IsTerminal() {
			active = append(active, cloneJob(job))
		}
	}
	return active, nil
}

func (r *EnvTestJobMemoryRepository) purgeExpiredLocked() {
	now := time.Now()
	for id, job := range r.jobs {
		if job.IsTerminal() && job.FinishedAt != nil {
			if now.Sub(*job.FinishedAt) > r.ttl {
				delete(r.jobs, id)
			}
		}
	}
}
