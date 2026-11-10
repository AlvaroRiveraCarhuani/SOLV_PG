#!/usr/bin/env bash
# Elimina solo los contenedores, volúmenes y redes del run de pruebas indicado.
# Con --stale retira únicamente recursos de runs anteriores que superen la
# antigüedad configurada; nunca borra por patrón de nombre.
# Nunca usa prune ni toca recursos sin la etiqueta exacta solv.test.run.
set -euo pipefail

STALE=0
if [[ "${1:-}" == "--stale" ]]; then
  STALE=1
fi

RUN_ID="${SOLV_TEST_RUN_ID:-}"
if [[ -z "$RUN_ID" ]]; then
  echo "cleanup-docker: falta SOLV_TEST_RUN_ID; se niega a limpiar sin un run acotado." >&2
  exit 2
fi
if [[ ! "$RUN_ID" =~ ^[A-Za-z0-9][A-Za-z0-9_.-]{0,62}$ ]]; then
  echo "cleanup-docker: SOLV_TEST_RUN_ID inválido: $RUN_ID" >&2
  exit 2
fi

if ! docker info >/dev/null 2>&1; then
  echo "cleanup-docker: Docker no está disponible; no se puede verificar la limpieza." >&2
  exit 1
fi

FILTER="label=solv.test.run=$RUN_ID"

if [[ "$STALE" -eq 1 ]]; then
  STALE_SECONDS="${SOLV_TEST_STALE_AFTER_SECONDS:-7200}"
  [[ "$STALE_SECONDS" =~ ^[0-9]+$ ]] || { echo "cleanup-docker: SOLV_TEST_STALE_AFTER_SECONDS inválido" >&2; exit 2; }
  now="$(date +%s)"
  is_stale() {
    local started="$1"
    [[ -z "$started" ]] && return 0
    local epoch
    epoch="$(date -d "$started" +%s 2>/dev/null)" || return 0
    (( now - epoch >= STALE_SECONDS ))
  }
  echo "cleanup-docker: retirando residuos de pruebas con más de ${STALE_SECONDS}s"
  mapfile -t stale_containers < <(docker ps -aq --filter "label=solv.test.run")
  for id in "${stale_containers[@]}"; do
    labels="$(docker inspect -f '{{ index .Config.Labels "solv.test.run" }}|{{ index .Config.Labels "solv.test.started" }}' "$id")"
    run_id="${labels%%|*}"
    started="${labels#*|}"
    [[ "$run_id" == "$RUN_ID" ]] && continue
    if is_stale "$started"; then docker rm -f "$id" >/dev/null; fi
  done
  mapfile -t stale_volumes < <(docker volume ls -q --filter "label=solv.test.run")
  for name in "${stale_volumes[@]}"; do
    labels="$(docker volume inspect -f '{{ index .Labels "solv.test.run" }}|{{ index .Labels "solv.test.started" }}' "$name")"
    run_id="${labels%%|*}"
    started="${labels#*|}"
    [[ "$run_id" == "$RUN_ID" ]] && continue
    if is_stale "$started"; then docker volume rm -f "$name" >/dev/null; fi
  done
  mapfile -t stale_networks < <(docker network ls -q --filter "label=solv.test.run")
  for id in "${stale_networks[@]}"; do
    labels="$(docker network inspect -f '{{ index .Labels "solv.test.run" }}|{{ index .Labels "solv.test.started" }}' "$id")"
    run_id="${labels%%|*}"
    started="${labels#*|}"
    [[ "$run_id" == "$RUN_ID" ]] && continue
    if is_stale "$started"; then docker network rm "$id" >/dev/null; fi
  done
  echo "cleanup-docker: barrido de residuos terminado"
  exit 0
fi

echo "cleanup-docker: limpiando recursos con $FILTER"
mapfile -t containers < <(docker ps -aq --filter "$FILTER")
if (( ${#containers[@]} > 0 )); then
  docker rm -f -- "${containers[@]}" >/dev/null
fi

mapfile -t volumes < <(docker volume ls -q --filter "$FILTER")
if (( ${#volumes[@]} > 0 )); then
  docker volume rm -f -- "${volumes[@]}" >/dev/null
fi

mapfile -t networks < <(docker network ls -q --filter "$FILTER")
if (( ${#networks[@]} > 0 )); then
  for network in "${networks[@]}"; do
    docker network rm "$network" >/dev/null
  done
fi

remaining_containers="$(docker ps -aq --filter "$FILTER")"
remaining_volumes="$(docker volume ls -q --filter "$FILTER")"
remaining_networks="$(docker network ls -q --filter "$FILTER")"
if [[ -n "$remaining_containers" || -n "$remaining_volumes" || -n "$remaining_networks" ]]; then
  echo "cleanup-docker: quedaron recursos etiquetados para $RUN_ID" >&2
  exit 1
fi

echo "cleanup-docker: limpieza terminada para $RUN_ID"
