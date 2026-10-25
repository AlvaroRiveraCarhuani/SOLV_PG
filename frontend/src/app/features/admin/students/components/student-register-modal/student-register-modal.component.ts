import {
  Component,
  ChangeDetectionStrategy,
  computed,
  inject,
  input,
  output,
  signal,
  viewChild
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { CreateStudentDTO } from '../../../services/admin-students.service';
import { TenantService } from '@core/services/tenant.service';
import { ModalShellComponent } from '@shared/components/modal-shell/modal-shell.component';
import { FormFieldComponent } from '@shared/components/form-field/form-field.component';
import { RosterUploadComponent } from '@shared/components/roster-upload/roster-upload.component';
import {
  LucideUserPlus,
  LucideAlertTriangle,
  LucideFileSpreadsheet,
  LucideUserCheck
} from '@lucide/angular';

export type RegisterTab = 'individual' | 'batch';

@Component({
  selector: 'student-register-modal',
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    CommonModule,
    FormsModule,
    ModalShellComponent,
    FormFieldComponent,
    RosterUploadComponent,
    LucideUserPlus,
    LucideAlertTriangle,
    LucideFileSpreadsheet,
    LucideUserCheck
  ],
  templateUrl: './student-register-modal.component.html',
  styleUrl: './student-register-modal.component.scss'
})
export class StudentRegisterModalComponent {
  private readonly tenantService = inject(TenantService);

  readonly rosterUpload = viewChild(RosterUploadComponent);
  readonly isSubmitting = input<boolean>(false);

  readonly closed = output<void>();
  readonly studentCreated = output<CreateStudentDTO>();
  readonly batchSubmit = output<string[]>();

  readonly currentTab = signal<RegisterTab>('individual');

  readonly firstName = signal('');
  readonly lastName = signal('');
  readonly email = signal('');
  readonly error = signal<string | null>(null);

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

  setTab(tab: RegisterTab): void {
    this.currentTab.set(tab);
    this.error.set(null);
  }

  onSubmitIndividual(): void {
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

  confirmBatch(): void {
    const emails = this.rosterUpload()?.parsedRoster()?.valid_emails ?? [];
    if (emails.length === 0) {
      return;
    }
    this.batchSubmit.emit(emails);
  }
}
