package domain

import "testing"

func TestEstimateCapacityIsLimitedByAvailableMemory(t *testing.T) {
	// 3.3 GB disponibles y labs de 512 MB: entran 6, muy por debajo del techo
	// de politica de 40. Esta es la correccion del valor que antes se calculaba
	// sobre el techo absoluto de RAM sin descontar lo ya usado.
	estimated, basis := EstimateCapacity(3353, 512, 40, 0)

	if estimated != 6 {
		t.Fatalf("expected 6 containers, got %d", estimated)
	}
	if basis != CapacityBasisMemAvailable {
		t.Fatalf("expected basis %q, got %q", CapacityBasisMemAvailable, basis)
	}
}

func TestEstimateCapacityIsLimitedByPolicyCap(t *testing.T) {
	// Con RAM de sobra el limite real es la politica del tenant, no la memoria.
	estimated, basis := EstimateCapacity(40000, 512, 40, 0)

	if estimated != 40 {
		t.Fatalf("expected the policy cap of 40, got %d", estimated)
	}
	if basis != CapacityBasisPolicy {
		t.Fatalf("expected basis %q, got %q", CapacityBasisPolicy, basis)
	}
}

func TestEstimateCapacityNeverReportsBelowRunning(t *testing.T) {
	// Sin RAM disponible pero con 5 laboratorios corriendo, el panel no puede
	// informar una capacidad menor que la realidad observable.
	estimated, _ := EstimateCapacity(0, 512, 40, 5)

	if estimated != 40 {
		t.Fatalf("expected the policy cap when memory is unmeasurable, got %d", estimated)
	}

	estimated, _ = EstimateCapacity(512, 512, 40, 5)
	if estimated != 5 {
		t.Fatalf("expected at least 5 running containers, got %d", estimated)
	}
}

func TestEstimateCapacityFallsBackToPolicyDefaults(t *testing.T) {
	// Un tenant sin politicas persistidas no puede producir una division por
	// cero ni un techo de cero.
	estimated, basis := EstimateCapacity(3353, 0, 0, 0)
	defaults := DefaultServerPolicies()

	if estimated <= 0 {
		t.Fatalf("expected a positive estimate, got %d", estimated)
	}
	if estimated > defaults.MaxContainers {
		t.Fatalf("estimate %d exceeded the default policy cap %d", estimated, defaults.MaxContainers)
	}
	if basis != CapacityBasisMemAvailable {
		t.Fatalf("expected basis %q, got %q", CapacityBasisMemAvailable, basis)
	}
}
