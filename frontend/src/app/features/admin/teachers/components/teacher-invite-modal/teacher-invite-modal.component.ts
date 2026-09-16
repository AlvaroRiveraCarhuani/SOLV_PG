import { Component, output, signal, HostListener } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { TeacherInvitationPayload } from '@core/models/admin.model';
import { LucideX, LucideUserPlus, LucideMail, LucideShieldCheck } from '@lucide/angular';

@Component({
  selector: 'solv-teacher-invite-modal',
  standalone: true,
  imports: [CommonModule, FormsModule, LucideX, LucideUserPlus, LucideMail, LucideShieldCheck],
  templateUrl: './teacher-invite-modal.component.html',
  styleUrl: './teacher-invite-modal.component.scss'
})
export class TeacherInviteModalComponent {
  close = output<void>();
  submit = output<TeacherInvitationPayload>();

  @HostListener('document:keydown.escape')
  handleEscape(): void {
    this.close.emit();
  }

  email = signal<string>('');
  sendEmail = signal<boolean>(true);
  isSubmitting = signal<boolean>(false);

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
