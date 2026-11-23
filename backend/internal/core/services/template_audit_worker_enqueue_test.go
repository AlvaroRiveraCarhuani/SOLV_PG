package services

import (
	"testing"
	"time"

	"solv-backend/internal/core/domain"
)

// EnqueueTemplate adelanta la auditoría: el espaciado por id desaparece y el
// reconciliador recibe el aviso en el canal.
func TestEnqueueTemplateClearsBackoffAndWakes(t *testing.T) {
	worker := &TemplateAuditWorker{
		wakeCh:          make(chan struct{}, 1),
		stopCh:          make(chan struct{}),
		templateRetry:   map[string]time.Time{"tpl-1": time.Now().Add(time.Hour)},
		templateBackoff: map[string]time.Duration{"tpl-1": 10 * time.Minute},
		now:             time.Now,
	}

	worker.EnqueueTemplate("tpl-1")

	worker.retryMu.Lock()
	_, hasRetry := worker.templateRetry["tpl-1"]
	_, hasBackoff := worker.templateBackoff["tpl-1"]
	worker.retryMu.Unlock()
	if hasRetry || hasBackoff {
		t.Fatalf("EnqueueTemplate debía limpiar el espaciado, quedó retry=%v backoff=%v", hasRetry, hasBackoff)
	}

	select {
	case <-worker.wakeCh:
	default:
		t.Fatal("EnqueueTemplate debía dejar un aviso pendiente en wakeCh")
	}

	// Un segundo aviso sin consumir no se acumula ni bloquea al emisor.
	done := make(chan struct{})
	go func() {
		worker.EnqueueTemplate("tpl-1")
		worker.EnqueueTemplate("tpl-1")
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("EnqueueTemplate bloqueó con el canal ya lleno")
	}
}

// El camino HTTP no puede esperar a que termine un escaneo: el worker toma
// w.mu durante toda la auditoría.
func TestEnqueueTemplateDoesNotWaitForRunningAudit(t *testing.T) {
	worker := &TemplateAuditWorker{
		wakeCh:          make(chan struct{}, 1),
		stopCh:          make(chan struct{}),
		templateRetry:   map[string]time.Time{},
		templateBackoff: map[string]time.Duration{},
		now:             time.Now,
	}

	worker.mu.Lock()

	done := make(chan struct{})
	go func() {
		worker.EnqueueTemplate("tpl-2")
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("EnqueueTemplate esperó al mutex de auditoría y bloqueó el request")
	}
	worker.mu.Unlock()
}

// Un re-escaneo que solo espera Trivy reaprovecha el smoke aprobado; si el
// smoke aún no corrió o cambió la imagen, se ejecuta de nuevo.
func TestKeepsPassedSmoke(t *testing.T) {
	cases := []struct {
		name     string
		item     *domain.AdminTemplateReviewItem
		expected bool
	}{
		{
			name:     "smoke aprobado y escaneo pendiente",
			item:     &domain.AdminTemplateReviewItem{SmokeTestStatus: "passed", SecurityAuditStatus: "pending"},
			expected: true,
		},
		{
			name:     "smoke aprobado y escaneo con error de infraestructura",
			item:     &domain.AdminTemplateReviewItem{SmokeTestStatus: "passed", SecurityAuditStatus: "error"},
			expected: true,
		},
		{
			name:     "ambos pendientes",
			item:     &domain.AdminTemplateReviewItem{SmokeTestStatus: "pending", SecurityAuditStatus: "pending"},
			expected: false,
		},
		{
			name:     "escaneo ya ejecutado",
			item:     &domain.AdminTemplateReviewItem{SmokeTestStatus: "passed", SecurityAuditStatus: "failed"},
			expected: false,
		},
		{
			name:     "sin plantilla",
			item:     nil,
			expected: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := keepsPassedSmoke(tc.item); got != tc.expected {
				t.Fatalf("keepsPassedSmoke = %v, quiero %v", got, tc.expected)
			}
		})
	}
}

// El motivo de rechazo solo se escribe cuando la auditoría retira la
// plantilla: una publicada conserva su historial.
func TestRejectionReasonFor(t *testing.T) {
	if got := rejectionReasonFor("APROBADA", "Trivy detectó 2 vulnerabilidades críticas"); got != "" {
		t.Fatalf("una aprobada no debe escribir motivo de rechazo, obtuve %q", got)
	}
	if got := rejectionReasonFor("RECHAZADA", "Faltan herramientas"); got != "Faltan herramientas" {
		t.Fatalf("motivo = %q, quiero el detalle del fallo", got)
	}
}
