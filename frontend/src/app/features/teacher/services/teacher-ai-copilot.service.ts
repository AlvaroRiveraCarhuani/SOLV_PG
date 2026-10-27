import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, map } from 'rxjs';

export interface AIAvailabilityStatus {
  enabled: boolean;
  provider: string;
  model: string;
  using_personal_key?: boolean;
  institutional_enabled?: boolean;
}

export interface AIGenerateExercisePayload {
  prompt: string;
  language: string;
  difficulty: 'easy' | 'medium' | 'hard';
  public_cases_count: number;
  hidden_cases_count: number;
  verify_with_runner?: boolean;
}

export interface AIUserOverrideDTO {
  enabled: boolean;
  provider: string;
  provider_label: string;
  model: string;
  base_url: string;
  has_api_key: boolean;
  api_key_masked: string;
  updated_at: string;
  institutional_enabled: boolean;
  institutional_provider_label: string;
  institutional_model: string;
}

export interface UpdateAIUserOverridePayload {
  enabled: boolean;
  provider: string;
  model: string;
  base_url?: string;
  api_key?: string;
  clear_api_key?: boolean;
}

export interface AIGeneratedTestCaseDTO {
  order_index: number;
  input: string;
  expected_output: string;
  visibility: 'example' | 'public' | 'hidden';
  weight: number;
  verified_by_runner: boolean;
}

export interface AIGenerateExerciseResponse {
  title: string;
  description: string;
  difficulty: 'easy' | 'medium' | 'hard';
  language: string;
  boilerplate: string;
  reference_solution: string;
  test_cases: AIGeneratedTestCaseDTO[];
  runner_verified_all: boolean;
  verification_warning?: string;
  latency_ms: number;
}

interface ApiResponse<T> {
  data: T;
  message?: string;
}

interface ApiErrorBody {
  message?: string;
  code?: string;
}

// El listado de modelos BYOK llega como { data: { models: [...] } }; por contrato
// previo también se aceptan el array plano y el body { models: [...] }. Cada modelo
// puede ser el id directamente o un objeto { id } según el proveedor.

function toByokModelId(m: unknown): string {
  if (typeof m === 'string') {
    return m;
  }
  if (typeof m === 'object' && m !== null && 'id' in m) {
    const id = (m as { id?: unknown }).id;
    return typeof id === 'string' ? id : '';
  }
  return '';
}

@Injectable({ providedIn: 'root' })
export class TeacherAiCopilotService {
  private readonly http = inject(HttpClient);

  getStatus(): Observable<AIAvailabilityStatus> {
    return this.http
      .get<ApiResponse<AIAvailabilityStatus>>('/api/v1/ai/status')
      .pipe(map((res) => res.data));
  }

  generateExercise(payload: AIGenerateExercisePayload): Observable<AIGenerateExerciseResponse> {
    return this.http
      .post<ApiResponse<AIGenerateExerciseResponse>>(
        '/api/v1/teacher/ai/generate-exercise',
        payload
      )
      .pipe(map((res) => res.data));
  }

  getMySettings(): Observable<AIUserOverrideDTO> {
    return this.http
      .get<ApiResponse<AIUserOverrideDTO>>('/api/v1/ai/my-settings')
      .pipe(map((res) => res.data));
  }

  updateMySettings(payload: UpdateAIUserOverridePayload): Observable<AIUserOverrideDTO> {
    return this.http
      .put<ApiResponse<AIUserOverrideDTO>>('/api/v1/ai/my-settings', payload)
      .pipe(map((res) => res.data));
  }

  testMyConnection(payload: { provider: string; model: string; api_key?: string; base_url?: string }): Observable<{ ok: boolean; message: string; latency_ms: number }> {
    return this.http
      .post<ApiResponse<{ ok: boolean; message: string; latency_ms: number }>>('/api/v1/ai/my-settings/test-connection', payload)
      .pipe(map((res) => res.data));
  }

  listMyModels(payload: { provider: string; api_key?: string; base_url?: string }): Observable<string[]> {
    return this.http
      .post<{ data?: unknown; models?: unknown }>('/api/v1/ai/my-settings/models', payload)
      .pipe(
        map((envelope) => {
          if (!envelope) return [];
          const body: unknown = envelope.data ?? envelope;
          if (Array.isArray(body)) {
            return body.map(toByokModelId);
          }
          const bodyModels = (body as { models?: unknown } | null)?.models;
          const models: unknown = bodyModels ?? envelope.models ?? [];
          if (Array.isArray(models)) {
            return models.map(toByokModelId);
          }
          return [];
        })
      );
  }

  resolveError(err: unknown): string {
    const body = (err as { error?: ApiErrorBody } | null)?.error;
    const status = (err as { status?: number } | null)?.status;

    if (body?.code === 'ai_disabled') {
      return body.message || 'El Copiloto IA está deshabilitado para esta institución.';
    }
    if (body?.code === 'ai_quota_exceeded') {
      return body.message || 'Se agotó la cuota horaria de IA. Intenta de nuevo más tarde.';
    }
    if (body?.message) {
      return body.message;
    }
    if (status === 401) {
      return 'Tu sesión expiró. Vuelve a iniciar sesión para usar el Copiloto IA.';
    }
    if (status === 403) {
      return 'No tienes permiso para generar borradores con IA.';
    }
    if (status === 429) {
      return 'Se agotó la cuota horaria de IA. Intenta de nuevo más tarde.';
    }
    if (status === 409) {
      return 'El Copiloto IA está deshabilitado para esta institución.';
    }
    return 'No se pudo contactar al Copiloto IA. Intenta nuevamente.';
  }
}
