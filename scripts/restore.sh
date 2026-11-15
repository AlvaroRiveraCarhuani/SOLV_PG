#!/usr/bin/env bash
# Restauración de un respaldo de SOLV.
#
# Por defecto NO toca la base viva: restaura en una base temporal, cuenta las
# tablas principales y deja el respaldo listo para inspeccionar. Reemplazar la
# base en producción es una decisión explícita (--promote).
#
#   bash scripts/restore.sh backups/2026-11/solv_db_20261113_030000.sql.gz
#   bash scripts/restore.sh <archivo> --promote --yes
#   bash scripts/restore.sh <archivo> --expect-sha256 <hash>
#
# Un respaldo que nunca se restauró no es un respaldo: este script es la prueba.

set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CONTAINER="${SOLV_DB_CONTAINER:-solv_db}"
DB_NAME="${SOLV_DB_NAME:-solv_db}"
DB_USER="${SOLV_DB_USER:-solv_user}"
BACKEND_SERVICE="${SOLV_BACKEND_SERVICE:-backend}"
# Permite probar la promoción sin tocar la pila real (SOLV_COMPOSE_CMD=true).
COMPOSE_CMD="${SOLV_COMPOSE_CMD:-docker compose}"

FILE=""
PROMOTE=0
ASSUME_YES=0
KEEP_TEMP=0
EXPECT_SHA=""

log() { printf '%s\n' "$*"; }
step() { printf '\n== %s\n' "$*"; }
die() { printf 'ERROR: %s\n' "$*" >&2; exit 1; }

usage() {
  cat <<'USO'
Uso: scripts/restore.sh <respaldo.sql|.sql.gz> [opciones]

  --promote           Reemplaza la base viva por la restaurada (detiene el backend).
  --yes               No pide confirmación (obligatorio si no hay terminal).
  --keep              Conserva la base temporal aunque la verificación pase.
  --expect-sha256 H   Aborta si el hash del archivo no coincide.
  -h, --help          Muestra esta ayuda.

Variables: SOLV_DB_CONTAINER, SOLV_DB_NAME, SOLV_DB_USER, SOLV_BACKEND_SERVICE.
USO
}

while [ $# -gt 0 ]; do
  case "$1" in
    --promote) PROMOTE=1 ;;
    --yes) ASSUME_YES=1 ;;
    --keep) KEEP_TEMP=1 ;;
    --expect-sha256)
      [ $# -ge 2 ] || die "--expect-sha256 necesita un valor"
      EXPECT_SHA="$2"
      shift
      ;;
    -h | --help)
      usage
      exit 0
      ;;
    -*)
      die "opción desconocida: $1 (ver --help)"
      ;;
    *)
      [ -z "$FILE" ] || die "solo se acepta un archivo por ejecución"
      FILE="$1"
      ;;
  esac
  shift
done

[ -n "$FILE" ] || { usage; exit 1; }
[ -f "$FILE" ] || die "no existe el archivo ${FILE}"
[ -s "$FILE" ] || die "el archivo ${FILE} está vacío"

command -v docker >/dev/null 2>&1 || die "docker no está instalado"

confirm() {
  [ "$ASSUME_YES" -eq 1 ] && return 0
  [ -t 0 ] || die "sin terminal interactiva: agregá --yes para confirmar"
  printf '%s\nEscribí SI para continuar: ' "$1"
  read -r answer
  [ "$answer" = "SI" ] || die "cancelado"
}

# shellcheck disable=SC2086 # COMPOSE_CMD se expande en palabras a propósito.
compose() {
  (cd "$PROJECT_ROOT" && $COMPOSE_CMD "$@")
}

psql_db() {
  local db="$1"
  shift
  docker exec -i "$CONTAINER" psql -v ON_ERROR_STOP=1 -U "$DB_USER" -d "$db" "$@"
}

step "Verificando el respaldo"
ACTUAL_SHA="$(sha256sum "$FILE" | awk '{print $1}')"
log "  archivo: ${FILE}"
log "  sha256:  ${ACTUAL_SHA}"
if [ -n "$EXPECT_SHA" ]; then
  [ "$ACTUAL_SHA" = "$EXPECT_SHA" ] || die "el sha256 no coincide (esperado ${EXPECT_SHA})"
  log "  hash verificado"
fi
case "$FILE" in
  *.gz)
    gzip -t "$FILE" || die "el archivo gzip está corrupto"
    log "  gzip íntegro"
    ;;
esac

RUNNING="$(docker inspect -f '{{.State.Running}}' "$CONTAINER" 2>/dev/null || echo false)"
[ "$RUNNING" = "true" ] || die "el contenedor ${CONTAINER} no está corriendo"

TS="$(date +%Y%m%d_%H%M%S)"
TEMP_DB="solv_restore_${TS}"

step "Restaurando en la base temporal ${TEMP_DB}"
psql_db postgres -c "DROP DATABASE IF EXISTS \"${TEMP_DB}\"" >/dev/null
psql_db postgres -c "CREATE DATABASE \"${TEMP_DB}\"" >/dev/null
log "  base temporal creada"

if [ "${FILE##*.}" = "gz" ]; then
  gunzip -c "$FILE" | docker exec -i "$CONTAINER" psql -q -v ON_ERROR_STOP=1 -U "$DB_USER" -d "$TEMP_DB" >/dev/null
else
  docker exec -i "$CONTAINER" psql -q -v ON_ERROR_STOP=1 -U "$DB_USER" -d "$TEMP_DB" <"$FILE" >/dev/null
fi
log "  datos cargados"

step "Verificando el contenido"
FAILED=0
for table in tenants users exercises submissions; do
  if count="$(psql_db "$TEMP_DB" -tA -c "SELECT count(*) FROM ${table}" 2>/dev/null)"; then
    log "  ${table}: ${count}"
  else
    log "  ${table}: NO EXISTE"
    FAILED=1
  fi
done
[ "$FAILED" -eq 0 ] || die "la restauración no contiene las tablas esperadas; no se promueve nada"

if [ "$PROMOTE" -eq 0 ]; then
  step "Listo para revisar (la base viva no se tocó)"
  log "  inspeccionar: docker exec -it ${CONTAINER} psql -U ${DB_USER} -d ${TEMP_DB}"
  log "  promover:     bash scripts/restore.sh \"${FILE}\" --promote --yes"
  if [ "$KEEP_TEMP" -eq 0 ]; then
    log "  la base temporal ${TEMP_DB} queda disponible para revisión"
  fi
  exit 0
fi

confirm "Se va a reemplazar la base ${DB_NAME}. La actual se conserva como ${DB_NAME}_pre_restore_${TS}."

step "Promoviendo la restauración"
compose stop "$BACKEND_SERVICE" >/dev/null
log "  backend detenido"

psql_db postgres -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname IN ('${DB_NAME}', '${TEMP_DB}') AND pid <> pg_backend_pid()" >/dev/null
psql_db postgres -c "ALTER DATABASE \"${DB_NAME}\" RENAME TO \"${DB_NAME}_pre_restore_${TS}\"" >/dev/null
psql_db postgres -c "ALTER DATABASE \"${TEMP_DB}\" RENAME TO \"${DB_NAME}\"" >/dev/null
log "  ${DB_NAME} reemplazada; anterior en ${DB_NAME}_pre_restore_${TS}"

compose start "$BACKEND_SERVICE" >/dev/null
log "  backend iniciado"

step "Comprobando que el backend responde"
ok=0
for _ in $(seq 1 40); do
  if curl -fsS http://127.0.0.1:3000/api/v1/version >/dev/null 2>&1; then
    ok=1
    break
  fi
  sleep 1
done
if [ "$ok" -eq 1 ]; then
  log "  /api/v1/version responde"
else
  log "  el backend no respondió todavía: revisá 'docker compose logs ${BACKEND_SERVICE}'"
fi

step "Resumen"
log "  base viva:        ${DB_NAME}"
log "  base anterior:    ${DB_NAME}_pre_restore_${TS}"
log "  Cuando la operación esté confirmada, eliminá la anterior con:"
log "    docker exec -it ${CONTAINER} psql -U ${DB_USER} -d postgres -c 'DROP DATABASE \"${DB_NAME}_pre_restore_${TS}\"'"
