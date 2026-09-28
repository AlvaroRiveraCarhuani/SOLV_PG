# Design: Fix Institutional Typography (Identidad Tab)

## Technical Approach

Frontend-only fix combining proposal approaches 1+2 onto spec `admin-branding-typography`: fail-closed inline URL validation mirroring backend cheap rules (HTTPS, `fonts.googleapis.com` allowlist, `family=` presence), font-aware `isDirty`/`canSave`, and all font rendering routed through the existing `TenantService.applyTenantFonts` funnel (`:root` `--font-sans` / `--font-mono`). Reachability stays backend-only via 422-code mirror. No backend changes, no new components; signals + `computed()` + `inject()` per Angular 22 rules, tokens only (`var(--font-*)`).

## Architecture Decisions

### Decision: Where cheap-rule validation lives

| Option | Tradeoff | Decision |
|---|---|---|
| Validate inline in component methods | Duplicates logic per slot, untestable | Reject |
| Pure helper in `curated-fonts.ts` + `computed()` signals | Single pure seam, mirrors `typography_service.go` cheap rules, unit-testable without DOM | Accept |

Pure `validateCustomFontUrl(raw): FontUrlCheck` owns HTTPS/host/`family=` checks; component exposes `customSansUrlError` / `customMonoUrlError` computeds that gate `Aplicar` buttons; service `validate()` stays branding-only while a small `mapFontErrorCode()` translates 422 codes to the same inline slots.

### Decision: Preview override vs session tokens coexistence

| Option | Tradeoff | Decision |
|---|---|---|
| Keep local `[style.font-family]` preview stacks | Preview diverges from app; selects/lists never update | Reject |
| Preview through `applyTenantFonts` with snapshot-revert | Whole tab reflects tokens; admin session visibly re-themed before save | Accept, tab-scoped |

On first staged preview, snapshot `initial` fonts; every catalog pick / valid custom apply calls `applyTenantFonts({ ...config, font_sans_family, font_mono_family })`. Cancel / Discard / reset re-applies the snapshot via the same funnel, so tokens always return to last-saved values. Navigation without cancel keeps preview until next `init()` — accepted and disclosed in copy ("Aplicar stages preview only").

### Decision: Dirty gating and unapplied input

| Option | Tradeoff | Decision |
|---|---|---|
| Extend `isDirty` only | `canSave` still ignores fonts | Reject |
| Compose `isDirty = brandingDirty || fontSansDirty || fontMonoDirty`; staged values only | Font-only change enables save; typed-but-unapplied URLs never persist | Accept |

`isDirty` ORs existing branding comparison with the two dead `font*Dirty` signals. `saveFonts()` reads only staged `fontSansValue`/`fontMonoValue`, never raw inputs; save/reset clears raw inputs and discloses "unapplied input dropped" via hint + toast path.

### Decision: Post-save apply and partial-save disclosure

| Option | Tradeoff | Decision |
|---|---|---|
| Reset `initial` only (today) | Session keeps stale tokens until cache expiry | Reject |
| Persist font fields into `tenantService.config` + `applyBranding`; explicit partial copy | Session matches save without reload; two-PUT semantics stay visible | Accept |

`saveBranding` tap preserves `font_sans_family` / `font_mono_family` when rebuilding `TenantConfig`; `saveFonts` success updates config and calls `applyBranding`. Fonts-PUT failure keeps branding-saved state and shows `fontError` as "branding saved, fonts rejected: {cause}".

## Data Flow

```
custom URL input ──→ validateCustomFontUrl (pure) ──→ computed error ──→ Aplicar gated
catalog pick / valid apply ──→ fontSansValue/MonoValue ──→ applyTenantFonts ──→ :root tokens ──→ tab
Guardar Marca ──→ PUT branding ──→ PUT fonts ──→ config.set + applyBranding (session) | 422 ──→ mapFontErrorCode ──→ inline
Cancel/Discard ──→ applyTenantFonts(initial snapshot) ──→ tokens revert
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `frontend/src/app/shared/curated-fonts.ts` | Modify | Add pure `validateCustomFontUrl` + `FontUrlCheck`; return missing-`family=` as warning instead of silent Inter fallback |
| `frontend/src/app/features/admin/configuracion/tabs/admin-config-identidad.component.ts` | Modify | Error computeds, Aplicar gating, `isDirty` ORs font signals, token-routed preview with snapshot, post-save config+apply, revert on cancel/reset |
| `frontend/src/app/features/admin/configuracion/tabs/admin-config-identidad.component.html` | Modify | Bind Aplicar `disabled` to computed errors, hint copy (Aplicar never persists), drop per-option `[style.font-family]`, inline font error slots |
| `frontend/src/app/features/admin/configuracion/admin-config-identidad.service.ts` | Modify | Keep font fields in `saveBranding` tap; add 422 font-code mapper used by component |
| `frontend/src/app/features/admin/configuracion/tabs/admin-config-identidad.component.spec.ts` | Modify | Font cases: catalog, valid/invalid URLs, font-only gating, two-PUT partial failure |
| `frontend/src/app/core/services/tenant.service.ts` | Reuse | No change; `applyTenantFonts` is the single funnel |

## Interfaces / Contracts

```ts
export type FontUrlIssue = 'https' | 'host' | 'family';
export interface FontUrlCheck { ok: boolean; issue?: FontUrlIssue; message?: string }
export function validateCustomFontUrl(raw: string): FontUrlCheck;
export function mapFontErrorCode(code: string): { slot: 'sans' | 'mono' | 'general'; message: string };
// BrandingValidation gains optional fontSansError? / fontMonoError? Strings reuse form-field error slots.
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | `validateCustomFontUrl` (non-HTTPS, foreign host, missing `family=`, valid) + `mapFontErrorCode` | Pure-function cases, no TestBed |
| Integration | Catalog change stages tokens; invalid URL blocks Aplicar; font-only change enables `Guardar Marca`; cancel reverts tokens; branding-ok/fonts-422 shows partial copy | Component spec with mocked `TenantService` (`config` signal + `applyTenantFonts` spy) |
| E2E | None | Out of scope for this change |

## Threat Matrix

N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary.

## Migration / Rollout

No migration required. Single frontend commit; revert restores prior behavior and tokens fall back to last saved fonts on next `/config/public` load. Run `lint:styles` before commit (token-only, no hex literals).

## Open Questions

- None blocking; preview-scope choice (tab-scoped snapshot-revert) confirmed by spec tab-scoped revert requirement.
