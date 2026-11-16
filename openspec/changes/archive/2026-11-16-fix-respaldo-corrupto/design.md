# Design: fix-respaldo-corrupto

## Technical Approach

Fix the client envelope first (`data.is_valid`, typed), then harden `TriggerBackup` fail-closed, then persist verify outcomes on the execution row. `Verify`/`Trigger` handlers are unchanged (the `GlobalResponse` envelope is already correct). Real `pg_dump` stays out of this change.

## Architecture Decisions

| Decision | Options | Tradeoff | Choice |
|---|---|---|---|
| Verify persistence location | (a) execution-row columns (b) new audit table (c) unpersisted | (b) needs new table + repo + read path, breaks the 400-line budget and dangles after retention deletes; (c) violates the spec + user confirmation | (a): nullable `last_verify_ok` / `last_verify_at` on `backup_executions`. Spec needs only id + `is_valid` + timestamp; row keeps list reload free of re-verify |
| Envelope fix placement | (a) typed service method (b) inline `any` in component | (b) repeats the original bug class (untyped `res.valid`) | (a): `verifyBackup` returns `Observable<ApiEnvelope<VerifyBackupResult>>`; component branches only on `res.data.is_valid` |
| `resolveError` shape | (a) keep `e.error.message` only (b) envelope-first chain | (a) misses transport-level `message` and surfaces the `error` code string | (b): `e.error.message` (envelope) → `e.message` → fallback. Note `e.error.message` already hits the envelope `message`; the gap is the missing fallbacks |
| `formatBytes` sub-MB | (a) B/KB/MB/GB ladder, 1 decimal (b) bytes-only below 1 MB | (b) unreadable for KB-range files | (a): `<=0 → 0 B`, `<1 KB → N B`, `<1 MB → N.N KB`, else `N.N MB/GB`. A non-empty file can never render `0 MB` |
| Trigger hardening order | single combined check vs staged gates | combined hides which gate failed | staged: Write err → Close err → Stat → 1024 B floor → re-read checksum. First failure marks `failed`, removes the partial file, stores no success checksum |
| Checksum source | MultiWriter stream hash vs file re-read | stream hash attests bytes sent, not bytes on disk | re-read via `os.ReadFile` after Close; success rows always have a complete on-disk file meeting the floor |
| Missing vs mismatch signal | (a) new backend reason code (b) existing shape | (a) costs a contract version for zero new information | (b): `computed_checksum === ""` means missing; `is_valid=false` with non-empty means mismatch. Explicit, tested, no shape change |
| Copy language | restore-flavored vs integrity-only | restore-flavored lies while output is header-only | integrity-only everywhere; no copy may promise restorability |

## Data Flow

Trigger:
`Component.triggerBackup ──→ POST /trigger ──→ BackupService.TriggerBackup ──→ staged gates ──→ row success|failed ──→ honest toast`

Verify:
`Component.verifyBackup ──→ POST /{id}/verify ──→ VerifyBackup (re-read + compare) ──→ persist last_verify_* ──→ {data:{is_valid}} ──→ branch OK|missing|mismatch`

List reload reads `last_verify_*` from the row; no re-verify.

## File Changes

| File | Action | Description |
|---|---|---|
| `backend/migrations/00008_backup_verify_outcome.sql` | Create | Idempotent `ADD COLUMN IF NOT EXISTS` for `last_verify_ok`, `last_verify_at` |
| `backend/internal/core/domain/backup.go` | Modify | `LastVerifyOK *bool`, `LastVerifyAt *time.Time` (pointers preserve NULL = never verified) |
| `backend/internal/infrastructure/storage/postgres/backup_repository.go` | Modify | Extend SELECT/UPDATE column lists with the two verify columns |
| `backend/internal/core/services/backup_service.go` | Modify | Trigger gates + `MinBackupSizeBytes = 1024` + re-read checksum + partial-file cleanup; `VerifyBackup` persists outcome without touching `status` |
| `frontend/.../admin-config-servidor.service.ts` | Modify | `VerifyBackupResult` interface, envelope-typed `verifyBackup`, envelope-first `resolveError` |
| `frontend/.../tabs/admin-config-servidor.component.ts` | Modify | `data.is_valid` branch, three-state copy, sub-MB `formatBytes`, honest trigger toast |
| `frontend/.../tabs/admin-config-servidor.component.html` | Modify | Verify-state indicator bound to `last_verify_ok/at` (NULL = never verified) |
| `backend/tests/integration/slice16_backups_test.go` | Modify | Below-floor → `failed`; missing/tampered → distinct outcomes + persisted columns |
| `frontend/.../tabs/admin-config-servidor.component.spec.ts` | Modify | Mocks use the real `{data:{is_valid}}` envelope; size + copy cases |

## Interfaces / Contracts

```go
const MinBackupSizeBytes int64 = 1024 // placeholder guard; real dumps are larger
type BackupExecution struct {
    LastVerifyOK *bool      `db:"last_verify_ok" json:"last_verify_ok,omitempty"`
    LastVerifyAt *time.Time `db:"last_verify_at" json:"last_verify_at,omitempty"`
}
```

```ts
interface VerifyBackupResult { execution_id: string; file_name: string; database_checksum: string; computed_checksum: string; is_valid: boolean; message: string; }
verifyBackup(id: string): Observable<ApiEnvelope<VerifyBackupResult>>
```

Persist rule: verify writes only the two verify columns; `status` stays a trigger concern. A failed persist must not flip `is_valid` (verify is a read path).

## Testing Strategy

| Layer | What to Test | Approach |
|---|---|---|
| Unit (Go) | gate order: Write/Close err, below-floor, re-read checksum | table test on `TriggerBackup` with temp dir |
| Integration (Go) | missing file vs tampered file outcomes + persisted `last_verify_*`; list reload keeps outcome | extend `slice16_backups_test.go` against tmp dir |
| Unit (Angular) | envelope branch OK/missing/mismatch; `formatBytes` 150 B → KB, never `0 MB`; trigger toast has no verified claim | `component.spec.ts` with real envelope mocks |

## Threat Matrix

N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary. File I/O uses service-generated paths only; the `pg_dump` subprocess is explicitly out of scope.

## Migration / Rollout

`00008` uses `ADD COLUMN IF NOT EXISTS`, both columns NULL-able, no backfill: zero-downtime, list treats NULL as never-verified. Rollback: revert the branch; unused columns remain harmless. No flags, no phased rollout.

## Open Questions

None — floor (1 KB), row persistence, envelope placement, and `pg_dump` exclusion are all fixed by the inputs.
