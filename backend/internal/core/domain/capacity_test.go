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

// El techo recomendado tiene que crecer con el servidor. Un valor fijo de 40
// dejaba un host de 128 GB trabajando a un tercio de su memoria.
func TestRecommendMaxContainersScalesWithHostRAM(t *testing.T) {
	cases := []struct {
		hostTotalMB int64
		ramPerLabMB int
		ceilingPct  int
		want        int
	}{
		{8192, 512, 85, 9},
		{16384, 512, 85, 23},
		{32768, 512, 85, 50},
		{65536, 512, 85, 104},
		{131072, 512, 85, 213},
		{131072, 256, 85, 427},
		{131072, 1024, 85, 106},
		{1048576, 512, 85, MaxAllowedContainers},
		{1024, 1024, 85, 1},
		{0, 512, 85, 0},
		{16384, 0, 85, 0},
	}

	for _, tc := range cases {
		got := RecommendMaxContainers(tc.hostTotalMB, tc.ramPerLabMB, tc.ceilingPct)
		if got != tc.want {
			t.Errorf("RecommendMaxContainers(%d MB, %d MB/lab, %d%%) = %d, esperado %d",
				tc.hostTotalMB, tc.ramPerLabMB, tc.ceilingPct, got, tc.want)
		}
	}
}

func TestRecommendMaxContainersFallsBackToDefaultCeiling(t *testing.T) {
	// Sin techo de seguridad válido se usa el de las políticas por defecto (85),
	// no se calcula sobre 0%.
	defaultCeiling := DefaultServerPolicies().HostSafetyCeilingPct
	if got, want := RecommendMaxContainers(65536, 512, 0), RecommendMaxContainers(65536, 512, defaultCeiling); got != want {
		t.Fatalf("expected %d with the default ceiling, got %d", want, got)
	}
}

func TestBuildCapacityRecommendationWithoutMeasuredHost(t *testing.T) {
	recommendation := BuildCapacityRecommendation(0, 512, 85)

	if recommendation.RecommendedMaxContainers != 0 {
		t.Fatalf("sin host medido no se recomienda nada, got %d", recommendation.RecommendedMaxContainers)
	}
	if recommendation.CapacityBasis != "unavailable" {
		t.Fatalf("expected basis unavailable, got %q", recommendation.CapacityBasis)
	}
	if recommendation.PlatformReserveMB != PlatformReserveMB {
		t.Fatalf("la recomendación debe declarar la reserva usada, got %d", recommendation.PlatformReserveMB)
	}
}

func TestBuildCapacityRecommendationReportsItsAssumptions(t *testing.T) {
	recommendation := BuildCapacityRecommendation(65536, 512, 85)

	if recommendation.RecommendedMaxContainers != 104 {
		t.Fatalf("expected 104 labs on a 64 GB host, got %d", recommendation.RecommendedMaxContainers)
	}
	if recommendation.CapacityBasis != CapacityBasisHostRAM {
		t.Fatalf("expected basis %q, got %q", CapacityBasisHostRAM, recommendation.CapacityBasis)
	}
	if recommendation.HostTotalMB != 65536 || recommendation.RAMPerLabMB != 512 || recommendation.HostSafetyCeilingPct != 85 {
		t.Fatalf("la recomendación debe devolver los supuestos usados, got %+v", recommendation)
	}
}
