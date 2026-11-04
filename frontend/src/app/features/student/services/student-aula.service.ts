import { Injectable, inject, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { firstValueFrom } from 'rxjs';

interface ApiResponse<T> {
  data: T;
  error?: string;
  message?: string;
}

/**
 * Aula abierta del alumno tal como la devuelve la sesión de clase: la materia,
 * la práctica que el docente asoció y la ventana de la clase. La práctica es la
 * de la sesión (ADR-059), así que el aula siempre apunta a un único entorno.
 */
export interface StudentAula {
  id: string;
  subject_id: string;
  subject_code: string;
  subject_name: string;
  exercise_id: string | null;
  exercise_title?: string;
  purpose: string;
  status: string;
  opens_at: string;
  ends_at: string;
  closes_at: string | null;
  frozen_at: string | null;
  grace_until: string | null;
  enrolled: number;
  present: number;
}

@Injectable({
  providedIn: 'root'
})
export class StudentAulaService {
  private readonly http = inject(HttpClient);

  readonly aulas = signal<StudentAula[]>([]);
  readonly loading = signal(false);
  readonly error = signal<string | null>(null);

  async load(): Promise<void> {
    this.loading.set(true);
    this.error.set(null);
    try {
      const response = await firstValueFrom(
        this.http.get<ApiResponse<StudentAula[]>>('/api/v1/student/sessions/active')
      );
      this.aulas.set(response.data ?? []);
    } catch (err) {
      this.aulas.set([]);
      this.error.set(
        (err as { error?: { message?: string } })?.error?.message?.trim() ||
          'No se pudo leer tu aula en este momento.'
      );
    } finally {
      this.loading.set(false);
    }
  }
}
