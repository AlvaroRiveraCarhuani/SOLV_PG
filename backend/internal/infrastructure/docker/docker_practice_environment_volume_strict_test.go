package docker

import (
	"archive/tar"
	"bytes"
	"errors"
	"strings"
	"testing"

	"solv-backend/internal/core/domain"
)

// Una entrega no puede perder archivos sin avisar: la lectura estricta falla y
// dice qué archivo no entra.
func TestParsePracticeEnvironmentSnapshotTarStrict_FallaSiNoEntra(t *testing.T) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for _, file := range []struct{ name, body string }{
		{"main.py", "print('hola')"},
		{"datos/big.txt", strings.Repeat("x", 200)},
	} {
		if err := tw.WriteHeader(&tar.Header{
			Name:     file.name,
			Mode:     0o644,
			Size:     int64(len(file.body)),
			Typeflag: tar.TypeReg,
		}); err != nil {
			t.Fatalf("no se pudo armar el tar: %v", err)
		}
		if _, err := tw.Write([]byte(file.body)); err != nil {
			t.Fatalf("no se pudo escribir el tar: %v", err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("no se pudo cerrar el tar: %v", err)
	}

	_, err := parsePracticeEnvironmentSnapshotTarStrict(buf.Bytes(), 64)
	if !errors.Is(err, domain.ErrPracticeEnvironmentSnapshotTooLarge) {
		t.Fatalf("se esperaba ErrPracticeEnvironmentSnapshotTooLarge, se obtuvo: %v", err)
	}
	if !strings.Contains(err.Error(), "big.txt") {
		t.Fatalf("el error debía nombrar el archivo que no entra: %v", err)
	}

	// La lectura normal sigue recortando: la usan los avisos y los respaldos.
	files, err := parsePracticeEnvironmentSnapshotTar(buf.Bytes(), 64)
	if err != nil {
		t.Fatalf("la lectura normal no debía fallar: %v", err)
	}
	for _, file := range files {
		if strings.Contains(file.Path, "big.txt") {
			t.Fatalf("la lectura normal no debía incluir el archivo grande: %+v", files)
		}
	}
}
