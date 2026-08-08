#!/usr/bin/env bash
set -euo pipefail

# Script idempotente para aprovisionamiento de registros DNS en desec.io
# Setea:
# 1. solv.dedyn.io A -> IP pública obtenida con ifconfig.me
# 2. *.solv.dedyn.io CNAME -> solv.dedyn.io.

if [ -f .env ]; then
  set -a
  source .env
  set +a
fi

DESEC_TOKEN="${DESEC_TOKEN:-}"
DESEC_DOMAIN="${DESEC_DOMAIN:-solv.dedyn.io}"

if [ -z "$DESEC_TOKEN" ]; then
  echo "Error: DESEC_TOKEN no está definido en el entorno o .env" >&2
  exit 1
fi

PUBLIC_IP=$(curl -s4 ifconfig.me || curl -s4 icanhazip.com || true)
if [ -z "$PUBLIC_IP" ]; then
  echo "Error: No se pudo obtener la IP pública del servidor" >&2
  exit 1
fi

echo "IP pública detectada: ${PUBLIC_IP}"
echo "Configurando registros DNS en desec.io para el dominio: ${DESEC_DOMAIN}..."

# 1. Apex A record
echo "Actualizando registro A (apex)..."
curl -s -X PUT "https://desec.io/api/v1/domains/${DESEC_DOMAIN}/rrsets/@/A/" \
  -H "Authorization: Token ${DESEC_TOKEN}" \
  -H "Content-Type: application/json" \
  -d "{
    \"subname\": \"\",
    \"type\": \"A\",
    \"records\": [\"${PUBLIC_IP}\"],
    \"ttl\": 3600
  }" > /dev/null

# 2. Wildcard CNAME record
echo "Actualizando registro CNAME (*)..."
curl -s -X PUT "https://desec.io/api/v1/domains/${DESEC_DOMAIN}/rrsets/\*/CNAME/" \
  -H "Authorization: Token ${DESEC_TOKEN}" \
  -H "Content-Type: application/json" \
  -d "{
    \"subname\": \"*\",
    \"type\": \"CNAME\",
    \"records\": [\"${DESEC_DOMAIN}.\"],
    \"ttl\": 3600
  }" > /dev/null

echo "Registros DNS configurados exitosamente en desec.io."
