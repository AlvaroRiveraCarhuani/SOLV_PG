import { Injectable, inject, signal } from '@angular/core';
import { HttpClient, HttpParams } from '@angular/common/http';
import { TeacherItem, TeacherInvitationPayload } from '@core/models/admin.model';
import { Observable, catchError, map, of, tap } from 'rxjs';

export interface InvitationResponse {
  id: string;
  email: string;
  token: string;
  invite_url: string;
  message: string;
  expires_at: string;
}

export interface TeacherCourse {
  id: string;
  name: string;
  code: string;
  students_count: number;
}

@Injectable({
  providedIn: 'root'
})
export class AdminTeachersService {
  private http = inject(HttpClient);

  teachers = signal<TeacherItem[]>([]);
  isLoading = signal<boolean>(false);
  error = signal<string | null>(null);
  total = signal<number>(0);

  lastGeneratedInvitation = signal<InvitationResponse | null>(null);

  fetchTeachers(search?: string, status?: string, origin?: string): void {
    this.isLoading.set(true);
    this.error.set(null);

    let params = new HttpParams();
    if (search && search.trim()) {
      params = params.set('search', search.trim());
    }
    if (status && status !== 'all') {
      params = params.set('status', status);
    }
    if (origin && origin !== 'all') {
      params = params.set('origin', origin);
    }

    this.http.get<{ data?: TeacherItem[]; teachers?: TeacherItem[]; total?: number }>('/api/v1/teachers', { params })
      .pipe(
        tap(res => {
          const list = res.data || res.teachers || [];
          this.teachers.set(list);
          this.total.set(res.total ?? list.length);
          this.isLoading.set(false);
        }),
        catchError(() => {
          this.error.set('No se pudo conectar al servidor de docentes');
          this.isLoading.set(false);
          return of(null);
        })
      )
      .subscribe();
  }

  inviteTeacher(payload: TeacherInvitationPayload): Observable<InvitationResponse | null> {
    this.isLoading.set(true);
    return this.http.post<InvitationResponse>('/api/v1/invitations/teachers', payload)
      .pipe(
        tap(res => {
          this.lastGeneratedInvitation.set(res);
          const newItem: TeacherItem = {
            id: res.id,
            full_name: payload.email.split('@')[0].replace('.', ' '),
            email: payload.email,
            origin: 'manual',
            status: 'pending',
            role_type: payload.role_type || 'titular',
            invited_at: 'Recién invitado',
            active_courses: 0
          };
          this.teachers.update(current => [newItem, ...current]);
          this.total.update(t => t + 1);
          this.isLoading.set(false);
        }),
        catchError(() => {
          this.isLoading.set(false);
          return of(null);
        })
      );
  }

  resendInvitation(teacherId: string): Observable<InvitationResponse | null> {
    return this.http.post<InvitationResponse>(`/api/v1/invitations/teachers/${teacherId}/resend`, {})
      .pipe(
        tap(res => {
          this.lastGeneratedInvitation.set(res);
          this.updateTeacherStatus(teacherId, 'pending', 'Hace unos segundos');
        }),
        catchError(() => {
          this.updateTeacherStatus(teacherId, 'pending', 'Hace unos segundos');
          return of(null);
        })
      );
  }

  renewInvitation(teacherId: string): Observable<InvitationResponse | null> {
    return this.http.post<InvitationResponse>(`/api/v1/invitations/teachers/${teacherId}/renew`, {})
      .pipe(
        tap(res => {
          this.lastGeneratedInvitation.set(res);
          this.updateTeacherStatus(teacherId, 'pending', 'Recién renovado (72h)');
        }),
        catchError(() => {
          this.updateTeacherStatus(teacherId, 'pending', 'Recién renovado (72h)');
          return of(null);
        })
      );
  }

  deleteInvitation(invitationId: string): Observable<boolean> {
    return this.http.delete(`/api/v1/invitations/teachers/${invitationId}`)
      .pipe(
        map(() => {
          this.teachers.update(current => current.filter(t => t.id !== invitationId));
          this.total.update(t => Math.max(0, t - 1));
          return true;
        }),
        catchError(() => of(false))
      );
  }

  getTeacherCourses(teacherId: string): Observable<TeacherCourse[]> {
    return this.http.get<{ data?: TeacherCourse[] }>(`/api/v1/teachers/${teacherId}/courses`)
      .pipe(
        map(res => res.data || []),
        catchError(() => of([]))
      );
  }

  reassignCourseTeacher(courseId: string, newTeacherId: string, reason?: string): Observable<any> {
    return this.http.post(`/api/v1/admin/courses/${courseId}/reassign`, {
      new_teacher_id: newTeacherId,
      reason: reason || 'Reasignación de titularidad docente por administración'
    });
  }

  getAllAvailableCourses(): Observable<{ id: string; name: string; teacher_name: string }[]> {
    return this.http.get<{ data?: any[] } | any[]>('/api/v1/admin/dashboard/courses-load')
      .pipe(
        map(res => {
          const list = Array.isArray(res) ? res : (res.data || []);
          return list.map(item => ({
            id: item.id,
            name: item.course_name,
            teacher_name: item.teacher_name || 'Sin asignar'
          }));
        }),
        catchError(() => of([]))
      );
  }

  clearLastInvitation(): void {
    this.lastGeneratedInvitation.set(null);
  }

  private updateTeacherStatus(teacherId: string, status: 'active' | 'pending' | 'expired', invitedAtText: string): void {
    this.teachers.update(current => 
      current.map(t => t.id === teacherId ? { ...t, status, invited_at: invitedAtText } : t)
    );
  }
}
