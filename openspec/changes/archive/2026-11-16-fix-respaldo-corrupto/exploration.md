## Exploration: fix-respaldo-corrupto

### Current State

**Trigger flow (Crear Respaldo Ahora).** `POST /api/v1/admin/backups/trigger`
(`router.go:359`) -> `BackupHandler.Trigger` (`backup_handler.go:99-113`) ->
`BackupService.TriggerBackup` (`backup_service.go:136-235`). The service:

1. Checks disk space (50 MB minimum, fail-closed with `backup_failed` notification).
2. Creates `solv_backup_<tenant-prefix-8>_<unix-ts>.dump.gz` in `backupDir`,
   inserts a `backup_executions` row with `status = in_progress`.
3. Writes ONLY a static header (~150 bytes) into a gzip stream — there is NO
   `pg_dump` invocation anywhere in the backend (only `exec.Command` in the
   repo is semgrep). The file is a placeholder, not a database dump.
4. Ignores all write errors (`_, _ = gzWriter.Write(...)`, `_ = gzWriter.Close()`,
   `_ = file.Close()`), then `os.Stat`s the file, stores `file_size_bytes` +
   hex SHA-256, and unconditionally marks `status = success`.

`NewBackupService` resolves `backupDir` from the explicit arg, else `BACKUP_DIR`
env, else `/tmp/solv_backups`. Production (`cmd/api/main.go:183`) passes `""`,
so the env/default applies. `/tmp` is ephemeral across container
restarts/redeploys — a row can outlive its file.

**Verify flow (Verificar).** `POST /api/v1/admin/backups/{id}/verify`
(`router.go:360`) -> `BackupHandler.Verify` (`backup_handler.go:115-137`,
wrapped in `GlobalResponse{data, error, message}` via `SendJSON`) ->
`BackupService.VerifyBackup` (`backup_service.go:255-296`), which recomputes
SHA-256 over the raw file bytes and compares with `sha256_checksum` in DB,
returning `VerifyBackupResponse{execution_id, file_name, database_checksum,
computed_checksum, is_valid, message}` inside the `data` envelope.
Missing file yields `is_valid=false` with "Archivo fisico no encontrado".

**Frontend contract mismatch (primary suspect for the red toast).**
`AdminConfigServidorService.verifyBackup` (`admin-config-servidor.service.ts:128-130`)
types the call as `post<{valid: boolean}>(.../verify)` and the component
(`admin-config-servidor.component.ts:309-321`) branches on `res.valid`.
The backend never returns a top-level `valid` field — the real shape is
`{data: {is_valid, ...}, error, message}`. So `res.valid` is always
`undefined` (falsy) and EVERY verify shows
`CORRUPTO: el checksum de <file> no coincide`, even when the backend computed
`is_valid=true`. The integration test (`slice16_backups_test.go:282-291`)
reads `body["data"]["is_valid"]` correctly — only the Angular client is wrong.
Note also `resolveError` reads `e?.error?.message`, but `SendError` puts the
human message in `message`, so backend verify/trigger errors surface as the
generic fallback string.

**Size + status display.** `formatBytes` (`admin-config-servidor.component.ts:323-328`)
renders anything under 1 MB as `X MB` with zero decimals, so the ~150-byte
placeholder header shows `0 MB`. Status badge (`component.html:237-239`) maps
`success -> Completado`, `failed -> Fallido`, else `En Progreso`. Because
trigger errors are swallowed, every placeholder row reads `Completado`,
which combined with `0 MB` and the always-corrupt toast is exactly the
reported screenshot. Filename `solv_backup_00000000_...` is consistent with
`DefaultTenantID = "00000000-0000-0000-0000-000000000001"` (`tenant.go:26`)
truncated to 8 chars — expected on single-node, not a bug by itself.

**Secondary integrity hazards (real, but not needed to explain the toast).**
Checksum is computed over the gzip byte stream via `io.MultiWriter` at write
time but the `gzWriter.Close()` error is discarded; a short write / full disk
after the 50 MB pre-check still lands as `success`. No post-write re-read,
no minimum-size guard, no `failed` transition on empty output. `VerifyBackup`
compares DB checksum vs current file bytes, so any post-creation mutation
(retention delete +同名 reuse, `BACKUP_DIR` change, `/tmp` eviction with a
stale row, partial overwrite) also yields mismatch — returned as
`is_valid=false`, never persisted back to the execution row.

Prior change `hardening-respaldos-mantenimiento` added range validation +
confirm-phrase + audit for config/maintenance; it did NOT touch dump
generation, checksum storage, or the verify client contract.

### Affected Areas
- `frontend/src/app/features/admin/configuracion/admin-config-servidor.service.ts` — `verifyBackup` unwraps the wrong shape (`{valid}` vs `{data: {is_valid}}`).
- `frontend/src/app/features/admin/configuracion/tabs/admin-config-servidor.component.ts` — branches on `res.valid` (always falsy); `formatBytes` renders sub-MB files as `0 MB`; `triggerBackup` toast claims "verificado" without calling verify.
- `backend/internal/core/services/backup_service.go` — `TriggerBackup` writes header-only placeholder, swallows all I/O errors, always marks `success`; no `pg_dump`, no size floor, no post-write verify.
- `backend/internal/delivery/http/backup_handler.go` — `Verify`/`Trigger` envelope behavior is correct but relied upon wrongly by the client; no status persistence of verify outcome.
- `backend/cmd/api/main.go` — backup dir resolves to `BACKUP_DIR` or `/tmp/solv_backups` (ephemeral); no persistent-volume guarantee.
- `backend/tests/integration/slice16_backups_test.go` — covers trigger/verify happy path against tmp dir; asserts `data.is_valid`, so it cannot catch the Angular contract bug.
- `frontend/src/app/features/admin/configuracion/tabs/admin-config-servidor.component.spec.ts` — mocks `verifyBackup` as `of({valid: true})`, mirroring the wrong contract instead of the real envelope.

### Approaches
1. **Fix verify contract + honest failure states (recommended)** — Frontend: unwrap `data.is_valid` (typed interface + envelope), fix `formatBytes` sub-MB rendering, stop claiming "verificado" on trigger. Backend: check `gzWriter.Close()`/`Write` errors, reject empty/below-floor outputs as `failed`, re-read file for checksum instead of trusting the write-stream hash.
   - Pros: kills the false CORRUPTO for healthy files; stops marking failed dumps Completado; small, reviewable diff.
   - Cons: does not produce real restorable dumps (placeholder remains).
   - Effort: Low/Medium

2. **Real pg_dump pipeline** — Shell out to `pg_dump` (custom/format + gzip), stream to file, capture stderr, fail-closed on nonzero exit, verify by re-read + `gzip -t` / restore dry-run.
   - Pros: backups become actually restorable; addresses the "why 0 MB if it came from Crear Respaldo" root expectation.
   - Cons: needs pg_dump binary in image, credential plumbing, longer trigger latency, rotation/locking design; larger change.
   - Effort: High

3. **Verify-result persistence + storage hardening only** — Persist verify outcome/status on the execution row, move `BACKUP_DIR` to a persistent volume, keep placeholder content.
   - Pros: history reflects reality; kills ephemeral-/tmp-induced mismatches.
   - Cons: leaves both the false-toast contract bug and the empty-dump problem in place; misleading on its own.
   - Effort: Medium

### Recommendation
Approach 1 as this change (`fix-respaldo-corrupto`): fix the client envelope
(`is_valid`), sub-MB size display, and fail-closed trigger (I/O error checks +
minimum-size floor + checksum from re-read). File Approach 2 (real `pg_dump`
restore-capable backups) as a follow-up change — it is a different risk class
(binary, secrets, latency) and exceeds the 400-line review budget combined
with the contract fix.

### Risks
- Users may have "verified" (red toast) known-good rows: after the fix those rows flip to OK while still containing header-only placeholders — do NOT present verify-OK as restore-capable without Approach 2.
- `BACKUP_DIR` on `/tmp` means some CORRUPTO rows may be genuinely unreadable (evicted files); the fix must keep distinguishing "file missing" from "checksum mismatch" in UI copy.
- `resolveError` shape mismatch (`error.message` vs envelope `message`) can mask backend failure reasons; fix alongside or trigger errors stay generic.
- Retention goroutine (`go applyRetentionPolicy`) deletes files + rows silently; concurrent verify during purge can report mismatch for a row that is about to disappear.

### Ready for Proposal
Yes — scope is bounded: frontend verify/size/toast copy, backend fail-closed trigger, tests mirroring the real envelope (`data.is_valid`) on both sides. Propose with: corrected `VerifyBackup` client interface, sub-MB `formatBytes` rule, minimum-size floor value, checksum-from-reread decision, and whether verify outcome is persisted to the row in this change or deferred.
