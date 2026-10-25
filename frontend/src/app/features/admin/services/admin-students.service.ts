import { Injectable, inject } from '@angular/core';
import { HttpClient, HttpParams } from '@angular/common/http';
import { Observable, map } from 'rxjs';

export interface AdminStudentItem {
  id: string;
  first_name: string;
  last_name: string;
  email: string;
  role: string;
  status: string;
  suspension_reason?: string | null;
  academic_status: string;
  enrolled_courses_count: number;
  active_workspaces_count: number;
  oom_strike_count: number;
  last_oom_killed_at?: string | null;
}

export interface AdminStudentCourseItem {
  subject_id: string;
  subject_code: string;
  subject_name: string;
  teacher_name: string;
  enrolled_at: string;
  workspace_id?: string | null;
  workspace_status?: string | null;
  memory_limit_mb?: number | null;
  oom_strike_count?: number | null;
  last_oom_killed_at?: string | null;
}

export interface ResetOOMResult {
  student_id: string;
  workspaces_reset_count: number;
  message: string;
}

export interface SubjectOption {
  id: string;
  code: string;
  name: string;
}

interface AcademicPeriodRaw {
  id: string;
  code: string;
  name: string;
  is_active?: boolean;
}

interface DataEnvelope<T> {
  data?: T;
  error?: string;
  message?: string;
}

export interface AcademicPeriodOption {
  id: string;
  code: string;
  name: string;
  is_active: boolean;
}

export interface CreateStudentDTO {
  first_name: string;
  last_name: string;
  email: string;
}

export interface BatchStudentsResult {
  total_processed: number;
  created_count: number;
}

export interface RosterEnrollResult {
  created_students: number;
  enrolled_students: number;
  duplicate_emails: string[];
  invalid_emails: string[];
  total_detected: number;
}

@Injectable({
  providedIn: 'root'
})
export class AdminStudentsService {
  private http = inject(HttpClient);
  private apiUrl = '/api/v1/admin/students';
  private academicPeriodsUrl = '/api/v1/admin/academic-periods';

  getStudents(search?: string, subjectId?: string, status?: string, periodId?: string): Observable<AdminStudentItem[]> {
    let params = new HttpParams();
    if (search && search.trim()) {
      params = params.set('search', search.trim());
    }
    if (subjectId && subjectId.trim() && subjectId !== 'all') {
      params = params.set('subject_id', subjectId.trim());
    }
    if (status && status.trim() && status !== 'all') {
      params = params.set('status', status.trim());
    }
    if (periodId && periodId.trim()) {
      params = params.set('period_id', periodId.trim());
    }

    return this.http.get<{ data: AdminStudentItem[]; error: string; message: string }>(this.apiUrl, { params }).pipe(
      map(res => res.data || [])
    );
  }

  getAcademicPeriods(): Observable<AcademicPeriodOption[]> {
    return this.http.get<DataEnvelope<AcademicPeriodRaw[]>>(this.academicPeriodsUrl).pipe(
      map(res => {
        const raw = res.data || [];
        return raw.map(item => ({
          id: item.id,
          code: item.code,
          name: item.name,
          is_active: item.is_active ?? false
        }));
      })
    );
  }

  createStudent(dto: CreateStudentDTO): Observable<AdminStudentItem> {
    return this.http.post<{ data: AdminStudentItem; error: string; message: string }>(this.apiUrl, dto).pipe(
      map(res => res.data)
    );
  }

  updateStudentStatus(studentId: string, status: 'active' | 'suspended', reason?: string): Observable<void> {
    return this.http.put<DataEnvelope<unknown>>(
      `${this.apiUrl}/${studentId}/status`,
      { status, reason }
    ).pipe(
      map(() => void 0)
    );
  }

  getStudentCourses(studentId: string): Observable<AdminStudentCourseItem[]> {
    return this.http.get<{ data: AdminStudentCourseItem[]; error: string; message: string }>(
      `${this.apiUrl}/${studentId}/courses`
    ).pipe(
      map(res => res.data || [])
    );
  }

  restartWorkspace(workspaceId: string): Observable<void> {
    return this.http.post<unknown>(`/api/v1/workspaces/${workspaceId}/restart`, {}).pipe(
      map(() => void 0)
    );
  }

  pauseWorkspace(workspaceId: string): Observable<void> {
    return this.http.post<unknown>(`/api/v1/workspaces/${workspaceId}/pause`, {}).pipe(
      map(() => void 0)
    );
  }

  resetStudentOOM(studentId: string, reason: string): Observable<ResetOOMResult> {
    return this.http.post<{ data: ResetOOMResult; error: string; message: string }>(
      `${this.apiUrl}/${studentId}/reset-oom`,
      { reason }
    ).pipe(
      map(res => res.data)
    );
  }

  getSubjects(): Observable<SubjectOption[]> {
    return this.http.get<DataEnvelope<SubjectOption[]>>('/api/v1/subjects').pipe(
      map(res => {
        const raw = res.data || [];
        return raw.map(item => ({
          id: item.id,
          code: item.code,
          name: item.name
        }));
      })
    );
  }

  batchCreateStudents(emails: string[]): Observable<BatchStudentsResult> {
    return this.http.post<BatchStudentsResult>('/api/v1/admin/students/batch', { emails });
  }

  rosterEnroll(subjectId: string, emails: string[]): Observable<RosterEnrollResult> {
    return this.http.post<RosterEnrollResult>(`/api/v1/admin/subjects/${subjectId}/roster`, { emails });
  }
}
