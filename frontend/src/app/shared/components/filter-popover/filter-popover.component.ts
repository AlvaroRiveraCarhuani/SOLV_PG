import { ChangeDetectionStrategy, Component, input, output, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { LucideSlidersHorizontal } from '@lucide/angular';
import { DismissibleDirective } from '@shared/directives/dismissible.directive';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';

let nextFilterPopoverId = 0;

/**
 * Disparador y panel del popover de filtros. Las columnas de opciones viven en
 * el contenido proyectado; el componente aporta la cabecera, el contador de
 * filtros activos y el cierre por clic fuera o Escape.
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
  readonly cleared = output<void>();

  readonly open = signal(false);
  readonly panelId = `filter-popover-${nextFilterPopoverId++}`;

  toggle(): void {
    this.open.update((isOpen) => !isOpen);
  }

  close(): void {
    this.open.set(false);
  }

  clear(): void {
    this.cleared.emit();
  }
}
