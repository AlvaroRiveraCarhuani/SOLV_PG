export type JudgeVerdict =
  | 'AC'
  | 'WA'
  | 'TLE'
  | 'MLE'
  | 'RE'
  | 'CE'
  | 'VE'
  | 'AST_BLOCKED'
  | 'AST_VIOLATION'
  | 'PENDING';

/** Claves de la tabla canónica de resultados pedagógicos. */
export type OutcomeKey =
  | 'AC'
  | 'WA'
  | 'TLE'
  | 'MLE'
  | 'RE'
  | 'CE'
  | 'VE'
  | 'AST_BLOCKED'
  | 'PENDING'
  | 'UNKNOWN';

export interface OutcomeCopy {
  /** Capa 1: lo único que ve una persona en una lista o en un badge. */
  label: string;
  /** Capa 2: una frase que explica qué pasó, sin jerga del juez. */
  detail: string;
  /** Clase visual del badge (la semántica de color vive en el SCSS compartido). */
  badgeClass: string;
}

/**
 * Fuente única del lenguaje humano de los resultados. El código del juez
 * (AC, WA, TLE...) sigue siendo un valor técnico interno: se conserva para
 * auditoría, comparadores y métricas, pero nunca se muestra ni se escribe a
 * mano en un template. Toda etiqueta, filtro, tooltip y opción de override
 * sale de esta tabla.
 */
export const OUTCOME_COPY: Record<OutcomeKey, OutcomeCopy> = {
  AC: {
    label: 'Cumple',
    detail: 'Pasa todos los casos de prueba de la práctica.',
    badgeClass: 'verdict-ac'
  },
  WA: {
    label: 'No cumple',
    detail: 'El programa se ejecutó, pero el resultado no coincide con lo esperado en al menos un caso.',
    badgeClass: 'verdict-wa'
  },
  TLE: {
    label: 'Tarda demasiado',
    detail: 'No terminó dentro del tiempo permitido para esta práctica.',
    badgeClass: 'verdict-tle'
  },
  MLE: {
    label: 'Se quedó sin memoria',
    detail: 'Usó más memoria de la que permite esta práctica.',
    badgeClass: 'verdict-tle'
  },
  RE: {
    label: 'Se cae al ejecutar',
    detail: 'El programa se detuvo con un error durante la ejecución.',
    badgeClass: 'verdict-re'
  },
  CE: {
    label: 'No compila',
    detail: 'El código no se pudo compilar: el error está en la sintaxis o en los tipos.',
    badgeClass: 'verdict-re'
  },
  VE: {
    label: 'Caso de prueba inválido',
    detail: 'La prueba tiene un problema de formato; no es un fallo del código del alumno.',
    badgeClass: 'verdict-re'
  },
  AST_BLOCKED: {
    label: 'Usó algo no permitido',
    detail: 'El código usó una instrucción que el enunciado no permite.',
    badgeClass: 'verdict-ast'
  },
  PENDING: {
    label: 'Sin calificar',
    detail: 'La entrega está registrada y espera la revisión del docente.',
    badgeClass: 'verdict-pending'
  },
  UNKNOWN: {
    label: 'Sin evaluar',
    detail: 'Todavía no hay resultado para esta entrega.',
    badgeClass: 'verdict-unknown'
  }
};

/** Variantes que comparten una misma entrada de la tabla. */
const OUTCOME_ALIASES: Record<string, OutcomeKey> = {
  AST_VIOLATION: 'AST_BLOCKED'
};

/**
 * Normaliza cualquier valor que llegue del backend a una clave de la tabla.
 * Valores vacíos o desconocidos caen en UNKNOWN en vez de romper el badge.
 */
export function resolveOutcomeKey(verdict: string | undefined | null): OutcomeKey {
  const normalized = (verdict || '').trim().toUpperCase();
  if (!normalized) {
    return 'UNKNOWN';
  }
  const aliased = OUTCOME_ALIASES[normalized];
  if (aliased) {
    return aliased;
  }
  return (Object.prototype.hasOwnProperty.call(OUTCOME_COPY, normalized)
    ? normalized
    : 'UNKNOWN') as OutcomeKey;
}

export function getVerdictOutcome(verdict: string | undefined | null): OutcomeCopy {
  return OUTCOME_COPY[resolveOutcomeKey(verdict)];
}

export function getVerdictHumanLabel(verdict: string | undefined | null): string {
  return getVerdictOutcome(verdict).label;
}

export function getVerdictOutcomeDetail(verdict: string | undefined | null): string {
  return getVerdictOutcome(verdict).detail;
}

export function getVerdictBadgeClass(verdict: string | undefined | null): string {
  return getVerdictOutcome(verdict).badgeClass;
}

// Opciones compartidas por los filtros del docente y del estudiante. El
// vocabulario visible es siempre humano: los codigos del juez quedan solo
// como valor tecnico interno.
// Unifica las dos variantes de restriccion de codigo bajo la opcion humana
// AST_BLOCKED: quien filtra por restriccion debe ver tambien AST_VIOLATION.
export function matchesVerdictFilter(verdict: string | undefined | null, filter: string): boolean {
  if (!filter || filter === 'all') {
    return true;
  }
  const normalizedVerdict = (verdict || '').trim().toUpperCase();
  const normalizedFilter = filter.trim().toUpperCase();
  if (normalizedFilter === 'AST_BLOCKED' || normalizedFilter === 'AST_VIOLATION') {
    return normalizedVerdict === 'AST_BLOCKED' || normalizedVerdict === 'AST_VIOLATION';
  }
  return normalizedVerdict === normalizedFilter;
}

/** Orden de los filtros de resultados en las vistas de entregas. */
const FILTER_OUTCOME_KEYS: OutcomeKey[] = ['PENDING', 'AC', 'WA', 'TLE', 'RE', 'AST_BLOCKED'];

/** Resultados que el docente puede asignar manualmente al corregir. */
const OVERRIDE_OUTCOME_KEYS: OutcomeKey[] = ['AC', 'WA', 'RE', 'TLE'];

function toOption(key: OutcomeKey): { id: string; label: string; value: string } {
  return { id: key, label: OUTCOME_COPY[key].label, value: key };
}

export const VERDICT_FILTER_OPTIONS: { id: string; label: string; value: string }[] = [
  { id: 'all', label: 'Todos los resultados', value: 'all' },
  ...FILTER_OUTCOME_KEYS.map(toOption)
];

export const VERDICT_OVERRIDE_OPTIONS: { id: string; label: string; value: string }[] =
  OVERRIDE_OUTCOME_KEYS.map(toOption);

export interface EvaluationResult {
  submission_id: string;
  verdict: JudgeVerdict;
  execution_time_ms: number;
  memory_used_kb: number;
  score: number;
  error_message?: string;
  ast_rule_infringed?: string;
  completed_at: string;
}
