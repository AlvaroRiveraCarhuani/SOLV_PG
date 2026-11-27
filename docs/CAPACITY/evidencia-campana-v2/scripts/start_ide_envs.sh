#!/usr/bin/env bash
# Arranca los 3 entornos IDE que acompañan al soak (2 a 3 exigidos).
set -u

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RAW="$ROOT/raw"
TOKEN=$(python3 -c 'import json,sys;print(json.load(open(sys.argv[1]))["token"])' "$RAW/token-ide.json")
SUBJECT="d0000000-0000-0000-0000-0000000000b1"
OUT="$RAW/soak_ide_environments.txt"

: > "$OUT"
for N in 1 2 3; do
  EX="f1000000-0000-0000-0000-00000000000${N}"
  START=$(date -u +%Y-%m-%dT%H:%M:%SZ)
  RES=$(curl -sS --insecure --max-time 300 \
    -H "Host: solv.dedyn.io" -H "Content-Type: application/json" \
    -H "Cookie: solv_session=$TOKEN" \
    -X POST https://127.0.0.1/api/v1/environments/start \
    -d "{\"subject_id\":\"$SUBJECT\",\"exercise_id\":\"$EX\"}" \
    -w '\nHTTP=%{http_code} TIME=%{time_total}')
  END=$(date -u +%Y-%m-%dT%H:%M:%SZ)
  echo "exercise=${EX} start_utc=${START} end_utc=${END}" >> "$OUT"
  echo "$RES" | tail -2 >> "$OUT"
  echo "ide ${EX} http=$(echo "$RES" | tail -1)"
done
