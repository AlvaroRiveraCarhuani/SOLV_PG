#!/usr/bin/env bash
# setup_test_db.sh — crea la base de datos solv_test y aplica las migraciones
# Ejecutar UNA VEZ antes de correr los tests de integración.
#
# Uso:
#   bash backend/tests/integration/testdb/setup_test_db.sh
#   TEST_DB_DSN="postgres://user:pass@host:5432/solv_test?sslmode=disable" \
#     bash backend/tests/integration/testdb/setup_test_db.sh

set -euo pipefail

DB_HOST="${PGHOST:-127.0.0.1}"
DB_PORT="${PGPORT:-5432}"
DB_USER="${PGUSER:-solv_user}"
DB_PASS="${PGPASSWORD:-solv_password}"
DB_NAME="solv_test"
MIGRATIONS_DIR="${MIGRATIONS_DIR:-backend/migrations}"

echo "=== Creando base de datos de tests: $DB_NAME ==="

run_psql() {
  if command -v psql &>/dev/null; then
    PGPASSWORD="$DB_PASS" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" "$@"
  elif docker ps --format '{{.Names}}' | grep -q "^solv_db$"; then
    docker exec -i solv_db psql -U "$DB_USER" "$@"
  else
    echo "Error: no se encontró psql ni el contenedor docker 'solv_db'." >&2
    exit 1
  fi
}

# Crear la BD si no existe (conectando a solv_db para emitir el CREATE)
run_psql -d "${DEFAULT_DB:-solv_db}" -tc "SELECT 1 FROM pg_database WHERE datname = '$DB_NAME'" | grep -q 1 \
  || run_psql -d "${DEFAULT_DB:-solv_db}" -c "CREATE DATABASE $DB_NAME;"

echo "=== Aplicando migraciones con goose ==="

TEST_DSN="postgres://${DB_USER}:${DB_PASS}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=disable"

# Si goose está instalado como binario, usarlo directamente.
if command -v goose &>/dev/null; then
  GOOSE_DRIVER=postgres GOOSE_DBSTRING="$TEST_DSN" goose -dir "$MIGRATIONS_DIR" up
else
  # Fallback: usar el runner de migraciones del backend.
  echo "goose CLI no encontrado. Usando cmd/migrate para aplicar migraciones..."
  (
    cd "$(git rev-parse --show-toplevel 2>/dev/null || pwd)/backend"
    DATABASE_URL="$TEST_DSN" go run ./cmd/migrate -dir ./migrations
  )
fi

echo "=== solv_test lista para tests de integración ==="
echo "Usar: TEST_DB_DSN=\"$TEST_DSN\" go test ./backend/tests/integration/... -v -timeout 120s"
