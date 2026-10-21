# Feature: Completar Refactorización Modular Pendiente

Branch: main | Delivery strategy: ask-on-risk | Test runner: go test ./... & npm run test:ci & CI=true npm run build

## Specs
- **S1**: Dividir los 10 archivos Go de backend >520 líneas (límite ADR-046) en archivos cohesivos por subdominio <520 líneas cada uno, sin romper contratos públicos ni firmas.
- **S2**: Modularizar `internal/infrastructure/docker` (`client.go` y `env_test_adapter.go`) en piezas cohesivas <520 líneas.
- **S3**: Dividir componentes Angular monolíticos en piezas presentacionales atómicas bajo Container-Presentational (`admin-teachers`, `admin-students`, `speed-grader`, `step-ide-environment`, `admin-manual`, `gradebook-export-modal`, `cases-table`).
- **S4**: Verificar `go vet ./...`, `go test ./...`, `npm run lint:styles`, `npm run test:ci` y `CI=true npm run build` en verde.

## Tasks
- [x] **T1** (S1) Mapear y dividir `backend/internal/delivery/http/admin_handler.go` (844 L → 147 L + 5 archivos de subdominio)
- [x] **T2** (S1) Mapear y dividir `backend/internal/delivery/http/evaluation_handler.go` (832 L → 386 L + 3 archivos de subdominio)
- [x] **T3** (S1) Mapear y dividir `backend/internal/core/services/admin_governance_service_test.go` (780 L → 467 L + 2 archivos de test)
- [x] **T4** (S1) Mapear y dividir `backend/internal/core/services/template_audit_worker.go` (691 L → 237 L + 3 archivos de subdominio)
- [x] **T5** (S1) Mapear y dividir `backend/internal/delivery/http/teacher_handler.go` (657 L → 97 L + 6 archivos de subdominio)
- [x] **T6** (S1) Mapear y dividir `backend/internal/infrastructure/storage/postgres/exercise_repository.go` (639 L → 319 L + 3 archivos de subdominio)
- [x] **T7** (S1) Mapear y dividir `backend/internal/core/services/format_validator.go` (584 L → 379 L + `format_validator_generate.go`)
- [x] **T8** (S1) Mapear y dividir `backend/internal/core/services/teacher_service.go` (562 L → 86 L + 6 archivos de subdominio)
- [x] **T9** (S1) Mapear y dividir `backend/internal/core/services/workspace_service.go` (526 L → 305 L + 2 archivos de subdominio)
- [x] **T10** (S1) Mapear y dividir `backend/internal/core/services/env_test_service.go` (524 L → 161 L + `env_test_job_execution.go`)
- [x] **T11** (S2) Modularizar `internal/infrastructure/docker` (`client.go` 632 L y `env_test_adapter.go` 545 L) en 6 archivos <500 L
- [x] **T12** (S3) Dividir componentes Angular grandes (`admin-teachers`, `admin-students`, `speed-grader`, `step-ide-environment`, `admin-manual`, `gradebook-export-modal`, `cases-table`)
- [x] **T13** (S4) Verificación integral: `go vet ./...`, `go test ./internal/...`, `npm run lint:styles`, `npm run test:ci` (87 suites, 622 tests OK) y `CI=true npm run build` (OK)

## Log
- **L1 (2026-10-21)**: Auditoría identificó 10 archivos Go >520 líneas, 2 archivos Docker >520 líneas y múltiples componentes Angular >500 líneas con plantillas y hojas de estilo monolíticas.
- **L2 (2026-10-21)**: Completada la refactorización modular integral:
  - Backend: 13 archivos monolíticos segregados en 35 archivos cohesivos por subdominio (<500 líneas cada uno). 0 archivos Go de código fuente superan el límite de ADR-046.
  - Frontend: `admin-teachers`, `admin-students`, `speed-grader`, `step-ide-environment`, `admin-manual`, `gradebook-export-modal` y `cases-table` divididos en subcomponentes presentacionales y módulos utilitarios puros.
  - Verificación: `go vet`, `go build`, `go test`, `lint:styles`, `test:ci` (622 tests pasando) y `CI=true npm run build` en verde.
