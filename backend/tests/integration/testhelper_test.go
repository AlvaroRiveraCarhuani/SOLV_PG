package integration

import (
	"testing"
)

func TestValidateTestDSNAllowsOnlyTestTargets(t *testing.T) {
	valid := []string{
		"postgres://solv_user:secreto@127.0.0.1:5433/solv_test_alrivera_123?sslmode=disable",
		"postgres://solv_user:secreto@localhost:5433/solv_migration_test?sslmode=disable",
		"postgres://solv_user:secreto@127.0.0.1:55432/solv_e2e?sslmode=disable",
		"postgres://solv_user:secreto@[::1]:55432/solv_test?sslmode=disable",
	}
	for _, dsn := range valid {
		t.Run(dsn, func(t *testing.T) {
			if err := validateTestDSN(dsn); err != nil {
				t.Fatalf("DSN de tests rechazado: %v", err)
			}
		})
	}
}

func TestValidateTestDSNRejectsDeploymentAndAmbiguousTargets(t *testing.T) {
	invalid := []string{
		"",
		"postgres://solv_user:secreto@127.0.0.1:5432/solv_db?sslmode=disable",
		"postgres://solv_user:secreto@127.0.0.1:5432/solv_test?sslmode=disable",
		"postgres://solv_user:secreto@localhost:5432/solv_test?sslmode=disable",
		"postgres://solv_user:secreto@127.0.0.1:5432/contest?sslmode=disable",
		"postgres://solv_user:secreto@db.despliegue.example:5432/solv_test?sslmode=disable",
		"postgres://solv_user:secreto@127.0.0.1:5434/solv_test?sslmode=disable",
		"postgres://solv_user:secreto@127.0.0.1:5432/solv_test?dbname=solv_db",
		"postgres://solv_user:secreto@127.0.0.1:5433/solv_test?host=db.despliegue.example",
		"postgres://solv_user:secreto@127.0.0.1:5433/solv_test?user=otro",
		"postgres://solv_user:secreto@127.0.0.1:5433/solv_test?password=otra",
		"postgres://solv_user:secreto@127.0.0.1:5433/solv_test?service=produccion",
		"postgres://solv_user:secreto@127.0.0.1:5433/solv_test?passfile=/tmp/pgpass",
		"postgres://solv_user:secreto@127.0.0.1:5432/",
		"postgres://solv_user:secreto@127.0.0.1:5432/solv_test/otra",
		"mysql://solv_user:secreto@127.0.0.1:5432/solv_test",
		"postgres://solv_user@127.0.0.1:5433/solv_test",
		"postgres://:secreto@127.0.0.1:5433/solv_test",
	}
	for _, dsn := range invalid {
		t.Run(dsn, func(t *testing.T) {
			if err := validateTestDSN(dsn); err == nil {
				t.Fatalf("DSN no autorizado aceptado: %s", dsn)
			}
		})
	}
}

func TestGetTestDSNRequiresExplicitConfig(t *testing.T) {
	t.Setenv("TEST_DB_DSN", "")
	t.Setenv("TEST_DATABASE_URL", "")
	t.Setenv("DATABASE_URL", "postgres://solv_user:secreto@127.0.0.1:5432/solv_db")
	if got := getTestDSN(); got != "" {
		t.Fatalf("getTestDSN() = %q, se esperaba vacío sin variables explícitas", got)
	}
}

func TestDatabaseName(t *testing.T) {
	cases := map[string]string{
		"postgres://u:p@127.0.0.1:5432/solv_db?sslmode=disable": "solv_db",
		"postgres://u:p@127.0.0.1:5432/solv_test":               "solv_test",
		"sin-ruta": "",
	}
	for dsn, want := range cases {
		if got := databaseName(dsn); got != want {
			t.Fatalf("databaseName(%q) = %q, se esperaba %q", dsn, got, want)
		}
	}
}
