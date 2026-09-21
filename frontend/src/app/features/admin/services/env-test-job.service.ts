import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, timer } from 'rxjs';
import { map, switchMap, takeWhile, catchError } from 'rxjs/operators';

export type EnvTestStatus = 'pending' | 'pulling' | 'testing' | 'success' | 'failed' | 'canceled';

export interface ToolResult {
  name: string;
  present: boolean;
  version?: string;
  path?: string;
}

export interface EnvTestProgress {
  bytes_done: number;
  bytes_total: number;
  layer_current: number;
  layers_total: number;
  percent: number;
  current_action?: string;
}

export interface EnvTestResult {
  tools: ToolResult[];
  exit_code: number;
  duration_ms: number;
}

export interface EnvTestJob {
  id: string;
  image: string;
  tools: string[];
  status: EnvTestStatus;
  progress: EnvTestProgress;
  result?: EnvTestResult;
  error_code?: string;
  error_message?: string;
  digest_unverified: boolean;
  created_at: string;
  updated_at: string;
  finished_at?: string;
}

export interface StartEnvTestRequest {
  image: string;
  tools: string[];
}

interface ApiResponse<T> {
  data: T;
  error: string;
  message: string;
}

@Injectable({
  providedIn: 'root'
})
export class EnvTestJobService {
  private http = inject(HttpClient);
  private baseUrl = '/api/v1/jobs/env-test';

  /**
   * Inicia una prueba asíncrona de entorno
   */
  startJob(req: StartEnvTestRequest): Observable<EnvTestJob> {
    return this.http.post<ApiResponse<EnvTestJob>>(this.baseUrl, req).pipe(
      map(res => res.data)
    );
  }

  /**
   * Consulta el estado y progreso de un job específico
   */
  getJob(id: string): Observable<EnvTestJob> {
    return this.http.get<ApiResponse<EnvTestJob>>(`${this.baseUrl}/${id}`).pipe(
      map(res => res.data)
    );
  }

  /**
   * Cancela un job activo
   */
  cancelJob(id: string): Observable<{ status: string }> {
    return this.http.post<ApiResponse<{ status: string }>>(`${this.baseUrl}/${id}/cancel`, {}).pipe(
      map(res => res.data)
    );
  }

  /**
   * Polling reactivo cada intervalMs milisegundos hasta alcanzar un estado terminal
   */
  pollJob(id: string, intervalMs = 1500): Observable<EnvTestJob> {
    return timer(0, intervalMs).pipe(
      switchMap(() => this.getJob(id)),
      takeWhile(job => !this.isTerminal(job.status), true)
    );
  }

  isTerminal(status: EnvTestStatus): boolean {
    return status === 'success' || status === 'failed' || status === 'canceled';
  }
}
