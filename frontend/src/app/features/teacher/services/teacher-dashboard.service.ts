import { Injectable, inject, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, tap, map } from 'rxjs';
import {
  TeacherCourseSummary,
  TeacherAttentionWidget,
  AcademicPeriod
} from '../models/teacher.models';

interface ApiResponse<T> {
  data: T;
  error?: string;
  message?: string;
}

@Injectable({
  providedIn: 'root'
})
export class TeacherDashboardService {
  private http = inject(HttpClient);

  readonly courses = signal<TeacherCourseSummary[]>([]);
  readonly attention = signal<TeacherAttentionWidget | null>(null);
  readonly periods = signal<AcademicPeriod[]>([]);
  readonly activePeriod = signal<AcademicPeriod | null>(null);
  readonly isLoading = signal<boolean>(false);

  getAcademicPeriods(): Observable<AcademicPeriod[]> {
    return this.http.get<ApiResponse<AcademicPeriod[]>>('/api/v1/academic-periods').pipe(
      map(res => res.data || []),
      tap(periods => {
        this.periods.set(periods);
        const active = periods.find(p => p.is_active) || periods[0] || null;
        this.activePeriod.set(active);
      })
    );
  }

  loadDashboardData(periodId?: string): Observable<{ courses: TeacherCourseSummary[]; attention: TeacherAttentionWidget }> {
    this.isLoading.set(true);
    const params = periodId ? `?period_id=${periodId}` : '';
    return this.http.get<ApiResponse<TeacherCourseSummary[]>>(`/api/v1/teacher/courses${params}`).pipe(
      map(res => res.data || []),
      tap(courses => this.courses.set(courses)),
      tap(() => {
        this.http.get<ApiResponse<TeacherAttentionWidget>>('/api/v1/teacher/attention').subscribe({
          next: res => {
            this.attention.set(res.data);
            this.isLoading.set(false);
          },
          error: () => this.isLoading.set(false)
        });
      }),
      map(courses => ({
        courses,
        attention: this.attention() ?? { critical: [], warning: [], standard: [] }
      }))
    );
  }

  getCourses(): Observable<TeacherCourseSummary[]> {
    return this.http.get<ApiResponse<TeacherCourseSummary[]>>('/api/v1/teacher/courses').pipe(
      map(res => res.data || []),
      tap(courses => this.courses.set(courses))
    );
  }

  getAttention(): Observable<TeacherAttentionWidget> {
    return this.http.get<ApiResponse<TeacherAttentionWidget>>('/api/v1/teacher/attention').pipe(
      map(res => res.data),
      tap(widget => this.attention.set(widget))
    );
  }
}
