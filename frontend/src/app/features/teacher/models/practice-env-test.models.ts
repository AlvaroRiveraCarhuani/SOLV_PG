export type PracticeEnvTestStatus =
  | 'pending'
  | 'pulling'
  | 'testing'
  | 'success'
  | 'failed'
  | 'canceled';

export interface PracticeEnvTestTool {
  name: string;
  present: boolean;
  version?: string;
  path?: string;
}

export interface PracticeEnvTestProgress {
  bytes_done: number;
  bytes_total: number;
  layer_current: number;
  layers_total: number;
  percent: number;
  current_action?: string;
}

export interface PracticeEnvTestResult {
  tools: PracticeEnvTestTool[];
  exit_code: number;
  duration_ms: number;
}

export interface PracticeEnvTestJob {
  id: string;
  exercise_id?: string;
  subject_id?: string;
  image: string;
  target_environment?: string;
  status: PracticeEnvTestStatus;
  progress: PracticeEnvTestProgress;
  result?: PracticeEnvTestResult;
  error_code?: string;
  error_message?: string;
  created_at: string;
  updated_at: string;
  finished_at?: string;
}
