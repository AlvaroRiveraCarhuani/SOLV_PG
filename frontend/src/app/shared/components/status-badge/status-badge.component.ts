import { Component, input, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { WorkspaceStatus } from '@core/models/workspace.model';

@Component({
  selector: 'status-badge',
  standalone: true,
  imports: [CommonModule],
  template: `
    <span class="status-badge" [ngClass]="badgeClass()">
      <span class="status-indicator"></span>
      <span class="status-label">{{ label() }}</span>
    </span>
  `,
  styles: [`
    .status-badge {
      display: inline-flex;
      align-items: center;
      gap: var(--space-1-5, 6px);
      padding: var(--space-1, 4px) var(--space-2-5, 10px);
      border-radius: var(--radius-full, 9999px);
      font-size: var(--font-size-xs, 12px);
      font-weight: 500;
      line-height: 1;
      border: 1px solid transparent;
      transition: all var(--transition-fast, 150ms ease);
    }

    .status-indicator {
      width: 6px;
      height: 6px;
      border-radius: 50%;
    }

    /* running = success (#16A34A) */
    .status-running {
      background-color: var(--state-running-bg, #F0FDF4);
      color: var(--state-running-text, #15803D);
      border-color: var(--state-running-border, #BBF7D0);

      .status-indicator {
        background-color: var(--state-running, #16A34A);
        box-shadow: 0 0 0 2px var(--state-running-border, #BBF7D0);
        animation: pulse-indicator 2s cubic-bezier(0.4, 0, 0.6, 1) infinite;
      }
    }

    /* hibernated = neutral (#64748B) */
    .status-hibernated {
      background-color: var(--state-hibernated-bg, #F8FAFC);
      color: var(--state-hibernated-text, #475569);
      border-color: var(--state-hibernated-border, #E2E8F0);

      .status-indicator {
        background-color: var(--state-hibernated, #64748B);
      }
    }

    /* pending = warning (#D97706) */
    .status-pending {
      background-color: var(--state-pending-bg, #FFFBEB);
      color: var(--state-pending-text, #B45309);
      border-color: var(--state-pending-border, #FDE68A);

      .status-indicator {
        background-color: var(--state-pending, #D97706);
      }
    }

    /* failed / oom_killed = error (#DC2626) */
    .status-failed,
    .status-oom-killed {
      background-color: var(--state-failed-bg, #FEF2F2);
      color: var(--state-failed-text, #B91C1C);
      border-color: var(--state-failed-border, #FECACA);

      .status-indicator {
        background-color: var(--state-failed, #DC2626);
      }
    }

    @keyframes pulse-indicator {
      0%, 100% {
        opacity: 1;
      }
      50% {
        opacity: 0.4;
      }
    }
  `]
})
export class StatusBadgeComponent {
  status = input.required<WorkspaceStatus | string>();

  badgeClass = computed(() => {
    const s = this.status().toLowerCase();
    switch (s) {
      case 'running':
        return 'status-running';
      case 'hibernated':
        return 'status-hibernated';
      case 'pending':
        return 'status-pending';
      case 'oom_killed':
        return 'status-oom-killed';
      case 'failed':
      default:
        return 'status-failed';
    }
  });

  label = computed(() => {
    const s = this.status().toLowerCase();
    switch (s) {
      case 'running':
        return 'Activo';
      case 'hibernated':
        return 'En Pausa';
      case 'pending':
        return 'Iniciando';
      case 'oom_killed':
        return 'Límite de Memoria';
      case 'failed':
      default:
        return 'Error';
    }
  });
}
