/**
 * Etiquetas visibles de los veredictos del juez.
 *
 * La interfaz no muestra las siglas internas (AC, WA, TLE...). El badge
 * traduce cada código con la tabla canónica de `verdict.model.ts`, así que los
 * flujos afirman la etiqueta humana y no el valor técnico.
 */
export const VERDICT_LABELS = {
  AC: 'Cumple',
  WA: 'No cumple',
  TLE: 'Tarda demasiado',
  MLE: 'Se quedó sin memoria',
  RE: 'Se cae al ejecutar',
  CE: 'No compila',
  VE: 'Caso de prueba inválido',
  AST_BLOCKED: 'Usó algo no permitido',
  AST_VIOLATION: 'Usó algo no permitido',
  PENDING: 'Sin calificar',
  UNKNOWN: 'Sin evaluar'
};

/** Etiqueta visible de un veredicto; un código desconocido cae en "Sin evaluar". */
export function verdictLabel(verdict) {
  const key = String(verdict || '')
    .trim()
    .toUpperCase();
  return VERDICT_LABELS[key] || VERDICT_LABELS.UNKNOWN;
}
