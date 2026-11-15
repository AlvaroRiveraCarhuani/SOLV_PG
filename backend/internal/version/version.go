// Package version expone los metadatos de compilación del binario.
//
// Los tres valores se inyectan con -ldflags en la imagen (ver
// backend/Dockerfile). En `go run` quedan en sus valores por defecto: eso es
// intencional, porque un build local no es una versión que se pueda soportar.
package version

import "runtime"

var (
	// Version es el tag de la imagen o de git describe.
	Version = "dev"
	// Commit es el hash corto del árbol desde el que se compiló.
	Commit = "unknown"
	// BuildDate es el instante UTC de compilación en RFC 3339.
	BuildDate = "unknown"
)

// GoVersion devuelve la versión del toolchain que compiló el binario.
func GoVersion() string {
	return runtime.Version()
}
