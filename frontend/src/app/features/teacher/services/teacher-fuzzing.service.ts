import { Injectable, inject, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, map, tap } from 'rxjs';
import { 
  FuzzGenerationRequest, 
  FuzzGenerationReport, 
  GeneratedFuzzCase, 
  ApplyFuzzCasesResponse 
} from '../models/teacher.models';

interface ApiResponse<T> {
  data: T;
  error?: string;
  message?: string;
}

@Injectable({
  providedIn: 'root'
})
export class TeacherFuzzingService {
  private http = inject(HttpClient);

  readonly isGenerating = signal<boolean>(false);
  readonly isApplying = signal<boolean>(false);
  readonly currentReport = signal<FuzzGenerationReport | null>(null);

  generateFuzzCases(req: FuzzGenerationRequest): Observable<FuzzGenerationReport> {
    this.isGenerating.set(true);
    return this.http.post<ApiResponse<FuzzGenerationReport>>(
      '/api/v1/teacher/exercises/generate-fuzz-cases',
      req
    ).pipe(
      map(res => res.data),
      tap(report => {
        // Inicializar seleccionados por defecto
        report.cases.forEach(c => c.selected = true);
        this.currentReport.set(report);
        this.isGenerating.set(false);
      })
    );
  }

  applyFuzzCases(exerciseId: string, cases: GeneratedFuzzCase[]): Observable<ApplyFuzzCasesResponse> {
    this.isApplying.set(true);
    return this.http.post<ApiResponse<ApplyFuzzCasesResponse>>(
      `/api/v1/teacher/exercises/${exerciseId}/apply-fuzz-cases`,
      { cases }
    ).pipe(
      map(res => res.data),
      tap(() => {
        this.isApplying.set(false);
      })
    );
  }
}
