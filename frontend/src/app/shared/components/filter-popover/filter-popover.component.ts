import {
  ChangeDetectionStrategy,
  Component,
  ElementRef,
  HostListener,
  computed,
  input,
  output,
  signal,
  viewChild
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { LucideSlidersHorizontal } from '@lucide/angular';
import { DismissibleDirective } from '@shared/directives/dismissible.directive';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';

let nextFilterPopoverId = 0;

const VIEWPORT_MARGIN = 12;
const FALLBACK_WIDTH = 440;

/**
 * Disparador y panel del popover de filtros. Las columnas de opciones viven en
 * el contenido proyectado; el componente aporta la cabecera, el contador de
 * filtros activos, el cierre por clic fuera o Escape y la alineación del panel.
 *
 * El panel se abre alineado a la izquierda del botón. Si no cabe hacia la
 * derecha, se alinea a la derecha; si tampoco cabe, se desplaza lo justo para
 * quedar dentro del viewport. Así no lo recorta el sidebar ni el borde derecho.
 */
@Component({
  selector: 'filter-popover',
  standalone: true,
  imports: [CommonModule, LucideSlidersHorizontal, DismissibleDirective, MachineDataDirective],
  templateUrl: './filter-popover.component.html',
  styleUrl: './filter-popover.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class FilterPopoverComponent {
  readonly buttonLabel = input<string>('Filtros');
  readonly activeCount = input<number>(0);
  readonly width = input<string>('440px');
  readonly triggerAriaLabel = input<string>('Abrir filtros');
  readonly canClear = input<boolean | undefined>(undefined);
  readonly cleared = output<void>();

  readonly open = signal(false);
  readonly panelLeft = signal(0);
  readonly panelWidth = signal(FALLBACK_WIDTH);
  readonly panelId = `filter-popover-${nextFilterPopoverId++}`;

  readonly isClearable = computed<boolean>(() => {
    const explicit = this.canClear();
    if (explicit !== undefined) {
      return explicit;
    }
    return this.activeCount() > 0;
  });

  private readonly trigger = viewChild.required<ElementRef<HTMLButtonElement>>('trigger');

  toggle(): void {
    if (this.open()) {
      this.close();
      return;
    }
    this.positionPanel();
    this.open.set(true);
  }

  close(): void {
    this.open.set(false);
  }

  clear(): void {
    if (!this.isClearable()) {
      return;
    }
    this.cleared.emit();
  }

  @HostListener('window:resize')
  onViewportResize(): void {
    if (this.open()) {
      this.positionPanel();
    }
  }

  private positionPanel(): void {
    const rect = this.trigger().nativeElement.getBoundingClientRect();
    const viewport = window.innerWidth || document.documentElement.clientWidth;
    const requested = Number.parseFloat(this.width());
    const panelWidth = Math.max(
      0,
      Math.min(Number.isFinite(requested) ? requested : FALLBACK_WIDTH, viewport - VIEWPORT_MARGIN * 2)
    );
    const rightSpace = viewport - rect.left - VIEWPORT_MARGIN;
    const leftSpace = rect.right - VIEWPORT_MARGIN;

    let left = 0;
    if (rightSpace < panelWidth) {
      left = leftSpace >= panelWidth ? rect.width - panelWidth : VIEWPORT_MARGIN - rect.left;
    }

    this.panelWidth.set(panelWidth);
    this.panelLeft.set(Math.round(left));
  }
}
