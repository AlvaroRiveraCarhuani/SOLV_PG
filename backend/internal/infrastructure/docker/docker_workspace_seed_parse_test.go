package docker

import (
	"errors"
	"testing"

	"solv-backend/internal/core/domain"
)

// Un tar ilegible no puede terminar en 500: el contenido queda sin determinar y
// el entorno debe ir a cuarentena, igual que cuando la lectura excede el tope.
func TestParseWorkspaceSeedTarRejectsCorruptArchiveAsAmbiguous(t *testing.T) {
	if _, err := parseWorkspaceSeedTar([]byte("esto no es un tar"), "", 1024, 8); !errors.Is(err, domain.ErrWorkspaceSeedVolumeAmbiguous) {
		t.Fatalf("se esperaba ErrWorkspaceSeedVolumeAmbiguous, llegó: %v", err)
	}
}
