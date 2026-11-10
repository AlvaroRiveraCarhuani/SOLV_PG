import { Component, input, output, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { LucideArrowUpDown, LucideArrowUp, LucideArrowDown } from '@lucide/angular';
import { SortDirection } from './sort-header.types';

@Component({
  selector: 'sort-header',
  standalone: true,
  imports: [CommonModule, LucideArrowUpDown, LucideArrowUp, LucideArrowDown],
  templateUrl: './sort-header.component.html',
  styleUrls: ['./sort-header.component.scss']
})
export class SortHeaderComponent {
  /** Columna que representa este header */
  column = input.required<string>();
  
  /** Columna actualmente activa en la tabla */
  activeColumn = input<string | null>(null);

  /** Dirección actual de ordenamiento */
  direction = input<SortDirection>('none');

  /** Evento emitido al hacer clic, con la nueva dirección */
  sortChange = output<{ column: string; direction: SortDirection }>();

  isActive = computed(() => this.activeColumn() === this.column() && this.direction() !== 'none');

  currentDirection = computed<SortDirection>(() => {
    return this.activeColumn() === this.column() ? this.direction() : 'none';
  });

  ariaSortValue = computed<string>(() => {
    const dir = this.currentDirection();
    if (dir === 'asc') return 'ascending';
    if (dir === 'desc') return 'descending';
    return 'none';
  });

  onToggle(): void {
    const current = this.currentDirection();
    // asc -> desc -> sin orden -> asc: el tercer clic limpia el orden de la tabla.
    const next: SortDirection =
      current === 'asc' ? 'desc' : current === 'desc' ? 'none' : 'asc';

    this.sortChange.emit({
      column: this.column(),
      direction: next
    });
  }
}
