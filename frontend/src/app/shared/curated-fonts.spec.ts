import { describe, it, expect } from 'vitest';
import { validateCustomFontUrl } from './curated-fonts';
import {
  mapFontErrorCode,
  mergeBrandingIntoConfig
} from '../features/admin/configuracion/admin-config-identidad.service';

describe('validateCustomFontUrl', () => {
  it('rejects non-HTTPS URLs with an https issue', () => {
    const check = validateCustomFontUrl('http://fonts.googleapis.com/css2?family=Inter');
    expect(check.ok).toBe(false);
    expect(check.issue).toBe('https');
    expect(check.message).toContain('https');
  });

  it('rejects HTTPS URLs outside the allowlist with a host issue', () => {
    const check = validateCustomFontUrl('https://cdn.evil.example.com/font.css?family=Inter');
    expect(check.ok).toBe(false);
    expect(check.issue).toBe('host');
    expect(check.message).toContain('fonts.googleapis.com');
  });

  it('warns on allowlisted URLs missing family= instead of silently falling back', () => {
    const check = validateCustomFontUrl('https://fonts.googleapis.com/css2?display=swap');
    expect(check.ok).toBe(false);
    expect(check.issue).toBe('family');
    expect(check.message).toContain('family');
  });

  it('accepts HTTPS allowlisted URLs with family=', () => {
    const check = validateCustomFontUrl(
      'https://fonts.googleapis.com/css2?family=Public+Sans:wght@400;600;700&display=swap'
    );
    expect(check.ok).toBe(true);
    expect(check.issue).toBeUndefined();
  });

  it('rejects empty input', () => {
    const check = validateCustomFontUrl('   ');
    expect(check.ok).toBe(false);
    expect(check.issue).toBe('https');
  });
});

describe('mapFontErrorCode', () => {
  it('maps font_slug_unknown to the general slot', () => {
    const mapped = mapFontErrorCode('font_slug_unknown');
    expect(mapped.slot).toBe('general');
    expect(mapped.message.length).toBeGreaterThan(0);
  });

  it('maps font_kind_mismatch to the general slot', () => {
    const mapped = mapFontErrorCode('font_kind_mismatch');
    expect(mapped.slot).toBe('general');
    expect(mapped.message.length).toBeGreaterThan(0);
  });

  it('maps font_format_invalid to the general slot', () => {
    const mapped = mapFontErrorCode('font_format_invalid');
    expect(mapped.slot).toBe('general');
    expect(mapped.message.length).toBeGreaterThan(0);
  });

  it('maps font_url_host_not_allowed to the general slot', () => {
    const mapped = mapFontErrorCode('font_url_host_not_allowed');
    expect(mapped.slot).toBe('general');
    expect(mapped.message).toContain('fonts.googleapis.com');
  });

  it('maps font_url_unreachable to the general slot', () => {
    const mapped = mapFontErrorCode('font_url_unreachable');
    expect(mapped.slot).toBe('general');
    expect(mapped.message.length).toBeGreaterThan(0);
  });

  it('falls back to the general slot for unknown codes', () => {
    const mapped = mapFontErrorCode('something_else');
    expect(mapped.slot).toBe('general');
    expect(mapped.message.length).toBeGreaterThan(0);
  });
});

describe('mergeBrandingIntoConfig', () => {
  const base = {
    tenant_id: 't1',
    slug: 'uab',
    institution_name: 'UAB',
    logo_url: '',
    tenant_primary_color: '#2563EB',
    support_email: 's@u.edu.bo',
    font_sans_family: 'cat:inter',
    font_mono_family: 'cat:jetbrains-mono'
  };

  it('preserves font fields when the payload carries them', () => {
    const merged = mergeBrandingIntoConfig(base, {
      institution_name: 'UAB',
      logo_url: '',
      tenant_primary_color: '#2563EB',
      support_email: 's@u.edu.bo',
      font_sans_family: 'cat:lato',
      font_mono_family: 'cat:fira-code'
    });
    expect(merged.font_sans_family).toBe('cat:lato');
    expect(merged.font_mono_family).toBe('cat:fira-code');
  });

  it('keeps saved fonts when the payload has no font fields', () => {
    const merged = mergeBrandingIntoConfig(base, {
      institution_name: 'UAB',
      logo_url: '',
      tenant_primary_color: '#16A34A',
      support_email: 's@u.edu.bo'
    });
    expect(merged.font_sans_family).toBe('cat:inter');
    expect(merged.font_mono_family).toBe('cat:jetbrains-mono');
    expect(merged.tenant_primary_color).toBe('#16A34A');
  });
});
