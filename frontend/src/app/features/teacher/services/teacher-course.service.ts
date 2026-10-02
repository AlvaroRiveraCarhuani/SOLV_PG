import { Injectable, inject, signal } from '@angular/core';
import { HttpClient, HttpParams } from '@angular/common/http';
import { Observable, tap, map } from 'rxjs';
import {
  TeacherLabStats,
  SubmissionQueueItem,
  CreateExerciseRequestDTO,
  UpdateExerciseRequestDTO,
  BulkTestCasesRequestDTO
} from '../models/teacher.models';

interface ApiResponse<T> {
  data: T;
  error?: string;
  message?: string;
}

@Injectable({
  providedIn: 'root'
})
export class TeacherCourseService {
  private http = inject(HttpClient);

  readonly labs = signal<TeacherLabStats[]>([]);
  readonly submissions = signal<SubmissionQueueItem[]>([]);
  readonly isLoading = signal<boolean>(false);

  getCourseLabs(subjectId: string): Observable<TeacherLabStats[]> {
    this.isLoading.set(true);
    return this.http.get<ApiResponse<TeacherLabStats[]>>(`/api/v1/teacher/courses/${subjectId}/labs`).pipe(
      map(res => res.data || []),
      tap(labs => {
        this.labs.set(labs);
        this.isLoading.set(false);
      })
    );
  }

  getCourseSubmissions(
    subjectId: string,
    filters?: { verdict?: string; exercise_id?: string }
  ): Observable<SubmissionQueueItem[]> {
    this.isLoading.set(true);
    let params = new HttpParams();
    if (filters?.verdict && filters.verdict !== 'all') {
      params = params.set('verdict', filters.verdict);
    }
    if (filters?.exercise_id && filters.exercise_id !== 'all') {
      params = params.set('exercise_id', filters.exercise_id);
    }

    return this.http.get<ApiResponse<SubmissionQueueItem[]>>(
      `/api/v1/teacher/courses/${subjectId}/submissions`,
      { params }
    ).pipe(
      map(res => res.data || []),
      tap(subs => {
        this.submissions.set(subs);
        this.isLoading.set(false);
      })
    );
  }

  createExercise(dto: CreateExerciseRequestDTO): Observable<{ id: string }> {
    return this.http.post<ApiResponse<{ id: string }>>('/api/v1/exercises', dto).pipe(
      map(res => res.data)
    );
  }

  updateExercise(exerciseId: string, dto: UpdateExerciseRequestDTO): Observable<void> {
    return this.http.put<ApiResponse<void>>(`/api/v1/exercises/${exerciseId}`, dto).pipe(
      map(() => void 0)
    );
  }

  publishExercise(exerciseId: string): Observable<void> {
    return this.http.post<ApiResponse<void>>(`/api/v1/exercises/${exerciseId}/publish`, {}).pipe(
      map(() => void 0)
    );
  }

  bulkUploadTestCases(exerciseId: string, dto: BulkTestCasesRequestDTO): Observable<{ imported_count: number }> {
    return this.http.post<ApiResponse<{ imported_count: number }>>(
      `/api/v1/exercises/${exerciseId}/test-cases/bulk`,
      dto
    ).pipe(
      map(res => res.data)
    );
  }

  exportGradesCsv(subjectId: string): Observable<Blob> {
    return this.http.get(`/api/v1/teacher/courses/${subjectId}/grades/export`, {
      params: new HttpParams().set('format', 'csv'),
      responseType: 'blob'
    });
  }
}
