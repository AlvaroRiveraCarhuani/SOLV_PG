package domain

import "testing"

func TestEvaluateDiskQuota(t *testing.T) {
	const mb = int64(1024 * 1024)
	casos := []struct {
		nombre   string
		usedMB   int64
		quotaMB  int64
		warnPct  int
		blockPct int
		esperado DiskQuotaStatus
	}{
		{nombre: "sin cuota nunca bloquea", usedMB: 9000, quotaMB: 0, warnPct: 80, blockPct: 100, esperado: DiskQuotaOK},
		{nombre: "por debajo del aviso", usedMB: 100, quotaMB: 1024, warnPct: 80, blockPct: 100, esperado: DiskQuotaOK},
		{nombre: "justo en el aviso", usedMB: 820, quotaMB: 1024, warnPct: 80, blockPct: 100, esperado: DiskQuotaWarning},
		{nombre: "en la cuota exacta bloquea", usedMB: 1024, quotaMB: 1024, warnPct: 80, blockPct: 100, esperado: DiskQuotaFull},
		{nombre: "por encima de la cuota bloquea", usedMB: 2048, quotaMB: 1024, warnPct: 80, blockPct: 100, esperado: DiskQuotaFull},
		{nombre: "umbrales inválidos caen a 80/100", usedMB: 900, quotaMB: 1024, warnPct: 0, blockPct: 0, esperado: DiskQuotaWarning},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			veredicto := EvaluateDiskQuota(caso.usedMB*mb, caso.quotaMB, caso.warnPct, caso.blockPct)
			if veredicto.Status != caso.esperado {
				t.Fatalf("estado = %s, esperado %s (%s)", veredicto.Status, caso.esperado, veredicto.Message)
			}
			if caso.quotaMB > 0 && veredicto.LimitMB != caso.quotaMB {
				t.Fatalf("límite = %d, esperado %d", veredicto.LimitMB, caso.quotaMB)
			}
		})
	}
}

func TestEvaluateDiskQuotaRedondeaHaciaArriba(t *testing.T) {
	veredicto := EvaluateDiskQuota(int64(1024*1024)+1, 1024, 80, 100)
	if veredicto.UsedMB != 2 {
		t.Fatalf("used_mb = %d, esperado 2", veredicto.UsedMB)
	}
}

func TestIsHostDiskCritical(t *testing.T) {
	casos := []struct {
		nombre   string
		freePct  float64
		floorPct float64
		esperado bool
	}{
		{nombre: "nodo con espacio", freePct: 34, floorPct: 5, esperado: false},
		{nombre: "justo en el piso", freePct: 5, floorPct: 5, esperado: false},
		{nombre: "por debajo del piso", freePct: 2.5, floorPct: 5, esperado: true},
		{nombre: "sin piso declarado usa 5", freePct: 3, floorPct: 0, esperado: true},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			if got := IsHostDiskCritical(caso.freePct, caso.floorPct); got != caso.esperado {
				t.Fatalf("IsHostDiskCritical(%v, %v) = %v, esperado %v", caso.freePct, caso.floorPct, got, caso.esperado)
			}
		})
	}
}

func TestCPUQuotaNanos(t *testing.T) {
	if got := CPUQuotaNanos(0); got != 0 {
		t.Fatalf("sin política debe quedar sin límite, got %d", got)
	}
	if got := CPUQuotaNanos(1500); got != 1_500_000_000 {
		t.Fatalf("1500 milésimas deben ser 1.5 núcleos, got %d", got)
	}
}
