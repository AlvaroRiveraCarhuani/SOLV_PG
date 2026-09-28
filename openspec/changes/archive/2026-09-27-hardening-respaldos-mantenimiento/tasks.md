# Tasks: Hardening Respaldos + Mantenimiento

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 280-350 |
| 400-line budget risk | Low |
| Chained PRs recommended | No |
| Suggested split | Single PR |
| Delivery strategy | ask-on-risk |
| Chain strategy | pending |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: pending
400-line budget risk: Low

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| 1 | Backend fail-closed validation + audit + lazy-clear | PR 1 (single) | `go test ./internal/core/services/ -run 'Backup|Maintenance' -count=1` | N/A (no new endpoint; covered by integration tests) | `backup_service.go`, `admin_academic_service.go`, handlers, middleware |
| 2 | Frontend gates + warnings on servidor tab | PR 1 (single) | `npx vitest run admin-config-servidor` | Manual: servidor tab, type invalid range + phrase | `admin-config-servidor.component.ts/.html` only |

## Phase 1: Backend validation - RED then GREEN

- [x] 1.1 RED: add table tests for `UpdateConfig` edges 0/1/168/169 and 0/1/365/366 in `backend/internal/core/services/backup_service_test.go`
- [x] 1.2 GREEN: add `BackupValidationError` in `backup_service.go`, strict checks, map via `errors.As` to 422 in `backup_handler.go`
- [x] 1.3 RED: add tests for phrase mismatch/casing, motive 9/10, past `until`, lazy-clear in `admin_academic_service_test.go`
- [x] 1.4 GREEN: fix `EnableMaintenanceDTO` in `admin_academic.go` (drop `required`, add `ConfirmPhrase`), validate + audit `MAINTENANCE_*` in service/handler, lazy-clear in `maintenance_middleware.go` and `GetStatus`

## Phase 2: Frontend gates and warnings

- [x] 2.1 Add `backupFrequencyError/backupRetentionError/canSaveBackup/purgeCount/confirmDisabled/vigenciaWarning` computeds in `admin-config-servidor.component.ts` (`inject()`, signals only)
- [x] 2.2 Bind `[error]`, `max="365"`, Save gate, phrase input, purge/vigencia warnings in `admin-config-servidor.component.html` (`@if`, `modal-shell`/`form-field`, tokens only)
- [x] 2.3 Pass `confirm_phrase` through in `admin-config-servidor.service.ts` without loosening types

## Phase 3: Verification

- [x] 3.1 Extend `backend/tests/integration/slice16_backups_test.go` (422 codes) and `slice14_maintenance_periods_test.go` (422 + audit verbatim + expiry reads off); run `go test ./... -count=1`
- [x] 3.2 Extend `admin-config-servidor.component.spec.ts` (invalid blocks save + no toast, phrase blocks, warnings show count); run vitest file
- [x] 3.3 Run `npm run lint:styles` for token-only SCSS; confirm toasts green `#15803D` / red only
