import { Component, input } from '@angular/core';
import { CommonModule } from '@angular/common';
import { KpiCardComponent, KpiGridComponent } from '@shared/components/kpi-card/kpi-card.component';
import { LucideUsers, LucideBookOpen, LucideClock } from '@lucide/angular';

export interface TeacherKpiCounts {
  all: number;
  active: number;
  noCourses: number;
  pending: number;
  expired: number;
}

@Component({
  selector: 'teacher-kpi-cards',
  standalone: true,
  imports: [CommonModule, KpiCardComponent, KpiGridComponent, LucideUsers, LucideBookOpen, LucideClock],
  templateUrl: './teacher-kpi-cards.component.html',
  styleUrl: './teacher-kpi-cards.component.scss'
})
export class TeacherKpiCardsComponent {
  readonly counts = input.required<TeacherKpiCounts>();
}
