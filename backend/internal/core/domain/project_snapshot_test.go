package domain

import (
	"fmt"
	"sort"
	"strings"
	"testing"
)

func TestParseProjectSnapshotSingleFile(t *testing.T) {
	code := "# === FILE: app.js ===\nconsole.log(1);\n"

	files, ok := ParseProjectSnapshot(code, "")
	if !ok {
		t.Fatal("se esperaba un snapshot válido")
	}
	if len(files) != 1 {
		t.Fatalf("se esperaba 1 archivo, se obtuvieron %d", len(files))
	}

	file := files[0]
	if file.Path != "app.js" {
		t.Errorf("path: esperado app.js, obtenido %q", file.Path)
	}
	if file.Language != "javascript" {
		t.Errorf("language: esperado javascript, obtenido %q", file.Language)
	}
	if file.Content != "console.log(1);\n" {
		t.Errorf("content: esperado %q, obtenido %q", "console.log(1);\n", file.Content)
	}
	if file.StartLine != 2 {
		t.Errorf("start_line: esperado 2, obtenido %d", file.StartLine)
	}
	if file.LineCount != 1 {
		t.Errorf("line_count: esperado 1, obtenido %d", file.LineCount)
	}
}

func TestParseProjectSnapshotTwoFilesWithOffsets(t *testing.T) {
	code := "# === FILE: app.js ===\n" +
		"import express from \"express\";\n" +
		"\n" +
		"# === FILE: package.json ===\n" +
		"{\"name\":\"practica-3\"}\n"

	files, ok := ParseProjectSnapshot(code, "")
	if !ok {
		t.Fatal("se esperaba un snapshot válido")
	}
	if len(files) != 2 {
		t.Fatalf("se esperaban 2 archivos, se obtuvieron %d", len(files))
	}

	app := files[0]
	if app.Path != "app.js" || app.StartLine != 2 || app.LineCount != 1 {
		t.Errorf("app.js inesperado: %+v", app)
	}
	if app.Content != "import express from \"express\";\n" {
		t.Errorf("content de app.js: %q", app.Content)
	}

	pkg := files[1]
	if pkg.Path != "package.json" || pkg.StartLine != 5 || pkg.LineCount != 1 {
		t.Errorf("package.json inesperado: %+v", pkg)
	}
	if pkg.Content != "{\"name\":\"practica-3\"}\n" {
		t.Errorf("content de package.json: %q", pkg.Content)
	}
}

func TestParseProjectSnapshotEmptyFile(t *testing.T) {
	code := "# === FILE: empty.py ===\n" +
		"\n" +
		"\n" +
		"# === FILE: main.py ===\n" +
		"print(1)\n"

	files, ok := ParseProjectSnapshot(code, "")
	if !ok {
		t.Fatal("se esperaba un snapshot válido")
	}
	if len(files) != 2 {
		t.Fatalf("se esperaban 2 archivos, se obtuvieron %d", len(files))
	}

	empty := files[0]
	if empty.Path != "empty.py" || empty.Content != "" || empty.LineCount != 0 {
		t.Errorf("archivo vacío inesperado: %+v", empty)
	}
	if empty.StartLine != 2 {
		t.Errorf("start_line de archivo vacío: esperado 2, obtenido %d", empty.StartLine)
	}

	main := files[1]
	if main.Content != "print(1)\n" || main.LineCount != 1 || main.StartLine != 5 {
		t.Errorf("main.py inesperado: %+v", main)
	}
}

func TestParseProjectSnapshotLastEmptyFile(t *testing.T) {
	code := "# === FILE: empty.py ===\n\n"

	files, ok := ParseProjectSnapshot(code, "")
	if !ok {
		t.Fatal("se esperaba un snapshot válido")
	}
	if len(files) != 1 {
		t.Fatalf("se esperaba 1 archivo, se obtuvieron %d", len(files))
	}
	if files[0].Content != "" || files[0].LineCount != 0 {
		t.Errorf("el último archivo vacío debe quedar sin contenido: %+v", files[0])
	}
}

func TestParseProjectSnapshotInvalidFormats(t *testing.T) {
	cases := []struct {
		name string
		code string
	}{
		{name: "vacío", code: ""},
		{name: "sin cabeceras", code: "print(1)\n"},
		{name: "cabecera sin cerrar", code: "# === FILE: a.py\nprint(1)\n"},
		{name: "cabecera con ruta vacía", code: "# === FILE:  ===\nprint(1)\n"},
		{name: "ruta absoluta", code: "# === FILE: /etc/passwd ===\nroot\n"},
		{name: "ruta con punto punto", code: "# === FILE: ../secret.py ===\nprint(1)\n"},
		{name: "ruta con punto punto interno", code: "# === FILE: src/../secret.py ===\nprint(1)\n"},
		{name: "ruta con backslash", code: "# === FILE: src\\main.py ===\nprint(1)\n"},
		{name: "ruta con caracter de control", code: "# === FILE: a\x00b.py ===\nprint(1)\n"},
		{
			name: "ruta duplicada",
			code: "# === FILE: a.py ===\nprint(1)\n\n# === FILE: a.py ===\nprint(2)\n",
		},
		{name: "cabecera indentada", code: "  # === FILE: a.py ===\nprint(1)\n"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			files, ok := ParseProjectSnapshot(tc.code, "")
			if ok {
				t.Fatalf("se esperaba ok=false, se obtuvieron %d archivos", len(files))
			}
			if files != nil {
				t.Fatalf("se esperaba files=nil, se obtuvo %#v", files)
			}
		})
	}
}

func TestParseProjectSnapshotNormalizesCRLF(t *testing.T) {
	code := "# === FILE: a.py ===\r\nprint(1)\r\n"

	files, ok := ParseProjectSnapshot(code, "")
	if !ok {
		t.Fatal("se esperaba un snapshot válido")
	}
	if len(files) != 1 || files[0].Content != "print(1)\n" {
		t.Fatalf("CRLF no normalizado: %+v", files)
	}
	if files[0].LineCount != 1 || files[0].StartLine != 2 {
		t.Errorf("offset inesperado tras CRLF: %+v", files[0])
	}
}

func TestParseProjectSnapshotHeaderLikeLineInsideContent(t *testing.T) {
	code := "# === FILE: a.py ===\n" +
		"print(1)\n" +
		"  # === FILE: fake.py ===\n" +
		"print(2)\n"

	files, ok := ParseProjectSnapshot(code, "")
	if !ok {
		t.Fatal("se esperaba un snapshot válido")
	}
	if len(files) != 1 {
		t.Fatalf("se esperaba 1 archivo, se obtuvieron %d", len(files))
	}
	if files[0].Content != "print(1)\n  # === FILE: fake.py ===\nprint(2)\n" {
		t.Errorf("content inesperado: %q", files[0].Content)
	}
	if files[0].LineCount != 3 {
		t.Errorf("line_count: esperado 3, obtenido %d", files[0].LineCount)
	}
}

func TestParseProjectSnapshotPreservesTrailingBlankLines(t *testing.T) {
	code := "# === FILE: a.py ===\n" +
		"a\n" +
		"\n" +
		"\n" +
		"# === FILE: b.py ===\n" +
		"b\n"

	files, ok := ParseProjectSnapshot(code, "")
	if !ok {
		t.Fatal("se esperaba un snapshot válido")
	}
	if len(files) != 2 {
		t.Fatalf("se esperaban 2 archivos, se obtuvieron %d", len(files))
	}
	if files[0].Content != "a\n\n" {
		t.Errorf("content de a.py: %q", files[0].Content)
	}
	if files[0].LineCount != 2 {
		t.Errorf("line_count de a.py: esperado 2, obtenido %d", files[0].LineCount)
	}
	if files[1].Content != "b\n" || files[1].LineCount != 1 {
		t.Errorf("b.py inesperado: %+v", files[1])
	}
}

func TestParseProjectSnapshotLanguageFallback(t *testing.T) {
	code := "# === FILE: script.xyz ===\nprint(1)\n"

	files, ok := ParseProjectSnapshot(code, "go")
	if !ok || len(files) != 1 {
		t.Fatalf("snapshot inválido: ok=%v files=%v", ok, files)
	}
	if files[0].Language != "go" {
		t.Errorf("language de extensión desconocida: esperado go, obtenido %q", files[0].Language)
	}

	files, ok = ParseProjectSnapshot(code, "")
	if !ok || len(files) != 1 {
		t.Fatalf("snapshot inválido sin lenguaje por defecto: ok=%v files=%v", ok, files)
	}
	if files[0].Language != "plaintext" {
		t.Errorf("language sin defecto: esperado plaintext, obtenido %q", files[0].Language)
	}

	headerFile, ok := ParseProjectSnapshot("# === FILE: include.h ===\nint x;\n", "")
	if !ok || len(headerFile) != 1 || headerFile[0].Language != "cpp" {
		t.Errorf("extensión .h: esperado cpp, obtenido %+v", headerFile)
	}
}

func TestParseProjectSnapshotSortsByPath(t *testing.T) {
	code := "# === FILE: z.py ===\nz\n" +
		"\n" +
		"# === FILE: a.py ===\na\n"

	files, ok := ParseProjectSnapshot(code, "")
	if !ok {
		t.Fatal("se esperaba un snapshot válido")
	}
	if len(files) != 2 || files[0].Path != "a.py" || files[1].Path != "z.py" {
		t.Fatalf("orden inesperado: %+v", files)
	}
	// El orden de salida cambia, pero los offsets siguen apuntando al snapshot.
	if files[0].StartLine != 5 || files[1].StartLine != 2 {
		t.Errorf("offsets inesperados tras ordenar: %+v", files)
	}
}

func TestParseProjectSnapshotLimits(t *testing.T) {
	if _, ok := ParseProjectSnapshot(strings.Repeat("a", MaxProjectSnapshotBytes+1), ""); ok {
		t.Error("un snapshot mayor a 2 MiB debe rechazarse")
	}

	var builder strings.Builder
	for i := 0; i <= MaxProjectSnapshotFiles; i++ {
		builder.WriteString(fmt.Sprintf("# === FILE: f%03d.py ===\nprint(%d)\n\n", i, i))
	}
	if _, ok := ParseProjectSnapshot(builder.String(), ""); ok {
		t.Errorf("un snapshot con más de %d archivos debe rechazarse", MaxProjectSnapshotFiles)
	}
}

func TestParseProjectSnapshotRoundTrip(t *testing.T) {
	original := []ProjectFile{
		{Path: "README.md", Content: "# Hola\n"},
		{Path: "a.py", Content: "print(1)\n"},
		{Path: "src/b.ts", Content: "export const x = 1;\n"},
	}
	code := buildSnapshotForTest(original)

	files, ok := ParseProjectSnapshot(code, "")
	if !ok {
		t.Fatal("se esperaba un snapshot válido")
	}
	if len(files) != len(original) {
		t.Fatalf("se esperaban %d archivos, se obtuvieron %d", len(original), len(files))
	}
	if rebuilt := buildSnapshotForTest(files); rebuilt != code {
		t.Errorf("la reconstrucción no coincide:\noriginal: %q\nrebuilt:  %q", code, rebuilt)
	}
}

func buildSnapshotForTest(files []ProjectFile) string {
	sorted := make([]ProjectFile, len(files))
	copy(sorted, files)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Path < sorted[j].Path })

	var builder strings.Builder
	for _, file := range sorted {
		if builder.Len() > 0 {
			builder.WriteString("\n")
		}
		builder.WriteString(fmt.Sprintf("# === FILE: %s ===\n", file.Path))
		builder.WriteString(file.Content)
		if !strings.HasSuffix(file.Content, "\n") {
			builder.WriteString("\n")
		}
	}
	return builder.String()
}
