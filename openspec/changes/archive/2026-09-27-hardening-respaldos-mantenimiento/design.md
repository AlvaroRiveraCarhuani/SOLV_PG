# Design: Hardening Respaldos + Mantenimiento

## Technical Approach

Fail-closed hardening of the servidor tab, mirroring two reviewed patterns: QoS strict validation (`PoliciesValidationError` + 422) for backup ranges, and the emergency-modal type-to-confirm for maintenance. Backend validates first (TDD), frontend mirrors with `computed()` gates so invalid input never reaches the success-toast path. No new endpoints, no migration, single PR.

## Architecture Decisions

| Option | Tradeoff | Decision |
|---|---|---|
| Auto-off: lazy-clear on read vs sweeper goroutine | Sweeper needs a worker/cron that does not exist; races with `updated_at`. Lazy-clear reuses existing read seams. | **Lazy-clear**: `GetStatus` and `MaintenanceMiddleware` detect `until` past, call `SetMaintenance(false)` once, emit `MAINTENANCE_AUTO_DISABLED`. No sweeper. |
| Retention count: client `backups` signal vs new count endpoint | New endpoint adds route + tests + review lines for data the client already holds. | **Client-side computed** from loaded `backups` signal (`started_at < now - N days`). No new endpoint. |
| Audit events: new `MAINTENANCE_*` vs reuse generic action | `audit_logs.action` is free-form (`ListFiltered` binds the string; no allowlist/seed gate verified). | **New events** `MAINTENANCE_ENABLED`, `MAINTENANCE_DISABLED`, `MAINTENANCE_AUTO_DISABLED` with `{reason, until, actor}` metadata, written in handler via `auditLogRepo` (copy `TEMPLATE_REVIEWED` pattern). |
| Template `max`: 90 vs 365 | 90 contradicts spec 1-365; split changes break one layer. | **Both to 365 together**: `max="365"` in template + backend range `1-365` in same change. |
| `Until`: `validate:"required"` vs optional | Tag rejects legitimate indefinite activations, contradicting frontend + service (empty = nil). | **Remove `required`**; empty string = indefinite + warning. Past date = 422 `maintenance_until_invalid`. |
| Number coercion: lenient vs strict | `type="number"` + `ngModel` yields `""`/`NaN`; `> 0` checks let them through to a false success toast. | **Strict both layers**: frontend treats `NaN`/empty/non-integer as invalid; backend rejects non `1-168` / `1-365` with 422. |

QoS mirror: `BackupValidationError{Code, Message}` + `errors.As` → 422 in handler, same as `PoliciesValidationError`. Emergency mirror: `confirmPhrase`/`confirmDisabled` exact-match `MANTENIMIENTO` + motive ≥ 10.

## Data Flow

```
Backup: input ──→ computed error ──→ canSaveBackup gate ──→ PUT /admin/backups/config ──→ UpdateConfig strict check ──→ 422 | persist
Retention: retention input ──→ purgeCount computed (backups signal) ──→ warning before save; purge itself unchanged
Maintenance: phrase + motive + until signals ──→ confirmDisabled ──→ POST /admin/maintenance/enable ──→ phrase/reason/until check ──→ SetMaintenance + MAINTENANCE_ENABLED audit
Read: GetStatus / middleware ──→ until expired? ──→ lazy SetMaintenance(false) + MAINTENANCE_AUTO_DISABLED ──→ read as off
```

## File Changes

| File | Action | Description |
|---|---|---|
| `backend/internal/core/domain/backup.go` | Modify | Document `1-168h / 1-365d` contract on `UpdateBackupConfigDTO` |
| `backend/internal/core/services/backup_service.go` | Modify | Strict range checks returning `BackupValidationError` (`backup_frequency_invalid`, `backup_retention_invalid`) |
| `backend/internal/delivery/http/backup_handler.go` | Modify | `errors.As` mapping to 422, mirroring `ServerPoliciesHandler` |
| `backend/internal/core/domain/admin_academic.go` | Modify | `EnableMaintenanceDTO`: drop `validate:"required"`, add `ConfirmPhrase`, document motive ≥ 10 |
| `backend/internal/core/services/admin_academic_service.go` | Modify | Phrase/reason/past-date validation + lazy auto-off helper emitting audit metadata |
| `backend/internal/delivery/http/admin_academic_handler.go` | Modify | 422 mappings (`maintenance_confirm_invalid`, `maintenance_reason_invalid`, `maintenance_until_invalid`) + `MAINTENANCE_*` audit writes |
| `backend/internal/delivery/http/maintenance_middleware.go` | Modify | Expired → lazy `SetMaintenance(false)` + auto-disabled audit, then `next` |
| `frontend/.../tabs/admin-config-servidor.component.ts` | Modify | `backupFrequencyError/RetentionError/canSaveBackup/purgeCount/confirmPhrase/confirmDisabled/vigenciaWarning` computeds; `inject()` + signals only |
| `frontend/.../tabs/admin-config-servidor.component.html` | Modify | `[error]` bindings, `max="365"`, Save gate, phrase input, warnings; `@if` flow, `modal-shell`/`form-field`, tokens only |

## Interfaces / Contracts

```go
type BackupValidationError struct{ Code, Message string }
func (e *BackupValidationError) Error() string { return e.Message }

type EnableMaintenanceDTO struct {
  Until         string `json:"until"` // optional RFC3339; "" = indefinite
  Reason        string `json:"reason"`
  ConfirmPhrase string `json:"confirm_phrase"`
}
```

```ts
backupFrequencyError = computed(() => validateInt(this.backupFrequency(), 1, 168));
canSaveBackup = computed(() => !this.backupFrequencyError() && !this.backupRetentionError() && !this.isSavingBackup());
purgeCount = computed(() => this.backups().filter(b => isOlderThan(b.started_at, this.backupRetention())).length);
confirmDisabled = computed(() => this.confirmPhrase() !== 'MANTENIMIENTO' || this.maintenanceReason().trim().length < 10 || this.isTogglingMaintenance());
```

422 codes: `backup_frequency_invalid`, `backup_retention_invalid`, `maintenance_confirm_invalid`, `maintenance_reason_invalid`, `maintenance_until_invalid`. Frontend `resolveError` reuses them; success toasts stay green `#15803D`, errors red.

## Testing Strategy

| Layer | What to Test | Approach |
|---|---|---|
| Unit (Go) | Range edges 0/1/168/169, 0/1/365/366, NaN-as-0; phrase mismatch/casing; motive 9/10; past until; lazy-clear transition | Table tests per `strict-tdd.md` RED first |
| Integration | `PUT config` 422 codes; `POST enable` 422 + audit row with verbatim motive; expiry read returns off | Extend `slice16_backups_test.go`, `slice14_maintenance_periods_test.go`; `go test ./... -count=1` |
| Frontend (vitest) | Invalid typing → inline error + disabled Save + no toast; phrase mismatch blocks; empty vigencia warns; retention warning shows count | Mirror `admin-config-servidor.component.spec.ts` happy-path style |

## Threat Matrix

N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary.

## Migration / Rollout

No migration required. Single PR, revert-safe; audit rows persist as inert history. Keep diff within 400-line review budget with tight tests.

## Open Questions

None. All six must-resolve items decided above.
