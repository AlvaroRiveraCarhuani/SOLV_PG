package seaweedfs

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"solv-backend/internal/core/domain"
)

// defaultSnapshotMaxFiles acota los archivos que se aceptan al leer un snapshot
// si el llamador no fija un tope.
const defaultSnapshotMaxFiles = 1024

// ArchiveOptions viene de configuración del backend (ADR-058 §5 y §6).
type ArchiveOptions struct {
	// ExcludePrefixes son rutas relativas a la raíz del ejercicio (p. ej. ".vscode/").
	// Un elemento que termina en "*" excluye por prefijo de nombre (".solv_*").
	ExcludePrefixes []string
	// MaxFiles es el tope de archivos incluidos en el snapshot.
	MaxFiles int
}

// PackageExercise escribe en w un tar.gz del directorio root, aplicando las
// exclusiones y el tope de archivos. Devuelve la cantidad de archivos incluidos.
// Los enlaces simbólicos se omiten para no sacar archivos fuera del ejercicio.
func PackageExercise(root string, opts ArchiveOptions, w io.Writer) (int, error) {
	gz := gzip.NewWriter(w)
	tw := tar.NewWriter(gz)
	count := 0

	walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if excluded(rel, d.Name(), opts.ExcludePrefixes) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() || !d.Type().IsRegular() {
			return nil
		}
		if count >= opts.MaxFiles {
			return fmt.Errorf("seaweedfs: el ejercicio supera %d archivos", opts.MaxFiles)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		hdr.Name = rel
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		src, err := os.Open(path)
		if err != nil {
			return err
		}
		defer src.Close()
		if _, err := io.Copy(tw, src); err != nil {
			return err
		}
		count++
		return nil
	})
	if walkErr != nil {
		return 0, walkErr
	}
	if err := tw.Close(); err != nil {
		return 0, err
	}
	if err := gz.Close(); err != nil {
		return 0, err
	}
	return count, nil
}

// PackageFiles escribe un tar.gz con archivos ya leídos del entorno del
// docente. Aplica las mismas exclusiones y el mismo tope que PackageExercise, y
// ordena las rutas para que el mismo contenido produzca el mismo objeto.
func PackageFiles(files map[string]string, opts ArchiveOptions, w io.Writer) (int, error) {
	type entry struct {
		path    string
		content string
	}
	entries := make([]entry, 0, len(files))
	for rawPath, content := range files {
		rel := strings.TrimPrefix(strings.TrimSpace(strings.ReplaceAll(rawPath, "\\", "/")), "./")
		if rel == "" {
			continue
		}
		entries = append(entries, entry{path: rel, content: content})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].path < entries[j].path })

	maxFiles := opts.MaxFiles
	if maxFiles <= 0 {
		maxFiles = defaultSnapshotMaxFiles
	}

	gz := gzip.NewWriter(w)
	tw := tar.NewWriter(gz)
	count := 0
	for _, item := range entries {
		if excluded(item.path, path.Base(item.path), opts.ExcludePrefixes) {
			continue
		}
		if count >= maxFiles {
			return 0, fmt.Errorf("seaweedfs: el ejercicio supera %d archivos", maxFiles)
		}
		content := []byte(item.content)
		hdr := &tar.Header{
			Typeflag: tar.TypeReg,
			Name:     item.path,
			Mode:     0o644,
			Size:     int64(len(content)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return 0, fmt.Errorf("seaweedfs: encabezado de %q: %w", item.path, err)
		}
		if _, err := tw.Write(content); err != nil {
			return 0, fmt.Errorf("seaweedfs: contenido de %q: %w", item.path, err)
		}
		count++
	}
	if err := tw.Close(); err != nil {
		return 0, err
	}
	if err := gz.Close(); err != nil {
		return 0, err
	}
	return count, nil
}

// UnpackSnapshot lee el tar.gz emitido y devuelve sus archivos regulares. Las
// rutas inseguras (absolutas o con ".."), los enlaces y los archivos internos de
// la plataforma se descartan: al alumno solo viaja el código del ejercicio.
func UnpackSnapshot(data []byte, maxFiles int) ([]domain.WorkspaceVolumeFile, error) {
	if maxFiles <= 0 {
		maxFiles = defaultSnapshotMaxFiles
	}
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("seaweedfs: abrir snapshot: %w", err)
	}
	defer func() { _ = gz.Close() }()

	reader := tar.NewReader(gz)
	files := make([]domain.WorkspaceVolumeFile, 0, 16)
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("seaweedfs: leer snapshot: %w", err)
		}
		if header == nil || header.Typeflag != tar.TypeReg {
			continue
		}
		rel, ok := safeSnapshotPath(header.Name)
		if !ok || isExcludedSnapshotPath(rel) {
			continue
		}
		if len(files) >= maxFiles {
			return nil, fmt.Errorf("seaweedfs: el snapshot supera %d archivos", maxFiles)
		}
		content, err := io.ReadAll(io.LimitReader(reader, header.Size))
		if err != nil {
			return nil, fmt.Errorf("seaweedfs: leer %q del snapshot: %w", rel, err)
		}
		files = append(files, domain.WorkspaceVolumeFile{Path: rel, Content: content})
	}
	return files, nil
}

// safeSnapshotPath normaliza una ruta del tar y rechaza las que escapan del
// volumen del alumno.
func safeSnapshotPath(name string) (string, bool) {
	cleaned := path.Clean(strings.TrimPrefix(strings.TrimSpace(strings.ReplaceAll(name, "\\", "/")), "./"))
	if cleaned == "." || cleaned == "" || cleaned == ".." || strings.HasPrefix(cleaned, "../") || strings.HasPrefix(cleaned, "/") {
		return "", false
	}
	return cleaned, true
}

// isExcludedSnapshotPath descarta los segmentos que nunca deben tocar el entorno
// del alumno, aunque el snapshot los traiga dentro.
func isExcludedSnapshotPath(rel string) bool {
	for _, segment := range strings.Split(rel, "/") {
		switch segment {
		case ".git", ".vscode", "node_modules":
			return true
		}
		if strings.HasPrefix(segment, ".solv_") {
			return true
		}
	}
	return false
}

// excluded decide si una entrada queda fuera del snapshot. Un patrón que
// termina en "*" excluye cualquier segmento de la ruta con ese prefijo, así que
// ".solv_*" cubre también ".solv_backups/<avance>/x.py".
func excluded(rel, name string, prefixes []string) bool {
	segments := strings.Split(rel, "/")
	for _, p := range prefixes {
		if strings.HasSuffix(p, "*") {
			prefix := strings.TrimSuffix(p, "*")
			if strings.HasPrefix(name, prefix) {
				return true
			}
			for _, segment := range segments {
				if strings.HasPrefix(segment, prefix) {
					return true
				}
			}
			continue
		}
		p = strings.TrimSuffix(p, "/")
		if rel == p || strings.HasPrefix(rel, p+"/") {
			return true
		}
	}
	return false
}
