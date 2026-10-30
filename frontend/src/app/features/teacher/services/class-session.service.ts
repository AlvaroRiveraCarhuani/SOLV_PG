import { Injectable, inject, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, catchError, finalize, map, tap, throwError } from 'rxjs';
import { ClassSession, ClassSessionBoard } from '../models/teacher.models';

interface ApiResponse<T> {
  data: T;
  error?: string;
  message?: string;
}

/**
 * Aula de la clase: abrir, cerrar y leer el tablero. El servidor resuelve los
 * conteos y el orden de los alumnos; el servicio solo los guarda.
 */
@Injectable({
  providedIn: 'root'
})
export class ClassSessionService {
  private readonly http = inject(HttpClient);

  readonly session = signal<ClassSession | null>(null);
  readonly board = signal<ClassSessionBoard | null>(null);
  readonly isLoading = signal<boolean>(false);
  readonly isOpening = signal<boolean>(false);
  readonly error = signal<string | null>(null);
  /** Hora en que el aula dejó el modo en vivo sin que el docente la cerrara. */
  readonly lastClosedAt = signal<string | null>(null);

  /** Aula abierta del docente, o nula. Un error de red no borra lo que ya se ve. */
  loadCurrent(): Observable<ClassSession | null> {
    return this.http.get<ApiResponse<ClassSession | null>>('/api/v1/teacher/sessions/open').pipe(
      map(res => res.data ?? null),
      tap(session => this.session.set(session)),
      catchError(err => {
        this.error.set(this.describe(err));
        return throwError(() => err);
      })
    );
  }

  /**
   * Tablero del aula: marcador y alumnos ordenados por urgencia docente. Si la
   * sesión ya venció, se devuelve cerrada y la pantalla deja el modo en vivo.
   */
  loadBoard(sessionId: string): Observable<ClassSessionBoard> {
    this.isLoading.set(true);
    return this.http.get<ApiResponse<ClassSessionBoard>>(`/api/v1/teacher/sessions/${sessionId}/board`).pipe(
      map(res => res.data),
      tap(board => {
        this.error.set(null);
        if (board.session.status !== 'open') {
          this.lastClosedAt.set(board.session.closes_at ?? board.session.ends_at);
          this.board.set(null);
          this.session.set(null);
          return;
        }
        this.lastClosedAt.set(null);
        this.board.set(board);
        this.session.set(board.session);
      }),
      catchError(err => {
        this.error.set(this.describe(err));
        return throwError(() => err);
      }),
      finalize(() => this.isLoading.set(false))
    );
  }

  /** Abre el aula de una materia, opcionalmente atada a una práctica y con su duración. */
  openAula(subjectId: string, exerciseId?: string | null, durationMinutes?: number | null): Observable<ClassSession> {
    this.isOpening.set(true);
    const payload: { subject_id: string; exercise_id?: string; duration_minutes?: number } = {
      subject_id: subjectId
    };
    if (exerciseId) {
      payload.exercise_id = exerciseId;
    }
    if (durationMinutes) {
      payload.duration_minutes = durationMinutes;
    }
    return this.http.post<ApiResponse<ClassSession>>('/api/v1/teacher/sessions', payload).pipe(
      map(res => res.data),
      tap(session => {
        this.session.set(session);
        this.lastClosedAt.set(null);
        this.error.set(null);
      }),
      catchError(err => {
        this.error.set(this.describe(err));
        return throwError(() => err);
      }),
      finalize(() => this.isOpening.set(false))
    );
  }

  /** Cierra el aula. Cerrar dos veces devuelve el mismo estado. */
  closeAula(sessionId: string): Observable<ClassSession> {
    return this.http.patch<ApiResponse<ClassSession>>(`/api/v1/teacher/sessions/${sessionId}/close`, {}).pipe(
      map(res => res.data),
      tap(() => {
        this.session.set(null);
        this.board.set(null);
        // Cierre a mano: no hay que anunciar que el aula se cerró sola.
        this.lastClosedAt.set(null);
        this.error.set(null);
      }),
      catchError(err => {
        this.error.set(this.describe(err));
        return throwError(() => err);
      })
    );
  }

  clearError(): void {
    this.error.set(null);
  }

  /** El backend devuelve el motivo legible en `error`; se usa ese, no un texto genérico. */
  private describe(err: unknown): string {
    const typed = err as { error?: { error?: string; message?: string }; message?: string };
    return typed?.error?.error || typed?.error?.message || typed?.message || 'No se pudo actualizar el aula.';
  }
}
