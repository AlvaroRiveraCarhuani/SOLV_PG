import { Injectable, inject } from '@angular/core';
import { HttpClient, HttpParams } from '@angular/common/http';
import { Observable, map } from 'rxjs';
import {
  CreateEnvironmentRequestPayload,
  EnvironmentRequest,
  EnvironmentRequestCreated,
  EnvironmentRequestPreview,
  EnvironmentRequestStackConfig,
  EnvironmentRequestUse,
  ReviewEnvironmentRequestPayload
} from '../templates/environment-request.models';

interface ApiResponse<T> {
  data: T;
  error?: string;
  message?: string;
}

export interface SuggestEnvironmentResponse {
  name: string;
  description: string;
  runtime_slug: string;
  web_platform_slug?: string;
  addon_slugs?: string[];
  database_slug?: string;
  base_ram_mb: number;
  rationale: string;
  missing_components?: { name: string; layer: string; category?: string; reason?: string }[];
}

/**
 * Solicitudes de entorno del docente. El docente envía una intención y el
 * backend arma la receta con el catálogo: aquí no viaja ninguna imagen.
 */
@Injectable({ providedIn: 'root' })
export class EnvironmentRequestService {
  private http = inject(HttpClient);
  private readonly baseUrl = '/api/v1/teacher/environment-requests';
  private readonly adminUrl = '/api/v1/admin/environment-requests';

  preview(uses: EnvironmentRequestUse[], stackConfig: EnvironmentRequestStackConfig): Observable<EnvironmentRequestPreview> {
    return this.http
      .post<ApiResponse<EnvironmentRequestPreview>>(`${this.baseUrl}/preview`, {
        uses,
        stack_config: stackConfig
      })
      .pipe(map(res => res.data));
  }

  create(payload: CreateEnvironmentRequestPayload): Observable<EnvironmentRequestCreated> {
    return this.http
      .post<ApiResponse<EnvironmentRequestCreated>>(this.baseUrl, payload)
      .pipe(map(res => ({ ...res.data, message: res.data?.message || res.message || '' })));
  }

  listMine(): Observable<EnvironmentRequest[]> {
    return this.http.get<ApiResponse<EnvironmentRequest[]>>(this.baseUrl).pipe(map(res => res.data || []));
  }

  listForReview(status?: string): Observable<EnvironmentRequest[]> {
    let params = new HttpParams();
    if (status) {
      params = params.set('status', status);
    }
    return this.http
      .get<ApiResponse<EnvironmentRequest[]>>(this.adminUrl, { params })
      .pipe(map(res => res.data || []));
  }

  review(requestId: string, payload: ReviewEnvironmentRequestPayload): Observable<EnvironmentRequest> {
    return this.http
      .patch<ApiResponse<EnvironmentRequest>>(`${this.adminUrl}/${requestId}`, payload)
      .pipe(map(res => res.data));
  }

  /**
   * Interpreta la descripción del docente y devuelve herramientas del catálogo.
   * Es una ayuda: si falla o no hay IA configurada, el formulario sigue
   * funcionando a mano.
   */
  suggestEnvironment(prompt: string): Observable<SuggestEnvironmentResponse> {
    return this.http
      .post<ApiResponse<SuggestEnvironmentResponse>>('/api/v1/teacher/ai/suggest-environment', { prompt })
      .pipe(map(res => res.data));
  }
}
