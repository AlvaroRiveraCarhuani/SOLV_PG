package services

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
)

type fakePreparationRunner struct {
	mu                sync.Mutex
	installs          int
	verifies          int
	installErr        error
	verifyErr         error
	transientFailures int
	blockUntilCancel  bool
	bytesToWrite      int
	lastInstallDir    string
}

func (f *fakePreparationRunner) RunInstall(
	ctx context.Context,
	_ domain.DependencyPreparationRequest,
	dir string,
	_ io.Writer,
) error {
	f.mu.Lock()
	f.installs++
	f.lastInstallDir = dir
	block, writeBytes, errInstall := f.blockUntilCancel, f.bytesToWrite, f.installErr
	transient := f.transientFailures > 0
	if transient {
		f.transientFailures--
	}
	f.mu.Unlock()

	if block {
		<-ctx.Done()
		return ctx.Err()
	}
	if transient {
		return fmt.Errorf("%w: corte de red simulado", domain.ErrDependencyPreparationTransient)
	}
	if errInstall != nil {
		return errInstall
	}
	if writeBytes > 0 {
		return os.WriteFile(filepath.Join(dir, "payload.bin"), make([]byte, writeBytes), 0o600)
	}
	return nil
}

func (f *fakePreparationRunner) RunVerify(
	_ context.Context,
	_ domain.DependencyPreparationRequest,
	_ string,
	_ io.Writer,
) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.verifies++
	return f.verifyErr
}

func (f *fakePreparationRunner) counts() (int, int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.installs, f.verifies
}

func newPreparationWorker(t *testing.T, runner domain.DependencyPreparationRunnerPort, limits domain.DependencyPreparationLimits) *DependencyPreparationWorker {
	t.Helper()
	worker, err := NewDependencyPreparationWorker(runner, filepath.Join(t.TempDir(), "deps"), limits)
	if err != nil {
		t.Fatalf("no se pudo crear el worker: %v", err)
	}
	return worker
}

func testPreparationRequest() domain.DependencyPreparationRequest {
	return domain.DependencyPreparationRequest{
		TenantID:       "00000000-0000-0000-0000-000000000001",
		VersionID:      "version-1",
		ComponentSlug:  "node-20",
		BaseImage:      "node:20-bookworm-slim",
		ImageDigest:    "sha256:aaaa",
		ProfileVersion: 1,
		ProjectPath:    "/home/workspace/proyecto",
		Install:        "npm ci",
		Verify:         "npm ls --depth=0",
		Network:        "build",
		LockfileHash:   "lock-1",
	}
}

func testPreparationLimits() domain.DependencyPreparationLimits {
	return domain.DependencyPreparationLimits{
		Timeout:      5 * time.Second,
		MaxBytes:     1 << 20,
		MinFreeBytes: 1,
	}
}

func TestDependencyPreparationWorkerPreparesAndReuses(t *testing.T) {
	runner := &fakePreparationRunner{bytesToWrite: 256}
	worker := newPreparationWorker(t, runner, testPreparationLimits())

	result, err := worker.Prepare(context.Background(), testPreparationRequest())
	if err != nil {
		t.Fatalf("se esperaba una preparación correcta: %v", err)
	}
	if result.Status != domain.DependencyPreparationPrepared || result.Reused {
		t.Fatalf("estado inesperado: %s (reutilizado=%v)", result.Status, result.Reused)
	}
	if result.Bytes < 256 {
		t.Fatalf("debe informar los bytes del árbol, obtuvo %d", result.Bytes)
	}
	if result.LogPath == "" {
		t.Fatal("toda preparación debe dejar un log")
	}
	if _, statErr := os.Stat(result.Dir); statErr != nil {
		t.Fatalf("el árbol preparado debe existir: %v", statErr)
	}
	if installs, verifies := runner.counts(); installs != 1 || verifies != 1 {
		t.Fatalf("debe instalar y comprobar una vez, obtuvo install=%d verify=%d", installs, verifies)
	}

	second, err := worker.Prepare(context.Background(), testPreparationRequest())
	if err != nil {
		t.Fatalf("la segunda preparación debe reutilizar: %v", err)
	}
	if !second.Reused {
		t.Fatal("la segunda preparación debe marcarse como reutilizada")
	}
	if installs, _ := runner.counts(); installs != 1 {
		t.Fatalf("no debe volver a instalar con la misma clave, obtuvo %d", installs)
	}
}

func TestDependencyPreparationWorkerRetriesTransientFailures(t *testing.T) {
	runner := &fakePreparationRunner{transientFailures: 1, bytesToWrite: 8}
	limits := testPreparationLimits()
	limits.RetryBackoff = time.Millisecond
	worker := newPreparationWorker(t, runner, limits)

	result, err := worker.Prepare(context.Background(), testPreparationRequest())
	if err != nil {
		t.Fatalf("el reintento debía resolver el fallo transitorio: %v", err)
	}
	if result.Status != domain.DependencyPreparationPrepared {
		t.Fatalf("el estado debe ser prepared, obtuvo %s", result.Status)
	}
	if result.Attempts != 2 {
		t.Fatalf("debe informar dos intentos, obtuvo %d", result.Attempts)
	}
	if installs, verifies := runner.counts(); installs != 2 || verifies != 1 {
		t.Fatalf("debe instalar dos veces y comprobar una, obtuvo install=%d verify=%d", installs, verifies)
	}
}

func TestDependencyPreparationWorkerDoesNotRetryDeterministicFailures(t *testing.T) {
	runner := &fakePreparationRunner{installErr: io.ErrUnexpectedEOF}
	limits := testPreparationLimits()
	limits.RetryBackoff = time.Millisecond
	worker := newPreparationWorker(t, runner, limits)

	result, err := worker.Prepare(context.Background(), testPreparationRequest())
	if err == nil {
		t.Fatal("un comando mal escrito debe fallar")
	}
	if result.Attempts != 1 {
		t.Fatalf("un fallo determinista no se reintenta, obtuvo %d intentos", result.Attempts)
	}
	if installs, _ := runner.counts(); installs != 1 {
		t.Fatalf("debe instalar una sola vez, obtuvo %d", installs)
	}
}

func TestDependencyPreparationWorkerStopsAfterMaxAttempts(t *testing.T) {
	runner := &fakePreparationRunner{transientFailures: 99}
	limits := testPreparationLimits()
	limits.MaxAttempts = 3
	limits.RetryBackoff = time.Millisecond
	worker := newPreparationWorker(t, runner, limits)

	result, err := worker.Prepare(context.Background(), testPreparationRequest())
	if err == nil {
		t.Fatal("con fallos transitorios sostenidos debe fallar")
	}
	if result.Attempts != 3 {
		t.Fatalf("debe agotar los tres intentos, obtuvo %d", result.Attempts)
	}
	if installs, _ := runner.counts(); installs != 3 {
		t.Fatalf("debe instalar tres veces, obtuvo %d", installs)
	}
	if !strings.Contains(err.Error(), "3 intentos") {
		t.Fatalf("el error debe nombrar los intentos, obtuvo: %v", err)
	}
}

func TestDependencyPreparationWorkerDoesNotRetryFailedVerify(t *testing.T) {
	runner := &fakePreparationRunner{verifyErr: io.ErrClosedPipe}
	limits := testPreparationLimits()
	limits.RetryBackoff = time.Millisecond
	worker := newPreparationWorker(t, runner, limits)

	result, err := worker.Prepare(context.Background(), testPreparationRequest())
	if err == nil {
		t.Fatal("una comprobación fallida debe fallar")
	}
	if installs, verifies := runner.counts(); installs != 1 || verifies != 1 {
		t.Fatalf("no debe repetir la instalación por un fallo de comprobación, obtuvo install=%d verify=%d", installs, verifies)
	}
	if result.Status != domain.DependencyPreparationFailed {
		t.Fatalf("el estado debe ser failed, obtuvo %s", result.Status)
	}
}

func TestDependencyPreparationWorkerFailsOnInstall(t *testing.T) {
	runner := &fakePreparationRunner{installErr: io.ErrUnexpectedEOF}
	worker := newPreparationWorker(t, runner, testPreparationLimits())

	result, err := worker.Prepare(context.Background(), testPreparationRequest())
	if err == nil {
		t.Fatal("la instalación fallida debe devolver error")
	}
	if result.Status != domain.DependencyPreparationFailed {
		t.Fatalf("el estado debe ser failed, obtuvo %s", result.Status)
	}
	if _, statErr := os.Stat(result.Dir); !os.IsNotExist(statErr) {
		t.Fatal("un árbol fallido no debe publicarse")
	}
	if result.LogPath == "" {
		t.Fatal("el fallo debe dejar log para el docente")
	}
	if installs, verifies := runner.counts(); installs != 1 || verifies != 0 {
		t.Fatalf("no debe comprobar tras un fallo de instalación, obtuvo install=%d verify=%d", installs, verifies)
	}
}

func TestDependencyPreparationWorkerFailsOnOfflineVerify(t *testing.T) {
	runner := &fakePreparationRunner{verifyErr: io.ErrClosedPipe}
	worker := newPreparationWorker(t, runner, testPreparationLimits())

	result, err := worker.Prepare(context.Background(), testPreparationRequest())
	if err == nil {
		t.Fatal("una comprobación sin red fallida debe devolver error")
	}
	if result.Status != domain.DependencyPreparationFailed {
		t.Fatalf("el estado debe ser failed, obtuvo %s", result.Status)
	}
	if _, statErr := os.Stat(result.Dir); !os.IsNotExist(statErr) {
		t.Fatal("un árbol que no comprueba sin red no debe publicarse")
	}
	if !strings.Contains(err.Error(), "sin red") {
		t.Fatalf("el error debe nombrar la comprobación sin red, obtuvo: %v", err)
	}
}

func TestDependencyPreparationWorkerEnforcesByteCeiling(t *testing.T) {
	runner := &fakePreparationRunner{bytesToWrite: 4096}
	limits := testPreparationLimits()
	limits.MaxBytes = 1024
	worker := newPreparationWorker(t, runner, limits)

	result, err := worker.Prepare(context.Background(), testPreparationRequest())
	if err == nil {
		t.Fatal("un árbol por encima del techo debe fallar")
	}
	if !strings.Contains(err.Error(), "techo") {
		t.Fatalf("el error debe nombrar el techo de bytes, obtuvo: %v", err)
	}
	if _, statErr := os.Stat(result.Dir); !os.IsNotExist(statErr) {
		t.Fatal("un árbol por encima del techo no debe publicarse")
	}
}

func TestDependencyPreparationWorkerEnforcesTimeout(t *testing.T) {
	runner := &fakePreparationRunner{blockUntilCancel: true}
	limits := testPreparationLimits()
	limits.Timeout = 40 * time.Millisecond
	worker := newPreparationWorker(t, runner, limits)

	started := time.Now()
	result, err := worker.Prepare(context.Background(), testPreparationRequest())
	if err == nil {
		t.Fatal("una instalación colgada debe cortarse por tiempo")
	}
	if result.Status != domain.DependencyPreparationFailed {
		t.Fatalf("el estado debe ser failed, obtuvo %s", result.Status)
	}
	if elapsed := time.Since(started); elapsed > 2*time.Second {
		t.Fatalf("el techo de tiempo no se respetó: %s", elapsed)
	}
}

func TestDependencyPreparationWorkerPrunesByReference(t *testing.T) {
	runner := &fakePreparationRunner{bytesToWrite: 8}
	worker := newPreparationWorker(t, runner, testPreparationLimits())

	first, err := worker.Prepare(context.Background(), testPreparationRequest())
	if err != nil {
		t.Fatalf("primera preparación: %v", err)
	}
	other := testPreparationRequest()
	other.VersionID = "version-2"
	second, err := worker.Prepare(context.Background(), other)
	if err != nil {
		t.Fatalf("segunda preparación: %v", err)
	}
	if first.Key == second.Key {
		t.Fatal("versiones distintas deben tener claves distintas")
	}

	if _, err := worker.PruneOrphanPreparedTrees(context.Background(), nil); err == nil {
		t.Fatal("la poda con conjunto nulo debe rechazarse")
	}

	pruned, err := worker.PruneOrphanPreparedTrees(context.Background(), map[string]bool{first.Key: true})
	if err != nil {
		t.Fatalf("la poda falló: %v", err)
	}
	if len(pruned) != 1 || pruned[0] != second.Key {
		t.Fatalf("debe podar solo la clave no referenciada, obtuvo %v", pruned)
	}
	if _, statErr := os.Stat(first.Dir); statErr != nil {
		t.Fatalf("la clave activa debe conservarse: %v", statErr)
	}
	if _, statErr := os.Stat(second.Dir); !os.IsNotExist(statErr) {
		t.Fatal("la clave no referenciada debe borrarse")
	}

	count, bytes, err := worker.Usage()
	if err != nil {
		t.Fatalf("Usage falló: %v", err)
	}
	if count != 1 || bytes <= 0 {
		t.Fatalf("el uso debe informar un árbol con bytes, obtuvo count=%d bytes=%d", count, bytes)
	}
}

func TestDependencyPreparationWorkerRejectsUnsafeRoot(t *testing.T) {
	runner := &fakePreparationRunner{}
	if _, err := NewDependencyPreparationWorker(runner, "deps", testPreparationLimits()); err == nil {
		t.Fatal("una raíz relativa debe rechazarse")
	}
	if _, err := NewDependencyPreparationWorker(runner, "/", testPreparationLimits()); err == nil {
		t.Fatal("la raíz del sistema debe rechazarse")
	}
	if _, err := NewDependencyPreparationWorker(nil, filepath.Join(t.TempDir(), "deps"), testPreparationLimits()); err == nil {
		t.Fatal("sin runner no hay worker")
	}
}

func TestCheckSameFilesystem(t *testing.T) {
	root := t.TempDir()
	if err := CheckSameFilesystem(root, root); err != nil {
		t.Fatalf("las mismas rutas deben compartir sistema de archivos: %v", err)
	}
	if err := CheckSameFilesystem(root, filepath.Join(root, "inexistente")); err == nil {
		t.Fatal("una ruta inexistente debe devolver error")
	}
}
