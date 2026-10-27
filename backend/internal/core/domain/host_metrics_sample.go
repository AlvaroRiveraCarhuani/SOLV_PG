package domain

import (
	"context"
	"time"
)

// Constantes de muestreo del host. Viven en el dominio porque las comparten el
// worker que escribe (core/services), el handler que lee (delivery/http) y el
// repositorio que persiste (infrastructure/storage/postgres).
const (
	// HostMetricsSampleRate es la frecuencia de muestreo: 1 fila por minuto.
	HostMetricsSampleRate = time.Minute
	// HostMetricsRetentionWindow conserva 7 dias de historial (D1, ADR-049).
	HostMetricsRetentionWindow = 7 * 24 * time.Hour
	// HostMetricsHistoryWindowMax limita la ventana consultable del panel a la
	// retencion vigente: preguntar por mas historia que la que se conserva no
	// tiene sentido y forzaria un barrido inutil.
	HostMetricsHistoryWindowMax = 10080 * time.Minute // 7 dias
)

// HostMetricsSample es una fila del historial persistido de telemetria del
// nodo. SampledAt es el momento real de la medicion, no la hora de insercion.
type HostMetricsSample struct {
	ID               int64     `db:"id" json:"id"`
	SampledAt        time.Time `db:"sampled_at" json:"sampled_at"`
	RAMPercent       float64   `db:"ram_percent" json:"ram_percent"`
	RAMUsedGB        float64   `db:"ram_used_gb" json:"ram_used_gb"`
	CPUPercent       float64   `db:"cpu_percent" json:"cpu_percent"`
	ActiveContainers int       `db:"active_containers" json:"active_containers"`
	TenantID         string    `db:"tenant_id" json:"tenant_id"`
}

// HostMetricsSampleStore persiste y lee el historial muestreado del host. Es la
// unica puerta entre el worker/panel y la base: nadie en core ni en delivery
// construye SQL sobre esta tabla.
type HostMetricsSampleStore interface {
	// Insert guarda una muestra; el ID lo genera la base.
	Insert(ctx context.Context, sample *HostMetricsSample) error
	// ListSince devuelve las muestras desde la marca dada, en orden cronologico
	// ascendente, con tope de filas para acotar 24h (1440 filas).
	ListSince(ctx context.Context, tenantID string, since time.Time, limit int) ([]HostMetricsSample, error)
	// PruneBefore elimina las muestras anteriores a la retencion y devuelve
	// cuantas filas se borraron (0 si no hay nada que podar).
	PruneBefore(ctx context.Context, olderThan time.Time) (int64, error)
}