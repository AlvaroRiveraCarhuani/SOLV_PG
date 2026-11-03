import { SolvClient } from '../client/solv-client';
import {
  AdvanceApplyResult,
  AdvanceDelivery,
  AdvancePreview,
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
      { method: 'POST', body: JSON.stringify({ selected_files: selectedFiles }) }
    );
  }

  dismiss(advanceId: string): Promise<unknown> {
    return this.client.request<unknown>(
      `/api/v1/class-advances/${encodeURIComponent(advanceId)}/dismiss`,
      { method: 'POST', body: JSON.stringify({}) }
    );
  }

  rollback(advanceId: string): Promise<AdvanceRollbackResult> {
    return this.client.request<AdvanceRollbackResult>(
      `/api/v1/class-advances/${encodeURIComponent(advanceId)}/rollback`,
      { method: 'POST', body: JSON.stringify({}) }
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
          student_ids: input.studentIds ?? []
        })
      }
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

  openSession(subjectId: string, durationMinutes = 90): Promise<{ id: string }> {
    return this.client.request<{ id: string }>('/api/v1/teacher/sessions', {
      method: 'POST',
      body: JSON.stringify({
        subject_id: subjectId,
        duration_minutes: durationMinutes
      })
    });
  }

  closeSession(sessionId: string): Promise<unknown> {
    return this.client.request<unknown>(
      `/api/v1/teacher/sessions/${encodeURIComponent(sessionId)}/close`,
      { method: 'PATCH', body: JSON.stringify({}) }
    );
  }
}
