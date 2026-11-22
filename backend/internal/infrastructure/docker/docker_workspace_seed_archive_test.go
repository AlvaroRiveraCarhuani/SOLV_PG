package docker

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"

	"solv-backend/internal/core/domain"
)

type workspaceArchiveTestEntry struct {
	name     string
	body     string
	typeflag byte
}

func buildWorkspaceSeedArchive(t *testing.T, entries []workspaceArchiveTestEntry) []byte {
	t.Helper()
	var buffer bytes.Buffer
	gz := gzip.NewWriter(&buffer)
	writer := tar.NewWriter(gz)
	for _, entry := range entries {
		typeflag := entry.typeflag
		if typeflag == 0 {
			typeflag = tar.TypeReg
		}
		header := &tar.Header{Name: entry.name, Mode: 0o644, Size: int64(len(entry.body)), Typeflag: typeflag}
		if typeflag == tar.TypeSymlink || typeflag == tar.TypeLink {
			header.Linkname = "otro"
			header.Size = 0
		}
		if err := writer.WriteHeader(header); err != nil {
			t.Fatalf("write header: %v", err)
		}
		if header.Size > 0 {
			if _, err := writer.Write([]byte(entry.body)); err != nil {
				t.Fatalf("write body: %v", err)
			}
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close tar: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("close gzip: %v", err)
	}
	return buffer.Bytes()
}

func workspaceSeedWithArchive(archive []byte, fileCount int, totalBytes int64) domain.WorkspaceSeed {
	sum := sha256.Sum256(archive)
	hash := hex.EncodeToString(sum[:])
	return domain.WorkspaceSeed{
		VersionID:         "version-1",
		ContentSHA256:     hash,
		Archive:           archive,
		ArchiveSHA256:     hash,
		ArchiveFileCount:  fileCount,
		ArchiveTotalBytes: totalBytes,
	}
}

func TestValidateWorkspaceSeedExtractsArchive(t *testing.T) {
	archive := buildWorkspaceSeedArchive(t, []workspaceArchiveTestEntry{
		{name: "proyecto/main.go", body: "package main\n"},
		{name: "proyecto/internal/app.go", body: "package internal\n"},
		{name: "proyecto/", body: "", typeflag: tar.TypeDir},
	})
	files, total, err := validateWorkspaceSeed(workspaceSeedWithArchive(archive, 2, int64(len("package main\npackage internal\n"))))
	if err != nil {
		t.Fatalf("validateWorkspaceSeed: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("archivos extraídos: %d, esperados 2", len(files))
	}
	if files["proyecto/main.go"] != "package main\n" || files["proyecto/internal/app.go"] != "package internal\n" {
		t.Fatalf("contenido extraído inesperado: %#v", files)
	}
	if total != len("package main\npackage internal\n") {
		t.Fatalf("total extraído %d", total)
	}
}

func TestValidateWorkspaceSeedRejectsUnsafeArchive(t *testing.T) {
	valid := buildWorkspaceSeedArchive(t, []workspaceArchiveTestEntry{{name: "main.go", body: "x\n"}})

	cases := []struct {
		name  string
		seed  domain.WorkspaceSeed
		check error
	}{
		{"huella distinta", func() domain.WorkspaceSeed {
			seed := workspaceSeedWithArchive(valid, 1, 2)
			seed.ArchiveSHA256 = "0000000000000000000000000000000000000000000000000000000000000000"
			return seed
		}(), domain.ErrWorkspaceSeedVersionInvalid},
		{"traversal", workspaceSeedWithArchive(buildWorkspaceSeedArchive(t, []workspaceArchiveTestEntry{{name: "../fuera.txt", body: "x\n"}}), 1, 2), domain.ErrWorkspaceSeedVersionInvalid},
		{"absoluta", workspaceSeedWithArchive(buildWorkspaceSeedArchive(t, []workspaceArchiveTestEntry{{name: "/etc/passwd", body: "x\n"}}), 1, 2), domain.ErrWorkspaceSeedVersionInvalid},
		{"sin normalizar", workspaceSeedWithArchive(buildWorkspaceSeedArchive(t, []workspaceArchiveTestEntry{{name: "a//b.txt", body: "x\n"}}), 1, 2), domain.ErrWorkspaceSeedVersionInvalid},
		{"symlink", workspaceSeedWithArchive(buildWorkspaceSeedArchive(t, []workspaceArchiveTestEntry{{name: "enlace", body: "", typeflag: tar.TypeSymlink}}), 1, 0), domain.ErrWorkspaceSeedVersionInvalid},
		{"metadatos", workspaceSeedWithArchive(buildWorkspaceSeedArchive(t, []workspaceArchiveTestEntry{{name: ".solv_workspace/init.json", body: "{}"}}), 1, 2), domain.ErrWorkspaceSeedVersionInvalid},
		{"duplicada", workspaceSeedWithArchive(buildWorkspaceSeedArchive(t, []workspaceArchiveTestEntry{{name: "a.txt", body: "x\n"}, {name: "a.txt", body: "y\n"}}), 2, 4), domain.ErrWorkspaceSeedVersionInvalid},
		{"conteo distinto", workspaceSeedWithArchive(valid, 5, 2), domain.ErrWorkspaceSeedVersionInvalid},
		{"tamaño distinto", workspaceSeedWithArchive(valid, 1, 99), domain.ErrWorkspaceSeedVersionInvalid},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, err := validateWorkspaceSeed(tc.seed); !errors.Is(err, tc.check) {
				t.Fatalf("se esperaba %v, llegó %v", tc.check, err)
			}
		})
	}
}
