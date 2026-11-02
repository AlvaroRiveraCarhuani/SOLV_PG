import type { ExerciseContract } from '../exercise-editor/steps/step-contract/format-builder/format-builder.models';

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

/**
 * Read model del panel docente: responde "¿qué me espera hoy?" con los conteos
 * ya resueltos. La vista no suma, no agrupa y no recorta.
 */
export interface TeacherOverviewPeriod {
  id: string;
  name: string;
  code: string;
  is_active: boolean;
}

export interface TeacherOverviewSession {
  id: string;
  subject_id: string;
  subject_code: string;
  subject_name: string;
  exercise_title?: string;
  opens_at: string;
  ends_at: string;
  closes_at: string | null;
  enrolled: number;
  present: number;
}

/** Práctica publicada que vence dentro de la ventana de la columna de pendientes. */
export interface TeacherOverviewDue {
  subject_id: string;
  subject_code: string;
  subject_name: string;
  exercise_id: string;
  exercise_title: string;
  /** Urgencia calculada en el servidor: overdue | today | soon | later | none. */
  urgency: 'overdue' | 'today' | 'soon' | 'later' | 'none';
  /** Entregas por calificar de la práctica. */
  pending_review: number;
  due_at: string | null;
}

export interface TeacherOverviewQueue {
  pending: number;
  practices: number;
  oldest_submitted_at: string | null;
}

export interface TeacherOverviewSubject {
  id: string;
  code: string;
  name: string;
  enrolled: number;
  pending_review: number;
  /** `null` cuando la materia no tiene una práctica por vencer. */
  without_submission: number | null;
  next_due_at: string | null;
  /** Práctica que vence: la tarjeta la nombra junto a la fecha. */
  next_due_title?: string;
}

export interface TeacherOverviewSignal {
  type: string;
  message: string;
}

export interface TeacherOverview {
  period: TeacherOverviewPeriod | null;
  session: TeacherOverviewSession | null;
  queue: TeacherOverviewQueue;
  /** Ordenado por fecha y limitado por el servidor: la vista no elige ni recorta. */
  due_soon: TeacherOverviewDue[];
  due_soon_total: number;
  /** Cuántos vencimientos quedaron fuera del límite, ya restado por el servidor. */
  due_soon_hidden: number;
  subjects: TeacherOverviewSubject[];
  signals: TeacherOverviewSignal[];
}

export interface TeacherLabStats {
  id: string;
  title: string;
  status: 'draft' | 'published';
  type?: 'algorithm' | 'database' | 'project' | string;
  environment_type?: 'JUEZ_EFIMERO' | 'IDE_PERSISTENTE' | string;
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

/**
 * Read model del banco de ejercicios del docente. La modalidad se deriva de
 * `environment_type` (JUEZ_EFIMERO | IDE_PERSISTENTE); `review_pending` usa la
 * semántica de cada entorno (veredictos por revisar en juez, entregas PENDING
 * de SpeedGrader en IDE).
 */
export interface TeacherExerciseBankItem {
  id: string;
  title: string;
  status: 'draft' | 'published' | 'closed';
  type: 'algorithm' | 'database' | 'project' | string;
  environment_type: 'JUEZ_EFIMERO' | 'IDE_PERSISTENTE' | string;
  language: string;
  memory_limit_mb: number;
  time_limit_ms: number;
  due_date?: string;
  subject_id: string;
  subject_code: string;
  subject_name: string;
  submissions_count: number;
  students_count: number;
  review_pending: number;
}

export interface SubmissionComment {
  id: string;
  tenant_id?: string;
  submission_id: string;
  author_id: string;
  author_name: string;
  line_number: number;
  /** Cierre opcional del bloque comentado; la línea inicial sigue siendo global. */
  end_line_number?: number;
  /** Ancla por archivo en modalidad IDE; nula en juez. */
  file_path?: string;
  local_line_number?: number;
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
  review_status?: ReviewStatus;
}

export type ReviewStatus = 'not_required' | 'pending' | 'in_review' | 'graded' | 'returned';

export type ReviewMode = 'judge' | 'project';

/** El modo de revisión sale del entorno del ejercicio, nunca de la rúbrica ni de los casos. */
export function reviewModeFromEnvironment(environmentType: string | null | undefined): ReviewMode {
  return environmentType === 'IDE_PERSISTENTE' ? 'project' : 'judge';
}

/** Entrega que espera corrección, agrupada por práctica. */
export interface ReviewQueueGroup {
  exercise_id: string;
  exercise_title: string;
  subject_id: string;
  subject_name: string;
  environment_type: string;
  pending: number;
  total: number;
  oldest_submitted_at: string;
}

/** Nota declarada de un alumno en una práctica. */
export interface ExerciseGradeCell {
  selected_submission_id?: string;
  score: number | null;
  source: 'auto' | 'manual';
  reason?: string;
}

export interface SelectExerciseGradeRequest {
  selected_submission_id: string;
  score?: number;
  reason?: string;
}

export interface TestCaseReview {
  input: string;
  expected_output: string;
  is_hidden: boolean;
  actual_output?: string;
  passed: boolean;
}

export interface ProjectFile {
  path: string;
  language: string;
  content: string;
  start_line: number;
  line_count: number;
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
  environment_type: string;
  language: string;
  /** En IDE se omite si el snapshot llegó separado en `files`. */
  boilerplate?: string;
  code?: string;
  verdict: string;
  score?: number;
  manual_override: boolean;
  override_reason?: string;
  graded_by?: string;
  graded_by_name?: string;
  execution_time_ms: number;
  memory_used_mb: number;
  ast_result?: unknown;
  generated_cases?: TestCaseReview[] | unknown;
  expected_complexity?: string;
  complexity_analysis?: ComplexityAnalysis;
  test_cases: TestCaseReview[];
  files_available: boolean;
  metrics_available: boolean;
  /** En juez el backend envía null. */
  files: ProjectFile[] | null;
  /** Proyecto base del ejercicio (IDE) para el diff por archivo. */
  base_files: ProjectFile[] | null;
  base_files_available: boolean;
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
  end_line_number?: number;
  file_path?: string;
  local_line_number?: number;
  local_end_line_number?: number;
  comment: string;
}

export interface EphemeralRunRequestDTO {
  code: string;
  language: string;
}

export interface ReviewEnvironmentAccess {
  url: string;
  expires_at: string;
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

export interface ASTCustomRuleDTO {
  language: string;
  pattern: string;
  type?: 'method' | 'function' | 'module' | 'syntax' | string;
  message: string;
}

export interface ASTRulesDTO {
  block_native_sort?: boolean;
  block_system_modules?: boolean;
  forbidden_imports?: string[];
  forbidden_functions?: string[];
  custom_rules?: ASTCustomRuleDTO[];
}

export interface ChecklistReportDTO {
  blockers: string[];
  warnings: string[];
  info: string[];
  can_publish: boolean;
}

export interface DifficultyMetricDTO {
  count: number;
  success_rate: number;
}

export interface TagMetricDTO {
  tag: string;
  count: number;
  success_rate: number;
}

export interface FailedCaseMetricDTO {
  exercise_title: string;
  case_index: number;
  fail_count: number;
}

export interface TimelineMetricDTO {
  date: string;
  count: number;
}

export interface CourseAnalyticsDTO {
  difficulty_distribution: Record<string, DifficultyMetricDTO>;
  top_tags: TagMetricDTO[];
  most_failed_cases: FailedCaseMetricDTO[];
  avg_resolution_time_by_difficulty: Record<string, number>;
  submissions_timeline: TimelineMetricDTO[];
}

export interface ComparatorDTO {
  id: 'exact' | 'float' | 'unordered' | 'custom';
  params?: Record<string, unknown>;
}

export interface TestCaseDTO {
  id?: string;
  order_index?: number;
  input: string;
  expected_output: string;
  visibility?: 'example' | 'public' | 'hidden';
  weight?: number;
  is_hidden?: boolean;
  is_sample?: boolean;
  description?: string;
}

export interface CreateExerciseRequestDTO {
  subject_id: string;
  title: string;
  description: string;
  type?: 'algorithm' | 'database' | 'workspace' | string;
  difficulty?: 'easy' | 'medium' | 'hard';
  tags?: string[];
  purpose?: 'class' | 'exam';
  per_student_seed?: boolean;
  language?: string;
  boilerplate?: string;
  template_id?: string;
  time_limit_ms?: number;
  memory_limit_mb?: number;
  due_date?: string;
  reference_solution?: string;
  ast_rules?: ASTRulesDTO;
  comparator?: ComparatorDTO;
  input_format?: ExerciseContract;
  expected_complexity?: string;
  test_cases?: TestCaseDTO[];
}

export interface UpdateExerciseRequestDTO {
  title?: string;
  description?: string;
  type?: 'algorithm' | 'database' | 'workspace' | string;
  difficulty?: 'easy' | 'medium' | 'hard';
  tags?: string[];
  purpose?: 'class' | 'exam';
  per_student_seed?: boolean;
  language?: string;
  boilerplate?: string;
  template_id?: string;
  time_limit_ms?: number;
  memory_limit_mb?: number;
  due_date?: string;
  reference_solution?: string;
  ast_rules?: ASTRulesDTO;
  comparator?: ComparatorDTO;
  input_format?: ExerciseContract;
  expected_complexity?: string;
  test_cases?: TestCaseDTO[];
}

export interface ExercisePayloadDTO extends CreateExerciseRequestDTO {
  environment_type?: string;
  db_init_script?: string;
  app_run_command?: string;
  app_port?: number;
  allow_resubmit?: boolean;
  auto_checkpoint?: boolean;
}

export interface BulkTestCasesRequestDTO {
  test_cases: TestCaseDTO[];
}

export interface PlagiarismRangeSource {
  file_path?: string;
  start_line: number;
  end_line: number;
}

export interface PlagiarismRange {
  source_a: PlagiarismRangeSource;
  source_b: PlagiarismRangeSource;
  token_count: number;
  exact_match: boolean;
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
  ranges: PlagiarismRange[];
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

export interface SubmissionKeystrokeEvent {
  id?: string;
  submission_id?: string;
  timestamp_ms: number;
  event_type: 'insert' | 'delete' | 'paste';
  position: number;
  content: string;
  paste_source_detected?: boolean;
  created_at?: string;
}

export interface SubmissionKeystrokeReport {
  submission_id: string;
  events: SubmissionKeystrokeEvent[];
  total_time_ms: number;
  paste_count: number;
  paste_percentage: number;
  total_chars_typed: number;
  total_chars_pasted: number;
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
  metrics_available?: boolean;
  metrics_updated_at?: string;
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

export interface ComplexityMeasurement {
  input_size: number;
  time_ms: number;
  memory_kb: number;
}

export interface ComplexityAnalysis {
  time_complexity: string;
  space_complexity: string;
  measurements: ComplexityMeasurement[];
  confidence: number;
}

export type ComplexityClass = 'O(1)' | 'O(log N)' | 'O(N)' | 'O(N log N)' | 'O(N^2)' | 'O(2^N)' | 'O(N!)' | 'Unknown';

export interface BenchmarkSample {
  input_size: number;
  execution_time_ms: number;
  memory_used_kb: number;
  operations_count?: number;
  status: 'pass' | 'tle' | 'mle' | 'error';
}

export interface BenchmarkReport {
  submission_id: string;
  exercise_id?: string;
  student_name?: string;
  expected_time_complexity?: ComplexityClass;
  detected_time_complexity: ComplexityClass;
  expected_space_complexity?: ComplexityClass;
  detected_space_complexity: ComplexityClass;
  r_squared: number;
  samples: BenchmarkSample[];
  is_optimal: boolean;
  summary: string;
  analyzed_at: string;
}

export interface RunBenchmarkRequestDTO {
  submission_id: string;
  input_sizes?: number[];
  preset?: 'small' | 'standard' | 'stress';
}

export interface CourseExerciseHeader {
  id: string;
  title: string;
}

export interface StudentGradesRow {
  student_id: string;
  student_name: string;
  student_email: string;
  /** `null` significa sin corregir; nunca se representa como 0. */
  grades: Record<string, number | null>;
  average: number;
  graded_count: number;
  pending_count: number;
}

export interface CourseGradesMatrix {
  subject_id: string;
  subject_name: string;
  subject_code: string;
  exercises: CourseExerciseHeader[];
  students: StudentGradesRow[];
}

export interface TeacherCourseModule {
  id: string;
  subject_id: string;
  title: string;
  description: string;
  order_index: number;
  pass_score: number;
  created_at: string;
  prerequisites?: string[];
  exercise_ids?: string[];
}

export interface CreateModuleDTO {
  title: string;
  description?: string;
  order_index?: number;
  pass_score?: number;
}

export interface UpdateModuleDTO {
  title: string;
  description?: string;
  order_index?: number;
  pass_score?: number;
}

export interface SetPrerequisitesDTO {
  prerequisite_module_ids: string[];
}

export interface ExerciseImportValidationItem {
  index: number;
  title: string;
  valid: boolean;
  errors?: string[];
  warnings?: string[];
}

export interface ExerciseImportResponse {
  mode: 'dry_run' | 'import';
  exercises_count?: number;
  exercises?: ExerciseImportValidationItem[];
  can_import: boolean;
  total_valid?: number;
  total_invalid?: number;
  imported_count?: number;
  imported_ids?: string[];
}

export interface ScriptCaseValidationItem {
  index: number;
  input: string;
  expected_output: string;
  valid: boolean;
  error?: string;
  description?: string;
}

export interface ScriptGenerationResponse {
  mode: 'dry_run' | 'import';
  execution_time_ms?: number;
  cases_count?: number;
  cases?: ScriptCaseValidationItem[];
  can_import: boolean;
  total_valid?: number;
  total_invalid?: number;
  imported_count?: number;
  imported_case_ids?: string[];
}

export interface ExerciseTemplateSummary {
  id: string;
  name: string;
  docker_image?: string;
  entrypoint?: string;
  base_ram_mb?: number;
  target_environment?: string;
  environment_type?: string;
  status?: string;
  slug?: string;
  language?: string;
  services_config?: Record<string, unknown>;
}

export interface ExerciseAlgorithmConfigDTO {
  test_cases?: TestCaseDTO[];
  ast_rules?: ASTRulesDTO;
  comparator?: ComparatorDTO;
  time_limit_ms?: number;
  memory_limit_mb?: number;
  input_format?: ExerciseContract;
}

export interface ExerciseDatabaseConfigDTO {
  engine?: string;
  init_script?: string;
  reference_solution?: string;
  validation_query?: string;
  expected_json?: string;
  time_limit_ms?: number;
  memory_limit_mb?: number;
}

export interface ExerciseIDEConfigDTO {
  allow_resubmit?: boolean;
  auto_checkpoint?: boolean;
  db_init_script?: string;
  app_run_command?: string;
  app_port?: number;
}

export interface ExerciseConfigDTO {
  algorithm?: ExerciseAlgorithmConfigDTO;
  database?: ExerciseDatabaseConfigDTO;
  ide?: ExerciseIDEConfigDTO;
  input_format?: ExerciseContract;
}

export interface ExerciseDetailDTO {
  id: string;
  subject_id?: string;
  title: string;
  description: string;
  type?: string;
  environment_type?: string;
  template_id?: string | null;
  difficulty?: 'easy' | 'medium' | 'hard' | string | null;
  tags?: string[];
  purpose?: string;
  per_student_seed?: boolean;
  due_date?: string | null;
  boilerplate?: string;
  status?: string;
  language?: string;
  time_limit_ms?: number;
  memory_limit_mb?: number;
  reference_solution?: string;
  stale?: boolean;
  last_valid_dry_run_at?: string | null;
  module_id?: string | null;
  expected_complexity?: string | null;
  config?: ExerciseConfigDTO;
  tenant_id?: string;
  template?: ExerciseTemplateSummary | null;
  test_cases?: TestCaseDTO[];
  created_at?: string;
}

export interface DryRunJobDTO {
  id: string;
  exercise_id: string;
  status: string;
  progress_current: number;
  progress_total: number;
  result?: Record<string, unknown> | null;
  error?: string;
  created_at?: string;
  updated_at?: string;
}

/**
 * Aula de una clase. Es una envoltura opcional de la práctica: sin sesión el
 * alumno trabaja igual, y el aula solo agrega el conjunto contra el que se
 * puede contar quién falta.
 */
export type ClassSessionStatus = 'open' | 'closed';

/**
 * Estado de un alumno en el aula. Los tres primeros son excepciones con dueño
 * docente; el orden del servidor ya los pone arriba.
 */
export type ClassSessionStudentState =
  | 'sin_empezar'
  | 'trabajando'
  | 'trabado'
  | 'entregado'
  | 'ausente';

export interface ClassSession {
  id: string;
  subject_id: string;
  subject_code: string;
  subject_name: string;
  exercise_id: string | null;
  exercise_title?: string;
  /** Propósito de la práctica de la sesión: `exam` congela los volúmenes al cerrar. */
  purpose?: 'class' | 'exam';
  status: ClassSessionStatus;
  opens_at: string;
  /** Fin previsto: al vencer, el aula se cierra sola. */
  ends_at: string;
  closes_at: string | null;
  /** Congelación manual del docente: con hora, no se emiten avances nuevos. */
  frozen_at?: string | null;
  /** Fin de la ventana de rezago; se fija al cerrar el aula. */
  grace_until?: string | null;
  enrolled: number;
  present: number;
}

/** A quién va dirigido un avance: toda la clase o los alumnos elegidos. */
export type AdvanceScope = 'class' | 'selected';

export interface EmitAdvanceRequest {
  exercise_id: string;
  scope: AdvanceScope;
  /** Solo con alcance `selected`; el servidor exige que estén en el aula. */
  student_ids?: string[];
}

export interface ClassAdvance {
  id: string;
  session_id: string;
  exercise_id: string;
  scope: AdvanceScope;
  snapshot_key: string;
  snapshot_sha256: string;
  snapshot_size_bytes: number;
  file_count: number;
  created_at: string;
}

/** Respuesta de la emisión: el avance y los alumnos que lo recibieron. */
export interface ClassAdvanceEmission {
  advance: ClassAdvance;
  recipients: string[];
}

/** Estado de un alumno frente a un avance emitido (ADR-058 §9). */
export type AdvanceRecipientStatus = 'pending' | 'applied' | 'dismissed' | 'failed' | 'expired';

export interface AdvanceRecipient {
  student_id: string;
  name: string;
  email: string;
  status: AdvanceRecipientStatus;
  decided_at: string | null;
}

export interface AdvanceRecipientCounts {
  total: number;
  applied: number;
  dismissed: number;
  pending: number;
  failed: number;
  expired: number;
}

/** Una emisión del aula con el estado de cada destinatario. */
export interface SessionAdvance {
  id: string;
  exercise_id: string;
  exercise_title: string;
  scope: AdvanceScope;
  file_count: number;
  created_at: string;
  counts: AdvanceRecipientCounts;
  recipients: AdvanceRecipient[];
}

/** Acciones que deja el aula en el historial (ADR-058 §12). */
export type ClassHistoryAction =
  | 'CLASS_ADVANCE_EMITTED'
  | 'CLASS_FROZEN'
  | 'CLASS_UNFROZEN'
  | 'CLASS_ADVANCE_GRACE_EXTENDED'
  | 'CLASS_ADVANCE_GRACE_DEADLINE_SET'
  | 'CLASS_ADVANCE_GRACE_CLOSED'
  | 'CLASS_ADVANCE_APPLIED'
  | 'CLASS_ADVANCE_DISMISSED';

/** Un movimiento del aula: quién lo hizo, cuándo y qué cambió. */
export interface ClassHistoryEvent {
  id: string;
  action: ClassHistoryAction;
  actor_id: string;
  actor_name: string;
  actor_email: string;
  actor_role: string;
  created_at: string;
  advance_id?: string;
  exercise_id?: string;
  scope?: AdvanceScope;
  recipients?: number;
  file_count?: number;
  files_written?: number;
  minutes?: number;
  previous_until?: string;
  until?: string;
}

export interface ClassSessionSummary {
  enrolled: number;
  present: number;
  not_started: number;
  working: number;
  submitted: number;
  stuck: number;
  absent: number;
  without_submission: number;
}

export interface ClassSessionStudent {
  student_id: string;
  name: string;
  email: string;
  state: ClassSessionStudentState;
  present: boolean;
  joined_at: string | null;
  last_seen_at: string | null;
  workspace_id: string | null;
  workspace_status: string | null;
  submitted_at: string | null;
  verdict: string | null;
  review_status: string | null;
  attempts: number;
}

export interface ClassSessionBoard {
  session: ClassSession;
  summary: ClassSessionSummary;
  students: ClassSessionStudent[];
}

/** Resultado del congelamiento: cuántos entornos quedaron en solo lectura. */
export interface FreezeReport {
  frozen: number;
  skipped: number;
  failed: number;
}





