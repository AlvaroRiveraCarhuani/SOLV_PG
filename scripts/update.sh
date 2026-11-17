#!/usr/bin/env bash

# ==============================================================================
# SOLV - Actualización reproducible del stack Docker
# ==============================================================================
#
# Reconstruye las imágenes con los metadatos de build, recrea los contenedores
# y comprueba que lo que quedó corriendo sea exactamente el commit del árbol.
#
# Uso:
#   ./scripts/update.sh              # backend + frontend
#   ./scripts/update.sh backend      # solo backend
#   ./scripts/update.sh frontend     # solo frontend
#   ./scripts/update.sh --dry-run    # muestra el plan y el estado actual
#
# Por qué existe: `docker compose build` sin SOLV_VERSION, SOLV_COMMIT y
# SOLV_BUILD_DATE produce imágenes que publican commit "unknown" en
# GET /api/v1/version. Sin metadatos no se puede demostrar qué código corre.
#
# Además, compose.yml resuelve la imagen como solv-<servicio>:${SOLV_VERSION:-dev}.
# Tras construir con un tag versionado, el tag `dev` seguiría apuntando a la
# imagen anterior, y un `docker compose up -d` sin variables haría rollback
# silencioso. El script realinea `dev` al final, recién cuando la verificación
# pasó.
# ==============================================================================

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

BACKEND_PORT="${BACKEND_PORT:-3000}"
API_URL="http://127.0.0.1:${BACKEND_PORT}/api/v1/version"
VERIFY_TIMEOUT="${VERIFY_TIMEOUT:-90}"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

info() { echo -e "${CYAN}==>${NC} $*"; }
ok()   { echo -e "    ${GREEN}✔${NC} $*"; }
warn() { echo -e "    ${YELLOW}⚠${NC} $*"; }
fail() { echo -e "    ${RED}✗${NC} $*" >&2; }

usage() {
    sed -n '7,14p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//'
}

# ==============================================================================
# ARGUMENTOS
# ==============================================================================

DRY_RUN=0
SERVICES=()

for arg in "$@"; do
    case "$arg" in
        --dry-run|-n)      DRY_RUN=1 ;;
        backend|frontend)  SERVICES+=("$arg") ;;
        all)               ;;
        -h|--help)         usage; exit 0 ;;
        *)
            fail "Opción desconocida: $arg"
            echo ""
            usage
            exit 2
            ;;
    esac
done

if [ ${#SERVICES[@]} -eq 0 ]; then
    SERVICES=(backend frontend)
fi

# ==============================================================================
# PRECONDICIONES
# ==============================================================================

for bin in git docker curl; do
    command -v "$bin" >/dev/null 2>&1 || {
        fail "Falta el comando '$bin' en PATH."
        exit 1
    }
done

if ! docker compose version >/dev/null 2>&1; then
    fail "Se requiere Docker Compose v2 ('docker compose')."
    exit 1
fi

# ==============================================================================
# METADATOS DE BUILD
# ==============================================================================
# Son los mismos valores que exporta el Makefile y que consume compose.yml
# como build args. Viajan al binario por -ldflags -X y los publica
# GET /api/v1/version.

SOLV_VERSION="$(git describe --tags --always --dirty 2>/dev/null || echo dev)"
SOLV_COMMIT="$(git rev-parse --short HEAD 2>/dev/null || echo unknown)"
SOLV_BUILD_DATE="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
export SOLV_VERSION SOLV_COMMIT SOLV_BUILD_DATE

# ==============================================================================
# ESTADO ACTUAL
# ==============================================================================

current_api_commit() {
    local body
    body="$(curl -fsS --max-time 3 "$API_URL" 2>/dev/null || true)"
    [ -n "$body" ] || { echo "sin respuesta"; return 0; }
    printf '%s' "$body" | sed -n 's/.*"commit":"\([^"]*\)".*/\1/p' | head -n 1
}

echo ""
echo -e "${BOLD}${CYAN}====================================================${NC}"
echo -e "${BOLD}${CYAN}       ACTUALIZACIÓN DEL STACK SOLV                 ${NC}"
echo -e "${BOLD}${CYAN}====================================================${NC}"
echo ""
echo -e "  ${BOLD}Servicios:${NC}   ${SERVICES[*]}"
echo -e "  ${BOLD}Versión:${NC}     $SOLV_VERSION"
echo -e "  ${BOLD}Commit:${NC}      $SOLV_COMMIT"
echo -e "  ${BOLD}Build date:${NC}  $SOLV_BUILD_DATE"
echo -e "  ${BOLD}Corriendo:${NC}   $(current_api_commit)"
echo ""

if [ -n "$(git status --porcelain 2>/dev/null)" ]; then
    warn "Hay cambios sin commitear: la imagen quedará marcada como '-dirty'."
    warn "Un commit limpio es la única forma de que el hash sea verificable."
    echo ""
fi

if [ "$DRY_RUN" -eq 1 ]; then
    info "Modo dry-run: no se reconstruye ni se reinicia nada."
    echo ""
    echo "    docker compose build ${SERVICES[*]}"
    echo "    docker compose up -d ${SERVICES[*]}"
    echo "    verificación de salud, commit y tag 'dev'"
    echo ""
    exit 0
fi

# ==============================================================================
# BUILD Y RECREACIÓN
# ==============================================================================

info "Construyendo imágenes (${SERVICES[*]})..."
docker compose build "${SERVICES[@]}"
ok "Imágenes construidas con tag versionado."

info "Recreando contenedores..."
docker compose up -d "${SERVICES[@]}"
ok "Contenedores recreados."

# ==============================================================================
# VERIFICACIÓN
# ==============================================================================

wait_healthy() {
    local service="$1"
    local deadline=$((SECONDS + VERIFY_TIMEOUT))
    local container status

    container="$(docker compose ps -q "$service" 2>/dev/null || true)"
    if [ -z "$container" ]; then
        fail "$service no tiene contenedor activo."
        return 1
    fi

    echo -n "    Esperando salud de $service"
    while [ $SECONDS -lt $deadline ]; do
        status="$(docker inspect -f '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' "$container" 2>/dev/null || echo unknown)"
        case "$status" in
            healthy|running)
                echo -e " ${GREEN}✔ ${status}${NC}"
                return 0
                ;;
            unhealthy|exited|dead)
                echo -e " ${RED}✗ ${status}${NC}"
                docker logs --tail 25 "$container" >&2 || true
                return 1
                ;;
        esac
        echo -n "."
        sleep 2
    done

    echo -e " ${RED}✗ timeout (${VERIFY_TIMEOUT}s, último estado: ${status:-unknown})${NC}"
    docker logs --tail 25 "$container" >&2 || true
    return 1
}

verify_image_in_use() {
    local service="$1"
    local container built running

    container="$(docker compose ps -q "$service" 2>/dev/null || true)"
    built="$(docker image inspect -f '{{.Id}}' "solv-${service}:${SOLV_VERSION}" 2>/dev/null || true)"
    running="$(docker inspect -f '{{.Image}}' "$container" 2>/dev/null || true)"

    if [ -z "$built" ] || [ -z "$running" ]; then
        fail "$service: no se pudo comparar imagen construida con imagen en uso."
        return 1
    fi

    if [ "$built" != "$running" ]; then
        fail "$service corre una imagen distinta de la recién construida."
        fail "  construida: ${built:0:19}   en uso: ${running:0:19}"
        return 1
    fi

    ok "$service corre la imagen recién construida (${built:7:12})."
    return 0
}

verify_backend_commit() {
    local deadline=$((SECONDS + VERIFY_TIMEOUT))
    local body="" commit=""

    echo -n "    Esperando que la API reporte el commit $SOLV_COMMIT"
    while [ $SECONDS -lt $deadline ]; do
        body="$(curl -fsS --max-time 3 "$API_URL" 2>/dev/null || true)"
        commit="$(printf '%s' "$body" | sed -n 's/.*"commit":"\([^"]*\)".*/\1/p' | head -n 1)"
        if [ "$commit" = "$SOLV_COMMIT" ]; then
            echo -e " ${GREEN}✔${NC}"
            return 0
        fi
        echo -n "."
        sleep 2
    done

    echo -e " ${RED}✗${NC}"
    fail "GET /api/v1/version no reporta el commit esperado."
    fail "  esperado: $SOLV_COMMIT   reportado: ${commit:-sin respuesta}"
    fail "  respuesta: ${body:-vacía}"
    return 1
}

info "Verificando el resultado..."
echo ""

for service in "${SERVICES[@]}"; do
    wait_healthy "$service"
done

for service in "${SERVICES[@]}"; do
    verify_image_in_use "$service"
done

backend_selected=0
for service in "${SERVICES[@]}"; do
    [ "$service" = "backend" ] && backend_selected=1
done

if [ "$backend_selected" -eq 1 ]; then
    verify_backend_commit
    echo ""
    echo -e "    ${BOLD}Respuesta de /api/v1/version:${NC}"
    echo "    $(curl -fsS --max-time 5 "$API_URL" 2>/dev/null || echo '(sin respuesta)')"
fi

# ==============================================================================
# REALINEAR EL TAG dev
# ==============================================================================
# Sólo después de verificar. compose.yml cae en `dev` cuando SOLV_VERSION no
# está definida; si `dev` quedara en la imagen anterior, un `docker compose
# up -d` a secas devolvería el código viejo sin avisar.

echo ""
info "Realineando el tag 'dev' con las imágenes verificadas..."
for service in "${SERVICES[@]}"; do
    docker tag "solv-${service}:${SOLV_VERSION}" "solv-${service}:dev"
    ok "solv-${service}:dev -> ${SOLV_VERSION}"
done

echo ""
echo -e "${BOLD}${GREEN}✔ Stack actualizado y verificado en ${SOLV_COMMIT}${NC}"
echo -e "   • Frontend:  http://localhost:4200 (o el dominio configurado)"
echo -e "   • Backend:   http://localhost:${BACKEND_PORT}/api/v1/version"
echo -e "   • Logs:      docker compose logs -f ${SERVICES[*]}"
echo ""
