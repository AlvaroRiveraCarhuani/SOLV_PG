package services

import "errors"

// Errores de admisión que la capa HTTP traduce a 503. El aprovisionamiento de bases de
// datos de aula ya no vive aquí: desde ADR-051 cada motor, incluido PostgreSQL, se
// asegura como satélite dedicado del nodo (ver workspace_satellite.go).
var (
	ErrHostMemoryExhausted      = errors.New("host physical memory depleted: admission control denied request (HTTP 503)")
	ErrOOMKilledCooldownPenalty = errors.New("workspace reached 3 consecutive OOMKilled strikes: 5-minute cooldown penalty enforced")
	ErrMaxContainersReached     = errors.New("tenant concurrency quota exhausted: max_containers admission denied request (HTTP 503)")
)
