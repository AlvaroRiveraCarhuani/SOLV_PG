## Exploration: fix-tipografia-institucional

### Current State

Institutional typography is a white-label pair (`font_sans_family` for UI, `font_mono_family`
for machine data) stored as `"cat:<slug>"` (curated catalog) or `"url:https://..."` (custom
Google Fonts CSS). The catalog (6 sans + 3 mono) is kept in sync between
`frontend/src/app/shared/curated-fonts.ts` and `backend/internal/core/services/typography_service.go`
(ADR-038 contract noted in both files).

How fonts reach the UI today:

- `TenantService.applyTenantFonts()` (`frontend/src/app/core/services/tenant.service.ts:145`)
  injects one `<link>` per Google Fonts CSS (idempotent, hashed id) and sets `--font-sans` /
  `--font-mono` on `:root`. Every view consumes these tokens (`body`, `select`/`option`,
  headings, mono surfaces); no hardcoded `font-family` literals were found in views —
  only `inherit` and `var(--font-*, fallback)` fallbacks. Tokens are defined once in
  `styles/_primitives.scss:56-57`; nothing in `_tenant.scss`/`_semantic.scss` overrides them.
- The Identidad tab (`admin-config-identidad.component.ts/html/scss`) keeps font choice in
  **local signals** (`fontSansValue`, `fontMonoValue`) and renders preview through **local
  inline stacks** (`fontPreviewStack`, `fontMonoPreviewStack` → `[style.font-family]`), NOT
  through `:root` tokens. The specimen and the emulated judge panel update instantly.
- Save is two sequential PUTs to `PUT /api/v1/admin/branding`: branding first, then fonts
  (`save()` → `saveFonts()`, component.ts:300-341). Backend validates fonts centrally in
  `ResolveFontConfig` (`typography_service.go:75`): `cat:` must exist in catalog and match
  the slot kind; `url:` must be HTTPS, host allowlist `fonts.googleapis.com`, and **reachable**
  (HTTP GET with 5s timeout). Failures return 422 with codes `font_slug_unknown`,
  `font_kind_mismatch`, `font_format_invalid`, `font_url_invalid`,
  `font_url_host_not_allowed`, `font_url_unreachable` (`admin_handler.go:341-358`).

Findings per reported problem (root causes, verified in code):

1. **Custom URL inputs accept anything.** `applyCustomSansUrl/MonoUrl` (component.ts:139-151)
   only check `trim() !== ''` — zero URL validation. `AdminConfigIdentidadService.validate()`
   covers name/color/logo/email only; fonts are absent. The hint text ("backend validates
   HTTPS, domain and reachability before saving") is true only at save time: the error
   surfaces as `fontError` banner AFTER the second PUT fails, while branding from the first
   PUT is already persisted (partial-success semantics). "Aplicar a UI / Aplicar a mono" only
   set the local signal (preview), never touch validation or persistence.
2. **Selects/dropdowns/lists do not reflect the chosen font.** Two stacked causes:
   - (a) The chosen font never reaches `:root` tokens in the current session. Preview works
     because it uses local inline styles; the rest of the app reads `var(--font-*)`, which
     only changes via `applyBranding` → `applyTenantFonts`. Worse, even after a successful
     save the local session keeps stale tokens: the `saveBranding` tap
     (admin-config-identidad.service.ts:93-107) rebuilds `TenantConfig` with only the 4
     branding fields (no `font_sans_family`/`font_mono_family`) and `saveFonts` success
     (component.ts:326-335) only resets `initial` + toast — it never updates
     `tenantService.config` nor calls `applyBranding`. Other sessions pick fonts up only
     after `/config/public` cache expiry (up to 5 min). So selects/lists (correctly wired to
     tokens, including global `select/option/optgroup` rules in `styles.scss:52-91` and the
     design-system select contract) keep showing the old font.
   - (b) Native OS dropdown popups render with system fonts regardless of CSS `option`
     styling in most browsers; the tab additionally styles each `<option>` with the literal
     `[style.font-family]="font.name"`, which renders a fallback unless that webfont's
     stylesheet was actually loaded (loading only happens in `applyTenantFonts`, i.e. after
     save + re-init).
3. **Preview updates but Guardar Marca stays disabled.** `canSave` (component.ts:103-105) =
   `validation().valid && isDirty() && !isSaving() && !isUploadingLogo()`, and `isDirty`
   (component.ts:90-101) compares only the 4 branding fields. `fontSansDirty`/`fontMonoDirty`
   (component.ts:120-121) are computed but **never consumed** — dead signals. Font-only
   changes therefore can never enable save. Semantics today: "Aplicar a UI / Aplicar a mono"
   = stage into local preview state; "Guardar Marca" = persist branding + fonts via two PUTs.
   An unapplied custom URL sitting in the input (never clicked "Aplicar") is silently dropped
   by save/reset.

Existing tests (`admin-config-identidad.component.spec.ts`, 11 cases) cover branding
dirty-gating, hex/logo validation and preview HSL, but contain **zero font cases** (no
catalog change, no custom URL, no `fontSansDirty`, no two-PUT flow). Backend
`typography_service_test.go` exists for the validator.

### Affected Areas

- `frontend/src/app/features/admin/configuracion/tabs/admin-config-identidad.component.ts` — custom URL apply without validation; `isDirty`/`canSave` ignore font signals; `saveFonts` never applies fonts locally
- `frontend/src/app/features/admin/configuracion/tabs/admin-config-identidad.component.html` — hint text over-promises; `Aplicar` buttons gated on non-empty only; per-option inline font styles
- `frontend/src/app/features/admin/configuracion/admin-config-identidad.service.ts` — `validate()` has no font rules; `saveBranding` tap drops font fields when updating `TenantService`
- `frontend/src/app/shared/curated-fonts.ts` — `fontValueToFamily` silently falls back to Inter when a custom URL lacks `family=` (masks garbage input in preview)
- `frontend/src/app/core/services/tenant.service.ts` — `applyTenantFonts` is the correct single funnel; needs reuse for instant local apply, not only init/save paths
- `backend/internal/core/services/typography_service.go` + `backend/internal/delivery/http/admin_handler.go:341-358` — validator is correct and fail-closed; frontend must mirror its cheap rules and 422 codes
- `frontend/src/app/features/admin/configuracion/tabs/admin-config-identidad.component.spec.ts` — no font coverage
- Out of scope: backup `CORRUPTO 0 MB` (separate later change, explicitly excluded)

### Approaches

1. **Fail-closed frontend mirroring + font-aware save gating** — validate custom URLs inline
   (HTTPS + `fonts.googleapis.com` allowlist + `family=` presence; reachability stays
   backend-only, mirrored from 422 codes to inline messages), include `fontSansDirty` /
   `fontMonoDirty` in `isDirty`/`canSave`, and block "Aplicar" on invalid URLs.
   - Pros: fixes problems 1 and 3 at the source; matches hardening change precedent
     (fail-closed + 422 mirror); no backend changes; small diff
   - Cons: duplicates allowlist constant FE/BE (already duplicated catalog today — accepted pattern)
   - Effort: Low/Medium

2. **Route all font application through tokens** — on catalog pick / valid custom apply, call
   `TenantService.applyTenantFonts` (or an extracted pure preview) so the whole tab/app
   previews via `:root` tokens; on `saveFonts` success, persist font fields into
   `tenantService.config` and call `applyBranding` so the session matches other views
   without reload.
   - Pros: fixes problem 2 structurally; removes preview-vs-app divergence (inline stacks
     vs tokens); makes selects/lists/combobox behave identically
   - Cons: applying un-saved fonts globally changes the admin's own session before save —
     needs explicit "preview scope" decision (tab-scoped vs app-scoped) to avoid confusion
   - Effort: Medium

3. **Replace catalog `<select>` with combobox primitive** — the design system already owns a
   `combobox` contract (closed Inter `sm`, open list Inter `xs`) that renders webfonts
   reliably, unlike native OS option popups.
   - Pros: solves the irreducible native-popup rendering limit; consistent with system inventory
   - Cons: bigger change; does not fix gating/persistence bugs alone — only makes sense stacked on 1+2
   - Effort: Medium

### Recommendation

Combine approach 1 (required: inline validation mirroring `typography_service.go` cheap rules,
font-aware `isDirty`/`canSave`, `Aplicar` gating, `family=`-missing warning instead of silent
Inter fallback) with approach 2 (required: `saveBranding` tap preserves/applies font fields;
decide preview scope — recommended tab-scoped token override that reverts on Cancel, so the
admin's session is never silently re-themed). Approach 3 is an optional follow-up once 1+2
land, since native-popup rendering is cosmetic next to the persistence bugs. Add frontend spec
cases for catalog change, invalid/valid custom URLs, font-only dirty gating, and two-PUT
partial failure; no backend changes needed (validator already fail-closed). Clarify button
semantics in UI copy: "Aplicar" = stage to preview, "Guardar Marca" = persist all (dirty-gated).

### Risks

- Two-PUT partial success (branding saved, fonts rejected) already exists — proposal must keep
  explicit copy for it, not introduce a new atomicity expectation without backend work
- 5s backend reachability probe runs inside the PUT; frontend must keep save UX non-blocking
  with `isSaving` feedback (already present)
- Custom URL without `family=` currently renders as Inter in preview — changing to a warning
  alters existing (buggy) behavior; acceptable, must be stated in proposal
- 400-line review budget: 1+2 plus tests may approach the limit; split validation/gating vs
  token-application if the task forecast says High

### Ready for Proposal

Yes — scope is typography-only, root causes are located with file:line evidence, no backend
changes required, and the hardening change (`hardening-respaldos-mantenimiento`) already
established the fail-closed + 422-mirror pattern to reuse. Tell the user the three reported
symptoms trace to three concrete defects (unvalidated apply, font fields dropped from
dirty-gate and from local post-save apply, native-popup limits) and propose approaches 1+2.
