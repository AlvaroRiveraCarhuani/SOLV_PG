package domain

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
)

var (
	// ErrWorkspaceProjectArchiveInvalid rechaza un archivo que no se puede
	// descomprimir sin ambigüedad: rutas inseguras, entradas especiales o un
	// formato que no es un tar.gz.
	ErrWorkspaceProjectArchiveInvalid = errors.New("el archivo del proyecto no es válido")
	// ErrWorkspaceProjectArchiveTooLarge se emite antes de escribir nada.
	ErrWorkspaceProjectArchiveTooLarge = errors.New("el archivo del proyecto excede el tamaño permitido")
	// ErrWorkspaceProjectArchiveStoreUnavailable indica que el depósito de
	// objetos no está configurado. La importación grande queda apagada y el
	// camino liviano sigue funcionando.
	ErrWorkspaceProjectArchiveStoreUnavailable = errors.New("el depósito de archivos del proyecto no está disponible")
	// ErrWorkspaceProjectArchiveDisabled mantiene la carga cerrada hasta que la
	// publicación y la siembra por extracción estén verificadas de punta a punta.
	ErrWorkspaceProjectArchiveDisabled = errors.New("la carga de proyectos grandes está deshabilitada")
)

// WorkspaceProjectArtifact identifica el tar.gz que contiene un proyecto
// inicial de árbol grande. El hash es la identidad: dos cargas iguales producen
// la misma clave y no se duplican en el depósito.
type WorkspaceProjectArtifact struct {
	Key    string `json:"key"`
	SHA256 string `json:"sha256"`
	Bytes  int64  `json:"bytes"`
	Files  int    `json:"files"`
}

// Valid comprueba que el artefacto esté completo antes de referenciarlo.
func (a WorkspaceProjectArtifact) Valid() error {
	if strings.TrimSpace(a.Key) == "" {
		return fmt.Errorf("%w: falta la clave del artefacto", ErrWorkspaceProjectArchiveInvalid)
	}
	if len(a.SHA256) != 64 {
		return fmt.Errorf("%w: el hash del artefacto no es un SHA-256", ErrWorkspaceProjectArchiveInvalid)
	}
	if a.Bytes <= 0 || a.Files <= 0 {
		return fmt.Errorf("%w: el artefacto no declara tamaño ni archivos", ErrWorkspaceProjectArchiveInvalid)
	}
	return nil
}

// WorkspaceProjectArtifactStore es el depósito de archivos. Put devuelve el
// SHA-256 y el tamaño de lo que guardó; Get entrega un lector que el llamador
// cierra. La implementación vive fuera del dominio.
type WorkspaceProjectArtifactStore interface {
	Put(ctx context.Context, key string, body io.Reader) (string, int64, error)
	Get(ctx context.Context, key string) (io.ReadCloser, error)
}

// WorkspaceProjectArtifactKey arma la clave del objeto. Incluye tenant y
// ejercicio para que una clave no se pueda reutilizar en otro ámbito, y el hash
// para que la misma carga no se duplique. No depende del número de versión: el
// artefacto existe antes de publicar.
func WorkspaceProjectArtifactKey(tenantID, exerciseID, sha256 string) (string, error) {
	tenantID = strings.TrimSpace(tenantID)
	exerciseID = strings.TrimSpace(exerciseID)
	sha256 = strings.TrimSpace(sha256)
	if tenantID == "" || exerciseID == "" || len(sha256) != 64 {
		return "", fmt.Errorf("%w: no se puede construir la clave del artefacto", ErrWorkspaceProjectArchiveInvalid)
	}
	key := path.Join("workspace-projects", tenantID, exerciseID, sha256+".tar.gz")
	if strings.Contains(key, "..") {
		return "", fmt.Errorf("%w: la clave del artefacto sale del prefijo", ErrWorkspaceProjectArchiveInvalid)
	}
	return key, nil
}
