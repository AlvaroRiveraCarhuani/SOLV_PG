package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"sync"
	"time"

	"solv-backend/internal/core/domain"
)

type EWMAProfilerServiceImpl struct {
	repo domain.LabTemplateRepository
	mu   sync.Mutex
	locks map[string]*sync.Mutex
}

func NewEWMAProfilerService(repo domain.LabTemplateRepository) domain.EWMAProfilerService {
	return &EWMAProfilerServiceImpl{
		repo:  repo,
		locks: make(map[string]*sync.Mutex),
	}
}

func (s *EWMAProfilerServiceImpl) getLockForSignature(sig string) *sync.Mutex {
	s.mu.Lock()
	defer s.mu.Unlock()

	l, exists := s.locks[sig]
	if !exists {
		l = &sync.Mutex{}
		s.locks[sig] = l
	}
	return l
}

func (s *EWMAProfilerServiceImpl) CalculateSignatureHash(baseImage string, setupScript string) string {
	hasher := sha256.New()
	hasher.Write([]byte(baseImage + "::" + setupScript))
	return hex.EncodeToString(hasher.Sum(nil))
}

func (s *EWMAProfilerServiceImpl) RecordSessionPeakAndRecalculate(ctx context.Context, baseImage string, setupScript string, peakRAMMB float64) (*domain.ResourceProfile, error) {
	sigHash := s.CalculateSignatureHash(baseImage, setupScript)

	// Concurrencia Nivel 1: Bloqueo en memoria por firma para ordenamiento local
	sigLock := s.getLockForSignature(sigHash)
	sigLock.Lock()
	defer sigLock.Unlock()

	// Verificar si la plantilla existe; si no, crearla con perfil por defecto
	existing, err := s.repo.GetBySignatureHash(ctx, sigHash)
	if err != nil {
		return nil, fmt.Errorf("error querying lab template profile: %w", err)
	}

	if existing == nil {
		newTemplate := &domain.LabTemplate{
			ID:            sigHash,
			Name:          fmt.Sprintf("Template-%s", sigHash[:8]),
			BaseImage:     baseImage,
			SetupScript:   setupScript,
			SignatureHash: sigHash,
			ResourceProfile: domain.ResourceProfile{
				SignatureHash: sigHash,
				BaseMemoryMB:  domain.DefaultBaseMemoryMB,
				MaxQuotaMB:    domain.HardQuotaMemoryMB,
				EWMAState: domain.EWMAState{
					CurrentEWMAMB: peakRAMMB,
					SampleCount:   1,
					LastUpdatedAt: time.Now(),
				},
			},
		}

		if err := s.repo.CreateOrUpdateProfile(ctx, newTemplate); err != nil {
			return nil, fmt.Errorf("failed to create initial lab template profile: %w", err)
		}
		log.Printf("[EWMA Profiler] Initialized new template profile %s with initial peak %.2f MB", sigHash[:8], peakRAMMB)
		return &newTemplate.ResourceProfile, nil
	}

	// Concurrencia Nivel 2: Transacción Atómica SELECT FOR UPDATE en PostgreSQL
	updatedTemplate, err := s.repo.UpdateProfileAtomic(ctx, sigHash, peakRAMMB)
	if err != nil {
		return nil, fmt.Errorf("failed to perform atomic EWMA update: %w", err)
	}

	log.Printf("[EWMA Profiler] Recalculated template %s: Peak=%.2f MB -> New EWMA=%.2f MB -> New Max Quota=%d MB (Samples: %d)",
		sigHash[:8], peakRAMMB, updatedTemplate.ResourceProfile.EWMAState.CurrentEWMAMB, updatedTemplate.ResourceProfile.MaxQuotaMB, updatedTemplate.ResourceProfile.EWMAState.SampleCount)

	return &updatedTemplate.ResourceProfile, nil
}
