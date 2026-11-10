package services

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"solv-backend/internal/core/domain"
)

const workspaceImportExerciseID = "b3f7c9a2-7a1f-4d01-8e5f-1a2b3c4d5e6f"

func TestWorkspaceImportFolderPreservesNestedPathsAsDraft(t *testing.T) {
	repo := &workspaceProjectRepoFake{}
	service := NewWorkspaceProjectService(repo, true)

	result, err := service.ImportFolder(context.Background(), "tenant-1", workspaceImportExerciseID, "user-1", 0, []WorkspaceProjectImportFile{
		{Path: "proyecto-go/go.mod", Content: []byte("module ejemplo\n\ngo 1.26\n")},
		{Path: "proyecto-go/main.go", Content: []byte("package main\n\nfunc main() {}\n")},
		{Path: "proyecto-go/internal/calculator/calculator.go", Content: []byte("package calculator\n")},
		{Path: "proyecto-go/tests/calculator_test.go", Content: []byte("package calculator\n")},
	})
	if err != nil {
		t.Fatalf("ImportFolder: %v", err)
	}
	if result.FileCount != 4 || result.TotalBytes == 0 {
		t.Fatalf("resumen inesperado: %+v", result)
	}
	if result.Draft == nil || repo.saveCalls != 1 || repo.publishCalls != 0 {
		t.Fatalf("importación inesperada: draft=%+v save=%d publish=%d", result.Draft, repo.saveCalls, repo.publishCalls)
	}
	for _, expected := range []string{
		"# === FILE: proyecto-go/go.mod ===",
		"# === FILE: proyecto-go/internal/calculator/calculator.go ===",
		"# === FILE: proyecto-go/tests/calculator_test.go ===",
	} {
		if !strings.Contains(result.Draft.TextSnapshot, expected) {
			t.Errorf("snapshot sin %q:\n%s", expected, result.Draft.TextSnapshot)
		}
	}
}

func TestWorkspaceImportFolderRejectsInvalidEntries(t *testing.T) {
	valid := []WorkspaceProjectImportFile{{Path: "main.go", Content: []byte("package main\n")}}
	cases := []struct {
		name  string
		files []WorkspaceProjectImportFile
	}{
		{"traversal", []WorkspaceProjectImportFile{{Path: "../secret.txt", Content: []byte("x\n")}}},
		{"absolute", []WorkspaceProjectImportFile{{Path: "/etc/passwd", Content: []byte("x\n")}}},
		{"backslash", []WorkspaceProjectImportFile{{Path: `src\\main.go`, Content: []byte("x\n")}}},
		{"duplicate", []WorkspaceProjectImportFile{{Path: "main.go", Content: []byte("a\n")}, {Path: "main.go", Content: []byte("b\n")}}},
		{"binary", []WorkspaceProjectImportFile{{Path: "main.go", Content: []byte{0x00, 0x01}}}},
		{"not utf8", []WorkspaceProjectImportFile{{Path: "main.go", Content: []byte{0xff, 0xfe}}}},
		{"snapshot header", []WorkspaceProjectImportFile{{Path: "main.go", Content: []byte("# === FILE: other.go ===\nx\n")}}},
		{"too many files", manyWorkspaceImportFiles(maxWorkspaceProjectFiles + 1)},
		{"file too large", []WorkspaceProjectImportFile{{Path: "main.go", Content: bytes.Repeat([]byte("x"), maxWorkspaceProjectFileBytes+1)}}},
		{"project too large", []WorkspaceProjectImportFile{
			{Path: "a.txt", Content: bytes.Repeat([]byte("a"), maxWorkspaceProjectFileBytes)},
			{Path: "b.txt", Content: bytes.Repeat([]byte("b"), maxWorkspaceProjectFileBytes)},
			{Path: "c.txt", Content: bytes.Repeat([]byte("c"), maxWorkspaceProjectFileBytes)},
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &workspaceProjectRepoFake{}
			service := NewWorkspaceProjectService(repo, false)
			if _, err := service.ImportFolder(context.Background(), "tenant-1", workspaceImportExerciseID, "user-1", 0, tc.files); err == nil {
				t.Fatal("ImportFolder aceptó una entrada inválida")
			}
			if repo.saveCalls != 0 {
				t.Fatalf("una importación inválida tocó el borrador: %d", repo.saveCalls)
			}
		})
	}

	if _, err := NewWorkspaceProjectService(&workspaceProjectRepoFake{}, false).ImportFolder(context.Background(), "tenant-1", workspaceImportExerciseID, "user-1", 0, valid); err != nil {
		t.Fatalf("la entrada de control debe ser válida: %v", err)
	}
}

func TestWorkspaceImportZipAcceptsNestedFiles(t *testing.T) {
	repo := &workspaceProjectRepoFake{}
	service := NewWorkspaceProjectService(repo, false)
	archive := buildWorkspaceZip(t, []workspaceZipEntry{
		{Name: "app/main.py", Body: "print('hola')\n"},
		{Name: "app/lib/util.py", Body: "def suma(a, b):\n    return a + b\n"},
		{Name: "app/", Mode: os.ModeDir},
	})

	result, err := service.ImportZip(context.Background(), "tenant-1", workspaceImportExerciseID, "user-1", 0, archive)
	if err != nil {
		t.Fatalf("ImportZip: %v", err)
	}
	if result.FileCount != 2 || !strings.Contains(result.Draft.TextSnapshot, "# === FILE: app/lib/util.py ===") {
		t.Fatalf("ZIP inesperado: %+v\n%s", result, result.Draft.TextSnapshot)
	}
}

func TestWorkspaceImportZipRejectsUnsafeArchives(t *testing.T) {
	cases := []struct {
		name    string
		entries []workspaceZipEntry
		raw     []byte
	}{
		{name: "zip slip", entries: []workspaceZipEntry{{Name: "../evil.txt", Body: "x\n"}}},
		{name: "absolute", entries: []workspaceZipEntry{{Name: "/etc/passwd", Body: "x\n"}}},
		{name: "symlink", entries: []workspaceZipEntry{{Name: "link", Body: "target", Mode: os.ModeSymlink}}},
		{name: "duplicate", entries: []workspaceZipEntry{{Name: "main.py", Body: "a\n"}, {Name: "main.py", Body: "b\n"}}},
		{name: "binary", entries: []workspaceZipEntry{{Name: "main.py", Body: string([]byte{0, 1, 2})}}},
		{name: "entry too large", entries: []workspaceZipEntry{{Name: "big.txt", Body: strings.Repeat("x", maxWorkspaceImportZipEntry+1)}}},
		{name: "corrupt", raw: []byte("not a zip")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &workspaceProjectRepoFake{}
			service := NewWorkspaceProjectService(repo, false)
			archive := tc.raw
			if archive == nil {
				archive = buildWorkspaceZip(t, tc.entries)
			}
			if _, err := service.ImportZip(context.Background(), "tenant-1", workspaceImportExerciseID, "user-1", 0, archive); err == nil {
				t.Fatal("ImportZip aceptó un archivo inseguro")
			}
			if repo.saveCalls != 0 {
				t.Fatalf("una importación inválida tocó el borrador: %d", repo.saveCalls)
			}
		})
	}
}

func TestWorkspaceImportPreservesRevisionConflict(t *testing.T) {
	repo := &workspaceProjectRepoFake{project: &domain.WorkspaceProject{ID: "project-1", ExerciseID: workspaceImportExerciseID}, saveErr: domain.ErrWorkspaceDraftRevisionConflict}
	service := NewWorkspaceProjectService(repo, false)
	_, err := service.ImportFolder(context.Background(), "tenant-1", workspaceImportExerciseID, "user-1", 7, []WorkspaceProjectImportFile{{Path: "main.py", Content: []byte("print(1)\n")}})
	if !errors.Is(err, domain.ErrWorkspaceDraftRevisionConflict) {
		t.Fatalf("conflicto no propagado: %v", err)
	}
}

func TestWorkspaceImportDoesNotPublishOrChangeVersions(t *testing.T) {
	repo := &workspaceProjectRepoFake{
		project:  &domain.WorkspaceProject{ID: "project-1", ExerciseID: workspaceImportExerciseID},
		versions: []*domain.WorkspaceProjectVersion{{ID: "version-1", VersionNumber: 1}},
	}
	service := NewWorkspaceProjectService(repo, true)
	_, err := service.ImportFolder(context.Background(), "tenant-1", workspaceImportExerciseID, "user-1", 0, []WorkspaceProjectImportFile{{Path: "main.py", Content: []byte("print(1)\n")}})
	if err != nil {
		t.Fatalf("ImportFolder: %v", err)
	}
	if repo.publishCalls != 0 || len(repo.versions) != 1 || repo.versions[0].ID != "version-1" {
		t.Fatalf("la importación publicó o alteró versiones: %+v", repo)
	}
}

func manyWorkspaceImportFiles(count int) []WorkspaceProjectImportFile {
	files := make([]WorkspaceProjectImportFile, 0, count)
	for i := 0; i < count; i++ {
		files = append(files, WorkspaceProjectImportFile{Path: fmt.Sprintf("f%03d.txt", i), Content: []byte("x\n")})
	}
	return files
}

type workspaceZipEntry struct {
	Name string
	Body string
	Mode os.FileMode
}

func buildWorkspaceZip(t *testing.T, entries []workspaceZipEntry) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for _, entry := range entries {
		header := &zip.FileHeader{Name: entry.Name, Method: zip.Deflate}
		header.SetMode(entry.Mode)
		file, err := writer.CreateHeader(header)
		if err != nil {
			t.Fatalf("crear entrada ZIP %s: %v", entry.Name, err)
		}
		if _, err := file.Write([]byte(entry.Body)); err != nil {
			t.Fatalf("escribir entrada ZIP %s: %v", entry.Name, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("cerrar ZIP: %v", err)
	}
	return buffer.Bytes()
}
