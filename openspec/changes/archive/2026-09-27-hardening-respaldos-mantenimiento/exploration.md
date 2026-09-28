## Exploration: hardening-respaldos-mantenimiento

### Current State

**Backup settings (frontend).** `admin-config-servidor.component.html` (lines 176-208) renders two `type="number"` inputs (`backup-freq` min 1 max 168, `backup-retention` min 1 max 90) bound via `[ngModel]` / `(ngModelChange)` to plain numeric signals (`backupFrequency = signal(6)`, `backupRetention = signal(7)` in `admin-config-servidor.component.ts` lines 85-86). There is NO inline validation, NO error message, NO `canSave` gate: the Save button is disabled only on `isSavingBackup()` (line 200). `saveBackupStrategy()` (lines 219-236) sends whatever the signals hold and shows success toast `Configuración de respaldos guardada.` unconditionally on HTTP 200 — this is the exact toast path behind the "saved" false positive. Because `type="number"` still lets users type free text (`12sdsadas`, `s`) and absurd magnitudes, and `ngModelChange` coerces to string/NaN without a guard, invalid values flow straight to the backend.

**Backup persist/validation (backend).** `PUT /api/v1/admin/backups/config` → `BackupHandler.UpdateConfig` (`backup_handler.go` lines 47-66) decodes `UpdateBackupConfigDTO` with zero range checks and delegates to `BackupService.UpdateConfig` (`backup_service.go` lines 45-85), which applies `if dto.LocalFrequencyHours > 0` / `if dto.LocalRetentionDays > 0` — i.e. any positive int passes, no upper bound, no 1-168 / 1-365 enforcement. `local_frequency_hours` / `local_retention_days` are plain `int` in `domain/backup.go` with no validate tags. DB defaults are 6h / 7d (`00001_baseline.sql` lines 452-453). Contrast with the QoS sibling: `ServerPoliciesService.Update` (`server_policies_service.go` lines 62-102) uses strict catalog/range validation returning typed `PoliciesValidationError` (`ram_limit_invalid`, `inactivity_invalid`, `max_containers_invalid` 1-500) — the established pattern to copy for backups.

**Retention purge.** `TriggerBackup` fires `go s.applyRetentionPolicy(tenantID)` (line 196); it reads current `LocalRetentionDays` and calls `GetExpiredExecutions(tenantID, retentionDays)` (`backup_repository.go` lines 176-196: `started_at < NOW() - ($2 || ' days')::INTERVAL`, fallback 7 if <= 0), then deletes file + row per expired execution. Purge is silent: no count returned, no pre-save warning, no audit log. There is no endpoint returning "how many backups would be deleted if retention drops to N", but `ListExecutions` returns `X-Total-Count` and the `backups` signal is already loaded client-side, so a pre-save count can be computed without a new endpoint (or via a cheap `HEAD`-style count query if TDD prefers backend truth).

**Maintenance modal.** Lives in the same servidor component (`admin-config-servidor.component.html` lines 267-325, logic lines 127-214). Activation requires `maintenanceReason.trim().length >= 10` (`maintenanceConfirmValid`, lines 132-137) with a soft inline error only when 1-9 chars typed; `maintenanceUntil` is optional `datetime-local` with a hint claiming auto-off. There is NO type-to-confirm field, NO empty-vigencia warning state, and the motive's purpose is unexplained in-UI (label says min 10 chars but not where it surfaces). On confirm, `enableMaintenance(until, reason)` posts `{ until, reason }` to `POST /api/v1/admin/maintenance/enable`.

**Motive/vigencia handling + audit.** Backend `EnableMaintenanceDTO` (`admin_academic.go` lines 55-58) has `Until string validate:"required"` (contradicts frontend optional + handler never validates) and `Reason` optional with NO min-length check. `MaintenanceService.EnableMaintenance` (lines 183-199) parses `Until` as RFC3339 (fallback `2006-01-02T15:04:05`), empty string = nil = indefinite. `DisableMaintenance` clears reason. Neither handler (`admin_academic_handler.go` lines 52-78) writes to `audit_logs` — unlike emergency actions (`ExecuteEmergencyAction` writes `EMERGENCY_*` with motive + impact) and template review (`TEMPLATE_REVIEWED` with reason). So maintenance activations are currently invisible in audit.

**Maintenance screen display.** No dedicated `maintenance-view` component exists (ADR-031 references `features/system/maintenance/maintenance-view.component.ts` but the path does not exist; no 503→maintenance redirect interceptor found in frontend). The only visible surface for motive/vigencia is the `.maintenance-info` block in the servidor tab (html lines 129-138: `Hasta:` + `Motivo:` full text when active). Non-admin users get a raw JSON 503 (`maintenance_middleware.go` lines 37-56: `{ error: maintenance_mode, message, until?, reason? }`). Auto-off on expiry does NOT exist as a write: the middleware only *bypasses the block* when `now > until` (line 39) but never clears `maintenance_mode`, so the admin tab still shows ACTIVO after expiry — the agreed "auto-off" needs a real state transition (lazy clear on read or a sweeper).

### Affected Areas
- `frontend/src/app/features/admin/configuracion/tabs/admin-config-servidor.component.ts` — backup signals lack validation computeds; maintenance modal lacks confirm-phrase signal + vigencia warning + retention pre-save count.
- `frontend/src/app/features/admin/configuracion/tabs/admin-config-servidor.component.html` — backup `form-field` needs `[error]` bindings + Save gate; maintenance modal needs type-to-confirm input, vigencia warning, motive purpose hint, retention reduction warning.
- `frontend/src/app/features/admin/configuracion/admin-config-servidor.service.ts` — `updateBackupConfig` / `enableMaintenance` pass through unvalidated DTOs; `loadAll` already provides `backups` signal for client-side retention count.
- `frontend/src/app/features/admin/configuracion/tabs/admin-config-servidor.component.spec.ts` — existing vitest tests cover happy paths only (save 12/14, motive >= 10); TDD must add red tests for invalid numerics, disabled Save, phrase mismatch, audit visibility.
- `backend/internal/core/domain/backup.go` — `UpdateBackupConfigDTO` needs range contract (1-168 h, 1-365 d); consider typed validation errors mirroring `PoliciesValidationError`.
- `backend/internal/core/services/backup_service.go` — `UpdateConfig` needs strict range validation returning 422-mappable errors; retention-count helper for pre-save warning.
- `backend/internal/delivery/http/backup_handler.go` — `UpdateConfig` needs 422 mapping for out-of-range (currently always 200/500).
- `backend/internal/core/domain/admin_academic.go` — `EnableMaintenanceDTO` needs `Reason` min-10 + optional `Until` + fixed confirmation phrase field (e.g. `MANTENIMIENTO`); `Until validate:"required"` contradicts agreed optional vigencia.
- `backend/internal/core/services/admin_academic_service.go` — `EnableMaintenance` needs reason-length + phrase + vigencia-past validation, plus auto-off transition and audit event emission.
- `backend/internal/delivery/http/admin_academic_handler.go` — enable/disable need 422 mappings + audit log writes (`MAINTENANCE_ENABLED/DISABLED` with motive, vigencia, actor).
- `backend/internal/delivery/http/maintenance_middleware.go` — expiry currently only bypasses block; needs lazy auto-off write or sweeper hook.
- `backend/internal/infrastructure/storage/postgres/tenant_repository.go` — `SetMaintenance`/`GetMaintenance` are the seams for auto-off + audit metadata.
- `backend/tests/integration/slice16_backups_test.go`, `slice14_maintenance_periods_test.go` — integration tests to extend under STRICT TDD (`cd backend && go test ./... -count=1`).

### Approaches
1. **Mirror QoS validation + emergency confirm pattern (recommended)** — Backend: typed range errors for backups (`backup_frequency_invalid`, `backup_retention_invalid`, 422) copying `PoliciesValidationError`; frontend: `computed()` error strings + disabled Save copying `policiesDirty/canSavePolicies`; maintenance type-to-confirm copying `AdminAuditoriaEmergenciasComponent.confirmPhrase/confirmDisabled` (exact-match `MANTENIMIENTO`, motive >= 10, audit `MAINTENANCE_*` copying `recordEmergencyAudit`).
   - Pros: consistent with two already-reviewed patterns; reviewers recognize the shape; backend 422 messages reusable by frontend `resolveError`.
   - Cons: touches both frontend and backend in one change; slightly larger review surface.
   - Effort: Medium
2. **Frontend-only clamping (min/max + inputmode)** — Clamp on input, `max=168/365`, strip non-digits, keep backend as-is.
   - Pros: smallest diff; fixes the screenshot symptoms fast.
   - Cons: backend still accepts absurd values via API; violates "never success toast on invalid" under direct API use; diverges from QoS precedent (backend-validated); TDD backend tests would still fail.
   - Effort: Low
3. **Backend-only strict validation, frontend passthrough** — 422 on invalid, frontend shows backend error toast after failed save.
   - Pros: single source of truth; API-safe.
   - Cons: violates agreed "error on type, Save disabled" (user only learns after clicking Save + failed roundtrip); keeps the false-positive toast path for edge cases.
   - Effort: Low

### Recommendation
Approach 1. It is the only option satisfying all agreed assumptions simultaneously: inline blocking + disabled Save (frontend `computed`), API safety (backend 422), GitHub-style potent confirmation (typed `MANTENIMIENTO` copied from emergency modal), motive visibility (full text in `.maintenance-info` + new audit event), vigencia warning + auto-off (middleware lazy-clear or sweeper), and retention pre-save count (computed from loaded `backups` or a count query). Split TDD work backend-first (`BackupService.UpdateConfig` range tests, `EnableMaintenance` phrase/reason/vigencia tests, audit tests) then frontend (disabled-Save tests, phrase-mismatch tests), per `strict-tdd.md` RED-GREEN-REFACTOR with `cd backend && go test ./... -count=1`.

### Risks
- HTML `max="90"` on retention conflicts with agreed 1-365: both template attr and backend must move to 365 together or a 91-365 value passes one layer and fails the other.
- `type="number"` + `ngModel` coercion edge: empty string → null/NaN can bypass `> 0` checks and render as 0; validation `computed` must treat NaN/empty as invalid, not as "unchanged".
- Auto-off has no existing writer: middleware read-path bypass vs real `SetMaintenance(false)` write is a design decision (lazy write on next request vs sweeper) with concurrency implications for `updated_at`/audit.
- Maintenance audit event name (`MAINTENANCE_*`) is new vocabulary: confirm against audit-log allowlist/seed data so the auditoria tab filter does not silently drop it.
- `EnableMaintenanceDTO.Until validate:"required"` tag contradicts optional vigencia: must be fixed or validator will 400 legitimate indefinite activations.

### Ready for Proposal
Yes — scope is bounded to the servidor tab + backup/maintenance backend seams above. Propose with: numeric ranges 1-168/1-365, inline-blocking contract, `MANTENIMIENTO` phrase contract, audit event names, auto-off mechanism choice, retention count source (client signal vs count endpoint). Review budget risk: Low-Medium (two domains but small files); single PR likely within 400 lines if tests are tight.
