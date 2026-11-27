#!/usr/bin/env bash
# Driver de la campaña v2. Ejecuta todo en orden y registra timestamps ISO-8601.
#
# Orden: baseline → siembra → E6 (nodo quieto) → E1 → saturación → soak → resumen → limpieza.
# E6 va antes de E1 a propósito: la provisión en frío se mide con el daemon de
# Docker sin cola de contenedores de evaluación.
set -u

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REPO="$(cd "$ROOT/../../.." && pwd)"
SCRIPTS="$ROOT/scripts"
RAW="$ROOT/raw"
LOG="$ROOT/campaign_run.log"
mkdir -p "$RAW"

ts() { date -u +%Y-%m-%dT%H:%M:%SZ; }
log() { printf '[%s] %s\n' "$(ts)" "$*" | tee -a "$LOG"; }

stage() {
  local name="$1"; shift
  log "stage=${name} start_utc=$(ts)"
  "$@" 2>&1 | tee -a "$LOG"
  local rc=${PIPESTATUS[0]}
  log "stage=${name} end_utc=$(ts) exit=${rc}"
  return "$rc"
}

cd "$REPO"

log "campaign_v2_boot commit=$(git rev-parse HEAD) host=$(hostname) kernel=$(uname -r)"

stage baseline bash "$SCRIPTS/00_host_baseline.sh" > "$RAW/host_baseline.txt" 2>&1
log "host_baseline=raw/host_baseline.txt bytes=$(wc -c < "$RAW/host_baseline.txt")"

log "stage=seed start_utc=$(ts)"
docker exec -i solv_db sh -lc 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -v ON_ERROR_STOP=1' \
  < "$SCRIPTS/01_seed_campaign.sql" 2>&1 | tee -a "$LOG"
log "stage=seed end_utc=$(ts) exit=${PIPESTATUS[0]}"

stage tokens python3 "$SCRIPTS/02_mint_tokens.py"

# Verificación mínima del flujo autenticado antes de medir.
log "stage=smoke start_utc=$(ts)"
TOKEN=$(python3 -c 'import json,sys;print(json.load(open(sys.argv[1]))["1"]["token"])' "$RAW/tokens-campaign.json")
curl -sS --insecure -o "$RAW/smoke_read.json" -w 'smoke_read_http=%{http_code} time=%{time_total}\n' \
  -H "Host: solv.dedyn.io" -H "Cookie: solv_session=$TOKEN" \
  "https://127.0.0.1/api/v1/exercises/f0000000-0000-0000-0000-000000000001" | tee -a "$LOG"
log "stage=smoke end_utc=$(ts)"

stage e6 python3 "$SCRIPTS/e6_cold_warm.py"

stage e1 bash "$SCRIPTS/run_e1.sh"

log "stage=breakpoint start_utc=$(ts)"
k6 run --quiet --summary-time-unit ms \
  --summary-trend-stats "avg,min,med,p(50),p(90),p(95),p(99),max,count" \
  --out "csv=$RAW/breakpoint.csv" \
  -e SUMMARY_OUT="$RAW/breakpoint_summary.json" \
  "$SCRIPTS/e1_breakpoint.js" 2>&1 | tee -a "$LOG"
log "stage=breakpoint end_utc=$(ts) exit=${PIPESTATUS[0]}"

log "settling_after_breakpoint=60s start_utc=$(ts)"
sleep 60
log "settling_after_breakpoint done_utc=$(ts) containers_running=$(docker ps -q | wc -l)"

stage soak bash "$SCRIPTS/run_soak.sh"

stage summarize python3 "$SCRIPTS/summarize.py"

log "campaign_v2_done_utc=$(ts)"
