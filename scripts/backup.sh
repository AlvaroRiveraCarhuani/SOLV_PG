#!/bin/bash
set -euo pipefail

BACKUP_DIR="${BACKUP_DIR:-/var/backups/solv}"
DATE=$(date +%Y%m%d_%H%M%S)
DB_NAME="${DB_NAME:-solv_db}"
DB_USER="${DB_USER:-solv_user}"
DB_HOST="${DB_HOST:-127.0.0.1}"
DB_PORT="${DB_PORT:-5432}"
FILES_DIR="${FILES_DIR:-/var/lib/solv/files}"
CONTAINER_NAME="${CONTAINER_NAME:-solv_db}"

mkdir -p "$BACKUP_DIR" 2>/dev/null || true

echo "Iniciando backup de base de datos $DB_NAME..."
if command -v pg_dump &>/dev/null; then
  if [ -z "${PGPASSWORD:-}" ] && [ -n "${DB_PASSWORD:-}" ]; then
    export PGPASSWORD="$DB_PASSWORD"
  fi
  pg_dump -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -Fc -f "$BACKUP_DIR/solv_db_$DATE.dump"
else
  docker exec "$CONTAINER_NAME" pg_dump -U "$DB_USER" -d "$DB_NAME" -Fc > "$BACKUP_DIR/solv_db_$DATE.dump"
fi

if [ -d "$FILES_DIR" ]; then
  echo "Iniciando backup de archivos adjuntos desde $FILES_DIR..."
  tar -czf "$BACKUP_DIR/solv_files_$DATE.tar.gz" -C "$(dirname "$FILES_DIR")" "$(basename "$FILES_DIR")"
fi

echo "Rotando backups antiguos (manteniendo ultimos 7 dias)..."
find "$BACKUP_DIR" -name "solv_db_*.dump" -mtime +7 -delete 2>/dev/null || true
find "$BACKUP_DIR" -name "solv_files_*.tar.gz" -mtime +7 -delete 2>/dev/null || true

if [ -n "${S3_BUCKET:-}" ]; then
  echo "Subiendo backup a s3://$S3_BUCKET/backups/..."
  aws s3 cp "$BACKUP_DIR/solv_db_$DATE.dump" "s3://$S3_BUCKET/backups/"
fi

echo "Backup completado: $BACKUP_DIR/solv_db_$DATE.dump"
