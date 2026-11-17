package services

import (
	"net"
	"strings"
	"testing"

	"solv-backend/internal/core/domain"
)

func parseIPForTest(t *testing.T, raw string) net.IP {
	t.Helper()
	ip := net.ParseIP(raw)
	if ip == nil {
		t.Fatalf("IP inválida en el test: %q", raw)
	}
	return ip
}

// El docente solo puede apuntar a endpoints públicos con https: cualquier otra
// combinación convertiría su credencial personal en un SSRF contra la red del
// servidor universitario.
func TestValidateUserPublicBaseURL(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantErr bool
	}{
		{name: "acepta https público", raw: "https://8.8.8.8/openai/v1"},
		{name: "normaliza la barra final", raw: "https://8.8.4.4/v1/"},
		{name: "rechaza http", raw: "http://8.8.8.8/v1", wantErr: true},
		{name: "rechaza loopback", raw: "https://127.0.0.1/v1", wantErr: true},
		{name: "rechaza IPv6 loopback", raw: "https://[::1]/v1", wantErr: true},
		{name: "rechaza red privada 10/8", raw: "https://10.1.2.3/v1", wantErr: true},
		{name: "rechaza red privada 192.168/16", raw: "https://192.168.1.10/v1", wantErr: true},
		{name: "rechaza link-local metadata", raw: "https://169.254.169.254/v1", wantErr: true},
		{name: "rechaza localhost por nombre", raw: "https://localhost/v1", wantErr: true},
		{name: "rechaza credenciales embebidas", raw: "https://user:pass@8.8.8.8/v1", wantErr: true},
		{name: "rechaza url vacía", raw: "   ", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validateUserPublicBaseURL(tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("esperaba error para %q, obtuve %q", tt.raw, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("no esperaba error para %q: %v", tt.raw, err)
			}
			if strings.HasSuffix(got, "/") {
				t.Errorf("la URL normalizada no debe terminar en barra: %q", got)
			}
		})
	}
}

func TestIsPublicInternetIP(t *testing.T) {
	if isPublicInternetIP(nil) {
		t.Fatal("nil no es una IP pública")
	}
	for _, raw := range []string{"127.0.0.1", "10.0.0.1", "172.16.5.4", "192.168.0.1", "169.254.169.254", "100.64.0.1"} {
		if isPublicInternetIP(parseIPForTest(t, raw)) {
			t.Errorf("%s debía considerarse no pública", raw)
		}
	}
	if !isPublicInternetIP(parseIPForTest(t, "8.8.8.8")) {
		t.Error("8.8.8.8 debía considerarse pública")
	}
}

func TestApplyUserCustomProvidersCreationsYRemoval(t *testing.T) {
	svc := NewAISettingsService(nil)

	existing := []domain.AICustomProviderStored{{
		ID:      "custom_viejo",
		Label:   "Proveedor viejo",
		BaseURL: "https://8.8.8.8/v1",
	}}

	dto := domain.UpdateAIUserOverrideDTO{
		CustomProviders: []domain.AIUserCustomProviderInputDTO{{
			Label:   "Groq Cloud",
			BaseURL: "https://8.8.8.8/openai/v1/",
			Model:   "llama-3.3-70b-versatile",
			APIKey:  "sk-personal-123456789",
		}},
	}

	got, err := svc.applyUserCustomProviders(existing, nil, dto)
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("esperaba 2 proveedores, obtuve %d", len(got))
	}

	var created *domain.AICustomProviderStored
	for i := range got {
		if got[i].Label == "Groq Cloud" {
			created = &got[i]
		}
	}
	if created == nil {
		t.Fatal("no se agregó el proveedor nuevo")
	}
	if created.ID != "custom_groq_cloud" {
		t.Errorf("id derivado inesperado: %q", created.ID)
	}
	if !domain.IsCustomAIProvider(created.ID) {
		t.Errorf("el id debe conservar el prefijo custom_: %q", created.ID)
	}
	if created.EncryptedAPIKey == "" {
		t.Error("la clave personal debe quedar cifrada")
	}
	if created.EncryptedAPIKey == "sk-personal-123456789" {
		t.Error("la clave personal no puede persistirse en texto plano")
	}
	if created.APIKeyMasked == "" {
		t.Error("la clave debe conservar su versión enmascarada")
	}
	if strings.HasSuffix(created.BaseURL, "/") {
		t.Errorf("la URL base debe guardarse sin barra final: %q", created.BaseURL)
	}

	removed, err := svc.applyUserCustomProviders(got, nil, domain.UpdateAIUserOverrideDTO{
		RemoveCustomProviders: []string{"custom_viejo"},
	})
	if err != nil {
		t.Fatalf("no esperaba error al eliminar: %v", err)
	}
	if len(removed) != 1 || removed[0].Label != "Groq Cloud" {
		t.Fatalf("la baja debía dejar solo el proveedor nuevo, obtuve %+v", removed)
	}
}

func TestApplyUserCustomProvidersRechazaColisionInstitucional(t *testing.T) {
	svc := NewAISettingsService(nil)

	institutional := []domain.AICustomProviderStored{{ID: "custom_institucional", Label: "Institucional"}}
	_, err := svc.applyUserCustomProviders(nil, institutional, domain.UpdateAIUserOverrideDTO{
		CustomProviders: []domain.AIUserCustomProviderInputDTO{{
			ID:      "custom_institucional",
			Label:   "Intento de reemplazo",
			BaseURL: "https://8.8.8.8/v1",
		}},
	})
	if err == nil {
		t.Fatal("no se puede sobrescribir un proveedor del catálogo institucional")
	}
}

func TestApplyUserCustomProvidersRechazaHTTP(t *testing.T) {
	svc := NewAISettingsService(nil)

	_, err := svc.applyUserCustomProviders(nil, nil, domain.UpdateAIUserOverrideDTO{
		CustomProviders: []domain.AIUserCustomProviderInputDTO{{
			Label:   "Local inseguro",
			BaseURL: "http://10.20.1.50:11434/v1",
		}},
	})
	if err == nil {
		t.Fatal("el docente no puede registrar endpoints http ni de LAN")
	}
}

// El catálogo se expone con su origen y el motor institucional se identifica
// por id, no solo por etiqueta: el panel del docente lo necesita para marcar la
// tarjeta que ya funciona sin clave personal.
func TestNewAIUserOverrideResponseDTOExponeOrigenYMotorInstitucional(t *testing.T) {
	institutional := domain.AISettingsStored{
		Provider: "custom_institucional",
		CustomProviders: []domain.AICustomProviderStored{
			{ID: "custom_institucional", Label: "Inception", BaseURL: "https://api.inceptionlabs.ai/v1"},
		},
	}
	override := domain.AIUserOverrideStored{
		Provider: "custom_propio",
		CustomProviders: []domain.AICustomProviderStored{
			{ID: "custom_propio", Label: "Groq Cloud", BaseURL: "https://api.groq.com/openai/v1"},
		},
	}

	resp := domain.NewAIUserOverrideResponseDTO(override, institutional)
	if resp.InstitutionalProvider != "custom_institucional" {
		t.Errorf("institutional_provider inesperado: %q", resp.InstitutionalProvider)
	}
	if len(resp.CustomProviders) != 2 {
		t.Fatalf("esperaba 2 proveedores en el catálogo, obtuve %d", len(resp.CustomProviders))
	}
	if resp.CustomProviders[0].Origin != domain.AIProviderOriginInstitutional {
		t.Errorf("el primero debía ser institucional: %q", resp.CustomProviders[0].Origin)
	}
	if resp.CustomProviders[1].Origin != domain.AIProviderOriginPersonal {
		t.Errorf("el segundo debía ser personal: %q", resp.CustomProviders[1].Origin)
	}
	if resp.ProviderLabel != "Groq Cloud" {
		t.Errorf("la etiqueta del proveedor propio se resuelve desde su catálogo: %q", resp.ProviderLabel)
	}
}
