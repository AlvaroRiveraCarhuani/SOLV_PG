import { Component, ChangeDetectionStrategy, model, input } from '@angular/core';
import { CommonModule } from '@angular/common';
import { LucideLayoutGrid, LucideTable } from '@lucide/angular';

export type ViewMode = 'cards' | 'table';

@Component({
  selector: 'view-switcher',
  standalone: true,
  imports: [CommonModule, LucideLayoutGrid, LucideTable],
  templateUrl: './view-switcher.component.html',
  styleUrls: ['./view-switcher.component.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class ViewSwitcherComponent {
  readonly mode = model<ViewMode>('table');
  readonly cardsLabel = input<string>('Tarjetas');
  readonly tableLabel = input<string>('Tabla');
  readonly ariaLabel = input<string>('Modo de visualización');

  selectMode(newMode: ViewMode): void {
    if (this.mode() !== newMode) {
      this.mode.set(newMode);
    }
  }
}
