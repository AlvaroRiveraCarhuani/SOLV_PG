package integration

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
)

// withTestIdentityFromHeaders es un fixture de integración: convierte los
// headers que el test usa como entrada explícita en el contexto verificado que
// produciría WithAuth/WithTenant. El código de producción no lee identidad de
// headers; este helper vive solo en el paquete de tests.
func withTestIdentityFromHeaders(r *http.Request) *http.Request {
	tenantID := strings.TrimSpace(r.Header.Get("X-Test-Tenant-Id"))
	if tenantID == "" {
		tenantID = domain.DefaultTenantID
	}
	return withTestIdentity(
		r,
		tenantID,
		strings.TrimSpace(r.Header.Get("X-Test-User-Id")),
		strings.TrimSpace(r.Header.Get("X-Test-User-Role")),
	)
}

func withTestIdentity(r *http.Request, tenantID, userID, role string) *http.Request {
	ctx := r.Context()
	if tenantID != "" {
		ctx = context.WithValue(ctx, domain.TenantIDKey, tenantID)
	}
	if userID != "" {
		ctx = context.WithValue(ctx, domain.UserIDKey, userID)
	}
	if role != "" {
		ctx = context.WithValue(ctx, domain.UserRoleKey, role)
	}
	return r.WithContext(ctx)
}

// getTestDSN devuelve el DSN de la base de datos de tests de integración.
// Solo acepta variables explícitas de tests; no existe un DSN predeterminado y
// nunca reutiliza DATABASE_URL del despliegue. TestMain valida host, puerto,
// nombre y credenciales antes de exportar el DSN a las pruebas que leen
// DATABASE_URL directamente.
func getTestDSN() string {
	if v := strings.TrimSpace(os.Getenv("TEST_DB_DSN")); v != "" {
		return v
	}
	if v := strings.TrimSpace(os.Getenv("TEST_DATABASE_URL")); v != "" {
		return v
	}
	return ""
}

// validateTestDSN rechaza DSN ambiguos y destinos que no sean de tests. La
// allowlist es cerrada: solo loopback, puertos de test y nombres descartables.
func validateTestDSN(dsn string) error {
	if strings.TrimSpace(dsn) == "" {
		return fmt.Errorf("falta TEST_DB_DSN o TEST_DATABASE_URL; no hay DSN de tests predeterminado ni se usa DATABASE_URL")
	}
	parsed, err := url.Parse(dsn)
	if err != nil {
		return fmt.Errorf("DSN de tests inválido: %w", err)
	}
	if parsed.Scheme != "postgres" && parsed.Scheme != "postgresql" {
		return fmt.Errorf("el DSN de tests debe usar postgres:// o postgresql://, no %q", parsed.Scheme)
	}
	host := parsed.Hostname()
	if host == "" {
		return fmt.Errorf("el DSN de tests no declara host")
	}
	port := parsed.Port()
	if port == "" {
		port = "5432"
	}
	name, err := url.PathUnescape(strings.TrimPrefix(parsed.Path, "/"))
	if err != nil {
		return fmt.Errorf("el nombre de base del DSN de tests no se puede decodificar: %w", err)
	}
	if name == "" || strings.Contains(name, "/") {
		return fmt.Errorf("el DSN de tests debe declarar un nombre de base simple")
	}

	// Un parámetro dbname/host/port/user/password/service/passfile puede
	// contradecir la URL y hacer que el
	// destino real no coincida con lo validado.
	for key := range parsed.Query() {
		switch strings.ToLower(key) {
		case "dbname", "host", "port", "user", "password", "service", "passfile":
			return fmt.Errorf("el DSN de tests no puede sobrescribir %q en la query", key)
		}
	}

	if !containsValue([]string{"127.0.0.1", "localhost", "::1"}, host) {
		return fmt.Errorf("host de base de tests no permitido: %q", host)
	}
	if !containsValue([]string{"5433", "55432"}, port) {
		return fmt.Errorf("puerto de base de tests no permitido: %q", port)
	}
	allowedName, err := regexp.Compile(`^solv_(test|migration_test|e2e)(_[a-z0-9_]+)?$`)
	if err != nil {
		return fmt.Errorf("patrón de nombre de base de tests inválido: %w", err)
	}
	if !allowedName.MatchString(name) {
		return fmt.Errorf("nombre de base de tests no permitido: %q", name)
	}
	if name == "solv_db" || name == "solv_dev_db" {
		return fmt.Errorf("nombre de base de tests no permitido: %q", name)
	}
	if parsed.User == nil || strings.TrimSpace(parsed.User.Username()) == "" {
		return fmt.Errorf("el DSN de tests debe declarar un usuario explícito")
	}
	if password, ok := parsed.User.Password(); !ok || password == "" {
		return fmt.Errorf("el DSN de tests debe declarar una contraseña explícita")
	}
	return nil
}

func containsValue(values []string, wanted string) bool {
	for _, value := range values {
		if strings.EqualFold(value, wanted) {
			return true
		}
	}
	return false
}

// requireInfra obliga a fallar en CI y en suites que declaran infraestructura
// obligatoria; en desarrollo local permite el Skip explicativo.
func requireInfra() bool {
	return os.Getenv("SOLV_REQUIRE_INFRA") == "1" || os.Getenv("SOLV_REQUIRE_DOCKER") == "1"
}

func skipOrFailInfra(t *testing.T, format string, args ...any) {
	t.Helper()
	if requireInfra() {
		t.Fatalf(format, args...)
	}
	t.Skipf(format, args...)
}

// requireDocker omite las pruebas que arrancan contenedores reales cuando se
// ejecuta con -short. En la suite Docker obligatoria, un skip desactivado por
// variable de entorno se convierte en fallo.
func requireDocker(t *testing.T) {
	t.Helper()
	if testing.Short() {
		if os.Getenv("SOLV_REQUIRE_DOCKER") == "1" {
			t.Fatal("la suite Docker obligatoria no puede ejecutarse con -short")
		}
		t.Skip("prueba con Docker omitida con -short")
	}
	if os.Getenv("SOLV_SKIP_DOCKER_TESTS") == "1" {
		if requireInfra() {
			t.Fatal("SOLV_SKIP_DOCKER_TESTS no puede omitir pruebas en una suite obligatoria")
		}
		t.Skip("prueba con Docker omitida por SOLV_SKIP_DOCKER_TESTS")
	}
	ensureTestRunID()
}

// ensureTestRunID da una identidad a los recursos Docker que crea una prueba
// ejecutada fuera del runner. Sin identidad no se puede hacer rollback ni
// limpieza acotada.
func ensureTestRunID() string {
	if runID := strings.TrimSpace(os.Getenv("SOLV_TEST_RUN_ID")); runID != "" {
		return runID
	}
	runID := fmt.Sprintf("solv-local-%d-%d", os.Getpid(), time.Now().UnixNano())
	_ = os.Setenv("SOLV_TEST_RUN_ID", runID)
	return runID
}

// databaseName extrae el nombre de base de un DSN de PostgreSQL. Se conserva
// para diagnósticos y pruebas de compatibilidad; la validación usa net/url.
func databaseName(dsn string) string {
	start := -1
	end := len(dsn)
	for i := len(dsn) - 1; i >= 0; i-- {
		if dsn[i] == '?' {
			end = i
			continue
		}
		if dsn[i] == '/' {
			start = i + 1
			break
		}
	}
	if start < 0 || start >= end {
		return ""
	}
	return dsn[start:end]
}
