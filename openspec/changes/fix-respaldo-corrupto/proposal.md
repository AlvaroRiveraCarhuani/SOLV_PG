# Proposal: fix-respaldo-corrupto

## Intent

Verificar shows CORRUPTO on healthy rows and trigger marks header-only files Completado at 0 MB. Restore honest backup status without claiming restorability.

## Scope

### In Scope
- Verify client unwraps `data.is_valid` envelope (typed interface); UI separates missing-file from checksum-mismatch copy.
- `formatBytes` renders sub-MB sizes (KB/B with one decimal); trigger toast drops the false verified claim.
- `TriggerBackup` fail-closed: check gzip Write/Close errors, enforce minimum-size floor, compute checksum by file re-read, mark `failed` on violation.
- Persist each verify outcome (`is_valid` and timestamp) on the backup execution row without changing execution status.
- `resolveError` reads envelope `message`; frontend spec and Go integration test assert the real `data.is_valid` shape.

### Out of Scope
- Real `pg_dump` execution and restore pipeline (separate change, different risk class).
- Moving `BACKUP_DIR` off ephemeral `/tmp` to a persistent volume.
- Retention-purge concurrency redesign.

## Capabilities

### New Capabilities
- `backup-verify-contract`: verify envelope handling, UI states (ok/mismatch/missing), sub-MB size display, fail-closed trigger rules.

### Modified Capabilities
- None (no main spec under `openspec/specs/` covers backups; prior `backup-config-validation` delta is untouched).

## Approach

Fix the client contract first (`res.valid` to `data.is_valid`), then harden trigger I/O (error checks plus size floor plus re-read checksum), and persist verify outcomes on the execution row. No dump-content change in this slice.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `frontend/.../admin-config-servidor.service.ts` | Modified | Typed verify response, envelope unwrap, error message path |
| `frontend/.../tabs/admin-config-servidor.component.ts` | Modified | `is_valid` branch, `formatBytes` sub-MB, honest trigger toast |
| `backend/internal/core/services/backup_service.go` | Modified | Fail-closed trigger, size floor, checksum from re-read |
| `backend/migrations/00008_backup_verify_outcome.sql` + backup domain/repository | Modified | Persist verify outcome and timestamp on each execution row |
| `backend/tests/integration/slice16_backups_test.go` | Modified | Failure-path coverage (I/O error, below-floor, missing file) |
| `frontend/.../tabs/admin-config-servidor.component.spec.ts` | Modified | Mock real envelope shape |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Verify-OK misread as restore-capable (files still header-only) | High | Toast and row copy state integrity only, never restorability |
| `/tmp` eviction keeps real missing-file rows | Med | Keep missing vs mismatch copy distinct |
| Concurrent verify during retention purge | Low | Persist the latest completed verify outcome on the execution row; accept transient file races |

## Rollback Plan

Revert the change branch and migration 00008 if schema rollback is approved; otherwise leave the nullable verify columns unused. Prior behavior (always-CORRUPTO toast, header-only success rows) returns.

## Dependencies

- None. No new binaries, secrets, or volumes.

## Success Criteria

- [ ] Healthy file verifies OK; missing file and tampered file each show their own copy, with each outcome persisted on the execution row.
- [ ] Trigger with failed write or below-floor output marks `failed`, never `Completado`.
- [ ] Sub-MB files never render as `0 MB`.
- [ ] Backend and frontend tests assert `data.is_valid`; diff stays within 400-line budget.

## Proposal question round

Assumptions needing review: size floor value (proposed 1 KB placeholder guard, real dumps larger); verify outcome is persisted on the execution row; no copy may promise restorability. Correct or request a second round.
