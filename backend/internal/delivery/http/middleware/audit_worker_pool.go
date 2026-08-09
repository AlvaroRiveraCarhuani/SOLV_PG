package middleware

import (
	"context"
	"log"
	"sync"

	"solv-backend/internal/core/domain"
)

type AuditWorkerPool struct {
	repo    domain.AuditLogRepository
	queue   chan *domain.AuditLog
	workers int
	wg      sync.WaitGroup
}

func NewAuditWorkerPool(repo domain.AuditLogRepository, bufferSize int, workerCount int) *AuditWorkerPool {
	if bufferSize <= 0 {
		bufferSize = 1000
	}
	if workerCount <= 0 {
		workerCount = 5
	}

	pool := &AuditWorkerPool{
		repo:    repo,
		queue:   make(chan *domain.AuditLog, bufferSize),
		workers: workerCount,
	}

	for i := 0; i < workerCount; i++ {
		pool.wg.Add(1)
		go pool.workerLoop(i)
	}

	return pool
}

func (p *AuditWorkerPool) workerLoop(workerID int) {
	defer p.wg.Done()
	for logEntry := range p.queue {
		ctx := context.Background()
		if err := p.repo.Create(ctx, logEntry); err != nil {
			log.Printf("[AuditWorkerPool #%d] Error persisting audit log (action=%s, actor=%s): %v",
				workerID, logEntry.Action, logEntry.ActorID, err)
		}
	}
}

func (p *AuditWorkerPool) Enqueue(logEntry *domain.AuditLog) {
	select {
	case p.queue <- logEntry:
	default:
		log.Printf("[AuditWorkerPool] WARNING: Queue buffer full (%d slots). Dropping audit log event (action=%s, actor=%s)",
			cap(p.queue), logEntry.Action, logEntry.ActorID)
	}
}

func (p *AuditWorkerPool) Shutdown() {
	close(p.queue)
	p.wg.Wait()
}
