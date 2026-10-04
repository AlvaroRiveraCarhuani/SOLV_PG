import { JUDGE_I18N, formatRangeError } from './judge-i18n';

describe('JUDGE_I18N Dictionary', () => {
  const expectedKeys = [
    'EJ-CMP-TITLE',
    'EJ-CMP-EXACT',
    'EJ-CMP-UNORDERED',
    'EJ-CMP-FLOAT',
    'EJ-CMP-CUSTOM',
    'EJ-CMP-UNKNOWN',
    'EJ-CHECKER-INVALID',
    'EJ-CMP-PARAMS',
    'EJ-REF-TITLE',
    'EJ-REF-HINT',
    'EJ-DRY-RUN',
    'EJ-DRY-PROGRESS',
    'EJ-STALE',
    'EJ-PUB-BLOCKED',
    'EJ-PUB-REF',
    'EJ-ERR-RANGE',
    'EJ-TELEMETRY'
  ];

  it('should contain exactly 17 canonical keys', () => {
    const actualKeys = Object.keys(JUDGE_I18N);
    expect(actualKeys.length).toBe(17);
    expect(actualKeys.sort()).toEqual(expectedKeys.sort());
  });

  it('should format EJ-ERR-RANGE with interpolation', () => {
    const formatted = formatRangeError('time_limit_ms');
    expect(formatted).toBe('Valor fuera del rango admitido: time_limit_ms');
  });

  it('should have exact expected text values for all keys', () => {
    expect(JUDGE_I18N['EJ-CMP-TITLE']).toBe('Comparador de salida');
    expect(JUDGE_I18N['EJ-CMP-EXACT']).toBe('Coincidencia exacta');
    expect(JUDGE_I18N['EJ-CMP-UNORDERED']).toBe('Sin orden');
    expect(JUDGE_I18N['EJ-CMP-FLOAT']).toBe('Tolerancia numérica');
    expect(JUDGE_I18N['EJ-CMP-CUSTOM']).toBe('Checker del docente');
    expect(JUDGE_I18N['EJ-CMP-UNKNOWN']).toBe('Comparador desconocido');
    expect(JUDGE_I18N['EJ-CHECKER-INVALID']).toBe('Checker inválido');
    expect(JUDGE_I18N['EJ-CMP-PARAMS']).toBe('Parámetros del comparador');
    expect(JUDGE_I18N['EJ-REF-TITLE']).toBe('Solución de referencia');
    expect(JUDGE_I18N['EJ-REF-HINT']).toBe('Se utiliza para el dry-run y la recalificación');
    expect(JUDGE_I18N['EJ-DRY-RUN']).toBe('Probar con referencia');
    expect(JUDGE_I18N['EJ-DRY-PROGRESS']).toBe('Caso en curso');
    expect(JUDGE_I18N['EJ-STALE']).toBe('Requiere nueva comprobación');
    expect(JUDGE_I18N['EJ-PUB-BLOCKED']).toBe('La publicación se encuentra bloqueada');
    expect(JUDGE_I18N['EJ-PUB-REF']).toBe('Falta la solución de referencia');
    expect(JUDGE_I18N['EJ-TELEMETRY']).toBe('Métricas de ejecución');
  });
});
