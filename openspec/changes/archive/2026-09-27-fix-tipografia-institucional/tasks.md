# Tasks: Fix Institutional Typography (Identidad Tab)

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 280–340 |
| 400-line budget risk | Low |
| Chained PRs recommended | No |
| Suggested split | Single PR |
| Delivery strategy | ask-on-risk |
| Chain strategy | pending |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: pending
400-line budget risk: Low

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| 1 | Pure validation + 422 mapper with unit cases | PR 1 | `npx jest curated-fonts` | N/A — pure functions, no runtime | `shared/curated-fonts.ts` + mapper only |
| 2 | Component gating, token preview/revert, post-save apply + template copy | PR 1 | `npx jest admin-config-identidad.component.spec` | Manual Identidad tab: pick font, custom URL, save, cancel | `admin-config-identidad.*` + service revertible alone |

## Phase 1: Foundation — pure helpers (TDD RED first)

- [x] 1.1 RED: add failing cases for `validateCustomFontUrl` (non-HTTPS, foreign host, missing `family=`, valid) + `mapFontErrorCode` (5 backend 422 codes) in `frontend/src/app/shared/curated-fonts.spec.ts` (or component spec pure block, no TestBed)
- [x] 1.2 GREEN: implement `FontUrlIssue`/`FontUrlCheck` + `validateCustomFontUrl` in `frontend/src/app/shared/curated-fonts.ts`; warn on missing `family=` instead of silent Inter fallback
- [x] 1.3 GREEN: implement `mapFontErrorCode` in `frontend/src/app/features/admin/configuracion/admin-config-identidad.service.ts` mapping 5 codes to sans/mono/general slots

## Phase 2: Core — component gating, preview, persistence

- [x] 2.1 Add `customSansUrlError`/`customMonoUrlError` computeds in `admin-config-identidad.component.ts` gating Aplicar; use `inject()`, `computed()`
- [x] 2.2 Route catalog pick / valid custom apply through `applyTenantFonts` with initial-snapshot; revert on Cancel/Discard/reset via same funnel
- [x] 2.3 Compose `isDirty = brandingDirty || fontSansDirty || fontMonoDirty`; `saveFonts()` reads staged values only, clears raw inputs with drop disclosure
- [x] 2.4 Persist font fields into `tenantService.config` on `saveBranding` tap; on `saveFonts` success call `applyBranding`; on fonts-422 show partial copy via `mapFontErrorCode`

## Phase 3: Template — gating and copy

- [x] 3.1 Bind Aplicar `disabled` to computed errors in `admin-config-identidad.component.html`; drop per-option `[style.font-family]`
- [x] 3.2 Add hint copy (Aplicar stages preview only) and inline font error slots reusing `form-field` error; tokens only, no hex/font literals

## Phase 4: Verification — specs and lint

- [x] 4.1 Extend `admin-config-identidad.component.spec.ts`: catalog stages tokens, invalid URL blocks Aplicar, font-only enables Guardar Marca, cancel reverts, branding-ok/fonts-422 partial copy
- [x] 4.2 Run focused specs + `npm run lint:styles` + build; fix token/hex violations before commit
