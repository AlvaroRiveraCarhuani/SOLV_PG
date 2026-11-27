#!/usr/bin/env python3
"""Gráfica del soak (memoria disponible, VmRSS y latencia) si matplotlib existe."""
import csv
from datetime import datetime
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
RAW = ROOT / "raw"

try:
    import matplotlib
    matplotlib.use("Agg")
    import matplotlib.pyplot as plt
except ModuleNotFoundError:
    raise SystemExit("matplotlib no disponible: se omite raw/soak_30m.png")


def read_metrics():
    with (RAW / "soak_metrics.csv").open(newline="") as fh:
        return list(csv.DictReader(fh))


def read_latency():
    """p95 de lectura por minuto de reloj (clave = epoch del minuto), y envíos."""
    reads, submits = {}, 0
    with (RAW / "soak_k6.csv").open(newline="") as fh:
        for row in csv.DictReader(fh):
            if row.get("metric_name") == "read_duration":
                stamp = float(row["timestamp"])
                bucket = int(stamp // 60 * 60)
                reads.setdefault(bucket, []).append(float(row["metric_value"]))
            elif row.get("metric_name") == "submit_duration":
                submits += 1
    return reads, submits


def p95(values):
    ordered = sorted(values)
    k = max(1, -(-95 * len(ordered) // 100))
    return ordered[k - 1]


def main():
    rows = read_metrics()
    t0 = datetime.strptime(rows[0]["timestamp_utc"], "%Y-%m-%dT%H:%M:%SZ")
    minutes = [float(r["minute"]) for r in rows]
    mem = [int(r["mem_available_mb"]) for r in rows]
    vmrss = [float(r["backend_vmrss_mb"]) for r in rows]
    load = [float(r["load1"]) for r in rows]
    reads, submits = read_latency()

    fig, axes = plt.subplots(3, 1, figsize=(11, 10), sharex=True)

    axes[0].plot(minutes, mem, label="MemAvailable del host (MiB)", color="#15803D")
    axes[0].plot(minutes, vmrss, label="VmRSS solv-api-dev (MiB)", color="#B45309")
    axes[0].set_ylabel("MiB")
    axes[0].legend(loc="center right")
    axes[0].set_title("Soak 30 min — 15 VUs de aula + 2 entornos IDE activos")

    axes[1].plot(minutes, load, color="#1D4ED8")
    axes[1].set_ylabel("loadavg 1 min")

    if reads:
        buckets = sorted(reads)
        xs = [(datetime.utcfromtimestamp(b) - t0).total_seconds() / 60 for b in buckets]
        axes[2].plot(xs, [p95(reads[b]) for b in buckets], color="#15803D",
                     marker="o", markersize=3, label="lectura p95 (ms)")
    axes[2].text(0.01, 0.85,
                 f"envíos de evaluación completados: {submits} "
                 "(todas las evaluaciones agotaron el timeout de 60 s)",
                 transform=axes[2].transAxes, fontsize=9, color="#B91C1C")
    axes[2].set_ylabel("ms")
    axes[2].set_xlabel("minutos desde el inicio del soak")
    axes[2].legend(loc="upper right")

    axes[2].set_xticks(list(range(0, int(max(minutes)) + 1, 5)))
    fig.tight_layout()
    out = RAW / "soak_30m.png"
    fig.savefig(out, dpi=120)
    print(f"grafica={out}")


if __name__ == "__main__":
    main()
