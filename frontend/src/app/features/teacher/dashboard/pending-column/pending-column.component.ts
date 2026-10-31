import { ChangeDetectionStrategy, Component, computed, input, signal } from '@angular/core';
import { RouterLink } from '@angular/router';
import { LucideChevronLeft, LucideChevronRight } from '@lucide/angular';

import { DateTextPipe } from '@shared/pipes/date-text.pipe';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import {
  TeacherOverviewDue,
  TeacherOverviewQueue,
  TeacherOverviewSession
} from '../../models/teacher.models';

/**
 * Columna de pendientes de Hoy: la lista accionable de la pantalla. Tres
 * bloques de orden fijo, cada uno visible solo con contenido y sin repetir las
 * cifras del canvas. Incluye minipaginador minimalista para vencimientos.
 */
@Component({
  selector: 'pending-column',
  standalone: true,
  imports: [RouterLink, DateTextPipe, MachineDataDirective, LucideChevronLeft, LucideChevronRight],
  templateUrl: './pending-column.component.html',
  styleUrl: './pending-column.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class PendingColumnComponent {
  readonly session = input<TeacherOverviewSession | null>(null);
  readonly queue = input.required<TeacherOverviewQueue>();
  readonly dueSoon = input<TeacherOverviewDue[]>([]);
  /** Vencimientos fuera del límite, ya restados por el servidor. */
  readonly dueSoonHidden = input<number>(0);

  readonly page = signal<number>(1);
  readonly pageSize = 4;

  readonly totalPages = computed(() => {
    const total = this.dueSoon().length;
    return Math.max(1, Math.ceil(total / this.pageSize));
  });

  readonly pagedDueSoon = computed(() => {
    const list = this.dueSoon();
    const current = Math.min(this.page(), this.totalPages());
    const start = (current - 1) * this.pageSize;
    return list.slice(start, start + this.pageSize);
  });

  prevPage(): void {
    if (this.page() > 1) {
      this.page.update(p => p - 1);
    }
  }

  nextPage(): void {
    if (this.page() < this.totalPages()) {
      this.page.update(p => p + 1);
    }
  }

  /** Etiqueta de urgencia; el cálculo viene del servidor. */
  urgencyLabel(urgency: TeacherOverviewDue['urgency']): string {
    switch (urgency) {
      case 'overdue':
        return 'vencida';
      case 'today':
        return 'vence hoy';
      case 'soon':
        return 'vence pronto';
      default:
        return 'vence';
    }
  }
}
