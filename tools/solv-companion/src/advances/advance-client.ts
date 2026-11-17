import { LONG_REQUEST_TIMEOUT_MS, SolvClient } from '../client/solv-client';
import {
  AdvanceApplyResult,
  AdvanceDelivery,
  AdvancePreview,
  AdvanceRollbackInventory,
  AdvanceRollbackResult,
  AdvanceScope,
  EmittedAdvance
} from '../domain/advance';

/** Resumen del aula que ya resuelve el backend para el docente. */
export interface SessionBoard {
  session: {
    id: string;
    status: string;
    frozen_at?: string | null;
    purpose?: string;
    ends_at?: string | null;
    opens_at?: string | null;
  };
  summary: {
    enrolled: number;
    present: number;
    submitted: number;
  };
  students: Array<{
    student_id: string;
    name: string;
    present: boolean;
  }>;
}

export interface EmitAdvanceInput {
  sessionId: string;
  exerciseId: string;
  scope: AdvanceScope;
  studentIds?: string[];
  /** Presente = solo esas rutas del entorno. Vacío u omitido = todo el entorno. */
  selectedFiles?: string[];
}

/**
 * Cliente de los avances del aula. Todas las rutas usan la sesión del editor:
 * el alumno solo ve lo dirigido a él y el docente solo su propia cátedra.
 */
export class AdvanceClient {
  constructor(private readonly client: SolvClient) {}

  pending(): Promise<AdvanceDelivery[]> {
    return this.client.request<AdvanceDelivery[]>('/api/v1/class-advances/pending', {
      method: 'GET'
    });
  }

  preview(advanceId: string): Promise<AdvancePreview> {
    return this.client.request<AdvancePreview>(
      `/api/v1/class-advances/${encodeURIComponent(advanceId)}/files`,
      { method: 'GET' }
    );
  }

  apply(advanceId: string, selectedFiles: string[]): Promise<AdvanceApplyResult> {
    return this.client.request<AdvanceApplyResult>(
      `/api/v1/class-advances/${encodeURIComponent(advanceId)}/apply`,
      { method: 'POST', body: JSON.stringify({ selected_files: selectedFiles }) },
      LONG_REQUEST_TIMEOUT_MS
    );
  }

  dismiss(advanceId: string): Promise<unknown> {
    return this.client.request<unknown>(
      `/api/v1/class-advances/${encodeURIComponent(advanceId)}/dismiss`,
      { method: 'POST', body: JSON.stringify({}) }
    );
  }

  rollback(advanceId: string, paths: string[] = []): Promise<AdvanceRollbackResult> {
    return this.client.request<AdvanceRollbackResult>(
      `/api/v1/class-advances/${encodeURIComponent(advanceId)}/rollback`,
      { method: 'POST', body: JSON.stringify({ paths }) },
      LONG_REQUEST_TIMEOUT_MS
    );
  }

  /** Archivos del respaldo que el alumno todavía puede deshacer. */
  rollbackFiles(advanceId: string): Promise<AdvanceRollbackInventory> {
    return this.client.request<AdvanceRollbackInventory>(
      `/api/v1/class-advances/${encodeURIComponent(advanceId)}/rollback`,
      { method: 'GET' }
    );
  }

  board(sessionId: string): Promise<SessionBoard | null> {
    return this.client.request<SessionBoard | null>(
      `/api/v1/teacher/sessions/${encodeURIComponent(sessionId)}/board`,
      { method: 'GET' }
    );
  }

  async emit(input: EmitAdvanceInput): Promise<EmittedAdvance> {
    return this.client.request<EmittedAdvance>(
      `/api/v1/teacher/sessions/${encodeURIComponent(input.sessionId)}/advances`,
      {
        method: 'POST',
        body: JSON.stringify({
          exercise_id: input.exerciseId,
          scope: input.scope,
          student_ids: input.studentIds ?? [],
          selected_files: input.selectedFiles ?? []
        })
      },
      LONG_REQUEST_TIMEOUT_MS
    );
  }

  setFrozen(sessionId: string, frozen: boolean): Promise<unknown> {
    const action = frozen ? 'freeze' : 'unfreeze';
    return this.client.request<unknown>(
      `/api/v1/teacher/sessions/${encodeURIComponent(sessionId)}/${action}`,
      { method: 'PATCH', body: JSON.stringify({}) }
    );
  }

  currentOpenSession(): Promise<{ id: string; status: string; subject_id: string } | null> {
    return this.client.currentTeacherSession();
  }

  /**
   * Abre el aula. Con `exerciseId` el aula queda atada a esa práctica, que es lo
   * que habilita emitir avances: una sesión sin práctica responde 409.
   */
  openSession(subjectId: string, durationMinutes = 90, exerciseId?: string): Promise<{ id: string }> {
    const payload: { subject_id: string; duration_minutes: number; exercise_id?: string } = {
      subject_id: subjectId,
      duration_minutes: durationMinutes
    };
    if (exerciseId) {
      payload.exercise_id = exerciseId;
    }
    return this.client.request<{ id: string }>('/api/v1/teacher/sessions', {
      method: 'POST',
      body: JSON.stringify(payload)
    });
  }

  closeSession(sessionId: string): Promise<unknown> {
    return this.client.request<unknown>(
      `/api/v1/teacher/sessions/${encodeURIComponent(sessionId)}/close`,
      { method: 'PATCH', body: JSON.stringify({}) }
    );
  }
}
