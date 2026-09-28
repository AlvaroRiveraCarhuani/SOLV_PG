import { describe, expect, it } from 'vitest';
import { MACHINE_DATA_CATALOG } from './machine-data-catalog.mjs';

describe('machine-data catalog integrity', () => {
  it('has unique names with category, example and origin', () => {
    const names = MACHINE_DATA_CATALOG.map((entry) => entry.name);
    expect(new Set(names).size).toBe(names.length);
    for (const entry of MACHINE_DATA_CATALOG) {
      expect(entry.name).toBeTruthy();
      expect(entry.category).toBeTruthy();
      expect(entry.example).toBeTruthy();
      expect(entry.since).toBeTruthy();
    }
  });

  it('covers id-suffixed technical identifiers via rule, not enumeration', () => {
    const names = new Set(MACHINE_DATA_CATALOG.map((entry) => entry.name));
    expect(names.has('id')).toBe(true);
  });
});
