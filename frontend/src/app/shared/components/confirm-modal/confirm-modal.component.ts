import { Component, HostListener, input, output } from '@angular/core';
import { CommonModule } from '@angular/common';
import { LucideAlertTriangle, LucideHelpCircle, LucideLoader2, LucideX } from '@lucide/angular';

@Component({
  selector: 'confirm-modal',
  standalone: true,
  imports: [CommonModule, LucideAlertTriangle, LucideHelpCircle, LucideLoader2, LucideX],
  template: `
    <div class="modal-backdrop" (click)="onCancel()">
      <div
        class="modal-card confirm-modal-card"
        role="alertdialog"
        aria-modal="true"
        [attr.aria-label]="title()"
        (click)="$event.stopPropagation()">
        <div class="modal-header">
          <div class="header-info">
            <div class="header-icon-box" [class.danger-icon-box]="type() === 'danger'">
              @if (type() === 'danger') {
                <svg lucideAlertTriangle class="header-icon"></svg>
              } @else {
                <svg lucideHelpCircle class="header-icon"></svg>
              }
            </div>
            <div>
              <h2 class="modal-title">{{ title() }}</h2>
              <p class="modal-subtitle">Revise el detalle antes de continuar.</p>
            </div>
          </div>
          <button class="btn-close" (click)="onCancel()" title="Cerrar" aria-label="Cerrar">
            <svg lucideX class="close-icon"></svg>
          </button>
        </div>

        <div class="modal-body">
          <p class="confirm-message">{{ message() }}</p>
        </div>

        <div class="modal-footer">
          <button class="btn-secondary" (click)="onCancel()" [disabled]="isLoading()">
            {{ cancelText() }}
          </button>
          <button
            class="btn-confirm-action"
            [class.btn-danger]="type() === 'danger'"
            [class.btn-primary]="type() !== 'danger'"
            [disabled]="isLoading()"
            (click)="onConfirm()">
            @if (isLoading()) {
              <svg lucideLoader2 class="motion-spin"></svg>
            }
            {{ confirmText() }}
          </button>
        </div>
      </div>
    </div>
  `,
  styleUrl: './confirm-modal.component.scss',
})
export class ConfirmModalComponent {
  readonly title = input<string>('¿Confirma esta acción?');
  readonly message = input<string>('');
  readonly confirmText = input<string>('Confirmar');
  readonly cancelText = input<string>('Cancelar');
  readonly type = input<'danger' | 'primary'>('primary');
  readonly isLoading = input<boolean>(false);

  readonly confirm = output<void>();
  // 'cancel' es parte de la API pública de este modal: renombrarlo rompe los templates consumidores.
  // eslint-disable-next-line @angular-eslint/no-output-native
  readonly cancel = output<void>();

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
