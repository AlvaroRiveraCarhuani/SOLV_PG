package domain

import (
	"path"
	"sort"
	"strings"
)

const (
	// MaxProjectSnapshotBytes acota el snapshot que SpeedGrader estructura en
	// archivos. Por encima de este límite se usa el visor plano.
	MaxProjectSnapshotBytes = 2 * 1024 * 1024
	// MaxProjectSnapshotFiles acota la cantidad de archivos del árbol.
	MaxProjectSnapshotFiles = 500

	projectSnapshotHeaderPrefix = "# === FILE: "
	projectSnapshotHeaderSuffix = " ==="
)

// ProjectFile es un archivo del snapshot multiarchivo de una entrega IDE. Es un
// read model derivado de Submission.Code: no se persiste ni se usa para acceder
// al disco. StartLine es la línea 1-based de la primera línea de Content dentro
// de Code, y permite traducir los comentarios que siguen usando la línea global.
type ProjectFile struct {
	Path      string `json:"path" db:"path"`
	Language  string `json:"language" db:"language"`
	Content   string `json:"content" db:"content"`
	StartLine int    `json:"start_line" db:"start_line"`
	LineCount int    `json:"line_count" db:"line_count"`
}

type snapshotHeader struct {
	path         string
	lineStart    int
	contentStart int
}

// ParseProjectSnapshot separa el snapshot que escribe buildProjectSnapshot en
// archivos navegables. Devuelve ok=false cuando el formato no es reconocible o
// cuando una ruta no es segura; en ese caso la vista usa el visor plano de Code.
//
// El formato de entrada es una cabecera "# === FILE: <ruta> ===" al inicio de
// una línea, seguida por el contenido del archivo. Entre bloques el writer deja
// una línea en blanco que se descarta. Un archivo formado solo por líneas en
// blanco se normaliza a contenido vacío, porque el formato no distingue ese
// caso de un archivo vacío.
func ParseProjectSnapshot(code, defaultLanguage string) ([]ProjectFile, bool) {
	if code == "" || len(code) > MaxProjectSnapshotBytes {
		return nil, false
	}

	normalized := strings.ReplaceAll(code, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")

	headers := scanSnapshotHeaders(normalized)
	if len(headers) == 0 || len(headers) > MaxProjectSnapshotFiles {
		return nil, false
	}

	seen := make(map[string]struct{}, len(headers))
	files := make([]ProjectFile, 0, len(headers))
	for i, header := range headers {
		if !validSnapshotPath(header.path) {
			return nil, false
		}
		if _, exists := seen[header.path]; exists {
			return nil, false
		}
		seen[header.path] = struct{}{}

		var content string
		if i+1 < len(headers) {
			// El bloque incluye la línea en blanco que el writer agrega antes de
			// la siguiente cabecera. Se descarta solo esa, no las líneas en
			// blanco que pertenecen al archivo.
			content = strings.TrimSuffix(
				normalized[header.contentStart:headers[i+1].lineStart],
				"\n",
			)
		} else {
			content = normalized[header.contentStart:]
		}
		if strings.Trim(content, "\n") == "" {
			content = ""
		}

		files = append(files, ProjectFile{
			Path:      header.path,
			Language:  snapshotLanguage(header.path, defaultLanguage),
			Content:   content,
			StartLine: strings.Count(normalized[:header.lineStart], "\n") + 2,
			LineCount: snapshotLineCount(content),
		})
	}

	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, true
}

func scanSnapshotHeaders(text string) []snapshotHeader {
	headers := make([]snapshotHeader, 0)
	for lineStart := 0; lineStart <= len(text); {
		lineEnd := strings.IndexByte(text[lineStart:], '\n')
		line := text[lineStart:]
		nextLine := len(text)
		if lineEnd >= 0 {
			line = text[lineStart : lineStart+lineEnd]
			nextLine = lineStart + lineEnd + 1
		}

		if len(line) >= len(projectSnapshotHeaderPrefix)+len(projectSnapshotHeaderSuffix) &&
			strings.HasPrefix(line, projectSnapshotHeaderPrefix) &&
			strings.HasSuffix(line, projectSnapshotHeaderSuffix) {
			rawPath := line[len(projectSnapshotHeaderPrefix) : len(line)-len(projectSnapshotHeaderSuffix)]
			headers = append(headers, snapshotHeader{
				path:         strings.TrimSpace(rawPath),
				lineStart:    lineStart,
				contentStart: nextLine,
			})
		}

		if lineEnd < 0 {
			break
		}
		lineStart = nextLine
	}
	return headers
}

// validSnapshotPath rechaza rutas absolutas, rutas con "..", separadores de
// Windows y caracteres de control. La ruta es solo un dato de lectura: nunca se
// resuelve contra el disco.
func validSnapshotPath(filePath string) bool {
	if filePath == "" || strings.HasPrefix(filePath, "/") || strings.HasPrefix(filePath, `\`) {
		return false
	}
	if strings.Contains(filePath, `\`) {
		return false
	}
	for _, r := range filePath {
		if r < 0x20 || r == 0x7f {
			return false
		}
	}
	for _, segment := range strings.Split(filePath, "/") {
		if segment == ".." {
			return false
		}
	}
	return true
}

// ProjectFileForGlobalLine devuelve el archivo que contiene una línea global del
// snapshot, o nil si la línea cae en una cabecera, en un separador o fuera de
// los archivos. Sirve para validar que un comentario de bloque no cruce archivos.
// ProjectFileByPath devuelve el archivo con la ruta exacta, o nil si el snapshot
// no lo contiene. Es el ancla que usan los comentarios IDE.
func ProjectFileByPath(files []ProjectFile, path string) *ProjectFile {
	for i := range files {
		if files[i].Path == path {
			return &files[i]
		}
	}
	return nil
}

func ProjectFileForGlobalLine(files []ProjectFile, globalLine int) *ProjectFile {
	for i := range files {
		file := &files[i]
		if file.LineCount <= 0 {
			continue
		}
		if globalLine >= file.StartLine && globalLine < file.StartLine+file.LineCount {
			return file
		}
	}
	return nil
}

func snapshotLineCount(content string) int {
	if content == "" {
		return 0
	}
	count := strings.Count(content, "\n")
	if !strings.HasSuffix(content, "\n") {
		count++
	}
	return count
}

func snapshotLanguage(filePath, defaultLanguage string) string {
	extension := strings.ToLower(strings.TrimPrefix(path.Ext(filePath), "."))
	switch extension {
	case "py":
		return "python"
	case "js", "mjs", "cjs", "jsx":
		return "javascript"
	case "ts", "tsx":
		return "typescript"
	case "java":
		return "java"
	case "c", "h", "cpp", "hpp", "cc":
		return "cpp"
	case "go":
		return "go"
	case "sql":
		return "sql"
	case "json":
		return "json"
	case "html", "htm":
		return "html"
	case "css":
		return "css"
	case "scss":
		return "scss"
	case "md":
		return "markdown"
	}
	if fallback := strings.TrimSpace(defaultLanguage); fallback != "" {
		return fallback
	}
	return "plaintext"
}
