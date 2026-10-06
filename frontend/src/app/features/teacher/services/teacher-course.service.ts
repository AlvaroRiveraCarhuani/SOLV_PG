import { Injectable, inject, signal } from '@angular/core';
import { HttpClient, HttpParams } from '@angular/common/http';
import { Observable, tap, map } from 'rxjs';
import {
  TeacherLabStats,
  SubmissionQueueItem,
  CreateExerciseRequestDTO,
  UpdateExerciseRequestDTO,
  BulkTestCasesRequestDTO,
  PlagiarismReport,
  CourseGradesMatrix,
  CourseAnalyticsDTO,
  TeacherCourseModule,
  CreateModuleDTO,
  UpdateModuleDTO,
  SetPrerequisitesDTO,
  ExerciseImportResponse,
  ScriptGenerationResponse
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

  getExercise(exerciseId: string): Observable<any> {
    return this.http.get<ApiResponse<any>>(`/api/v1/exercises/${exerciseId}`).pipe(
      map(res => res.data)
    );
  }

  validateInputFormat(contract: any, input: string): Observable<{ valid: boolean; error?: string }> {
    return this.http.post<ApiResponse<{ valid: boolean; error?: string }>>('/api/v1/exercises/validate-input', {
      contract,
      input
    }).pipe(
      map(res => res.data)
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

  startDryRun(exerciseId: string): Observable<any> {
    return this.http.post<ApiResponse<any>>(`/api/v1/exercises/${exerciseId}/dry-run`, {}).pipe(
      map(res => res.data)
    );
  }

  getDryRunJob(jobId: string, exerciseId?: string): Observable<any> {
    const url = exerciseId 
      ? `/api/v1/exercises/${exerciseId}/dry-run/jobs/${jobId}`
      : `/api/v1/dry-run/jobs/${jobId}`;
    return this.http.get<ApiResponse<any>>(url).pipe(
      map(res => res.data)
    );
  }

  getExerciseChecklist(exerciseId: string, referenceSolution?: { code: string; language: string }): Observable<any> {
    return this.http.post<ApiResponse<any>>(`/api/v1/exercises/${exerciseId}/checklist`, {
      reference_solution: referenceSolution
    }).pipe(
      map(res => res.data)
    );
  }

  generateCases(contract: any, count: number, seed?: number): Observable<{ cases: Array<{ input: string; output: string | null }> }> {
    return this.http.post<ApiResponse<{ cases: Array<{ input: string; output: string | null }> }>>('/api/v1/exercises/generate-cases', {
      contract,
      count,
      seed: seed !== undefined && seed !== null && !isNaN(seed) ? Number(seed) : undefined
    }).pipe(
      map(res => res.data)
    );
  }

  calculateOutputs(payload: {
    language: string;
    source_code: string;
    inputs: string[];
    time_limit_ms?: number;
    memory_limit_mb?: number;
  }): Observable<{
    outputs: Array<{
      index: number;
      input: string;
      expected_output: string;
      status: string;
      execution_time_ms: number;
      error_details?: string;
    }>;
  }> {
    return this.http.post<ApiResponse<{
      outputs: Array<{
        index: number;
        input: string;
        expected_output: string;
        status: string;
        execution_time_ms: number;
        error_details?: string;
      }>;
    }>>('/api/v1/exercises/calculate-outputs', payload).pipe(
      map(res => res.data)
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

  getCourseGradesMatrix(subjectId: string): Observable<CourseGradesMatrix> {
    return this.http.get<ApiResponse<CourseGradesMatrix>>(`/api/v1/teacher/courses/${subjectId}/grades/matrix`).pipe(
      map(res => res.data)
    );
  }

  exportGradesCsv(subjectId: string): Observable<Blob> {
    return this.http.get(`/api/v1/teacher/courses/${subjectId}/grades/export`, {
      params: new HttpParams().set('format', 'csv'),
      responseType: 'blob'
    });
  }


  analyzePlagiarism(subjectId: string, exerciseId?: string): Observable<PlagiarismReport> {
    let params = new HttpParams();
    if (exerciseId && exerciseId !== 'all') {
      params = params.set('exercise_id', exerciseId);
    }
    return this.http.get<ApiResponse<PlagiarismReport>>(
      `/api/v1/teacher/courses/${subjectId}/plagiarism`,
      { params }
    ).pipe(
      map(res => res.data)
    );
  }

  getCourseAnalytics(courseId: string): Observable<CourseAnalyticsDTO> {
    return this.http.get<ApiResponse<CourseAnalyticsDTO>>(`/api/v1/teacher/courses/${courseId}/analytics`).pipe(
      map(res => res.data)
    );
  }

  getCourseModules(subjectId: string): Observable<TeacherCourseModule[]> {
    return this.http.get<ApiResponse<TeacherCourseModule[]>>(`/api/v1/courses/${subjectId}/modules`).pipe(
      map(res => res.data || [])
    );
  }

  createModule(subjectId: string, dto: CreateModuleDTO): Observable<TeacherCourseModule> {
    return this.http.post<ApiResponse<TeacherCourseModule>>(`/api/v1/courses/${subjectId}/modules`, dto).pipe(
      map(res => res.data)
    );
  }

  updateModule(subjectId: string, moduleId: string, dto: UpdateModuleDTO): Observable<TeacherCourseModule> {
    return this.http.put<ApiResponse<TeacherCourseModule>>(`/api/v1/courses/${subjectId}/modules/${moduleId}`, dto).pipe(
      map(res => res.data)
    );
  }

  deleteModule(subjectId: string, moduleId: string): Observable<void> {
    return this.http.delete<ApiResponse<void>>(`/api/v1/courses/${subjectId}/modules/${moduleId}`).pipe(
      map(() => void 0)
    );
  }

  setModulePrerequisites(subjectId: string, moduleId: string, prerequisiteModuleIds: string[]): Observable<void> {
    return this.http.put<ApiResponse<void>>(`/api/v1/courses/${subjectId}/modules/${moduleId}/prerequisites`, {
      prerequisite_module_ids: prerequisiteModuleIds
    }).pipe(
      map(() => void 0)
    );
  }

  assignExercisesToModule(subjectId: string, moduleId: string, exerciseIds: string[]): Observable<void> {
    return this.http.put<ApiResponse<void>>(`/api/v1/courses/${subjectId}/modules/${moduleId}/exercises`, {
      exercise_ids: exerciseIds
    }).pipe(
      map(() => void 0)
    );
  }

  assignExerciseModule(exerciseId: string, moduleId: string | null): Observable<void> {
    return this.http.put<ApiResponse<void>>(`/api/v1/exercises/${exerciseId}/module`, {
      module_id: moduleId
    }).pipe(
      map(() => void 0)
    );
  }

  importExercises(courseId: string, file: File, dryRun = false): Observable<ExerciseImportResponse> {
    const formData = new FormData();
    formData.append('file', file, file.name);

    let params = new HttpParams();
    if (dryRun) {
      params = params.set('dry_run', 'true');
    }

    return this.http.post<ApiResponse<ExerciseImportResponse>>(
      `/api/v1/teacher/courses/${courseId}/exercises/import`,
      formData,
      { params }
    ).pipe(
      map(res => res.data)
    );
  }

  generateCasesFromScript(exerciseId: string, script: string, dryRun = true): Observable<ScriptGenerationResponse> {
    let params = new HttpParams();
    if (dryRun) {
      params = params.set('dry_run', 'true');
    }

    return this.http.post<ApiResponse<ScriptGenerationResponse>>(
      `/api/v1/teacher/exercises/${exerciseId}/generate-from-script`,
      { script },
      { params }
    ).pipe(
      map(res => res.data)
    );
  }
}
