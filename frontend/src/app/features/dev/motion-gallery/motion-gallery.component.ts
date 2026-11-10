import { ChangeDetectionStrategy, Component, OnDestroy, signal } from '@angular/core';
import { RouterLink } from '@angular/router';
import { LucideArrowLeft, LucideLoader2, LucidePlay, LucideRotateCcw } from '@lucide/angular';

import { SkeletonComponent } from '@shared/components/skeleton/skeleton.component';
import { StatusBadgeComponent } from '@shared/components/status-badge/status-badge.component';
import { AiProgressIndicatorComponent } from '@shared/components/ai-progress-indicator/ai-progress-indicator.component';

type DemoMode = 'normal' | 'reduced';

/** Entradas del catálogo global; el orden define la cascada de "Reproducir entradas". */
const ENTRY_IDS = ['fade', 'rise', 'panel', 'backdrop', 'dropdown', 'drawer', 'chip', 'stagger'] as const;

/** Salidas del catálogo global con su duración real, para saber cuándo restaurar la muestra. */
const EXIT_DURATION: Record<string, number> = {
  'fade-out': 150,
  'backdrop-out': 150,
  'panel-out': 150,
  'drawer-out': 150,
  'dropdown-out': 80,
  'modal-out': 150,
};

const EXIT_IDS = Object.keys(EXIT_DURATION);
const HIGHLIGHT_MS = 1800;

/**
 * Galería interna del sistema de animaciones. No es una vista de producto:
 * existe para que quien toca el frontend vea la primitiva real, su duración y
 * su comportamiento con movimiento reducido antes de escribir una animación.
 */
@Component({
  selector: 'motion-gallery',
  standalone: true,
  imports: [
    RouterLink,
    LucideArrowLeft,
    LucideLoader2,
    LucidePlay,
    LucideRotateCcw,
    SkeletonComponent,
    StatusBadgeComponent,
    AiProgressIndicatorComponent,
  ],
  templateUrl: './motion-gallery.component.html',
  styleUrl: './motion-gallery.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class MotionGalleryComponent implements OnDestroy {
  private readonly timers = new Map<string, ReturnType<typeof setTimeout>>();
  private readonly prefersReduced =
    typeof matchMedia === 'function' && matchMedia('(prefers-reduced-motion: reduce)').matches;

  readonly systemPrefersReduced = signal(this.prefersReduced);
  /** Arranca con la preferencia real del sistema; el control permite forzarla solo acá. */
  readonly mode = signal<DemoMode>(this.prefersReduced ? 'reduced' : 'normal');

  readonly staggerRows = [0, 1, 2, 3];

  private readonly entryRuns = signal<Record<string, number>>({});
  readonly exiting = signal<Record<string, boolean>>({});
  readonly highlighting = signal(false);

  /**
   * Clave de repetición de una entrada. Al cambiar, Angular recrea el nodo y la
   * animación vuelve a correr desde el principio sin trucos de `void offsetWidth`.
   */
  tokenOf(id: string): number[] {
    return [this.entryRuns()[id] ?? 0];
  }

  setMode(mode: DemoMode): void {
    if (this.mode() === mode) {
      return;
    }
    this.mode.set(mode);
    this.resetDemos();
    this.playEntries(0);
  }

  playEntry(id: string): void {
    this.entryRuns.update((runs) => ({ ...runs, [id]: (runs[id] ?? 0) + 1 }));
  }

  playEntries(staggerMs = 120): void {
    ENTRY_IDS.forEach((id, index) => {
      this.later(`entry-${id}`, index * staggerMs, () => this.playEntry(id));
    });
  }

  /**
   * Las salidas dejan el elemento en su estado final, así que la muestra se
   * restaura al terminar para poder repetirla. La restauración es un corte, no
   * una entrada: la galería no debe inventar una animación que no existe.
   */
  playExit(id: string): void {
    const duration = EXIT_DURATION[id] ?? 150;
    this.stop(`exit-${id}`);
    this.exiting.update((state) => ({ ...state, [id]: false }));
    this.later(`exit-${id}`, 30, () => {
      this.exiting.update((state) => ({ ...state, [id]: true }));
      this.later(`exit-reset-${id}`, duration + 160, () => {
        this.exiting.update((state) => ({ ...state, [id]: false }));
      });
    });
  }

  playExits(): void {
    EXIT_IDS.forEach((id, index) => {
      this.later(`exit-start-${id}`, index * 220, () => this.playExit(id));
    });
  }

  highlight(): void {
    this.later('highlight-off', HIGHLIGHT_MS, () => this.highlighting.set(false));
    this.highlighting.set(true);
  }

  resetDemos(): void {
    this.timers.forEach((timer) => clearTimeout(timer));
    this.timers.clear();
    this.entryRuns.set({});
    this.exiting.set({});
    this.highlighting.set(false);
  }

  ngOnDestroy(): void {
    this.resetDemos();
  }

  private later(key: string, delay: number, action: () => void): void {
    this.stop(key);
    this.timers.set(
      key,
      setTimeout(() => {
        this.timers.delete(key);
        action();
      }, delay),
    );
  }

  private stop(key: string): void {
    const timer = this.timers.get(key);
    if (timer) {
      clearTimeout(timer);
      this.timers.delete(key);
    }
  }
}
