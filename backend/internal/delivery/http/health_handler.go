package httpdelivery

import (
	"context"
	"net/http"
	"time"
)

// HealthHandler es el chequeo que consulta un vigilante externo cada pocos
// minutos. A diferencia de /api/v1/version, responde 503 cuando la base no
// contesta: un servicio de vigilancia entiende códigos HTTP, no cuerpos JSON.
type HealthHandler struct {
	ping func(ctx context.Context) error
}

type healthResponse struct {
	Status   string `json:"status"`
	Database string `json:"database,omitempty"`
}

func NewHealthHandler(ping func(ctx context.Context) error) *HealthHandler {
	return &HealthHandler{ping: ping}
}

// GetHealth responde 200 si la API y su base están disponibles, y 503 si no.
// El tiempo límite es corto a propósito: un chequeo que tarda más que el
// intervalo del vigilante produce una caída falsa.
func (h *HealthHandler) GetHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	if h.ping != nil {
		if err := h.ping(ctx); err != nil {
			SendJSON(w, http.StatusServiceUnavailable, healthResponse{
				Status:   "degraded",
				Database: "unavailable",
			}, "La base de datos no responde")
			return
		}
	}

	SendJSON(w, http.StatusOK, healthResponse{Status: "ok", Database: "ok"}, "Sistema disponible")
}
