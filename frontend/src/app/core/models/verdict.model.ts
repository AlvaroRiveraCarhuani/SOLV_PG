export type JudgeVerdict =
  | 'AC'
  | 'WA'
  | 'TLE'
  | 'RE'
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
      return 'verdict-tle';
    case 'RE':
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
      return 'Correcto';
    case 'WA':
      return 'Respuesta incorrecta';
    case 'TLE':
      return 'Tiempo excedido';
    case 'RE':
      return 'Error de ejecución';
    case 'AST_BLOCKED':
    case 'AST_VIOLATION':
      return 'Regla de código';
    case 'PENDING':
      return 'Evaluando';
    default:
      return 'Sin evaluar';
  }
}

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
