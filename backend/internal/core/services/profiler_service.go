package services

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"

	"solv-backend/internal/core/domain"
)

// CalculateP95Profile calcula empíricamente el límite de RAM de una imagen (en MB).
func CalculateP95Profile(ctx context.Context, dockerCli domain.ContainerOrchestrator, image string, samples int) (int, error) {
	if samples <= 0 {
		return 0, fmt.Errorf("samples must be greater than 0")
	}

	results := make(chan int64, samples)
	sem := make(chan struct{}, 2) // Max 2 concurrent dry-runs
	var wg sync.WaitGroup

	for i := 0; i < samples; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}        // Acquire semaphore
			defer func() { <-sem }() // Release semaphore

			maxRAM, err := dockerCli.ExecuteDryRun(ctx, image)
			if err == nil && maxRAM > 0 {
				results <- maxRAM
			}
		}()
	}

	wg.Wait()
	close(results)

	var validSamples []int64
	for r := range results {
		validSamples = append(validSamples, r)
	}

	if len(validSamples) == 0 {
		return 0, fmt.Errorf("all dry-runs failed to collect memory usage")
	}

	// Sort samples
	sort.Slice(validSamples, func(i, j int) bool { return validSamples[i] < validSamples[j] })

	// Calculate P95
	p95Index := int(math.Ceil(float64(len(validSamples))*0.95)) - 1
	if p95Index < 0 {
		p95Index = 0
	}
	p95Bytes := validSamples[p95Index]

	// Convert bytes to MB
	p95MB := int(math.Ceil(float64(p95Bytes) / (1024 * 1024)))
	return p95MB, nil
}
