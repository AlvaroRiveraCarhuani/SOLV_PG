import { Component, ChangeDetectionStrategy, input, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink } from '@angular/router';
import { 
  LucideCheckCircle2, 
  LucideAlertTriangle, 
  LucideShieldAlert, 
  LucideClock, 
  LucideArrowRight 
} from '@lucide/angular';
import { TeacherAttentionWidget } from '../../../models/teacher.models';
import { DateTextPipe } from '@shared/pipes/date-text.pipe';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { SkeletonLoaderComponent } from '@shared/components/skeleton/skeleton-loader.component';

export interface GroupedPendingReview {
  exercise_title: string;
  count: number;
  first_submission_id: string;
  latest_submitted_at: string;
}

@Component({
  selector: 'dashboard-attention-widget',
  standalone: true,
  imports: [
    CommonModule,
    RouterLink,
    LucideCheckCircle2,
    LucideAlertTriangle,
    LucideShieldAlert,
    LucideClock,
    LucideArrowRight,
    DateTextPipe,
    MachineDataDirective,
    SkeletonLoaderComponent
  ],
  templateUrl: './dashboard-attention-widget.component.html',
  styleUrl: './dashboard-attention-widget.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class DashboardAttentionWidgetComponent {
  readonly attention = input<TeacherAttentionWidget | null>(null);
  readonly isLoading = input<boolean>(false);

  readonly criticalCount = computed(() => this.attention()?.critical?.length ?? 0);
  readonly warningCount = computed(() => this.attention()?.warning?.length ?? 0);
  readonly standardCount = computed(() => this.attention()?.standard?.length ?? 0);

  readonly visibleCritical = computed(() => (this.attention()?.critical || []).slice(0, 3));
  readonly visibleWarning = computed(() => (this.attention()?.warning || []).slice(0, 2));

  readonly groupedPendingReviews = computed<GroupedPendingReview[]>(() => {
    const list = this.attention()?.standard || [];
    const map = new Map<string, GroupedPendingReview>();
    for (const item of list) {
      const title = item.exercise_title || 'Práctica sin título';
      const existing = map.get(title);
      if (!existing) {
        map.set(title, {
          exercise_title: title,
          count: 1,
          first_submission_id: item.submission_id,
          latest_submitted_at: item.submitted_at
        });
      } else {
        existing.count++;
        if (new Date(item.submitted_at) > new Date(existing.latest_submitted_at)) {
          existing.latest_submitted_at = item.submitted_at;
        }
      }
    }
    return Array.from(map.values());
  });

  readonly visibleGroupedReviews = computed<GroupedPendingReview[]>(() => {
    return this.groupedPendingReviews().slice(0, 4);
  });

  readonly totalAlertsCount = computed(() => {
    return this.visibleCritical().length + this.visibleWarning().length + this.groupedPendingReviews().length;
  });
}
