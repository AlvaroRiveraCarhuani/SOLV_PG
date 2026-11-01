import { Injectable, OnDestroy, inject, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { firstValueFrom } from 'rxjs';

interface ApiResponse<T> {
  data: T;
  error?: string;
  message?: string;
}

/** Aviso pendiente de un avance emitido por el docente (ADR-058). */
export interface PendingClassAdvance {
  id: string;
  session_id: string;
  exercise_id: string;
  file_count: number;
  created_at: string;
  subject_id: string;
  session_status: string;
  grace_until?: string | null;
  purpose: string;
}

/** Qué cambiaría en el entorno del alumno si aplica el aviso. */
export interface AdvanceFilePreview {
  advance_id: string;
  exercise_id: string;
  file_count: number;
  added: string[];
  overwritten: string[];
}

/** Resultado de aplicar el avance: qué se agregó y qué se reemplazó. */
export interface AdvanceApplyResult {
  advance_id: string;
  files_written: number;
  overwritten: string[];
  added: string[];
  backup_path?: string | null;
}

const NOTICE_INTERVAL_MS = 60_000;

/**
 * Avisos de avance del docente en el laboratorio del alumno.
 *
 * Consulta cada minuto, en el mismo ciclo que el latido de sesión, porque así
 * lo define el flujo: no hay WebSocket ni SSE. Un aviso puede tardar hasta un
 * minuto en aparecer y la pantalla lo dice.
 */
@Injectable({
  providedIn: 'root'
})
export class ClassAdvanceNoticeService implements OnDestroy {
  private readonly http = inject(HttpClient);
  private timer?: ReturnType<typeof setInterval>;

  readonly pending = signal<PendingClassAdvance[]>([]);

  start(): void {
    if (this.timer) {
      return;
    }
    void this.refresh();
    this.timer = setInterval(() => void this.refresh(), NOTICE_INTERVAL_MS);
  }

  stop(): void {
    if (this.timer) {
      clearInterval(this.timer);
      this.timer = undefined;
    }
  }

  ngOnDestroy(): void {
    this.stop();
  }

  async refresh(): Promise<void> {
    if (typeof document !== 'undefined' && document.hidden) {
      return;
    }

    try {
      const response = await firstValueFrom(
        this.http.get<ApiResponse<PendingClassAdvance[]>>('/api/v1/class-advances/pending')
      );
      this.pending.set(response.data ?? []);
    } catch {
      // Sin red o sin sesión: el siguiente minuto lo reintenta.
    }
  }

  /** Archivos que se agregan y que se reemplazan, sin tocar el entorno. */
  async preview(advanceId: string): Promise<AdvanceFilePreview> {
    const response = await firstValueFrom(
      this.http.get<ApiResponse<AdvanceFilePreview>>(`/api/v1/class-advances/${advanceId}/files`)
    );
    return response.data;
  }

  /** Aplica el avance con respaldo previo y quita el aviso de la lista. */
  async apply(advanceId: string): Promise<AdvanceApplyResult> {
    const response = await firstValueFrom(
      this.http.post<ApiResponse<AdvanceApplyResult>>(`/api/v1/class-advances/${advanceId}/apply`, {})
    );
    this.remove(advanceId);
    return response.data;
  }

  /** Descarta el aviso sin tocar el entorno del alumno. */
  async dismiss(advanceId: string): Promise<void> {
    await firstValueFrom(
      this.http.post(`/api/v1/class-advances/${advanceId}/dismiss`, {})
    );
    this.remove(advanceId);
  }

  private remove(advanceId: string): void {
    this.pending.update(list => list.filter(advance => advance.id !== advanceId));
  }

  /**
   * Motivo del rechazo tal como lo redacta el servidor (clase congelada,
   * ventana vencida, examen, laboratorio detenido). Sin motivo, un texto propio.
   */
  static reason(err: unknown, fallback: string): string {
    const message = (err as { error?: { message?: string } })?.error?.message;
    return message?.trim() || fallback;
  }
}
