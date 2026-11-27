#!/usr/bin/env python3
"""Resume las métricas crudas de la campaña v2 en tablas listas para el anexo.

Fuentes: raw/e1/*.csv (k6 CSV), raw/e6_*.csv, raw/soak_metrics.csv,
raw/soak_k6.csv. Salida: raw/metrics_summary.json + tabla por consola.
"""
import csv
import json
import math
from collections import Counter, defaultdict
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
RAW = ROOT / "raw"


def percentile(values, p):
    if not values:
        return None
    ordered = sorted(values)
    k = max(1, math.ceil(p / 100 * len(ordered)))
    return ordered[k - 1]


def stats(values):
    values = [v for v in values if v is not None]
    if not values:
        return {"n": 0}
    n = len(values)
    mean = sum(values) / n
    var = sum((v - mean) ** 2 for v in values) / (n - 1) if n > 1 else 0.0
    return {
        "n": n,
        "mean": round(mean, 2),
        "stddev": round(math.sqrt(var), 2),
        "p50": round(percentile(values, 50), 2),
        "p95": round(percentile(values, 95), 2),
        "p99": round(percentile(values, 99), 2),
        "min": round(min(values), 2),
        "max": round(max(values), 2),
    }


def read_k6_csv(path):
    rows = []
    if not path.exists():
        return rows
    with path.open(newline="") as fh:
        for row in csv.DictReader(fh):
            rows.append(row)
    return rows


def http_breakdown(rows):
    """N, códigos y tasa de éxito por etiqueta `name` de la petición."""
    by_name = defaultdict(Counter)
    for row in rows:
        if row.get("metric_name") != "http_reqs":
            continue
        by_name[row.get("name") or ""][row.get("status") or "0"] += 1
    out = {}
    for name, codes in by_name.items():
        total = sum(codes.values())
        ok_codes = {"200", "201"}
        ok = sum(v for c, v in codes.items() if c in ok_codes)
        out[name] = {"total": total, "ok": ok,
                     "ok_pct": round(100 * ok / total, 2) if total else 0.0,
                     "codes": dict(sorted(codes.items()))}
    return out


def trend_values(rows, metric):
    return [float(r["metric_value"]) for r in rows
            if r.get("metric_name") == metric and r.get("metric_value") not in ("", None)]


def request_durations(rows, name):
    """Duración HTTP reportada por k6 para las peticiones con etiqueta `name`."""
    return [float(r["metric_value"]) for r in rows
            if r.get("metric_name") == "http_req_duration" and r.get("name") == name
            and r.get("metric_value") not in ("", None)]


def level_http(rows_list):
    merged = defaultdict(Counter)
    for rows in rows_list:
        for name, codes in http_breakdown(rows).items():
            merged[name].update(codes["codes"])
    out = {}
    for name, codes in merged.items():
        total = sum(codes.values())
        ok = sum(v for c, v in codes.items() if c in {"200", "201"})
        out[name] = {"total": total, "ok": ok,
                     "ok_pct": round(100 * ok / total, 2) if total else 0.0,
                     "codes": dict(sorted(codes.items()))}
    return out


def e1_summary():
    out = {"replicas": [], "levels": {}}
    for level in (10, 20, 35):
        per_level_read, per_level_submit = [], []
        level_rows, level_eval = [], []
        for rep in (1, 2, 3):
            path = RAW / "e1" / f"L{level}_r{rep}.csv"
            rows = read_k6_csv(path)
            if not rows:
                continue
            http = http_breakdown(rows)
            entry = {
                "level": level, "replica": rep,
                "read_http": http.get("read", {}),
                "evaluation_http": http.get("evaluation", {}),
                "submission_http": http.get("submission", {}),
                "status_http": http.get("submission_status", {}),
                "read": stats(trend_values(rows, "read_duration")),
                "submit": stats(trend_values(rows, "submit_duration")),
                "evaluation": stats(request_durations(rows, "evaluation")),
            }
            out["replicas"].append(entry)
            per_level_read += trend_values(rows, "read_duration")
            per_level_submit += trend_values(rows, "submit_duration")
            level_rows.append(rows)
            level_eval += request_durations(rows, "evaluation")
        out["levels"][str(level)] = {
            "read": stats(per_level_read),
            "submit": stats(per_level_submit),
            "evaluation": stats(level_eval),
            "http": level_http(level_rows),
            "replicas": 3,
        }
    return out


def breakpoint_summary():
    rows = read_k6_csv(RAW / "breakpoint.csv")
    if not rows:
        return {}
    return {
        "http": http_breakdown(rows),
        "read": stats(trend_values(rows, "read_duration")),
        "evaluation": stats(request_durations(rows, "evaluation")),
        "submit": stats(trend_values(rows, "submit_duration")),
    }


def e6_summary():
    def load(name):
        path = RAW / name
        if not path.exists():
            return []
        with path.open(newline="") as fh:
            return list(csv.DictReader(fh))

    cold, warm = load("e6_cold.csv"), load("e6_warm.csv")
    cold_ms = [float(r["duration_ms"]) for r in cold if r.get("duration_ms")]
    warm_ms = [float(r["resume_ms"]) for r in warm if r.get("resume_ms")]
    pause_ms = [float(r["pause_ms"]) for r in warm if r.get("pause_ms")]
    return {
        "cold": {
            "stats": stats(cold_ms),
            "http": dict(Counter(r["http"] for r in cold)),
            "image_cached": dict(Counter(r.get("image_cached") for r in cold)),
            "volume_state": dict(Counter(r.get("volume_state") for r in cold)),
        },
        "warm": {
            "resume": stats(warm_ms),
            "pause": stats(pause_ms),
            "http": dict(Counter(r["resume_http"] for r in warm)),
            "pause_http": dict(Counter(r["pause_http"] for r in warm)),
            "docker_state": dict(Counter(r.get("docker_state_after_pause") for r in warm)),
        },
    }


def soak_summary():
    metrics_path = RAW / "soak_metrics.csv"
    if not metrics_path.exists():
        return {}
    with metrics_path.open(newline="") as fh:
        rows = list(csv.DictReader(fh))
    if not rows:
        return {}

    def at(minute):
        best = min(rows, key=lambda r: abs(float(r["minute"]) - minute))
        rss = float(best["backend_vmrss_mb"])
        valid = rss >= 20  # descarta la muestra tomada del supervisor Air
        return {
            "minute": float(best["minute"]),
            "timestamp_utc": best["timestamp_utc"],
            "mem_available_mb": int(best["mem_available_mb"]),
            "backend_vmrss_mb": rss if valid else None,
            "vmrss_valid": valid,
            "load1": float(best["load1"]),
            "disk_free_gb": float(best["disk_free_gb"]),
            "containers_running": int(best["containers_running"]),
            "env_containers": int(best["env_containers"]),
        }

    cuts = [at(m) for m in (0, 10, 20, 30)]
    # La primera muestra quedó tomada del supervisor Air (6.79 MiB) por un fallo
    # de instrumentación corregido a los 3 minutos; se excluye de la serie.
    vmrss = [float(r["backend_vmrss_mb"]) for r in rows if float(r["backend_vmrss_mb"]) >= 20]
    mem = [int(r["mem_available_mb"]) for r in rows]

    buckets = defaultdict(lambda: {"total": 0, "errors": 0, "read": [], "submit": []})
    for row in read_k6_csv(RAW / "soak_k6.csv"):
        if row.get("metric_name") == "http_reqs":
            ts = row.get("timestamp", "")
            bucket = ts[11:16]  # HH:MM
            buckets[bucket]["total"] += 1
            if row.get("status") not in {"200", "201"}:
                buckets[bucket]["errors"] += 1
        elif row.get("metric_name") == "read_duration":
            ts = row.get("timestamp", "")
            buckets[ts[11:16]]["read"].append(float(row["metric_value"]))
        elif row.get("metric_name") == "submit_duration":
            ts = row.get("timestamp", "")
            buckets[ts[11:16]]["submit"].append(float(row["metric_value"]))

    per_minute = []
    for minute, data in sorted(buckets.items()):
        per_minute.append({
            "minute_of_day": minute,
            "requests": data["total"],
            "errors": data["errors"],
            "error_pct": round(100 * data["errors"] / data["total"], 2) if data["total"] else 0.0,
            "read_p95": percentile(data["read"], 95) or 0,
            "submit_p95": percentile(data["submit"], 95) or 0,
        })

    return {
        "cuts": cuts,
        "vmrss": stats(vmrss),
        "mem_available": stats(mem),
        "min_mem_available_mb": min(mem),
        "max_vmrss_mb": max(vmrss) if vmrss else None,
        "growth_vmrss_mb": round(vmrss[-1] - vmrss[0], 2) if len(vmrss) > 1 else None,
        "per_minute": per_minute,
        "k6": {
            "http": http_breakdown(read_k6_csv(RAW / "soak_k6.csv")),
            "read": stats(trend_values(read_k6_csv(RAW / "soak_k6.csv"), "read_duration")),
            "submit": stats(trend_values(read_k6_csv(RAW / "soak_k6.csv"), "submit_duration")),
        },
    }


def calibration_summary():
    out = {}
    for path in sorted((RAW / "calibration").glob("*.csv")) if (RAW / "calibration").exists() else []:
        rows = read_k6_csv(path)
        out[path.stem] = {
            "http": http_breakdown(rows),
            "read": stats(trend_values(rows, "read_duration")),
            "evaluation": stats(request_durations(rows, "evaluation")),
            "submit": stats(trend_values(rows, "submit_duration")),
        }
    return out


def fmt(s):
    if not s or not s.get("n"):
        return "n=0"
    return (f"n={s['n']} · {s['mean']}±{s['stddev']} ms · "
            f"p50 {s['p50']} · p95 {s['p95']} · p99 {s['p99']} · "
            f"mín {s['min']} · máx {s['max']}")


def main():
    summary = {
        "e1": e1_summary(),
        "breakpoint": breakpoint_summary(),
        "e6": e6_summary(),
        "soak": soak_summary(),
        "calibration": calibration_summary(),
    }
    (RAW / "metrics_summary.json").write_text(json.dumps(summary, indent=2, ensure_ascii=False))

    print("== E1 por nivel (3 réplicas agregadas) ==")
    for level, data in summary["e1"]["levels"].items():
        print(f"{level} VUs · lectura:    {fmt(data['read'])}")
        print(f"{level} VUs · evaluación: {fmt(data['evaluation'])}")
        print(f"{level} VUs · envío:      {fmt(data['submit'])}")
        print(f"{level} VUs · http: {json.dumps(data['http'], ensure_ascii=False)}")
    print("\n== E1 por réplica (HTTP) ==")
    for r in summary["e1"]["replicas"]:
        print(f"L{r['level']} r{r['replica']} lectura {r['read_http']} envío {r['submission_http']}")
    print("\n== Punto de quiebre ==")
    print(json.dumps(summary["breakpoint"].get("http", {}), ensure_ascii=False))
    print("\n== E6 ==")
    print(json.dumps(summary["e6"], ensure_ascii=False, indent=2))
    print("\n== Soak ==")
    print(json.dumps({k: v for k, v in summary["soak"].items() if k != "per_minute"},
                     ensure_ascii=False, indent=2))
    if summary["calibration"]:
        print("\n== Calibración de tiempo de espera ==")
        print(json.dumps(summary["calibration"], ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
