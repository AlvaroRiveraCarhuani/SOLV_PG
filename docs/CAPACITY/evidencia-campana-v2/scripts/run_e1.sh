#!/usr/bin/env bash
# E1: 10/20/35 VUs, 3 réplicas de 60 s por nivel, 30 s de enfriamiento.
set -u

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SCRIPTS="$ROOT/scripts"
RAW="$ROOT/raw"
OUT="$RAW/e1"
mkdir -p "$OUT"

ts() { date -u +%Y-%m-%dT%H:%M:%SZ; }
echo "e1_start_utc=$(ts)"

for LEVEL in 10 20 35; do
  for REP in 1 2 3; do
    START=$(ts)
    echo "e1 level=${LEVEL} replica=${REP} start_utc=${START}"
    k6 run \
      --quiet \
      --summary-time-unit ms \
      --summary-trend-stats "avg,min,med,p(50),p(90),p(95),p(99),max,count" \
      --out "csv=$OUT/L${LEVEL}_r${REP}.csv" \
      -e VUS="$LEVEL" -e DURATION=60s -e RAMP=10s \
      -e SUMMARY_OUT="$OUT/L${LEVEL}_r${REP}_summary.json" \
      "$SCRIPTS/e1_classroom.js" \
      > "$OUT/L${LEVEL}_r${REP}.log" 2>&1
    RC=$?
    END=$(ts)
    echo "e1 level=${LEVEL} replica=${REP} end_utc=${END} exit=${RC}"
    if [ "$REP" -lt 3 ]; then
      echo "e1 cooldown=30s start_utc=$(ts)"
      sleep 30
      echo "e1 cooldown_end_utc=$(ts)"
    fi
  done
done

echo "e1_done_utc=$(ts)"
