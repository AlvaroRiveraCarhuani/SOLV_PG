import { Injectable, OnDestroy, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { firstValueFrom } from 'rxjs';

interface ApiResponse<T> {
  data: T;
  error?: string;
  message?: string;
}

interface ActiveAula {
  id: string;
  subject_name: string;
}

const PRESENCE_INTERVAL_MS = 60_000;

/**
 * Señal de presencia del alumno en el aula.
 *
 * La señal es la dependencia que declara la sesión de aula: sin ella el tablero
 * del docente solo podría contar inscripciones, y "quién falta" sería una
 * suposición. Late una vez por minuto y solo mientras la pestaña está visible;
 * un aula cerrada responde 409 y el cliente deja de intentarlo.
 */
@Injectable({
  providedIn: 'root'
})
export class ClassPresenceService implements OnDestroy {
  private readonly http = inject(HttpClient);
  private timer?: ReturnType<typeof setInterval>;
  private readonly closedAulas = new Set<string>();

  start(): void {
    if (this.timer) {
      return;
    }
    void this.ping();
    this.timer = setInterval(() => void this.ping(), PRESENCE_INTERVAL_MS);
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

  private async ping(): Promise<void> {
    if (typeof document !== 'undefined' && document.hidden) {
      return;
    }

    try {
      const response = await firstValueFrom(
        this.http.get<ApiResponse<ActiveAula[]>>('/api/v1/student/sessions/active')
      );
      const aulas = (response.data ?? []).filter(aula => !this.closedAulas.has(aula.id));
      for (const aula of aulas) {
        await this.beat(aula.id);
      }
    } catch {
      // Sin red o sin sesión: el siguiente minuto lo reintenta.
    }
  }

  private async beat(sessionId: string): Promise<void> {
    try {
      await firstValueFrom(
        this.http.post<ApiResponse<{ ok: boolean }>>(
          `/api/v1/student/sessions/${sessionId}/heartbeat`,
          {}
        )
      );
    } catch (err) {
      const status = (err as { status?: number }).status;
      if (status === 409 || status === 403 || status === 404) {
        // El aula se cerró o el alumno dejó de pertenecer a ella: no se insiste.
        this.closedAulas.add(sessionId);
      }
    }
  }
}
