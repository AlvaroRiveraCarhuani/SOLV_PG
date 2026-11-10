package integration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	"solv-backend/internal/core/domain"
	dockerinfra "solv-backend/internal/infrastructure/docker"
)

func phase3Seed(t *testing.T, versionID, snapshot string) domain.WorkspaceSeed {
	t.Helper()
	files, ok := domain.ParseProjectSnapshot(snapshot, "")
	if !ok || len(files) == 0 {
		t.Fatalf("snapshot de prueba inválido")
	}
	seed := domain.WorkspaceSeed{VersionID: versionID}
	for _, file := range files {
		seed.Files = append(seed.Files, domain.WorkspaceSeedFile{Path: file.Path, Content: file.Content})
	}
	sum := sha256.Sum256([]byte(snapshot))
	seed.ContentSHA256 = hex.EncodeToString(sum[:])
	return seed
}

func phase3NewVolume(t *testing.T) (*dockerinfra.Client, string) {
	t.Helper()
	requireDocker(t)
	client, err := dockerinfra.NewClient()
	if err != nil {
		t.Fatalf("docker client: %v", err)
	}
	volume := "solv_seed_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if err := client.EnsureVolumeExists(context.Background(), volume); err != nil {
		t.Fatalf("crear volumen de prueba: %v", err)
	}
	t.Cleanup(func() {
		if err := client.RemoveVolume(context.Background(), volume); err != nil {
			t.Logf("limpiar volumen %s: %v", volume, err)
		}
	})
	return client, volume
}

func phase3SnapshotFiles(t *testing.T, client *dockerinfra.Client, volume string) map[string]string {
	t.Helper()
	files, err := client.SnapshotPracticeEnvironmentVolume(context.Background(), volume, 4*1024*1024)
	if err != nil {
		t.Fatalf("leer volumen: %v", err)
	}
	result := make(map[string]string, len(files))
	for _, file := range files {
		result[file.Path] = string(file.Content)
	}
	return result
}

func TestWorkspaceSeedDocker_PreservesTreeAndRefusesForeignContent(t *testing.T) {
	client, volume := phase3NewVolume(t)
	ctx := context.Background()
	snapshot := "# === FILE: README.md ===\n# Proyecto V1\n\n# === FILE: src/main.py ===\nprint('v1')\n"
	seed := phase3Seed(t, "11111111-1111-4111-8111-111111111111", snapshot)

	if err := client.PrepareWorkspaceVolume(ctx, volume, seed); err != nil {
		t.Fatalf("sembrado inicial: %v", err)
	}
	files := phase3SnapshotFiles(t, client, volume)
	if files["README.md"] != "# Proyecto V1\n" || files["src/main.py"] != "print('v1')\n" {
		t.Fatalf("contenido o estructura inesperados: %+v", files)
	}
	if _, ok := files[".solv_workspace/init.json"]; !ok {
		t.Fatalf("falta el marcador de inicialización: %+v", files)
	}
	if err := client.VerifyWorkspaceVolume(ctx, volume, seed); err != nil {
		t.Fatalf("verificar marcador: %v", err)
	}

	// Reintentar la misma versión es idempotente.
	if err := client.PrepareWorkspaceVolume(ctx, volume, seed); err != nil {
		t.Fatalf("reintento idempotente: %v", err)
	}
	again := phase3SnapshotFiles(t, client, volume)
	if again["src/main.py"] != "print('v1')\n" {
		t.Fatalf("el reintento alteró el proyecto: %+v", again)
	}
	// El trabajo posterior del alumno no se reemplaza ni al reabrir ni al
	// publicar otra versión.
	if err := client.WritePracticeEnvironmentVolumeFiles(ctx, volume, map[string]string{
		"src/main.py": "print('trabajo del alumno')\n",
	}); err != nil {
		t.Fatalf("modificar trabajo del alumno: %v", err)
	}
	if err := client.VerifyWorkspaceVolume(ctx, volume, seed); err != nil {
		t.Fatalf("el marcador debía seguir válido tras la modificación: %v", err)
	}
	if err := client.PrepareWorkspaceVolume(ctx, volume, seed); err != nil {
		t.Fatalf("reabrir tras modificar no debía fallar: %v", err)
	}
	modified := phase3SnapshotFiles(t, client, volume)
	if modified["src/main.py"] != "print('trabajo del alumno')\n" {
		t.Fatalf("el trabajo del alumno se perdió: %+v", modified)
	}

	// Una versión nueva no reemplaza un volumen ya sembrado.
	snapshotV2 := "# === FILE: README.md ===\n# Proyecto V2\n"
	seedV2 := phase3Seed(t, "22222222-2222-4222-8222-222222222222", snapshotV2)
	if err := client.PrepareWorkspaceVolume(ctx, volume, seedV2); !errors.Is(err, domain.ErrWorkspaceSeedVolumeConflict) {
		t.Fatalf("V2 debía rechazarse, obtuvo %v", err)
	}
	afterV2 := phase3SnapshotFiles(t, client, volume)
	if afterV2["README.md"] != "# Proyecto V1\n" || afterV2["src/main.py"] != "print('trabajo del alumno')\n" {
		t.Fatalf("V2 modificó el volumen existente: %+v", afterV2)
	}
	if err := client.VerifyWorkspaceVolume(ctx, volume, seed); err != nil {
		t.Fatalf("el marcador V1 debía seguir siendo válido: %v", err)
	}

	// Un volumen con contenido ajeno no se toca.
	otherClient, otherVolume := phase3NewVolume(t)
	if err := otherClient.WritePracticeEnvironmentVolumeFiles(ctx, otherVolume, map[string]string{"trabajo.txt": "del alumno\n"}); err != nil {
		t.Fatalf("preparar contenido ajeno: %v", err)
	}
	if err := otherClient.PrepareWorkspaceVolume(ctx, otherVolume, seed); !errors.Is(err, domain.ErrWorkspaceSeedVolumeNotEmpty) {
		t.Fatalf("contenido ajeno debía rechazarse, obtuvo %v", err)
	}
	otherFiles := phase3SnapshotFiles(t, otherClient, otherVolume)
	if otherFiles["trabajo.txt"] != "del alumno\n" {
		t.Fatalf("el contenido ajeno cambió: %+v", otherFiles)
	}
	if err := otherClient.VerifyWorkspaceVolume(ctx, otherVolume, seed); !errors.Is(err, domain.ErrWorkspaceSeedVolumeAmbiguous) {
		t.Fatalf("un volumen sin marcador debía requerir revisión, obtuvo %v", err)
	}
	if err := otherClient.WritePracticeEnvironmentVolumeFiles(ctx, otherVolume, map[string]string{
		".solv_workspace/init.json": `{"version_id":`,
	}); err != nil {
		t.Fatalf("preparar marcador inválido: %v", err)
	}
	if err := otherClient.VerifyWorkspaceVolume(ctx, otherVolume, seed); !errors.Is(err, domain.ErrWorkspaceSeedVolumeAmbiguous) {
		t.Fatalf("un marcador inválido debía requerir revisión, obtuvo %v", err)
	}
}

func TestWorkspaceSeedDocker_ResumesOnlyItsPendingJournal(t *testing.T) {
	client, volume := phase3NewVolume(t)
	ctx := context.Background()
	snapshot := "# === FILE: a.txt ===\nA\n\n# === FILE: dir/b.txt ===\nB\n"
	seed := phase3Seed(t, "33333333-3333-4333-8333-333333333333", snapshot)

	// Simula una caída después de escribir el diario y un archivo parcial.
	pending := `{"version_id":"` + seed.VersionID + `","content_sha256":"` + seed.ContentSHA256 + `","file_count":2,"total_bytes":2,"state":"pending"}`
	if err := client.WritePracticeEnvironmentVolumeFiles(ctx, volume, map[string]string{
		".solv_workspace/pending.json": pending,
		"a.txt":                        "A\n",
	}); err != nil {
		t.Fatalf("preparar pendiente parcial: %v", err)
	}
	if err := client.PrepareWorkspaceVolume(ctx, volume, seed); err != nil {
		t.Fatalf("reanudar pendiente: %v", err)
	}
	files := phase3SnapshotFiles(t, client, volume)
	if files["a.txt"] != "A\n" || files["dir/b.txt"] != "B\n" {
		t.Fatalf("la reanudación no completó el árbol: %+v", files)
	}
	if _, ok := files[".solv_workspace/pending.json"]; ok {
		t.Fatalf("el diario pendiente debía retirarse: %+v", files)
	}

	// Un archivo diferente bajo el mismo diario no se sobrescribe.
	otherClient, otherVolume := phase3NewVolume(t)
	if err := otherClient.WritePracticeEnvironmentVolumeFiles(ctx, otherVolume, map[string]string{
		".solv_workspace/pending.json": pending,
		"a.txt":                        "contenido distinto\n",
	}); err != nil {
		t.Fatalf("preparar pendiente modificado: %v", err)
	}
	if err := otherClient.PrepareWorkspaceVolume(ctx, otherVolume, seed); !errors.Is(err, domain.ErrWorkspaceSeedVolumeAmbiguous) {
		t.Fatalf("un pendiente modificado debía quedar en cuarentena, obtuvo %v", err)
	}
	after := phase3SnapshotFiles(t, otherClient, otherVolume)
	if after["a.txt"] != "contenido distinto\n" {
		t.Fatalf("el archivo modificado se sobrescribió: %+v", after)
	}
}
