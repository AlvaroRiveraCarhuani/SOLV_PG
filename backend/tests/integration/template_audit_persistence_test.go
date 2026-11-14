package integration

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"solv-backend/internal/core/domain"
	"solv-backend/internal/infrastructure/database"
	"solv-backend/internal/infrastructure/storage/postgres"
)

// La degradación por auditoría y el reinicio por cambio técnico son
// condiciones SQL. Un repositorio simulado no distingue una fila que cambió de
// estado de una que solo recibió el argumento correcto.
func TestTemplateAuditPersistence(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = getTestDSN()
	}
	db, err := database.NewPostgresDB(dsn)
	if err != nil {
		skipOrFailInfra(t, "PostgreSQL no disponible para la auditoría de plantillas: %v", err)
		return
	}
	defer func() { _ = db.GetDB().Close() }()

	repo := postgres.NewPostgresAdminGovernanceRepository(db.GetDB())
	ctx := context.Background()

	tenantID := uuid.NewString()
	suffix := tenantID[:8]
	if _, errTenant := db.GetDB().Exec(`
		INSERT INTO tenants (id, name, slug) VALUES ($1, 'Auditoría de plantillas', $2)
	`, tenantID, "it-audit-"+suffix); errTenant != nil {
		t.Fatalf("no se pudo sembrar el tenant de la prueba: %v", errTenant)
	}
	t.Cleanup(func() {
		_, _ = db.GetDB().Exec(`DELETE FROM lab_templates WHERE tenant_id = $1`, tenantID)
		_, _ = db.GetDB().Exec(`DELETE FROM tenants WHERE id = $1`, tenantID)
	})

	seedTemplate := func(t *testing.T, status, name string, extra ...string) string {
		t.Helper()
		id := uuid.NewString()
		reason := ""
		if len(extra) > 0 {
			reason = extra[0]
		}
		if _, errSeed := db.GetDB().Exec(`
			INSERT INTO lab_templates (
				id, tenant_id, name, docker_image, base_ram_mb, status,
				rejection_reason, smoke_test_status, security_audit_status
			) VALUES ($1, $2, $3, 'python:3.12-slim-bookworm', 512, $4, $5, 'pending', 'pending')
		`, id, tenantID, name, status, reason); errSeed != nil {
			t.Fatalf("no se pudo sembrar la plantilla %q: %v", name, errSeed)
		}
		return id
	}

	readAudit := func(t *testing.T, id string) (status, reason, smoke, security string, critical int, audited bool) {
		t.Helper()
		errRead := db.GetDB().QueryRow(`
			SELECT status, COALESCE(rejection_reason, ''), COALESCE(smoke_test_status, ''),
			       COALESCE(security_audit_status, ''), COALESCE(cve_critical_count, 0),
			       security_audited_at IS NOT NULL
			FROM lab_templates WHERE id = $1
		`, id).Scan(&status, &reason, &smoke, &security, &critical, &audited)
		if errRead != nil {
			t.Fatalf("no se pudo leer la auditoría de %s: %v", id, errRead)
		}
		return
	}

	t.Run("una auditoría aprobada limpia el motivo anterior", func(t *testing.T) {
		id := seedTemplate(t, "PENDIENTE_AUDITORIA", "Auditoría aprobada "+suffix, "motivo de un rechazo anterior")
		if errUpdate := repo.UpdateAuditResults(
			ctx, id,
			"passed", "", "passed", 0, 0, []byte(`{"ok":true}`),
			"APROBADA", "",
		); errUpdate != nil {
			t.Fatalf("no se pudo persistir la auditoría aprobada: %v", errUpdate)
		}
		status, reason, smoke, security, critical, audited := readAudit(t, id)
		if status != "APROBADA" || smoke != "passed" || security != "passed" {
			t.Fatalf("estado inesperado: status=%q smoke=%q security=%q", status, smoke, security)
		}
		if reason != "" {
			t.Fatalf("el motivo anterior debía limpiarse, quedó %q", reason)
		}
		if critical != 0 || !audited {
			t.Fatalf("esperado 0 críticos y fecha de auditoría, obtenido críticos=%d auditada=%v", critical, audited)
		}
	})

	t.Run("un smoke fallido degrada y publica el motivo", func(t *testing.T) {
		id := seedTemplate(t, "APROBADA", "Smoke fallido "+suffix)
		const motivo = "Falta la herramienta: ripgrep"
		if errUpdate := repo.UpdateAuditResults(
			ctx, id,
			"failed", motivo, "passed", 0, 0, []byte(`{"ok":true}`),
			"RECHAZADA", motivo,
		); errUpdate != nil {
			t.Fatalf("no se pudo persistir el smoke fallido: %v", errUpdate)
		}
		status, reason, smoke, _, _, _ := readAudit(t, id)
		if status != "RECHAZADA" || smoke != "failed" {
			t.Fatalf("estado inesperado: status=%q smoke=%q", status, smoke)
		}
		if reason != motivo {
			t.Fatalf("motivo inesperado: %q", reason)
		}
	})

	t.Run("un Trivy con críticos degrada y guarda el conteo", func(t *testing.T) {
		id := seedTemplate(t, "APROBADA", "CVE crítico "+suffix)
		const motivo = "Trivy detectó 3 vulnerabilidades críticas"
		if errUpdate := repo.UpdateAuditResults(
			ctx, id,
			"passed", "", "failed", 3, 1, []byte(`{"Critical":3,"High":1}`),
			"RECHAZADA", motivo,
		); errUpdate != nil {
			t.Fatalf("no se pudo persistir el hallazgo crítico: %v", errUpdate)
		}
		status, reason, _, security, critical, _ := readAudit(t, id)
		if status != "RECHAZADA" || security != "failed" || critical != 3 {
			t.Fatalf("estado inesperado: status=%q security=%q critical=%d", status, security, critical)
		}
		if !strings.Contains(reason, "3") {
			t.Fatalf("el motivo debía mencionar los 3 críticos, quedó %q", reason)
		}
	})

	t.Run("la cola de auditoría incluye la plantilla pendiente", func(t *testing.T) {
		id := seedTemplate(t, "PENDIENTE_AUDITORIA", "Pendiente de cola "+suffix)
		pending, errList := repo.ListPendingAuditTemplates(ctx)
		if errList != nil {
			t.Fatalf("no se pudo consultar la cola de auditoría: %v", errList)
		}
		found := false
		for _, item := range pending {
			if item.ID == id {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("la plantilla %s no apareció en la cola de auditoría", id)
		}
	})

	t.Run("un cambio técnico reinicia la auditoría", func(t *testing.T) {
		id := seedTemplate(t, "APROBADA", "Cambio técnico "+suffix)
		if _, errSeed := db.GetDB().Exec(`
			UPDATE lab_templates
			SET smoke_test_status = 'passed', security_audit_status = 'passed',
			    cve_critical_count = 2, security_audited_at = NOW()
			WHERE id = $1
		`, id); errSeed != nil {
			t.Fatalf("no se pudo preparar la auditoría anterior: %v", errSeed)
		}

		updated, errUpdate := repo.UpdateOfficialTemplate(ctx, tenantID, id, "", domain.UpdateOfficialTemplateDTO{
			Name:              "Cambio técnico " + suffix,
			DockerImage:       "python:3.13-slim-bookworm",
			BaseRamMB:         768,
			TargetEnvironment: "IDE_PERSISTENTE",
			ToolsDeclared:     []string{"git"},
		})
		if errUpdate != nil {
			t.Fatalf("no se pudo actualizar la plantilla: %v", errUpdate)
		}
		if updated.Status != "PENDIENTE_AUDITORIA" {
			t.Fatalf("esperado PENDIENTE_AUDITORIA, obtenido %q", updated.Status)
		}
		status, _, smoke, security, _, _ := readAudit(t, id)
		if status != "PENDIENTE_AUDITORIA" || smoke != "pending" || security != "pending" {
			t.Fatalf("reinicio inesperado: status=%q smoke=%q security=%q", status, smoke, security)
		}
	})

	t.Run("pausada y suspendida conservan su estado", func(t *testing.T) {
		for _, previous := range []string{"PAUSADA", "SUSPENDIDA"} {
			previous := previous
			t.Run(previous, func(t *testing.T) {
				id := seedTemplate(t, previous, previous+" "+suffix)
				updated, errUpdate := repo.UpdateOfficialTemplate(ctx, tenantID, id, "", domain.UpdateOfficialTemplateDTO{
					Name:              previous + " " + suffix,
					DockerImage:       "python:3.13-slim-bookworm",
					BaseRamMB:         768,
					TargetEnvironment: "IDE_PERSISTENTE",
				})
				if errUpdate != nil {
					t.Fatalf("no se pudo actualizar la plantilla %s: %v", previous, errUpdate)
				}
				if updated.Status != previous {
					t.Fatalf("el estado %s debía conservarse, quedó %q", previous, updated.Status)
				}
				_, _, smoke, security, _, _ := readAudit(t, id)
				if smoke != "pending" || security != "pending" {
					t.Fatalf("la auditoría debía reiniciarse en %s: smoke=%q security=%q", previous, smoke, security)
				}
			})
		}
	})
}
