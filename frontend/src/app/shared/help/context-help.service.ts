import { Injectable, computed, effect, signal } from '@angular/core';
import { getContextHelpEntry } from './help-content.constants';
import { ContextHelpEntry, HelpGuideSet } from './help.types';

/**
 * Estado del centro de ayuda; una sola instancia por aplicación.
 *
 * El botón `?` de la barra superior abre el panel de la pantalla registrada.
 * Cada vista declara su clave al montarse y, si tiene guías propias, registra
 * las acciones que el panel puede lanzar. El registro funciona como una pila:
 * una pestaña puede añadir su contenido sobre el de su sección y devolverlo al
 * salir, sin que el panel conozca las pantallas.
 */
@Injectable({ providedIn: 'root' })
export class ContextHelpService {
  private readonly openSignal = signal<boolean>(false);
  private readonly screenStack = signal<string[]>([]);
  private readonly guidesSignal = signal<Record<string, HelpGuideSet>>({});
  private returnFocusTo: HTMLElement | null = null;

  readonly isOpen = this.openSignal.asReadonly();

  /** Clave de la vista visible: la última registrada. */
  readonly screenKey = computed<string | null>(() => {
    const stack = this.screenStack();
    return stack.length > 0 ? stack[stack.length - 1] : null;
  });

  /** Entrada contextual de la pantalla actual; null si no tiene explicación. */
  readonly entry = computed<ContextHelpEntry | null>(() => {
    const key = this.screenKey();
    return key ? getContextHelpEntry(key) : null;
  });

  /** Guías registradas por la pantalla actual; nunca inventa recorridos. */
  readonly guides = computed<HelpGuideSet>(() => {
    const key = this.screenKey();
    return key ? this.guidesSignal()[key] ?? {} : {};
  });

  /** Sin explicación ni guías no se ofrece el acceso: el manual sigue en el menú. */
  readonly hasContent = computed<boolean>(() => {
    const guides = this.guides();
    return this.entry() !== null || !!guides.tour || !!guides.tutorial;
  });

  constructor() {
    // Si la vista deja de tener ayuda mientras el panel está abierto, se cierra
    // en lugar de quedar mostrando un panel vacío.
    effect(() => {
      if (this.openSignal() && !this.hasContent()) {
        this.openSignal.set(false);
        this.returnFocusTo = null;
      }
    });
  }

  registerScreen(key: string): void {
    if (!getContextHelpEntry(key)) return;
    this.screenStack.update((stack) => (stack.includes(key) ? stack : [...stack, key]));
  }

  unregisterScreen(key: string): void {
    this.screenStack.update((stack) => stack.filter((current) => current !== key));
  }

  registerGuides(key: string, guides: HelpGuideSet): void {
    this.guidesSignal.update((current) => ({ ...current, [key]: guides }));
  }

  unregisterGuides(key: string): void {
    this.guidesSignal.update(({ [key]: _removed, ...rest }) => rest);
  }

  open(trigger?: HTMLElement | null): void {
    if (this.openSignal()) return;
    // El panel se destruye al cerrarse, así que el foco de retorno se guarda
    // aquí: el servicio sobrevive al componente. El disparador se recibe de
    // forma explícita para que el foco vuelva al botón `?` aunque el documento
    // no tenga el foco en ese instante.
    this.returnFocusTo =
      trigger ?? (typeof document !== 'undefined' ? (document.activeElement as HTMLElement | null) : null);
    this.openSignal.set(true);
  }

  close(): void {
    if (!this.openSignal()) return;
    this.openSignal.set(false);
    const target = this.returnFocusTo;
    this.returnFocusTo = null;
    if (target) setTimeout(() => target.focus(), 0);
  }

  toggle(trigger?: HTMLElement | null): void {
    if (this.openSignal()) {
      this.close();
      return;
    }
    this.open(trigger);
  }
}
