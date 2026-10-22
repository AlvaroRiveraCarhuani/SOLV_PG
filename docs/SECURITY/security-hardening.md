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

---

## 5. Isolation of Hidden Test Cases in Judge Responses (GOB-SEGURO-01)

The judge must never reveal the expected output of a hidden test case to the role `student`, because a single leaked value is enough to hardcode the answer and obtain an **AC** without solving the problem.

| Channel | Control |
| :--- | :--- |
| `GET /api/v1/exercises/{id}` (role `student`) | Public DTO without `test_cases`, `reference_solution`, `validation_query` or `expected_json` (`ToExercisePublicResponse`). |
| `POST /api/v1/evaluations` — `failed_test_case` | `input` and `expected_output` masked with `[OCULTO POR EXAMEN]` for hidden cases and exam mode with per-student seed. |
| `POST /api/v1/evaluations` — `case_results[].message` | Comparator detail replaced with `[CASO DE EVALUACIÓN OCULTO]` for hidden cases. The detail of the comparator embeds the expected output (`Esperado: "..."`), which used to leak hidden answers through the per-case verdict list. |
| Student UI (`student-exercise-solver`) | Renders only `public_test_cases` (`visibility = example`) and never receives the reference solution. |

* **Implementation:** `backend/internal/core/services/evaluation_algorithm_runner.go` (masking) and `backend/internal/infrastructure/docker/strategies/lang/base_runner.go` (origin of the comparator detail).
* **Regression tests:** `backend/internal/core/services/evaluation_hidden_case_isolation_test.go` (unit) and `frontend/e2e/flows/02-medium/J03-verdict-matrix-and-hidden-isolation.spec.mjs` (end-to-end, asserts on the HTTP payload and the DOM).

## 6. Custom AST Rules Enforcement (GOB-SEGURO-02)

The rules a teacher declares with the **Prohibir** button of the `ast-rules-builder` are persisted in `ast_rules.custom_rules`. The static pre-check evaluates them before the container is created, so the enumeration the teacher sees in the UI is the one that decides the verdict.

| Control | Detail |
| :--- | :--- |
| Matching | Whole identifiers only (never raw substrings): prohibiting `sorted` does not fire on `is_sorted` nor on `sorted_list`. Dotted patterns (`math.sqrt`) match the qualified name. |
| False positives | Comments and string literals are stripped before tokenizing, so a commented mention or a literal does not trigger the rule. |
| Scope | A rule applies only to the language it was declared for. A rule with several comma-separated patterns is evaluated pattern by pattern. |
| Verdict | `AST_BLOCKED` with `execution_time_ms = 0` and a structured `actual_json` (`rule_id` = `custom_rule:<language>:<type>:<pattern>`, `pattern`, `target`, `line`, `message`) so the teacher can audit which rule was violated. |
| Ordering | The platform Semgrep barrier runs first and keeps its own message and `actual_json`; the custom rules are evaluated right after it. |

* **Implementation:** `backend/internal/core/services/ast_custom_rules.go` (lexer and matcher), `backend/internal/core/services/evaluation_algorithm_runner.go` (pre-check step) and `backend/internal/core/domain/ports.go` (`ASTAnalyzer.ValidateCustomRules`).
* **Regression tests:** `backend/internal/core/services/ast_custom_rules_test.go` (unit) and `frontend/e2e/flows/02-medium/J04-ast-security-barrier.spec.mjs` (end-to-end).

## 7. Verdict `AST_VIOLATION` in the UI (GOB-SEGURO-03)

`AST_VIOLATION` is the verdict emitted by the static regex barrier (the `block_native_sort` and `block_system_modules` policies). It is part of `JudgeVerdict` and the badge renders it with the same blocking semantics as `AST_BLOCKED` (shield icon, `verdict-ast` class, human label "Regla de código"), so it never degrades to "Sin evaluar".

* **Implementation:** `frontend/src/app/core/models/verdict.model.ts` and `frontend/src/app/shared/components/verdict-badge/verdict-badge.component.ts`.
* **Regression tests:** `frontend/src/app/shared/components/verdict-badge/verdict-badge.component.spec.ts` (unit) and `frontend/e2e/flows/02-medium/J04-ast-security-barrier.spec.mjs` (end-to-end).

## 8. Isolation of Seeded Exam Cases (GOB-SEGURO-04)

In exams with `per_student_seed` the generated input is derived from `sha256(exercise:student)`, so every student gets a different variant. The expected output of those cases is the answer of the exercise and must never reach the student, while the teacher still needs it to review the submission.

| Channel | Control |
| :--- | :--- |
| `POST /api/v1/evaluations` (role `student`) | `StudentEvaluationResponse`: `generated_cases` without `expected_output`, `failed_test_case` without the expected output of hidden cases. |
| WebSocket `EVALUATION_COMPLETED` | Emits the same sanitized DTO. |
| `POST /api/v1/submissions` | The client payload is not trusted: the server persists the authoritative snapshot computed during the evaluation (`evaluation_generated_cases`). |
| `GET /api/v1/submissions/{id}` (role `student`) | `403` when the submission belongs to another student; the generated cases are sanitized. |
| `GET /api/v1/exercises/{id}/submissions` (role `student`) | The generated cases of the listed submissions are sanitized. |

* **Implementation:** `backend/internal/delivery/http/dto/evaluation_dto.go`, `backend/internal/core/services/evaluation_algorithm_runner.go` (snapshot persistence), `backend/internal/core/services/submission_service.go` and `backend/internal/infrastructure/storage/postgres/generated_case_snapshot_repository.go` (migration `00025`).
* **Regression tests:** `backend/internal/delivery/http/dto/evaluation_dto_test.go` (unit) and `frontend/e2e/flows/03-hard/J05-curricular-unlock-and-exam-seed.spec.mjs` (end-to-end).

### Closed gaps

| Id | Gap | Status |
| :--- | :--- | :--- |
| GOB-SEGURO-02 | The `ast-rules-builder` "Prohibir" rules were persisted as `ast_rules.custom_rules` without being evaluated by the static analyzer. | **Closed** — see section 6. |
| GOB-SEGURO-03 | The verdict `AST_VIOLATION` was not part of `JudgeVerdict` in the frontend, so the badge rendered it as "Sin evaluar". | **Closed** — see section 7. |
