export type JudgeVerdict = 'AC' | 'WA' | 'TLE' | 'RE' | 'AST_BLOCKED' | 'PENDING';

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
