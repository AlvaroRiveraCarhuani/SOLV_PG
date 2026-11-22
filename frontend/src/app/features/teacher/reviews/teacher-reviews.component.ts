import {
  ChangeDetectionStrategy,
  Component,
  DestroyRef,
  HostListener,
  OnDestroy,
  OnInit,
  computed,
  inject,
  signal
} from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import {
  LucideArrowRight,
  LucideCheckCircle2,
  LucideExternalLink,
  LucideListChecks,
  LucideRefreshCw
} from '@lucide/angular';
import { DateTextPipe } from '@shared/pipes/date-text.pipe';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { SkeletonLoaderComponent } from '@shared/components/skeleton/skeleton-loader.component';
import { SearchBarComponent } from '@shared/components/search-bar/search-bar.component';
import { ContextHelpService } from '@shared/help/context-help.service';
import { ProductTourService } from '@shared/help/product-tour.service';
import { InteractiveTutorialService } from '@shared/help/interactive-tutorial.service';
import { TeacherCourseService } from '../services/teacher-course.service';
import { TeacherDashboardService } from '../services/teacher-dashboard.service';
import { ReviewQueueGroup, SubmissionQueueItem } from '../models/teacher.models';
import {
  TeacherReviewsTutorialContext,
  buildTeacherReviewsTour,
  buildTeacherReviewsTutorial
} from './teacher-reviews-help.definitions';

type QueueSort = 'oldest' | 'newest' | 'name';
type WaitTone = 'ok' | 'warn' | 'late';

const DAY_MS = 86_400_000;

@Component({
  selector: 'teacher-reviews',
  standalone: true,
  imports: [
    CommonModule,
    RouterLink,
    DateTextPipe,
    MachineDataDirective,
    SkeletonLoaderComponent,
    SearchBarComponent,
    LucideArrowRight,
    LucideCheckCircle2,
    LucideExternalLink,
    LucideListChecks,
    LucideRefreshCw
  ],
  templateUrl: './teacher-reviews.component.html',
  styleUrl: './teacher-reviews.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class TeacherReviewsComponent implements OnInit, OnDestroy {
  private courseService = inject(TeacherCourseService);
  private dashboardService = inject(TeacherDashboardService);
  private route = inject(ActivatedRoute);
  private router = inject(Router);
  private destroyRef = inject(DestroyRef);
  private readonly help = inject(ContextHelpService);
  private readonly tour = inject(ProductTourService);
  private readonly tutorial = inject(InteractiveTutorialService);

  /** Materia preseleccionada desde el panel: filtra la cola en el servidor. */
  readonly subjectFilter = signal<string | null>(null);

  readonly groups = signal<ReviewQueueGroup[]>([]);
  readonly selectedGroup = signal<ReviewQueueGroup | null>(null);
  readonly submissions = signal<SubmissionQueueItem[]>([]);
  readonly isLoading = signal(true);
  readonly isLoadingSubmissions = signal(false);
  readonly error = signal<string | null>(null);

  /** Búsqueda y orden son locales: la cola del servidor ya llega acotada. */
  readonly searchTerm = signal('');
  readonly sortOrder = signal<QueueSort>('oldest');
  readonly highlightedIndex = signal(0);
  /** Hora del último refresco de la cola, mostrada en el pie. */
  readonly updatedAt = signal<string | null>(null);

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

  readonly subjectOptions = computed(() => {
    const byId = new Map<string, string>();
    for (const course of this.dashboardService.courses()) {
      byId.set(course.id, course.name);
    }
    for (const group of this.groups()) {
      if (!byId.has(group.subject_id)) {
        byId.set(group.subject_id, group.subject_name);
      }
    }
    return [...byId.entries()].map(([value, label]) => ({ value, label }));
  });

  /** Espera más larga de la cola, en días completos. */
  readonly maxWaitDays = computed(() => {
    const list = this.groups();
    if (list.length === 0) {
      return 0;
    }
    return Math.max(...list.map(group => this.daysFrom(group.oldest_submitted_at)));
  });

  readonly sortLabel = computed(() => {
    switch (this.sortOrder()) {
      case 'newest':
        return 'Más recientes';
      case 'name':
        return 'Nombre A–Z';
      default:
        return 'Más antiguas primero';
    }
  });

  readonly visibleSubmissions = computed(() => {
    const term = this.searchTerm().trim().toLowerCase();
    const filtered = term
      ? this.submissions().filter(
          item =>
            item.student_name.toLowerCase().includes(term) ||
            item.student_email.toLowerCase().includes(term)
        )
      : this.submissions();

    const order = this.sortOrder();
    return [...filtered].sort((a, b) => {
      if (order === 'name') {
        return a.student_name.localeCompare(b.student_name, 'es');
      }
      const byDate = a.submitted_at.localeCompare(b.submitted_at);
      return order === 'newest' ? -byDate : byDate;
    });
  });

  readonly gradedInSelection = computed(() => {
    const group = this.selectedGroup();
    if (!group) {
      return 0;
    }
    return Math.max(0, group.total - group.pending);
  });

  readonly tutorialContext = computed<TeacherReviewsTutorialContext>(() => ({
    hasGroups: () => this.groups().length > 0,
    hasSelectedGroup: () => this.selectedGroup() !== null,
    submissionsCount: () => this.submissions().length,
    isFilterActive: () => this.subjectFilter() !== null
  }));

  ngOnInit(): void {
    this.help.registerScreen('teacher-reviews');
    this.help.registerGuides('teacher-reviews', {
      tour: {
        label: 'Recorrido por la Cola de Entregas',
        hint: 'Explicación del agrupamiento, tiempos de espera y acceso a calificar',
        run: () => this.tour.start(buildTeacherReviewsTour())
      },
      tutorial: {
        label: 'Cómo despachar entregas en espera',
        hint: 'Paso a paso para priorizar por antigüedad y calificar con SpeedGrader',
        run: () => this.tutorial.start(buildTeacherReviewsTutorial(this.tutorialContext()))
      }
    });

    // Las materias alimentan el selector de la barra de filtros.
    this.dashboardService.getCourses().subscribe({ error: () => undefined });

    this.route.queryParamMap.pipe(takeUntilDestroyed(this.destroyRef)).subscribe(params => {
      this.subjectFilter.set(params.get('subject_id'));
      this.load();
    });
  }

  ngOnDestroy(): void {
    this.help.unregisterGuides('teacher-reviews');
    this.help.unregisterScreen('teacher-reviews');
    this.tutorial.exit();
  }

  /** Quitar el filtro vuelve a la cola completa del período. */
  clearFilter(): void {
    this.router.navigate([], { relativeTo: this.route, queryParams: {} });
  }

  /** Cambiar de materia vuelve a consultar la cola del servidor. */
  onSubjectSelect(event: Event): void {
    const value = (event.target as HTMLSelectElement).value;
    this.router.navigate([], {
      relativeTo: this.route,
      queryParams: value ? { subject_id: value } : {}
    });
  }

  onSortSelect(event: Event): void {
    const value = (event.target as HTMLSelectElement).value as QueueSort;
    this.sortOrder.set(value);
    this.highlightedIndex.set(0);
  }

  onSearch(term: string): void {
    this.searchTerm.set(term);
    this.highlightedIndex.set(0);
  }

  load(): void {
    this.isLoading.set(true);
    this.error.set(null);

    this.courseService.getReviewQueue(this.subjectFilter() ?? undefined).subscribe({
      next: (groups) => {
        this.groups.set(groups);
        this.isLoading.set(false);
        this.updatedAt.set(new Date().toISOString());

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
        this.error.set('No se pudo cargar la cola de revisión. Intente de nuevo.');
        this.isLoading.set(false);
      }
    });
  }

  select(group: ReviewQueueGroup): void {
    this.selectedGroup.set(group);
    this.isLoadingSubmissions.set(true);
    this.highlightedIndex.set(0);

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

  /** Abre la primera entrega pendiente de la práctica elegida. */
  openNextPending(): void {
    const next = this.visibleSubmissions()[0];
    if (next) {
      this.openReview(next);
    }
  }

  /** Atajos del pie: ↑ ↓ mover, Enter calificar. */
  @HostListener('document:keydown', ['$event'])
  onKeydown(event: KeyboardEvent): void {
    if (event.defaultPrevented || event.altKey || event.ctrlKey || event.metaKey) {
      return;
    }
    const target = event.target as HTMLElement | null;
    if (target?.closest('input, textarea, select, [contenteditable="true"]')) {
      return;
    }

    const list = this.visibleSubmissions();
    if (list.length === 0) {
      return;
    }

    if (event.key === 'ArrowDown') {
      event.preventDefault();
      this.highlightedIndex.update(index => Math.min(index + 1, list.length - 1));
      return;
    }
    if (event.key === 'ArrowUp') {
      event.preventDefault();
      this.highlightedIndex.update(index => Math.max(index - 1, 0));
      return;
    }
    if (event.key === 'Enter') {
      const item = list[this.highlightedIndex()];
      if (item) {
        event.preventDefault();
        this.openReview(item);
      }
    }
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

  openReview(item: SubmissionQueueItem): void {
    this.router.navigate(this.reviewLink(item));
  }

  /** Días completos transcurridos desde la entrega. */
  waitDays(item: SubmissionQueueItem): number {
    return this.daysFrom(item.submitted_at);
  }

  /** Semáforo de espera: verde < 3 d, ámbar 3–14 d, rojo > 14 d. */
  waitTone(item: SubmissionQueueItem): WaitTone {
    const days = this.waitDays(item);
    if (days < 3) {
      return 'ok';
    }
    return days <= 14 ? 'warn' : 'late';
  }

  initials(name: string): string {
    const parts = (name || '')
      .trim()
      .split(/\s+/)
      .filter(Boolean);
    if (parts.length === 0) {
      return '—';
    }
    const first = parts[0].charAt(0);
    const second = parts.length > 1 ? parts[1].charAt(0) : '';
    return `${first}${second}`.toUpperCase();
  }

  progressPercent(group: ReviewQueueGroup): number {
    if (!group.total) {
      return 0;
    }
    return Math.min(100, Math.round((group.pending / group.total) * 100));
  }

  private daysFrom(value: string): number {
    const timestamp = Date.parse(value);
    if (Number.isNaN(timestamp)) {
      return 0;
    }
    return Math.max(0, Math.floor((Date.now() - timestamp) / DAY_MS));
  }
}
