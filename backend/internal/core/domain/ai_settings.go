package domain

import (
	"errors"
	"net/url"
	"strings"
	"unicode/utf8"
)

// Proveedores del módulo de Inteligencia Artificial Híbrida (Nube + Servidor
// Local Universitario en LAN). La configuración se persiste por institución en
// tenants.config["ai_settings"].
const (
	AIProviderGemini     = "gemini"
	AIProviderOpenAI     = "openai"
	AIProviderOpenRouter = "openrouter"
	AIProviderDeepSeek   = "deepseek"
	AIProviderOllamaLAN  = "ollama_lan"
)

// BaseAIProvidersList contiene los IDs reservados del catálogo base oficial.
var BaseAIProvidersList = []string{
	AIProviderGemini,
	AIProviderOpenAI,
	AIProviderOpenRouter,
	AIProviderDeepSeek,
	AIProviderOllamaLAN,
}

// AIProviderLabels asigna nombres legibles canónicos a los proveedores base.
var AIProviderLabels = map[string]string{
	AIProviderGemini:     "Google Gemini",
	AIProviderOpenAI:     "OpenAI",
	AIProviderOpenRouter: "OpenRouter",
	AIProviderDeepSeek:   "DeepSeek",
	AIProviderOllamaLAN:  "Ollama LAN",
}

// AIProviderBaseURLs URLs base oficiales de los proveedores en la nube.
// Son fijas: el backend las fuerza y jamás acepta una URL enviada por el
// cliente para estos proveedores (escudo anti-SSRF).
var AIProviderBaseURLs = map[string]string{
	AIProviderGemini:     "https://generativelanguage.googleapis.com/v1beta/openai",
	AIProviderOpenAI:     "https://api.openai.com/v1",
	AIProviderOpenRouter: "https://openrouter.ai/api/v1",
	AIProviderDeepSeek:   "https://api.deepseek.com",
}

// Valores por defecto y rangos de la configuración de IA.
const (
	DefaultAIModel          = "gemini-2.5-flash"
	DefaultAIHourlyQuota    = 15
	MinAIHourlyQuota        = 1
	MaxAIHourlyQuota        = 200
	DefaultAITimeoutSeconds = 30
	MinAITimeoutSeconds     = 5
	MaxAITimeoutSeconds     = 120
	MaxCustomProviders      = 20
)

// Errores de negocio del módulo de IA.
var (
	// ErrAIQuotaExceeded se emite al superar hourly_quota_per_user dentro de la hora en curso.
	ErrAIQuotaExceeded = errors.New("se superó la cuota horaria de invocaciones de IA para este usuario")
	// ErrAIDisabled se emite al invocar el motor con la IA deshabilitada para el instituto.
	ErrAIDisabled = errors.New("la inteligencia artificial está deshabilitada para este instituto")
	// ErrAIMissingAPIKey es la regla de activación de proveedores en la nube (HTTP 400).
	ErrAIMissingAPIKey = errors.New("Se requiere una API Key para habilitar un proveedor en la nube")
	// ErrAIMissingModel es la regla de activación general: model no puede estar vacío si enabled=true.
	ErrAIMissingModel = errors.New("se requiere seleccionar un modelo para habilitar la inteligencia artificial")
)

// IsBaseAIProvider reporta si el proveedor pertenece a los 5 proveedores base inmutables.
func IsBaseAIProvider(provider string) bool {
	switch provider {
	case AIProviderGemini, AIProviderOpenAI, AIProviderOpenRouter, AIProviderDeepSeek, AIProviderOllamaLAN:
		return true
	}
	return false
}

// IsCustomAIProvider reporta si el proveedor sigue el formato custom_<slug>.
func IsCustomAIProvider(provider string) bool {
	if !strings.HasPrefix(provider, "custom_") {
		return false
	}
	slug := strings.TrimPrefix(provider, "custom_")
	if len(slug) < 1 || len(slug) > 50 {
		return false
	}
	for _, r := range slug {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-') {
			return false
		}
	}
	return true
}

// IsValidAIProvider reporta si el proveedor pertenece al catálogo base o es un custom válido.
func IsValidAIProvider(provider string) bool {
	return IsBaseAIProvider(provider) || IsCustomAIProvider(provider)
}

// AIProviderList devuelve los IDs de proveedores soportados en orden estable
// para mensajes de validación y documentación.
func AIProviderList() string {
	return "gemini, openai, openrouter, deepseek, ollama_lan (o custom_<slug>)"
}

// IsCloudAIProvider reporta si el proveedor corre en la nube oficial (API Key obligatoria
// y URL base oficial fija, nunca editable por el cliente).
func IsCloudAIProvider(provider string) bool {
	switch provider {
	case AIProviderGemini, AIProviderOpenAI, AIProviderOpenRouter, AIProviderDeepSeek:
		return true
	}
	return false
}

// OfficialAIBaseURL devuelve la URL base fija de un proveedor en la nube.
func OfficialAIBaseURL(provider string) (string, bool) {
	base, ok := AIProviderBaseURLs[provider]
	return base, ok
}

// AIProviderLabel devuelve el nombre legible para un proveedor (base o custom).
func AIProviderLabel(provider string, customProviders []AICustomProviderStored) string {
	if label, ok := AIProviderLabels[provider]; ok {
		return label
	}
	for _, cp := range customProviders {
		if cp.ID == provider {
			if strings.TrimSpace(cp.Label) != "" {
				return cp.Label
			}
			break
		}
	}
	if strings.HasPrefix(provider, "custom_") {
		slug := strings.TrimPrefix(provider, "custom_")
		if slug != "" {
			return slug
		}
	}
	return provider
}

// AIProviderProfileStored almacena el perfil persistido de un proveedor base en tenants.config.
type AIProviderProfileStored struct {
	Model           string `json:"model"`
	BaseURL         string `json:"base_url"`
	EncryptedAPIKey string `json:"encrypted_api_key,omitempty"`
	APIKeyMasked    string `json:"api_key_masked,omitempty"`
}

// AICustomProviderStored almacena un proveedor personalizado compatible con OpenAI en tenants.config.
type AICustomProviderStored struct {
	ID              string `json:"id"`
	Label           string `json:"label"`
	IconURL         string `json:"icon_url,omitempty"`
	BaseURL         string `json:"base_url"`
	Model           string `json:"model"`
	EncryptedAPIKey string `json:"encrypted_api_key,omitempty"`
	APIKeyMasked    string `json:"api_key_masked,omitempty"`
}

// AISettingsStored es la estructura persistida en tenants.config["ai_settings"].
// EncryptedAPIKey es base64(nonce + ciphertext) con AES-256-GCM; la clave en texto
// plano nunca se persiste ni se expone fuera del servicio.
type AISettingsStored struct {
	Enabled            bool                               `json:"enabled"`
	Provider           string                             `json:"provider"`              // "gemini" | "openai" | "openrouter" | "deepseek" | "ollama_lan" | custom_<slug>
	Model              string                             `json:"model"`                 // ej. "gemini-2.5-flash", "llama-3.3-70b-versatile", "qwen2.5-coder:7b"
	BaseURL            string                             `json:"base_url"`              // fija en cloud; validada contra SSRF en ollama_lan y custom_*
	EncryptedAPIKey    string                             `json:"encrypted_api_key,omitempty"`
	APIKeyMasked       string                             `json:"api_key_masked,omitempty"`
	HourlyQuotaPerUser int                                `json:"hourly_quota_per_user"` // 1..200 (default 15)
	TimeoutSeconds     int                                `json:"timeout_seconds"`       // 5..120 (default 30)
	UpdatedAt          string                             `json:"updated_at,omitempty"`
	ProviderProfiles   map[string]AIProviderProfileStored `json:"provider_profiles,omitempty"`
	CustomProviders    []AICustomProviderStored           `json:"custom_providers,omitempty"`
}

// DefaultAISettings valores vigentes cuando el tenant nunca configuró la IA.
// Ya no pre-asigna modelo ficticio: model inicia en "" hasta configuración real.
func DefaultAISettings() AISettingsStored {
	baseURL, _ := OfficialAIBaseURL(AIProviderGemini)
	profiles := make(map[string]AIProviderProfileStored)
	for _, p := range BaseAIProvidersList {
		bURL, _ := OfficialAIBaseURL(p)
		profiles[p] = AIProviderProfileStored{
			Model:   "",
			BaseURL: bURL,
		}
	}
	return AISettingsStored{
		Enabled:            false,
		Provider:           AIProviderGemini,
		Model:              "",
		BaseURL:            baseURL,
		HourlyQuotaPerUser: DefaultAIHourlyQuota,
		TimeoutSeconds:     DefaultAITimeoutSeconds,
		ProviderProfiles:   profiles,
		CustomProviders:    make([]AICustomProviderStored, 0),
	}
}

// AIProviderProfileDTO es la vista pública del perfil guardado de un proveedor base.
type AIProviderProfileDTO struct {
	Model        string `json:"model"`
	BaseURL      string `json:"base_url"`
	HasAPIKey    bool   `json:"has_api_key"`
	APIKeyMasked string `json:"api_key_masked"`
}

// AICustomProviderDTO es la vista pública y contrato de entrada de un proveedor custom.
type AICustomProviderDTO struct {
	ID           string `json:"id"`
	Label        string `json:"label"`
	IconURL      string `json:"icon_url,omitempty"`
	BaseURL      string `json:"base_url"`
	Model        string `json:"model"`
	APIKey       string `json:"api_key,omitempty"`
	ClearAPIKey  bool   `json:"clear_api_key,omitempty"`
	HasAPIKey    bool   `json:"has_api_key"`
	APIKeyMasked string `json:"api_key_masked"`
}

// AISettingsResponseDTO es la respuesta pública de administración.
// NUNCA incluye encrypted_api_key ni la clave en texto plano.
type AISettingsResponseDTO struct {
	Enabled            bool                            `json:"enabled"`
	Provider           string                          `json:"provider"`
	ProviderLabel      string                          `json:"provider_label"`
	Model              string                          `json:"model"`
	BaseURL            string                          `json:"base_url"`
	HasAPIKey          bool                            `json:"has_api_key"`
	APIKeyMasked       string                          `json:"api_key_masked"`
	HourlyQuotaPerUser int                             `json:"hourly_quota_per_user"`
	TimeoutSeconds     int                             `json:"timeout_seconds"`
	UpdatedAt          string                          `json:"updated_at"`
	ProviderProfiles   map[string]AIProviderProfileDTO `json:"provider_profiles"`
	CustomProviders    []AICustomProviderDTO           `json:"custom_providers"`
}

// AIStatusDTO es el estado liviano para GET /api/v1/ai/status.
type AIStatusDTO struct {
	Enabled       bool   `json:"enabled"`
	Provider      string `json:"provider"`
	ProviderLabel string `json:"provider_label"`
	Model         string `json:"model"`
}

// UpdateAISettingsDTO es el payload de PUT /api/v1/admin/ai/settings.
type UpdateAISettingsDTO struct {
	Enabled            bool                  `json:"enabled"`
	Provider           string                `json:"provider"`
	Model              string                `json:"model"`
	BaseURL            string                `json:"base_url"`
	APIKey             string                `json:"api_key"`
	ClearAPIKey        bool                  `json:"clear_api_key"`
	HourlyQuotaPerUser int                   `json:"hourly_quota_per_user"`
	TimeoutSeconds     int                   `json:"timeout_seconds"`
	CustomProviders    []AICustomProviderDTO `json:"custom_providers"`
}

// TestAIConnectionRequestDTO es el payload de POST /api/v1/admin/ai/test-connection.
type TestAIConnectionRequestDTO struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
	BaseURL  string `json:"base_url"`
	APIKey   string `json:"api_key"`
}

// TestAIConnectionResponseDTO es la respuesta de la prueba de conexión real.
type TestAIConnectionResponseDTO struct {
	OK        bool   `json:"ok"`
	Provider  string `json:"provider"`
	Model     string `json:"model"`
	LatencyMS int64  `json:"latency_ms"`
	Message   string `json:"message"`
}

// AIModelItemDTO representa un modelo descubierto dinámicamente desde el proveedor.
type AIModelItemDTO struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Description   string `json:"description,omitempty"`
	Tier          string `json:"tier"`          // "free" | "paid" | "local" | "by_account"
	PricingHint   string `json:"pricing_hint"` // ej. "Gratis", "De pago", "Local · Sin costo de API", "Según plan de la cuenta"
	ContextLength int    `json:"context_length,omitempty"`
}

// ListAIModelsRequestDTO es el payload de POST /api/v1/admin/ai/models.
type ListAIModelsRequestDTO struct {
	Provider string `json:"provider"`
	BaseURL  string `json:"base_url,omitempty"`
	APIKey   string `json:"api_key,omitempty"`
	Refresh  bool   `json:"refresh,omitempty"`
}

// ListAIModelsResponseDTO es la respuesta de POST /api/v1/admin/ai/models.
type ListAIModelsResponseDTO struct {
	Provider string           `json:"provider"`
	Cached   bool             `json:"cached"`
	Count    int              `json:"count"`
	Models   []AIModelItemDTO `json:"models"`
}

// NewAISettingsResponseDTO convierte el estado persistido en la respuesta pública.
func NewAISettingsResponseDTO(stored AISettingsStored) AISettingsResponseDTO {
	profilesDTO := make(map[string]AIProviderProfileDTO, len(BaseAIProvidersList))
	for _, p := range BaseAIProvidersList {
		prof, ok := stored.ProviderProfiles[p]
		if !ok {
			offURL, _ := OfficialAIBaseURL(p)
			prof = AIProviderProfileStored{
				Model:   "",
				BaseURL: offURL,
			}
		}
		bURL := prof.BaseURL
		if IsCloudAIProvider(p) {
			offURL, _ := OfficialAIBaseURL(p)
			bURL = offURL
		}
		profilesDTO[p] = AIProviderProfileDTO{
			Model:        prof.Model,
			BaseURL:      bURL,
			HasAPIKey:    strings.TrimSpace(prof.EncryptedAPIKey) != "",
			APIKeyMasked: prof.APIKeyMasked,
		}
	}

	customDTOs := make([]AICustomProviderDTO, 0, len(stored.CustomProviders))
	for _, cp := range stored.CustomProviders {
		customDTOs = append(customDTOs, AICustomProviderDTO{
			ID:           cp.ID,
			Label:        cp.Label,
			IconURL:      cp.IconURL,
			BaseURL:      cp.BaseURL,
			Model:        cp.Model,
			HasAPIKey:    strings.TrimSpace(cp.EncryptedAPIKey) != "",
			APIKeyMasked: cp.APIKeyMasked,
		})
	}

	return AISettingsResponseDTO{
		Enabled:            stored.Enabled,
		Provider:           stored.Provider,
		ProviderLabel:      AIProviderLabel(stored.Provider, stored.CustomProviders),
		Model:              stored.Model,
		BaseURL:            stored.BaseURL,
		HasAPIKey:          strings.TrimSpace(stored.EncryptedAPIKey) != "",
		APIKeyMasked:       stored.APIKeyMasked,
		HourlyQuotaPerUser: stored.HourlyQuotaPerUser,
		TimeoutSeconds:     stored.TimeoutSeconds,
		UpdatedAt:          stored.UpdatedAt,
		ProviderProfiles:   profilesDTO,
		CustomProviders:    customDTOs,
	}
}

// NewAIStatusDTO convierte el estado persistido en el estado liviano.
func NewAIStatusDTO(stored AISettingsStored) AIStatusDTO {
	return AIStatusDTO{
		Enabled:       stored.Enabled,
		Provider:      stored.Provider,
		ProviderLabel: AIProviderLabel(stored.Provider, stored.CustomProviders),
		Model:         stored.Model,
	}
}

// ValidateCustomProviderURL valida que icon_url sea una URL http/https de hasta 500 caracteres.
func ValidateCustomProviderIconURL(iconURL string) bool {
	trimmed := strings.TrimSpace(iconURL)
	if trimmed == "" {
		return true
	}
	if len(trimmed) > 500 {
		return false
	}
	u, err := url.Parse(trimmed)
	if err != nil || u.Host == "" {
		return false
	}
	scheme := strings.ToLower(u.Scheme)
	return scheme == "http" || scheme == "https"
}

// ValidateCustomProviderLabel valida que el label tenga entre 1 y 60 caracteres.
func ValidateCustomProviderLabel(label string) bool {
	n := utf8.RuneCountInString(strings.TrimSpace(label))
	return n >= 1 && n <= 60
}
