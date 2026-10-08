# Security Hardening & Vulnerability Mitigation

## Overview

This document specifies the security controls implemented in the SOLV platform to ensure production readiness, resource protection, protection against brute-force attacks, and prevention of web application security vulnerabilities.

---

## 1. Rate Limiting Middleware

Rate limiting is enforced at two distinct levels using token-bucket rate limiters (`golang.org/x/time/rate`).

### Global Rate Limiting (IP-based)
- **Limit:** 100 requests per minute per IP address (`~1.66 req/sec`).
- **Burst:** 20 requests.
- **Behavior when exceeded:** Returns `HTTP 429 Too Many Requests` with response header `Retry-After: 60` and JSON payload `{"error": "Too Many Requests"}`.

### Authentication Failure Protection (Brute-Force Mitigation)
- **Limit:** Maximum 5 failed login/callback attempts per IP within a 15-minute sliding window.
- **Behavior when exceeded:** The IP address is blocked for 15 minutes.
- **Response:** `HTTP 429 Too Many Requests` with header `Retry-After: 900` and message `{"error": "Demasiados intentos fallidos de autenticación. Bloqueado temporalmente."}`.

### Submission Rate Limiting (Student-based)
- **Limit:** Maximum 10 submissions per minute per authenticated student ID (or IP fallback).
- **Burst:** 5 submissions.
- **Target Route:** `POST /api/v1/submissions`.
- **Behavior when exceeded:** Returns `HTTP 429 Too Many Requests` with header `Retry-After: 60` and message `{"error": "Límite de envíos por minuto excedido (máximo 10/minuto)."}`.

---

## 2. HTTP Security Headers Middleware

Every outgoing HTTP response from the backend service passes through `SecurityHeadersMiddleware`, injecting the following mandatory security headers:

| Header | Value | Purpose |
| :--- | :--- | :--- |
| `Content-Security-Policy` | `default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'` | Mitigates XSS and unauthorized dynamic code loading. |
| `Strict-Transport-Security` | `max-age=31536000; includeSubDomains` | Enforces HTTPS connections for 1 year (HSTS). |
| `X-Content-Type-Options` | `nosniff` | Prevents MIME-type sniffing by browsers. |
| `X-Frame-Options` | `DENY` | Prevents clickjacking by blocking iframe rendering. |
| `X-XSS-Protection` | `1; mode=block` | Enables legacy browser cross-site scripting filters. |
| `Referrer-Policy` | `strict-origin-when-cross-origin` | Limits referrer information leakage to external domains. |

---

## 3. Strict CORS Middleware

CORS policies are enforced via `CORSMiddleware` in `backend/internal/delivery/http/middleware/cors.go`:

- **Configuration:** Allowed origins are loaded from the environment variable `ALLOWED_ORIGINS` (comma-separated).
- **Default (Development):** If `ALLOWED_ORIGINS` is unset, default local development origins (`http://localhost:4200`, `http://127.0.0.1:4200`, `http://localhost:3000`, `http://127.0.0.1:3000`) are accepted.
- **Validation:** Incoming requests with an `Origin` header that do not match the allowed origin list are rejected with `HTTP 403 Forbidden`.
- **Preflight:** HTTP `OPTIONS` requests are handled cleanly, returning standard CORS headers and `HTTP 200 OK`.

---

## 4. Vulnerability Scanning Workflow

Security scanning is automated via `scripts/security-scan.sh`:

1. **Go Static Analysis:** Runs `govulncheck ./...` (or `go vet ./...` fallback) to detect vulnerable Go dependencies.
2. **Node.js Audit:** Runs `npm audit --audit-level=high` in `frontend/` to identify vulnerable packages.
3. **Container Image Inspection:** Scans runner base images (`python:3.11-slim`, `openjdk:21-slim`, `gcc:13`) using `trivy` or `docker scout`.
