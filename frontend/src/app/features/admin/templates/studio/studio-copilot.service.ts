import { Injectable, inject, signal } from '@angular/core';
import {
  AdminConfigIaService,
  AISuggestTemplateResponse
} from '../../configuracion/admin-config-ia.service';
import { TargetEnvironment } from '../../services/admin-templates.service';
import { AiOperation, AiOperations } from '@shared/services/ai-operation';

/**
 * Borrador del Studio que sobrevive a la navegación. Solo se guardan los datos
 * del formulario; las listas del catálogo se recargan desde el backend.
 */
export interface StudioDraft {
  name: string;
  description: string;
  selectedCategoryId: string | null;
  targetEnvironment: TargetEnvironment;
  studioMode: 'visual' | 'code';
  blueprintTab: 'human' | 'manifest';
  selectedRuntimeId: string | null;
  selectedWebPlatformId: string | null;
  selectedAddonIds: string[];
  selectedDatabaseId: string;
  dockerImage: string;
  dockerfileContent: string;
  hasCustomDockerfile: boolean;
  needsLocalBuild: boolean;
  setupScript: string;
  toolsDeclared: string;
  entrypoint: string;
  timeoutMS: number;
  sampleInput: string;
  baseRamMB: number;
  isNameManuallyEdited: boolean;
  isRamManuallyEdited: boolean;
}

const SUGGEST_TEMPLATE_KEY = 'suggest-template';

/**
 * Estado del Copiloto IA del Studio de Plantillas.
 *
 * La llamada al modelo vive en una operación con dueño en la raíz
 * ({@link AiOperations}), así que cerrar el modal o navegar a otra vista ya no
 * la aborta. Acá queda además el borrador del constructor, que también
 * sobrevive a la navegación.
 */
@Injectable({
  providedIn: 'root'
})
export class StudioCopilotService {
  private readonly aiService = inject(AdminConfigIaService);
  private readonly aiOperations = inject(AiOperations);

  readonly prompt = signal<string>('');
  readonly applied = signal<boolean>(false);
  readonly draft = signal<StudioDraft | null>(null);

  private readonly operation: AiOperation<AISuggestTemplateResponse> = this.aiOperations.get(
    SUGGEST_TEMPLATE_KEY,
    {
      resolveError: (err: unknown) => this.aiService.resolveError(err),
      notify: {
        success: 'Propuesta del Copiloto lista para revisar.',
        cancel: 'Operación del Copiloto cancelada.'
      }
    }
  );

  readonly isSuggesting = this.operation.isRunning;
  readonly proposal = this.operation.result;
  readonly elapsedSeconds = this.operation.elapsedSeconds;

  suggest(targetEnvironment: TargetEnvironment): void {
    const prompt = this.prompt().trim();
    if (!prompt) return;

    this.applied.set(false);
    this.operation.run(
      this.aiService.suggestTemplate({ prompt, target_environment: targetEnvironment })
    );
  }

  /** Cancela la operación en vuelo. Aborta el request y limpia el estado. */
  cancel(): void {
    this.operation.cancel();
  }

  discard(): void {
    this.proposal.set(null);
    this.applied.set(false);
  }

  markApplied(): void {
    this.applied.set(true);
  }

  saveDraft(draft: StudioDraft): void {
    this.draft.set(draft);
  }

  clearDraft(): void {
    this.draft.set(null);
  }
}
