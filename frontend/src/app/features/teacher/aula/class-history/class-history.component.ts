import { ChangeDetectionStrategy, Component, input, output } from '@angular/core';
import { LucideRotateCcw } from '@lucide/angular';

import { DateTextPipe } from '@shared/pipes/date-text.pipe';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { ClassHistoryEvent } from '../../models/teacher.models';

const ACTION_LABELS: Record<string, string> = {
  CLASS_ADVANCE_EMITTED: 'Emitió un avance',
  CLASS_FROZEN: 'Congeló la clase',
  CLASS_UNFROZEN: 'Descongeló la clase',
  CLASS_ADVANCE_GRACE_EXTENDED: 'Sumó minutos a la ventana',
  CLASS_ADVANCE_GRACE_DEADLINE_SET: 'Fijó una fecha límite',
  CLASS_ADVANCE_GRACE_CLOSED: 'Cerró la ventana',
  CLASS_ADVANCE_APPLIED: 'Aplicó un avance',
  CLASS_ADVANCE_DISMISSED: 'Descartó un avance'
};

const ROLE_LABELS: Record<string, string> = {
  teacher: 'Docente',
  student: 'Alumno',
  admin: 'Admin'
};

/**
 * Historial de una clase: los movimientos del docente y las decisiones de los
 * alumnos, del más nuevo al más viejo. Es solo lectura; el alumno no lo ve.
 */
@Component({
  selector: 'class-history',
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [DateTextPipe, MachineDataDirective, LucideRotateCcw],
  templateUrl: './class-history.component.html',
  styleUrl: './class-history.component.scss'
})
export class ClassHistoryComponent {
  readonly events = input<ClassHistoryEvent[]>([]);
  readonly isLoading = input(false);
  readonly error = input<string | null>(null);

  readonly refresh = output<void>();

  actionLabel(action: string): string {
    return ACTION_LABELS[action] ?? 'Movimiento del aula';
  }

  actorRoleLabel(role: string): string {
    return ROLE_LABELS[role] ?? role;
  }

  onRefresh(): void {
    this.refresh.emit();
  }
}
