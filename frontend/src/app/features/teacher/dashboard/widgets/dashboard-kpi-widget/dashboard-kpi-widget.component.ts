import { Component, ChangeDetectionStrategy, input } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';

@Component({
  selector: 'dashboard-kpi-widget',
  standalone: true,
  imports: [CommonModule, MachineDataDirective],
  templateUrl: './dashboard-kpi-widget.component.html',
  styleUrl: './dashboard-kpi-widget.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class DashboardKpiWidgetComponent {
  readonly totalStudents = input<number>(0);
  readonly activeNow = input<number>(0);
  readonly pendingReviews = input<number>(0);
  readonly atRisk = input<number>(0);
}
