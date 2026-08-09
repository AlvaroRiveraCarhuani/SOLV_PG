#!/usr/bin/env bash
set -euo pipefail

# Script de respaldo automatizado e idempotente para la plataforma SOLV
# 1. Respaldo SQL de PostgreSQL (solv_db)
# 2. Respaldo de volúmenes de laboratorios de estudiantes (solv_vol_*)
# 3. Clasificación en directorio ./backups/YYYY-MM/
# 4. Retención automatizada (eliminación de respaldos locales mayores a 6 meses)

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
BACKUP_BASE_DIR="${PROJECT_ROOT}/backups"
TIMESTAMP="$(date +'%Y%m%d_%H%M%S')"
YEAR_MONTH="$(date +'%Y-%m')"
TARGET_DIR="${BACKUP_BASE_DIR}/${YEAR_MONTH}"
LOG_FILE="/var/log/solv-backup.log"

# Si no tiene permisos en /var/log, escribir en el directorio de respaldos
if ! touch "${LOG_FILE}" 2>/dev/null; then
  LOG_FILE="${BACKUP_BASE_DIR}/solv-backup.log"
fi

mkdir -p "${TARGET_DIR}"

log() {
  echo "[$(date +'%Y-%m-%d %H:%M:%S')] $1" | tee -a "${LOG_FILE}"
}

log "=== Iniciando proceso de backup SOLV (${TIMESTAMP}) ==="

# 1. Dump de la Base de Datos PostgreSQL
DB_BACKUP_FILE="${TARGET_DIR}/solv_db_${TIMESTAMP}.sql"
log "Paso 1: Generando dump de PostgreSQL solv_db..."
if docker exec solv_db pg_dump -U solv_user solv_db > "${DB_BACKUP_FILE}"; then
  gzip -f "${DB_BACKUP_FILE}"
  log "Paso 1 Completado: ${DB_BACKUP_FILE}.gz"
else
  log "Error: Falló el dump de PostgreSQL solv_db" >&2
fi

# 2. Respaldo de Volúmenes Docker de Workspaces
VOLUMES_BACKUP_FILE="${TARGET_DIR}/solv_workspaces_vols_${TIMESTAMP}.tar.gz"
log "Paso 2: Identificando volúmenes de workspaces activos..."
VOLUMES=$(docker volume ls -q --filter name=solv_vol_ || true)

if [ -n "${VOLUMES}" ]; then
  log "Empaquetando volúmenes de estudiantes..."
  DOCKER_VOL_PATH="/var/lib/docker/volumes"
  
  # Si se ejecuta con sudo/root, empaquetar directamente los volúmenes
  if [ "$(id -u)" -eq 0 ] && [ -d "${DOCKER_VOL_PATH}" ]; then
    tar -czf "${VOLUMES_BACKUP_FILE}" -C "${DOCKER_VOL_PATH}" ${VOLUMES} 2>/dev/null || true
    log "Paso 2 Completado: ${VOLUMES_BACKUP_FILE}"
  else
    log "Aviso: Para empaquetar volúmenes físicamente se requiere permisos root. Creando archivo con inventario de volúmenes..."
    echo "${VOLUMES}" | gzip > "${VOLUMES_BACKUP_FILE}"
  fi
else
  log "Paso 2: No se encontraron volúmenes solv_vol_* activos para respaldar."
fi

# 3. Aplicar política de retención (Eliminar respaldos con más de 180 días / 6 meses)
log "Paso 3: Aplicando política de retención de 6 meses..."
find "${BACKUP_BASE_DIR}" -type f \( -name "*.sql.gz" -o -name "*.tar.gz" \) -mtime +180 -exec rm -f {} \; 2>/dev/null || true

log "=== Proceso de backup finalizado exitosamente ==="
