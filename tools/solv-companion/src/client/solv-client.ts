import { SolvSettings } from '../config/solv-settings';
import { SubmissionComment } from '../domain/comment';
import { KeystrokeEvent } from '../domain/keystroke';

export class SolvApiError extends Error {
  constructor(
    message: string,
    readonly status?: number
  ) {
    super(message);
    this.name = 'SolvApiError';
  }
}

interface ApiEnvelope<T> {
  data?: T;
  message?: string;
  error?: string;
}

const REQUEST_TIMEOUT_MS = 10000;

/** Único punto de salida HTTP de la extensión. El token viaja como Bearer y nunca se registra ni se muestra. */
export class SolvClient {
  constructor(private readonly settings: SolvSettings) {}

  private async request<T>(path: string, init: RequestInit): Promise<T> {
    const baseUrl = this.settings.apiBaseUrl;
    // Preferir token de entorno (inyectado por OpenVSCode Server), fallback a SecretStorage.
    const token = await this.settings.getTokenEnv();
    const submissionId = this.settings.submissionId;
    if (!baseUrl || !submissionId || !token) {
      throw new SolvApiError('Falta configurar la conexión con SOLV.');
    }

    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), REQUEST_TIMEOUT_MS);
    try {
      const response = await fetch(`${baseUrl}${path}`, {
        ...init,
        signal: controller.signal,
        headers: {
          Accept: 'application/json',
          Authorization: `Bearer ${token}`,
          'Content-Type': 'application/json'
        }
      });
      const text = await response.text();
      const payload = this.parsePayload<T>(text);

      if (!response.ok) {
        throw new SolvApiError(
          this.errorMessage(payload, response.status),
          response.status
        );
      }

      return payload?.data !== undefined ? payload.data : (payload as T);
    } catch (error) {
      if (error instanceof SolvApiError) {
        throw error;
      }
      if (error instanceof Error && error.name === 'AbortError') {
        throw new SolvApiError('La API de SOLV no respondió a tiempo.');
      }
      throw new SolvApiError('No se pudo conectar con la API de SOLV.');
    } finally {
      clearTimeout(timeout);
    }
  }

  async fetchComments(): Promise<SubmissionComment[]> {
    const comments = await this.request<SubmissionComment[]>(this.commentsPath(), {
      method: 'GET'
    });
    return Array.isArray(comments) ? comments : [];
  }

  async sendKeystrokeEvents(events: KeystrokeEvent[]): Promise<void> {
    if (events.length === 0) {
      return;
    }
    await this.request<unknown>(this.keystrokePath(), {
      method: 'POST',
      body: JSON.stringify({ events })
    });
  }

  private commentsPath(): string {
    return `/api/v1/student/submissions/${encodeURIComponent(
      this.settings.submissionId
    )}/comments`;
  }

  private keystrokePath(): string {
    return `/api/v1/submissions/${encodeURIComponent(this.settings.submissionId)}/keystroke-events`;
  }

  private parsePayload<T>(text: string): ApiEnvelope<T> | null {
    if (text.trim() === '') {
      return null;
    }
    try {
      return JSON.parse(text) as ApiEnvelope<T>;
    } catch {
      return null;
    }
  }

  private errorMessage(payload: ApiEnvelope<unknown> | null, status: number): string {
    if (payload?.message) {
      return payload.message;
    }
    if (payload?.error) {
      return payload.error;
    }
    if (status === 401) {
      return 'El token de SOLV no es válido o expiró.';
    }
    if (status === 404) {
      return 'La entrega no existe o no pertenece a la sesión.';
    }
    return `La API de SOLV respondió con estado ${status}.`;
  }
}