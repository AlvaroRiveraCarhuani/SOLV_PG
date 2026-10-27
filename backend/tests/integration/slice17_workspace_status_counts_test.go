package integration

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"solv-backend/internal/infrastructure/database"
	"solv-backend/internal/infrastructure/storage/postgres"
)

// seedTenantWithSubject inserta un tenant y un subject de prueba, devolviendo
// sus IDs. El subject es necesario porque workspaces.subject_id tiene FK a
// subjects(id) ON DELETE RESTRICT.
func seedTenantWithSubject(t *testing.T, ctx context.Context, sqlDB *sqlx.DB, name, slug string) (tenantID, subjectID string) {
	t.Helper()
	tenantID = uuid.NewString()
	subjectID = uuid.NewString()

	if _, err := sqlDB.ExecContext(ctx, `
		INSERT INTO tenants (id, name, slug) VALUES ($1, $2, $3)
	`, tenantID, name, slug); err != nil {
		t.Fatalf("failed to seed tenant: %v", err)
	}
	if _, err := sqlDB.ExecContext(ctx, `
		INSERT INTO subjects (id, tenant_id, name, code) VALUES ($1, $2, $3, $4)
	`, subjectID, tenantID, name+" Subject", "SUB-"+slug); err != nil {
		t.Fatalf("failed to seed subject: %v", err)
	}
	return tenantID, subjectID
}

// countWorkspacesByStatusLive accede al metodo concreto del repositorio postgres,
// que no forma parte de la interfaz domain.WorkspaceRepository.
func countWorkspacesByStatusLive(t *testing.T, sqlDB *sqlx.DB, ctx context.Context, tenantID string) (map[string]int, error) {
	t.Helper()
	repoIface := postgres.NewPostgresWorkspaceRepository(sqlDB)
	repo, ok := repoIface.(*postgres.PostgresWorkspaceRepository)
	if !ok {
		t.Fatalf("test database returned unexpected repository type")
	}
	return repo.CountWorkspacesByStatus(ctx, tenantID)
}

// TestCountWorkspacesByStatusLiveDB valida el agregado SQL real contra la base
// de tests de integración (solv_test), no contra un mock: una columna mal
// aliasada o un GROUP BY roto solo se detectan contra un Postgres real.
//
// Aislamiento:
//   - Escribe solo en el tenant de prueba creado acá, nunca en filas existentes.
//   - Limpia sus filas, subject y tenant al terminar (t.Cleanup).
//   - NUNCA contamina solv_db: getTestDSN apunta a solv_test.
func TestCountWorkspacesByStatusLiveDB(t *testing.T) {
	ctx := context.Background()
	db, err := database.NewPostgresDB(getTestDSN())
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}
	sqlDB := db.GetDB()

	tenantID, subjectID := seedTenantWithSubject(t, ctx, sqlDB, "StatusCounts Test Tenant", "status-counts-test-tenant")
	otherTenantID, otherSubjectID := seedTenantWithSubject(t, ctx, sqlDB, "Otra Tenant", "otra-tenant-status-counts")

	t.Cleanup(func() {
		_, _ = sqlDB.ExecContext(ctx, `DELETE FROM workspaces WHERE tenant_id IN ($1, $2)`, tenantID, otherTenantID)
		_, _ = sqlDB.ExecContext(ctx, `DELETE FROM subjects WHERE id IN ($1, $2)`, subjectID, otherSubjectID)
		_, _ = sqlDB.ExecContext(ctx, `DELETE FROM tenants WHERE id IN ($1, $2)`, tenantID, otherTenantID)
	})

	statuses := []string{"running", "running", "hibernated", "failed", "oom_killed", "pending"}
	for i, status := range statuses {
		targetTenant := tenantID
		targetSubject := subjectID
		if i == 4 {
			// Un oom_killed vive en el OTRO tenant: no debe sumarse al desglose
			// del tenant bajo prueba.
			targetTenant = otherTenantID
			targetSubject = otherSubjectID
		}
		wsID := uuid.NewString()
		if _, err := sqlDB.ExecContext(ctx, `
			INSERT INTO workspaces (id, student_id, subject_id, tenant_id, status, access_url)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, wsID, uuid.NewString(), targetSubject, targetTenant, status, "http://test.solv.local"); err != nil {
			t.Fatalf("failed to seed workspace %d (%s): %v", i, status, err)
		}
	}

	counts, err := countWorkspacesByStatusLive(t, sqlDB, ctx, tenantID)
	if err != nil {
		t.Fatalf("CountWorkspacesByStatus failed: %v", err)
	}

	expected := map[string]int{
		"running":     2,
		"hibernated": 1,
		"failed":      1,
		"pending":     1,
	}
	// oom_killed del otro tenant NO debe aparecer en este desglose.
	if len(counts) != len(expected) {
		t.Fatalf("expected %d distinct statuses, got %d (%v)", len(expected), len(counts), counts)
	}
	for status, want := range expected {
		got := counts[status]
		if got != want {
			t.Errorf("status %q: expected %d, got %d (counts=%v)", status, want, got, counts)
		}
	}
	if counts["oom_killed"] != 0 {
		t.Errorf("oom_killed from another tenant leaked into the breakdown: got %d", counts["oom_killed"])
	}

	t.Logf("CountWorkspacesByStatus live-DB assertion passed: %v (tenant=%s)", counts, tenantID)
}

// TestCountWorkspacesByStatusIsolation confirma, con una segunda lectura, que el
// desglose del otro tenant no ve filas ajenas.
func TestCountWorkspacesByStatusIsolation(t *testing.T) {
	ctx := context.Background()
	db, err := database.NewPostgresDB(getTestDSN())
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}
	sqlDB := db.GetDB()

	tenantID, subjectID := seedTenantWithSubject(t, ctx, sqlDB, "Isolation Tenant", "isolation-tenant-counts")
	otherTenantID, otherSubjectID := seedTenantWithSubject(t, ctx, sqlDB, "Otra", "otra-isolation-counts")

	t.Cleanup(func() {
		_, _ = sqlDB.ExecContext(ctx, `DELETE FROM workspaces WHERE tenant_id IN ($1, $2)`, tenantID, otherTenantID)
		_, _ = sqlDB.ExecContext(ctx, `DELETE FROM subjects WHERE id IN ($1, $2)`, subjectID, otherSubjectID)
		_, _ = sqlDB.ExecContext(ctx, `DELETE FROM tenants WHERE id IN ($1, $2)`, tenantID, otherTenantID)
	})

	insertWS := func(tenant, subject, status string) {
		if _, err := sqlDB.ExecContext(ctx, `
			INSERT INTO workspaces (id, student_id, subject_id, tenant_id, status, access_url)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, uuid.NewString(), uuid.NewString(), subject, tenant, status, "http://test.solv.local"); err != nil {
			t.Fatalf("failed to seed workspace (%s): %v", status, err)
		}
	}

	insertWS(tenantID, subjectID, "hibernated")
	insertWS(tenantID, subjectID, "hibernated")
	insertWS(otherTenantID, otherSubjectID, "running")
	insertWS(otherTenantID, otherSubjectID, "running")
	insertWS(otherTenantID, otherSubjectID, "failed")

	counts, err := countWorkspacesByStatusLive(t, sqlDB, ctx, tenantID)
	if err != nil {
		t.Fatalf("CountWorkspacesByStatus failed: %v", err)
	}

	if counts["hibernated"] != 2 {
		t.Errorf("expected 2 hibernated for this tenant, got %d (%v)", counts["hibernated"], counts)
	}
	if counts["running"] != 0 || counts["failed"] != 0 {
		t.Errorf("another tenant's workspaces leaked into the breakdown: %v", counts)
	}

	t.Logf("isolation assertion passed: %v", counts)
}

// TestCountWorkspacesByStatusReportsCounts valida el desglose completo con las
// cinco filas presentes en un solo tenant.
func TestCountWorkspacesByStatusReportsCounts(t *testing.T) {
	ctx := context.Background()
	db, err := database.NewPostgresDB(getTestDSN())
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}
	sqlDB := db.GetDB()

	tenantID, subjectID := seedTenantWithSubject(t, ctx, sqlDB, "ReportsCounts Tenant", "reports-counts-tenant")
	t.Cleanup(func() {
		_, _ = sqlDB.ExecContext(ctx, `DELETE FROM workspaces WHERE tenant_id = $1`, tenantID)
		_, _ = sqlDB.ExecContext(ctx, `DELETE FROM subjects WHERE id = $1`, subjectID)
		_, _ = sqlDB.ExecContext(ctx, `DELETE FROM tenants WHERE id = $1`, tenantID)
	})

	statuses := []string{"running", "hibernated", "hibernated", "failed", "oom_killed"}
	for _, status := range statuses {
		if _, err := sqlDB.ExecContext(ctx, `
			INSERT INTO workspaces (id, student_id, subject_id, tenant_id, status, access_url)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, uuid.NewString(), uuid.NewString(), subjectID, tenantID, status, "http://test.solv.local"); err != nil {
			t.Fatalf("failed to seed workspace (%s): %v", status, err)
		}
	}

	counts, err := countWorkspacesByStatusLive(t, sqlDB, ctx, tenantID)
	if err != nil {
		t.Fatalf("CountWorkspacesByStatus failed: %v", err)
	}

	expected := map[string]int{"running": 1, "hibernated": 2, "failed": 1, "oom_killed": 1}
	if len(counts) != len(expected) {
		t.Fatalf("expected %d distinct statuses, got %d (%v)", len(expected), len(counts), counts)
	}
	var mismatch []string
	for status, want := range expected {
		if counts[status] != want {
			mismatch = append(mismatch, fmt.Sprintf("%s:want=%d/got=%d", status, want, counts[status]))
		}
	}
	if len(mismatch) > 0 {
		t.Errorf("breakdown mismatch: %v (counts=%v)", mismatch, counts)
	}

	t.Logf("full-breakdown assertion passed: %v", counts)
}
