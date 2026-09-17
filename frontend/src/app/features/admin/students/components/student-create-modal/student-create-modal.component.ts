import { Component, input, output, signal, HostListener } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { CreateStudentDTO } from '../../../services/admin-students.service';
import { LucideUserPlus, LucideX } from '@lucide/angular';

@Component({
  selector: 'solv-student-create-modal',
  standalone: true,
  imports: [CommonModule, FormsModule, LucideUserPlus, LucideX],
  templateUrl: './student-create-modal.component.html',
  styleUrls: ['./student-create-modal.component.scss']
})
export class StudentCreateModalComponent {
  isSubmitting = input<boolean>(false);
  close = output<void>();
  studentCreated = output<CreateStudentDTO>();

  firstName = signal<string>('');
  lastName = signal<string>('');
  email = signal<string>('');
  error = signal<string | null>(null);

  @HostListener('document:keydown.escape')
  handleEscape(): void {
    if (!this.isSubmitting()) {
      this.close.emit();
    }
  }

  onSubmit(): void {
    const fn = this.firstName().trim();
    const ln = this.lastName().trim();
    const em = this.email().trim().toLowerCase();

    if (!fn || !ln || !em) {
      this.error.set('Todos los campos son obligatorios.');
      return;
    }

    if (!em.includes('@') || !em.includes('.')) {
      this.error.set('Ingresá un correo electrónico válido.');
      return;
    }

    this.error.set(null);
    this.studentCreated.emit({
      first_name: fn,
      last_name: ln,
      email: em
    });
  }
}
