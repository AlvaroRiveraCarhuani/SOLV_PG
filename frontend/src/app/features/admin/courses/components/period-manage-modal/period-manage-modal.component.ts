import { Component, EventEmitter, Input, Output, HostListener, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { AcademicPeriod, AdminCoursesService, CreateAcademicPeriodDTO } from '../../../services/admin-courses.service';
import { ConfirmModalComponent } from '@shared/components/confirm-modal/confirm-modal.component';
import { 
  LucideCalendar, 
  LucidePlus, 
  LucideCheck, 
  LucideX, 
  LucideTrash2, 
  LucideAlertCircle,
  LucidePencil
} from '@lucide/angular';

@Component({
  selector: 'solv-period-manage-modal',
  standalone: true,
  imports: [
    CommonModule, 
    FormsModule, 
    ConfirmModalComponent,
    LucideCalendar, 
    LucidePlus, 
    LucideCheck, 
    LucideX, 
    LucideTrash2, 
    LucideAlertCircle,
    LucidePencil
  ],
  templateUrl: './period-manage-modal.component.html',
  styleUrls: ['./period-manage-modal.component.scss']
})
export class PeriodManageModalComponent {
  private coursesService = inject(AdminCoursesService);

  @Input() periods: AcademicPeriod[] = [];
  @Output() close = new EventEmitter<void>();
  @Output() periodCreated = new EventEmitter<AcademicPeriod>();
  @Output() periodActivated = new EventEmitter<AcademicPeriod>();

  periodToDelete = signal<AcademicPeriod | null>(null);

  @HostListener('document:keydown.escape')
  handleEscape(): void {
    if (this.periodToDelete()) {
      this.cancelDeletePeriod();
      return;
    }
    if (this.editingPeriodId()) {
      this.cancelEdit();
      return;
    }
    this.close.emit();
  }

  // Estado del formulario de alta
  showCreateForm = signal<boolean>(false);
  name = signal<string>('');
  code = signal<string>('');
  startDate = signal<string>('');
  endDate = signal<string>('');

  isSubmitting = signal<boolean>(false);
  formError = signal<string | null>(null);

  toggleCreateForm(): void {
    this.showCreateForm.update(v => !v);
    this.formError.set(null);
  }

  submitNewPeriod(): void {
    if (!this.name().trim() || !this.code().trim() || !this.startDate() || !this.endDate()) {
      this.formError.set('Todos los campos son obligatorios.');
      return;
    }

    if (this.endDate() < this.startDate()) {
      this.formError.set('La fecha de fin debe ser posterior a la fecha de inicio.');
      return;
    }

    this.isSubmitting.set(true);
    this.formError.set(null);

    const dto: CreateAcademicPeriodDTO = {
      name: this.name().trim(),
      code: this.code().trim().toUpperCase(),
      start_date: this.startDate(),
      end_date: this.endDate()
    };

    this.coursesService.createPeriod(dto).subscribe({
      next: (created: AcademicPeriod) => {
        this.isSubmitting.set(false);
        this.name.set('');
        this.code.set('');
        this.startDate.set('');
        this.endDate.set('');
        this.showCreateForm.set(false);
        this.periodCreated.emit(created);
      },
      error: (err: { error?: { error?: string } }) => {
        this.isSubmitting.set(false);
        const msg = err.error?.error || 'No se pudo crear el periodo. Verifique que el código no exista.';
        this.formError.set(msg);
      }
    });
  }

  formatDate(isoStr?: string): string {
    if (!isoStr) return '';
    const datePart = isoStr.split('T')[0];
    const parts = datePart.split('-');
    if (parts.length === 3) {
      return `${parts[2]}/${parts[1]}/${parts[0]}`;
    }
    return datePart;
  }

  isPeriodExpired(period: AcademicPeriod): boolean {
    if (!period.end_date) return false;
    const datePart = period.end_date.split('T')[0];
    const endDate = new Date(datePart + 'T23:59:59');
    return endDate < new Date();
  }

  setActive(period: AcademicPeriod): void {
    if (period.is_active) return;
    if (this.isPeriodExpired(period)) {
      this.formError.set(`No se puede activar el periodo "${period.name}" porque su fecha de finalización ya expiró.`);
      return;
    }
    this.formError.set(null);

    this.coursesService.updatePeriod(period.id, { is_active: true }).subscribe({
      next: (updated: AcademicPeriod) => {
        this.periodActivated.emit(updated);
      },
      error: (err: any) => {
        const detail = err.error?.error || err.error?.message || err.statusText || 'Error de servidor';
        this.formError.set(`No se pudo activar el periodo "${period.name}": ${detail}`);
      }
    });
  }

  deactivatePeriod(period: AcademicPeriod): void {
    this.formError.set(null);
    this.coursesService.updatePeriod(period.id, { is_active: false }).subscribe({
      next: (updated: AcademicPeriod) => {
        this.periodActivated.emit(updated);
      },
      error: (err: any) => {
        const detail = err.error?.error || err.error?.message || 'Error al desactivar el periodo';
        this.formError.set(detail);
      }
    });
  }

  // Estado de edición de periodo existente
  editingPeriodId = signal<string | null>(null);
  editName = signal<string>('');
  editCode = signal<string>('');
  editStartDate = signal<string>('');
  editEndDate = signal<string>('');
  isSavingEdit = signal<boolean>(false);

  startEdit(period: AcademicPeriod): void {
    this.formError.set(null);
    this.editingPeriodId.set(period.id);
    this.editName.set(period.name);
    this.editCode.set(period.code);
    this.editStartDate.set(period.start_date ? period.start_date.split('T')[0] : '');
    this.editEndDate.set(period.end_date ? period.end_date.split('T')[0] : '');
  }

  cancelEdit(): void {
    this.editingPeriodId.set(null);
    this.formError.set(null);
  }

  saveEdit(period: AcademicPeriod): void {
    if (!this.editName().trim() || !this.editCode().trim() || !this.editStartDate() || !this.editEndDate()) {
      this.formError.set('Todos los campos son obligatorios.');
      return;
    }

    if (this.editEndDate() < this.editStartDate()) {
      this.formError.set('La fecha de conclusión debe ser posterior a la de inicio.');
      return;
    }

    this.isSavingEdit.set(true);
    this.formError.set(null);

    this.coursesService.updatePeriod(period.id, {
      name: this.editName().trim(),
      code: this.editCode().trim().toUpperCase(),
      start_date: this.editStartDate(),
      end_date: this.editEndDate()
    }).subscribe({
      next: (updated: AcademicPeriod) => {
        this.isSavingEdit.set(false);
        this.editingPeriodId.set(null);
        this.periodActivated.emit(updated);
      },
      error: (err: any) => {
        this.isSavingEdit.set(false);
        const detail = err.error?.error || err.error?.message || 'Error al actualizar el periodo.';
        this.formError.set(detail);
      }
    });
  }

  requestDeletePeriod(period: AcademicPeriod): void {
    if (period.is_active) {
      this.formError.set('No se puede eliminar el periodo actualmente activo.');
      return;
    }
    this.periodToDelete.set(period);
  }

  cancelDeletePeriod(): void {
    this.periodToDelete.set(null);
  }

  executeDeletePeriod(period: AcademicPeriod): void {
    this.coursesService.deletePeriod(period.id).subscribe({
      next: () => {
        this.cancelDeletePeriod();
        this.formError.set(null);
      },
      error: (err: any) => {
        this.cancelDeletePeriod();
        const detail = err.error?.error || err.error?.message || 'Tiene materias registradas asociadas';
        this.formError.set(`No se pudo eliminar el periodo: ${detail}`);
      }
    });
  }
}
