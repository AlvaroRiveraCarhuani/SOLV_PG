#!/usr/bin/env python3
"""Firma cookies de sesión (JWT HS256) para los estudiantes de la campaña v2.

Reutiliza el mismo contrato de claims que backend/cmd/gentoken: user_id, email,
role, tenant_id, jti, exp. El secreto se lee de .env y nunca se imprime.
Salida: raw/tokens-campaign.json con la forma {indice: {token, student_id}}.
"""
import base64
import hashlib
import hmac
import json
import os
import secrets
import subprocess
import sys
import time
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[4]
RAW = Path(__file__).resolve().parents[1] / "raw"
TENANT = "00000000-0000-0000-0000-000000000001"
DB_CONTAINER = "solv_db"


def b64url(data: bytes) -> str:
    return base64.urlsafe_b64encode(data).rstrip(b"=").decode()


def read_secret() -> str:
    env_path = REPO_ROOT / ".env"
    for line in env_path.read_text().splitlines():
        line = line.strip()
        if line.startswith("JWT_SECRET="):
            return line.split("=", 1)[1].strip().strip('"').strip("'")
    raise SystemExit("JWT_SECRET no está definido en .env")


def psql(sql: str) -> str:
    out = subprocess.run(
        ["docker", "exec", DB_CONTAINER, "sh", "-lc",
         'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -tA -F"|" -c ' + json.dumps(sql)],
        capture_output=True, text=True, check=True,
    )
    return out.stdout.strip()


def mint(secret: str, user_id: str, email: str, role: str, days: int = 30) -> str:
    now = int(time.time())
    header = {"alg": "HS256", "typ": "JWT"}
    payload = {
        "user_id": user_id,
        "email": email,
        "role": role,
        "tenant_id": TENANT,
        "jti": secrets.token_hex(16),
        "exp": now + days * 86400,
    }
    signing_input = b64url(json.dumps(header, separators=(",", ":")).encode()) + "." + \
                    b64url(json.dumps(payload, separators=(",", ":")).encode())
    sig = hmac.new(secret.encode(), signing_input.encode(), hashlib.sha256).digest()
    return signing_input + "." + b64url(sig)


def main() -> None:
    secret = read_secret()
    rows = [r for r in psql(
        "SELECT id, email FROM users "
        "WHERE email LIKE 'campaign-v2-s%@test.local' OR email = 'campaign-v2-ide@test.local' "
        "ORDER BY email").splitlines() if r]
    if not rows:
        raise SystemExit("no hay usuarios de campaña: ejecuta 01_seed_campaign.sql")

    tokens, ide = {}, None
    for row in rows:
        uid, email = row.split("|", 1)
        entry = {"token": mint(secret, uid, email, "student"), "student_id": uid, "email": email}
        if email == "campaign-v2-ide@test.local":
            ide = entry
        else:
            tokens[str(len(tokens) + 1)] = entry

    RAW.mkdir(parents=True, exist_ok=True)
    payload = {"generated_at_utc": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
               "expires_in_days": 30, "students": tokens}
    (RAW / "tokens-campaign.json").write_text(json.dumps(tokens, indent=2, ensure_ascii=False))
    (RAW / "token-ide.json").write_text(json.dumps(ide, indent=2, ensure_ascii=False))
    print(f"tokens firmados: {len(tokens)} estudiantes de aula + ide={'sí' if ide else 'no'}")


if __name__ == "__main__":
    sys.exit(main())
