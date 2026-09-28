import { describe, expect, it } from 'vitest';
import {
  validateGlobalControlTypography,
  validateGlobalTimeTypography,
  validateTemplate
} from './date-format-gate.mjs';

describe('date-format-gate template validation', () => {
  it('accepts dateText interpolation inside a semantic time element', () => {
    expect(validateTemplate('src/example.component.html', '<time>{{ createdAt | dateText }}</time>')).toEqual([]);
  });

  it('rejects dateText interpolation outside a semantic time element with a file path', () => {
    expect(validateTemplate('src/example.component.html', '<span>{{ createdAt | dateText }}</span>')).toEqual([
      expect.stringContaining('src/example.component.html'),
    ]);
  });

  it('rejects Angular date pipes, including inside time elements', () => {
    expect(validateTemplate('src/example.component.html', '<time>{{ createdAt | date }}</time>')).toEqual([
      expect.stringContaining('src/example.component.html'),
    ]);
  });

  it('rejects styles without a global time rule', () => {
    expect(validateGlobalTimeTypography('.font-mono { font-family: var(--font-mono); }')).toEqual([
      expect.stringContaining('falta la regla global time'),
    ]);
  });

  it('rejects a global time rule that does not use the mono font', () => {
    expect(validateGlobalTimeTypography('time { font-family: var(--font-sans); font-variant-numeric: tabular-nums; }')).toEqual([
      expect.stringContaining('var(--font-mono)'),
    ]);
  });

  it('rejects a global time rule without tabular numbers', () => {
    expect(validateGlobalTimeTypography('time { font-family: var(--font-mono); }')).toEqual([
      expect.stringContaining('números tabulares'),
    ]);
  });

  it('accepts global controls that use the configured UI font', () => {
    expect(
      validateGlobalControlTypography(
        'input, select, textarea, optgroup, option, button { font-family: var(--font-sans) !important; }'
      )
    ).toEqual([]);
  });

  it('rejects global controls that use the machine font', () => {
    expect(
      validateGlobalControlTypography(
        'input, select, textarea, optgroup, option, button { font-family: var(--font-mono); }'
      )
    ).toEqual([expect.stringContaining('var(--font-sans)')]);
  });
});