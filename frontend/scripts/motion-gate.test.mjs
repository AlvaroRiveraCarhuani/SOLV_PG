import { describe, expect, it } from 'vitest';

import {
  MOTION_EXCEPTIONS,
  extractAnimations,
  extractKeyframes,
  keyframeName,
  validateExceptionRegistry,
  validateMotionScss,
} from './motion-gate.mjs';

const PRIMITIVES = {
  'solv-fade-in': 'from { opacity: 0; } to { opacity: 1; }',
  'solv-rise': 'from { opacity: 0; transform: translateY(8px); } to { opacity: 1; transform: none; }',
};

const FILE = 'src/app/example/example.component.scss';
const REDUCED = '\n@media (prefers-reduced-motion: reduce) {\n  .pulse { animation: none; }\n}\n';

function validate(source, options = {}) {
  return validateMotionScss(FILE, source, { primitives: PRIMITIVES, ...options });
}

describe('motion-gate', () => {
  it('extracts keyframes and animation declarations with their line numbers', () => {
    const source = '.a {\n  animation: solv-fade-in 150ms ease;\n}\n@keyframes local-pulse {\n  to { opacity: 0.5; }\n}\n';
    expect(extractKeyframes(source)).toEqual([expect.objectContaining({ name: 'local-pulse', line: 4 })]);
    expect(extractAnimations(source)).toEqual([{ value: 'solv-fade-in 150ms ease', line: 2 }]);
  });

  it('reads the animation name from shorthands and ignores time, timing and keywords', () => {
    expect(keyframeName('1.4s var(--motion-ease-standard) infinite solv-spin')).toBe('solv-spin');
    expect(keyframeName('pulse 1.5s ease-in-out infinite alternate')).toBe('pulse');
    expect(keyframeName('none')).toBeNull();
    expect(keyframeName('inherit')).toBeNull();
  });

  it('accepts components that only use global primitives', () => {
    expect(validate('.card {\n  animation: solv-fade-in var(--motion-fast) both;\n}\n' + REDUCED)).toEqual([]);
  });

  it('accepts a local loop that is turned off under reduced motion', () => {
    expect(
      validate('@keyframes live-ring {\n  to { opacity: 0; }\n}\n.dot {\n  animation: live-ring 1.8s infinite;\n}\n' + REDUCED)
    ).toEqual([]);
  });

  it('accepts a registered exception in its own file', () => {
    expect(
      validateMotionScss(
        MOTION_EXCEPTIONS.bannerSlideDown.file,
        '@keyframes bannerSlideDown {\n  from { transform: translateY(-6px); }\n}\n.banner {\n  animation: bannerSlideDown 200ms ease;\n}\n' + REDUCED,
        { primitives: PRIMITIVES, exceptions: MOTION_EXCEPTIONS }
      )
    ).toEqual([]);
  });

  it('rejects a redefinition of a system primitive', () => {
    expect(validate('@keyframes solv-fade-in {\n  from { opacity: 0; }\n}\n')).toEqual([
      expect.stringContaining('redefine una primitiva'),
    ]);
  });

  it('rejects spin-like keyframe names instead of .motion-spin', () => {
    expect(validate('@keyframes spinner {\n  to { transform: rotate(1turn); }\n}\n')).toEqual([
      expect.stringContaining('.motion-spin'),
    ]);
  });

  it('rejects a local keyframe whose body duplicates a primitive', () => {
    expect(
      validate('@keyframes appear {\n  from { opacity: 0; }\n  to { opacity: 1; }\n}\n' + REDUCED)
    ).toEqual([expect.stringContaining('repite el movimiento de "solv-fade-in"')]);
  });

  it('rejects an unjustified local entry animation', () => {
    expect(
      validate('@keyframes slide-down {\n  from { transform: translateY(-2px); }\n}\n.banner {\n  animation: slide-down 200ms ease;\n}\n' + REDUCED)
    ).toEqual([expect.stringContaining('entrada local sin justificar')]);
  });

  it('rejects an exception reused outside its declared file', () => {
    expect(
      validateMotionScss(
        'src/app/other/other.component.scss',
        '@keyframes bannerSlideDown {\n  from { transform: translateY(-6px); }\n}\n.banner {\n  animation: bannerSlideDown 200ms ease;\n}\n' + REDUCED,
        { primitives: PRIMITIVES, exceptions: MOTION_EXCEPTIONS }
      )
    ).toEqual([expect.stringContaining('entrada local sin justificar')]);
  });

  it('rejects an animation without its reduced-motion block', () => {
    expect(validate('@keyframes live-ring {\n  to { opacity: 0; }\n}\n.dot {\n  animation: live-ring 1.8s infinite;\n}\n')).toEqual([
      expect.stringContaining('prefers-reduced-motion'),
    ]);
  });

  it('does not require reduced motion when the file only turns animations off', () => {
    expect(validate('.dot {\n  animation: none;\n}\n')).toEqual([]);
    expect(validate('.dot {\n  animation-delay: 35ms;\n}\n')).toEqual([]);
  });

  it('rejects a reference to a keyframe that exists nowhere', () => {
    expect(validate('.dot {\n  animation: ghost-pulse 1s infinite;\n}\n' + REDUCED)).toEqual([
      expect.stringContaining('"ghost-pulse" no está definida'),
    ]);
  });

  it('accepts the global catalog of styles.scss as known keyframes', () => {
    expect(
      validate('.toast {\n  animation: slideInGlobalToast 0.25s ease;\n}\n' + REDUCED, {
        knownGlobals: new Set(['slideInGlobalToast']),
      })
    ).toEqual([]);
  });

  it('ignores commented-out keyframes and animation declarations', () => {
    expect(
      validate('// @keyframes spin { to { transform: rotate(1turn); } }\n/* animation: spin 1s infinite; */\n.card {\n  color: red;\n}\n')
    ).toEqual([]);
  });

  it('requires every exception to declare a file and a reason', () => {
    expect(validateExceptionRegistry({ local: { reason: 'x' } })).toEqual([
      expect.stringContaining('no declara archivo'),
      expect.stringContaining('necesita un motivo'),
    ]);
    expect(validateExceptionRegistry({ local: { file: FILE, reason: 'Pulso de estado sin sustituto textual.' } })).toEqual([]);
  });
});
