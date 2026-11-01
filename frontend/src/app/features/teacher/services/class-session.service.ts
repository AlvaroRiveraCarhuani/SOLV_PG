import { Injectable, inject, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, catchError, finalize, map, of, tap, throwError } from 'rxjs';
import { ClassHistoryEvent, ClassSession, ClassSessionBoard, FreezeReport } from '../models/teacher.models';

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
  readonly isChangingFreeze = signal<boolean>(false);
  readonly isExtendingGrace = signal<boolean>(false);
  /** Conteo del último congelamiento de examen: cierre, reintento o excepción. */
  readonly lastFreezeReport = signal<FreezeReport | null>(null);
  readonly isFreezingExam = signal<boolean>(false);
  /**
   * Tablero de un examen ya cerrado. La lectura del aula se reserva a la sesión
   * abierta; esta copia conserva el estado de cada entorno para reabrir a un
   * alumno sin volver a abrir la clase.
   */
  readonly closedBoard = signal<ClassSessionBoard | null>(null);
  readonly isLoadingClosedBoard = signal<boolean>(false);
  /** Hora en que el aula dejó el modo en vivo sin que el docente la cerrara. */
  readonly lastClosedAt = signal<string | null>(null);
  /**
   * Última aula cerrada que esta pantalla conoce. Extender la ventana de rezago
   * exige una sesión cerrada y su identificador: sin esto, cerrar el aula lo
   * perdería y la ventana quedaría fuera de alcance.
   */
  readonly lastClosedSession = signal<ClassSession | null>(null);
  /** Movimientos del aula: emisiones, congelaciones, ventana y decisiones. */
  readonly history = signal<ClassHistoryEvent[]>([]);
  readonly isLoadingHistory = signal<boolean>(false);
  readonly historyError = signal<string | null>(null);

  /** Aula abierta del docente, o nula. Un error de red no borra lo que ya se ve. */
  loadCurrent(): Observable<ClassSession | null> {
    return this.http.get<ApiResponse<ClassSession | null>>('/api/v1/teacher/sessions/open').pipe(
      map(res => res.data ?? null),
      tap(session => {
        this.session.set(session);
        if (session) {
          // Una clase nueva no hereda el conteo del examen anterior.
          this.lastFreezeReport.set(null);
          this.closedBoard.set(null);
        }
      }),
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
          this.lastClosedSession.set(board.session);
          this.board.set(null);
          this.session.set(null);
          return;
        }
        this.lastClosedAt.set(null);
        this.lastClosedSession.set(null);
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

  /**
   * Historial de la clase, del más nuevo al más viejo. Un error no vacía la
   * lista anterior: se conserva lo último que el docente vio.
   */
  loadHistory(sessionId: string): Observable<ClassHistoryEvent[]> {
    this.isLoadingHistory.set(true);
    this.historyError.set(null);
    return this.http.get<ApiResponse<ClassHistoryEvent[]>>(`/api/v1/teacher/sessions/${sessionId}/history`).pipe(
      map(res => res.data ?? []),
      tap(events => this.history.set(events)),
      catchError(err => {
        this.historyError.set(this.describe(err));
        return of(this.history());
      }),
      finalize(() => this.isLoadingHistory.set(false))
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
        this.lastClosedSession.set(null);
        this.lastFreezeReport.set(null);
        this.closedBoard.set(null);
        // El historial de la clase anterior no pertenece a la nueva.
        this.history.set([]);
        this.historyError.set(null);
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
    // El cierre de un examen agrega el conteo del congelamiento en la misma
    // respuesta: la pantalla no tiene que preguntar dos veces (ADR-005 §5.5).
    type CloseAulaData = ClassSession & { freeze_report?: FreezeReport };
    return this.http.patch<ApiResponse<CloseAulaData>>(`/api/v1/teacher/sessions/${sessionId}/close`, {}).pipe(
      map(res => res.data),
      tap(session => {
        this.session.set(null);
        this.board.set(null);
        this.closedBoard.set(null);
        this.lastFreezeReport.set(session.freeze_report ?? null);
        // Cierre a mano: no hay que anunciar que el aula se cerró sola.
        this.lastClosedAt.set(null);
        // La sesión cerrada queda a mano para extender su ventana de rezago.
        this.lastClosedSession.set(session);
        this.error.set(null);
      }),
      catchError(err => {
        this.error.set(this.describe(err));
        return throwError(() => err);
      })
    );
  }

  /**
   * Congela o descongela la clase. Congelar corta las emisiones nuevas, no el
   * trabajo del alumno; la sesión devuelta es la fuente de verdad del cambio.
   */
  setFrozen(sessionId: string, frozen: boolean): Observable<ClassSession> {
    this.isChangingFreeze.set(true);
    const action = frozen ? 'freeze' : 'unfreeze';
    return this.http.patch<ApiResponse<ClassSession>>(`/api/v1/teacher/sessions/${sessionId}/${action}`, {}).pipe(
      map(res => res.data),
      tap(session => {
        this.session.set(session);
        this.error.set(null);
      }),
      catchError(err => {
        this.error.set(this.describe(err));
        return throwError(() => err);
      }),
      finalize(() => this.isChangingFreeze.set(false))
    );
  }

  /**
   * Congela los volúmenes de un examen ya cerrado. Repite la operación: los que
   * ya estaban en solo lectura cuentan como omitidos.
   */
  freezeExam(sessionId: string): Observable<FreezeReport> {
    this.isFreezingExam.set(true);
    return this.http.post<ApiResponse<FreezeReport>>(`/api/v1/teacher/sessions/${sessionId}/exam-freeze`, {}).pipe(
      map(res => res.data),
      tap(report => {
        this.lastFreezeReport.set(report);
        this.error.set(null);
      }),
      catchError(err => {
        this.error.set(this.describe(err));
        return throwError(() => err);
      }),
      finalize(() => this.isFreezingExam.set(false))
    );
  }

  /**
   * Devuelve la escritura al entorno de un alumno del examen. Exige motivo: la
   * excepción queda en la auditoría de la clase.
   */
  unfreezeStudent(sessionId: string, studentId: string, reason: string): Observable<FreezeReport> {
    return this.http
      .post<ApiResponse<FreezeReport>>(`/api/v1/teacher/sessions/${sessionId}/students/${studentId}/unfreeze`, { reason })
      .pipe(
        map(res => res.data),
        tap(() => this.error.set(null)),
        catchError(err => {
          this.error.set(this.describe(err));
          return throwError(() => err);
        })
      );
  }

  /**
   * Tablero de un aula cerrada. El tablero en vivo se limpia al cerrar, así que
   * esta lectura conserva las filas y el estado de cada entorno.
   */
  loadClosedBoard(sessionId: string): Observable<ClassSessionBoard> {
    this.isLoadingClosedBoard.set(true);
    return this.http.get<ApiResponse<ClassSessionBoard>>(`/api/v1/teacher/sessions/${sessionId}/board`).pipe(
      map(res => res.data),
      tap(board => this.closedBoard.set(board)),
      catchError(err => {
        this.error.set(this.describe(err));
        return throwError(() => err);
      }),
      finalize(() => this.isLoadingClosedBoard.set(false))
    );
  }

  /**
   * Suma minutos al final vigente de la ventana de disponibilidad. Si la
   * ventana ya venció, el servidor cuenta desde ahora y reabre los vencidos.
   */
  addGrace(sessionId: string, minutes: number): Observable<ClassSession> {
    return this.updateGrace(sessionId, { add_minutes: minutes });
  }

  /**
   * Fija una fecha límite concreta para una tarea. El servidor exige que sea
   * futura y que no supere el máximo desde el cierre de la clase.
   */
  setGraceDeadline(sessionId: string, until: string): Observable<ClassSession> {
    return this.updateGrace(sessionId, { until });
  }

  /** Cierra la ventana en el momento: los pendientes pasan a vencidos. */
  closeGrace(sessionId: string): Observable<ClassSession> {
    return this.updateGrace(sessionId, { close: true });
  }

  private updateGrace(
    sessionId: string,
    payload: { add_minutes: number } | { until: string } | { close: true }
  ): Observable<ClassSession> {
    this.isExtendingGrace.set(true);
    return this.http.patch<ApiResponse<ClassSession>>(`/api/v1/teacher/sessions/${sessionId}/grace`, payload).pipe(
      map(res => res.data),
      tap(session => {
        this.lastClosedSession.set(session);
        if (this.session()?.id === session.id) {
          this.session.set(session);
        }
        this.error.set(null);
      }),
      catchError(err => {
        this.error.set(this.describe(err));
        return throwError(() => err);
      }),
      finalize(() => this.isExtendingGrace.set(false))
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
