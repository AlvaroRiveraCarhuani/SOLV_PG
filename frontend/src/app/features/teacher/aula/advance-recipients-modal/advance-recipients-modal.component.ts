import { ChangeDetectionStrategy, Component, input, output } from '@angular/core';
import { CommonModule } from '@angular/common';
import { LucideUsers } from '@lucide/angular';
import { ModalShellComponent } from '@shared/components/modal-shell/modal-shell.component';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { DateTextPipe } from '@shared/pipes/date-text.pipe';
import { AdvanceRecipientStatus, SessionAdvance } from '../../models/teacher.models';

const STATUS_LABELS: Record<AdvanceRecipientStatus, string> = {
  pending: 'Pendiente',
  applied: 'Aplicó',
  dismissed: 'Descartó',
  failed: 'Falló',
  expired: 'Venció'
};

/**
 * Detalle de una emisión: quién aplicó el avance, quién lo descartó y quién
 * sigue pendiente. La lista llega con nombre y correo desde el servidor, así que
 * la pantalla no cruza identificadores con el tablero del aula.
 */
@Component({
  selector: 'advance-recipients-modal',
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    CommonModule,
    ModalShellComponent,
    MachineDataDirective,
    DateTextPipe,
    LucideUsers
  ],
  templateUrl: './advance-recipients-modal.component.html',
  styleUrl: './advance-recipients-modal.component.scss'
})
export class AdvanceRecipientsModalComponent {
  readonly advance = input.required<SessionAdvance>();

  readonly closed = output<void>();

  statusLabel(status: AdvanceRecipientStatus): string {
    return STATUS_LABELS[status] ?? status;
  }

  onClose(): void {
    this.closed.emit();
  }
}
