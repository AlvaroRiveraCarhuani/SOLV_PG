package httpdelivery

import (
	"net/http"
	"sync"
	"time"

	"solv-backend/internal/core/domain"
)

type ConfigHandler struct {
	tenantRepo domain.TenantRepository
	cache      sync.Map
	cacheTTL   time.Duration
}

type cacheEntry struct {
	data      []byte
	expiresAt time.Time
}

func NewConfigHandler(tenantRepo domain.TenantRepository) *ConfigHandler {
	return &ConfigHandler{
		tenantRepo: tenantRepo,
		cacheTTL:   5 * time.Minute,
	}
}

func (h *ConfigHandler) GetPublicConfig(w http.ResponseWriter, r *http.Request) {
	// UAB Tenant por defecto
	const defaultTenantID = "00000000-0000-0000-0000-000000000001"

	// 1. Intentar leer de la caché
	if val, ok := h.cache.Load(defaultTenantID); ok {
		entry := val.(cacheEntry)
		if time.Now().Before(entry.expiresAt) {
			w.Header().Set("Content-Type", "application/json")
			w.Write(entry.data)
			return
		}
	}

	// 2. Consultar base de datos
	tenant, err := h.tenantRepo.GetByID(r.Context(), defaultTenantID)
	if err != nil {
		http.Error(w, "tenant not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(tenant.Config)

	// 3. Guardar en caché
	h.cache.Store(defaultTenantID, cacheEntry{
		data:      tenant.Config,
		expiresAt: time.Now().Add(h.cacheTTL),
	})
}
