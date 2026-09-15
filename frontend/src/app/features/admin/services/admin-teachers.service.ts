import { Injectable, inject, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { TeacherItem, TeacherInvitationPayload } from '@core/models/admin.model';
import { catchError, of, tap } from 'rxjs';

@Injectable({
  providedIn: 'root'
})
export class AdminTeachersService {
  private http = inject(HttpClient);

  teachers = signal<TeacherItem[]>([]);
  isLoading = signal<boolean>(false);
  error = signal<string | null>(null);

  constructor() {
    this.fetchTeachers();
  }

  fetchTeachers(): void {
    this.isLoading.set(true);
    this.error.set(null);

    this.http.get<{ data?: TeacherItem[]; teachers?: TeacherItem[] }>('/api/v1/teachers')
      .pipe(
        tap(res => {
          const list = res.data || res.teachers;
          if (list && list.length > 0) {
            this.teachers.set(list);
          } else {
            this.teachers.set(this.buildSeedTeachers());
          }
          this.isLoading.set(false);
        }),
        catchError(() => {
          this.teachers.set(this.buildSeedTeachers());
          this.isLoading.set(false);
          return of(null);
        })
      )
      .subscribe();
  }

  inviteTeacher(payload: TeacherInvitationPayload): void {
    this.isLoading.set(true);
    this.http.post<{ message?: string; id?: string }>('/api/v1/invitations/teachers', payload)
      .pipe(
        tap(() => {
          this.addLocalInvitation(payload);
          this.isLoading.set(false);
        }),
        catchError(() => {
          // Fallback optimista para desarrollo y pruebas de UI
          this.addLocalInvitation(payload);
          this.isLoading.set(false);
          return of(null);
        })
      )
      .subscribe();
  }

  resendInvitation(teacherId: string): void {
    this.http.post(`/api/v1/invitations/teachers/${teacherId}/resend`, {})
      .pipe(
        tap(() => {
          this.updateTeacherStatus(teacherId, 'pending', 'Hace unos segundos');
        }),
        catchError(() => {
          this.updateTeacherStatus(teacherId, 'pending', 'Hace unos segundos');
          return of(null);
        })
      )
      .subscribe();
  }

  renewInvitation(teacherId: string): void {
    this.http.post(`/api/v1/invitations/teachers/${teacherId}/renew`, {})
      .pipe(
        tap(() => {
          this.updateTeacherStatus(teacherId, 'pending', 'Recién renovado (72h)');
        }),
        catchError(() => {
          this.updateTeacherStatus(teacherId, 'pending', 'Recién renovado (72h)');
          return of(null);
        })
      )
      .subscribe();
  }

  private addLocalInvitation(payload: TeacherInvitationPayload): void {
    const newItem: TeacherItem = {
      id: `tch-${Math.random().toString(36).substring(2, 9)}`,
      full_name: payload.email.split('@')[0].replace('.', ' '),
      email: payload.email,
      origin: 'manual',
      status: 'pending',
      role_type: payload.role_type,
      invited_at: 'Recién invitado',
      active_courses: 0
    };
    this.teachers.update(current => [newItem, ...current]);
  }

  private updateTeacherStatus(teacherId: string, status: 'active' | 'pending' | 'expired', invitedAtText: string): void {
    this.teachers.update(current => 
      current.map(t => t.id === teacherId ? { ...t, status, invited_at: invitedAtText } : t)
    );
  }

  private buildSeedTeachers(): TeacherItem[] {
    return [
      {
        id: 'tch-01',
        full_name: 'Margaret Hamilton',
        email: 'mhamilton@uab.edu.bo',
        origin: 'manual',
        status: 'active',
        role_type: 'titular',
        invited_at: '15-Ago-2026',
        last_login: 'Hoy 08:30',
        active_courses: 2
      },
      {
        id: 'tch-02',
        full_name: 'Ada Lovelace',
        email: 'alovelace@uab.edu.bo',
        origin: 'gclassroom',
        status: 'active',
        role_type: 'titular',
        invited_at: '10-Ago-2026',
        last_login: 'Ayer 16:40',
        active_courses: 3
      },
      {
        id: 'tch-03',
        full_name: 'Tim Berners-Lee',
        email: 'tberners@uab.edu.bo',
        origin: 'manual',
        status: 'pending',
        role_type: 'titular',
        invited_at: 'Hace 24h',
        active_courses: 0
      },
      {
        id: 'tch-04',
        full_name: 'Linus Torvalds',
        email: 'ltorvalds@uab.edu.bo',
        origin: 'manual',
        status: 'expired',
        role_type: 'auxiliar',
        invited_at: 'Hace 4d',
        active_courses: 0
      }
    ];
  }
}
