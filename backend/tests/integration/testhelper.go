package integration

import (
	"fmt"
	"os"
)

// getTestDSN devuelve el DSN de la base de datos de tests de integración.
// Prioridad:
//  1. Variable de entorno TEST_DB_DSN (permite override en CI).
//  2. Variable de entorno DATABASE_URL (convención de la plataforma), con
//     reemplazo del nombre de base a "solv_test" para aislar los tests.
//  3. DSN por defecto apuntando a solv_test en localhost.
//
// NUNCA conectar tests a solv_db directamente: contamina el dashboard de producción/dev.
func getTestDSN() string {
	if v := os.Getenv("TEST_DB_DSN"); v != "" {
		return v
	}
	if v := os.Getenv("TEST_DATABASE_URL"); v != "" {
		return v
	}
	// Si hay DATABASE_URL, reemplazar el nombre de BD para apuntar a solv_test.
	// Esto funciona con DSNs del tipo postgres://user:pass@host:port/solv_db?...
	if v := os.Getenv("DATABASE_URL"); v != "" {
		return replaceDatabaseName(v, "solv_test")
	}
	return "postgres://solv_user:solv_password@127.0.0.1:5432/solv_test?sslmode=disable"
}

// replaceDatabaseName reemplaza el nombre de base de datos en un DSN de PostgreSQL.
// Soporta el formato: postgres://user:pass@host:port/dbname?params
func replaceDatabaseName(dsn, newDB string) string {
	// Buscar el último '/' antes del '?' (o fin de string)
	slashIdx := -1
	for i := len(dsn) - 1; i >= 0; i-- {
		if dsn[i] == '?' {
			// ignorar la query string
			continue
		}
		if dsn[i] == '/' {
			slashIdx = i
			break
		}
	}
	if slashIdx < 0 {
		return dsn
	}
	queryStart := len(dsn)
	for i := slashIdx + 1; i < len(dsn); i++ {
		if dsn[i] == '?' {
			queryStart = i
			break
		}
	}
	return fmt.Sprintf("%s/%s%s", dsn[:slashIdx], newDB, dsn[queryStart:])
}
