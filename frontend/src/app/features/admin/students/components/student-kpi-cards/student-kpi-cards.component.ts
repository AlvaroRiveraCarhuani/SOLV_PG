import { Component, input } from '@angular/core';
import { CommonModule } from '@angular/common';
import { KpiCardComponent, KpiGridComponent } from '@shared/components/kpi-card/kpi-card.component';
import { LucideUsers, LucideCpu, LucideAlertTriangle, LucideShieldAlert } from '@lucide/angular';

@Component({
  selector: 'student-kpi-cards',
  standalone: true,
  imports: [
    CommonModule,
    KpiCardComponent,
    KpiGridComponent,
    LucideUsers,
    LucideCpu,
    LucideAlertTriangle,
    LucideShieldAlert
  ],
  templateUrl: './student-kpi-cards.component.html',
  styleUrls: ['./student-kpi-cards.component.scss']
})
export class StudentKpiCardsComponent {
  readonly enrolledStudentsCount = input<number>(0);
  readonly totalStudents = input<number>(0);
  readonly activeWorkspacesTotal = input<number>(0);
  readonly strikesStudentsCount = input<number>(0);
  readonly blockedStudentsCount = input<number>(0);
}
