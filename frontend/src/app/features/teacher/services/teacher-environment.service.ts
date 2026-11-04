import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, map } from 'rxjs';

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
  environment_id: string;
  exercise_id?: string | null;
  subject_id?: string;
  template_id?: string | null;
  status: string;
  access_url: string;
  volume_name?: string;
}

/**
 * Apertura del IDE de una práctica del docente. El entorno pertenece siempre a
 * una práctica (ADR-059): ya no hay entorno por materia que cargar ni pausar.
 */
@Injectable({
  providedIn: 'root'
})
export class TeacherEnvironmentService {
  private readonly http = inject(HttpClient);

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
}
