/**
 * Diccionario canónico i18n para el módulo del juez de ejercicios (spec.md §4).
 * Contiene exactamente las 17 claves oficiales EJ-*.
 */
export const JUDGE_I18N = {
  'EJ-CMP-TITLE': 'Comparador de salida',
  'EJ-CMP-EXACT': 'Coincidencia exacta',
  'EJ-CMP-UNORDERED': 'Sin orden',
  'EJ-CMP-FLOAT': 'Tolerancia numérica',
  'EJ-CMP-CUSTOM': 'Checker del docente',
  'EJ-CMP-UNKNOWN': 'Comparador desconocido',
  'EJ-CHECKER-INVALID': 'Checker inválido',
  'EJ-CMP-PARAMS': 'Parámetros del comparador',
  'EJ-REF-TITLE': 'Solución de referencia',
  'EJ-REF-HINT': 'Se utiliza para el dry-run y la recalificación',
  'EJ-DRY-RUN': 'Probar con referencia',
  'EJ-DRY-PROGRESS': 'Caso en curso',
  'EJ-STALE': 'Requiere nueva comprobación',
  'EJ-PUB-BLOCKED': 'La publicación se encuentra bloqueada',
  'EJ-PUB-REF': 'Falta la solución de referencia',
  'EJ-ERR-RANGE': 'Valor fuera del rango admitido: {campo}',
  'EJ-TELEMETRY': 'Métricas de ejecución'
} as const;

export type JudgeI18nKey = keyof typeof JUDGE_I18N;

/**
 * Formatea el mensaje de error de rango interpolando el nombre del campo.
 */
export function formatRangeError(campo: string): string {
  return JUDGE_I18N['EJ-ERR-RANGE'].replace('{campo}', campo);
}
