#!/usr/bin/env bash
# Crea una base PostgreSQL descartable, aplica las migraciones y ejecuta un
# comando con TEST_DB_DSN y DATABASE_URL apuntando a esa base.
#
# Uso:
#   tools/test-impact/test-db.sh -- go test -short ./tests/integration/...
#   tools/test-impact/test-db.sh --keep -- go test ./tests/integration/...
#
# Variables obligatorias (sin valores por defecto):
#   TEST_DB_HOST=127.0.0.1|localhost|::1
#   TEST_DB_PORT=5433|55432
#   TEST_DB_USER
#   TEST_DB_PASSWORD
# Opcional: TEST_DB_ADMIN_DB (postgres).
#
# No usa PGHOST/PGPORT/PGUSER/PGPASSWORD heredados, no usa DATABASE_URL y no
# cae al contenedor solv_dev_db. Valida el destino antes de crear la base.
set -euo pipefail

KEEP=0
if [[ "${1:-}" == "--keep" ]]; then
  KEEP=1
  shift
fi
if [[ "${1:-}" != "--" ]]; then
  echo "Uso: $0 [--keep] -- <comando...>" >&2
  exit 2
fi
shift

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

fail() {
  echo "test-db: $*" >&2
  exit 2
}

TEST_DB_HOST="${TEST_DB_HOST:-}"
TEST_DB_PORT="${TEST_DB_PORT:-}"
TEST_DB_USER="${TEST_DB_USER:-}"
TEST_DB_PASSWORD="${TEST_DB_PASSWORD:-}"
ADMIN_DB="${TEST_DB_ADMIN_DB:-postgres}"

case "$TEST_DB_HOST" in
  127.0.0.1|localhost|::1) ;;
  *) fail "TEST_DB_HOST debe ser loopback (127.0.0.1, localhost o ::1); recibido: ${TEST_DB_HOST:-<vacío>}" ;;
esac
case "$TEST_DB_PORT" in
  5433|55432) ;;
  *) fail "TEST_DB_PORT debe ser 5433 o 55432; el puerto 5432 del despliegue no se acepta" ;;
esac
[[ -n "$TEST_DB_USER" ]] || fail "falta TEST_DB_USER"
[[ -n "$TEST_DB_PASSWORD" ]] || fail "falta TEST_DB_PASSWORD"
[[ "$TEST_DB_USER" =~ ^[A-Za-z0-9_]+$ ]] || fail "TEST_DB_USER contiene caracteres no seguros para el DSN"
[[ "$TEST_DB_PASSWORD" =~ ^[A-Za-z0-9_.~-]+$ ]] || fail "TEST_DB_PASSWORD contiene caracteres que requieren codificación; use un secreto de test simple"
[[ "$ADMIN_DB" =~ ^[A-Za-z_][A-Za-z0-9_]*$ ]] || fail "TEST_DB_ADMIN_DB no es un identificador simple"
case "$ADMIN_DB" in
  solv_db|solv_dev_db) fail "TEST_DB_ADMIN_DB no puede ser una base de desarrollo o despliegue" ;;
esac

DB_NAME="solv_test_$(printf '%s' "${USER:-local}" | tr '[:upper:]' '[:lower:]' | tr -cd 'a-z0-9_')_$$"
DSN_HOST="$TEST_DB_HOST"
if [[ "$DSN_HOST" == "::1" ]]; then
  DSN_HOST="[::1]"
fi
DSN="postgres://${TEST_DB_USER}:${TEST_DB_PASSWORD}@${DSN_HOST}:${TEST_DB_PORT}/${DB_NAME}?sslmode=disable"

if ! command -v psql >/dev/null 2>&1; then
  fail "falta psql; no se usa docker exec sobre solv_dev_db como fallback"
fi

run_psql() {
  env -u PGHOST -u PGPORT -u PGUSER -u PGDATABASE -u PGSERVICE \
    -u PGSERVICEFILE -u PGOPTIONS -u PGPASSFILE \
    PGPASSWORD="$TEST_DB_PASSWORD" \
    psql -h "$TEST_DB_HOST" -p "$TEST_DB_PORT" -U "$TEST_DB_USER" "$@"
}

cleanup() {
  local status=$?
  if [[ "$KEEP" -eq 1 ]]; then
    echo "Base conservada para depuración: $DB_NAME"
    return "$status"
  fi
  if ! run_psql -d "$ADMIN_DB" -v ON_ERROR_STOP=1 \
    -c "DROP DATABASE IF EXISTS \"$DB_NAME\" WITH (FORCE);" >/dev/null; then
    echo "test-db: no se pudo eliminar la base descartable $DB_NAME" >&2
    if [[ "$status" -eq 0 ]]; then
      exit 1
    fi
  fi
  return "$status"
}
trap cleanup EXIT

echo "Creando base de tests descartable: $DB_NAME"
run_psql -d "$ADMIN_DB" -v ON_ERROR_STOP=1 -c "CREATE DATABASE \"$DB_NAME\";" >/dev/null

echo "Aplicando migraciones..."
(
  cd "$REPO_ROOT/backend"
  DATABASE_URL="$DSN" TEST_DB_DSN="$DSN" go run ./cmd/migrate -dir ./migrations -action=up
)

echo "Ejecutando: $*"
(
  cd "$REPO_ROOT/backend"
  DATABASE_URL="$DSN" TEST_DB_DSN="$DSN" TEST_DATABASE_URL="$DSN" SOLV_REQUIRE_INFRA=1 "$@"
)
