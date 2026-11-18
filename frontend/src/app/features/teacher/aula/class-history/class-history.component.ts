import { ChangeDetectionStrategy, Component, computed, input, signal } from '@angular/core';
import { LucideChevronDown } from '@lucide/angular';

import { DateTextPipe } from '@shared/pipes/date-text.pipe';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { ClassHistoryEvent } from '../../models/teacher.models';

const ACTION_LABELS: Record<string, string> = {
  CLASS_ADVANCE_EMITTED: 'Emitió un avance',
  // El mecanismo se llama congelar en el contrato; la pantalla habla de solo lectura.
  CLASS_FROZEN: 'Dejó la clase en solo lectura',
  CLASS_UNFROZEN: 'Devolvió la escritura a la clase',
  CLASS_ADVANCE_GRACE_SET: 'Ajustó la ventana',
  // Acciones anteriores al 2026-11-18: se conservan para las filas ya escritas.
  CLASS_ADVANCE_GRACE_EXTENDED: 'Sumó minutos a la ventana',
  CLASS_ADVANCE_GRACE_DEADLINE_SET: 'Fijó una fecha límite',
  CLASS_ADVANCE_GRACE_CLOSED: 'Cerró la ventana',
  CLASS_ADVANCE_APPLIED: 'Aplicó un avance',
  CLASS_ADVANCE_DISMISSED: 'Descartó un avance',
  CLASS_ADVANCE_ROLLBACK: 'Deshizo un avance'
};

const ROLE_LABELS: Record<string, string> = {
  teacher: 'Docente',
  student: 'Alumno',
  admin: 'Admin'
};

/** Un día de la clase con sus movimientos, del más nuevo al más viejo. */
interface HistoryDay {
  key: string;
  label: string | null;
  events: ClassHistoryEvent[];
}

// La clave es la fecha local, no la del servidor: el docente lee su calendario.
function dayKey(date: Date): string {
  return `${date.getFullYear()}-${date.getMonth() + 1}-${date.getDate()}`;
}

/**
 * Historial de una clase: los movimientos del docente y las decisiones de los
 * alumnos, del más nuevo al más viejo. Es solo lectura; el alumno no lo ve.
 *
 * Arranca plegado: es el registro para consultar cuando algo se pregunta, no la
 * operación del momento (ADR-087). El encabezado queda siempre visible.
 */
@Component({
  selector: 'class-history',
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [DateTextPipe, MachineDataDirective, LucideChevronDown],
  templateUrl: './class-history.component.html',
  styleUrl: './class-history.component.scss'
})
export class ClassHistoryComponent {
  readonly events = input<ClassHistoryEvent[]>([]);
  readonly isLoading = input(false);
  readonly error = input<string | null>(null);

  readonly isOpen = signal(false);

  // Agrupa por día: sin separadores, dos jornadas se leen como una sola lista.
  readonly days = computed<HistoryDay[]>(() => {
    const now = new Date();
    const today = dayKey(now);
    const yesterday = dayKey(new Date(now.getFullYear(), now.getMonth(), now.getDate() - 1));
    const days: HistoryDay[] = [];

    for (const event of this.events()) {
      const created = new Date(event.created_at);
      const key = Number.isNaN(created.getTime()) ? event.created_at : dayKey(created);
      const current = days.at(-1);
      if (current && current.key === key) {
        current.events.push(event);
        continue;
      }
      const label = key === today ? 'Hoy' : key === yesterday ? 'Ayer' : null;
      days.push({ key, label, events: [event] });
    }

    return days;
  });

  toggle(): void {
    this.isOpen.update(open => !open);
  }

  // El actor no se repite en cada fila: solo cuando cambia respecto de la anterior.
  showsActor(day: HistoryDay, index: number): boolean {
    if (index === 0) return true;
    const current = day.events[index];
    const previous = day.events[index - 1];
    return current.actor_id !== previous.actor_id || current.actor_role !== previous.actor_role;
  }

  actionLabel(action: string): string {
    return ACTION_LABELS[action] ?? 'Movimiento del aula';
  }

  actorRoleLabel(role: string): string {
    return ROLE_LABELS[role] ?? role;
  }

}
