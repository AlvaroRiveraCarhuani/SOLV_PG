import { Injectable, signal } from '@angular/core';
import type { Config, Driver } from 'driver.js';
import { markHelpGuideSeen, isHelpGuideSeen } from './help-preferences';
import { TourDefinition, TourStepDefinition } from './help.types';

/**
 * Recorridos visuales con Driver.js.
 *
 * La librería sólo resalta elementos reales y muestra el texto de cada paso.
 * La decisión de avanzar, la progresión y el estado de un tutorial viven en
 * Angular (ver InteractiveTutorialService).
 */

const BASE_CONFIG: Config = {
  showProgress: true,
  progressText: 'Paso {{current}} de {{total}}',
  nextBtnText: 'Siguiente',
  prevBtnText: 'Atrás',
  doneBtnText: 'Entendido',
  allowClose: true,
  overlayClickBehavior: 'close',
  smoothScroll: true,
  stagePadding: 6,
  stageRadius: 10,
  popoverClass: 'solv-help-popover',
  waitForElement: 2500
};

@Injectable({ providedIn: 'root' })
export class ProductTourService {
  /** Indica si hay un recorrido con botones activo. */
  readonly isRunning = signal<boolean>(false);

  private instance: Driver | null = null;
  private returnFocusTo: HTMLElement | null = null;
  private escapeHandler: ((event: KeyboardEvent) => void) | null = null;

  hasSeen(id: string): boolean {
    return isHelpGuideSeen('tour', id);
  }

  /**
   * Inicia un recorrido. Devuelve false cuando ninguno de sus pasos existe en
   * la pantalla actual, para no abrir un tour vacío.
   */
  async start(tour: TourDefinition): Promise<boolean> {
    const available = tour.steps.filter((step) => this.resolve(step.selector) !== null);
    if (available.length === 0) return false;

    this.destroy();
    this.returnFocusTo = typeof document !== 'undefined' ? (document.activeElement as HTMLElement) : null;
    const { driver } = await import('driver.js');
    const instance = driver({
      ...BASE_CONFIG,
      steps: available.map((step) => ({
        element: step.selector,
        popover: {
          title: step.title,
          description: step.description,
          side: step.side ?? 'bottom'
        },
        // El layout administrativo desplaza el contenido dentro de
        // .shell-content, no en la ventana. Centrar el objetivo aquí evita
        // que un paso señale un elemento que quedó fuera de la vista.
        onHighlightStarted: (element?: Element) => this.centerElement(element)
      })),
      onDestroyed: () => {
        this.instance = null;
        this.isRunning.set(false);
        this.unbindEscape();
        markHelpGuideSeen('tour', tour.id);
        this.restoreFocus();
      }
    });

    this.instance = instance;
    this.isRunning.set(true);
    this.bindEscape();
    this.centerElement(this.resolve(available[0].selector));
    instance.drive();
    return true;
  }

  /**
   * Resalta un elemento para el tutorial. Sin popover: el texto y los botones
   * los pinta Angular, no Driver.js.
   */
  async highlight(step: TourStepDefinition | null): Promise<void> {
    if (!step?.selector) {
      this.destroy();
      return;
    }

    const element = this.resolve(step.selector);
    if (!element) {
      this.destroy();
      return;
    }

    const instance = this.instance ?? (await this.createHighlightInstance());
    this.centerElement(element);
    instance.highlight({ element });
  }

  destroy(): void {
    this.unbindEscape();
    this.instance?.destroy();
    this.instance = null;
    this.isRunning.set(false);
    this.restoreFocus();
  }

  /** El recorrido puede abrirse desde el centro de ayuda, con el foco fuera
   * del popover; Escape debe cerrarlo igualmente. */
  private bindEscape(): void {
    if (typeof document === 'undefined' || this.escapeHandler) return;
    this.escapeHandler = (event: KeyboardEvent) => {
      if (event.key === 'Escape' && this.instance) {
        this.instance.destroy();
      }
    };
    document.addEventListener('keydown', this.escapeHandler);
  }

  private unbindEscape(): void {
    if (typeof document === 'undefined' || !this.escapeHandler) return;
    document.removeEventListener('keydown', this.escapeHandler);
    this.escapeHandler = null;
  }

  private restoreFocus(): void {
    const target = this.returnFocusTo;
    this.returnFocusTo = null;
    if (target) setTimeout(() => target.focus(), 0);
  }

  private async createHighlightInstance(): Promise<Driver> {
    const { driver } = await import('driver.js');
    const instance = driver({
      ...BASE_CONFIG,
      showProgress: false,
      allowClose: false,
      overlayClickBehavior: 'none',
      disableActiveInteraction: false,
      steps: [],
      onDestroyed: () => {
        this.instance = null;
      }
    });
    this.instance = instance;
    return instance;
  }

  private resolve(selector: string): Element | null {
    if (typeof document === 'undefined') return null;
    return document.querySelector(selector);
  }

  /** Centra el objetivo en su contenedor con scroll; sin animación para que
   * Driver.js mida la posición definitiva al dibujar el resaltado. */
  private centerElement(element: Element | null | undefined): void {
    if (!element || typeof window === 'undefined') return;
    const rect = element.getBoundingClientRect();
    const fullyVisible = rect.height <= window.innerHeight && rect.top >= 0 && rect.bottom <= window.innerHeight;
    if (fullyVisible) return;
    element.scrollIntoView({ block: 'center', inline: 'nearest', behavior: 'auto' });
  }
}
