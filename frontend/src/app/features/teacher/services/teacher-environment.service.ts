import { Injectable, inject, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, catchError, finalize, map, tap, throwError } from 'rxjs';
import { WorkspaceInstance } from '@core/models/workspace.model';

interface ApiResponse<T> {
  data: T;
  error?: string;
  message?: string;
}

/**
 * Entorno de una práctica concreta del docente: su volumen, su plantilla y la
 * URL para abrirlo en una pestaña nueva.
 */
export interface PracticeIDESession {
  workspace_id: string;
  exercise_id?: string | null;
  subject_id?: string;
  template_id?: string | null;
  status: string;
  access_url: string;
  volume_name?: string;
}

/**
 * Entorno de clase del docente en una materia. Es el mismo entorno desde el que
 * se emite un avance (ADR-058 §5): lo que el docente escribe acá es lo que se
 * copia al aula, así que la pantalla lo lee antes de ofrecer abrirlo. Sin
 * lectura previa, cargar la vista levantaría un contenedor que nadie pidió.
 */
@Injectable({
  providedIn: 'root'
})
export class TeacherEnvironmentService {
  private readonly http = inject(HttpClient);

  readonly environment = signal<WorkspaceInstance | null>(null);
  readonly isLoading = signal<boolean>(false);
  readonly isStarting = signal<boolean>(false);
  readonly error = signal<string | null>(null);

  /** Entorno vigente del docente en la materia, o nulo si todavía no abrió uno. */
  load(subjectId: string): Observable<WorkspaceInstance | null> {
    this.isLoading.set(true);
    this.environment.set(null);
    this.error.set(null);
    return this.http.get<ApiResponse<WorkspaceInstance | null>>(`/api/v1/workspaces?subject_id=${subjectId}`).pipe(
      map(res => this.normalize(res.data ?? null)),
      tap(environment => this.environment.set(environment)),
      catchError(err => {
        this.error.set(this.describe(err));
        return throwError(() => err);
      }),
      finalize(() => this.isLoading.set(false))
    );
  }

  /**
   * Abre el entorno de la materia. Es idempotente: con un entorno corriendo
   * devuelve el mismo, y con uno en pausa lo reanuda.
   */
  open(subjectId: string): Observable<WorkspaceInstance> {
    this.isStarting.set(true);
    this.error.set(null);
    return this.http.post<ApiResponse<WorkspaceInstance>>('/api/v1/workspaces/start', { subject_id: subjectId }).pipe(
      map(res => this.normalize(res.data) as WorkspaceInstance),
      tap(environment => this.environment.set(environment)),
      catchError(err => {
        this.error.set(this.describe(err));
        return throwError(() => err);
      }),
      finalize(() => this.isStarting.set(false))
    );
  }

  /**
   * Abre el IDE de una práctica. Levanta el contenedor con el volumen y la
   * plantilla de esa práctica y devuelve la URL de acceso; el volumen es propio
   * por ejercicio, así que el código de una práctica no aparece en otra.
   */
  openPracticeIDE(courseId: string, exerciseId: string): Observable<PracticeIDESession> {
    return this.http
      .post<ApiResponse<PracticeIDESession>>(
        `/api/v1/teacher/courses/${courseId}/exercises/${exerciseId}/ide`,
        {}
      )
      .pipe(map(res => res.data));
  }

  /** Mensaje del backend para un fallo al abrir el IDE de una práctica. */
  describePracticeIDEError(err: unknown): string {
    const typed = err as { error?: { message?: string; error?: string }; message?: string };
    return (
      typed?.error?.message ||
      typed?.error?.error ||
      typed?.message ||
      'No se pudo abrir el IDE de la práctica.'
    );
  }

  /** Pausa el entorno sin perder el volumen: el código del docente queda guardado. */
  pause(workspaceId: string): Observable<WorkspaceInstance> {
    this.isStarting.set(true);
    this.error.set(null);
    return this.http.post<ApiResponse<WorkspaceInstance>>(`/api/v1/workspaces/${workspaceId}/pause`, {}).pipe(
      map(res => this.normalize(res.data) as WorkspaceInstance),
      tap(environment => this.environment.set(environment)),
      catchError(err => {
        this.error.set(this.describe(err));
        return throwError(() => err);
      }),
      finalize(() => this.isStarting.set(false))
    );
  }

  /**
   * El backend devuelve el identificador como `workspace_id`; el modelo de la
   * vista lo lee como `id`. Se normaliza una vez, acá.
   */
  private normalize(raw: WorkspaceInstance | null): WorkspaceInstance | null {
    if (!raw) {
      return null;
    }
    return {
      ...raw,
      id: raw.id || raw.workspace_id || '',
      last_activity_at: raw.last_activity_at || raw.last_heartbeat_at
    };
  }

  private describe(err: unknown): string {
    const typed = err as { error?: { error?: string; message?: string }; message?: string };
    return typed?.error?.error || typed?.error?.message || typed?.message || 'No se pudo actualizar el entorno de clase.';
  }
}
