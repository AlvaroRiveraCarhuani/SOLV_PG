# Proposal: Hardening Respaldos + Mantenimiento

## Intent

Backup settings and maintenance activation fail open: invalid values get a success toast, maintenance enables without potent confirmation, nothing reaches audit. This change makes both flows fail closed.

## Scope

### In Scope
- Backup hardening front+back: frequency 1-168h, retention 1-365d, positive integers; inline error on type, Save disabled, never success toast on invalid; backend 422 mirroring `PoliciesValidationError`; template `max` to 365.
- Maintenance type-to-confirm (`MANTENIMIENTO` gates Confirm); motive min 10 chars, full text on screen + `audit_logs`.
- Empty-vigencia warning + auto-off on expiry (mechanism in design).
- Retention-reduction pre-save purge-count warning (no extra write-to-confirm).

### Out of Scope
- Destructive purge change beyond the warning.
- Maintenance-view page or 503 redirect; new count endpoint if `backups` signal suffices (design decides).
- Emergency-action or template-review behavior (patterns reused only).

## Capabilities

### New Capabilities
- `backup-config-validation`: ranges, inline-blocking contract, 422 errors, retention warning.
- `maintenance-activation`: type-to-confirm, motive/vigencia rules, auto-off, audit events.

### Modified Capabilities
- None (empty `openspec/specs/`).

## Approach

Mirror QoS validation and emergency-modal confirm. Backend-first TDD, then frontend. Fix `Until validate:"required"`; NaN/empty invalid. Signals/`@if`/`inject()`, tokens only, green toasts, no `solv-` prefix.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `tabs/admin-config-servidor.component.ts/.html` | Modified | Validation computeds, Save gate, phrase input, warnings |
| `admin-config-servidor.service.ts` | Modified | Validated DTO passthrough |
| `core/domain/backup.go`, `admin_academic.go` | Modified | Range + phrase/motive/vigencia contracts |
| `core/services/backup_service.go`, `admin_academic_service.go` | Modified | Strict validation, auto-off, audit emission |
| `delivery/http/backup_handler.go`, `admin_academic_handler.go`, `maintenance_middleware.go` | Modified | 422 mappings, audit writes, expiry transition |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| `max=90` vs 365 mismatch | Med | Move both to 365 together |
| NaN/empty bypass | Med | Treat as invalid both layers |
| Auto-off write race | Med | Lazy-clear vs sweeper in design + test |
| `MAINTENANCE_*` dropped by filter | Low | Confirm allowlist in design |
| Review over 400 lines | Low | Tight tests; ask-on-risk |

## Rollback Plan

Revert single PR. No migration; purge unchanged, no data recovery. Audit rows stay as inert history.

## Dependencies

- None.

## Success Criteria

- [ ] Invalid backup blocks inline, disables Save, never toasts success; API 422.
- [ ] Maintenance needs `MANTENIMIENTO`, motive >= 10 on screen + audit, warning + auto-off.
- [ ] Retention reduction shows purge-count warning before save.
- [ ] `go test ./... -count=1` passes; review within 400 lines.
