/**
 * Tipos del sistema de ayuda del administrador.
 *
 * Tres mecanismos distintos comparten estos tipos:
 *  - Ayuda contextual: entradas que se muestran en el panel lateral.
 *  - Recorrido visual: pasos que Driver.js dibuja sobre elementos reales.
 *  - Tutorial interactivo: pasos con una comprobación del estado real de la
 *    pantalla; la progresión la decide Angular, nunca la librería.
 */

export type HelpSide = 'top' | 'bottom' | 'left' | 'right';

export interface HelpItem {
  label: string;
  text: string;
}

export interface ContextHelpEntry {
  key: string;
  title: string;
  intro: string;
  items: HelpItem[];
  /** Ancla del manual administrativo; null cuando no hay sección equivalente. */
  manualFragment: string | null;
}

/** Acción que el centro de ayuda puede lanzar desde el panel. */
export interface HelpGuideAction {
  label: string;
  /** Línea breve que explica qué hace la guía. */
  hint?: string;
  run: () => void;
}

/** Guías disponibles en la pantalla registrada; las ausentes no se muestran. */
export interface HelpGuideSet {
  tour?: HelpGuideAction;
  tutorial?: HelpGuideAction;
}

export interface TourStepDefinition {
  /** Selector CSS de un elemento real de la pantalla. */
  selector: string;
  title: string;
  description: string;
  side?: HelpSide;
}

export interface TourDefinition {
  id: string;
  title: string;
  steps: TourStepDefinition[];
}

export interface TutorialStepCheck {
  /** Comprueba el estado real; nunca se simula el avance. */
  isComplete: () => boolean;
  pendingHint: string;
}

export interface TutorialStepDefinition {
  id: string;
  title: string;
  body: string;
  selector?: string;
  side?: HelpSide;
  check?: TutorialStepCheck;
}

export interface TutorialDefinition {
  id: string;
  title: string;
  steps: TutorialStepDefinition[];
  manualFragment: string | null;
}
