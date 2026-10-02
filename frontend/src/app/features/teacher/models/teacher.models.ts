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
  due_date?: string;
}

export interface UpdateExerciseRequestDTO {
  title?: string;
  description?: string;
  due_date?: string;
}

export interface BulkTestCasesRequestDTO {
  test_cases: Array<{
    input: string;
    expected_output: string;
    is_hidden: boolean;
  }>;
}
