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
  recent_submissions: any[];
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

@Injectable({
  providedIn: 'root'
})
export class StudentService {
  private http = inject(HttpClient);

  dashboardData = signal<StudentDashboardData | null>(null);
  dueAssignments = signal<DueAssignment[]>([]);
  loading = signal<boolean>(false);
  error = signal<string | null>(null);

  async loadDashboard(): Promise<void> {
    this.loading.set(true);
    this.error.set(null);

    try {
      const [dash, due] = await Promise.all([
        firstValueFrom(this.http.get<StudentDashboardData>('/api/v1/student/dashboard', { withCredentials: true })).catch(() => null),
        firstValueFrom(this.http.get<any>('/api/v1/student/assignments/due', { withCredentials: true })).catch(() => null)
      ]);

      if (dash) {
        this.dashboardData.set(dash);
      } else {
        // Fallback enriquecido con datos demostrativos según DASHBOARD.md
        this.dashboardData.set({
          student_id: 'current-user',
          tenant_id: 'uab',
          subjects: [
            {
              subject: { id: 'subj-1', name: 'Programación II', code: 'SIS-211' },
              active_workspace: {
                id: 'ws-101',
                tenant_id: 'uab',
                student_id: 'current-user',
                subject_id: 'subj-1',
                status: 'running',
                type: 'IDE_PERSISTENTE',
                access_url: 'http://localhost:3000',
                memory_limit_mb: 256,
                created_at: new Date().toISOString()
              }
            },
            {
              subject: { id: 'subj-2', name: 'Sistemas Operativos', code: 'SIS-312' },
              active_workspace: {
                id: 'ws-102',
                tenant_id: 'uab',
                student_id: 'current-user',
                subject_id: 'subj-2',
                status: 'hibernated',
                type: 'IDE_PERSISTENTE',
                access_url: '',
                memory_limit_mb: 256,
                created_at: new Date().toISOString()
              }
            },
            {
              subject: { id: 'subj-3', name: 'Bases de Datos', code: 'SIS-223' },
              active_workspace: undefined
            }
          ],
          recent_submissions: []
        });
      }

      if (due && due.data) {
        this.dueAssignments.set(due.data);
      } else if (Array.isArray(due)) {
        this.dueAssignments.set(due);
      } else {
        // Fallback de asignaciones urgentes según DASHBOARD.md
        this.dueAssignments.set([
          {
            exercise_id: 'ex-01',
            title: 'Lab #04: Estructuras Dinámicas y Árboles',
            subject_id: 'subj-1',
            subject_name: 'Programación II',
            subject_code: 'SIS-211',
            due_date: new Date(Date.now() + 86400000).toISOString(),
            type: 'algorithm'
          },
          {
            exercise_id: 'ex-02',
            title: 'Práctica #02: Sincronización y Mutex',
            subject_id: 'subj-2',
            subject_name: 'Sistemas Operativos',
            subject_code: 'SIS-312',
            due_date: new Date(Date.now() + 172800000).toISOString(),
            type: 'project'
          }
        ]);
      }
    } catch (err: any) {
      this.error.set('No se pudo sincronizar el estado académico');
    } finally {
      this.loading.set(false);
    }
  }

  async startWorkspace(subjectId: string): Promise<WorkspaceInstance> {
    const resp = await firstValueFrom(
      this.http.post<WorkspaceInstance>('/api/v1/workspaces/start', { subject_id: subjectId })
    );
    await this.loadDashboard();
    return resp;
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
}

