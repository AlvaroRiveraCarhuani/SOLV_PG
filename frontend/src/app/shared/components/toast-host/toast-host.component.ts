import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import {
  LucideAlertCircle,
  LucideCheckCircle2,
  LucideInfo,
  LucideX
} from '@lucide/angular';
import { ToastService } from '@shared/services/toast.service';

/**
 * Punto único de renderizado del toast global. Se monta una sola vez en el
 * layout para que cualquier aviso sobreviva a la navegación entre vistas.
 */
@Component({
  selector: 'toast-host',
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [LucideAlertCircle, LucideCheckCircle2, LucideInfo, LucideX],
  template: `
    @if (toast(); as t) {
      <div
        class="toast-notification"
        [class.toast-error]="t.type === 'error'"
        role="status"
        aria-live="polite"
      >
        @if (t.type === 'error') {
          <svg lucideAlertCircle class="toast-icon"></svg>
        } @else if (t.type === 'info') {
          <svg lucideInfo class="toast-icon"></svg>
        } @else {
          <svg lucideCheckCircle2 class="toast-icon"></svg>
        }
        <span>{{ t.message }}</span>
        <button
          type="button"
          class="toast-dismiss"
          (click)="dismiss(t.id)"
          aria-label="Cerrar aviso"
        >
          <svg lucideX class="toast-icon"></svg>
        </button>
      </div>
    }
  `,
  styleUrl: './toast-host.component.scss'
})
export class ToastHostComponent {
  private readonly toastService = inject(ToastService);

  readonly toast = this.toastService.toast;

  dismiss(id: number): void {
    this.toastService.dismiss(id);
  }
}
