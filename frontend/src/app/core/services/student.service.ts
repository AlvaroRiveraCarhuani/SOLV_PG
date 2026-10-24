import { Injectable, inject, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { firstValueFrom } from 'rxjs';
import { WorkspaceInstance } from '@core/models/workspace.model';

export interface StudentSubjectItem {
  subject: {
    id: string;
    name: string;
    code: string;
  };
  active_workspace?: WorkspaceInstance;
  total_exercises?: number;
  completed_exercises?: number;
  progress_percent?: number;
}

export interface StudentRecentSubmission {
  id: string;
  exercise_id: string;
  exercise_title?: string;
  subject_code?: string;
  verdict: string;
  score?: number | null;
  execution_time_ms: number;
  memory_used_mb: number;
  submitted_at: string;
}

export interface DueAssignment {
  exercise_id: string;
  title: string;
  subject_id: string;
  subject_name: string;
  subject_code: string;
  due_date: string;
  type: string;
}

export interface StudentDashboardData {
  student_id: string;
  tenant_id: string;
  subjects: StudentSubjectItem[];
  recent_submissions: StudentRecentSubmission[];
}

export type StudentAcademicStatus = 'pending' | 'in_progress' | 'completed' | 'overdue';

export interface StudentEvaluationItem {
  exercise_id: string;
  title: string;
  description: string;
  type: string;
  environment_type: string;
  purpose: 'class' | 'exam' | string;
  difficulty?: string;
  due_date?: string | null;
  subject_id: string;
  subject_name: string;
  subject_code: string;
  attempts_count: number;
  best_score?: number | null;
  last_verdict?: string;
  last_submitted_at?: string | null;
  academic_status: StudentAcademicStatus;
}

export interface StudentSubmissionHistoryItem {
  id: string;
  exercise_id: string;
  exercise_title?: string;
  exercise_type?: string;
  exercise_purpose?: string;
  subject_id?: string;
  subject_code?: string;
  subject_name?: string;
  code: string;
  verdict: string;
  score?: number | null;
  execution_time_ms: number;
  memory_used_mb: number;
  manual_override?: boolean;
  override_reason?: string;
  submitted_at: string;
}

export interface WeakTag {
  tag: string;
  success_rate: number;
  attempts: number;
}

export interface RecommendationItem {
  exercise_id: string;
  title: string;
  difficulty: string;
  matched_tag: string;
  reason: string;
}

export interface StudentRecommendationsDTO {
  has_enough_data: boolean;
  weak_tags: WeakTag[];
  recommendations: RecommendationItem[];
  message?: string;
}

export interface CurricularExercise {
  id: string;
  title: string;
  difficulty?: string;
  purpose: string;
  best_score?: number;
  attempts: number;
  submittable: boolean;
}

export interface CurricularModuleMap {
  id: string;
  title: string;
  description: string;
  order_index: number;
  pass_score: number;
  state: 'locked' | 'unlocked' | 'in_progress' | 'completed';
  lock_reason?: string;
  prerequisite_module_ids: string[];
  exercises: CurricularExercise[];
}

export interface CourseCurricularMap {
  course_id: string;
  modules: CurricularModuleMap[];
  unassigned_exercises: CurricularExercise[];
}

export interface StudentPublicTestCaseDTO {
  input: string;
  expected_output: string;
}

export interface StudentPublicExerciseDTO {
  id: string;
  subject_id?: string;
  title: string;
  description: string;
  type: string;
  environment_type?: string;
  difficulty?: string;
  tags?: string[];
  purpose?: string;
  per_student_seed?: boolean;
  language?: string;
  boilerplate?: string;
  time_limit_ms?: number;
  memory_mb?: number;
  memory_limit_mb?: number;
  constraints?: string[];
  public_test_cases?: StudentPublicTestCaseDTO[];
  allow_resubmit?: boolean;
  auto_checkpoint?: boolean;
}

export interface WorkspaceProjectSubmissionDTO {
  submission_id: string;
  workspace_id: string;
  exercise_id: string;
  verdict: string;
  score?: number | null;
}

export interface StudentCaseResultDTO {
  index: number;
  verdict: 'AC' | 'WA' | 'TLE' | 'RE' | 'AST_BLOCKED' | 'AST_VIOLATION' | 'SYSTEM_ERROR';
  duration_ms: number;
  message?: string;
}

export interface StudentEvaluationResultDTO {
  verdict: 'AC' | 'WA' | 'TLE' | 'RE' | 'AST_BLOCKED' | 'AST_VIOLATION' | 'SYSTEM_ERROR';
  execution_time_ms: number;
  memory_used_mb: number;
  message: string;
  failed_test_case?: {
    input: string;
    expected_output: string;
  };
  case_results?: StudentCaseResultDTO[];
  generated_cases?: unknown;
}

export interface CreateStudentSubmissionPayload {
  exercise_id: string;
  student_id: string;
  code: string;
  verdict: string;
  execution_time_ms: number;
  memory_used_mb: number;
  generated_cases?: unknown;
}

@Injectable({
  providedIn: 'root'
})
export class StudentService {
  private http = inject(HttpClient);

  dashboardData = signal<StudentDashboardData | null>(null);
  dueAssignments = signal<DueAssignment[]>([]);
  evaluations = signal<StudentEvaluationItem[]>([]);
  submissionsHistory = signal<StudentSubmissionHistoryItem[]>([]);
  loading = signal<boolean>(false);
  error = signal<string | null>(null);

  async loadDashboard(): Promise<void> {
    this.loading.set(true);
    this.error.set(null);

    try {
      const [dash, due] = await Promise.all([
        firstValueFrom(this.http.get<StudentDashboardData>('/api/v1/student/dashboard', { withCredentials: true })).catch(() => null),
        firstValueFrom(this.http.get<DueAssignment[] | { data?: DueAssignment[] }>('/api/v1/student/assignments/due', { withCredentials: true })).catch(() => null)
      ]);

      if (dash) {
        this.dashboardData.set(dash);
      } else {
        this.dashboardData.set({
          student_id: '',
          tenant_id: '',
          subjects: [],
          recent_submissions: []
        });
      }

      this.dueAssignments.set(Array.isArray(due) ? due : (due?.data ?? []));
    } catch {
      this.error.set('No se pudo sincronizar el estado académico');
    } finally {
      this.loading.set(false);
    }
  }

  async loadEvaluations(): Promise<StudentEvaluationItem[]> {
    try {
      const resp = await firstValueFrom(
        this.http.get<StudentEvaluationItem[] | { data?: StudentEvaluationItem[] }>(
          '/api/v1/student/evaluations',
          { withCredentials: true }
        )
      );
      const list = Array.isArray(resp) ? resp : (resp?.data ?? []);
      this.evaluations.set(list);
      return list;
    } catch {
      this.evaluations.set([]);
      return [];
    }
  }

  async loadSubmissionsHistory(): Promise<StudentSubmissionHistoryItem[]> {
    try {
      const resp = await firstValueFrom(
        this.http.get<StudentSubmissionHistoryItem[] | { data?: StudentSubmissionHistoryItem[] }>(
          '/api/v1/student/submissions',
          { withCredentials: true }
        )
      );
      const list = Array.isArray(resp) ? resp : (resp?.data ?? []);
      this.submissionsHistory.set(list);
      return list;
    } catch {
      this.submissionsHistory.set([]);
      return [];
    }
  }

  async startWorkspace(subjectId: string, exerciseId?: string): Promise<WorkspaceInstance> {
    const payload = {
      subject_id: subjectId,
      ...(exerciseId ? { exercise_id: exerciseId } : {})
    };
    const resp = await firstValueFrom(
      this.http.post<WorkspaceInstance | { data?: WorkspaceInstance }>(
        '/api/v1/workspaces/start',
        payload,
        { withCredentials: true }
      )
    );
    const raw = resp && 'data' in resp && resp.data ? resp.data : (resp as WorkspaceInstance);
    const normalized: WorkspaceInstance = {
      ...raw,
      id: raw.id || raw.workspace_id || '',
      last_activity_at: raw.last_activity_at || raw.last_heartbeat_at
    };
    await this.loadDashboard();
    return normalized;
  }

  async submitWorkspaceProject(
    workspaceId: string,
    payload: { exercise_id: string; files?: Record<string, string> }
  ): Promise<WorkspaceProjectSubmissionDTO> {
    const resp = await firstValueFrom(
      this.http.post<WorkspaceProjectSubmissionDTO | { data?: WorkspaceProjectSubmissionDTO }>(
        `/api/v1/workspaces/${workspaceId}/submit`,
        payload,
        { withCredentials: true }
      )
    );
    return resp && 'data' in resp && resp.data ? resp.data : (resp as WorkspaceProjectSubmissionDTO);
  }

  async pauseWorkspace(workspaceId: string): Promise<void> {
    await firstValueFrom(
      this.http.post(`/api/v1/workspaces/${workspaceId}/pause`, {})
    );
    await this.loadDashboard();
  }

  async getCourseRecommendations(courseId: string): Promise<StudentRecommendationsDTO> {
    try {
      const resp = await firstValueFrom(
        this.http.get<{ success?: boolean; data?: StudentRecommendationsDTO } | StudentRecommendationsDTO>(
          `/api/v1/student/courses/${courseId}/recommendations`,
          { withCredentials: true }
        )
      );
      if (resp && 'data' in resp && resp.data) {
        return resp.data;
      }
      if (resp && 'has_enough_data' in resp) {
        return resp as StudentRecommendationsDTO;
      }
      return {
        has_enough_data: false,
        weak_tags: [],
        recommendations: [],
        message: 'No hay datos disponibles'
      };
    } catch {
      return {
        has_enough_data: false,
        weak_tags: [],
        recommendations: [],
        message: 'No se pudieron cargar las recomendaciones'
      };
    }
  }

  async getCourseCurricularMap(courseId: string): Promise<CourseCurricularMap | null> {
    try {
      const resp = await firstValueFrom(
        this.http.get<{ data: CourseCurricularMap; message?: string } | CourseCurricularMap>(
          `/api/v1/student/courses/${courseId}/map`,
          { withCredentials: true }
        )
      );
      if (resp && 'data' in resp && resp.data) {
        return resp.data;
      }
      if (resp && 'modules' in resp) {
        return resp as CourseCurricularMap;
      }
      return null;
    } catch {
      return null;
    }
  }

  async saveKeystrokeEvents(submissionId: string, events: unknown[]): Promise<void> {
    if (!submissionId || !events || events.length === 0) return;
    try {
      await firstValueFrom(
        this.http.post(`/api/v1/submissions/${submissionId}/keystroke-events`, { events }, { withCredentials: true })
      );
    } catch {
      // Ignored for non-blocking telemetry
    }
  }

  async getPublicExercise(exerciseId: string): Promise<StudentPublicExerciseDTO | null> {
    if (!exerciseId) return null;
    try {
      const resp = await firstValueFrom(
        this.http.get<{ data?: StudentPublicExerciseDTO } | StudentPublicExerciseDTO>(
          `/api/v1/exercises/${exerciseId}`,
          { withCredentials: true }
        )
      );
      if (resp && 'data' in resp && resp.data) {
        return resp.data;
      }
      if (resp && 'id' in resp) {
        return resp as StudentPublicExerciseDTO;
      }
      return null;
    } catch {
      return null;
    }
  }

  async evaluateExercise(payload: {
    exercise_id: string;
    language: string;
    source_code_b64: string;
  }): Promise<StudentEvaluationResultDTO> {
    const resp = await firstValueFrom(
      this.http.post<{ data?: StudentEvaluationResultDTO } | StudentEvaluationResultDTO>(
        '/api/v1/evaluations',
        payload,
        { withCredentials: true }
      )
    );
    if (resp && 'data' in resp && resp.data) {
      return resp.data;
    }
    return resp as StudentEvaluationResultDTO;
  }

  async submitExercise(payload: CreateStudentSubmissionPayload): Promise<{ id: string }> {
    const resp = await firstValueFrom(
      this.http.post<{ id?: string; data?: { id?: string } }>(
        '/api/v1/submissions',
        payload,
        { withCredentials: true }
      )
    );
    const id = resp?.id || resp?.data?.id || '';
    return { id };
  }
}

