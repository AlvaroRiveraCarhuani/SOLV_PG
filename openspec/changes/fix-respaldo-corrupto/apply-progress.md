# Apply Progress: fix-respaldo-corrupto

## Current work unit: PR2 backend evidence closure

This corrective apply completed the missing backend evidence and reconciled the planning contract. No frontend tasks were implemented. Tasks 3.1-3.4 and 4.2-4.3 remain pending for PR3.

### Cumulative completed backend tasks

- [x] 1.1 Added idempotent nullable `last_verify_ok` and `last_verify_at` migration columns.
- [x] 1.2 Added nullable verification outcome fields to `BackupExecution`.
- [x] 1.3 Persisted and loaded verification fields through the backup repository.
- [x] 2.1 Covered gzip destination Write and Close failures, below-floor rejection, and checksum calculation from completed-file re-read.
- [x] 2.2 Preserved fail-closed staged trigger gates, the 1024-byte floor, partial-file cleanup, and checksum re-read.
- [x] 2.3 Persisted verify outcomes separately without changing execution status.
- [x] 4.1 Verified missing and tampered outcomes, persistence after list reload, and unchanged execution status.

### TDD cycle evidence

| Task | Test file | Layer | Safety net | RED | GREEN | TRIANGULATE | REFACTOR |
|---|---|---|---|---|---|---|---|
| 1.1-1.3 | `backend/tests/integration/slice16_backups_test.go` | PostgreSQL integration | Existing backend suite | Inherited from the interrupted apply; not repeated in this corrective unit | Focused and full backend suites passed after migration 00008 | Missing/tampered results and list reload cover distinct outcomes | No production changes in this corrective unit |
| 2.1 | `backend/internal/core/services/backup_service_io_test.go`, `backend/internal/core/services/backup_service_test.go`, `backend/tests/integration/slice16_backups_test.go` | Unit + integration | `go test ./internal/core/services -run 'TestBackupService_TriggerBackup' -count=1` and Slice16 integration both passed before edits | New tests failed to compile with undefined `newGzipWriter` and `readBackupFile` fields | Focused tests passed; gzip destination Write/Close failures persist `failed` without checksum; injected completed-file bytes determine the stored checksum | Two separate I/O fault cases and two distinct re-read payloads pass; below-floor and content-source error cases remain covered | Simplified test path construction with `filepath.Join`, removed the unused stream hasher, and reran focused service and Slice16 tests successfully |
| 2.2 | `backend/internal/core/services/backup_service.go` | Unit + integration | Existing backup service and Slice16 tests passed before edits | Prior RED from task 2.1 established the missing error-path behavior | Fail-closed gates remain enabled; fixture output succeeds only when compressed size meets 1024 bytes | Write error, Close error, below-floor, and re-read paths are exercised | Narrow writer/read functions default to `gzip.NewWriter` and `os.ReadFile`; tests pass |
| 2.3, 4.1 | `backend/tests/integration/slice16_backups_test.go` | PostgreSQL integration | Migration 00008 already applied to `solv_test` | Prior integration failures exposed missing schema setup and broken fixture assumptions | Missing/tampered outcomes, persistence reload, and status invariance pass | Missing and tampered responses use distinct checksums/messages | Typed SQL result structs and PostgreSQL placeholders retained |

### Work unit evidence

- Strict TDD mode: enabled (`openspec/config.yaml`, `testing-capabilities` cache).
- Focused service command: `cd backend && go test ./internal/core/services -run 'TestBackupService_TriggerBackup(FailsClosedOnGzipIOErrors|ChecksumsReReadBytes)' -count=1` — PASS. Four table scenarios cover gzip destination Write failure, gzip Close failure, and two distinct completed-file re-read byte sequences.
- Relevant service command: `cd backend && go test ./internal/core/services -run 'TestBackupService_TriggerBackup' -count=1` — PASS.
- Runtime harness: `cd backend && go test ./tests/integration -run 'TestSlice16_Backups' -count=1` — PASS (`solv-backend/tests/integration`, live HTTP + PostgreSQL path, 0.206s).
- Full backend runner: previously passed in the prior backend slice (`cd backend && go test ./... -count=1`, integration package completed in 186.660s); not repeated because the current changes are isolated service-I/O tests/seams and the focused runtime harness passes.
- Test database setup: migration 00008 had already been applied to `solv_test`; no schema changes in this corrective unit.
- Fixture contract: integration uses deterministic SQL-comment payloads to exceed the compressed-file floor; this is test data, not a real PostgreSQL dump. The default header-only source continues to fail closed below 1024 bytes.
- Rollback boundary: remove `backend/internal/core/services/backup_service_io_test.go` and revert only the new gzip-writer/read-file seams and their call sites in `backup_service.go`. Revert this unit's proposal/tasks/apply-progress changes only. Preserve the existing PR1 implementation and unrelated dirty hunks.

### PR boundary

- Strategy: chained PRs, `stacked-to-main`; no PR or commit was created.
- PR1: backend persistence, fail-closed trigger, and initial integration coverage; recorded authored diff 379 lines including migration 00008, under the 400-line cap.
- PR2 (current): backend gate-evidence closure plus narrow planning reconciliation; about 180 source/test lines and estimated planning-record edits about 100 lines, about 280 total.
- PR3: frontend envelope/UI and Angular verification; current task-listed frontend working diff is 140 lines, estimate only and not implementation evidence from this apply. PR3 targets PR2 until it merges, then `main`.
- Dependency chain: `PR1 (main) → PR2 (PR1; main after PR1 merges) → PR3 (PR2; main after PR2 merges)`.
- Out of scope: frontend changes in this apply, real `pg_dump`, commits, and PR creation.

### Deviations and risks

- The injectable content source remains test data only; the default source remains header-only. The 1024-byte fail-closed floor is unchanged.
- Added narrow internal seams for constructing the gzip writer and reading the completed file so tests can deterministically inject output Write/Close faults and prove checksum input. Production defaults remain the standard gzip writer and `os.ReadFile`.
- The proposal now includes verify-outcome persistence on the execution row, matching the authoritative spec, design, and tasks. The spec requirement was not weakened.
- Existing frontend and unrelated backend dirty files were preserved and not edited.
