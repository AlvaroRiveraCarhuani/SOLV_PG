package services

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// CuratedFont describe una fuente aprobada del catálogo white-label (ADR-038).
// Los pesos están alineados al contrato del design system SOLV: UI 400-700,
// mono 400-600.
type CuratedFont struct {
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Stack       string `json:"stack"`
	GoogleQuery string `json:"google_query"`
	Weights     []int  `json:"weights"`
	Kind        string `json:"kind"` // "sans" | "mono"
}

// CuratedFontCatalog catálogo cerrado de fuentes aprobadas por legibilidad.
// Cualquier valor fuera de este catálogo solo puede provenir de la vía de URL
// custom validada.
var CuratedFontCatalog = []CuratedFont{
	// --- UI (sans) ---
	{Slug: "inter", Name: "Inter", Stack: "'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif", GoogleQuery: "Inter:wght@400;500;600;700", Weights: []int{400, 500, 600, 700}, Kind: "sans"},
	{Slug: "source-sans-3", Name: "Source Sans 3", Stack: "'Source Sans 3', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif", GoogleQuery: "Source+Sans+3:wght@400;500;600;700", Weights: []int{400, 500, 600, 700}, Kind: "sans"},
	{Slug: "open-sans", Name: "Open Sans", Stack: "'Open Sans', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif", GoogleQuery: "Open+Sans:wght@400;500;600;700", Weights: []int{400, 500, 600, 700}, Kind: "sans"},
	{Slug: "public-sans", Name: "Public Sans", Stack: "'Public Sans', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif", GoogleQuery: "Public+Sans:wght@400;500;600;700", Weights: []int{400, 500, 600, 700}, Kind: "sans"},
	{Slug: "lato", Name: "Lato", Stack: "'Lato', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif", GoogleQuery: "Lato:wght@400;700", Weights: []int{400, 500, 600, 700}, Kind: "sans"},
	{Slug: "roboto", Name: "Roboto", Stack: "'Roboto', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif", GoogleQuery: "Roboto:wght@400;500;600;700", Weights: []int{400, 500, 600, 700}, Kind: "sans"},
	// --- Datos de máquina (mono) ---
	{Slug: "jetbrains-mono", Name: "JetBrains Mono", Stack: "'JetBrains Mono', 'Fira Code', Menlo, Monaco, Consolas, monospace", GoogleQuery: "JetBrains+Mono:wght@400;500;600", Weights: []int{400, 500, 600}, Kind: "mono"},
	{Slug: "fira-code", Name: "Fira Code", Stack: "'Fira Code', 'JetBrains Mono', Menlo, Monaco, Consolas, monospace", GoogleQuery: "Fira+Code:wght@400;500;600", Weights: []int{400, 500, 600}, Kind: "mono"},
	{Slug: "ibm-plex-mono", Name: "IBM Plex Mono", Stack: "'IBM Plex Mono', 'JetBrains Mono', Menlo, Monaco, Consolas, monospace", GoogleQuery: "IBM+Plex+Mono:wght@400;500;600", Weights: []int{400, 500, 600}, Kind: "mono"},
}

// FindCuratedFont localiza una fuente del catálogo por slug.
func FindCuratedFont(slug string) *CuratedFont {
	for i := range CuratedFontCatalog {
		if CuratedFontCatalog[i].Slug == slug {
			return &CuratedFontCatalog[i]
		}
	}
	return nil
}

// CuratedFontSlugs devuelve los slugs válidos del catálogo (para mensajes de error).
func CuratedFontSlugs() string {
	slugs := make([]string, 0, len(CuratedFontCatalog))
	for _, f := range CuratedFontCatalog {
		slugs = append(slugs, f.Slug)
	}
	return strings.Join(slugs, ", ")
}

const allowedFontCSSHost = "fonts.googleapis.com"

// FontValidationError describe por qué se rechazó una configuración tipográfica.
type FontValidationError struct {
	Code    string
	Message string
}

func (e *FontValidationError) Error() string { return e.Message }

// ResolveFontConfig valida y resuelve la pareja de fuentes del tenant.
//
// Reglas (proposal tenant-typography):
//   - "" significa "sin cambio / default" (Inter / JetBrains Mono) y es válido.
//   - Un valor "cat:slug" referencia el catálogo curado.
//   - Un valor "url:https://..." usa una hoja CSS de Google Fonts validada.
func ResolveFontConfig(fontSans, fontMono string) (sans string, mono string, err *FontValidationError) {
	sans, sErr := resolveSingleFont(fontSans, "sans")
	if sErr != nil {
		return "", "", sErr
	}
	mono, mErr := resolveSingleFont(fontMono, "mono")
	if mErr != nil {
		return "", "", mErr
	}
	return sans, mono, nil
}

func resolveSingleFont(value, kind string) (string, *FontValidationError) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil // sin cambio
	}

	if strings.HasPrefix(value, "cat:") {
		slug := strings.TrimPrefix(value, "cat:")
		font := FindCuratedFont(slug)
		if font == nil {
			return "", &FontValidationError{
				Code:    "font_slug_unknown",
				Message: fmt.Sprintf("La fuente \"%s\" no pertenece al catálogo curado. Opciones válidas: %s", slug, CuratedFontSlugs()),
			}
		}
		if font.Kind != kind {
			return "", &FontValidationError{
				Code:    "font_kind_mismatch",
				Message: fmt.Sprintf("La fuente \"%s\" es de tipo %s y se solicitó para %s.", slug, font.Kind, fontSlotLabel(kind)),
			}
		}
		return value, nil
	}

	if strings.HasPrefix(value, "url:") {
		rawURL := strings.TrimPrefix(value, "url:")
		if verr := validateCustomFontURL(rawURL); verr != nil {
			return "", verr
		}
		return value, nil
	}

	return "", &FontValidationError{
		Code:    "font_format_invalid",
		Message: fmt.Sprintf("Valor de fuente inválido para %s: use \"cat:slug\" o \"url:https://...\".", fontSlotLabel(kind)),
	}
}

func fontSlotLabel(kind string) string {
	if kind == "mono" {
		return "fuente de datos de máquina (mono)"
	}
	return "fuente de interfaz (UI)"
}

// validateCustomFontURL exige HTTPS, host permitido (Google Fonts) y que la
// hoja sea alcanzable antes de aceptar la configuración.
func validateCustomFontURL(rawURL string) *FontValidationError {
	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return &FontValidationError{
			Code:    "font_url_invalid",
			Message: "La URL custom debe iniciar con https:// y apuntar a una hoja CSS válida.",
		}
	}
	if u.Host != allowedFontCSSHost {
		return &FontValidationError{
			Code:    "font_url_host_not_allowed",
			Message: fmt.Sprintf("Solo se permiten hojas CSS de %s (dominio permitido por gobernanza).", allowedFontCSSHost),
		}
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(rawURL)
	if err != nil {
		return &FontValidationError{
			Code:    "font_url_unreachable",
			Message: "No se pudo alcanzar la URL de la fuente. Verifica que esté disponible e inténtalo de nuevo.",
		}
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &FontValidationError{
			Code:    "font_url_unreachable",
			Message: fmt.Sprintf("La URL de la fuente respondió con estado %d. Verifica que la hoja CSS exista.", resp.StatusCode),
		}
	}
	return nil
}
