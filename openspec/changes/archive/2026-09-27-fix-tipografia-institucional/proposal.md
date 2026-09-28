# Proposal: Fix Institutional Typography (Identidad Tab)

## Intent

Three reported symptoms trace to three frontend defects: custom font URLs accepted without validation, chosen fonts never reaching `:root` tokens (or the local session after save), and `Guardar Marca` staying disabled for font-only changes because font dirty signals are never consumed. Fix all three frontend-only; the backend validator is already fail-closed.

## Scope

### In Scope
- Inline custom-URL validation mirroring backend cheap rules (HTTPS, `fonts.googleapis.com` allowlist, `family=` presence); block "Aplicar" on invalid input; map 422 codes to inline messages (reachability stays backend-only)
- Font-aware `isDirty`/`canSave` consuming `fontSansDirty`/`fontMonoDirty`; warn instead of silent Inter fallback when a custom URL lacks `family=`
- Route preview and post-save through `TenantService.applyTenantFonts` (`:root` tokens); post-save persists font fields into `tenantService.config` and calls `applyBranding`
- Tab-scoped preview override that reverts on Cancel/Discard (assumption, confirm in spec)
- Frontend spec cases: catalog change, valid/invalid custom URLs, font-only gating, two-PUT partial failure
- UI copy: "Aplicar" = stage to preview; "Guardar Marca" = persist all

### Out of Scope
- Backend changes (validator, 422 codes, 5s probe unchanged)
- Combobox replacement for catalog selects (follow-up change)
- Backup `CORRUPTO 0 MB` issue (separate change)
- Atomic single-PUT save (existing partial-success semantics kept and documented)

## Capabilities

### New Capabilities
- `admin-branding-typography`: institutional font selection, inline validation, token-based preview, and persistence in the Identidad tab

### Modified Capabilities
- None (`openspec/specs/` is empty; no existing spec behavior changes)

## Approach

Combine explore approaches 1+2: fail-closed frontend mirroring + font-aware save gating, with all font application routed through `:root` tokens via the existing `applyTenantFonts` funnel. Standalone signals, `computed()` gating, `inject()` DI; no new components. Tokens only (`var(--font-*)`), no font literals; run `lint:styles` before commit.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `frontend/.../tabs/admin-config-identidad.component.ts` | Modified | URL validation, dirty gating, token apply, post-save apply |
| `frontend/.../tabs/admin-config-identidad.component.html` | Modified | Aplicar gating, hint copy, drop per-option inline font styles |
| `frontend/.../admin-config-identidad.service.ts` | Modified | Font rules in `validate()`; keep font fields in save tap |
| `frontend/src/app/shared/curated-fonts.ts` | Modified | Surface missing-`family=` instead of silent fallback |
| `frontend/.../tabs/admin-config-identidad.component.spec.ts` | Modified | Font cases (catalog, URLs, gating, partial failure) |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Two-PUT partial success confuses user | Med | Explicit copy for branding-saved/fonts-rejected state |
| Native OS dropdown popups ignore webfonts | High | Documented limit; combobox follow-up, not this change |
| 400-line budget overflow (1+2 + tests) | Med | Split validation/gating vs token application if forecast High |

## Rollback Plan

Revert the single frontend commit; tokens fall back to last saved fonts on next `/config/public` load. No migration or backend state involved.

## Dependencies

- None (backend validator and 422 codes already shipped)

## Success Criteria

- [ ] Invalid custom URL blocked inline before save; valid URL applies to preview
- [ ] Font-only change enables `Guardar Marca`; save updates session without reload
- [ ] New spec cases pass; `lint:styles` and build green
