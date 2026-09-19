import { Component, input, output, signal, computed, inject, HostListener } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { CreateStudentDTO } from '../../../services/admin-students.service';
import { TenantService } from '@core/services/tenant.service';
import { LucideUserPlus, LucideX, LucideAlertTriangle } from '@lucide/angular';

@Component({
  selector: 'solv-student-create-modal',
  standalone: true,
  imports: [CommonModule, FormsModule, LucideUserPlus, LucideX, LucideAlertTriangle],
  templateUrl: './student-create-modal.component.html',
  styleUrls: ['./student-create-modal.component.scss']
})
export class StudentCreateModalComponent {
  private readonly tenantService = inject(TenantService);

  isSubmitting = input<boolean>(false);
  close = output<void>();
  studentCreated = output<CreateStudentDTO>();

  firstName = signal<string>('');
  lastName = signal<string>('');
  email = signal<string>('');
  error = signal<string | null>(null);

  // Dominio institucional dinámico
  institutionDomain = computed<string>(() => {
    const cfg = this.tenantService.config();
    if (cfg?.support_email && cfg.support_email.includes('@')) {
      return cfg.support_email.split('@')[1].toLowerCase().trim();
    }
    if (cfg?.base_domain) {
      const parts = cfg.base_domain.toLowerCase().trim().split('.');
      if (parts.length > 2) {
        return parts.slice(1).join('.');
      }
      return cfg.base_domain;
    }
    return '';
  });

  enteredDomain = computed<string>(() => {
    const em = this.email().trim().toLowerCase();
    const atIdx = em.lastIndexOf('@');
    if (atIdx === -1) return '';
    return em.slice(atIdx + 1);
  });

  isExternalDomain = computed<boolean>(() => {
    const entered = this.enteredDomain();
    const inst = this.institutionDomain();
    if (!entered || !inst || !entered.includes('.')) return false;
    return entered !== inst && !entered.endsWith('.' + inst);
  });

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
