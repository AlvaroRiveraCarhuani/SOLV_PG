# Feature: Lint Harness (golangci-lint + ESLint/angular-eslint)
Objetivo: agregar determinismo al control de calidad de SOLV_PG. Branch: feat/lint-harness. Runner: `make lint:backend` (Go) y `npm run lint:ts` (frontend).
Test-first: N/A — tooling de lint; la verificación funcional es la propia ejecución del linter y `go build ./...` / `ng build` de respaldo.

## Specs
S1 (verbatim usuario): "queria agregar mejor control cuando desarrolle codigo en SOLV_PG, tipo aveces se arman muchas inoconsistencias, seguridad, iu/ux"
S2: Instalar golangci-lint (v2, config `.golangci.yml` en backend/) con linters de corrección y seguridad (errcheck/gosec/errorlint/bodyclose/noctx entre ellos) y dejarlo pasando sobre el código actual sin desactivar reglas de seguridad.
S3: Instalar ESLint flat config con typescript-eslint + angular-eslint en frontend/, dejándolo pasando sobre el código actual sin desactivar reglas de seguridad/correctitud.
S4: Wiring en CI (.github/workflows/ci.yml, job lint): agregar pasos `golangci-lint run` (backend) y `npm run lint:ts` (frontend).
S5: Ambos linters documentados en CONTRIBUTING.md como comando local obligatorio antes de commitear.

## Tasks
- T1 (S2, delegated backend): golangci-lint v2.15.0 + backend/.golangci.yml + make lint\:backend — HECHO. 747→88 hallazgos (gosec 59 + SA1019 29 restantes, report-only). go build/vet exit 0. Spot-check padre confirmado.
- T2 (S3, delegated frontend): ESLint 10.13 + typescript-eslint + angular-eslint 22.6 + frontend/eslint.config.mjs + npm run lint:ts — HECHO (parcial). 297→205 hallazgos (172 no-explicit-any, 24 no-output-native, 8 prefer-control-flow, 1 no-input-rename). tsc app+spec exit 0. Sin reglas desactivadas.
- T3 (S4-S5, inline): wiring CI + docs — HECHO. ci.yml: pasos golangci-lint y lint:ts con continue-on-error (decisión usuario: no bloqueante temporal). CONTRIBUTING.md: comandos locales documentados. YAML validado.
- Commits work-unit en feat/lint-harness: backend / frontend / CI+docs (ver git log).
- Próximo (fuera de este feature): triage del backlog — 59 gosec (prioridad: G124 cookies, G402 InsecureSkipVerify, G120 multipart), migración Docker SDK SA1019 (requiere go.mod), 172 any (activar strict en tsconfig incremental), 24 no-output-native + 8 prefer-control-flow (refactor de API de componentes).

## Log
L1 (verbatim): "bueno l oque queria en si es el tema de que queria agregar mejor control cuando desarrolle codigo en SOLV_PG, tipo aveces se arman muchas inoconsistencias, seguridad, iu/ux , que deberia de implementar, harnes? mas skills? algo de un repositorio de github que sea util?"
L2 (verbatim): "dale, entonces avmos por los 4 puntos o que, empieza con el punto 1 y 2"
