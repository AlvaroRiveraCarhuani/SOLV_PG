#!/usr/bin/env bash
# Verifica el resultado de los jobs de CI. Un job seleccionado por el plan debe
# terminar en success; uno no seleccionado debe quedar skipped. Un output
# incompleto también falla: no se interpreta como "no requerido".
set -euo pipefail

IMPACT_RESULT="${IMPACT_RESULT:-}"
LINT_RESULT="${LINT_RESULT:-}"
E2E_EXPECTED="${E2E_EXPECTED:-${E2E_SELECTED:-}}"
IMPACT_MODE="${IMPACT_MODE:-desconocido}"
IMPACT_REASON="${IMPACT_REASON:-sin plan}"

echo "Modo de impacto: ${IMPACT_MODE}"
echo "Nivel de impacto: ${IMPACT_LEVEL:-desconocido}"
echo "Motivo: ${IMPACT_REASON}"
echo "impact=${IMPACT_RESULT} lint=${LINT_RESULT} assets=${ASSET_RESULT:-} e2e=${E2E_RESULT:-} esperado=${E2E_EXPECTED}"

if [[ "${LINT_ADVISORY:-false}" == "true" ]]; then
  echo "::warning::golangci-lint o ESLint reportaron hallazgos advisory; no son gates obligatorios."
fi

if [[ "$IMPACT_RESULT" != "success" || "$LINT_RESULT" != "success" ]]; then
  echo "::error::El plan de impacto o los gates no terminaron en success." >&2
  exit 1
fi

for value in \
  "${UNIT_EXPECTED:-}" "${INTEGRATION_EXPECTED:-}" "${DOCKER_EXPECTED:-}" \
  "${FRONTEND_EXPECTED:-}" "${MIGRATION_EXPECTED:-}" "${BUILD_EXPECTED:-}" \
  "${SECURITY_EXPECTED:-}" "$E2E_EXPECTED"; do
  if [[ "$value" != "true" && "$value" != "false" ]]; then
    echo "::error::El plan no publicó un valor booleano completo para todos los jobs." >&2
    exit 1
  fi
done

if [[ "${ASSET_EXPECTED:-}" != "true" && "${ASSET_EXPECTED:-}" != "false" ]]; then
  echo "::error::El plan no publicó asset_validation." >&2
  exit 1
fi

for entry in \
  "unit:${UNIT_RESULT:-}:$UNIT_EXPECTED" \
  "assets:${ASSET_RESULT:-}:$ASSET_EXPECTED" \
  "integration:${INTEGRATION_RESULT:-}:$INTEGRATION_EXPECTED" \
  "docker:${DOCKER_RESULT:-}:$DOCKER_EXPECTED" \
  "frontend:${FRONTEND_RESULT:-}:$FRONTEND_EXPECTED" \
  "e2e:${E2E_RESULT:-}:$E2E_EXPECTED" \
  "migrations:${MIGRATION_RESULT:-}:$MIGRATION_EXPECTED" \
  "build:${BUILD_RESULT:-}:$BUILD_EXPECTED" \
  "security:${SECURITY_RESULT:-}:$SECURITY_EXPECTED"; do
  IFS=: read -r name result expected <<< "$entry"
  if [[ "$expected" == "true" ]]; then
    if [[ "$result" != "success" ]]; then
      echo "::error::El job obligatorio $name terminó en '$result' y no en success." >&2
      exit 1
    fi
  elif [[ "$result" != "skipped" ]]; then
    echo "::error::El job $name no estaba seleccionado pero terminó en '$result'." >&2
    exit 1
  fi
done

echo "Pipeline en verde para el plan aplicado."