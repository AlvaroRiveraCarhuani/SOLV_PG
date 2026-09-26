package httpdelivery

import (
	"encoding/json"
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
	slug := r.URL.Query().Get("slug")
	tenantID := r.Header.Get("X-Tenant-Id")
	cacheKey := "default"
	if slug != "" {
		cacheKey = "slug:" + slug
	} else if tenantID != "" {
		cacheKey = "tenant:" + tenantID
	}

	// 1. Intentar leer de la caché
	if val, ok := h.cache.Load(cacheKey); ok {
		entry := val.(cacheEntry)
		if time.Now().Before(entry.expiresAt) {
			w.Header().Set("Content-Type", "application/json")
			w.Write(entry.data)
			return
		}
	}

	// 2. Consultar base de datos
	var tenant *domain.Tenant
	var err error
	if slug != "" {
		tenant, err = h.tenantRepo.GetBySlug(r.Context(), slug)
	} else if tenantID != "" {
		tenant, err = h.tenantRepo.GetByID(r.Context(), tenantID)
	} else {
		tenant, err = h.tenantRepo.GetByID(r.Context(), domain.DefaultTenantID)
	}

	if err != nil {
		http.Error(w, `{"error":"tenant_not_found","message":"Institución no encontrada"}`, http.StatusNotFound)
		return
	}

	// 3. Enriquecer configuración con tenant_id y slug
	var configMap map[string]interface{}
	if len(tenant.Config) > 0 {
		_ = json.Unmarshal(tenant.Config, &configMap)
	}
	if configMap == nil {
		configMap = make(map[string]interface{})
	}
	configMap["tenant_id"] = tenant.ID
	configMap["slug"] = tenant.Slug
	if _, ok := configMap["institution_name"]; !ok {
		configMap["institution_name"] = tenant.Name
	}

	data, err := json.Marshal(configMap)
	if err != nil {
		http.Error(w, `{"error":"serialization_error"}`, http.StatusInternalServerError)
		return
	}

	// 4. Guardar en caché
	h.cache.Store(cacheKey, cacheEntry{
		data:      data,
		expiresAt: time.Now().Add(h.cacheTTL),
	})

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}
