package services_test

import (
	"strings"
	"testing"

	"solv-backend/internal/core/services"
)

func TestResolveFontConfig_CatalogSlugs(t *testing.T) {
	sans, mono, err := services.ResolveFontConfig("cat:source-sans-3", "cat:fira-code")
	if err != nil {
		t.Fatalf("expected valid catalog fonts, got %v", err)
	}
	if sans != "cat:source-sans-3" || mono != "cat:fira-code" {
		t.Fatalf("expected resolved values, got sans=%q mono=%q", sans, mono)
	}
}

func TestResolveFontConfig_EmptyMeansNoChange(t *testing.T) {
	sans, mono, err := services.ResolveFontConfig("", "")
	if err != nil || sans != "" || mono != "" {
		t.Fatalf("expected empty passthrough, got sans=%q mono=%q err=%v", sans, mono, err)
	}
}

func TestResolveFontConfig_UnknownSlug(t *testing.T) {
	_, _, err := services.ResolveFontConfig("cat:comic-sans-pro", "")
	if err == nil {
		t.Fatal("expected error for unknown slug")
	}
	if err.Code != "font_slug_unknown" {
		t.Fatalf("expected font_slug_unknown, got %s", err.Code)
	}
	if !strings.Contains(err.Message, "catálogo curado") {
		t.Fatalf("expected actionable message, got %q", err.Message)
	}
}

func TestResolveFontConfig_KindMismatch(t *testing.T) {
	// jetbrains-mono es mono: no puede usarse como fuente UI
	_, _, err := services.ResolveFontConfig("cat:jetbrains-mono", "")
	if err == nil {
		t.Fatal("expected error for kind mismatch")
	}
	if err.Code != "font_kind_mismatch" {
		t.Fatalf("expected font_kind_mismatch, got %s", err.Code)
	}
}

func TestResolveFontConfig_CustomURLValidation(t *testing.T) {
	cases := []struct {
		name     string
		url      string
		wantCode string
	}{
		{"http inseguro", "url:http://fonts.googleapis.com/css2?family=Public+Sans", "font_url_invalid"},
		{"dominio no permitido", "url:https://cdn.evil.example.com/font.css", "font_url_host_not_allowed"},
		{"hoja inexistente", "url:https://fonts.googleapis.com/css2?family=NoExisteFuenteXYZ123", "font_url_unreachable"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := services.ResolveFontConfig(tc.url, "")
			if err == nil {
				t.Fatalf("expected error %s, got nil", tc.wantCode)
			}
			if err.Code != tc.wantCode {
				t.Fatalf("expected %s, got %s (%s)", tc.wantCode, err.Code, err.Message)
			}
		})
	}
}

func TestResolveFontConfig_CustomURLValid(t *testing.T) {
	// Hoja real de Google Fonts: alcanzable y de dominio permitido
	sans, _, err := services.ResolveFontConfig("url:https://fonts.googleapis.com/css2?family=Public+Sans:wght@400;600;700", "")
	if err != nil {
		t.Fatalf("expected valid custom URL, got %v (%s)", err, err.Message)
	}
	if sans != "url:https://fonts.googleapis.com/css2?family=Public+Sans:wght@400;600;700" {
		t.Fatalf("unexpected resolved value %q", sans)
	}
}

func TestResolveFontConfig_InvalidFormat(t *testing.T) {
	_, _, err := services.ResolveFontConfig("Inter", "")
	if err == nil || err.Code != "font_format_invalid" {
		t.Fatalf("expected font_format_invalid, got %v", err)
	}
}
