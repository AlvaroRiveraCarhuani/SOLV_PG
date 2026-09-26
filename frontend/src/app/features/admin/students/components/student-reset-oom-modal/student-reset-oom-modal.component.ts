import { Component, input, output, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { AdminStudentItem } from '../../../services/admin-students.service';
import { ModalShellComponent } from '@shared/components/modal-shell/modal-shell.component';
import { FormFieldComponent } from '@shared/components/form-field/form-field.component';
import { LucideAlertTriangle, LucideRotateCcw } from '@lucide/angular';

@Component({
  selector: 'student-reset-oom-modal',
  standalone: true,
  imports: [CommonModule, FormsModule, ModalShellComponent, FormFieldComponent, LucideAlertTriangle, LucideRotateCcw],
  templateUrl: './student-reset-oom-modal.component.html',
  styleUrls: ['./student-reset-oom-modal.component.scss']
})
export class StudentResetOOMModalComponent {
  student = input.required<AdminStudentItem>();
  isSubmitting = input<boolean>(false);

  confirmReset = output<string>();
  close = output<void>();

  reason = signal<string>('');
  error = signal<string | null>(null);

  onSubmit(): void {
    const trimmed = this.reason().trim();
    if (!trimmed || trimmed.length < 10) {
      this.error.set('El motivo de justificación debe contener al menos 10 caracteres.');
      return;
    }

    this.error.set(null);
    this.confirmReset.emit(trimmed);
  }
}
