export interface AcademicPeriod {
  id: string;
  name: string;
  code: string;
  is_active: boolean;
  start_date?: string;
  end_date?: string;
}

export interface TeacherCourseSummary {
  id: string;
  name: string;
  code: string;
  academic_period_id?: string;
  students_count: number;
  active_now: number;
  pending_review: number;
  at_risk: number;
}

export interface AttentionCriticalAlert {
  type: 'oom_killed' | string;
  student_id: string;
  student_name: string;
  workspace_id: string;
  subject_id: string;
  occurred_at: string;
}

export interface AttentionWarningAlert {
  type: 'ast_blocked' | string;
  student_id: string;
  student_name: string;
  exercise_id: string;
  rule_violated: string;
  occurred_at: string;
}

export interface AttentionStandardAlert {
  type: 'pending_review' | string;
  submission_id: string;
  student_name: string;
  exercise_title: string;
  submitted_at: string;
}

export interface TeacherAttentionWidget {
  critical: AttentionCriticalAlert[];
  warning: AttentionWarningAlert[];
  standard: AttentionStandardAlert[];
}

export interface TeacherLabStats {
  id: string;
  title: string;
  status: 'draft' | 'published';
  type?: 'algorithm' | 'database' | 'workspace' | string;
  language?: string;
  boilerplate?: string;
  template_id?: string;
  memory_limit_mb?: number;
  time_limit_ms?: number;
  due_date?: string;
  submissions_count: number;
  students_count: number;
  auto_graded: number;
  pending_review: number;
  at_risk: number;
  verdicts_summary?: Record<string, number>;
}

export interface SubmissionComment {
  id: string;
  tenant_id?: string;
  submission_id: string;
  author_id: string;
  author_name: string;
  line_number: number;
  comment: string;
  created_at: string;
}

export interface SubmissionQueueItem {
  id: string;
  exercise_id: string;
  exercise_title: string;
  student_id: string;
  student_name: string;
  student_email: string;
  verdict: string;
  score?: number;
  manual_override: boolean;
  execution_time_ms: number;
  memory_used_mb: number;
  submitted_at: string;
  comments_count: number;
}

export interface TestCaseReview {
  input: string;
  expected_output: string;
  is_hidden: boolean;
  actual_output?: string;
  passed: boolean;
}

export interface TeacherSubmissionReviewDTO {
  id: string;
  exercise_id: string;
  exercise_title: string;
  subject_id: string;
  subject_name: string;
  student_id: string;
  student_name: string;
  student_email: string;
  code: string;
  verdict: string;
  score?: number;
  manual_override: boolean;
  override_reason?: string;
  graded_by?: string;
  graded_by_name?: string;
  execution_time_ms: number;
  memory_used_mb: number;
  ast_result?: unknown;
  test_cases: TestCaseReview[];
  comments: SubmissionComment[];
  next_submission_id?: string;
  prev_submission_id?: string;
  submitted_at: string;
}

export interface OverrideRequestDTO {
  verdict: string;
  override_reason: string;
  score?: number;
}

export interface AddCommentRequestDTO {
  line_number: number;
  comment: string;
}

export interface EphemeralRunRequestDTO {
  code: string;
  language: string;
}

export interface EphemeralRunResult {
  submission_id: string;
  exercise_id: string;
  verdict: string;
  execution_time_ms: number;
  memory_used_mb: number;
  message: string;
  actual_json?: string;
}

export interface CreateExerciseRequestDTO {
  subject_id: string;
  title: string;
  description: string;
  type?: 'algorithm' | 'database' | 'workspace' | string;
  language?: string;
  boilerplate?: string;
  template_id?: string;
  time_limit_ms?: number;
  memory_limit_mb?: number;
  due_date?: string;
}

export interface UpdateExerciseRequestDTO {
  title?: string;
  description?: string;
  type?: 'algorithm' | 'database' | 'workspace' | string;
  language?: string;
  boilerplate?: string;
  template_id?: string;
  time_limit_ms?: number;
  memory_limit_mb?: number;
  due_date?: string;
}

export interface BulkTestCasesRequestDTO {
  test_cases: Array<{
    input: string;
    expected_output: string;
    is_hidden: boolean;
  }>;
}

export interface PlagiarismMatch {
  submission_id_a: string;
  student_id_a: string;
  student_name_a: string;
  submission_id_b: string;
  student_id_b: string;
  student_name_b: string;
  exercise_id: string;
  exercise_title: string;
  similarity: number;
  risk_level: 'critical' | 'warning' | 'info';
  matching_tokens: number;
  total_tokens_a: number;
  total_tokens_b: number;
  common_structures: string[];
}

export interface PlagiarismReport {
  subject_id: string;
  subject_name?: string;
  exercise_id?: string;
  exercise_title?: string;
  analyzed_at: string;
  total_submissions: number;
  suspect_pairs_count: number;
  matches: PlagiarismMatch[];
}

export interface TimelineKeyframe {
  offset_ms: number;
  action: 'insert' | 'delete' | 'paste' | 'checkpoint' | 'ast_change' | string;
  content: string;
  cursor_line: number;
  is_paste: boolean;
  char_count: number;
  ast_node_count?: number;
  cyclomatic_complexity?: number;
  description?: string;
  lines_added?: number;
  lines_deleted?: number;
}

export interface SubmissionTimeline {
  submission_id: string;
  student_id: string;
  student_name: string;
  total_duration_seconds: number;
  total_keystrokes: number;
  paste_events_count: number;
  paste_percentage: number;
  suspicious_paste_flag: boolean;
  keyframes: TimelineKeyframe[];
}

export interface LiveWorkspaceSession {
  workspace_id: string;
  container_id: string;
  student_id: string;
  student_name: string;
  student_email: string;
  subject_id: string;
  subject_name: string;
  status: string;
  memory_limit_mb: number;
  memory_used_mb?: number;
  cpu_percent?: number;
  oom_strikes: number;
  last_heartbeat: string;
  is_attached: boolean;
  activity_state?: 'typing' | 'idle' | 'executing' | 'oom_warning';
  wpm?: number;
  memory_history?: number[];
  cpu_history?: number[];
}

export interface TutorCommandResponse {
  container_id: string;
  command: string;
  output: string;
  exit_code: number;
  executed_at: string;
}

export interface FuzzGenerationRequest {
  exercise_id?: string;
  target_language: string;
  parameter_types: string[];
  categories: string[];
  reference_code?: string;
  count?: number;
}

export interface GeneratedFuzzCase {
  category: string;
  description: string;
  input: string;
  expected: string;
  is_public: boolean;
  weight: number;
  risk_level: 'critical' | 'warning' | 'boundary' | 'stress' | string;
  selected?: boolean;
}

export interface FuzzGenerationReport {
  total_generated: number;
  categories: Record<string, number>;
  cases: GeneratedFuzzCase[];
}

export interface ApplyFuzzCasesResponse {
  added_count: number;
  exercise_id: string;
}



