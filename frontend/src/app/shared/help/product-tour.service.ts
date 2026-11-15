import { Injectable, signal } from '@angular/core';
import type { Config, Driver, PopoverDOM } from 'driver.js';
import { markHelpGuideSeen, isHelpGuideSeen } from './help-preferences';
import { TourDefinition, TourStepDefinition } from './help.types';

/**
 * Recorridos visuales con Driver.js.
 *
 * La librería sólo resalta elementos reales y muestra el texto de cada paso.
 * La decisión de avanzar, la progresión y el estado de un tutorial viven en
 * Angular (ver InteractiveTutorialService).
 */

const POPOVER_GAP = 12;
/** Banda mínima para mover el popover fuera del objetivo sin dejarlo ilegible. */
const MIN_POPOVER_BAND = 112;

function overlaps(a: DOMRect, b: DOMRect): boolean {
  return a.left < b.right && a.right > b.left && a.top < b.bottom && a.bottom > b.top;
}

function clamp(value: number, min: number, max: number): number {
  return Math.min(Math.max(value, min), Math.max(min, max));
}

/**
 * Impide que el popover cubra el elemento que explica.
 *
 * Driver.js elige el lado con más espacio, pero en pantallas angostas el
 * objetivo puede ocupar casi toda la altura y el popover termina encima. Aquí
 * se busca la banda libre superior o inferior y, si no alcanza, se limita la
 * altura del popover con desplazamiento interno. Si el objetivo mide más que la
 * ventana no hay banda posible y se conserva la posición de Driver.js.
 */
function keepPopoverClear(popover: PopoverDOM): void {
  if (typeof window === 'undefined') return;

  const wrapper = popover.wrapper;
  const target = document.querySelector('.driver-active-element');

  wrapper.style.removeProperty('max-height');
  wrapper.style.removeProperty('overflow-y');
  popover.arrow.style.removeProperty('display');

  if (!target) return;

  const box = wrapper.getBoundingClientRect();
  const area = target.getBoundingClientRect();
  if (!overlaps(box, area)) return;

  const viewportHeight = window.innerHeight;
  const below = viewportHeight - area.bottom - POPOVER_GAP;
  const above = area.top - POPOVER_GAP;
  const band = Math.max(below, above);
  if (band < MIN_POPOVER_BAND) return;

  let height = box.height;
  if (band < height) {
    height = Math.max(MIN_POPOVER_BAND, band);
    wrapper.style.maxHeight = `${height}px`;
    wrapper.style.overflowY = 'auto';
  }

  const top = below >= above ? area.bottom + POPOVER_GAP : area.top - POPOVER_GAP - height;
  wrapper.style.top = `${clamp(top, POPOVER_GAP, viewportHeight - height - POPOVER_GAP)}px`;

  const width = Math.min(box.width, window.innerWidth - POPOVER_GAP * 2);
  const left = clamp(box.left, POPOVER_GAP, window.innerWidth - width - POPOVER_GAP);
  wrapper.style.left = `${left}px`;

  // El lado de la flecha ya no corresponde a la nueva posición.
  popover.arrow.style.display = 'none';
}

let latestPopover: PopoverDOM | null = null;
let adjustmentFrame = 0;
let settleTimer: number | undefined;

/**
 * Driver.js pinta el popover, desplaza la página y recién después lo posiciona.
 * El ajuste se repite tras el cuadro de render y cuando termina el desplazamiento
 * suave; si la página se mueve, se vuelve a aplicar sobre la posición final.
 */
function schedulePopoverAdjustment(popover: PopoverDOM | null = latestPopover): void {
  if (typeof window === 'undefined' || !popover) return;
  latestPopover = popover;

  if (adjustmentFrame) window.cancelAnimationFrame(adjustmentFrame);
  adjustmentFrame = window.requestAnimationFrame(() => {
    adjustmentFrame = window.requestAnimationFrame(() => {
      adjustmentFrame = 0;
      keepPopoverClear(popover);
    });
  });

  window.clearTimeout(settleTimer);
  settleTimer = window.setTimeout(() => keepPopoverClear(popover), 420);
}

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
  waitForElement: 2500,
  onPopoverRender: (popover) => schedulePopoverAdjustment(popover),
  onHighlighted: () => schedulePopoverAdjustment()
};

@Injectable({ providedIn: 'root' })
export class ProductTourService {
  /** Indica si hay un recorrido con botones activo. */
  readonly isRunning = signal<boolean>(false);

  private instance: Driver | null = null;
  private returnFocusTo: HTMLElement | null = null;
  private escapeHandler: ((event: KeyboardEvent) => void) | null = null;
  private stopTrackingPopover: (() => void) | null = null;

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
        latestPopover = null;
        this.stopTracking();
        this.isRunning.set(false);
        this.unbindEscape();
        markHelpGuideSeen('tour', tour.id);
        this.restoreFocus();
      }
    });

    this.instance = instance;
    this.isRunning.set(true);
    this.bindEscape();
    this.trackPopover();
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
    this.stopTracking();
    this.instance?.destroy();
    this.instance = null;
    latestPopover = null;
    this.isRunning.set(false);
    this.restoreFocus();
  }

  /**
   * Vuelve a ajustar el popover cuando la página se desplaza o cambia de
   * tamaño: Driver.js recalcula su posición y puede volver a tapar el objetivo.
   */
  private trackPopover(): void {
    if (this.stopTrackingPopover || typeof window === 'undefined') return;
    const refresh = () => schedulePopoverAdjustment();
    window.addEventListener('resize', refresh);
    document.addEventListener('scroll', refresh, true);
    this.stopTrackingPopover = () => {
      window.removeEventListener('resize', refresh);
      document.removeEventListener('scroll', refresh, true);
    };
  }

  private stopTracking(): void {
    this.stopTrackingPopover?.();
    this.stopTrackingPopover = null;
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
