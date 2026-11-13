/**
 * Registro local de guías ya vistas por el administrador.
 *
 * Evita repetir un recorrido en cada visita. Sólo guarda identificadores; no
 * almacena progreso paso a paso ni datos de la plantilla en edición.
 */

const STORAGE_KEY = 'solv_admin_help_seen';

export type HelpGuideKind = 'tour' | 'tutorial';

interface SeenRegistry {
  tours: string[];
  tutorials: string[];
}

function readRegistry(): SeenRegistry {
  const empty: SeenRegistry = { tours: [], tutorials: [] };
  try {
    if (typeof window === 'undefined' || !window.localStorage) return empty;
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (!raw) return empty;
    const parsed = JSON.parse(raw) as Partial<SeenRegistry>;
    return {
      tours: Array.isArray(parsed.tours) ? parsed.tours : [],
      tutorials: Array.isArray(parsed.tutorials) ? parsed.tutorials : []
    };
  } catch {
    return empty;
  }
}

export function isHelpGuideSeen(kind: HelpGuideKind, id: string): boolean {
  const registry = readRegistry();
  return kind === 'tour' ? registry.tours.includes(id) : registry.tutorials.includes(id);
}

export function markHelpGuideSeen(kind: HelpGuideKind, id: string): void {
  try {
    if (typeof window === 'undefined' || !window.localStorage) return;
    const registry = readRegistry();
    const list = kind === 'tour' ? registry.tours : registry.tutorials;
    if (list.includes(id)) return;
    list.push(id);
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(registry));
  } catch {
    // Almacenamiento restringido: la guía se ofrece igual, sólo no recuerda.
  }
}

export function forgetHelpGuides(): void {
  try {
    if (typeof window === 'undefined' || !window.localStorage) return;
    window.localStorage.removeItem(STORAGE_KEY);
  } catch {
    // Sin almacenamiento no hay nada que borrar.
  }
}
