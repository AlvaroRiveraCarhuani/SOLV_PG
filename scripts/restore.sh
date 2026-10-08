#!/bin/bash
set -euo pipefail

if [ $# -eq 0 ]; then
  echo "Uso: $0 <archivo_backup.dump>"
  exit 1
fi

BACKUP_FILE="$1"
if [ ! -f "$BACKUP_FILE" ]; then
  echo "ERROR: El archivo de backup '$BACKUP_FILE' no existe."
  exit 1
fi

DB_USER="${DB_USER:-solv_user}"
DB_HOST="${DB_HOST:-127.0.0.1}"
DB_PORT="${DB_PORT:-5432}"
PROD_DB="${DB_NAME:-solv_db}"
TEMP_DB="solv_restore_test"
CONTAINER_NAME="${CONTAINER_NAME:-solv_db}"

run_createdb() {
  local target_db="$1"
  if command -v createdb &>/dev/null; then
    if [ -z "${PGPASSWORD:-}" ] && [ -n "${DB_PASSWORD:-}" ]; then
      export PGPASSWORD="$DB_PASSWORD"
    fi
    createdb -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" "$target_db"
  else
    docker exec "$CONTAINER_NAME" createdb -U "$DB_USER" "$target_db"
  fi
}

run_dropdb() {
  local target_db="$1"
  if command -v dropdb &>/dev/null; then
    if [ -z "${PGPASSWORD:-}" ] && [ -n "${DB_PASSWORD:-}" ]; then
      export PGPASSWORD="$DB_PASSWORD"
    fi
    dropdb -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" --if-exists "$target_db"
  else
    docker exec "$CONTAINER_NAME" dropdb -U "$DB_USER" --if-exists "$target_db"
  fi
}

run_pg_restore() {
  local target_db="$1"
  local file="$2"
  if command -v pg_restore &>/dev/null; then
    if [ -z "${PGPASSWORD:-}" ] && [ -n "${DB_PASSWORD:-}" ]; then
      export PGPASSWORD="$DB_PASSWORD"
    fi
    pg_restore -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$target_db" --no-owner --role="$DB_USER" "$file" || true
  else
    docker exec -i "$CONTAINER_NAME" pg_restore -U "$DB_USER" -d "$target_db" --no-owner --role="$DB_USER" < "$file" || true
  fi
}

run_psql_query() {
  local target_db="$1"
  local query="$2"
  if command -v psql &>/dev/null; then
    if [ -z "${PGPASSWORD:-}" ] && [ -n "${DB_PASSWORD:-}" ]; then
      export PGPASSWORD="$DB_PASSWORD"
    fi
    psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$target_db" -t -c "$query"
  else
    docker exec "$CONTAINER_NAME" psql -U "$DB_USER" -d "$target_db" -t -c "$query"
  fi
}

echo "Creando base de datos temporal '$TEMP_DB' para validacion de integridad..."
run_dropdb "$TEMP_DB"
run_createdb "$TEMP_DB"

echo "Restaurando backup en base temporal..."
run_pg_restore "$TEMP_DB" "$BACKUP_FILE"

echo "Ejecutando validacion de integridad..."
EXERCISE_COUNT=$(run_psql_query "$TEMP_DB" "SELECT COUNT(*) FROM exercises;" | tr -d '[:space:]')

if [ -z "$EXERCISE_COUNT" ] || [ "$EXERCISE_COUNT" -lt 1 ]; then
  echo "ERROR: Validacion fallo (ejercicios encontrados: ${EXERCISE_COUNT:-0})"
  run_dropdb "$TEMP_DB"
  exit 1
fi

echo "Validacion exitosa ($EXERCISE_COUNT ejercicios encontrados). Promoviendo a produccion ($PROD_DB)..."
run_dropdb "$PROD_DB"
run_createdb "$PROD_DB"
run_pg_restore "$PROD_DB" "$BACKUP_FILE"
run_dropdb "$TEMP_DB"

echo "Restore completado exitosamente"
