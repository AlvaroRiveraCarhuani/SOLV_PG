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

export function getVerdictBadgeClass(verdict: string): string {
  switch ((verdict || '').trim().toUpperCase()) {
    case 'AC':
      return 'verdict-ac';
    case 'WA':
      return 'verdict-wa';
    case 'TLE':
    case 'MLE':
      return 'verdict-tle';
    case 'RE':
    case 'CE':
    case 'VE':
      return 'verdict-re';
    case 'AST_BLOCKED':
    case 'AST_VIOLATION':
      return 'verdict-ast';
    case 'PENDING':
      return 'verdict-pending';
    default:
      return 'verdict-unknown';
  }
}

export function getVerdictHumanLabel(verdict: string): string {
  switch ((verdict || '').trim().toUpperCase()) {
    case 'AC':
      return 'Solución correcta';
    case 'WA':
      return 'Resultado incorrecto';
    case 'TLE':
      return 'Tiempo límite excedido';
    case 'MLE':
      return 'Límite de memoria excedido';
    case 'RE':
      return 'Error durante la ejecución';
    case 'CE':
      return 'Error de compilación';
    case 'AST_BLOCKED':
    case 'AST_VIOLATION':
      return 'Restricción de código incumplida';
    case 'VE':
      return 'Error en validación de prueba';
    case 'PENDING':
      return 'Evaluando entrega';
    default:
      return 'Sin evaluar';
  }
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

export const VERDICT_FILTER_OPTIONS: { id: string; label: string; value: string }[] = [
  { id: 'all', label: 'Todos los resultados', value: 'all' },
  { id: 'AC', label: 'Solución correcta', value: 'AC' },
  { id: 'WA', label: 'Resultado incorrecto', value: 'WA' },
  { id: 'TLE', label: 'Tiempo límite excedido', value: 'TLE' },
  { id: 'RE', label: 'Error durante la ejecución', value: 'RE' },
  { id: 'AST_BLOCKED', label: 'Restricción de código incumplida', value: 'AST_BLOCKED' }
];

export const VERDICT_OVERRIDE_OPTIONS: { id: string; label: string; value: string }[] = [
  { id: 'AC', label: 'Solución correcta', value: 'AC' },
  { id: 'WA', label: 'Resultado incorrecto', value: 'WA' },
  { id: 'RE', label: 'Error durante la ejecución', value: 'RE' },
  { id: 'TLE', label: 'Tiempo límite excedido', value: 'TLE' }
];

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
