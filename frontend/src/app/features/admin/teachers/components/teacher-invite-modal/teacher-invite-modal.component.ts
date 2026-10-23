import { Component, output, signal, computed, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { 
  DirectTeacherPayload, 
  BatchDirectTeacherPayload, 
  ParsedRosterResult 
} from '@core/models/admin.model';
import { TenantService } from '@core/services/tenant.service';
import { AdminTeachersService } from '@features/admin/services/admin-teachers.service';
import { ModalShellComponent } from '@shared/components/modal-shell/modal-shell.component';
import { FormFieldComponent } from '@shared/components/form-field/form-field.component';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { 
  LucideUserPlus, 
  LucideMail, 
  LucideAlertTriangle,
  LucideUpload,
  LucideFileSpreadsheet,
  LucideCheckCircle2,
  LucideFileCheck,
  LucideUserCheck
} from '@lucide/angular';

export type ModalTab = 'individual' | 'batch';

@Component({
  selector: 'teacher-invite-modal',
  standalone: true,
  imports: [
    CommonModule, 
    FormsModule, 
    ModalShellComponent, 
    FormFieldComponent, 
    MachineDataDirective,
    LucideUserPlus, 
    LucideMail, 
    LucideAlertTriangle,
    LucideUpload,
    LucideFileSpreadsheet,
    LucideCheckCircle2,
    LucideFileCheck,
    LucideUserCheck
  ],
  templateUrl: './teacher-invite-modal.component.html',
  styleUrl: './teacher-invite-modal.component.scss'
})
export class TeacherInviteModalComponent {
  private readonly tenantService = inject(TenantService);
  private readonly teachersService = inject(AdminTeachersService);

  closed = output<void>();
  directSubmit = output<DirectTeacherPayload>();
  batchSubmit = output<BatchDirectTeacherPayload>();

  currentTab = signal<ModalTab>('individual');

  // Modo Individual
  email = signal<string>('');
  firstName = signal<string>('');
  lastName = signal<string>('');
  sendNotification = signal<boolean>(true);
  isSubmitting = signal<boolean>(false);

  // Modo Importar Planilla
  selectedFile = signal<File | null>(null);
  isDragging = signal<boolean>(false);
  isParsing = signal<boolean>(false);
  parseError = signal<string | null>(null);
  parsedRoster = signal<ParsedRosterResult | null>(null);
  sendNotificationBatch = signal<boolean>(true);
  isSubmittingBatch = signal<boolean>(false);

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
    const clean = this.email().trim().toLowerCase();
    const atIdx = clean.lastIndexOf('@');
    if (atIdx === -1) return '';
    return clean.slice(atIdx + 1);
  });

  isExternalDomain = computed<boolean>(() => {
    const entered = this.enteredDomain();
    const inst = this.institutionDomain();
    if (!entered || !inst || !entered.includes('.')) return false;
    return entered !== inst && !entered.endsWith('.' + inst);
  });

  isValidEmail = computed<boolean>(() => {
    const e = this.email().trim();
    return e.length > 5 && e.includes('@') && e.includes('.');
  });

  setTab(tab: ModalTab): void {
    this.currentTab.set(tab);
  }

  // --- Handlers Registro Individual ---
  onSubmitIndividual(): void {
    if (!this.isValidEmail() || this.isSubmitting()) return;
    this.isSubmitting.set(true);
    this.directSubmit.emit({
      email: this.email().trim(),
      first_name: this.firstName().trim() || undefined,
      last_name: this.lastName().trim() || undefined,
      role_type: 'titular',
      send_notification: this.sendNotification()
    });
  }

  // --- Handlers Importar Planilla ---
  onDragOver(e: DragEvent): void {
    e.preventDefault();
    e.stopPropagation();
    this.isDragging.set(true);
  }

  onDragLeave(e: DragEvent): void {
    e.preventDefault();
    e.stopPropagation();
    this.isDragging.set(false);
  }

  onDrop(e: DragEvent): void {
    e.preventDefault();
    e.stopPropagation();
    this.isDragging.set(false);
    const files = e.dataTransfer?.files;
    if (files && files.length > 0) {
      this.processFile(files[0]);
    }
  }

  onFileInputChange(e: Event): void {
    const input = e.target as HTMLInputElement;
    if (input.files && input.files.length > 0) {
      this.processFile(input.files[0]);
    }
  }

  processFile(file: File): void {
    const ext = file.name.slice(file.name.lastIndexOf('.')).toLowerCase();
    if (ext !== '.xlsx' && ext !== '.csv') {
      this.parseError.set('Formato no admitido. Por favor selecciona un archivo .xlsx o .csv.');
      return;
    }

    this.selectedFile.set(file);
    this.parseError.set(null);
    this.isParsing.set(true);
    this.parsedRoster.set(null);

    this.teachersService.parseRosterFile(file).subscribe({
      next: (res: ParsedRosterResult | null) => {
        this.isParsing.set(false);
        if (!res || res.total_detected === 0) {
          this.parseError.set('No se encontraron correos electrónicos en el archivo seleccionado.');
        } else {
          this.parsedRoster.set(res);
        }
      },
      error: () => {
        this.isParsing.set(false);
        this.parseError.set('No se pudo leer el archivo. Comprueba que no esté dañado e intenta de nuevo.');
      }
    });
  }

  resetBatchFile(): void {
    this.selectedFile.set(null);
    this.parsedRoster.set(null);
    this.parseError.set(null);
  }

  onSubmitBatch(): void {
    const roster = this.parsedRoster();
    if (!roster || roster.valid_emails.length === 0 || this.isSubmittingBatch()) return;
    this.isSubmittingBatch.set(true);
    this.batchSubmit.emit({
      emails: roster.valid_emails,
      send_notification: this.sendNotificationBatch()
    });
  }
}
