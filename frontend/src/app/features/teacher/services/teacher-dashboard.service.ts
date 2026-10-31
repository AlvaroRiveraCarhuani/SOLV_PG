import { Injectable, inject, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, tap, map, finalize } from 'rxjs';
import {
  TeacherCourseSummary,
  AcademicPeriod,
  TeacherOverview
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
  readonly periods = signal<AcademicPeriod[]>([]);
  readonly activePeriod = signal<AcademicPeriod | null>(null);
  readonly overview = signal<TeacherOverview | null>(null);
  readonly isLoading = signal<boolean>(false);

  getAcademicPeriods(): Observable<AcademicPeriod[]> {
    return this.http.get<ApiResponse<AcademicPeriod[]>>('/api/v1/admin/academic-periods').pipe(
      map(res => res.data || []),
      tap(periods => {
        this.periods.set(periods);
        const active = periods.find(p => p.is_active) || periods[0] || null;
        this.activePeriod.set(active);
      })
    );
  }

  /**
   * Carga el resumen del panel docente. Una sola respuesta: aula, cola,
   * entornos abiertos y materias con sus conteos. Un error de red no borra el
   * resumen ya cargado.
   */
  loadOverview(periodId?: string): Observable<TeacherOverview> {
    this.isLoading.set(true);
    const params = periodId ? `?period_id=${periodId}` : '';
    return this.http.get<ApiResponse<TeacherOverview>>(`/api/v1/teacher/overview${params}`).pipe(
      map(res => res.data),
      tap(overview => this.overview.set(overview)),
      finalize(() => this.isLoading.set(false))
    );
  }

  /** Resumen de materias del docente. */
  loadDashboardData(periodId?: string): Observable<{ courses: TeacherCourseSummary[] }> {
    this.isLoading.set(true);
    const params = periodId ? `?period_id=${periodId}` : '';
    return this.http.get<ApiResponse<TeacherCourseSummary[]>>(`/api/v1/teacher/courses${params}`).pipe(
      map(res => res.data || []),
      tap(courses => this.courses.set(courses)),
      map(courses => ({ courses })),
      finalize(() => this.isLoading.set(false))
    );
  }

  getCourses(): Observable<TeacherCourseSummary[]> {
    return this.http.get<ApiResponse<TeacherCourseSummary[]>>('/api/v1/teacher/courses').pipe(
      map(res => res.data || []),
      tap(courses => this.courses.set(courses))
    );
  }
}
