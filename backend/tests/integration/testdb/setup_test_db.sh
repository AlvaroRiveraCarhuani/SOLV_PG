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

# Crear la BD si no existe (usando el superusuario postgres o el usuario configurado)
PGPASSWORD="$DB_PASS" psql \
  -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" \
  -c "CREATE DATABASE $DB_NAME;" 2>/dev/null \
  || echo "Info: la base '$DB_NAME' ya existe, continuando."

echo "=== Aplicando migraciones con goose ==="

TEST_DSN="postgres://${DB_USER}:${DB_PASS}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=disable"

# Si goose está instalado como binario, usarlo directamente.
if command -v goose &>/dev/null; then
  GOOSE_DRIVER=postgres GOOSE_DBSTRING="$TEST_DSN" goose -dir "$MIGRATIONS_DIR" up
else
  # Fallback: compilar y correr el runner embebido del backend.
  echo "goose CLI no encontrado. Usando go run para aplicar migraciones..."
  TEST_DB_DSN="$TEST_DSN" \
  MIGRATIONS_DIR="$MIGRATIONS_DIR" \
  go run ./backend/cmd/api/main.go &
  PID=$!
  sleep 3
  kill $PID 2>/dev/null || true
fi

echo "=== solv_test lista para tests de integración ==="
echo "Usar: TEST_DB_DSN=\"$TEST_DSN\" go test ./backend/tests/integration/... -v -timeout 120s"
