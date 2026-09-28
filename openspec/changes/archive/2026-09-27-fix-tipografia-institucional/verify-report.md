```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: sha256:6415854a46788e8cb6c87552517e0272f07b762112a63b8dab386b445d97917d
verdict: pass_with_warnings
blockers: 0
critical_findings: 0
requirements: 7/7
scenarios: 13/13
test_command: npx ng test --include='**/curated-fonts.spec.ts' --include='**/admin-config-identidad.component.spec.ts' --include='**/admin-config-tipografia.spec.ts'
test_exit_code: 0
test_output_hash: sha256:4ed26ffd7b4704bbacd6dde3a1d566e9085fd601e59766c73b6adb07839662f6
build_command: npm run build
build_exit_code: 0
build_output_hash: sha256:eabc4ae3d699b2e73b1cadedf8ac35d2f7d1ab82750f354c4cdaff672fdd3953
```

## Verification Report

**Change**: fix-tipografia-institucional
**Version**: N/A (no versioned spec; single `admin-branding-typography` spec)
**Mode**: Strict TDD

### Completeness
| Metric | Value |
|--------|-------|
| Tasks total | 12 |
| Tasks complete | 12 |
| Tasks incomplete | 0 |

All tasks in `openspec/changes/fix-tipografia-institucional/tasks.md` are checked `[x]`
(Phase 1: 1.1–1.3 pure helpers; Phase 2: 2.1–2.4 component; Phase 3: 3.1–3.2 template;
Phase 4: 4.1–4.2 specs and lint). No pending task blocks verification.

### Scope Guard
Verified the change touches exactly 8 frontend files; unrelated dirty worktree files
(audit, backend, traefik, README, compose) were excluded from every check and nothing
was committed:
- `frontend/src/app/shared/curated-fonts.ts` (modified)
- `frontend/src/app/shared/curated-fonts.spec.ts` (new)
- `frontend/src/app/features/admin/configuracion/admin-config-identidad.service.ts`
- `frontend/src/app/features/admin/configuracion/tabs/admin-config-identidad.component.ts`
- `frontend/src/app/features/admin/configuracion/tabs/admin-config-identidad.component.html`
- `frontend/src/app/features/admin/configuracion/tabs/admin-config-identidad.component.scss`
- `frontend/src/app/features/admin/configuracion/tabs/admin-config-identidad.component.spec.ts`
- `frontend/src/app/features/admin/configuracion/tabs/admin-config-tipografia.spec.ts`
No backend file was modified by this change (`tenant.service.ts` reused untouched).

### Remediation Since Prior Report
Prior report (evidence revision
`sha256:2e0cdff288d82269db628cc480691958c73c1a34cb0953cf0dd278ad38e17a86`,
verdict `fail`) flagged 1 CRITICAL (hint-copy scenario UNTESTED) and 1 PARTIAL
(drop-disclosure toast copy unasserted). The micro test-only fix
(req-apply-tipo-micro-002, recorded in apply-progress #208) remediated both with zero
production change:
- Added `typography hint disambiguates Aplicar (preview) from Guardar Marca (persist)`
  asserting `.typography-hint` textContent contains both `Aplicar` and `Guardar Marca`
  (spec L293-300).
- Extended `save drops unapplied custom input` with `toast()?.type === 'success'` plus
  `toast()?.message` containing `Se descartó la URL sin aplicar` (spec L289-290).
- Supporting edit: `setup()` stores the TestBed fixture in an outer variable
  (was function-local). Rollback boundary: those 2 test blocks plus the fixture
  refactor in `admin-config-identidad.component.spec.ts` only.

### Build & Tests Execution
**Build**: ✅ Passed (exit 0; only pre-existing mermaid CommonJS warnings)
```text
npm run build → dist/frontend written, build_exit=0
```

**Lint**: ✅ Passed (exit 0)
```text
npm run lint:styles → inline-styles-gate OK, stylelint clean, lint_exit=0
```

**Tests**: ✅ 43 passed / 0 failed / 0 skipped across 3 files
```text
npx ng test --include='**/curated-fonts.spec.ts' \
  --include='**/admin-config-identidad.component.spec.ts' \
  --include='**/admin-config-tipografia.spec.ts'
Test Files  3 passed (3)
Tests  43 passed (43)
```
Per-file: `curated-fonts.spec.ts` 13, `admin-config-identidad.component.spec.ts` 21
(was 20; +1 new hint test, drop test extended),
`admin-config-tipografia.spec.ts` 9. NOTE: bare `npx vitest run` cannot resolve the
`@core`/`@shared` tsconfig aliases (no vitest.config in repo); `ng test`
(`@angular/build:unit-test`) is the canonical runner (see SUGGESTION).

**Coverage**: ➖ Not available (no coverage threshold or tool configured for focused runs;
changed-file coverage not measured — informational only, never blocking).

### TDD Compliance
| Check | Result | Details |
|-------|--------|---------|
| TDD Evidence reported | ⚠️ | Prose evidence in apply-progress (#208), no formal TDD Cycle Evidence table |
| All tasks have tests | ✅ | 12/12 tasks map to the 3 spec files |
| RED confirmed (tests exist) | ✅ | 3/3 test files exist with real assertions; new exports (`validateCustomFontUrl`, `mapFontErrorCode`, `mergeBrandingIntoConfig`) and new component members referenced |
| GREEN confirmed (tests pass) | ✅ | 43/43 pass on execution via canonical runner |
| Triangulation adequate | ✅ | `validateCustomFontUrl`: 5 cases (non-HTTPS, foreign host, missing family, valid, empty); `mapFontErrorCode`: 6 cases (5 codes + unknown); `mergeBrandingIntoConfig`: 2 cases; component behaviors multi-case |
| Safety Net for modified files | ✅ | Apply reports 11/11 pre-existing tests green before edits; consistent with the 11 non-font tests retained in the identidad spec |

**TDD Compliance**: 5/6 checks fully passed, 1 WARNING (informal evidence format)

### Test Layer Distribution
| Layer | Tests | Files | Tools |
|-------|-------|-------|-------|
| Unit | 13 | 1 | vitest via `ng test` |
| Integration | 30 | 2 | vitest + TestBed, mocked TenantService/IdentidadService |
| E2E | 0 | 0 | out of scope per design |
| **Total** | **43** | **3** | |

### Changed File Coverage
Coverage analysis skipped — no coverage tool detected (informational, not a failure).

### Assertion Quality
Audited all 3 test files plus the 2 remediated blocks line by line: no tautologies,
no orphan empty checks, no type-only assertions standing alone, no
production-code-free tests (the hint test renders the template via TestBed fixture;
the drop test calls `save()` before asserting the toast), no ghost loops, no
smoke-test-only cases, no CSS-class or implementation-detail assertions (the
`.typography-hint` selector is a DOM query target, not an asserted value), no
mock-heavy files.

**Assertion quality**: ✅ All assertions verify real behavior

### Quality Metrics
**Linter**: ✅ No errors (`lint:styles` exit 0, token-only SCSS, zero hex literals)
**Type Checker**: ✅ No errors (build `ng build` exit 0, strict AOT compile clean)

### Spec Compliance Matrix
| Requirement | Scenario | Test | Result |
|-------------|----------|------|--------|
| Custom URL inline validation | Non-HTTPS URL rejected | `curated-fonts.spec` > rejects non-HTTPS + `identidad.spec` > invalid custom URL exposes inline error | ✅ COMPLIANT |
| Custom URL inline validation | Non-allowlisted host rejected | `curated-fonts.spec` > rejects HTTPS outside allowlist + `identidad.spec` > foreign host custom URL | ✅ COMPLIANT |
| Custom URL inline validation | Missing family parameter warns | `curated-fonts.spec` > warns on allowlisted URLs missing family= | ✅ COMPLIANT |
| Custom URL inline validation | Valid URL stages preview | `curated-fonts.spec` > accepts HTTPS allowlisted + `identidad.spec` > valid custom URL stages preview | ✅ COMPLIANT |
| Custom URL inline validation | Backend 422 codes map inline | `curated-fonts.spec` > mapFontErrorCode ×6 + `identidad.spec` > branding-ok/fonts-422 + `tipografia.spec` > URL custom 422 | ✅ COMPLIANT |
| Font-aware save gating | Font-only change enables save | `identidad.spec` > font-only change enables Guardar Marca | ✅ COMPLIANT |
| Font-aware save gating | Unapplied input never persists | `identidad.spec` > save drops unapplied custom input (drop + success toast copy `Se descartó la URL sin aplicar` asserted, L287-290) | ✅ COMPLIANT |
| Token-routed preview with tab-scoped revert | Preview reaches tokens | `identidad.spec` > catalog pick stages tokens through applyTenantFonts + `tipografia.spec` > selección desde catálogo / preview antes de aplicar | ✅ COMPLIANT |
| Token-routed preview with tab-scoped revert | Cancel reverts preview | `identidad.spec` > cancel reverts staged preview tokens + `tipografia.spec` > cancelar restaura | ✅ COMPLIANT |
| Post-save session apply | Save updates session | `identidad.spec` > successful save applies fonts to session + `tipografia.spec` > guardado exitoso persiste ambos valores | ✅ COMPLIANT |
| Partial-save disclosure | Branding saved, fonts rejected | `identidad.spec` > branding-ok/fonts-422 shows partial copy + `tipografia.spec` > URL custom 422 aislado en banner | ✅ COMPLIANT |
| Action copy semantics | Copy disambiguates actions | `identidad.spec` > typography hint disambiguates Aplicar (preview) from Guardar Marca (persist), L293-300 | ✅ COMPLIANT |
| Automated spec coverage | Font cases pass | Full focused suite: 43/43 green | ✅ COMPLIANT |

**Compliance summary**: 13/13 scenarios compliant

Notes:
- 422 mapping routes all 5 codes to the `general` slot by design (response carries no
  slot context); the component renders them in the typography banner with partial-save
  copy, which satisfies "the matching field shows the corresponding inline message".

### Correctness (Static Evidence)
| Requirement | Status | Notes |
|------------|--------|-------|
| Custom URL inline validation | ✅ Implemented | `validateCustomFontUrl` mirrors backend cheap rules (HTTPS, `fonts.googleapis.com` allowlist, `family=`); reachability stays backend-only |
| Font-aware save gating | ✅ Implemented | `isDirty` ORs `fontSansDirty`/`fontMonoDirty`; `saveFonts` reads staged values only, clears raw inputs with drop disclosure |
| Token-routed preview with revert | ✅ Implemented | All preview stages route through `applyTenantFonts`; cancel/reset re-applies `initial` snapshot via same funnel; zero `[style.font-family]` bindings remain (grep count 0) |
| Post-save session apply | ✅ Implemented | `saveFonts` success updates `tenantService.config` and calls `applyBranding`; no reload |
| Partial-save disclosure | ✅ Implemented | Fonts-PUT 422 keeps branding-saved state, sets `Marca guardada, fuentes rechazadas: {cause}` via `mapFontErrorCode` + `fontErrorCode` extractor for flat `{error: code}` body |
| Action copy semantics | ✅ Implemented | `Aplicar a UI/mono` stages preview only; hint copy states Guardar Marca persists; `form-field` error slots reused |
| Automated spec coverage | ✅ Implemented | 43 automated cases cover catalog, valid/invalid URLs, gating, two-PUT partial failure, hint copy, drop disclosure |

### Coherence (Design)
| Decision | Followed? | Notes |
|----------|-----------|-------|
| Pure helper in `curated-fonts.ts` + `computed()` signals | ✅ Yes | `validateCustomFontUrl`/`FontUrlCheck`; `customSansUrlError`/`customMonoUrlError` gate Aplicar |
| Preview through `applyTenantFonts` with snapshot-revert, tab-scoped | ✅ Yes | `stageFontPreview`/`revertFontPreview` via `initial` snapshot; navigation caveat disclosed in copy |
| `isDirty` ORs font signals; staged values only | ✅ Yes | Font-only edits enable Guardar Marca; unapplied input dropped with disclosure |
| Persist fonts into `tenantService.config` + `applyBranding`; explicit partial copy | ✅ Yes | `mergeBrandingIntoConfig` preserves fonts on branding PUT; 422 path shows partial copy |
| Angular 22 rules (`inject()`, signals, `@if`/`@for`, standalone, no `solv-` prefix) | ✅ Yes | Verified in component/template; selector `admin-config-identidad` |
| Design-system tokens only, no hex/font literals | ✅ Yes | `lint:styles` clean; selects use `var(--font-sans)` per selection-controls rule |
| `tenant.service.ts` reuse, no change | ✅ Yes | File untouched; single funnel preserved |

### Issues Found
**CRITICAL**: None
**WARNING**:
1. Scoped diff (~598 tracked + 119 new ≈ 717 lines) exceeds the 400-line review budget
   the tasks phase forecast as Low risk — delivery as a single unit needs an explicit
   exception or a chained split (ask-on-risk: orchestrator decision).
2. Apply-progress carries TDD evidence as prose (#208: safety net 11/11, RED via
   compile errors) without the formal TDD Cycle Evidence table strict-TDD expects.
**SUGGESTION**:
1. Add a root `vitest.config.ts` (or document `ng test` as the canonical runner):
   bare `npx vitest run` fails on `@core`/`@shared` aliases, which misleads future runs.
2. Manual Identidad tab harness (pick font, custom URL, save, cancel in a live admin
   session) recorded as **pending** — behavior is covered by 30 integration tests, but
   no live-browser pass was executed in this phase.

### Verdict
**PASS WITH WARNINGS** — 12/12 tasks complete, 13/13 scenarios compliant with 43/43
tests green, lint and build clean. Remaining items are non-blocking: review-budget
overrun needs an orchestrator delivery decision, TDD evidence is prose rather than
tabular, and the live-tab pass is pending. Nothing was committed.
