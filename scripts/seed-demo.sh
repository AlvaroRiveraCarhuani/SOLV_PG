#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

if [ -f "${ROOT_DIR}/.env" ]; then
    # shellcheck disable=SC1091
    set -a
    . "${ROOT_DIR}/.env"
    set +a
fi

echo "==> [1/4] Limpiando contenedores y volúmenes huérfanos de pruebas..."
# Contenedores efímeros de evaluación, escáneres Trivy y laboratorios previos
orphan_containers=$(docker ps -aq \
    --filter "ancestor=python:3.11-slim" \
    --filter "ancestor=aquasec/trivy:0.55.0" \
    --filter "name=solv-environment-" \
    --filter "name=solv-review-env-" 2>/dev/null || true)

# Incluye contenedores por digest de python:3.11-slim que puedan no coincidir con --filter ancestor
extra_eval_containers=$(docker ps -a --format '{{.ID}} {{.Image}} {{.Names}}' | awk '$2 ~ /^python:3\.11-slim/ || $2 ~ /^aquasec\/trivy/ || $3 ~ /^solv-environment-/ || $3 ~ /^solv-review-env-/ {print $1}')

all_orphans=$(printf "%s\n%s\n" "${orphan_containers}" "${extra_eval_containers}" | awk 'NF' | sort -u)
if [ -n "${all_orphans}" ]; then
    echo "${all_orphans}" | xargs -r docker rm -f >/dev/null
fi

# Limpia únicamente volúmenes de workspaces/reviews previos (nunca pg_data, seaweed_data, solv_backups ni cachés)
orphan_volumes=$(docker volume ls -q | grep -E '^(solv_workspace_|solv-review-env-)' || true)
if [ -n "${orphan_volumes}" ]; then
    echo "${orphan_volumes}" | xargs -r docker volume rm >/dev/null || true
fi

mkdir -p /tmp/solv-eval

echo "==> [2/4] Verificando disponibilidad de solv_db..."
if ! docker exec solv_db pg_isready -U solv_user -d solv_db >/dev/null 2>&1; then
    echo "Error: el contenedor solv_db no está listo." >&2
    exit 1
fi

echo "==> [3/4] Aplicando seed idempotente backend/seeds/demo_defensa.sql..."
docker exec -i solv_db psql -U solv_user -d solv_db -v ON_ERROR_STOP=1 < "${ROOT_DIR}/backend/seeds/demo_defensa.sql"

echo "==> [4/4] Generando accesos de sesión para los 3 roles UAB..."
ADMIN_TOKEN=$(cd "${ROOT_DIR}/backend" && go run ./cmd/gentoken/main.go -role=admin -email="roberto.paredes@uab.edu.bo" -days=7 -raw)
TEACHER_TOKEN=$(cd "${ROOT_DIR}/backend" && go run ./cmd/gentoken/main.go -role=teacher -email="carlos.mendoza@uab.edu.bo" -days=7 -raw)
STUDENT_TOKEN=$(cd "${ROOT_DIR}/backend" && go run ./cmd/gentoken/main.go -role=student -email="valeria.flores@uab.edu.bo" -days=7 -raw)

echo ""
echo "=================================================================="
echo " Entorno de defensa UAB listo"
echo "=================================================================="
echo "1) Admin      (Ing. Roberto Paredes · roberto.paredes@uab.edu.bo)"
echo "   document.cookie = \"solv_session=${ADMIN_TOKEN}; path=/\"; location.reload();"
echo ""
echo "2) Docente    (Ing. Carlos Mendoza Ríos · carlos.mendoza@uab.edu.bo)"
echo "   document.cookie = \"solv_session=${TEACHER_TOKEN}; path=/\"; location.reload();"
echo ""
echo "3) Estudiante (Valeria Flores Mamani · valeria.flores@uab.edu.bo)"
echo "   document.cookie = \"solv_session=${STUDENT_TOKEN}; path=/\"; location.reload();"
echo "=================================================================="
