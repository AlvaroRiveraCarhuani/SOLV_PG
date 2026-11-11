package domain

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

var ErrInvalidDependencyLink = errors.New("enlace de dependencias inválido")

// WorkspaceDependencyTree es el árbol preparado de una versión y las rutas que
// deben aparecer en el volumen del alumno. El árbol contiene el proyecto y sus
// dependencias; enlazar el árbol completo pisaría el proyecto ya sembrado, así
// que solo viajan las rutas declaradas en dependency_paths.
type WorkspaceDependencyTree struct {
	// RootPath es la ruta absoluta del árbol en el nodo.
	RootPath string
	// Key identifica la preparación; queda escrita en el volumen para no
	// repetir el enlace sobre un volumen que ya lo recibió.
	Key   string
	Paths []string
}

// Validate rechaza rutas que puedan tocar algo que no sea una dependencia.
func (t WorkspaceDependencyTree) Validate() error {
	root := strings.TrimSpace(t.RootPath)
	if root == "" || !filepath.IsAbs(root) || strings.Contains(root, "..") {
		return fmt.Errorf("%w: la raíz del árbol debe ser absoluta y no salir de su ruta", ErrInvalidDependencyLink)
	}
	if strings.TrimSpace(t.Key) == "" {
		return fmt.Errorf("%w: falta la clave de la preparación", ErrInvalidDependencyLink)
	}
	if len(t.Paths) == 0 {
		return fmt.Errorf("%w: no hay rutas de dependencias para enlazar", ErrInvalidDependencyLink)
	}
	if len(t.Paths) > 32 {
		return fmt.Errorf("%w: demasiadas rutas de dependencias", ErrInvalidDependencyLink)
	}
	for _, raw := range t.Paths {
		clean := filepath.Clean(strings.TrimSpace(raw))
		if clean == "" || clean == "." || filepath.IsAbs(clean) || !filepath.IsLocal(clean) {
			return fmt.Errorf("%w: ruta de dependencia no válida: %q", ErrInvalidDependencyLink, raw)
		}
		if clean == ".solv_workspace" || strings.HasPrefix(clean, ".solv_workspace"+string(filepath.Separator)) {
			return fmt.Errorf("%w: la ruta %q pertenece al diario del volumen", ErrInvalidDependencyLink, raw)
		}
		if clean != strings.TrimSpace(raw) {
			return fmt.Errorf("%w: la ruta %q no está normalizada", ErrInvalidDependencyLink, raw)
		}
	}
	return nil
}

// WorkspaceDependencyLinker es una capacidad opcional del orquestador. Copia las
// rutas preparadas al volumen con enlaces duros cuando el sistema de archivos lo
// permite, y con copia normal cuando no.
type WorkspaceDependencyLinker interface {
	LinkWorkspaceDependencies(ctx context.Context, volumeName string, tree WorkspaceDependencyTree) error
}
