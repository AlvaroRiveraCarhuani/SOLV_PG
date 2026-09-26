import { Component, EventEmitter, Input, Output, signal, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { AdminTemplateItem } from '../../../services/admin-templates.service';
import { ModalShellComponent } from '@shared/components/modal-shell/modal-shell.component';
import { FormFieldComponent } from '@shared/components/form-field/form-field.component';
import { LucideAlertTriangle, LucideXCircle } from '@lucide/angular';

@Component({
  selector: 'template-reject-modal',
  standalone: true,
  imports: [CommonModule, FormsModule, ModalShellComponent, FormFieldComponent, LucideAlertTriangle, LucideXCircle],
  templateUrl: './template-reject-modal.component.html',
  styleUrls: ['./template-reject-modal.component.scss']
})
export class TemplateRejectModalComponent {
  @Input({ required: true }) template!: AdminTemplateItem;
  @Output() rejected = new EventEmitter<{ id: string; reason: string }>();
  @Output() closed = new EventEmitter<void>();

  rejectionReason = signal<string>('');
  isSubmitting = signal<boolean>(false);

  charCount = computed(() => this.rejectionReason().trim().length);
  isValid = computed(() => this.charCount() >= 10);

  confirmReject(): void {
    if (!this.isValid() || this.isSubmitting()) return;
    this.isSubmitting.set(true);
    this.rejected.emit({
      id: this.template.id,
      reason: this.rejectionReason().trim()
    });
  }

  closeModal(): void {
    if (this.isSubmitting()) return;
    this.closed.emit();
  }
}
