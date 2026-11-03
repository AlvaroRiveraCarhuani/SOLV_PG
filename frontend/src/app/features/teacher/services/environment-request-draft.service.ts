import { Injectable, signal } from '@angular/core';
import { EnvironmentMissingComponent, EnvironmentRequestUse } from '../templates/environment-request.models';

/**
 * Borrador de la solicitud de entorno.
 *
 * Vive en la raíz, igual que el borrador del estudio del administrador: cerrar
 * el modal o navegar a otra vista no pierde lo que el docente escribió. Solo se
 * guardan los datos del formulario; el catálogo se relee del backend.
 */
export interface EnvironmentRequestDraft {
  description: string;
  title: string;
  titleTouched: boolean;
  subjectId: string;
  subjectName: string;
  uses: EnvironmentRequestUse[];
  runtimeId: string;
  webPlatformId: string;
  databaseId: string;
  addonIds: string[];
  missingFromAI: EnvironmentMissingComponent[];
  aiNotice: string | null;
}

const EMPTY_DRAFT: EnvironmentRequestDraft = {
  description: '',
  title: '',
  titleTouched: false,
  subjectId: '',
  subjectName: '',
  uses: ['practice'],
  runtimeId: '',
  webPlatformId: '',
  databaseId: '',
  addonIds: [],
  missingFromAI: [],
  aiNotice: null
};

@Injectable({
  providedIn: 'root'
})
export class EnvironmentRequestDraftService {
  private readonly _draft = signal<EnvironmentRequestDraft>({ ...EMPTY_DRAFT });
  readonly draft = this._draft.asReadonly();

  patch(partial: Partial<EnvironmentRequestDraft>): void {
    this._draft.update(current => ({ ...current, ...partial }));
  }

  /** Se llama cuando la solicitud ya salió: el borrador cumplió su función. */
  clear(): void {
    this._draft.set({ ...EMPTY_DRAFT });
  }

  hasContent(): boolean {
    const draft = this._draft();
    return (
      draft.description.trim().length > 0 ||
      draft.addonIds.length > 0 ||
      Boolean(draft.runtimeId || draft.webPlatformId || draft.databaseId)
    );
  }
}
