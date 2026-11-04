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

func TestParsePracticeEnvironmentSnapshotTar_FiltersAndSorts(t *testing.T) {
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

	files, err := parsePracticeEnvironmentSnapshotTar(raw, 512*1024)
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

func TestParsePracticeEnvironmentSnapshotTar_EnforcesByteBudget(t *testing.T) {
	raw := buildTestTar(t, []struct {
		name    string
		content []byte
	}{
		{"./a.py", []byte("aaaa")},
		{"./b.py", []byte("bbbb")},
	})

	files, err := parsePracticeEnvironmentSnapshotTar(raw, 6)
	if err != nil {
		t.Fatalf("error inesperado al parsear el tar: %v", err)
	}
	if len(files) != 1 || files[0].Path != "a.py" {
		t.Fatalf("esperado solo el primer archivo dentro del presupuesto, obtenido: %+v", files)
	}
}

func TestIsDirectPracticeEnvironmentChild(t *testing.T) {
	cases := []struct {
		name string
		rel  string
		root string
		want bool
	}{
		{name: "hija directa", rel: ".solv_backups/abc", root: ".solv_backups", want: true},
		{name: "raíz sola", rel: ".solv_backups", root: ".solv_backups", want: false},
		{name: "nieta", rel: ".solv_backups/abc/main.py", root: ".solv_backups", want: false},
		{name: "otra carpeta", rel: "src/abc", root: ".solv_backups", want: false},
		{name: "escape", rel: "../.solv_backups/abc", root: ".solv_backups", want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isDirectPracticeEnvironmentChild(tc.rel, tc.root); got != tc.want {
				t.Fatalf("isDirectPracticeEnvironmentChild(%q, %q) = %v, quería %v", tc.rel, tc.root, got, tc.want)
			}
		})
	}
}

func TestShellQuote(t *testing.T) {
	cases := map[string]string{
		"/environment/.solv_backups/abc": "'/environment/.solv_backups/abc'",
		"/environment/a'b":               "'/environment/a'\\''b'",
	}
	for value, want := range cases {
		if got := shellQuote(value); got != want {
			t.Fatalf("shellQuote(%q) = %q, quería %q", value, got, want)
		}
	}
}
