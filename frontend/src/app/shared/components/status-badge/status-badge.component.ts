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
  styleUrl: './status-badge.component.scss',
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
