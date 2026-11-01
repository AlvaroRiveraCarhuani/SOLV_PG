import { ChangeDetectionStrategy, Component, computed, inject, input, output, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { LucideLockOpen } from '@lucide/angular';

import { ModalShellComponent } from '@shared/components/modal-shell/modal-shell.component';
import { ClassSessionService } from '../../services/class-session.service';
import { ClassSessionStudent, FreezeReport } from '../../models/teacher.models';

/**
 * Excepción individual del examen cerrado: devuelve la escritura al volumen de
 * un alumno. El motivo es obligatorio y queda en la auditoría de la clase
 * (ADR-005 §5.2).
 */
@Component({
  selector: 'unfreeze-student-modal',
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [CommonModule, FormsModule, ModalShellComponent, LucideLockOpen],
  templateUrl: './unfreeze-student-modal.component.html',
  styleUrl: './unfreeze-student-modal.component.scss'
})
export class UnfreezeStudentModalComponent {
  private readonly sessionService = inject(ClassSessionService);

  readonly sessionId = input.required<string>();
  readonly student = input.required<ClassSessionStudent>();

  readonly closed = output<void>();
  readonly unfrozen = output<FreezeReport>();

  readonly reason = signal('');
  readonly isSending = signal(false);
  readonly errorMessage = signal('');

  readonly canSubmit = computed(() => this.reason().trim().length > 0 && !this.isSending());

  onReasonChange(value: string): void {
    this.reason.set(value);
    this.errorMessage.set('');
  }

  submit(): void {
    if (!this.canSubmit()) {
      return;
    }
    const reason = this.reason().trim();
    this.isSending.set(true);
    this.errorMessage.set('');
    this.sessionService.unfreezeStudent(this.sessionId(), this.student().student_id, reason).subscribe({
      next: report => {
        this.isSending.set(false);
        this.unfrozen.emit(report);
      },
      error: err => {
        this.isSending.set(false);
        // El backend devuelve el motivo legible en `error`; se muestra ese.
        const typed = err as { error?: { error?: string; message?: string }; message?: string };
        this.errorMessage.set(
          typed?.error?.error || typed?.error?.message || typed?.message || 'No se pudo reabrir el entorno.'
        );
      }
    });
  }

  onClose(): void {
    if (this.isSending()) {
      return;
    }
    this.closed.emit();
  }
}
