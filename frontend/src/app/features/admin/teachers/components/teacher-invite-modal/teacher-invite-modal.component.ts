import { Component, output, signal, computed, inject, HostListener } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { TeacherInvitationPayload } from '@core/models/admin.model';
import { TenantService } from '@core/services/tenant.service';
import { LucideX, LucideUserPlus, LucideMail, LucideShieldCheck, LucideAlertTriangle } from '@lucide/angular';

@Component({
  selector: 'solv-teacher-invite-modal',
  standalone: true,
  imports: [CommonModule, FormsModule, LucideX, LucideUserPlus, LucideMail, LucideShieldCheck, LucideAlertTriangle],
  templateUrl: './teacher-invite-modal.component.html',
  styleUrl: './teacher-invite-modal.component.scss'
})
export class TeacherInviteModalComponent {
  private readonly tenantService = inject(TenantService);

  close = output<void>();
  submit = output<TeacherInvitationPayload>();

  @HostListener('document:keydown.escape')
  handleEscape(): void {
    this.close.emit();
  }

  email = signal<string>('');
  sendEmail = signal<boolean>(true);
  isSubmitting = signal<boolean>(false);

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

  isValidEmail(): boolean {
    const e = this.email().trim();
    return e.length > 5 && e.includes('@') && e.includes('.');
  }

  onSubmit(): void {
    if (!this.isValidEmail() || this.isSubmitting()) return;
    this.isSubmitting.set(true);
    this.submit.emit({
      email: this.email().trim(),
      role: 'teacher',
      role_type: 'titular',
      send_email: this.sendEmail()
    });
  }
}
