import { ChangeDetectionStrategy, Component, DestroyRef, OnInit, computed, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import {
  LucideArrowRight,
  LucideCheckCircle2,
  LucideClock,
  LucideListChecks
} from '@lucide/angular';
import { DateTextPipe } from '@shared/pipes/date-text.pipe';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { SkeletonLoaderComponent } from '@shared/components/skeleton/skeleton-loader.component';
import { TeacherCourseService } from '../services/teacher-course.service';
import { TeacherDashboardService } from '../services/teacher-dashboard.service';
import { ReviewQueueGroup, SubmissionQueueItem } from '../models/teacher.models';

@Component({
  selector: 'teacher-reviews',
  standalone: true,
  imports: [
    CommonModule,
    RouterLink,
    DateTextPipe,
    MachineDataDirective,
    SkeletonLoaderComponent,
    LucideArrowRight,
    LucideCheckCircle2,
    LucideClock,
    LucideListChecks
  ],
  templateUrl: './teacher-reviews.component.html',
  styleUrl: './teacher-reviews.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class TeacherReviewsComponent implements OnInit {
  private courseService = inject(TeacherCourseService);
  private dashboardService = inject(TeacherDashboardService);
  private route = inject(ActivatedRoute);
  private router = inject(Router);
  private destroyRef = inject(DestroyRef);

  /** Materia preseleccionada desde el panel: filtra la cola en el servidor. */
  readonly subjectFilter = signal<string | null>(null);

  readonly groups = signal<ReviewQueueGroup[]>([]);
  readonly selectedGroup = signal<ReviewQueueGroup | null>(null);
  readonly submissions = signal<SubmissionQueueItem[]>([]);
  readonly isLoading = signal(true);
  readonly isLoadingSubmissions = signal(false);
  readonly error = signal<string | null>(null);

  readonly totalPending = computed(() =>
    this.groups().reduce((total, group) => total + group.pending, 0)
  );

  readonly filterName = computed(() => {
    const subjectId = this.subjectFilter();
    if (!subjectId) {
      return '';
    }
    return (
      this.dashboardService.courses().find(course => course.id === subjectId)?.name ??
      this.groups()[0]?.subject_name ??
      'Materia elegida'
    );
  });

  ngOnInit(): void {
    this.route.queryParamMap.pipe(takeUntilDestroyed(this.destroyRef)).subscribe(params => {
      const subjectId = params.get('subject_id');
      this.subjectFilter.set(subjectId);
      if (subjectId) {
        // Solo el nombre: el filtro de la cola lo aplica el servidor.
        this.dashboardService.getCourses().subscribe({ error: () => undefined });
      }
      this.load();
    });
  }

  /** Quitar el filtro vuelve a la cola completa del período. */
  clearFilter(): void {
    this.router.navigate([], { relativeTo: this.route, queryParams: {} });
  }

  load(): void {
    this.isLoading.set(true);
    this.error.set(null);

    this.courseService.getReviewQueue(this.subjectFilter() ?? undefined).subscribe({
      next: (groups) => {
        this.groups.set(groups);
        this.isLoading.set(false);

        const current = this.selectedGroup();
        const stillThere = current
          ? groups.find((g) => g.subject_id === current.subject_id && g.exercise_id === current.exercise_id)
          : undefined;
        const next = stillThere ?? groups[0] ?? null;

        if (next) {
          this.select(next);
        } else {
          this.selectedGroup.set(null);
          this.submissions.set([]);
        }
      },
      error: () => {
        this.error.set('No se pudo cargar la cola de revisión. Intenta de nuevo.');
        this.isLoading.set(false);
      }
    });
  }

  select(group: ReviewQueueGroup): void {
    this.selectedGroup.set(group);
    this.isLoadingSubmissions.set(true);

    this.courseService.getCourseSubmissions(group.subject_id, { exercise_id: group.exercise_id }).subscribe({
      next: (items) => {
        // El backend devuelve todas las entregas de la práctica; la cola solo
        // muestra las que esperan corrección, de la más antigua a la más nueva.
        const pending = items
          .filter((item) => item.review_status === 'pending' || item.review_status === 'in_review')
          .sort((a, b) => a.submitted_at.localeCompare(b.submitted_at));
        this.submissions.set(pending);
        this.isLoadingSubmissions.set(false);
      },
      error: () => {
        this.submissions.set([]);
        this.isLoadingSubmissions.set(false);
      }
    });
  }

  trackGroup(_index: number, group: ReviewQueueGroup): string {
    return `${group.subject_id}:${group.exercise_id}`;
  }

  trackSubmission(_index: number, item: SubmissionQueueItem): string {
    return item.id;
  }

  isSelected(group: ReviewQueueGroup): boolean {
    const current = this.selectedGroup();
    return !!current && current.subject_id === group.subject_id && current.exercise_id === group.exercise_id;
  }

  reviewLink(item: SubmissionQueueItem): string[] {
    return ['/teacher/revision', item.id];
  }

  courseLink(group: ReviewQueueGroup): string[] {
    return ['/teacher/cursos', group.subject_id];
  }
}
