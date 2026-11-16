# Tasks: fix-respaldo-corrupto

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 379 (PR1) + about 280 (PR2) + 140 (PR3); each slice stays below 400 |
| 400-line budget risk | High for the combined change; Low per planned PR slice |
| Chained PRs recommended | Yes |
| Suggested split | PR1 backend implementation; PR2 backend test-evidence closure; PR3 frontend contract and UI |
| Delivery strategy | ask-on-risk |
| Chain strategy | stacked-to-main |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: High

### Suggested Work Units

| Unit | Goal and boundary | Likely PR / target | Changed-line estimate | Focused test command | Runtime harness | Rollback boundary |
|------|------------------|-------------------|----------------------|----------------------|-----------------|-------------------|
| 1 | Backend persistence, fail-closed trigger, and initial integration coverage; starts at the current backend baseline and ends with schema, repository, service, and Slice16 behavior implemented. | PR1 → `main` | 379 (recorded backend implementation including migration 00008) | `cd backend && go test ./tests/integration -run 'TestSlice16_Backups' -count=1` | Live HTTP + PostgreSQL Slice16 integration; fixture bytes only, no real `pg_dump` | Roll back migration 00008 and only backup-specific hunks in domain, repository, service, and Slice16 files. |
| 2 | Close task 2.1 evidence gaps with gzip destination Write/Close fault tests and re-read checksum assertions; reconcile proposal, task forecast, and cumulative apply progress. | PR2 → PR1, then `main` after PR1 merges | About 280 (about 180 current service/test lines plus about 100 planning-record edits) | `cd backend && go test ./internal/core/services -run 'TestBackupService_TriggerBackup(FailsClosedOnGzipIOErrors|ChecksumsReReadBytes)' -count=1` | Run the focused Slice16 HTTP + PostgreSQL integration after unit coverage; no real `pg_dump` | Remove `backup_service_io_test.go`, revert only the narrow test seams in `backup_service.go`, and revert this apply's proposal/tasks/apply-progress edits. Preserve PR1 behavior. |
| 3 | Frontend envelope branch, outcome display, sub-MB formatting, honest trigger copy, and style validation; starts after backend PR2 and ends with client tests/lint. | PR3 → PR2, then `main` after PR2 merges | 140 observed changed lines in the four task-listed Angular files; estimate only, not frontend apply evidence | `cd frontend && npx ng test` and `cd frontend && npm run lint:styles` | Angular tests use mocked API envelopes; no live backend runtime boundary | Revert only the four task-listed Angular service/component/template/spec files. |

Stack: `PR1 (main) → PR2 (PR1; main after PR1 merges) → PR3 (PR2; main after PR2 merges)`. The frontend slice must remain after the backend slices. Estimates count the scoped implementation and the current task-listed frontend diff; unrelated dirty worktree files are excluded. Each PR remains below the 400 changed-line cap. No PRs or commits are created by this apply.

## Phase 1: Foundation (migration + domain)

- [x] 1.1 Create `backend/migrations/00008_backup_verify_outcome.sql` with idempotent nullable `last_verify_ok`, `last_verify_at`
- [x] 1.2 Add `LastVerifyOK *bool`, `LastVerifyAt *time.Time` to `BackupExecution` in `backend/internal/core/domain/backup.go`
- [x] 1.3 Extend SELECT/UPDATE column lists in `backend/internal/infrastructure/storage/postgres/backup_repository.go`

## Phase 2: Core (trigger gates + verify persist)

- [x] 2.1 RED: table tests cover gzip destination Write/Close errors, below-floor rejection, and checksum from completed-file re-read in backend service tests and `backend/tests/integration/slice16_backups_test.go`
- [x] 2.2 GREEN: staged gates in `backend/internal/core/services/backup_service.go` (`MinBackupSizeBytes=1024`, partial cleanup, re-read checksum)
- [x] 2.3 GREEN: `VerifyBackup` persists only `last_verify_*` (never flips `status`/`is_valid` on persist failure)

## Phase 3: Integration (Angular client + view)

- [x] 3.1 RED: envelope `{data:{is_valid}}` OK/missing/mismatch mocks in `tabs/admin-config-servidor.component.spec.ts`
- [x] 3.2 GREEN: typed `VerifyBackupResult` + envelope `verifyBackup` + envelope-first `resolveError` in `admin-config-servidor.service.ts`
- [x] 3.3 GREEN: `data.is_valid` branch, three-state copy, `formatBytes` KB ladder, honest toast in `tabs/admin-config-servidor.component.ts`
- [x] 3.4 GREEN: verify-state indicator bound to `last_verify_ok/at` in `tabs/admin-config-servidor.component.html`

## Phase 4: Verification

- [x] 4.1 Go: tampered vs missing outcomes + reload-keeps-outcome pass (`slice16_backups_test.go`)
- [x] 4.2 Angular: 150 B renders KB never `0 MB`, toast has no verified claim (`component.spec.ts`) — 18/18 tests pass (`ng test --include='**/admin-config-servidor.component.spec.ts'`)
- [x] 4.3 Lint: `npm run lint:styles` (no hex/font literals; signals, `@if/@for`, `inject()`, clean selectors) — OK, 0 componentes con styles inline
