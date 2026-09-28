# admin-branding-typography Specification

## Purpose

Institutional font selection (sans for UI text, mono for machine data) in the Identidad tab: inline validation, token-based preview, and persistence without reload.

## Requirements

### Requirement: Custom URL inline validation

The system MUST validate custom font URLs inline against the backend cheap rules (HTTPS, `fonts.googleapis.com` allowlist, `family=` presence) and MUST block preview-apply on invalid input. Reachability MUST stay backend-only.

#### Scenario: Non-HTTPS URL rejected

- GIVEN a custom font URL without HTTPS
- WHEN the admin requests preview-apply
- THEN apply is blocked with an inline HTTPS error

#### Scenario: Non-allowlisted host rejected

- GIVEN an HTTPS URL outside the allowlist
- WHEN the admin requests preview-apply
- THEN apply is blocked with an inline host error

#### Scenario: Missing family parameter warns

- GIVEN an allowlisted URL without `family=`
- WHEN the admin requests preview-apply
- THEN apply is blocked with an inline warning (no silent fallback)

#### Scenario: Valid URL stages preview

- GIVEN an HTTPS allowlisted URL with `family=`
- WHEN the admin requests preview-apply
- THEN the font stages into preview without persisting

#### Scenario: Backend 422 codes map inline

- GIVEN a save rejected with `font_slug_unknown`, `font_kind_mismatch`, `font_format_invalid`, `font_url_host_not_allowed`, or `font_url_unreachable`
- WHEN the response arrives
- THEN the matching field shows the corresponding inline message

### Requirement: Font-aware save gating

The system MUST treat font-only changes as dirty and MUST enable persistence when fonts differ from saved values.

#### Scenario: Font-only change enables save

- GIVEN only the sans or mono selection differs
- WHEN dirty state computes
- THEN the form is dirty and Guardar Marca is enabled

#### Scenario: Unapplied input never persists

- GIVEN a custom URL typed but never preview-applied
- WHEN the admin saves or resets
- THEN the unapplied input is dropped and disclosed

### Requirement: Token-routed preview with tab-scoped revert

The system MUST render font preview through `:root` tokens (`--font-sans`, `--font-mono`) and MUST revert preview overrides on Cancel or Discard.

#### Scenario: Preview reaches tokens

- GIVEN a catalog pick or valid custom apply
- WHEN preview stages
- THEN the tokens update and the whole tab reflects the font

#### Scenario: Cancel reverts preview

- GIVEN staged but unsaved font preview
- WHEN the admin cancels or discards
- THEN tokens revert to the last saved fonts

### Requirement: Post-save session apply

The system MUST apply saved fonts to the local session immediately after a successful save, with no reload.

#### Scenario: Save updates session

- GIVEN a successful fonts save
- WHEN the save completes
- THEN local session tokens match the saved fonts without reload

### Requirement: Partial-save disclosure

The system MUST disclose double-PUT partial success explicitly when branding persists but fonts are rejected.

#### Scenario: Branding saved, fonts rejected

- GIVEN the branding PUT succeeds and the fonts PUT fails
- WHEN the failure arrives
- THEN the UI states branding is saved and fonts were rejected, with cause

### Requirement: Action copy semantics

The system MUST label preview-staging as "Aplicar" and full persistence as "Guardar Marca", with hints stating Aplicar never persists.

#### Scenario: Copy disambiguates actions

- GIVEN the typography section of the Identidad tab
- WHEN the admin reads the actions
- THEN hints state Aplicar stages preview and Guardar Marca persists all

### Requirement: Automated spec coverage

The system MUST cover catalog change, valid and invalid custom URLs, font-only gating, and two-PUT partial failure with automated frontend cases.

#### Scenario: Font cases pass

- GIVEN the typography cases listed above
- WHEN the frontend suite runs
- THEN all listed cases pass

Known limit: native OS dropdown popups may render system fonts regardless of webfont styling; this change MUST NOT claim native-popup fidelity (combobox follow-up out of scope).
