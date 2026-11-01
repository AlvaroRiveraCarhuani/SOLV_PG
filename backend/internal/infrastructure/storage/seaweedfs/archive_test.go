package seaweedfs

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func writeTree(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, content := range files {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func listArchive(t *testing.T, data []byte) []string {
	t.Helper()
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	tr := tar.NewReader(gz)
	var names []string
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, hdr.Name)
	}
	sort.Strings(names)
	return names
}

func TestPackageExercise_ExcludesConfiguredPaths(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, map[string]string{
		"main.go":                   "package main",
		"src/app.go":                "package src",
		".vscode/settings.json":     "{}",
		".solv_initialized":         "1",
		".solv_backups/x/old.go":    "old",
		".git/HEAD":                 "ref",
		"node_modules/pkg/index.js": "x",
	})
	opts := ArchiveOptions{
		ExcludePrefixes: []string{".vscode/", ".solv_*", ".git/", "node_modules/"},
		MaxFiles:        100,
	}
	var buf bytes.Buffer
	n, err := PackageExercise(root, opts, &buf)
	if err != nil {
		t.Fatalf("PackageExercise: %v", err)
	}
	got := listArchive(t, buf.Bytes())
	want := []string{"main.go", "src/app.go"}
	if n != len(want) || len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("archivos = %v (n=%d), want %v", got, n, want)
	}
}

func TestPackageExercise_RejectsTooManyFiles(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, map[string]string{"a.go": "a", "b.go": "b", "c.go": "c"})
	var buf bytes.Buffer
	_, err := PackageExercise(root, ArchiveOptions{MaxFiles: 2}, &buf)
	if err == nil {
		t.Fatal("se esperaba error por superar el tope de archivos")
	}
}

// El snapshot viaja del entorno del docente al del alumno: las rutas que escapan
// del volumen y los archivos internos de la plataforma no pueden llegar.
func TestUnpackSnapshot_DropsUnsafeAndPlatformPaths(t *testing.T) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	entries := map[string]string{
		"main.go":               "package main\n",
		"src/app.go":            "package src\n",
		"../escape.go":          "x",
		"/etc/passwd":           "x",
		".solv_backups/a/b.go":  "x",
		"node_modules/b.js":     "x",
		".vscode/settings.json": "{}",
	}
	for name, content := range entries {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(content)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}

	files, err := UnpackSnapshot(buf.Bytes(), 100)
	if err != nil {
		t.Fatalf("UnpackSnapshot: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("archivos = %+v, se esperaban solo los del ejercicio", files)
	}
	paths := []string{files[0].Path, files[1].Path}
	sort.Strings(paths)
	if paths[0] != "main.go" || paths[1] != "src/app.go" {
		t.Errorf("rutas = %v", paths)
	}
}

func TestUnpackSnapshot_RejectsTooManyFiles(t *testing.T) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for _, name := range []string{"a.go", "b.go"} {
		content := []byte("x")
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: 1, Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(content); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}

	if _, err := UnpackSnapshot(buf.Bytes(), 1); err == nil {
		t.Fatal("se esperaba error por superar el tope de archivos")
	}
}

// Al emitir un avance el contenido llega del entorno del docente: las
// exclusiones configuradas se aplican antes de empaquetar.
func TestPackageFiles_AppliesConfiguredExclusions(t *testing.T) {
	files := map[string]string{
		"main.go":                   "package main\n",
		"src/app.go":                "package src\n",
		".vscode/settings.json":     "{}",
		".solv_initialized":         "1",
		".solv_backups/old/a.go":    "old",
		"node_modules/pkg/index.js": "x",
	}
	opts := ArchiveOptions{
		ExcludePrefixes: []string{".vscode/", ".solv_*", ".git/", "node_modules/"},
		MaxFiles:        100,
	}
	var buf bytes.Buffer
	n, err := PackageFiles(files, opts, &buf)
	if err != nil {
		t.Fatalf("PackageFiles: %v", err)
	}
	got := listArchive(t, buf.Bytes())
	want := []string{"main.go", "src/app.go"}
	if n != len(want) || len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("archivos = %v (n=%d), want %v", got, n, want)
	}
}

func TestPackageFiles_RejectsTooManyFiles(t *testing.T) {
	files := map[string]string{"a.go": "a", "b.go": "b"}
	var buf bytes.Buffer
	if _, err := PackageFiles(files, ArchiveOptions{MaxFiles: 1}, &buf); err == nil {
		t.Fatal("se esperaba error por superar el tope de archivos")
	}
}
