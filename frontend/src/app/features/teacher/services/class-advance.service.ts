import { Injectable, inject, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, catchError, finalize, map, of, tap, throwError } from 'rxjs';
import { ClassAdvanceEmission, EmitAdvanceRequest, SessionAdvance } from '../models/teacher.models';

interface ApiResponse<T> {
  data: T;
  error?: string;
  message?: string;
}

/**
 * Emisión de avances del docente (ADR-058). El servidor empaqueta el código del
 * entorno de clase en el momento de emitir: el alumno recibe esa versión aunque
 * el docente siga editando después.
 */
@Injectable({
  providedIn: 'root'
})
export class ClassAdvanceService {
  private readonly http = inject(HttpClient);

  readonly isEmitting = signal<boolean>(false);
  readonly error = signal<string | null>(null);
  /** Última emisión de esta pantalla: cuántos alumnos la recibieron. Retiene
   *  la práctica para que el aviso nombre qué se emitió. */
  readonly lastEmission = signal<{ title: string; recipients: number } | null>(null);

  /** Emisiones del aula con el estado de cada destinatario. */
  readonly sessionAdvances = signal<SessionAdvance[]>([]);
  readonly isLoadingAdvances = signal<boolean>(false);
  readonly advancesError = signal<string | null>(null);

  emit(sessionId: string, request: EmitAdvanceRequest, title: string): Observable<ClassAdvanceEmission> {
    this.isEmitting.set(true);
    return this.http
      .post<ApiResponse<ClassAdvanceEmission>>(`/api/v1/teacher/sessions/${sessionId}/advances`, request)
      .pipe(
        map(res => res.data),
        tap(emission => {
          this.lastEmission.set({ title, recipients: emission.recipients.length });
          this.error.set(null);
        }),
        catchError(err => {
          this.error.set(this.describe(err));
          return throwError(() => err);
        }),
        finalize(() => this.isEmitting.set(false))
      );
  }

  /**
   * Lee quién aplicó, quién descartó y quién sigue pendiente en el aula. Un
   * error no vacía la lista anterior: la pantalla conserva lo último que supo.
   */
  loadSessionAdvances(sessionId: string): Observable<SessionAdvance[]> {
    this.isLoadingAdvances.set(true);
    this.advancesError.set(null);
    return this.http
      .get<ApiResponse<SessionAdvance[]>>(`/api/v1/teacher/sessions/${sessionId}/advances`)
      .pipe(
        map(res => res.data ?? []),
        tap(list => this.sessionAdvances.set(list)),
        catchError(err => {
          this.advancesError.set(this.describe(err));
          return of(this.sessionAdvances());
        }),
        finalize(() => this.isLoadingAdvances.set(false))
      );
  }

  clearNotice(): void {
    this.lastEmission.set(null);
  }

  clearError(): void {
    this.error.set(null);
  }

  /** El backend devuelve el motivo legible en `error`; se usa ese, no un texto genérico. */
  private describe(err: unknown): string {
    const typed = err as { error?: { error?: string; message?: string }; message?: string };
    return typed?.error?.error || typed?.error?.message || typed?.message || 'No se pudo emitir el avance.';
  }
}
