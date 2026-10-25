import { Injectable, inject, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { map, tap } from 'rxjs/operators';

export type AIProviderType =
  | 'gemini'
  | 'openai'
  | 'openrouter'
  | 'deepseek'
  | 'ollama_lan'
  | `custom_${string}`;

/** Perfil guardado de un proveedor base (GET /api/v1/admin/ai/settings). */
export interface AIProviderProfileDTO {
  model: string;
  base_url: string;
  has_api_key: boolean;
  api_key_masked: string;
}

/** Proveedor personalizado compatible con OpenAI (entrada y respuesta del PUT). */
export interface AICustomProviderDTO {
  id: string;
  label: string;
  icon_url?: string;
  base_url: string;
  model: string;
  api_key?: string;
  clear_api_key?: boolean;
  has_api_key?: boolean;
  api_key_masked?: string;
}

export interface AISettingsDTO {
  enabled: boolean;
  provider: AIProviderType;
  provider_label?: string;
  model: string;
  base_url: string;
  has_api_key: boolean;
  api_key_masked: string;
  hourly_quota_per_user: number;
  timeout_seconds: number;
  updated_at?: string;
  provider_profiles?: Record<string, AIProviderProfileDTO>;
  custom_providers?: AICustomProviderDTO[];
}

export interface UpdateAISettingsDTO {
  enabled: boolean;
  provider: AIProviderType;
  model: string;
  base_url: string;
  api_key: string;
  clear_api_key: boolean;
  hourly_quota_per_user: number;
  timeout_seconds: number;
  custom_providers: AICustomProviderDTO[];
}

export interface TestAIConnectionPayload {
  provider: AIProviderType;
  model: string;
  base_url: string;
  api_key: string;
}

export interface TestAIConnectionResponse {
  ok: boolean;
  provider: AIProviderType;
  model: string;
  latency_ms: number;
  message: string;
}

export interface AIStatusDTO {
  enabled: boolean;
  provider: AIProviderType;
  model: string;
}

export interface AISuggestComponentPayload {
  prompt: string;
  layer?: 'runtime' | 'addon' | 'database';
  target_runtime?: string;
}

export interface AISuggestedComponentDTO {
  name: string;
  slug?: string;
  layer: 'runtime' | 'addon' | 'database';
  sub_category?: string;
  description?: string;
  icon_key?: string;
  base_image?: string;
  compatible_runtimes?: string[];
  install_mode?: string;
  package_names?: string[];
  build_layer_snippet?: string;
  setup_snippet?: string;
  judge_entrypoint?: string;
  tools_provided?: string[];
  ram_delta_mb?: number;
}

export interface AISuggestComponentResponse {
  component: AISuggestedComponentDTO;
  rationale: string;
  latency_ms: number;
}

export type AIModelTier = 'free' | 'paid' | 'local' | 'by_account';

export interface AIModelItemDTO {
  id: string;
  name: string;
  tier: AIModelTier;
  pricing_hint: string;
  context_length?: number;
}

export interface ListAIModelsPayload {
  provider: AIProviderType;
  base_url: string;
  api_key: string;
  refresh?: boolean;
}

export interface ListAIModelsResponse {
  provider: AIProviderType;
  models: AIModelItemDTO[];
  cached: boolean;
  count?: number;
  fetched_at?: string;
}

interface ApiEnvelope<T> {
  data: T;
  message?: string;
}

/** Service for AI configuration */
@Injectable({ providedIn: 'root' })
export class AdminConfigIaService {
  private readonly http = inject(HttpClient);
  private readonly _settings = signal<AISettingsDTO | null>(null);
  readonly settings = this._settings.asReadonly();

  /** GET settings */
  getSettings(): Observable<AISettingsDTO> {
    return this.http
      .get<ApiEnvelope<AISettingsDTO> | AISettingsDTO>('/api/v1/admin/ai/settings')
      .pipe(
        map((res) => (res as ApiEnvelope<AISettingsDTO>).data ?? (res as AISettingsDTO)),
        tap((data) => this._settings.set(data))
      );
  }

  /** PUT update */
  updateSettings(dto: UpdateAISettingsDTO): Observable<AISettingsDTO> {
    return this.http
      .put<ApiEnvelope<AISettingsDTO> | AISettingsDTO>('/api/v1/admin/ai/settings', dto)
      .pipe(
        map((res) => (res as ApiEnvelope<AISettingsDTO>).data ?? (res as AISettingsDTO)),
        tap((data) => this._settings.set(data))
      );
  }

  /** POST test connection */
  testConnection(payload: TestAIConnectionPayload): Observable<TestAIConnectionResponse> {
    return this.http
      .post<ApiEnvelope<TestAIConnectionResponse> | TestAIConnectionResponse>(
        '/api/v1/admin/ai/test-connection',
        payload
      )
      .pipe(
        map((res) => (res as ApiEnvelope<TestAIConnectionResponse>).data ?? (res as TestAIConnectionResponse))
      );
  }

  /** GET status */
  getStatus(): Observable<AIStatusDTO> {
    return this.http
      .get<ApiEnvelope<AIStatusDTO> | AIStatusDTO>('/api/v1/ai/status')
      .pipe(
        map((res) => (res as ApiEnvelope<AIStatusDTO>).data ?? (res as AIStatusDTO))
      );
  }

  /** POST suggest component */
  suggestComponent(payload: AISuggestComponentPayload): Observable<AISuggestComponentResponse> {
    return this.http
      .post<ApiEnvelope<AISuggestComponentResponse> | AISuggestComponentResponse>(
        '/api/v1/admin/ai/suggest-component',
        payload
      )
      .pipe(
        map((res) => (res as ApiEnvelope<AISuggestComponentResponse>).data ?? (res as AISuggestComponentResponse))
      );
  }

  /** POST list models from provider */
  listModels(payload: ListAIModelsPayload): Observable<ListAIModelsResponse> {
    return this.http
      .post<ApiEnvelope<ListAIModelsResponse> | ListAIModelsResponse>(
        '/api/v1/admin/ai/models',
        payload
      )
      .pipe(map((res) => (res as ApiEnvelope<ListAIModelsResponse>).data ?? (res as ListAIModelsResponse)));
  }

  resolveError(err: unknown): string {
    const e = err as { error?: { message?: string; error?: string }; message?: string };
    return e?.error?.message ?? e?.message ?? 'Ocurrió un error al procesar la configuración de IA.';
  }
}
