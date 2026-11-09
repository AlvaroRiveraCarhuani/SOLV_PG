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

// publicTenantConfig es la lista blanca de campos que puede consumir una
// sesión anónima. tenants.config contiene configuración interna (IA, QoS y
// credenciales cifradas); nunca se serializa completo en este endpoint.
type publicTenantConfig struct {
	TenantID           string `json:"tenant_id"`
	Slug               string `json:"slug"`
	InstitutionName    string `json:"institution_name"`
	LogoURL            string `json:"logo_url"`
	TenantPrimaryColor string `json:"tenant_primary_color,omitempty"`
	BaseDomain         string `json:"base_domain,omitempty"`
	SupportEmail       string `json:"support_email,omitempty"`
	FontSansFamily     string `json:"font_sans_family,omitempty"`
	FontMonoFamily     string `json:"font_mono_family,omitempty"`
}

func NewConfigHandler(tenantRepo domain.TenantRepository) *ConfigHandler {
	return &ConfigHandler{
		tenantRepo: tenantRepo,
		cacheTTL:   5 * time.Minute,
	}
}

func (h *ConfigHandler) GetPublicConfig(w http.ResponseWriter, r *http.Request) {
	slug := r.URL.Query().Get("slug")
	cacheKey := "default"
	if slug != "" {
		cacheKey = "slug:" + slug
	}

	// 1. Intentar leer de la caché
	if val, ok := h.cache.Load(cacheKey); ok {
		entry := val.(cacheEntry)
		if time.Now().Before(entry.expiresAt) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(entry.data)
			return
		}
	}

	// 2. Consultar base de datos
	var tenant *domain.Tenant
	var err error
	if slug != "" {
		tenant, err = h.tenantRepo.GetBySlug(r.Context(), slug)
	} else {
		tenant, err = h.tenantRepo.GetByID(r.Context(), domain.DefaultTenantID)
	}

	if err != nil {
		http.Error(w, `{"error":"tenant_not_found","message":"Institución no encontrada"}`, http.StatusNotFound)
		return
	}

	// 3. Copiar solo los campos públicos; el config interno no sale de aquí.
	var stored publicTenantConfig
	if len(tenant.Config) > 0 {
		_ = json.Unmarshal(tenant.Config, &stored)
	}
	public := publicTenantConfig{
		TenantID:           tenant.ID,
		Slug:               tenant.Slug,
		InstitutionName:    stored.InstitutionName,
		LogoURL:            stored.LogoURL,
		TenantPrimaryColor: stored.TenantPrimaryColor,
		BaseDomain:         stored.BaseDomain,
		SupportEmail:       stored.SupportEmail,
		FontSansFamily:     stored.FontSansFamily,
		FontMonoFamily:     stored.FontMonoFamily,
	}
	if public.InstitutionName == "" {
		public.InstitutionName = tenant.Name
	}

	data, err := json.Marshal(public)
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
	_, _ = w.Write(data)
}
