import { Injectable } from '@angular/core';

/**
 * Borradores de los formularios que solo existen mientras su modal está
 * montado.
 *
 * Comparte el criterio de {@link AiOperations}: el alcance lo decide el
 * destinatario del resultado. El Copiloto del Studio y la solicitud del docente
 * ya tienen su propio store porque su resultado sobrevive a la vista; acá viven
 * los formularios de modal, que se destruyen al cerrar y volverían vacíos con
 * una llamada de IA todavía en curso o recién terminada.
 *
 * Cada flujo captura los campos que el motor puede sobrescribir, no el catálogo:
 * las listas se releen del backend.
 */
@Injectable({
  providedIn: 'root'
})
export class AiFlowDrafts {
  private readonly drafts = new Map<string, unknown>();

  get<T>(key: string): T | null {
    return (this.drafts.get(key) as T | undefined) ?? null;
  }

  save<T>(key: string, draft: T): void {
    this.drafts.set(key, draft);
  }

  clear(key: string): void {
    this.drafts.delete(key);
  }
}
