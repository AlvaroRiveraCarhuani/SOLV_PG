#!/bin/bash
set -euo pipefail

DB_USER="${DB_USER:-solv_user}"
DB_PASSWORD="${DB_PASSWORD:-solv_password}"
DB_HOST="${DB_HOST:-127.0.0.1}"
DB_PORT="${DB_PORT:-5432}"
TEST_DB="solv_migration_test"
CONTAINER_NAME="${CONTAINER_NAME:-solv_db}"

if [ -z "${PGPASSWORD:-}" ]; then
  export PGPASSWORD="$DB_PASSWORD"
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

run_createdb() {
  local target_db="$1"
  if command -v createdb &>/dev/null; then
    createdb -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" "$target_db"
  else
    docker exec "$CONTAINER_NAME" createdb -U "$DB_USER" "$target_db"
  fi
}

run_dropdb() {
  local target_db="$1"
  if command -v dropdb &>/dev/null; then
    dropdb -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" --if-exists "$target_db"
  else
    docker exec "$CONTAINER_NAME" dropdb -U "$DB_USER" --if-exists "$target_db"
  fi
}

run_psql_stdin() {
  local target_db="$1"
  if command -v psql &>/dev/null; then
    psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$target_db"
  else
    docker exec -i "$CONTAINER_NAME" psql -U "$DB_USER" -d "$target_db"
  fi
}

run_psql_query() {
  local target_db="$1"
  local query="$2"
  if command -v psql &>/dev/null; then
    psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$target_db" -t -c "$query"
  else
    docker exec "$CONTAINER_NAME" psql -U "$DB_USER" -d "$target_db" -t -c "$query"
  fi
}

echo "Preparando base de datos de prueba para drill de rollback: $TEST_DB..."
run_dropdb "$TEST_DB"
run_createdb "$TEST_DB"

echo "Aplicando dump anonimizado..."
"$SCRIPT_DIR/anonymize-dump.sh" | run_psql_stdin "$TEST_DB" > /dev/null

TEST_DSN="postgres://$DB_USER:$DB_PASSWORD@$DB_HOST:$DB_PORT/$TEST_DB?sslmode=disable"

echo "Aplicando migraciones hacia adelante (up)..."
cd "$PROJECT_ROOT/backend"
go run ./cmd/migrate -dir ./migrations -dsn "$TEST_DSN" -action up

echo "Revirtiendo migraciones hacia atras (down)..."
go run ./cmd/migrate -dir ./migrations -dsn "$TEST_DSN" -action down

echo "Validando limpieza post-rollback..."
ORPHAN_TABLES=$(run_psql_query "$TEST_DB" "
  SELECT COUNT(*) FROM information_schema.tables 
  WHERE table_schema = 'public' 
  AND table_name NOT IN ('schema_migrations', 'goose_db_version');
" | tr -d '[:space:]')

if [ -z "$ORPHAN_TABLES" ]; then
  ORPHAN_TABLES=0
fi

if [ "$ORPHAN_TABLES" -gt 0 ]; then
  echo "ERROR: Quedaron $ORPHAN_TABLES tablas huerfanas despues del rollback"
  run_dropdb "$TEST_DB"
  exit 1
fi

run_dropdb "$TEST_DB"
echo "Drill de rollback exitoso: todas las migraciones son reversibles"
