import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, map } from 'rxjs';

export interface AIAvailabilityStatus {
  enabled: boolean;
  provider: string;
  model: string;
}

export interface AIGenerateExercisePayload {
  prompt: string;
  language: string;
  difficulty: 'easy' | 'medium' | 'hard';
  public_cases_count: number;
  hidden_cases_count: number;
  verify_with_runner?: boolean;
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
