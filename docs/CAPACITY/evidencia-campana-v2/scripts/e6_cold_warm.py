#!/usr/bin/env python3
"""E6 — provisión en frío vs. operación en caliente de un IDE persistente.

Frío    : se elimina contenedor + fila + volumen previos y se mide POST
          /api/v1/environments/start (creación desde cero, imagen en caché local).
Caliente: hibernación (pause) + espera de 5 s + reanudación (start), verificando
          por Docker que el contenedor existe y está realmente pausado antes de
          medir la reanudación.

N = 12 por condición, 3 bloques de 4 corridas con 30 s entre bloques.
Salida: raw/e6_cold.csv, raw/e6_warm.csv, raw/e6_raw.json
"""
import csv
import http.client
import json
import os
import ssl
import subprocess
import time
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
RAW = ROOT / "raw"
TENANT = "00000000-0000-0000-0000-000000000001"
HOST = "127.0.0.1"
HOST_HEADER = "solv.dedyn.io"
SUBJECT = "d0000000-0000-0000-0000-0000000000b1"
N_RUNS = int(os.environ.get("E6_RUNS", "12"))
BLOCK = int(os.environ.get("E6_BLOCK", "4"))
BLOCKS = max(1, N_RUNS // BLOCK)
BLOCK_GAP = 30
HIBERNATE_WAIT = 5

TOKEN = json.loads((RAW / "token-ide.json").read_text())["token"]


def ts() -> str:
    return time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())


def now_ms() -> float:
    return time.perf_counter()


def api(method: str, path: str, payload=None, timeout=300):
    ctx = ssl._create_unverified_context()
    conn = http.client.HTTPSConnection(HOST, 443, context=ctx, timeout=timeout)
    headers = {"Host": HOST_HEADER, "Cookie": f"solv_session={TOKEN}"}
    body = None
    if payload is not None:
        body = json.dumps(payload)
        headers["Content-Type"] = "application/json"
    start = now_ms()
    try:
        conn.request(method, path, body=body, headers=headers)
        res = conn.getresponse()
        raw = res.read().decode("utf-8", "replace")
        elapsed = (now_ms() - start) * 1000
        try:
            parsed = json.loads(raw) if raw else {}
        except json.JSONDecodeError:
            parsed = {"_raw": raw[:400]}
        return res.status, parsed, elapsed
    except Exception as exc:  # timeout, reset, TLS
        return 0, {"_error": f"{type(exc).__name__}: {exc}"}, (now_ms() - start) * 1000
    finally:
        conn.close()


def psql(sql: str) -> str:
    out = subprocess.run(
        ["docker", "exec", "solv_db", "sh", "-lc",
         'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -tA -c ' + json.dumps(sql)],
        capture_output=True, text=True,
    )
    return out.stdout.strip()


def docker(*args: str, check=False) -> str:
    out = subprocess.run(["docker", *args], capture_output=True, text=True)
    if check and out.returncode != 0:
        raise RuntimeError(f"docker {' '.join(args)}: {out.stderr.strip()}")
    return out.stdout.strip()


def env_row():
    row = psql(
        "SELECT id, COALESCE(container_id,''), status FROM environments "
        f"WHERE student_id='c0000000-0000-0000-0002-0000000000f1' AND subject_id='{SUBJECT}' "
        "AND exercise_id='f1000000-0000-0000-0000-000000000001'"
    )
    if not row:
        return None
    parts = row.split("|")
    return {"id": parts[0], "container_id": parts[1], "status": parts[2]} if len(parts) >= 3 else None


def image_cached() -> bool:
    tpl = psql("SELECT docker_image FROM lab_templates WHERE id='b395313f-701f-49cf-807e-638230241b3c'")
    if not tpl:
        return False
    return subprocess.run(["docker", "image", "inspect", tpl], capture_output=True).returncode == 0


def container_paused(container_id: str) -> "bool|None":
    if not container_id:
        return None
    out = subprocess.run(
        ["docker", "inspect", container_id, "--format", "{{.State.Paused}}|{{.State.Status}}"],
        capture_output=True, text=True)
    if out.returncode != 0:
        return None
    return out.stdout.strip()


def volume_of(container_id: str) -> str:
    if not container_id:
        return ""
    out = subprocess.run(
        ["docker", "inspect", container_id, "--format",
         "{{range .Mounts}}{{if eq .Type \"volume\"}}{{.Name}} {{end}}{{end}}"],
        capture_output=True, text=True)
    return out.stdout.strip()


def reset_to_zero() -> dict:
    """Deja el alumno IDE sin contenedor, sin fila y sin volumen."""
    info = {"terminate_http": None, "volume": "", "volume_removed": False, "row_deleted": False}
    row = env_row()
    if row:
        if row["container_id"]:
            info["volume"] = volume_of(row["container_id"])
        status, _, _ = api("DELETE", f"/api/v1/environments/{row['id']}")
        info["terminate_http"] = status
        if row["container_id"]:
            docker("rm", "-f", row["container_id"])
        psql(f"DELETE FROM environments WHERE id='{row['id']}'")
        info["row_deleted"] = True
    # Volumen huérfano del intento (si el contenedor ya no existía).
    if not info["volume"]:
        vols = docker("volume", "ls", "-q").splitlines()
        info["volume"] = next((v for v in vols if "c0000000-0000-0000-0002-0000000000f1" in v), "")
    if info["volume"]:
        info["volume_removed"] = docker("volume", "rm", "-f", info["volume"]).lower().find("error") == -1
        info["volume"] = info["volume"]
    return info


def cold_run(iteration: int, block: int) -> dict:
    reset = reset_to_zero()
    cached = image_cached()
    start_utc = ts()
    status, body, elapsed = api("POST", "/api/v1/environments/start",
                                {"subject_id": SUBJECT,
                                 "exercise_id": "f1000000-0000-0000-0000-000000000001"})
    end_utc = ts()
    data = body.get("data") or {}
    row = env_row() or {}
    return {
        "condition": "cold", "iteration": iteration, "block": block,
        "start_utc": start_utc, "end_utc": end_utc,
        "duration_ms": round(elapsed, 2), "http": status,
        "environment_id": data.get("id", row.get("id", "")),
        "container_id": data.get("container_id", row.get("container_id", "")),
        "image_cached": cached,
        "volume_state": "removed_before_run" if reset.get("volume_removed") else "no_volume_found",
        "volume_name": reset.get("volume", ""),
        "error": body.get("error", "") if status != 200 else "",
    }


def ensure_running() -> dict:
    status, body, elapsed = api("POST", "/api/v1/environments/start",
                                {"subject_id": SUBJECT,
                                 "exercise_id": "f1000000-0000-0000-0000-000000000001"})
    return {"http": status, "ms": round(elapsed, 2), "body": body}


def warm_run(iteration: int, block: int) -> dict:
    warm = ensure_running()
    row = env_row() or {}
    env_id = row.get("id", "")
    cid = row.get("container_id", "")

    start_utc = ts()
    pause_status, pause_body, pause_ms = api("POST", f"/api/v1/environments/{env_id}/pause")
    state_after_pause = container_paused(cid)
    time.sleep(HIBERNATE_WAIT)
    resume_status, resume_body, resume_ms = api("POST", "/api/v1/environments/start",
                                                {"subject_id": SUBJECT,
                                                 "exercise_id": "f1000000-0000-0000-0000-000000000001"})
    end_utc = ts()
    data = resume_body.get("data") or {}
    return {
        "condition": "warm", "iteration": iteration, "block": block,
        "start_utc": start_utc, "end_utc": end_utc,
        "ensure_running_http": warm["http"], "ensure_running_ms": warm["ms"],
        "pause_ms": round(pause_ms, 2), "pause_http": pause_status,
        "docker_state_after_pause": state_after_pause,
        "resume_ms": round(resume_ms, 2), "resume_http": resume_status,
        "container_id": data.get("container_id", cid),
        "environment_id": data.get("id", env_id),
        "error": resume_body.get("error", "") if resume_status != 200 else "",
    }


def write_csv(path: Path, rows: list, fields: list) -> None:
    with path.open("w", newline="") as fh:
        writer = csv.DictWriter(fh, fieldnames=fields)
        writer.writeheader()
        for row in rows:
            writer.writerow({k: row.get(k, "") for k in fields})


def main() -> None:
    RAW.mkdir(parents=True, exist_ok=True)
    print(f"e6_start_utc={ts()} image_cached={image_cached()}")

    cold, warm = [], []
    for block in range(BLOCKS):
        for i in range(BLOCK):
            n = block * BLOCK + i + 1
            row = cold_run(n, block + 1)
            cold.append(row)
            print(f"e6 cold n={n} block={block+1} http={row['http']} ms={row['duration_ms']} "
                  f"vol={row['volume_state']} start={row['start_utc']} end={row['end_utc']}")
            write_csv(RAW / "e6_cold.csv",
                      cold, ["condition", "iteration", "block", "start_utc", "end_utc",
                             "duration_ms", "http", "image_cached", "volume_state",
                             "environment_id", "container_id", "error"])
        if block < BLOCKS - 1:
            print(f"e6_cold_block_gap=30s at {ts()}")
            time.sleep(BLOCK_GAP)

    # Deja el entorno en marcha antes de la serie caliente.
    print(f"e6 warm_prepare http={ensure_running()['http']} at {ts()}")
    for block in range(BLOCKS):
        for i in range(BLOCK):
            n = block * BLOCK + i + 1
            row = warm_run(n, block + 1)
            warm.append(row)
            print(f"e6 warm n={n} block={block+1} pause_http={row['pause_http']} "
                  f"pause_ms={row['pause_ms']} state={row['docker_state_after_pause']} "
                  f"resume_http={row['resume_http']} resume_ms={row['resume_ms']}")
            write_csv(RAW / "e6_warm.csv",
                      warm, ["condition", "iteration", "block", "start_utc", "end_utc",
                             "ensure_running_http", "ensure_running_ms", "pause_ms", "pause_http",
                             "docker_state_after_pause", "resume_ms", "resume_http",
                             "environment_id", "container_id", "error"])
        if block < BLOCKS - 1:
            print(f"e6_warm_block_gap=30s at {ts()}")
            time.sleep(BLOCK_GAP)

    (RAW / "e6_raw.json").write_text(json.dumps(
        {"generated_at_utc": ts(), "n_per_condition": N_RUNS, "blocks": BLOCKS, "runs_per_block": BLOCK,
         "hibernate_wait_s": HIBERNATE_WAIT, "cold": cold, "warm": warm},
        indent=2, ensure_ascii=False))
    print(f"e6_done_utc={ts()}")


if __name__ == "__main__":
    main()
