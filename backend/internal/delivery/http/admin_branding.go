package httpdelivery

import (
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strings"

	"solv-backend/internal/core/services"
	"solv-backend/internal/delivery/http/middleware"
)

type UpdateBrandingDTO struct {
	LogoURL            string `json:"logo_url"`
	InstitutionName    string `json:"institution_name"`
	TenantPrimaryColor string `json:"tenant_primary_color"`
	SupportEmail       string `json:"support_email"`
	// Tipografía white-label (proposal tenant-typography): "", "cat:slug" o "url:https://..."
	FontSansFamily string `json:"font_sans_family"`
	FontMonoFamily string `json:"font_mono_family"`
}

// brandingHexPattern misma regla que el frontend (validate() en
// admin-config-identidad.service.ts): #RRGGBB estricto, 6 dígitos.
var brandingHexPattern = regexp.MustCompile(`^#([A-Fa-f0-9]{6})$`)

// brandingCacheBusterPattern quita el ?t=<ts> (o &t=<ts>) que el frontend
// agrega tras subir el logo para evitar la caché de 5 min del navegador.
var brandingCacheBusterPattern = regexp.MustCompile(`([?&])t=\d+$`)

func (h *AdminHandler) UpdateBranding(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantIDFromContext(r.Context())
	if err != nil || tenantID == "" {
		http.Error(w, `{"error":"Tenant ID missing in context"}`, http.StatusUnauthorized)
		return
	}

	var dto UpdateBrandingDTO
	// Se lee el cuerpo crudo para detectar presencia explícita de claves:
	// support_email "" significa "sin soporte" y debe limpiar el valor
	// vigente (el resto de campos vacíos siguen sin pisar, contrato merge).
	var rawBody map[string]json.RawMessage
	bodyBytes, readErr := io.ReadAll(r.Body)
	if readErr != nil {
		http.Error(w, `{"error":"Invalid request payload"}`, http.StatusBadRequest)
		return
	}
	if err := json.Unmarshal(bodyBytes, &dto); err != nil {
		http.Error(w, `{"error":"Invalid request payload"}`, http.StatusBadRequest)
		return
	}
	_ = json.Unmarshal(bodyBytes, &rawBody)

	tenant, err := h.tenantRepo.GetByID(r.Context(), tenantID)
	if err != nil || tenant == nil {
		http.Error(w, `{"error":"Tenant not found"}`, http.StatusNotFound)
		return
	}

	var currentConfig map[string]interface{}
	if len(tenant.Config) > 0 {
		if err := json.Unmarshal(tenant.Config, &currentConfig); err != nil {
			currentConfig = make(map[string]interface{})
		}
	}
	if currentConfig == nil {
		currentConfig = make(map[string]interface{})
	}

	if dto.LogoURL != "" {
		// El frontend agrega ?t=<ts> como cache-buster tras subir el logo;
		// se persiste la URL canónica sin ese query.
		canonicalLogo := brandingCacheBusterPattern.ReplaceAllString(dto.LogoURL, "")
		currentConfig["logo_url"] = canonicalLogo
	}
	if dto.InstitutionName != "" {
		currentConfig["institution_name"] = dto.InstitutionName
	}
	if dto.TenantPrimaryColor != "" {
		if !brandingHexPattern.MatchString(strings.TrimSpace(dto.TenantPrimaryColor)) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnprocessableEntity)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error":   "color_invalid",
				"message": "tenant_primary_color debe usar el formato #RRGGBB (ej. #2563EB).",
			})
			return
		}
		currentConfig["tenant_primary_color"] = dto.TenantPrimaryColor
	}
	// support_email es opcional por universidad: clave presente con "" limpia
	// el valor (oculta el bloque de ayuda); clave ausente no pisa.
	if rawBody != nil {
		if _, present := rawBody["support_email"]; present {
			email := strings.TrimSpace(dto.SupportEmail)
			if email != "" && !strings.Contains(email, "@") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnprocessableEntity)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error":   "email_invalid",
					"message": "support_email no tiene un formato válido.",
				})
				return
			}
			currentConfig["support_email"] = email
		}
	} else if dto.SupportEmail != "" {
		currentConfig["support_email"] = dto.SupportEmail
	}

	// Tipografía white-label: validación centralizada (catálogo curado o URL
	// custom de dominio permitido y alcanzable). "" = sin cambio del valor vigente.
	if dto.FontSansFamily != "" || dto.FontMonoFamily != "" {
		_, _, fontErr := services.ResolveFontConfig(dto.FontSansFamily, dto.FontMonoFamily)
		if fontErr != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnprocessableEntity)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error":   fontErr.Code,
				"message": fontErr.Message,
			})
			return
		}
		if dto.FontSansFamily != "" {
			currentConfig["font_sans_family"] = dto.FontSansFamily
		}
		if dto.FontMonoFamily != "" {
			currentConfig["font_mono_family"] = dto.FontMonoFamily
		}
	}

	newConfigBytes, err := json.Marshal(currentConfig)
	if err != nil {
		http.Error(w, `{"error":"Failed to serialize config"}`, http.StatusInternalServerError)
		return
	}

	if err := h.tenantRepo.UpdateConfig(r.Context(), tenantID, newConfigBytes); err != nil {
		http.Error(w, `{"error":"Failed to update branding in database"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "updated",
		"message": "Branding configuration updated successfully",
		"config":  currentConfig,
	})
}
