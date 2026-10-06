import { Component, OnInit, computed, inject, input, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink, ActivatedRoute } from '@angular/router';
import {
  LucideBarChart3,
  LucideArrowLeft,
  LucideRefreshCw,
  LucideClock,
  LucideTag,
  LucideAlertTriangle,
  LucideTrendingUp,
  LucideLayers
} from '@lucide/angular';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { TeacherCourseService } from '../../services/teacher-course.service';
import { CourseAnalyticsDTO } from '../../models/teacher.models';

@Component({
  selector: 'analytics-dashboard',
  standalone: true,
  imports: [
    CommonModule,
    RouterLink,
    LucideBarChart3,
    LucideArrowLeft,
    LucideRefreshCw,
    LucideClock,
    LucideTag,
    LucideAlertTriangle,
    LucideTrendingUp,
    LucideLayers,
    MachineDataDirective
  ],
  templateUrl: './analytics-dashboard.component.html',
  styleUrl: './analytics-dashboard.component.scss'
})
export class AnalyticsDashboardComponent implements OnInit {
  private courseService = inject(TeacherCourseService);
  private route = inject(ActivatedRoute);

  courseId = input<string>('');

  readonly analytics = signal<CourseAnalyticsDTO | null>(null);
  readonly isLoading = signal<boolean>(false);
  readonly error = signal<string | null>(null);
  readonly lastUpdated = signal<Date | null>(null);

  // Computeds
  readonly difficultyEasy = computed(() => {
    return this.analytics()?.difficulty_distribution?.['easy'] || { count: 0, success_rate: 0 };
  });

  readonly difficultyMedium = computed(() => {
    return this.analytics()?.difficulty_distribution?.['medium'] || { count: 0, success_rate: 0 };
  });

  readonly difficultyHard = computed(() => {
    return this.analytics()?.difficulty_distribution?.['hard'] || { count: 0, success_rate: 0 };
  });

  readonly totalExercisesWithDifficulty = computed(() => {
    const e = this.difficultyEasy().count;
    const m = this.difficultyMedium().count;
    const h = this.difficultyHard().count;
    return e + m + h;
  });

  readonly totalSubmissionsTimeline = computed(() => {
    const timeline = this.analytics()?.submissions_timeline || [];
    return timeline.reduce((sum, item) => sum + item.count, 0);
  });

  readonly maxTimelineCount = computed(() => {
    const timeline = this.analytics()?.submissions_timeline || [];
    const max = Math.max(...timeline.map(t => t.count), 0);
    return max > 0 ? max : 1;
  });

  ngOnInit(): void {
    this.loadAnalytics();
  }

  loadAnalytics(): void {
    const id = this.courseId() || this.route.snapshot.paramMap.get('courseId') || this.route.snapshot.paramMap.get('id') || '';
    if (!id) return;

    this.isLoading.set(true);
    this.error.set(null);

    this.courseService.getCourseAnalytics(id).subscribe({
      next: (data) => {
        this.analytics.set(data);
        this.isLoading.set(false);
        this.lastUpdated.set(new Date());
      },
      error: (err) => {
        this.error.set(err?.error?.message || err?.message || 'Error al cargar analíticas del curso');
        this.isLoading.set(false);
      }
    });
  }

  formatSeconds(seconds: number): string {
    if (!seconds || seconds <= 0) return '0s';
    if (seconds < 60) return `${seconds}s`;
    const mins = Math.floor(seconds / 60);
    const remainingSecs = seconds % 60;
    if (mins < 60) {
      return remainingSecs > 0 ? `${mins}m ${remainingSecs}s` : `${mins}m`;
    }
    const hours = Math.floor(mins / 60);
    const remainingMins = mins % 60;
    return `${hours}h ${remainingMins}m`;
  }

  formatPercentage(rate: number): string {
    if (rate === undefined || rate === null || isNaN(rate)) return '0.0%';
    return `${(rate * 100).toFixed(1)}%`;
  }

  getEffectiveCourseId(): string {
    return this.courseId() || this.route.snapshot.paramMap.get('courseId') || this.route.snapshot.paramMap.get('id') || '';
  }
}
