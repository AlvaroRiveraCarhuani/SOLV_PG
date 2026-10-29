package services

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"solv-backend/internal/core/domain"
)

const aiSettingsConfigKey = "ai_settings"

// defaultAISecret se usa cuando JWT_SECRET está vacío (entornos de test/CI).
const defaultAISecret = "solv-ai-settings-dev-secret"

// Límite de lectura de la respuesta del motor de IA (2 MiB).
const aiMaxResponseBytes = 2 << 20

// AISettingsService gestiona la configuración de Inteligencia Artificial Híbrida
// (Nube + Servidor Local Universitario en LAN + Proveedores Custom) por institución,
// persistida en tenants.config["ai_settings"]. Ofrece cifrado AES-256-GCM de la API Key,
// escudo anti-SSRF para servidores locales y custom, cuota por hora por usuario y un
// cliente HTTP OpenAI-compatible para JSON estructurado.
type AISettingsService struct {
	tenantRepo domain.TenantRepository
	secret     []byte
	client     *http.Client

	quotaMu     sync.Mutex
	quotas      map[string]*aiQuotaUsage
	modelsMu    sync.RWMutex
	modelsCache map[string]aiModelsCacheEntry
	nowFn       func() time.Time
}

// aiModelsCacheEntry almacena en memoria los modelos descubiertos con TTL de 5m.
type aiModelsCacheEntry struct {
	models    []domain.AIModelItemDTO
	expiresAt time.Time
}

// aiQuotaUsage ventana de una hora por (tenant, usuario).
type aiQuotaUsage struct {
	windowStart time.Time
	count       int
}

// NewAISettingsService deriva la clave AES-256 con sha256(JWT_SECRET); si la
// variable está vacía usa un fallback determinista (solo entornos de test/CI).
func NewAISettingsService(tenantRepo domain.TenantRepository) *AISettingsService {
	secret := strings.Trim(strings.TrimSpace(os.Getenv("JWT_SECRET")), `"`)
	if secret == "" {
		secret = defaultAISecret
	}
	key := sha256.Sum256([]byte(secret))
	return &AISettingsService{
		tenantRepo:  tenantRepo,
		secret:      key[:],
		client:      &http.Client{},
		quotas:      make(map[string]*aiQuotaUsage),
		modelsCache: make(map[string]aiModelsCacheEntry),
		nowFn:       time.Now,
	}
}

// WithTransport inyecta un http.RoundTripper personalizable (tests unitarios).
func (s *AISettingsService) WithTransport(rt http.RoundTripper) *AISettingsService {
	s.client = &http.Client{Transport: rt}
	return s
}

// Get devuelve la configuración vigente del tenant o los defaults si el tenant
// nunca configuró la IA. Los valores fuera de rango o inválidos se sanean.
func (s *AISettingsService) Get(ctx context.Context, tenantID string) (*domain.AISettingsStored, error) {
	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil || tenant == nil {
		return nil, fmt.Errorf("tenant not found")
	}

	defaults := domain.DefaultAISettings()
	if len(tenant.Config) == 0 {
		return &defaults, nil
	}

	var configMap map[string]interface{}
	if err := json.Unmarshal(tenant.Config, &configMap); err != nil {
		return &defaults, nil
	}
	raw, ok := configMap[aiSettingsConfigKey]
	if !ok {
		return &defaults, nil
	}
	rawBytes, err := json.Marshal(raw)
	if err != nil {
		return &defaults, nil
	}
	stored := defaults
	if err := json.Unmarshal(rawBytes, &stored); err != nil {
		return &defaults, nil
	}

	// Saneo defensivo: el catálogo de proveedores y las URLs oficiales son la
	// fuente de verdad, no el JSON almacenado.
	if !domain.IsValidAIProvider(stored.Provider) {
		stored.Provider = defaults.Provider
	}

	if domain.IsCustomAIProvider(stored.Provider) {
		found := false
		for _, cp := range stored.CustomProviders {
			if cp.ID == stored.Provider {
				found = true
				if strings.TrimSpace(cp.BaseURL) != "" {
					stored.BaseURL = cp.BaseURL
				}
				break
			}
		}
		if !found {
			stored.Provider = defaults.Provider
		}
	}

	if domain.IsCloudAIProvider(stored.Provider) {
		official, _ := domain.OfficialAIBaseURL(stored.Provider)
		stored.BaseURL = official
	} else if strings.TrimSpace(stored.BaseURL) == "" && !domain.IsCustomAIProvider(stored.Provider) {
		stored.BaseURL = defaults.BaseURL
	}

	if stored.HourlyQuotaPerUser < domain.MinAIHourlyQuota || stored.HourlyQuotaPerUser > domain.MaxAIHourlyQuota {
		stored.HourlyQuotaPerUser = defaults.HourlyQuotaPerUser
	}
	if stored.TimeoutSeconds < domain.MinAITimeoutSeconds || stored.TimeoutSeconds > domain.MaxAITimeoutSeconds {
		stored.TimeoutSeconds = defaults.TimeoutSeconds
	}

	if stored.ProviderProfiles == nil {
		stored.ProviderProfiles = make(map[string]domain.AIProviderProfileStored)
	}
	for _, p := range domain.BaseAIProvidersList {
		prof, exists := stored.ProviderProfiles[p]
		if !exists {
			offURL, _ := domain.OfficialAIBaseURL(p)
			prof = domain.AIProviderProfileStored{
				Model:   "",
				BaseURL: offURL,
			}
		}
		if stored.Provider == p {
			if prof.EncryptedAPIKey == "" && stored.EncryptedAPIKey != "" {
				prof.EncryptedAPIKey = stored.EncryptedAPIKey
				prof.APIKeyMasked = stored.APIKeyMasked
			}
			if prof.Model == "" && stored.Model != "" {
				prof.Model = stored.Model
			}
		}
		if domain.IsCloudAIProvider(p) {
			offURL, _ := domain.OfficialAIBaseURL(p)
			prof.BaseURL = offURL
		}
		stored.ProviderProfiles[p] = prof
	}

	if stored.CustomProviders == nil {
		stored.CustomProviders = make([]domain.AICustomProviderStored, 0)
	}

	if stored.UserOverrides == nil {
		stored.UserOverrides = make(map[string]domain.AIUserOverrideStored)
	}

	return &stored, nil
}

// Update valida, cifra la API Key si corresponde y persiste la configuración.
// Devuelve la respuesta pública (sin encrypted_api_key).
func (s *AISettingsService) Update(ctx context.Context, tenantID string, dto domain.UpdateAISettingsDTO) (*domain.AISettingsResponseDTO, error) {
	current, err := s.Get(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	if !domain.IsValidAIProvider(dto.Provider) {
		return nil, &AISettingsValidationError{
			Code:    "provider_invalid",
			Message: fmt.Sprintf("proveedor inválido: %q. Valores permitidos: %s", dto.Provider, domain.AIProviderList()),
		}
	}

	// 1. Procesar custom_providers (creación, edición, eliminación y escudo anti-exfiltración de API Key).
	var finalCustomProviders []domain.AICustomProviderStored
	if dto.CustomProviders != nil {
		if len(dto.CustomProviders) > domain.MaxCustomProviders {
			return nil, &AISettingsValidationError{
				Code:    "custom_providers_limit_exceeded",
				Message: fmt.Sprintf("se permite un máximo de %d proveedores personalizados por institución", domain.MaxCustomProviders),
			}
		}

		existingCustom := make(map[string]domain.AICustomProviderStored, len(current.CustomProviders))
		for _, cp := range current.CustomProviders {
			existingCustom[cp.ID] = cp
		}

		seenIDs := make(map[string]bool)
		finalCustomProviders = make([]domain.AICustomProviderStored, 0, len(dto.CustomProviders))

		for _, cpDTO := range dto.CustomProviders {
			id := strings.TrimSpace(cpDTO.ID)
			if id == "" {
				id = generateCustomProviderID(cpDTO.Label)
			}
			if !domain.IsCustomAIProvider(id) {
				return nil, &AISettingsValidationError{
					Code:    "custom_provider_invalid",
					Message: fmt.Sprintf("el ID del proveedor personalizado %q debe comenzar con 'custom_' y contener solo caracteres alfanuméricos, guiones o guiones bajos", id),
				}
			}
			if domain.IsBaseAIProvider(id) {
				return nil, &AISettingsValidationError{
					Code:    "custom_provider_invalid",
					Message: fmt.Sprintf("el ID %q está reservado para el catálogo base", id),
				}
			}
			if seenIDs[id] {
				return nil, &AISettingsValidationError{
					Code:    "custom_provider_invalid",
					Message: fmt.Sprintf("ID de proveedor personalizado duplicado: %q", id),
				}
			}
			seenIDs[id] = true

			if !domain.ValidateCustomProviderLabel(cpDTO.Label) {
				return nil, &AISettingsValidationError{
					Code:    "custom_provider_invalid",
					Message: "el nombre (label) del proveedor personalizado es obligatorio y debe tener entre 1 y 60 caracteres",
				}
			}

			if !domain.ValidateCustomProviderIconURL(cpDTO.IconURL) {
				return nil, &AISettingsValidationError{
					Code:    "custom_provider_invalid",
					Message: "icon_url debe ser una URL válida (http:// o https://) de hasta 500 caracteres",
				}
			}

			baseURL := strings.TrimSpace(cpDTO.BaseURL)
			if baseURL == "" {
				return nil, &AISettingsValidationError{
					Code:    "base_url_required",
					Message: fmt.Sprintf("la URL base para el proveedor %q es obligatoria", id),
				}
			}
			if err := validateLANBaseURL(baseURL); err != nil {
				return nil, err
			}
			baseURL = strings.TrimRight(baseURL, "/")

			existing, exists := existingCustom[id]
			encryptedKey := ""
			maskedKey := ""
			if exists {
				encryptedKey = existing.EncryptedAPIKey
				maskedKey = existing.APIKeyMasked
			}

			newKey := strings.TrimSpace(cpDTO.APIKey)
			if dto.Provider == id && strings.TrimSpace(dto.APIKey) != "" {
				newKey = strings.TrimSpace(dto.APIKey)
			}
			clearKey := cpDTO.ClearAPIKey || (dto.Provider == id && dto.ClearAPIKey)

			if exists {
				oldHost := extractHost(existing.BaseURL)
				newHost := extractHost(baseURL)
				hostChanged := oldHost != "" && newHost != "" && !strings.EqualFold(oldHost, newHost)
				if hostChanged && newKey == "" {
					// Escudo Anti-Exfiltración: El host cambió y no se envió nueva API Key -> invalidar clave previa
					encryptedKey = ""
					maskedKey = ""
				}
			}

			if clearKey {
				encryptedKey = ""
				maskedKey = ""
			} else if newKey != "" {
				cipherText, errCipher := s.encryptAPIKey(newKey)
				if errCipher != nil {
					return nil, fmt.Errorf("no se pudo cifrar la API Key de %s: %w", id, errCipher)
				}
				encryptedKey = cipherText
				maskedKey = maskAPIKey(newKey)
			}

			model := strings.TrimSpace(cpDTO.Model)
			if dto.Provider == id && strings.TrimSpace(dto.Model) != "" {
				model = strings.TrimSpace(dto.Model)
			} else if model == "" && exists {
				model = existing.Model
			}

			finalCustomProviders = append(finalCustomProviders, domain.AICustomProviderStored{
				ID:              id,
				Label:           strings.TrimSpace(cpDTO.Label),
				IconURL:         strings.TrimSpace(cpDTO.IconURL),
				BaseURL:         baseURL,
				Model:           model,
				EncryptedAPIKey: encryptedKey,
				APIKeyMasked:    maskedKey,
			})
		}
	} else {
		finalCustomProviders = make([]domain.AICustomProviderStored, len(current.CustomProviders))
		copy(finalCustomProviders, current.CustomProviders)

		// Si el proveedor activo es custom y se enviaron cambios a nivel raíz, sincronizarlos
		if domain.IsCustomAIProvider(dto.Provider) {
			for i, cp := range finalCustomProviders {
				if cp.ID == dto.Provider {
					if strings.TrimSpace(dto.BaseURL) != "" {
						newBaseURL, err := resolveAIBaseURL(dto.Provider, dto.BaseURL)
						if err != nil {
							return nil, err
						}
						oldHost := extractHost(cp.BaseURL)
						newHost := extractHost(newBaseURL)
						if oldHost != "" && newHost != "" && !strings.EqualFold(oldHost, newHost) && strings.TrimSpace(dto.APIKey) == "" {
							cp.EncryptedAPIKey = ""
							cp.APIKeyMasked = ""
						}
						cp.BaseURL = newBaseURL
					}
					if dto.ClearAPIKey {
						cp.EncryptedAPIKey = ""
						cp.APIKeyMasked = ""
					} else if strings.TrimSpace(dto.APIKey) != "" {
						cipherText, errCipher := s.encryptAPIKey(strings.TrimSpace(dto.APIKey))
						if errCipher != nil {
							return nil, fmt.Errorf("no se pudo cifrar la API Key de %s: %w", cp.ID, errCipher)
						}
						cp.EncryptedAPIKey = cipherText
						cp.APIKeyMasked = maskAPIKey(strings.TrimSpace(dto.APIKey))
					}
					if strings.TrimSpace(dto.Model) != "" {
						cp.Model = strings.TrimSpace(dto.Model)
					}
					finalCustomProviders[i] = cp
					break
				}
			}
		}
	}

	// 2. Si el proveedor activo es custom, verificar que exista en la lista final
	if domain.IsCustomAIProvider(dto.Provider) {
		found := false
		for _, cp := range finalCustomProviders {
			if cp.ID == dto.Provider {
				found = true
				break
			}
		}
		if !found {
			return nil, &AISettingsValidationError{
				Code:    "provider_invalid",
				Message: fmt.Sprintf("proveedor personalizado %q no existe en la lista de proveedores", dto.Provider),
			}
		}
	}

	// 3. Si el proveedor activo es base, actualizar su perfil preservando los otros
	if current.ProviderProfiles == nil {
		current.ProviderProfiles = make(map[string]domain.AIProviderProfileStored)
	}

	if domain.IsBaseAIProvider(dto.Provider) {
		prof := current.ProviderProfiles[dto.Provider]

		rawBaseURL := strings.TrimSpace(dto.BaseURL)
		if rawBaseURL == "" && dto.Provider == domain.AIProviderOllamaLAN {
			rawBaseURL = strings.TrimSpace(prof.BaseURL)
			if rawBaseURL == "" {
				rawBaseURL = strings.TrimSpace(current.BaseURL)
			}
		}
		baseURL, err := resolveAIBaseURL(dto.Provider, rawBaseURL)
		if err != nil {
			return nil, err
		}

		encrypted := prof.EncryptedAPIKey
		masked := prof.APIKeyMasked
		newKey := strings.TrimSpace(dto.APIKey)
		switch {
		case dto.ClearAPIKey:
			encrypted, masked = "", ""
		case newKey != "":
			cipherText, errCipher := s.encryptAPIKey(newKey)
			if errCipher != nil {
				return nil, fmt.Errorf("no se pudo cifrar la API Key: %w", errCipher)
			}
			encrypted = cipherText
			masked = maskAPIKey(newKey)
		}

		model := strings.TrimSpace(dto.Model)
		if model == "" {
			model = prof.Model
		}

		prof.Model = model
		prof.BaseURL = baseURL
		prof.EncryptedAPIKey = encrypted
		prof.APIKeyMasked = masked
		current.ProviderProfiles[dto.Provider] = prof
	}

	// 4. Determinar los valores activos del proveedor seleccionado
	var activeModel string
	var activeBaseURL string
	var activeEncryptedKey string
	var activeMaskedKey string

	if domain.IsBaseAIProvider(dto.Provider) {
		prof := current.ProviderProfiles[dto.Provider]
		activeModel = prof.Model
		activeBaseURL = prof.BaseURL
		activeEncryptedKey = prof.EncryptedAPIKey
		activeMaskedKey = prof.APIKeyMasked
	} else {
		for _, cp := range finalCustomProviders {
			if cp.ID == dto.Provider {
				activeModel = cp.Model
				activeBaseURL = cp.BaseURL
				activeEncryptedKey = cp.EncryptedAPIKey
				activeMaskedKey = cp.APIKeyMasked
				break
			}
		}
	}

	// 5. Regla de activación: para enabled=true es obligatorio tener un modelo real seleccionado.
	if dto.Enabled {
		if strings.TrimSpace(activeModel) == "" {
			return nil, &AISettingsValidationError{
				Code:    "model_required",
				Message: domain.ErrAIMissingModel.Error(),
			}
		}
		if domain.IsCloudAIProvider(dto.Provider) && strings.TrimSpace(activeEncryptedKey) == "" {
			return nil, &AISettingsValidationError{
				Code:    "api_key_required",
				Message: domain.ErrAIMissingAPIKey.Error(),
			}
		}
	}

	quota := dto.HourlyQuotaPerUser
	if quota == 0 {
		quota = domain.DefaultAIHourlyQuota
	}
	if quota < domain.MinAIHourlyQuota || quota > domain.MaxAIHourlyQuota {
		return nil, &AISettingsValidationError{
			Code:    "hourly_quota_invalid",
			Message: fmt.Sprintf("hourly_quota_per_user debe estar entre %d y %d. Recibido: %d", domain.MinAIHourlyQuota, domain.MaxAIHourlyQuota, dto.HourlyQuotaPerUser),
		}
	}

	timeout := dto.TimeoutSeconds
	if timeout == 0 {
		timeout = domain.DefaultAITimeoutSeconds
	}
	if timeout < domain.MinAITimeoutSeconds || timeout > domain.MaxAITimeoutSeconds {
		return nil, &AISettingsValidationError{
			Code:    "timeout_invalid",
			Message: fmt.Sprintf("timeout_seconds debe estar entre %d y %d. Recibido: %d", domain.MinAITimeoutSeconds, domain.MaxAITimeoutSeconds, dto.TimeoutSeconds),
		}
	}

	stored := domain.AISettingsStored{
		Enabled:            dto.Enabled,
		Provider:           dto.Provider,
		Model:              activeModel,
		BaseURL:            activeBaseURL,
		EncryptedAPIKey:    activeEncryptedKey,
		APIKeyMasked:       activeMaskedKey,
		HourlyQuotaPerUser: quota,
		TimeoutSeconds:     timeout,
		UpdatedAt:          s.nowFn().UTC().Format(time.RFC3339),
		ProviderProfiles:   current.ProviderProfiles,
		CustomProviders:    finalCustomProviders,
		// La cascada BYOK personal pertenece al usuario, no a la configuración
		// institucional: la actualización del administrador jamás la pisa.
		UserOverrides: current.UserOverrides,
	}

	if err := s.persist(ctx, tenantID, stored); err != nil {
		return nil, err
	}

	s.modelsMu.Lock()
	prefix := tenantID + "|"
	for k := range s.modelsCache {
		if strings.HasPrefix(k, prefix) {
			delete(s.modelsCache, k)
		}
	}
	s.modelsMu.Unlock()

	resp := domain.NewAISettingsResponseDTO(stored)
	return &resp, nil
}

func (s *AISettingsService) persist(ctx context.Context, tenantID string, stored domain.AISettingsStored) error {
	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil || tenant == nil {
		return fmt.Errorf("tenant not found")
	}

	var configMap map[string]interface{}
	if len(tenant.Config) > 0 {
		if err := json.Unmarshal(tenant.Config, &configMap); err != nil {
			configMap = make(map[string]interface{})
		}
	}
	if configMap == nil {
		configMap = make(map[string]interface{})
	}

	configMap[aiSettingsConfigKey] = stored
	newConfig, err := json.Marshal(configMap)
	if err != nil {
		return fmt.Errorf("error serializing ai settings: %w", err)
	}
	return s.tenantRepo.UpdateConfig(ctx, tenantID, newConfig)
}

// ConsumeQuota consume un punto de la cuota horaria del usuario dentro de la
// ventana en curso y devuelve domain.ErrAIQuotaExceeded al superar el límite.
// Protegido por sync.Mutex: el conteo vive únicamente en memoria.
func (s *AISettingsService) ConsumeQuota(tenantID, userID string, limitPerHour int) error {
	if limitPerHour < domain.MinAIHourlyQuota || limitPerHour > domain.MaxAIHourlyQuota {
		limitPerHour = domain.DefaultAIHourlyQuota
	}

	s.quotaMu.Lock()
	defer s.quotaMu.Unlock()

	key := tenantID + "|" + userID
	now := s.nowFn()
	usage, ok := s.quotas[key]
	if !ok || now.Sub(usage.windowStart) >= time.Hour {
		s.quotas[key] = &aiQuotaUsage{windowStart: now, count: 1}
		return nil
	}
	if usage.count >= limitPerHour {
		return domain.ErrAIQuotaExceeded
	}
	usage.count++
	return nil
}

// CompleteJSON invoca al motor OpenAI-compatible configurado pidiendo JSON
// estricto: resuelve la cascada BYOK del usuario, consume 1 punto de cuota
// institucional (solo si opera con la base del instituto), descifra la API Key,
// aplica el presupuesto de tiempo automático del perfil de motor y devuelve el
// contenido junto a la latencia en milisegundos.
// action identifica la operación para el presupuesto de tiempo y los mensajes
// de diagnóstico.
func (s *AISettingsService) CompleteJSON(ctx context.Context, tenantID, userID, action, systemPrompt, userPrompt string) (string, int64, error) {
	settings, err := s.Get(ctx, tenantID)
	if err != nil {
		return "", 0, err
	}

	effective := *settings
	profile := classifyAIEngine(settings.Provider, settings.Model)

	if override, ok := resolveUserOverride(settings, userID); ok {
		// BYOK: la credencial personal prevalece, opera incluso con la IA
		// institucional deshabilitada y no consume la cuota horaria compartida.
		effective.Provider = override.Provider
		effective.Model = override.Model
		effective.BaseURL = userOverrideBaseURL(settings, override)
		effective.EncryptedAPIKey = override.EncryptedAPIKey
		profile = classifyAIEngine(override.Provider, override.Model)
	} else {
		if !settings.Enabled {
			return "", 0, domain.ErrAIDisabled
		}
		quotaLimit := settings.HourlyQuotaPerUser
		if profile == domain.AIEngineProfileLocalGPU {
			// Motor local: no hay costo por token externo, se eleva el piso
			// anti-abuso a 60 solicitudes por hora.
			quotaLimit = maxInt(quotaLimit, 60)
		}
		if err := s.ConsumeQuota(tenantID, userID, quotaLimit); err != nil {
			return "", 0, err
		}
	}

	apiKey := ""
	if strings.TrimSpace(effective.EncryptedAPIKey) != "" {
		apiKey, err = s.decryptAPIKey(effective.EncryptedAPIKey)
		if err != nil {
			return "", 0, fmt.Errorf("no se pudo descifrar la API Key guardada: %w", err)
		}
	}

	provider, err := s.providerFor(&effective)
	if err != nil {
		return "", 0, err
	}
	reqCtx, cancel := context.WithTimeout(ctx, effectiveAITimeout(&effective, profile, action))
	defer cancel()

	payload := aiChatCompletionRequest{
		Model: effective.Model,
		Messages: []aiChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature:    0.2,
		ResponseFormat: aiResponseFormat{Type: "json_object"},
	}
	content, latencyMS, err := provider.Complete(reqCtx, payload, apiKey)
	if err != nil {
		return "", latencyMS, err
	}
	return content, latencyMS, nil
}

// resolveUserOverride devuelve la cascada BYOK personal activa y válida de un
// usuario. Un override ausente, deshabilitado o incompleto se trata como
// inexistente: la operación cae en la base institucional.
func resolveUserOverride(settings *domain.AISettingsStored, userID string) (domain.AIUserOverrideStored, bool) {
	var zero domain.AIUserOverrideStored
	if settings == nil || strings.TrimSpace(userID) == "" {
		return zero, false
	}
	override, ok := settings.UserOverrides[userID]
	if !ok || !override.Enabled {
		return zero, false
	}
	if !domain.IsValidAIProvider(override.Provider) {
		return zero, false
	}
	if strings.TrimSpace(override.Model) == "" {
		return zero, false
	}
	if domain.IsCloudAIProvider(override.Provider) && strings.TrimSpace(override.EncryptedAPIKey) == "" {
		return zero, false
	}
	return override, true
}

// userOverrideBaseURL resuelve la URL de la credencial personal: oficial y
// fija en la nube, la propia en servidores LAN y la del catálogo custom como
// último recurso.
func userOverrideBaseURL(settings *domain.AISettingsStored, override domain.AIUserOverrideStored) string {
	if domain.IsCloudAIProvider(override.Provider) {
		official, _ := domain.OfficialAIBaseURL(override.Provider)
		return official
	}
	if base := strings.TrimSpace(override.BaseURL); base != "" {
		return strings.TrimRight(base, "/")
	}
	for _, cp := range settings.CustomProviders {
		if cp.ID == override.Provider {
			return cp.BaseURL
		}
	}
	return ""
}

// classifyAIEngine resuelve el perfil de gobernanza del motor activo.
func classifyAIEngine(provider, model string) string {
	return domain.ClassifyAIEngine(provider, model)
}

// effectiveAITimeout resuelve el presupuesto de tiempo final:
// min(MaxAITimeoutSeconds, max(presupuestoAutomático, timeoutGuardado)).
// Así un tope viejo de 30 s guardado en la base jamás recorta una operación
// pesada ni un modelo gratuito, y un tope mayor fijado en Modo Experto se
// respeta.
func effectiveAITimeout(settings *domain.AISettingsStored, profile, action string) time.Duration {
	stored := settings.TimeoutSeconds
	if stored <= 0 {
		stored = domain.DefaultAITimeoutSeconds
	}
	seconds := domain.AIEngineAutoTimeoutSeconds(profile, action)
	if seconds < stored {
		seconds = stored
	}
	if seconds > domain.MaxAITimeoutSeconds {
		seconds = domain.MaxAITimeoutSeconds
	}
	if seconds < domain.MinAITimeoutSeconds {
		seconds = domain.MinAITimeoutSeconds
	}
	return time.Duration(seconds) * time.Second
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// TestConnection realiza una verificación real contra el motor (o el
// http.RoundTripper inyectable) pidiendo el JSON mínimo {"ok":true} y mide
// latency_ms. Los fallos de validación local devuelven error (HTTP 400 en el
// handler); los fallos del motor remoto devuelven ok=false sin error.
func (s *AISettingsService) TestConnection(ctx context.Context, tenantID string, req domain.TestAIConnectionRequestDTO) (*domain.TestAIConnectionResponseDTO, error) {
	stored, err := s.Get(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	provider := strings.TrimSpace(req.Provider)
	if provider == "" {
		provider = stored.Provider
	}
	if !domain.IsValidAIProvider(provider) {
		return nil, &AISettingsValidationError{
			Code:    "provider_invalid",
			Message: fmt.Sprintf("proveedor inválido: %q. Valores permitidos: %s", provider, domain.AIProviderList()),
		}
	}

	var profileBaseURL string
	var profileEncryptedKey string
	var profileModel string

	if domain.IsBaseAIProvider(provider) {
		if prof, ok := stored.ProviderProfiles[provider]; ok {
			profileBaseURL = prof.BaseURL
			profileEncryptedKey = prof.EncryptedAPIKey
			profileModel = prof.Model
		} else if stored.Provider == provider {
			profileBaseURL = stored.BaseURL
			profileEncryptedKey = stored.EncryptedAPIKey
			profileModel = stored.Model
		}
	} else if domain.IsCustomAIProvider(provider) {
		for _, cp := range stored.CustomProviders {
			if cp.ID == provider {
				profileBaseURL = cp.BaseURL
				profileEncryptedKey = cp.EncryptedAPIKey
				profileModel = cp.Model
				break
			}
		}
		if profileBaseURL == "" && stored.Provider == provider {
			profileBaseURL = stored.BaseURL
			profileEncryptedKey = stored.EncryptedAPIKey
			profileModel = stored.Model
		}
	}

	model := strings.TrimSpace(req.Model)
	if model == "" {
		model = profileModel
	}

	rawBaseURL := strings.TrimSpace(req.BaseURL)
	if rawBaseURL == "" {
		rawBaseURL = strings.TrimSpace(profileBaseURL)
	}
	baseURL, err := resolveAIBaseURL(provider, rawBaseURL)
	if err != nil {
		return nil, err
	}

	apiKey := strings.TrimSpace(req.APIKey)
	if apiKey == "" && strings.TrimSpace(profileEncryptedKey) != "" {
		apiKey, err = s.decryptAPIKey(profileEncryptedKey)
		if err != nil {
			return nil, fmt.Errorf("no se pudo descifrar la API Key guardada: %w", err)
		}
	}

	probe := *stored
	probe.Provider = provider
	probe.Model = model
	probe.BaseURL = baseURL

	payload := aiChatCompletionRequest{
		Model: model,
		Messages: []aiChatMessage{
			{Role: "system", Content: "Sos un verificador de conectividad. Respondé únicamente con JSON válido."},
			{Role: "user", Content: `Devolvé exactamente este JSON: {"ok":true}`},
		},
		Temperature:    0,
		ResponseFormat: aiResponseFormat{Type: "json_object"},
	}

	resp := &domain.TestAIConnectionResponseDTO{
		OK:       false,
		Provider: provider,
		Model:    model,
	}

	aiProvider, err := s.providerFor(&probe)
	if err != nil {
		return nil, err
	}
	reqCtx, cancel := context.WithTimeout(ctx, effectiveAITimeout(&probe, classifyAIEngine(provider, model), "test-connection"))
	defer cancel()

	content, latencyMS, callErr := aiProvider.Complete(reqCtx, payload, apiKey)
	resp.LatencyMS = latencyMS
	if callErr != nil {
		resp.Message = fmt.Sprintf("No se pudo verificar la conexión: %s", callErr.Error())
		return resp, nil
	}
	if !json.Valid([]byte(extractJSONObject(content))) {
		resp.Message = "No se pudo verificar la conexión: el motor no devolvió JSON válido"
		return resp, nil
	}

	resp.OK = true
	resp.Message = "Conexión verificada correctamente con el motor de IA."
	return resp, nil
}

// extractJSONObject recorta razonamiento previo, bloques de texto sueltos y
// fences Markdown para quedarse con el objeto JSON que el motor devolvió.
// Modelos de razonamiento (deepseek-r1, nemotron, qwq) responden a menudo
// "pensando... ```json {\"ok\":true}```"; la prueba de conexión debe aceptarlos.
func extractJSONObject(content string) string {
	trimmed := strings.TrimSpace(content)
	start := strings.IndexByte(trimmed, '{')
	if start < 0 {
		return trimmed
	}
	end := strings.LastIndexByte(trimmed, '}')
	if end < start {
		return trimmed
	}
	return strings.TrimSpace(trimmed[start : end+1])
}

// GetUserOverride devuelve la cascada BYOK personal del usuario junto al
// estado institucional. Nunca expone el ciphertext de la API Key.
func (s *AISettingsService) GetUserOverride(ctx context.Context, tenantID, userID string) (*domain.AIUserOverrideResponseDTO, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, &AISettingsValidationError{
			Code:    "user_required",
			Message: "no se pudo identificar al usuario para su configuración personal de IA",
		}
	}
	stored, err := s.Get(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	override, ok := stored.UserOverrides[userID]
	if !ok || strings.TrimSpace(override.Provider) == "" {
		// Sin override todavía: se pre-carga con el proveedor por defecto, pero
		// sin modelo ni credenciales personales para no confundir al docente con
		// la configuración del motor institucional.
		override = domain.AIUserOverrideStored{
			Enabled:  false,
			Provider: stored.Provider,
			Model:    "",
			BaseURL:  "",
		}
	}
	resp := domain.NewAIUserOverrideResponseDTO(override, *stored)
	return &resp, nil
}

// UpdateUserOverride valida, cifra y persiste la API Key personal (BYOK) del
// usuario. Aplica las mismas garantías que la configuración institucional:
// catálogo de proveedores, URL oficial en la nube, escudo anti-SSRF en
// ollama_lan/custom_* y cifrado AES-256-GCM.
func (s *AISettingsService) UpdateUserOverride(ctx context.Context, tenantID, userID string, dto domain.UpdateAIUserOverrideDTO) (*domain.AIUserOverrideResponseDTO, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, &AISettingsValidationError{
			Code:    "user_required",
			Message: "no se pudo identificar al usuario para su configuración personal de IA",
		}
	}
	stored, err := s.Get(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	provider := strings.TrimSpace(dto.Provider)
	if provider == "" {
		provider = stored.Provider
	}
	if !domain.IsValidAIProvider(provider) {
		return nil, &AISettingsValidationError{
			Code:    "provider_invalid",
			Message: fmt.Sprintf("proveedor inválido: %q. Valores permitidos: %s", provider, domain.AIProviderList()),
		}
	}

	previous := stored.UserOverrides[userID]

	rawBaseURL := strings.TrimSpace(dto.BaseURL)
	if rawBaseURL == "" {
		rawBaseURL = strings.TrimSpace(previous.BaseURL)
	}
	if rawBaseURL == "" && provider == stored.Provider {
		rawBaseURL = strings.TrimSpace(stored.BaseURL)
	}
	if rawBaseURL == "" && domain.IsCustomAIProvider(provider) {
		for _, cp := range stored.CustomProviders {
			if cp.ID == provider {
				rawBaseURL = cp.BaseURL
				break
			}
		}
	}
	baseURL, err := resolveAIBaseURL(provider, rawBaseURL)
	if err != nil {
		return nil, err
	}

	model := strings.TrimSpace(dto.Model)
	if model == "" {
		model = strings.TrimSpace(previous.Model)
	}
	if model == "" && provider == stored.Provider {
		model = strings.TrimSpace(stored.Model)
	}

	encryptedKey := previous.EncryptedAPIKey
	maskedKey := previous.APIKeyMasked
	switch {
	case dto.ClearAPIKey:
		encryptedKey, maskedKey = "", ""
	case strings.TrimSpace(dto.APIKey) != "":
		cipherText, errCipher := s.encryptAPIKey(strings.TrimSpace(dto.APIKey))
		if errCipher != nil {
			return nil, fmt.Errorf("no se pudo cifrar la API Key personal: %w", errCipher)
		}
		encryptedKey = cipherText
		maskedKey = maskAPIKey(strings.TrimSpace(dto.APIKey))
	}

	if dto.Enabled {
		if strings.TrimSpace(model) == "" {
			return nil, &AISettingsValidationError{
				Code:    "model_required",
				Message: domain.ErrAIMissingModel.Error(),
			}
		}
		if domain.IsCloudAIProvider(provider) && strings.TrimSpace(encryptedKey) == "" {
			return nil, &AISettingsValidationError{
				Code:    "api_key_required",
				Message: domain.ErrAIMissingAPIKey.Error(),
			}
		}
	}

	override := domain.AIUserOverrideStored{
		Enabled:         dto.Enabled,
		Provider:        provider,
		Model:           model,
		BaseURL:         baseURL,
		EncryptedAPIKey: encryptedKey,
		APIKeyMasked:    maskedKey,
		UpdatedAt:       s.nowFn().UTC().Format(time.RFC3339),
	}
	stored.UserOverrides[userID] = override

	if err := s.persist(ctx, tenantID, *stored); err != nil {
		return nil, err
	}

	resp := domain.NewAIUserOverrideResponseDTO(override, *stored)
	return &resp, nil
}

// GetEffectiveStatus devuelve el estado liviano resuelto para el usuario:
// refleja el motor personal (BYOK) cuando está activo y conserva el estado
// institucional por separado.
func (s *AISettingsService) GetEffectiveStatus(ctx context.Context, tenantID, userID string) (*domain.AIStatusDTO, error) {
	stored, err := s.Get(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	resp := domain.NewAIStatusDTO(*stored)
	if override, ok := resolveUserOverride(stored, userID); ok {
		profile := domain.ClassifyAIEngine(override.Provider, override.Model)
		resp.Enabled = true
		resp.Provider = override.Provider
		resp.ProviderLabel = domain.AIProviderLabel(override.Provider, stored.CustomProviders)
		resp.Model = override.Model
		resp.EngineProfile = profile
		resp.AutoTimeoutRangeLabel = domain.AIEngineAutoTimeoutRangeLabel(profile)
		resp.UsingPersonalKey = true
	}
	return &resp, nil
}

// TestUserConnection prueba la conexión con las credenciales personales del
// usuario cuando no llegan en el payload.
func (s *AISettingsService) TestUserConnection(ctx context.Context, tenantID, userID string, req domain.TestAIConnectionRequestDTO) (*domain.TestAIConnectionResponseDTO, error) {
	req = s.fillUserCredentials(ctx, tenantID, userID, req.Provider, req.Model, req.BaseURL, req.APIKey)
	if err := requireUserCredential(req.Provider, req.BaseURL, req.APIKey); err != nil {
		return nil, err
	}
	return s.TestConnection(ctx, tenantID, req)
}

// ListUserModels lista los modelos disponibles con las credenciales personales
// del usuario cuando no llegan en el payload.
func (s *AISettingsService) ListUserModels(ctx context.Context, tenantID, userID string, req domain.ListAIModelsRequestDTO) (*domain.ListAIModelsResponseDTO, error) {
	filled := s.fillUserCredentials(ctx, tenantID, userID, req.Provider, "", req.BaseURL, req.APIKey)
	if err := requireUserCredential(filled.Provider, filled.BaseURL, filled.APIKey); err != nil {
		return nil, err
	}
	req.Provider = filled.Provider
	req.BaseURL = filled.BaseURL
	req.APIKey = filled.APIKey
	return s.ListModels(ctx, tenantID, req)
}

// requireUserCredential corta la cascada BYOK cuando el usuario todavía no
// aportó una credencial propia. ListModels y TestConnection completan con la
// credencial institucional guardada, comportamiento correcto para el panel del
// administrador pero incorrecto acá: el docente no debe descubrir el catálogo
// del motor institucional desde su configuración personal.
func requireUserCredential(provider, baseURL, apiKey string) error {
	if strings.TrimSpace(provider) == "" {
		return &AISettingsValidationError{
			Code:    "provider_required",
			Message: "seleccioná un proveedor de IA",
		}
	}
	if domain.IsCloudAIProvider(provider) {
		if strings.TrimSpace(apiKey) == "" {
			return &AISettingsValidationError{
				Code:    "api_key_required",
				Message: domain.ErrAIMissingAPIKey.Error(),
			}
		}
		return nil
	}
	if strings.TrimSpace(baseURL) == "" {
		return &AISettingsValidationError{
			Code:    "base_url_required",
			Message: "la URL base del servidor local es obligatoria (ej: http://10.20.1.50:11434/v1)",
		}
	}
	return nil
}

// fillUserCredentials completa provider/model/base_url/api_key con la cascada
// BYOK del usuario solo cuando el payload viene vacío, para que la prueba de
// conexión y el catálogo usen la cuenta personal sin consumir la institucional.
func (s *AISettingsService) fillUserCredentials(ctx context.Context, tenantID, userID, provider, model, baseURL, apiKey string) domain.TestAIConnectionRequestDTO {
	filled := domain.TestAIConnectionRequestDTO{Provider: provider, Model: model, BaseURL: baseURL, APIKey: apiKey}
	if strings.TrimSpace(apiKey) != "" {
		return filled
	}
	stored, err := s.Get(ctx, tenantID)
	if err != nil {
		return filled
	}
	override, ok := resolveUserOverride(stored, userID)
	if !ok {
		return filled
	}
	if strings.TrimSpace(filled.Provider) == "" || filled.Provider == override.Provider {
		if strings.TrimSpace(filled.Provider) == "" {
			filled.Provider = override.Provider
		}
		if strings.TrimSpace(filled.Model) == "" {
			filled.Model = override.Model
		}
		if strings.TrimSpace(filled.BaseURL) == "" {
			filled.BaseURL = override.BaseURL
		}
		if plain, errKey := s.decryptAPIKey(override.EncryptedAPIKey); errKey == nil {
			filled.APIKey = plain
		}
	}
	return filled
}

// ListModels consulta dinámicamente los modelos disponibles en el proveedor
// respetando el escudo anti-SSRF, reutilizando la API Key guardada (o en memoria)
// y aplicando una caché en memoria de 5 minutos por (tenant, proveedor, baseURL).
func (s *AISettingsService) ListModels(ctx context.Context, tenantID string, req domain.ListAIModelsRequestDTO) (*domain.ListAIModelsResponseDTO, error) {
	stored, err := s.Get(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	provider := strings.TrimSpace(req.Provider)
	if provider == "" {
		provider = stored.Provider
	}
	if !domain.IsValidAIProvider(provider) {
		return nil, &AISettingsValidationError{
			Code:    "provider_invalid",
			Message: fmt.Sprintf("proveedor inválido: %q. Valores permitidos: %s", provider, domain.AIProviderList()),
		}
	}

	var profileBaseURL string
	var profileEncryptedKey string

	if domain.IsBaseAIProvider(provider) {
		if prof, ok := stored.ProviderProfiles[provider]; ok {
			profileBaseURL = prof.BaseURL
			profileEncryptedKey = prof.EncryptedAPIKey
		} else if stored.Provider == provider {
			profileBaseURL = stored.BaseURL
			profileEncryptedKey = stored.EncryptedAPIKey
		}
	} else if domain.IsCustomAIProvider(provider) {
		for _, cp := range stored.CustomProviders {
			if cp.ID == provider {
				profileBaseURL = cp.BaseURL
				profileEncryptedKey = cp.EncryptedAPIKey
				break
			}
		}
		if profileBaseURL == "" && stored.Provider == provider {
			profileBaseURL = stored.BaseURL
			profileEncryptedKey = stored.EncryptedAPIKey
		}
	}

	rawBaseURL := strings.TrimSpace(req.BaseURL)
	if rawBaseURL == "" {
		rawBaseURL = strings.TrimSpace(profileBaseURL)
	}
	baseURL, err := resolveAIBaseURL(provider, rawBaseURL)
	if err != nil {
		return nil, err
	}

	apiKey := strings.TrimSpace(req.APIKey)
	usingStoredKey := apiKey == ""
	if usingStoredKey && strings.TrimSpace(profileEncryptedKey) != "" {
		apiKey, err = s.decryptAPIKey(profileEncryptedKey)
		if err != nil {
			return nil, fmt.Errorf("no se pudo descifrar la API Key guardada: %w", err)
		}
	}

	if domain.IsCloudAIProvider(provider) && apiKey == "" {
		return nil, &AISettingsValidationError{
			Code:    "api_key_required",
			Message: domain.ErrAIMissingAPIKey.Error(),
		}
	}

	cacheKey := fmt.Sprintf("%s|%s|%s", tenantID, provider, baseURL)

	// Caché en memoria de 5 minutos: solo aplica si se usan credenciales guardadas y refresh == false
	if usingStoredKey && !req.Refresh {
		s.modelsMu.RLock()
		entry, found := s.modelsCache[cacheKey]
		s.modelsMu.RUnlock()
		if found && s.nowFn().Before(entry.expiresAt) {
			modelsCopy := make([]domain.AIModelItemDTO, len(entry.models))
			copy(modelsCopy, entry.models)
			return &domain.ListAIModelsResponseDTO{
				Provider: provider,
				Cached:   true,
				Count:    len(modelsCopy),
				Models:   modelsCopy,
			}, nil
		}
	}

	probe := *stored
	probe.Provider = provider
	probe.BaseURL = baseURL

	aiProvider, err := s.providerFor(&probe)
	if err != nil {
		return nil, err
	}

	reqCtx, cancel := context.WithTimeout(ctx, aiSettingsTimeout(&probe))
	defer cancel()

	models, err := aiProvider.ListModels(reqCtx, provider, apiKey)
	if err != nil {
		return nil, err
	}
	if models == nil {
		models = make([]domain.AIModelItemDTO, 0)
	}

	if usingStoredKey {
		s.modelsMu.Lock()
		s.modelsCache[cacheKey] = aiModelsCacheEntry{
			models:    models,
			expiresAt: s.nowFn().Add(5 * time.Minute),
		}
		s.modelsMu.Unlock()
	}

	return &domain.ListAIModelsResponseDTO{
		Provider: provider,
		Cached:   false,
		Count:    len(models),
		Models:   models,
	}, nil
}

// AIProvider es el contrato de un motor de IA enchufable. Cada proveedor
// implementa cómo ejecutar una chat completion y devolver el contenido del
// primer mensaje junto a la latencia en milisegundos, así como listar modelos.
type AIProvider interface {
	Complete(ctx context.Context, payload aiChatCompletionRequest, apiKey string) (string, int64, error)
	ListModels(ctx context.Context, providerID, apiKey string) ([]domain.AIModelItemDTO, error)
}

// openAICompatibleProvider habla el wire protocol OpenAI /chat/completions:
// es la implementación base de los proveedores cloud (gemini, openai,
// openrouter, deepseek), del servidor local LAN (ollama_lan) y de custom_providers.
type openAICompatibleProvider struct {
	client  *http.Client
	baseURL string
	nowFn   func() time.Time
}

func newOpenAICompatibleProvider(client *http.Client, baseURL string, nowFn func() time.Time) *openAICompatibleProvider {
	return &openAICompatibleProvider{client: client, baseURL: strings.TrimRight(baseURL, "/"), nowFn: nowFn}
}

func (p *openAICompatibleProvider) Complete(ctx context.Context, payload aiChatCompletionRequest, apiKey string) (string, int64, error) {
	start := p.nowFn()

	content, latencyMS, statusCode, err := p.completeOnce(ctx, payload, apiKey, start)
	if err == nil {
		return content, latencyMS, nil
	}

	// Blindaje de transporte (S10): varios modelos gratuitos de OpenRouter
	// rechazan "response_format": {"type":"json_object"} con HTTP 400. Se
	// reintenta una única vez sin response_format acumulando la latencia total.
	if statusCode == http.StatusBadRequest && strings.TrimSpace(payload.ResponseFormat.Type) != "" {
		retry := payload
		retry.ResponseFormat = aiResponseFormat{}
		content, retryLatencyMS, _, retryErr := p.completeOnce(ctx, retry, apiKey, start)
		if retryErr == nil {
			return content, retryLatencyMS, nil
		}
		return "", retryLatencyMS, retryErr
	}

	return "", latencyMS, err
}

// completeOnce ejecuta un intento de /chat/completions. La latencia se mide
// desde start (compartido entre intentos) hasta después de io.ReadAll, de modo
// que refleje la descarga completa del stream JSON y no solo el TTFB de los
// headers HTTP.
func (p *openAICompatibleProvider) completeOnce(ctx context.Context, payload aiChatCompletionRequest, apiKey string, start time.Time) (string, int64, int, error) {
	elapsed := func() int64 { return p.nowFn().Sub(start).Milliseconds() }

	body, err := json.Marshal(payload)
	if err != nil {
		return "", elapsed(), 0, fmt.Errorf("no se pudo serializar la petición de IA: %w", err)
	}

	endpoint := p.baseURL + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", elapsed(), 0, fmt.Errorf("petición de IA inválida: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(apiKey) != "" {
		httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	}

	httpResp, err := p.client.Do(httpReq)
	if err != nil {
		if isAITimeout(err, ctx) {
			return "", elapsed(), 0, fmt.Errorf("%w: %w", domain.ErrAITimeout, err)
		}
		return "", elapsed(), 0, fmt.Errorf("fallo de conexión con el motor de IA: %w", err)
	}
	defer func() { _ = httpResp.Body.Close() }()

	raw, err := io.ReadAll(io.LimitReader(httpResp.Body, aiMaxResponseBytes))
	if err != nil {
		if isAITimeout(err, ctx) {
			return "", elapsed(), 0, fmt.Errorf("%w: %w", domain.ErrAITimeout, err)
		}
		return "", elapsed(), 0, fmt.Errorf("no se pudo leer la respuesta del motor de IA: %w", err)
	}

	latencyMS := elapsed()
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return "", latencyMS, httpResp.StatusCode, fmt.Errorf("el motor de IA respondió HTTP %d: %s", httpResp.StatusCode, truncateForLog(string(raw), 200))
	}

	var parsed aiChatCompletionResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", latencyMS, httpResp.StatusCode, fmt.Errorf("respuesta inesperada del motor de IA: %w", err)
	}
	if len(parsed.Choices) == 0 || strings.TrimSpace(parsed.Choices[0].Message.Content) == "" {
		return "", latencyMS, httpResp.StatusCode, fmt.Errorf("la respuesta del motor de IA no incluye contenido")
	}
	return parsed.Choices[0].Message.Content, latencyMS, httpResp.StatusCode, nil
}

// isAITimeout reporta si el fallo corresponde a un agotamiento de plazo del
// contexto de la petición o del contexto padre.
func isAITimeout(err error, ctx context.Context) bool {
	return errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded)
}

var excludedModelKeywords = []string{
	"embedding",
	"whisper",
	"tts",
	"dall-e",
	"moderation",
	"babbage",
	"davinci",
	"curie",
	"ada",
	"aqa",
	"imagen",
	"veo",
}

type rawOpenAIModelResponse struct {
	Data []rawOpenAIModelItem `json:"data"`
}

type rawOpenAIModelItem struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Description   string            `json:"description"`
	ContextLength int               `json:"context_length"`
	Pricing       *rawOpenAIPricing `json:"pricing"`
}

type rawOpenAIPricing struct {
	Prompt     interface{} `json:"prompt"`
	Completion interface{} `json:"completion"`
}

func isZeroPricing(v interface{}) bool {
	if v == nil {
		return false
	}
	switch val := v.(type) {
	case string:
		s := strings.TrimSpace(val)
		if s == "0" || s == "0.0" {
			return true
		}
		f, err := strconv.ParseFloat(s, 64)
		return err == nil && f == 0
	case float64:
		return val == 0
	case int:
		return val == 0
	case json.Number:
		f, err := val.Float64()
		return err == nil && f == 0
	}
	return false
}

func (p *openAICompatibleProvider) ListModels(ctx context.Context, providerID, apiKey string) ([]domain.AIModelItemDTO, error) {
	endpoint := p.baseURL + "/models"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("petición de modelos inválida: %w", err)
	}
	if strings.TrimSpace(apiKey) != "" {
		httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	}

	httpResp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("fallo de conexión al consultar modelos de IA: %w", err)
	}
	defer func() { _ = httpResp.Body.Close() }()

	raw, err := io.ReadAll(io.LimitReader(httpResp.Body, aiMaxResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("no se pudo leer la respuesta de modelos de IA: %w", err)
	}
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return nil, fmt.Errorf("el proveedor de IA respondió HTTP %d al listar modelos: %s", httpResp.StatusCode, truncateForLog(string(raw), 200))
	}

	var parsed rawOpenAIModelResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("respuesta inesperada al consultar modelos de IA: %w", err)
	}

	result := make([]domain.AIModelItemDTO, 0)
	seen := make(map[string]bool)

	for _, item := range parsed.Data {
		rawID := strings.TrimSpace(item.ID)
		if rawID == "" {
			continue
		}

		// Normalización de Gemini (y proveedores que anteponen models/)
		cleanID := rawID
		if strings.HasPrefix(cleanID, "models/") {
			cleanID = strings.TrimPrefix(cleanID, "models/")
		}

		if seen[cleanID] {
			continue
		}

		// Filtro de exclusión de modelos no aptos para chat / copiloto
		idLower := strings.ToLower(cleanID)
		excluded := false
		for _, kw := range excludedModelKeywords {
			if strings.Contains(idLower, kw) {
				excluded = true
				break
			}
		}
		if excluded {
			continue
		}

		seen[cleanID] = true

		name := strings.TrimSpace(item.Name)
		if name == "" {
			name = cleanID
		}

		var tier string
		var pricingHint string

		switch providerID {
		case domain.AIProviderOpenRouter:
			isFree := strings.HasSuffix(cleanID, ":free")
			if !isFree && item.Pricing != nil {
				if isZeroPricing(item.Pricing.Prompt) && isZeroPricing(item.Pricing.Completion) {
					isFree = true
				}
			}
			if isFree {
				tier = "free"
				pricingHint = "Gratis"
			} else {
				tier = "paid"
				pricingHint = "De pago"
			}
		case domain.AIProviderOllamaLAN:
			tier = "local"
			pricingHint = "Local · Sin costo de API"
		default:
			// gemini, openai, deepseek, custom_*
			tier = "by_account"
			pricingHint = "Según plan de la cuenta"
		}

		result = append(result, domain.AIModelItemDTO{
			ID:            cleanID,
			Name:          name,
			Description:   strings.TrimSpace(item.Description),
			Tier:          tier,
			PricingHint:   pricingHint,
			ContextLength: item.ContextLength,
		})
	}

	if providerID == domain.AIProviderOpenRouter {
		sort.SliceStable(result, func(i, j int) bool {
			iFree := result[i].Tier == "free"
			jFree := result[j].Tier == "free"
			if iFree != jFree {
				return iFree
			}
			return strings.ToLower(result[i].ID) < strings.ToLower(result[j].ID)
		})
	} else {
		sort.SliceStable(result, func(i, j int) bool {
			return strings.ToLower(result[i].ID) < strings.ToLower(result[j].ID)
		})
	}

	return result, nil
}

// providerFor devuelve la implementación de AIProvider para la configuración
// vigente. Todos los proveedores soportados (base y custom) hablan el protocolo
// OpenAI-compatible hoy.
func (s *AISettingsService) providerFor(stored *domain.AISettingsStored) (AIProvider, error) {
	if !domain.IsValidAIProvider(stored.Provider) {
		return nil, &AISettingsValidationError{
			Code:    "provider_invalid",
			Message: fmt.Sprintf("proveedor inválido: %q. Valores permitidos: %s", stored.Provider, domain.AIProviderList()),
		}
	}
	return newOpenAICompatibleProvider(s.client, stored.BaseURL, s.nowFn), nil
}

// aiSettingsTimeout devuelve el timeout institucional de la configuración.
func aiSettingsTimeout(settings *domain.AISettingsStored) time.Duration {
	timeout := time.Duration(settings.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = time.Duration(domain.DefaultAITimeoutSeconds) * time.Second
	}
	return timeout
}

// AISettingsValidationError es el error tipado de validación de configuración
// de IA; el handler HTTP lo traduce a HTTP 400.
type AISettingsValidationError struct {
	Code    string
	Message string
}

func (e *AISettingsValidationError) Error() string { return e.Message }

// resolveAIBaseURL aplica el escudo anti-SSRF y devuelve la URL base efectiva.
// Para proveedores en la nube siempre devuelve la URL oficial e ignora el input.
// Para ollama_lan y custom_* aplica validateLANBaseURL estrictamente.
func resolveAIBaseURL(provider, raw string) (string, error) {
	if domain.IsCloudAIProvider(provider) {
		official, ok := domain.OfficialAIBaseURL(provider)
		if !ok {
			return "", &AISettingsValidationError{
				Code:    "provider_invalid",
				Message: fmt.Sprintf("proveedor en la nube sin URL oficial: %q", provider),
			}
		}
		return official, nil
	}

	base := strings.TrimSpace(raw)
	if base == "" {
		if provider == domain.AIProviderOllamaLAN {
			return "", &AISettingsValidationError{
				Code:    "base_url_required",
				Message: "la URL base del servidor local LAN es obligatoria (ej: http://10.20.1.50:11434/v1)",
			}
		}
		return "", &AISettingsValidationError{
			Code:    "base_url_required",
			Message: "la URL base del proveedor es obligatoria",
		}
	}
	if err := validateLANBaseURL(base); err != nil {
		return "", err
	}
	return strings.TrimRight(base, "/"), nil
}

// validateLANBaseURL rechaza hosts loopback, no especificados, link-local
// (metadata cloud 169.254.0.0/16) y nombres internos de contenedores Docker
// sin dominio. Permite IPs privadas LAN (10/172.16-31/192.168) y dominios con punto.
func validateLANBaseURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return &AISettingsValidationError{
			Code:    "base_url_invalid",
			Message: fmt.Sprintf("URL base inválida: %q", raw),
		}
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return &AISettingsValidationError{
			Code:    "base_url_invalid",
			Message: "la URL base debe usar el esquema http:// o https://",
		}
	}

	host := strings.ToLower(u.Hostname())
	if host == "" {
		return &AISettingsValidationError{
			Code:    "base_url_invalid",
			Message: fmt.Sprintf("URL base sin host: %q", raw),
		}
	}
	blocked := func(reason string) error {
		return &AISettingsValidationError{
			Code:    "base_url_blocked",
			Message: fmt.Sprintf("URL base no permitida (%s): %s. Use una IP privada de LAN o un dominio con punto", host, reason),
		}
	}

	if host == "localhost" || host == "0.0.0.0" || host == "::1" {
		return blocked("loopback")
	}
	if ip := net.ParseIP(host); ip != nil {
		switch {
		case ip.IsLoopback():
			return blocked("loopback")
		case ip.IsUnspecified():
			return blocked("ip no especificada")
		case ip.IsLinkLocalUnicast():
			return blocked("rango link-local / metadata cloud")
		}
		return nil
	}
	if !strings.Contains(host, ".") {
		return blocked("nombre interno de contenedor sin dominio")
	}
	return nil
}

// extractHost extrae el host (con puerto si aplica) en minúsculas para comparar cambios de dominio.
func extractHost(rawURL string) string {
	raw := strings.TrimSpace(rawURL)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return strings.ToLower(u.Host)
}

// generateCustomProviderID genera un ID con prefijo custom_ a partir del label.
func generateCustomProviderID(label string) string {
	var b strings.Builder
	b.WriteString("custom_")
	for _, r := range strings.ToLower(strings.TrimSpace(label)) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else if r == ' ' || r == '-' || r == '_' {
			b.WriteRune('_')
		}
	}
	res := strings.Trim(b.String(), "_")
	if res == "custom" || res == "" {
		res = fmt.Sprintf("custom_%d", time.Now().UnixNano()%1000000)
	}
	if len(res) > 50 {
		res = res[:50]
	}
	return res
}

// maskAPIKey enmascara la clave: primeros 4 + "..." + últimos 4; si mide 8 o
// menos caracteres devuelve "****".
func maskAPIKey(key string) string {
	k := strings.TrimSpace(key)
	if len(k) <= 8 {
		return "****"
	}
	return k[:4] + "..." + k[len(k)-4:]
}

func (s *AISettingsService) encryptAPIKey(plain string) (string, error) {
	block, err := aes.NewCipher(s.secret)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := gcm.Seal(nonce, nonce, []byte(plain), nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

func (s *AISettingsService) decryptAPIKey(encoded string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(s.secret)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", fmt.Errorf("payload cifrado de API Key demasiado corto")
	}
	nonce, ciphertext := raw[:gcm.NonceSize()], raw[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

func truncateForLog(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

// aiChatCompletionRequest es el payload OpenAI-compatible de /chat/completions.
type aiChatCompletionRequest struct {
	Model          string           `json:"model"`
	Messages       []aiChatMessage  `json:"messages"`
	Temperature    float64          `json:"temperature"`
	ResponseFormat aiResponseFormat `json:"response_format"`
}

type aiChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type aiResponseFormat struct {
	Type string `json:"type"`
}

type aiChatCompletionResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}
