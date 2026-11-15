package httpdelivery

import (
	"context"
	"net/http"

	"solv-backend/internal/version"
)

// MigrationVersionReader devuelve la última migración aplicada en la base.
// Se inyecta como función para que el handler no dependa de sqlx y se pueda
// probar sin base de datos.
type MigrationVersionReader func(ctx context.Context) (int64, error)

// VersionHandler informa qué versión está corriendo y hasta qué migración
// llegó la base. Es el dato que hace posible diagnosticar una instalación
// ajena y detectar si quedó atrás respecto de la imagen publicada.
type VersionHandler struct {
	migrationVersion MigrationVersionReader
}

type versionResponse struct {
	Version          string `json:"version"`
	Commit           string `json:"commit"`
	BuildDate        string `json:"build_date"`
	GoVersion        string `json:"go_version"`
	MigrationVersion *int64 `json:"migration_version"`
	Database         string `json:"database"`
}

func NewVersionHandler(read MigrationVersionReader) *VersionHandler {
	return &VersionHandler{migrationVersion: read}
}

// GetVersion responde 200 aunque la base no esté disponible: el estado de la
// base es parte de la respuesta, no la causa de un error. Así el chequeo de
// salud distingue "el proceso vive pero no llega a Postgres" de "el proceso
// no responde".
func (h *VersionHandler) GetVersion(w http.ResponseWriter, r *http.Request) {
	resp := versionResponse{
		Version:   version.Version,
		Commit:    version.Commit,
		BuildDate: version.BuildDate,
		GoVersion: version.GoVersion(),
		Database:  "ok",
	}

	if h.migrationVersion != nil {
		if applied, err := h.migrationVersion(r.Context()); err != nil {
			resp.Database = "unavailable"
		} else {
			resp.MigrationVersion = &applied
		}
	}

	w.Header().Set("Cache-Control", "no-store")
	SendJSON(w, http.StatusOK, resp, "Versión del sistema")
}
