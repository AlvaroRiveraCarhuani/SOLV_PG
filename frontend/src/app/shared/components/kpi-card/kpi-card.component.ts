import { Component, ChangeDetectionStrategy, input, computed } from '@angular/core';
import { CommonModule } from '@angular/common';

export type KpiVariant = 'default' | 'primary' | 'running' | 'warning' | 'danger' | 'purple' | 'blue' | 'green' | 'amber' | 'red';

@Component({
  selector: 'kpi-card',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './kpi-card.component.html',
  styleUrl: './kpi-card.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class KpiCardComponent {
  readonly label = input.required<string>();
  readonly value = input.required<string | number>();
  readonly detail = input<string>();
  readonly totalBadge = input<string>();
  readonly variant = input<KpiVariant>('default');

  readonly resolvedVariant = computed<string>(() => {
    const v = this.variant();
    switch (v) {
      case 'blue':
      case 'primary':
        return 'primary';
      case 'green':
      case 'running':
        return 'running';
      case 'amber':
      case 'warning':
        return 'warning';
      case 'red':
      case 'danger':
        return 'danger';
      case 'purple':
        return 'purple';
      default:
        return 'default';
    }
  });
}

@Component({
  selector: 'kpi-grid',
  standalone: true,
  imports: [CommonModule],
  template: `
    <div class="kpi-grid" [style.--kpi-grid-cols]="columns()">
      <ng-content />
    </div>
  `,
  styleUrl: './kpi-grid.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class KpiGridComponent {
  readonly columns = input<number>(4);
}
