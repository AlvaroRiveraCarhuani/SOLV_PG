package domain

import (
	"reflect"
	"testing"
)

// TestEvaluateHealthOperational confirma el caso nominal: sin hechos adversos el
// panel informa operational y sin motivos (no inventa un tooltip).
func TestEvaluateHealthOperational(t *testing.T) {
	status, reasons := EvaluateHealth(HealthEvaluationInput{
		DockerHealthy:    true,
		OrphanedLabs:     0,
		OOMKilledLabs:    0,
		UnknownLabs:      0,
		RAMPercent:       40,
		SafetyCeilingPct: 85,
	})
	if status != HealthStatusOperational {
		t.Fatalf("expected operational, got %q", status)
	}
	if len(reasons) != 0 {
		t.Fatalf("expected no reasons on a healthy host, got %v", reasons)
	}
}

// TestEvaluateHealthDockerDown critica el daemon inaccesible aunque todo lo
// demas este limpio: sin Docker no hay entornos verificables.
func TestEvaluateHealthDockerDown(t *testing.T) {
	status, reasons := EvaluateHealth(HealthEvaluationInput{
		DockerHealthy:    false,
		OrphanedLabs:     0,
		OOMKilledLabs:    0,
		UnknownLabs:      0,
		RAMPercent:       30,
		SafetyCeilingPct: 85,
	})
	if status != HealthStatusCritical {
		t.Fatalf("expected critical, got %q", status)
	}
	if !reflect.DeepEqual(reasons, []string{HealthReasonDockerUnreachable}) {
		t.Fatalf("expected [docker_unreachable], got %v", reasons)
	}
}

// TestEvaluateHealthOrphanedCritical comprueba que un contenedor huerfano
// deriva en critical: hay registros cuyo entorno fisico desaparecio.
func TestEvaluateHealthOrphanedCritical(t *testing.T) {
	status, reasons := EvaluateHealth(HealthEvaluationInput{
		DockerHealthy:    true,
		OrphanedLabs:     2,
		OOMKilledLabs:    0,
		UnknownLabs:      0,
		RAMPercent:       30,
		SafetyCeilingPct: 85,
	})
	if status != HealthStatusCritical {
		t.Fatalf("expected critical with orphans, got %q", status)
	}
	if !reflect.DeepEqual(reasons, []string{HealthReasonOrphanedWorkspaces}) {
		t.Fatalf("expected [orphaned_workspaces], got %v", reasons)
	}
}

// TestEvaluateHealthOOMMetrics degrada con laboratorios detenidos por memoria.
func TestEvaluateHealthOOMMetrics(t *testing.T) {
	status, reasons := EvaluateHealth(HealthEvaluationInput{
		DockerHealthy:    true,
		OrphanedLabs:     0,
		OOMKilledLabs:    3,
		UnknownLabs:      0,
		RAMPercent:       30,
		SafetyCeilingPct: 85,
	})
	if status != HealthStatusDegraded {
		t.Fatalf("expected degraded with OOMs, got %q", status)
	}
	if !reflect.DeepEqual(reasons, []string{HealthReasonOOMKilledWorkspaces}) {
		t.Fatalf("expected [oom_killed_workspaces], got %v", reasons)
	}
}

// TestEvaluateHealthUnknownDegrades comprueba que estados sin verificar bajan
// la salud sin llegar a critical: el registro sigue intacto y reintenta.
func TestEvaluateHealthUnknownDegrades(t *testing.T) {
	status, reasons := EvaluateHealth(HealthEvaluationInput{
		DockerHealthy:    true,
		OrphanedLabs:     0,
		OOMKilledLabs:    0,
		UnknownLabs:      4,
		RAMPercent:       30,
		SafetyCeilingPct: 85,
	})
	if status != HealthStatusDegraded {
		t.Fatalf("expected degraded with unknown states, got %q", status)
	}
	if !reflect.DeepEqual(reasons, []string{HealthReasonUnknownWorkspaces}) {
		t.Fatalf("expected [unknown_workspaces], got %v", reasons)
	}
}

// TestEvaluateHealthRAMOverCeiling degrada cuando la RAM medida supera el techo
// de seguridad de la politica vigente.
func TestEvaluateHealthRAMOverCeiling(t *testing.T) {
	status, reasons := EvaluateHealth(HealthEvaluationInput{
		DockerHealthy:    true,
		RAMPercent:       91.4,
		SafetyCeilingPct: 85,
	})
	if status != HealthStatusDegraded {
		t.Fatalf("expected degraded with RAM over ceiling, got %q", status)
	}
	if !reflect.DeepEqual(reasons, []string{HealthReasonRAMAboveSafetyCeiling}) {
		t.Fatalf("expected [ram_above_safety_ceiling], got %v", reasons)
	}
}

// TestEvaluateHealthRAMAtCeilingIsOperational el limite es estricto: en el
// techo o por debajo no se marca degradado.
func TestEvaluateHealthRAMAtCeilingIsOperational(t *testing.T) {
	status, _ := EvaluateHealth(HealthEvaluationInput{
		DockerHealthy:    true,
		RAMPercent:       85.0,
		SafetyCeilingPct: 85,
	})
	if status != HealthStatusOperational {
		t.Fatalf("expected operational at ceiling, got %q", status)
	}
}

// TestEvaluateHealthDefaultCeiling usa el techo de la politica por defecto
// cuando el input no trae uno (fallo de lectura de politica no debe bloquear).
func TestEvaluateHealthDefaultCeiling(t *testing.T) {
	status, reasons := EvaluateHealth(HealthEvaluationInput{
		DockerHealthy: true,
		RAMPercent:    90,
	})
	defaults := DefaultServerPolicies()
	if float64(defaults.HostSafetyCeilingPct) >= 90 {
		t.Fatalf("test assumption broken: default ceiling %d must be < 90", defaults.HostSafetyCeilingPct)
	}
	if status != HealthStatusDegraded {
		t.Fatalf("expected degraded with default ceiling, got %q", status)
	}
	if len(reasons) != 1 || reasons[0] != HealthReasonRAMAboveSafetyCeiling {
		t.Fatalf("expected [ram_above_safety_ceiling], got %v", reasons)
	}
}

// TestEvaluateHealthStackedConditions verifica la matriz completa: con todos
// los hechos adversos presentes el estado es critical y los motivos se
// acumulan en orden estable (docker, orphaned, oom, unknown, ram).
func TestEvaluateHealthStackedConditions(t *testing.T) {
	status, reasons := EvaluateHealth(HealthEvaluationInput{
		DockerHealthy:    false,
		OrphanedLabs:     1,
		OOMKilledLabs:    1,
		UnknownLabs:      1,
		RAMPercent:       99,
		SafetyCeilingPct: 85,
	})
	if status != HealthStatusCritical {
		t.Fatalf("expected critical (docker+orphaned manda), got %q", status)
	}
	expected := []string{
		HealthReasonDockerUnreachable,
		HealthReasonOrphanedWorkspaces,
		HealthReasonOOMKilledWorkspaces,
		HealthReasonUnknownWorkspaces,
		HealthReasonRAMAboveSafetyCeiling,
	}
	if !reflect.DeepEqual(reasons, expected) {
		t.Fatalf("expected %v, got %v", expected, reasons)
	}
}

// TestEvaluateHealthDegradedNotCritical con solo causas menores el estado queda
// en degraded, no en critical: la matriz prioriza la severidad.
func TestEvaluateHealthDegradedNotCritical(t *testing.T) {
	status, _ := EvaluateHealth(HealthEvaluationInput{
		DockerHealthy:    true,
		OOMKilledLabs:    1,
		UnknownLabs:      1,
		RAMPercent:       90,
		SafetyCeilingPct: 85,
	})
	if status != HealthStatusDegraded {
		t.Fatalf("expected degraded (no docker/orphaned), got %q", status)
	}
}
