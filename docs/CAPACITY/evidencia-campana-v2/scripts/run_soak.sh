#!/usr/bin/env bash
# Soak de 30 minutos: 15 VUs de aula + 2-3 entornos IDE activos + muestreo del host.
set -u

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SCRIPTS="$ROOT/scripts"
RAW="$ROOT/raw"
mkdir -p "$RAW"

ts() { date -u +%Y-%m-%dT%H:%M:%SZ; }

echo "soak_start_utc=$(ts)"
bash "$SCRIPTS/start_ide_envs.sh"

python3 "$SCRIPTS/soak_sampler.py" > "$RAW/soak_sampler.log" 2>&1 &
SAMPLER=$!
echo "soak_sampler_pid=$SAMPLER start_utc=$(ts)"

# 33 minutos de muestreo: arranque de k6 + 30 min + cierre.
SOAK_SECONDS=1980 k6 run \
  --quiet \
  --summary-time-unit ms \
  --summary-trend-stats "avg,min,med,p(50),p(90),p(95),p(99),max,count" \
  --out "csv=$RAW/soak_k6.csv" \
  -e SUMMARY_OUT="$RAW/soak_summary.json" \
  "$SCRIPTS/soak_classroom.js" \
  > "$RAW/soak_k6.log" 2>&1
RC=$?
echo "soak_k6_end_utc=$(ts) exit=$RC"

wait "$SAMPLER" || true
echo "soak_sampler_end_utc=$(ts)"
echo "soak_done_utc=$(ts)"
