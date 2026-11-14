package services

import (
	"strings"
	"testing"
)

func TestResolveAuditOutcome(t *testing.T) {
	cases := []struct {
		name          string
		current       string
		smokeStatus   string
		smokeOutput   string
		secStatus     string
		cveCritical   int
		wantStatus    string
		wantReasonSub string
	}{
		{
			name:        "una plantilla en auditoría se publica cuando todo pasa",
			current:     "PENDIENTE_AUDITORIA",
			smokeStatus: "passed",
			secStatus:   "passed",
			wantStatus:  "APROBADA",
		},
		{
			name:        "una plantilla aprobada sigue aprobada cuando todo pasa",
			current:     "APROBADA",
			smokeStatus: "passed",
			secStatus:   "passed",
			wantStatus:  "APROBADA",
		},
		{
			name:          "el smoke fallido retira una plantilla aprobada",
			current:       "APROBADA",
			smokeStatus:   "failed",
			smokeOutput:   "Faltan las siguientes herramientas declaradas en la imagen: pip",
			secStatus:     "passed",
			wantStatus:    "RECHAZADA",
			wantReasonSub: "Faltan las siguientes herramientas",
		},
		{
			name:          "el smoke fallido rechaza una plantilla que aún no se publicó",
			current:       "PENDIENTE_AUDITORIA",
			smokeStatus:   "failed",
			smokeOutput:   "La imagen no contiene las herramientas requeridas",
			secStatus:     "passed",
			wantStatus:    "RECHAZADA",
			wantReasonSub: "no contiene las herramientas",
		},
		{
			name:          "Trivy con CVEs críticos retira una plantilla aprobada",
			current:       "APROBADA",
			smokeStatus:   "passed",
			secStatus:     "failed",
			cveCritical:   2,
			wantStatus:    "RECHAZADA",
			wantReasonSub: "2 vulnerabilidades críticas",
		},
		{
			name:          "un escaneo que no se pudo completar tampoco publica",
			current:       "PENDIENTE_AUDITORIA",
			smokeStatus:   "passed",
			secStatus:     "failed",
			wantStatus:    "RECHAZADA",
			wantReasonSub: "No se pudo completar el escaneo",
		},
		{
			name:          "smoke y Trivy fallidos dejan ambos motivos",
			current:       "APROBADA",
			smokeStatus:   "failed",
			smokeOutput:   "Faltan herramientas",
			secStatus:     "failed",
			cveCritical:   1,
			wantStatus:    "RECHAZADA",
			wantReasonSub: "Faltan herramientas. Trivy detectó 1 vulnerabilidad crítica",
		},
		{
			name:        "una pausada no se auto-reactiva con una auditoría correcta",
			current:     "PAUSADA",
			smokeStatus: "passed",
			secStatus:   "passed",
			wantStatus:  "PAUSADA",
		},
		{
			name:          "una suspendida conserva su estado aunque falle la auditoría",
			current:       "SUSPENDIDA",
			smokeStatus:   "failed",
			smokeOutput:   "Faltan herramientas",
			secStatus:     "passed",
			wantStatus:    "SUSPENDIDA",
			wantReasonSub: "Faltan herramientas",
		},
		{
			name:        "una rechazada no vuelve al catálogo por una auditoría correcta",
			current:     "RECHAZADA",
			smokeStatus: "passed",
			secStatus:   "passed",
			wantStatus:  "RECHAZADA",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotStatus, gotReason := resolveAuditOutcome(
				tc.current, tc.smokeStatus, tc.smokeOutput, tc.secStatus, tc.cveCritical,
			)
			if gotStatus != tc.wantStatus {
				t.Fatalf("estado final = %q, quiero %q (motivo: %q)", gotStatus, tc.wantStatus, gotReason)
			}
			if tc.wantReasonSub != "" && !strings.Contains(gotReason, tc.wantReasonSub) {
				t.Fatalf("motivo = %q, quiero que contenga %q", gotReason, tc.wantReasonSub)
			}
			if tc.wantReasonSub == "" && gotReason != "" {
				t.Fatalf("no esperaba motivo, obtuve %q", gotReason)
			}
		})
	}
}
