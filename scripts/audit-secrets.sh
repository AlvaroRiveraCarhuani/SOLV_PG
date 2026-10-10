#!/bin/bash
set -euo pipefail

echo "=== Auditoría de Secretos y Configuración Faltante ==="

EXIT_CODE=0

# 1. Verificar que .env y *.dump están en .gitignore
echo "1. Verificando reglas de .gitignore..."
if ! grep -q "^\.env$" .gitignore 2>/dev/null && ! grep -q "\.env" .gitignore 2>/dev/null; then
  echo "FAIL: .env no está presente en .gitignore"
  EXIT_CODE=1
else
  echo "✓ Rule .env en .gitignore validada"
fi

if ! grep -q "\*\.dump" .gitignore 2>/dev/null && ! grep -q "\.dump" .gitignore 2>/dev/null; then
  echo "FAIL: *.dump no está presente en .gitignore"
  EXIT_CODE=1
else
  echo "✓ Rule *.dump en .gitignore validada"
fi

# 2. Escaneo de secretos en working tree
echo "2. Escaneando working tree por patrones de secretos..."

FOUND_SECRETS=0

# Patrones sensibles
PATTERNS=(
  "AKIA[0-9A-Z]{16}"
  "-----BEGIN[ A-Z0-9_-]*PRIVATE KEY-----"
  "jwt_secret[[:space:]]*=[[:space:]]*['\"][^'\"]{8,}['\"]"
)

for pattern in "${PATTERNS[@]}"; do
  MATCHES=$(git grep -E "$pattern" -- ':!scripts/audit-secrets.sh' ':!docs/' ':!*.md' 2>/dev/null || true)
  if [ -n "$MATCHES" ]; then
    echo "FAIL: Coincidencia de secreto encontrada:"
    echo "$MATCHES"
    FOUND_SECRETS=1
  fi
done

if [ "$FOUND_SECRETS" -eq 1 ]; then
  EXIT_CODE=1
else
  echo "✓ Sin secretos expuestos en working tree"
fi

if [ "$EXIT_CODE" -eq 0 ]; then
  echo "=== Auditoría de Secretos: PASÓ (VERDE) ==="
  exit 0
else
  echo "=== Auditoría de Secretos: FALLÓ (ROJO) ==="
  exit 1
fi
