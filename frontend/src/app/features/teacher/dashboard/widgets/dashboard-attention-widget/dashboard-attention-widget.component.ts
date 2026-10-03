import { Component, ChangeDetectionStrategy, input, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink } from '@angular/router';
import { 
  LucideCheckCircle2, 
  LucideFlame, 
  LucideShieldAlert, 
  LucideClock, 
  LucideArrowRight 
} from '@lucide/angular';
import { TeacherAttentionWidget } from '../../../models/teacher.models';
import { DateTextPipe } from '@shared/pipes/date-text.pipe';
import { SkeletonLoaderComponent } from '@shared/components/skeleton/skeleton-loader.component';

@Component({
  selector: 'dashboard-attention-widget',
  standalone: true,
  imports: [
    CommonModule,
    RouterLink,
    LucideCheckCircle2,
    LucideFlame,
    LucideShieldAlert,
    LucideClock,
    LucideArrowRight,
    DateTextPipe,
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

  readonly totalAlertsCount = computed(() => {
    return this.criticalCount() + this.warningCount() + this.standardCount();
  });
}
