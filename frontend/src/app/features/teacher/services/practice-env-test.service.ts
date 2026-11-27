import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, of, throwError, timer } from 'rxjs';
import { catchError, map, switchMap, takeWhile } from 'rxjs/operators';

import {
  PracticeEnvTestJob,
  PracticeEnvTestStatus,
  StudentPracticeDatabase
} from '../models/practice-env-test.models';

interface ApiResponse<T> {
  data: T;
  error?: string;
  message?: string;
}

@Injectable({
  providedIn: 'root'
})
export class PracticeEnvTestService {
  private http = inject(HttpClient);

  start(courseId: string, exerciseId: string): Observable<PracticeEnvTestJob> {
    return this.http
      .post<ApiResponse<PracticeEnvTestJob>>(
        `/api/v1/teacher/courses/${courseId}/exercises/${exerciseId}/env-test`,
        {}
      )
      .pipe(
        map(res => res.data),
        catchError(err => throwError(() => new Error(this.startErrorMessage(err))))
      );
  }

  /**
   * Mensaje de la prueba en palabras: un nodo cerca de su capacidad o con un
   * build en curso no es un fallo del entorno ni obliga a calcular memoria.
   */
  startErrorMessage(err: unknown): string {
    const status = (err as { status?: number } | null)?.status;
    const serverMessage = (err as { error?: { message?: string } } | null)?.error?.message;
    if (status === 503) {
      return 'El servidor está cerca de su capacidad con laboratorios activos; intenta de nuevo en unos minutos.';
    }
    if (status === 429) {
      return 'El servidor está construyendo otra imagen en este momento; intenta de nuevo en unos minutos.';
    }
    return serverMessage || 'No se pudo iniciar la prueba de entorno.';
  }

  latest(exerciseId: string): Observable<PracticeEnvTestJob | null> {
    return this.http
      .get<ApiResponse<PracticeEnvTestJob | null>>(
        `/api/v1/teacher/exercises/${exerciseId}/env-test/latest`
      )
      .pipe(
        map(res => res.data ?? null),
        catchError(err => (err?.status === 404 ? of(null) : throwError(() => err)))
      );
  }

  listStudentDatabases(courseId: string, exerciseId: string): Observable<StudentPracticeDatabase[]> {
    return this.http
      .get<ApiResponse<StudentPracticeDatabase[]>>(
        `/api/v1/teacher/courses/${courseId}/exercises/${exerciseId}/student-databases`
      )
      .pipe(map(res => res.data ?? []));
  }

  resetStudentDatabase(
    courseId: string,
    exerciseId: string,
    studentId: string
  ): Observable<{ student_id: string }> {
    return this.http
      .post<ApiResponse<{ student_id: string }>>(
        `/api/v1/teacher/courses/${courseId}/exercises/${exerciseId}/students/${studentId}/database/reset`,
        {}
      )
      .pipe(map(res => res.data));
  }

  getJob(jobId: string): Observable<PracticeEnvTestJob> {
    return this.http
      .get<ApiResponse<PracticeEnvTestJob>>(`/api/v1/jobs/env-test/${jobId}`)
      .pipe(map(res => res.data));
  }

  cancelJob(jobId: string): Observable<{ status: string }> {
    return this.http
      .post<ApiResponse<{ status: string }>>(`/api/v1/jobs/env-test/${jobId}/cancel`, {})
      .pipe(map(res => res.data));
  }

  pollJob(jobId: string, intervalMs = 2000): Observable<PracticeEnvTestJob> {
    return timer(0, intervalMs).pipe(
      switchMap(() => this.getJob(jobId)),
      takeWhile(job => !this.isTerminal(job.status), true)
    );
  }

  isTerminal(status: PracticeEnvTestStatus): boolean {
    return status === 'success' || status === 'failed' || status === 'canceled';
  }
}
