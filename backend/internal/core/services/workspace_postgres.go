package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/jmoiron/sqlx"
)

var (
	ErrHostMemoryExhausted      = errors.New("host physical memory depleted: admission control denied request (HTTP 503)")
	ErrOOMKilledCooldownPenalty = errors.New("workspace reached 3 consecutive OOMKilled strikes: 5-minute cooldown penalty enforced")
	ErrMaxContainersReached     = errors.New("tenant concurrency quota exhausted: max_containers admission denied request (HTTP 503)")
)

func ensureSatellitePostgresDatabase(ctx context.Context, db *sqlx.DB, dbName string) error {
	for _, ch := range dbName {
		if (ch < 'a' || ch > 'z') && (ch < '0' || ch > '9') && ch != '_' {
			return fmt.Errorf("invalid characters in database name: %s", dbName)
		}
	}
	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)`
	if err := db.GetContext(ctx, &exists, checkQuery, dbName); err != nil {
		return fmt.Errorf("failed to check database existence: %w", err)
	}
	if !exists {
		createQuery := fmt.Sprintf(`CREATE DATABASE "%s"`, dbName)
		if _, err := db.ExecContext(ctx, createQuery); err != nil {
			if !strings.Contains(err.Error(), "already exists") {
				return fmt.Errorf("failed to create satellite database %s: %w", dbName, err)
			}
		}
		log.Printf("[Satellite DB] Created new isolated multi-tenant database: %s", dbName)
	}
	return nil
}

func getPostgresContainerHost() string {
	h := strings.TrimSpace(os.Getenv("POSTGRES_CONTAINER_HOST"))
	if h != "" {
		return h
	}
	if insideDocker := os.Getenv("INSIDE_DOCKER"); insideDocker == "true" {
		return "solv-postgres"
	}
	return "host.docker.internal"
}
