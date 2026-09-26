import { Component, EventEmitter, Input, Output, signal, computed, HostListener } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { AdminTemplateItem } from '../../../services/admin-templates.service';
import { LucideX, LucideAlertCircle, LucidePlay } from '@lucide/angular';

@Component({
  selector: 'template-status-modal',
  standalone: true,
  imports: [CommonModule, FormsModule, LucideX, LucideAlertCircle, LucidePlay],
  templateUrl: './template-status-modal.component.html',
  styleUrls: ['./template-status-modal.component.scss']
})
export class TemplateStatusModalComponent {
  @Input({ required: true }) template!: AdminTemplateItem;
  @Output() confirmed = new EventEmitter<{ template: AdminTemplateItem; reason: string }>();
  @Output() closed = new EventEmitter<void>();

  reason = signal<string>('');
  isSubmitting = signal<boolean>(false);

  isPausing = computed(() => {
    return this.template.status === 'approved' || this.template.status === 'APROBADA';
  });

  isValid = computed(() => {
    if (!this.isPausing()) return true;
    return this.reason().trim().length >= 10;
  });

  @HostListener('document:keydown.escape')
  onEscape(): void {
    this.closeModal();
  }

  confirm(): void {
    if (!this.isValid() || this.isSubmitting()) return;
    this.isSubmitting.set(true);
    this.confirmed.emit({
      template: this.template,
      reason: this.reason().trim()
    });
  }

  closeModal(): void {
    if (this.isSubmitting()) return;
    this.closed.emit();
  }
}
