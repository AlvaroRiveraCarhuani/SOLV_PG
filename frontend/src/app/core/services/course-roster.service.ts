import { Injectable, inject } from '@angular/core';
import { HttpClient, HttpParams } from '@angular/common/http';
import { Observable, map } from 'rxjs';

/** Origen de la matrícula tal como lo persiste `enrollments.source_system`. */
export type EnrollmentSource = 'institutional_roster' | 'teacher_manual' | string;

export interface CourseStudent {
  student_id: string;
  full_name: string;
  email: string;
  enrolled_at: string;
  source_system: EnrollmentSource;
  workspace_status: string;
}

export interface CourseAvailableStudent {
  id: string;
  first_name: string;
  last_name: string;
  email: string;
}

export interface CourseRosterResult {
  created_students: number;
  enrolled_students: number;
  removed_students: number;
  skipped_not_found: number;
  matched_by_external_id: number;
  duplicate_emails: string[];
  invalid_emails: string[];
  total_detected: number;
}

interface ApiEnvelope<T> {
  data?: T;
  message?: string;
}

interface RosterListPayload {
  subject_id?: string;
  students?: CourseStudent[];
}

interface AvailableListPayload {
  subject_id?: string;
  students?: CourseAvailableStudent[];
}

/**
 * Padrón de una materia. La misma API sirve al administrador y al docente
 * titular: el backend deriva el permiso y el origen de la matrícula del rol de
 * la sesión, nunca del payload.
 */
@Injectable({ providedIn: 'root' })
export class CourseRosterService {
  private readonly http = inject(HttpClient);

  listStudents(subjectId: string): Observable<CourseStudent[]> {
    return this.http
      .get<ApiEnvelope<RosterListPayload> | RosterListPayload>(`/api/v1/courses/${subjectId}/students`)
      .pipe(
        map((res) => {
          const body = (res as ApiEnvelope<RosterListPayload>).data ?? (res as RosterListPayload);
          return body?.students ?? [];
        })
      );
  }

  searchAvailableStudents(subjectId: string, query: string): Observable<CourseAvailableStudent[]> {
    let params = new HttpParams();
    const trimmed = query.trim();
    if (trimmed) {
      params = params.set('q', trimmed);
    }
    return this.http
      .get<ApiEnvelope<AvailableListPayload> | AvailableListPayload>(
        `/api/v1/courses/${subjectId}/available-students`,
        { params }
      )
      .pipe(
        map((res) => {
          const body = (res as ApiEnvelope<AvailableListPayload>).data ?? (res as AvailableListPayload);
          return body?.students ?? [];
        })
      );
  }

  enrollStudent(subjectId: string, studentId: string): Observable<CourseStudent> {
    return this.http
      .post<ApiEnvelope<CourseStudent> | CourseStudent>(`/api/v1/courses/${subjectId}/students`, {
        student_id: studentId
      })
      .pipe(map((res) => (res as ApiEnvelope<CourseStudent>).data ?? (res as CourseStudent)));
  }

  removeStudent(subjectId: string, studentId: string): Observable<void> {
    return this.http
      .delete<unknown>(`/api/v1/courses/${subjectId}/students/${studentId}`)
      .pipe(map(() => void 0));
  }

  /**
   * Aplica una planilla completa. En la sesión del docente el backend fuerza
   * `teacher_manual` y desactiva la reconciliación: su planilla suma altas de
   * cátedra, no reemplaza la nómina oficial.
   */
  enrollRoster(subjectId: string, emails: string[]): Observable<CourseRosterResult> {
    return this.http
      .post<ApiEnvelope<CourseRosterResult> | CourseRosterResult>(`/api/v1/courses/${subjectId}/roster`, {
        emails
      })
      .pipe(map((res) => (res as ApiEnvelope<CourseRosterResult>).data ?? (res as CourseRosterResult)));
  }
}
