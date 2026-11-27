#!/usr/bin/env python3
"""Muestrea el host cada 60 s durante el soak y escribe raw/soak_metrics.csv.

Columnas: timestamp real UTC, minuto, MemAvailable del host, VmRSS del proceso
Go de solv-backend, loadavg 1 min, disco libre, contenedores activos y
contenedores de entorno.
"""
import csv
import json
import os
import subprocess
import sys
import time
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
RAW = ROOT / "raw"
DURATION_S = int(os.environ.get("SOAK_SECONDS", "1980"))  # 33 min: cubre 30 min + arranque/cierre
INTERVAL = 60


def ts() -> str:
    return time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())


def mem_available_mb() -> int:
    for line in Path("/proc/meminfo").read_text().splitlines():
        if line.startswith("MemAvailable:"):
            return int(line.split()[1]) // 1024
    return -1


def backend_vmrss_mb() -> float:
    # El binario Go se llama solv-api-dev dentro del contenedor de desarrollo;
    # "air" es solo el supervisor y su RSS no representa al orquestador.
    cmd = ('p=$(pgrep -x solv-api-dev | head -1); '
           'if [ -n "$p" ]; then awk "/^VmRSS:/{print \\$2}" /proc/$p/status; fi')
    out = subprocess.run(["docker", "exec", "solv_backend", "sh", "-lc", cmd],
                         capture_output=True, text=True)
    try:
        return round(int(out.stdout.strip()) / 1024, 2)
    except (ValueError, TypeError):
        return -1


def disk_free_gb() -> float:
    st = os.statvfs("/")
    return round(st.f_bavail * st.f_frsize / 1e9, 2)


def containers() -> "tuple[int,int]":
    all_ids = subprocess.run(["docker", "ps", "-q"], capture_output=True, text=True).stdout.split()
    env_ids = subprocess.run(["docker", "ps", "-q", "--filter", "name=solv-environment"],
                             capture_output=True, text=True).stdout.split()
    return len(all_ids), len(env_ids)


def load1() -> float:
    return float(Path("/proc/loadavg").read_text().split()[0])


def main() -> None:
    RAW.mkdir(parents=True, exist_ok=True)
    path = RAW / "soak_metrics.csv"
    resume = path.exists() and path.stat().st_size > 0
    first_ts = None
    if resume:
        with path.open(newline="") as fh:
            rows = list(csv.DictReader(fh))
        if rows:
            first_ts = time.mktime(time.strptime(rows[0]["timestamp_utc"], "%Y-%m-%dT%H:%M:%SZ"))
    start = first_ts or time.time()
    with path.open("a" if resume else "w", newline="") as fh:
        writer = csv.writer(fh)
        if not resume:
            writer.writerow(["timestamp_utc", "minute", "mem_available_mb", "backend_vmrss_mb",
                             "load1", "disk_free_gb", "containers_running", "env_containers"])
            writer.writerow([ts(), 0, mem_available_mb(), backend_vmrss_mb(), load1(),
                             disk_free_gb(), *containers()])
            fh.flush()
        while time.time() - start < DURATION_S:
            time.sleep(INTERVAL)
            minute = round((time.time() - start) / 60, 2)
            total, env = containers()
            writer.writerow([ts(), minute, mem_available_mb(), backend_vmrss_mb(),
                             load1(), disk_free_gb(), total, env])
            fh.flush()
    print(f"soak sampler done at {ts()}")


if __name__ == "__main__":
    sys.exit(main())
