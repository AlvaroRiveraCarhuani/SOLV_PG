package httpdelivery

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"solv-backend/internal/core/domain"
)

// TenantLogoHandler gestiona la subida del imagotipo institucional
// (submódulo 14.6 / wireframe CONFIGURACION.md). El archivo se almacena en
// disco bajo LOGO_UPLOAD_DIR (default: ./uploads/logos) y la URL pública
// resultante se persiste en tenants.config como logo_url.
type TenantLogoHandler struct {
	tenantRepo domain.TenantRepository
	uploadDir  string
}

func NewTenantLogoHandler(tenantRepo domain.TenantRepository) *TenantLogoHandler {
	dir := os.Getenv("LOGO_UPLOAD_DIR")
	if dir == "" {
		dir = "./uploads/logos"
	}
	_ = os.MkdirAll(dir, 0750)
	return &TenantLogoHandler{tenantRepo: tenantRepo, uploadDir: dir}
}

const (
	maxLogoBytes = 2 << 20 // 2 MB, suficiente para un imagotipo vectorial/raster
)

var allowedLogoExtensions = map[string]string{
	".png":  "image/png",
	".svg":  "image/svg+xml",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".webp": "image/webp",
}

// UploadLogo procesa multipart/form-data (campo "logo"), valida extensión y
// tamaño, guarda el archivo con nombre derivado del tenant y actualiza
// logo_url en tenants.config.
func (h *TenantLogoHandler) UploadLogo(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantFromCtx(r)
	if tenantID == "" {
		SendError(w, http.StatusUnauthorized, "unauthorized", "Tenant ID requerido")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxLogoBytes+1024)
	if err := r.ParseMultipartForm(maxLogoBytes); err != nil {
		SendError(w, http.StatusUnprocessableEntity, "file_too_large", fmt.Sprintf("El logo no debe exceder %d MB.", maxLogoBytes/(1<<20)))
		return
	}

	file, header, err := r.FormFile("logo")
	if err != nil {
		SendError(w, http.StatusBadRequest, "missing_file", "Campo multipart \"logo\" requerido")
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if _, allowed := allowedLogoExtensions[ext]; !allowed {
		SendError(w, http.StatusUnprocessableEntity, "file_type_invalid", "Formato no permitido. Usa PNG, SVG, JPG o WEBP.")
		return
	}

	// Nombre determinístico por tenant: sobrescritura controlada del imagotipo
	fileName := fmt.Sprintf("logo_%s%s", tenantID, ext)
	targetPath := filepath.Join(h.uploadDir, fileName)

	out, err := os.Create(targetPath)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al almacenar el logo")
		return
	}
	defer out.Close()

	if _, err := io.Copy(out, file); err != nil {
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al escribir el logo")
		return
	}

	// URL pública relativa servida por el propio backend (endpoint de branding público)
	publicURL := fmt.Sprintf("/api/v1/public/branding/logo/%s%s", tenantID, ext)

	if err := h.persistLogoURL(r, tenantID, publicURL); err != nil {
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al actualizar la configuración del tenant")
		return
	}

	SendJSON(w, http.StatusOK, map[string]string{
		"logo_url": publicURL,
		"file":     fileName,
	}, "Logo institucional actualizado exitosamente")
}

func (h *TenantLogoHandler) persistLogoURL(r *http.Request, tenantID, publicURL string) error {
	tenant, err := h.tenantRepo.GetByID(r.Context(), tenantID)
	if err != nil || tenant == nil {
		return fmt.Errorf("tenant not found")
	}

	var configMap map[string]interface{}
	if len(tenant.Config) > 0 {
		_ = json.Unmarshal(tenant.Config, &configMap)
	}
	if configMap == nil {
		configMap = make(map[string]interface{})
	}
	configMap["logo_url"] = publicURL

	newConfig, err := json.Marshal(configMap)
	if err != nil {
		return fmt.Errorf("error serializing config")
	}
	return h.tenantRepo.UpdateConfig(r.Context(), tenantID, newConfig)
}

// ServePublicLogo sirve el imagotipo almacenado (ruta pública, cacheable).
// El segmento {filename} tiene forma <tenantId><ext> (ej. abc123.png);
// se separa por extensión para evitar el patrón inválido {tenantId}{ext}
// que net/http ServeMux rechaza. Se valida la extensión contra la
// allowlist para evitar traversal.
func (h *TenantLogoHandler) ServePublicLogo(w http.ResponseWriter, r *http.Request) {
	filename := r.PathValue("filename")
	ext := strings.ToLower(filepath.Ext(filename))
	tenantID := strings.TrimSuffix(filename, ext)
	if tenantID == "" {
		http.Error(w, `{"error":"missing_tenant"}`, http.StatusBadRequest)
		return
	}
	contentType, allowed := allowedLogoExtensions[ext]
	if !allowed {
		http.Error(w, `{"error":"invalid_type"}`, http.StatusBadRequest)
		return
	}

	targetPath := filepath.Join(h.uploadDir, fmt.Sprintf("logo_%s%s", tenantID, ext))
	data, err := os.ReadFile(targetPath)
	if err != nil {
		http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=300") // 5 min, alineado a la caché de config
	_, _ = w.Write(data)
}
