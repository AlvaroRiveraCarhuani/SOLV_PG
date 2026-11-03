package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
)

// El presupuesto ya no es una tabla fija: sigue la latencia medida del motor.
func TestAIAdaptiveTimeoutFollowsObservedLatency(t *testing.T) {
	fast := domain.AIEngineProfileFastCloud

	if got := domain.AIAdaptiveTimeoutSeconds(fast, "suggest-component", domain.AILatencySample{}); got != 60 {
		t.Fatalf("sin historial esperaba 60s, got %ds", got)
	}

	// Un motor que viene tardando 30 s recibe 90 s (3×), por encima del piso.
	slow := domain.AILatencySample{EWMAMS: 30000, Samples: 4}
	if got := domain.AIAdaptiveTimeoutSeconds(fast, "suggest-component", slow); got != 90 {
		t.Fatalf("con 30s observados esperaba 90s, got %ds", got)
	}

	// Tras agotar el plazo, el siguiente intento crece 1,4× sobre lo esperado.
	timedOut := domain.AILatencySample{EWMAMS: 60000, Samples: 3, TimedOut: true}
	if got := domain.AIAdaptiveTimeoutSeconds(fast, "suggest-component", timedOut); got != 84 {
		t.Fatalf("tras un plazo agotado esperaba 84s, got %ds", got)
	}

	// Lo que espera una persona no pasa del techo interactivo, aunque el perfil
	// o el modelo permitan más: una espera mayor se percibe como pantalla colgada.
	huge := domain.AILatencySample{EWMAMS: 900000, Samples: 5}
	if got := domain.AIAdaptiveTimeoutSeconds(fast, "suggest-component", huge); got != domain.AIInteractiveTimeoutSeconds {
		t.Fatalf("esperaba el techo interactivo %ds, got %ds", domain.AIInteractiveTimeoutSeconds, got)
	}
	// Las tareas de fondo sí conservan el tope global.
	if got := domain.AIAdaptiveTimeoutSeconds(fast, "generate-exercise", huge); got != domain.MaxAITimeoutSeconds {
		t.Fatalf("esperaba tope %ds, got %ds", domain.MaxAITimeoutSeconds, got)
	}
	fastest := domain.AILatencySample{EWMAMS: 100, Samples: 2}
	if got := domain.AIAdaptiveTimeoutSeconds(fast, "suggest-template", fastest); got != 90 {
		t.Fatalf("un motor rápido no debe bajar del piso del perfil: got %ds", got)
	}

	// El presupuesto efectivo usa lo aprendido sin saltarse el tope del administrador.
	settings := &domain.AISettingsStored{
		Provider:       domain.AIProviderOpenAI,
		Model:          "gpt-4o-mini",
		TimeoutSeconds: 60,
		LatencyProfile: map[string]domain.AILatencySample{
			domain.AILatencyKey(domain.AIProviderOpenAI, "gpt-4o-mini", "suggest-component"): slow,
		},
	}
	if got := int(effectiveAITimeout(settings, fast, "suggest-component") / time.Second); got != 90 {
		t.Fatalf("effectiveAITimeout debía aprender 90s, got %ds", got)
	}
	expert := *settings
	expert.TimeoutSeconds = 240
	if got := int(effectiveAITimeout(&expert, fast, "suggest-component") / time.Second); got != 240 {
		t.Fatalf("el tope del administrador sigue mandando: got %ds", got)
	}
}

func TestNextAILatencySampleSmoothsAndFlagsTimeouts(t *testing.T) {
	at := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)

	first := domain.NextAILatencySample(domain.AILatencySample{}, 10000, false, at)
	if first.EWMAMS != 10000 || first.Samples != 1 || first.TimedOut {
		t.Fatalf("primera muestra inesperada: %+v", first)
	}
	if first.UpdatedAt != at.Format(time.RFC3339) {
		t.Fatalf("sello de tiempo inesperado: %q", first.UpdatedAt)
	}

	// (10000×0,6 + 20000×0,4) = 14000
	second := domain.NextAILatencySample(first, 20000, false, at)
	if second.EWMAMS != 14000 || second.Samples != 2 {
		t.Fatalf("promedio móvil inesperado: %+v", second)
	}

	timedOut := domain.NextAILatencySample(second, 20000, true, at)
	if !timedOut.TimedOut {
		t.Fatal("el plazo agotado debía quedar marcado")
	}
	// Un plazo agotado no dice cuánto tardó el motor: no toca el promedio.
	if timedOut.EWMAMS != 14000 || timedOut.Samples != 2 {
		t.Fatalf("el plazo agotado ensució el promedio: %+v", timedOut)
	}

	recovered := domain.NextAILatencySample(timedOut, 12000, false, at)
	if recovered.TimedOut {
		t.Fatal("una llamada exitosa debía limpiar la marca de plazo agotado")
	}
	if recovered.Samples != 3 {
		t.Fatalf("esperaba 3 muestras, got %d", recovered.Samples)
	}
	if recovered.EWMAMS != 13200 {
		t.Fatalf("esperaba promedio 13200, got %d", recovered.EWMAMS)
	}
}

// La latencia medida se guarda junto a la configuración sin romperla.
func TestAISettingsRecordsLatencyProfile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"name\":\"Entorno\"}"}}]}`))
	}))
	defer server.Close()

	repo := newTestAISettingsRepo(`{}`)
	svc := NewAISettingsService(repo)
	ctx := context.Background()

	if _, err := svc.Update(ctx, "t-1", domain.UpdateAISettingsDTO{
		Enabled:            true,
		Provider:           domain.AIProviderOpenAI,
		Model:              "gpt-4o-mini",
		APIKey:             "sk-test-123",
		HourlyQuotaPerUser: 5,
		TimeoutSeconds:     60,
	}); err != nil {
		t.Fatalf("no se pudo configurar la IA: %v", err)
	}
	svc.WithTransport(&rewriteAITestTransport{base: mustParseURL(t, server.URL)})

	if _, _, err := svc.CompleteJSON(ctx, "t-1", "u-1", "suggest-template", "sys", "usr"); err != nil {
		t.Fatalf("llamada inesperada con error: %v", err)
	}

	tenant, err := repo.GetByID(ctx, "t-1")
	if err != nil || tenant == nil {
		t.Fatalf("no se pudo leer el tenant: %v", err)
	}

	var config map[string]json.RawMessage
	if err := json.Unmarshal(tenant.Config, &config); err != nil {
		t.Fatalf("configuración ilegible: %v", err)
	}
	var aiConfig domain.AISettingsStored
	if err := json.Unmarshal(config["ai_settings"], &aiConfig); err != nil {
		t.Fatalf("ai_settings ilegible: %v", err)
	}

	sample, ok := aiConfig.LatencyProfile[domain.AILatencyKey(domain.AIProviderOpenAI, "gpt-4o-mini", "suggest-template")]
	if !ok || sample.Samples != 1 {
		t.Fatalf("no se guardó la latencia medida: %+v", aiConfig.LatencyProfile)
	}

	// El resto de la configuración sigue intacta: modelo, clave cifrada y estado.
	if !aiConfig.Enabled || aiConfig.Model != "gpt-4o-mini" || aiConfig.EncryptedAPIKey == "" {
		t.Fatalf("la latencia pisó la configuración vigente: %+v", aiConfig)
	}
}
