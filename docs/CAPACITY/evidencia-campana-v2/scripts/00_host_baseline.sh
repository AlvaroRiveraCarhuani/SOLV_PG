#!/usr/bin/env bash
# Estado inicial controlado del nodo piloto antes de la campaña v2.
# Uso: bash scripts/00_host_baseline.sh > raw/host_baseline.txt
set -u

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
cd "$REPO_ROOT"

ts() { date -u +%Y-%m-%dT%H:%M:%SZ; }

echo "# host_baseline — campaña experimental v2 (SOLV)"
echo "captured_at_utc: $(ts)"
echo "repo: $(pwd)"
echo "commit: $(git rev-parse HEAD)"
echo "commit_subject: $(git log -1 --pretty=%s)"
echo "branch: $(git rev-parse --abbrev-ref HEAD)"
echo "dirty_worktree: $(git status --porcelain | wc -l) archivos modificados"
echo
echo "## uname"
uname -a
echo
echo "## cpu"
nproc
lscpu | grep -E 'Model name|^CPU\(s\)|Thread|Core|Socket'
echo
echo "## memoria (free -m)"
free -m
echo
echo "## loadavg / uptime (reposo)"
cat /proc/loadavg
uptime
echo
echo "## disco (df -h /)"
df -h /
echo
echo "## temperatura y frecuencia (reposo)"
for z in /sys/class/thermal/thermal_zone*/temp; do
  [ -r "$z" ] || continue
  printf '%s: %s m°C\n' "$z" "$(cat "$z")"
done
if command -v sensors >/dev/null 2>&1; then sensors 2>/dev/null | sed -n '1,40p'; else echo "sensors: no instalado"; fi
echo
echo "## docker ps"
docker ps --format 'table {{.Names}}\t{{.Image}}\t{{.Status}}\t{{.Ports}}'
echo
echo "## docker version"
docker version --format 'client={{.Client.Version}} server={{.Server.Version}}'
echo
echo "## contenedores / volumenes / redes"
echo "containers_running: $(docker ps -q | wc -l)"
echo "containers_total: $(docker ps -aq | wc -l)"
echo "volumes_total: $(docker volume ls -q | wc -l)"
docker network ls --format '{{.Name}}\t{{.Driver}}'
echo
echo "## procesos del backend (VmRSS del proceso Go dentro del contenedor)"
docker exec solv_backend sh -lc 'p=$(pgrep -f "solv-backend|/tmp/go-build|air" | head -1); echo "pid=$p"; grep -E "VmRSS|VmHWM" /proc/$p/status 2>/dev/null' 2>/dev/null || echo "no se pudo leer VmRSS en solv_backend"
echo
echo "## limites del rate limiter declarados en .env"
grep -E '^RATE_LIMIT_IP_' .env 2>/dev/null || echo "RATE_LIMIT_IP_* no declarado (aplica el default del binario)"
echo
echo "## topologia de redes de examen"
docker network inspect solv-exam-net --format '{{.Name}} internal={{.Internal}} driver={{.Driver}}' 2>/dev/null || echo "solv-exam-net no existe"
