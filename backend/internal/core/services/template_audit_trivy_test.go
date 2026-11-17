package services

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestTrivyScanArgsSkipsTheDBUpdate comprueba que el escaneo no vuelva a bajar
// la base CVE. El worker la sincroniza antes de mirar la cola, así que un
// refresco dentro del escaneo solo agrega 133 MB de red y 1,7 GB de escritura
// por plantilla auditada, con el mismo resultado.
func TestTrivyScanArgsSkipsTheDBUpdate(t *testing.T) {
	const image = "solv-local/stack:136f61d22016"
	args := trivyScanArgs(image)

	if !hasArg(args, "--skip-db-update") {
		t.Fatalf("el escaneo debía incluir --skip-db-update, obtuve %v", args)
	}
	// Sin red, dejar que Trivy intente actualizar el índice Java aborta el
	// escaneo con "network is unreachable" y la plantilla se reintenta sin fin.
	if !hasArg(args, "--skip-java-db-update") {
		t.Fatalf("el escaneo debía incluir --skip-java-db-update, obtuve %v", args)
	}
	if hasArg(args, "--download-db-only") {
		t.Fatalf("el escaneo no debe descargar la base, obtuve %v", args)
	}
	if last := args[len(args)-1]; last != image {
		t.Fatalf("la imagen debía ir al final, obtuve %q en %v", last, args)
	}
	if hasArg(args, "LOW") || hasArg(args, "MEDIUM") {
		t.Fatalf("el umbral de severidad debía seguir en CRITICAL,HIGH, obtuve %v", args)
	}
	if !hasArg(args, "CRITICAL,HIGH") {
		t.Fatalf("faltó el umbral de severidad, obtuve %v", args)
	}
}

// TestTrivyResourcesStayBelowOneCore evita que la descarga o el escaneo vuelvan
// a pedir un núcleo entero: medio núcleo calienta la mitad y alcanza para el
// trabajo, que es auxiliar al host.
func TestTrivyResourcesStayBelowOneCore(t *testing.T) {
	resources := trivyResources()
	if resources.NanoCPUs <= 0 || resources.NanoCPUs > 500_000_000 {
		t.Fatalf("el techo de CPU debía ser medio núcleo como máximo, obtuve %d", resources.NanoCPUs)
	}
	if want := int64(1 << 30); resources.Memory != want {
		t.Fatalf("el techo de memoria debía seguir en 1 GiB, obtuve %d", resources.Memory)
	}
}

// TestAuditCadenceStaysGentle protege las dos esperas que generaban ruido
// constante: el reintento de un minuto sin base CVE y el reconciliador de 10 s.
func TestAuditCadenceStaysGentle(t *testing.T) {
	if trivyDBRetryBase < 5*time.Minute {
		t.Fatalf("el reintento sin base CVE no debe bajar de 5 minutos, obtuve %s", trivyDBRetryBase)
	}
	if auditReconcileInterval < 30*time.Second {
		t.Fatalf("la cola de auditoría no debe revisarse más seguido que cada 30 s, obtuve %s", auditReconcileInterval)
	}
	if trivyDBRefresh != 24*time.Hour {
		t.Fatalf("la sincronización diaria de la base CVE no cambia, obtuve %s", trivyDBRefresh)
	}
}

// TestJavaDBIsFreshSpacesTheDownload evita que un índice de casi 1 GB se baje
// en cada arranque o cada día: solo se refresca cuando pasó una semana.
func TestJavaDBIsFreshSpacesTheDownload(t *testing.T) {
	dir := t.TempDir()
	jdbc := filepath.Join(dir, "trivy", "java-db")
	if err := os.MkdirAll(jdbc, 0o755); err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 11, 17, 12, 0, 0, 0, time.UTC)
	worker := &TemplateAuditWorker{cacheDir: dir, now: func() time.Time { return now }}

	if worker.javaDBIsFresh() {
		t.Fatal("sin metadata no hay índice vigente")
	}

	writeMeta := func(updated time.Time) {
		t.Helper()
		body := []byte(`{"Version":1,"UpdatedAt":"` + updated.Format(time.RFC3339) + `"}`)
		if err := os.WriteFile(filepath.Join(jdbc, "metadata.json"), body, 0o600); err != nil {
			t.Fatal(err)
		}
	}

	writeMeta(now.Add(-48 * time.Hour))
	if !worker.javaDBIsFresh() {
		t.Fatal("un índice de hace dos días debe considerarse vigente")
	}

	writeMeta(now.Add(-trivyJavaDBMaxAge - time.Hour))
	if worker.javaDBIsFresh() {
		t.Fatal("un índice de más de una semana debe refrescarse")
	}

	if err := os.WriteFile(filepath.Join(jdbc, "metadata.json"), []byte("no es json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if worker.javaDBIsFresh() {
		t.Fatal("una metadata ilegible debe forzar la descarga")
	}
}

func hasArg(args []string, want string) bool {
	for _, arg := range args {
		if arg == want {
			return true
		}
	}
	return false
}
