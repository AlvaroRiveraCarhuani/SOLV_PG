import { Injectable, computed, inject, signal } from '@angular/core';
import { ProductTourService } from './product-tour.service';
import { markHelpGuideSeen, isHelpGuideSeen } from './help-preferences';
import { TutorialDefinition, TutorialStepDefinition } from './help.types';

/**
 * Tutoriales interactivos controlados por Angular.
 *
 * Cada paso puede declarar una comprobación del estado real de la pantalla:
 * el botón de avance permanece deshabilitado hasta que esa condición se
 * cumple de verdad. Driver.js sólo resalta el elemento del paso actual.
 */
@Injectable({ providedIn: 'root' })
export class InteractiveTutorialService {
  private readonly tour = inject(ProductTourService);

  private readonly definitionSignal = signal<TutorialDefinition | null>(null);
  private readonly indexSignal = signal<number>(0);
  private returnFocusTo: HTMLElement | null = null;

  readonly definition = this.definitionSignal.asReadonly();
  readonly index = this.indexSignal.asReadonly();
  readonly isActive = computed<boolean>(() => this.definitionSignal() !== null);

  readonly stepCount = computed<number>(() => this.definitionSignal()?.steps.length ?? 0);

  readonly currentStep = computed<TutorialStepDefinition | null>(() => {
    const definition = this.definitionSignal();
    if (!definition) return null;
    return definition.steps[this.indexSignal()] ?? null;
  });

  readonly isFirstStep = computed<boolean>(() => this.indexSignal() === 0);

  readonly isLastStep = computed<boolean>(() => this.indexSignal() >= this.stepCount() - 1);

  /** Lee el estado real de la pantalla a través de la comprobación del paso. */
  readonly currentCheckPassed = computed<boolean>(() => {
    const check = this.currentStep()?.check;
    return check ? check.isComplete() : true;
  });

  hasSeen(id: string): boolean {
    return isHelpGuideSeen('tutorial', id);
  }

  start(definition: TutorialDefinition): void {
    if (definition.steps.length === 0) return;
    this.returnFocusTo = typeof document !== 'undefined' ? (document.activeElement as HTMLElement) : null;
    this.definitionSignal.set(definition);
    this.indexSignal.set(0);
    this.syncHighlight();
  }

  next(): void {
    if (!this.isActive() || !this.currentCheckPassed()) return;

    if (this.isLastStep()) {
      this.complete();
      return;
    }

    this.indexSignal.update((index) => index + 1);
    this.syncHighlight();
  }

  previous(): void {
    if (!this.isActive() || this.isFirstStep()) return;
    this.indexSignal.update((index) => index - 1);
    this.syncHighlight();
  }

  exit(): void {
    this.definitionSignal.set(null);
    this.indexSignal.set(0);
    this.tour.destroy();
    const target = this.returnFocusTo;
    this.returnFocusTo = null;
    if (target) setTimeout(() => target.focus(), 0);
  }

  private complete(): void {
    const id = this.definitionSignal()?.id;
    if (id) {
      markHelpGuideSeen('tutorial', id);
    }
    this.exit();
  }

  private syncHighlight(): void {
    const step = this.currentStep();
    if (!step?.selector) {
      this.tour.destroy();
      return;
    }
    void this.tour.highlight({
      selector: step.selector,
      title: step.title,
      description: step.body,
      side: step.side
    });
  }
}
