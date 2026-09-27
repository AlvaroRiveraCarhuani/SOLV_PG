import { Component, EventEmitter, Input, Output, HostListener } from '@angular/core';
import { CommonModule } from '@angular/common';
import { LucideAlertTriangle, LucideHelpCircle, LucideX } from '@lucide/angular';

@Component({
  selector: 'confirm-modal',
  standalone: true,
  imports: [CommonModule, LucideAlertTriangle, LucideHelpCircle, LucideX],
  template: `
    <div class="modal-backdrop" (click)="onCancel()">
      <div class="modal-card confirm-modal-card" (click)="$event.stopPropagation()">
        <div class="modal-header">
          <div class="header-info">
            <div class="header-icon-box" [class.danger-icon-box]="type === 'danger'">
              @if (type === 'danger') {
                <svg lucideAlertTriangle class="header-icon"></svg>
              } @else {
                <svg lucideHelpCircle class="header-icon"></svg>
              }
            </div>
            <div>
              <h2 class="modal-title">{{ title }}</h2>
              <p class="modal-subtitle">Confirmación de seguridad</p>
            </div>
          </div>
          <button class="btn-close" (click)="onCancel()" title="Cerrar modal" aria-label="Cerrar">
            <svg lucideX class="close-icon"></svg>
          </button>
        </div>

        <div class="modal-body">
          <p class="confirm-message">{{ message }}</p>
        </div>

        <div class="modal-footer">
          <button class="btn-outline" (click)="onCancel()">
            {{ cancelText }}
          </button>
          <button 
            class="btn-confirm-action" 
            [class.btn-danger]="type === 'danger'"
            [class.btn-primary]="type !== 'danger'"
            (click)="onConfirm()">
            {{ confirmText }}
          </button>
        </div>
      </div>
    </div>
  `,
  styleUrl: './confirm-modal.component.scss',
})
export class ConfirmModalComponent {
  @Input() title = '¿Confirmar acción?';
  @Input() message = '';
  @Input() confirmText = 'Confirmar';
  @Input() cancelText = 'Cancelar';
  @Input() type: 'danger' | 'primary' = 'primary';

  @Output() confirm = new EventEmitter<void>();
  @Output() cancel = new EventEmitter<void>();

  @HostListener('document:keydown.escape')
  handleEscape(): void {
    this.onCancel();
  }

  onConfirm(): void {
    this.confirm.emit();
  }

  onCancel(): void {
    this.cancel.emit();
  }
}
