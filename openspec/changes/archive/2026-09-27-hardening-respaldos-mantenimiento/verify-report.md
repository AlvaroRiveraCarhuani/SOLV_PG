```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: sha256:314b66cc57c8db00def1b64d1e05bb7c8a9aa41d24f3560bcf4375b3eeb18fbc
verdict: pass_with_warnings
blockers: 0
critical_findings: 0
requirements: 7/7
scenarios: 12/12
test_command: cd backend && go test ./... -count=1
test_exit_code: 0
test_output_hash: sha256:314b66cc57c8db00def1b64d1e05bb7c8a9aa41d24f3560bcf4375b3eeb18fbc
build_command: cd backend && go build ./...
build_exit_code: 0
build_output_hash: sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
```

## Verification Report

**Change**: hardening-respaldos-mantenimiento
**Version**: N/A (new specs, empty openspec/specs/)
**Mode**: Strict TDD

### Completeness
| Metric | Value |
|--------|-------|
| Tasks total | 10 (1.1, 1.2, 1.3, 1.4, 2.1, 2.2, 2.3, 3.1, 3.2, 3.3) |
| Tasks complete | 10 |
| Tasks incomplete | 0 |

All tasks checked in `openspec/changes/hardening-respaldos-mantenimiento/tasks.md`. Full verification proceeds (no blocked gate). Re-run of prior FAIL: the single blocker (C1, pre-existing slice05 QoS failure) is resolved by a test-only remediation (+37 lines in `slice05_qos_memory_test.go`, uncommitted per scope).

### Build & Tests Execution
**Build**: ✅ Passed (`cd backend && go build ./...`, exit 0, empty output)

**Tests (declared command)**: ✅ Passed, exit 0
```text
cd backend && go test ./... -count=1
ok  solv-backend/internal/core/domain
ok  solv-backend/internal/core/services
ok  solv-backend/internal/delivery/http
ok  solv-backend/internal/infrastructure/docker
ok  solv-backend/internal/infrastructure/storage/memory
ok  solv-backend/tests/integration (188.8s, includes slice05 green)
```

**Slice05 isolated confirmation**: ✅ PASS
```text
go test ./tests/integration/ -run 'TestQoSAutoBurstingAndDualValidation' -count=1 -v
--- PASS: TestQoSAutoBurstingAndDualValidation (1.87s)
    slice05_qos_memory_test.go:193: PASS: ... Initial RAM: 256 MB
```

**Change-scoped test runs (all green)**:
```text
go test ./internal/core/services/ -run 'Backup|Maintenance' -count=1 → ok
go test ./tests/integration/ -run 'Backup|Maintenance' -count=1 → ok
  TestSlice14_MaintenancePeriods PASS (subtest 4: 422 + audit verbatim + expiry read)
  TestSlice16_Backups_CompleteSuite PASS (subtest 8: 422 ranges + valid borders)
npx ng test --watch=false --include='**/admin-config-servidor.component.spec.ts'
  1 file passed, 12/12 tests passed
```

**Coverage**: changed-function coverage (whole-file % diluted by pre-existing untested functions in same files): `EnableMaintenance` 87.5%, `GetStatus` 100%, `ClearExpiredMaintenance` 62.5%, `UpdateConfig` (backup) 69.0%. Informational only, no gate.

### TDD Compliance
| Check | Result | Details |
|-------|--------|---------|
| TDD Evidence reported | Partial | Prose summary in apply-progress, no formal TDD Cycle Evidence table |
| All tasks have tests | Yes | 7/7 task groups map to test files (unit + integration + frontend) |
| RED confirmed (tests exist) | Yes | `backup_service_test.go`, `maintenance_hardening_test.go`, extended `component.spec.ts`, extended slice14/slice16 files all present |
| GREEN confirmed (tests pass) | Yes | Full suite exit 0; all scoped runs pass on execution |
| Triangulation adequate | Yes | Distinct boundary values per behavior (0/1/168/169, 0/1/365/366, phrase/casing/empty, motive 9/10, past/future/garbage until, NaN/decimal) |
| Safety Net for modified files | Partial | No explicit pre-modification run recorded; compensated by full-suite run with zero regressions in all touched packages |

**TDD Compliance**: 4/6 full, 2 partial (reporting gaps, behavior independently verified)

### Test Layer Distribution
| Layer | Tests | Files | Tools |
|-------|-------|-------|-------|
| Unit | 9 top-level (22 w/ subtests) + 12 frontend | 3 | go test, vitest |
| Integration | 2 suites (slice14 subtest 4, slice16 subtest 8) + slice05 QoS | 3 | go test + live PostgreSQL (solv_test) |
| E2E | 0 | 0 | not installed |
| **Total** | **23+ top-level** | **8** | |

### Changed File Coverage
| File | Line % | Branch % | Uncovered Lines | Rating |
|------|--------|----------|-----------------|--------|
| `backup_service.go` (`UpdateConfig` fn) | 69.0% (fn) | — | pre-existing paths outside strict checks | Acceptable (diluted) |
| `admin_academic_service.go` (`EnableMaintenance` fn) | 87.5% (fn) | — | — | Acceptable |
| `admin_academic_service.go` (`GetStatus`/`ClearExpiredMaintenance` fn) | 100% / 62.5% (fn) | — | non-expiry branches covered by integration | Acceptable |

File-level % is diluted by pre-existing untested functions in the same files (e.g. `TriggerBackup`, period CRUD at 0.0%). All changed lines are exercised by the passing tests above.

### Assertion Quality
Backend unit (`backup_service_test.go` re-read this run): every case calls production code (`UpdateConfig`), asserts typed error codes (`BackupValidationError.Code` per boundary), asserts failure on nil error (`t.Fatalf` when invalid input yields no error). Case tables are non-empty literals (no ghost loops), expectations vary per case (accept vs reject with distinct codes). No tautologies, no empty-only checks, no type-only assertions.
Frontend (`admin-config-servidor.component.spec.ts`): `toBe(true)` hits are value assertions on production computeds (`canSaveBackup()`, `confirmDisabled()`, `maintenanceConfirmValid()`), combined with behavioral assertions (service call args, error strings, toast null). No smoke-only, no CSS/mock-count assertions.

**Assertion quality**: ✅ All assertions verify real behavior (0 CRITICAL, 0 WARNING)

### Quality Metrics
**Linter**: `lint:styles` clean (prior run, no style files touched by remediation); no hex literals in changed component HTML/SCSS
**Type Checker**: `go vet` clean on all touched backend packages (prior run, remediation is test-only); frontend spec compiles and runs under ng test
**Angular architecture**: standalone, `inject()`, signals/computed/effect only; `@if` flow, zero `*ngIf`/`*ngFor`; `modal-shell`/`form-field` primitives; selector `admin-config-servidor` (no `solv-` prefix); tokens only
**Design system**: no toast color literals in changed files; `max="365"` aligned with backend 1-365

### Spec Compliance Matrix
| Requirement | Scenario | Test | Result |
|-------------|----------|------|--------|
| Frequency Range | Valid frequency accepted | `backup_service_test.go > freq 1/168 accepted` + slice16 subtest 8 | ✅ COMPLIANT |
| Frequency Range | Out-of-range or non-integer rejected | `freq 0/169 rejected (backup_frequency_invalid)` + `NaN o decimal` + slice16 subtest 8 (422) | ✅ COMPLIANT |
| Retention Range | Boundary values accepted | `retention 1/365 accepted` + slice16 subtest 8 | ✅ COMPLIANT |
| Retention Range | Out-of-range rejected | `retention 0/366 rejected (backup_retention_invalid)` + slice16 subtest 8 (422) | ✅ COMPLIANT |
| Inline Blocking and Save Gate | Invalid input blocks save | spec `respaldo inválido: error inline, Save bloqueado y sin toast` (service not called, toast null) | ✅ COMPLIANT |
| Retention Reduction Warning | Lowering retention warns | spec `bajar retención advierte...` (`purgeCount()`=1, purge path untouched, slice16 subtests 4-6 green) | ✅ COMPLIANT |
| Type-to-Confirm Phrase | Exact phrase enables | `EnableHappyPath` + spec `activación...` (`enableMaintenance` called with `MANTENIMIENTO`) + slice14 subtest 4 | ✅ COMPLIANT |
| Type-to-Confirm Phrase | Phrase mismatch blocks | 3 backend phrase cases + spec `frase distinta...` (`confirmDisabled`, service not called) | ✅ COMPLIANT |
| Motive Length and Visibility | Short motive rejected | `motive 9 chars rejected (maintenance_reason_invalid)` + slice14 subtest 4 (422) | ✅ COMPLIANT |
| Motive Length and Visibility | Motive recorded and visible | `EnableHappyPath` (verbatim persist) + handler `MAINTENANCE_ENABLED` audit + slice14 subtest 4 (metadata verbatim assert) + status block `Motivo:` line | ✅ COMPLIANT |
| Optional Vigencia with Warning and Auto-Off | Empty vigencia warns | spec `vigencia vacía advierte...` + `EnableHappyPath` (empty until → nil indefinite) | ✅ COMPLIANT |
| Optional Vigencia with Warning and Auto-Off | Expiry deactivates | `GetStatus_LazyClearExpired` + middleware `ClearExpiredMaintenance` + handler `MAINTENANCE_AUTO_DISABLED` audit + slice14 subtest 4 (expiry read) | ✅ COMPLIANT |

**Compliance summary**: 12/12 scenarios compliant

### Correctness (Static Evidence)
| Requirement | Status | Notes |
|------------|--------|-------|
| Frequency/retention strict ranges + 422 | ✅ Implemented | `BackupValidationError` + `errors.As` → 422 in `backup_handler.go`, mirrors `PoliciesValidationError` |
| Inline blocking + Save gate | ✅ Implemented | `validateBackupInt` pure fn; `canSaveBackup` computed; `saveBackupStrategy` early-returns |
| Retention purge-count warning, purge unchanged | ✅ Implemented | `purgeCount` computed from `backups` signal; no change to rotation path |
| Phrase/motive/vigencia validation + 422 | ✅ Implemented | `MaintenanceValidationError` 3 codes; `ConfirmPhrase` added; `Until` optional |
| MAINTENANCE_* audit with verbatim motive | ✅ Implemented | `writeMaintenanceAudit` (ENABLED/DISABLED/AUTO_DISABLED); middleware auto-off audited on `GetMaintenanceStatus` reads (design-accepted) |
| Lazy auto-off, no sweeper | ✅ Implemented | `ClearExpiredMaintenance` in `GetStatus` + `maintenance_middleware.go` |
| Frontend gates (phrase, vigencia warning, max 365) | ✅ Implemented | `confirmDisabled`, `vigenciaWarning`, `maintenanceUntilError`, `confirm_phrase` passthrough in service |
| Slice05 QoS base hermetic | ✅ Implemented | Test-only `stubTenantRepo` (ram_limit_mb 256) + `SetPoliciesService`; root cause was missing policies injection (default 512), not DB state |

### Coherence (Design)
| Decision | Followed? | Notes |
|----------|-----------|-------|
| Lazy-clear on read, no sweeper | ✅ Yes | `GetStatus` + middleware; `MAINTENANCE_AUTO_DISABLED` on status reads |
| Client-side purge count, no new endpoint | ✅ Yes | `purgeCount` computed; no new route |
| New `MAINTENANCE_*` audit events | ✅ Yes | ACTION strings + `{reason, until, actor}` metadata |
| Both template max and backend to 365 together | ✅ Yes | `max="365"` + strict 1-365 in same change |
| `Until` optional (drop `required`) | ✅ Yes | Empty = indefinite + warning; past = 422 |
| Strict coercion both layers | ✅ Yes | `validateBackupInt` rejects NaN/empty/decimal; backend rejects non 1-168/1-365 |

### Issues Found
**CRITICAL**: None
**WARNING**:
- W1: apply-progress has no formal TDD Cycle Evidence table (prose summary only). TDD behavior independently verified (RED files exist, GREEN passes on full suite, triangulated, assertion audit clean).
- W2: slice05 remediation (`stubTenantRepo` +37, test-only) is uncommitted by verify scope (do NOT commit per instructions). Needs a commit before archive; no code risk.
- W3: changed-file coverage diluted by pre-existing untested functions in the same files; changed functions 62.5-100%. Informational only.
- W4: authored size ~851 lines vs 400-line review budget; preflight `exception-ok` (maintainer size:exception accepted). Noted for archive/PR slicing, not a code blocker.
- W5: unrelated worktree dirt exists (other changes in progress); verification scoped to change files only.
**SUGGESTION**:
- S1: future apply reports should include the formal TDD Cycle Evidence table so verify does not need to reconstruct it.

### Verdict
PASS WITH WARNINGS
Full backend suite exits 0 with slice05 green after test-only remediation; 12/12 scenarios compliant, zero regressions — warnings are process-level (evidence table format, uncommitted test stub, size exception, worktree dirt).
