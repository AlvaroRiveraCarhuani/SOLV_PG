package services

import (
	"errors"
	"testing"

	"solv-backend/internal/core/domain"
)

func TestWorkspaceSeedFromVersionRejectsInvalidHashAndSnapshot(t *testing.T) {
	version := &domain.WorkspaceProjectVersion{
		ID:            "version-v1",
		TextSnapshot:  "# === FILE: main.py ===\nprint(1)\n",
		ContentSHA256: "hash-que-no-corresponde",
	}
	if _, err := workspaceSeedFromVersion(version); !errors.Is(err, domain.ErrWorkspaceSeedVersionInvalid) {
		t.Fatalf("un hash distinto debía rechazarse, obtuvo %v", err)
	}
	if _, err := workspaceSeedFromVersion(&domain.WorkspaceProjectVersion{ID: "vacio"}); !errors.Is(err, domain.ErrWorkspaceSeedVersionInvalid) {
		t.Fatalf("una versión vacía debía rechazarse, obtuvo %v", err)
	}
}
