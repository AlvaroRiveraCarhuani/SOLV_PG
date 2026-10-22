package docker

import (
	"archive/tar"
	"bytes"
	"testing"
)

func buildTestTar(t *testing.T, entries []struct {
	name    string
	content []byte
}) []byte {
	t.Helper()

	var buf bytes.Buffer
	writer := tar.NewWriter(&buf)
	for _, entry := range entries {
		if err := writer.WriteHeader(&tar.Header{
			Typeflag: tar.TypeReg,
			Name:     entry.name,
			Mode:     0o644,
			Size:     int64(len(entry.content)),
		}); err != nil {
			t.Fatalf("no se pudo escribir el header %q: %v", entry.name, err)
		}
		if _, err := writer.Write(entry.content); err != nil {
			t.Fatalf("no se pudo escribir el contenido %q: %v", entry.name, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("no se pudo cerrar el tar: %v", err)
	}
	return buf.Bytes()
}

func TestParseWorkspaceSnapshotTar_FiltersAndSorts(t *testing.T) {
	raw := buildTestTar(t, []struct {
		name    string
		content []byte
	}{
		{"./utils.py", []byte("x = 2\n")},
		{"./main.py", []byte("print(1)\n")},
		{"./node_modules/lib/index.js", []byte("module.exports = 1\n")},
		{"./.git/HEAD", []byte("ref: refs/heads/main\n")},
		{"./__pycache__/main.cpython-312.pyc", []byte("binario")},
		{"./logo.png", []byte{0x89, 0x50, 0x4e, 0x47, 0x00, 0x01}},
	})

	files, err := parseWorkspaceSnapshotTar(raw, 512*1024)
	if err != nil {
		t.Fatalf("error inesperado al parsear el tar: %v", err)
	}

	if len(files) != 2 {
		t.Fatalf("esperados 2 archivos de código, obtenidos %d: %+v", len(files), files)
	}
	if files[0].Path != "main.py" || files[1].Path != "utils.py" {
		t.Errorf("orden inesperado: %q, %q", files[0].Path, files[1].Path)
	}
}

func TestParseWorkspaceSnapshotTar_EnforcesByteBudget(t *testing.T) {
	raw := buildTestTar(t, []struct {
		name    string
		content []byte
	}{
		{"./a.py", []byte("aaaa")},
		{"./b.py", []byte("bbbb")},
	})

	files, err := parseWorkspaceSnapshotTar(raw, 6)
	if err != nil {
		t.Fatalf("error inesperado al parsear el tar: %v", err)
	}
	if len(files) != 1 || files[0].Path != "a.py" {
		t.Fatalf("esperado solo el primer archivo dentro del presupuesto, obtenido: %+v", files)
	}
}
