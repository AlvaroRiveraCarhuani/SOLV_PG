#!/bin/bash
set -euo pipefail

DB_NAME="${DB_NAME:-solv_db}"
DB_USER="${DB_USER:-solv_user}"
DB_HOST="${DB_HOST:-127.0.0.1}"
DB_PORT="${DB_PORT:-5432}"
CONTAINER_NAME="${CONTAINER_NAME:-solv_db}"

if command -v pg_dump &>/dev/null; then
  if [ -z "${PGPASSWORD:-}" ] && [ -n "${DB_PASSWORD:-}" ]; then
    export PGPASSWORD="$DB_PASSWORD"
  fi
  pg_dump -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" --clean --if-exists | \
    sed -E "s/@([a-zA-Z0-9.-]+\.[a-zA-Z]{2,})/@anonymized.test/g" | \
    sed -E "s/'[A-Z][a-z]+ [A-Z][a-z]+'/'Anonymized User'/g"
else
  docker exec "$CONTAINER_NAME" pg_dump -U "$DB_USER" -d "$DB_NAME" --clean --if-exists | \
    sed -E "s/@([a-zA-Z0-9.-]+\.[a-zA-Z]{2,})/@anonymized.test/g" | \
    sed -E "s/'[A-Z][a-z]+ [A-Z][a-z]+'/'Anonymized User'/g"
fi
