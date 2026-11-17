package services

import (
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

func hasArg(args []string, want string) bool {
	for _, arg := range args {
		if arg == want {
			return true
		}
	}
	return false
}
