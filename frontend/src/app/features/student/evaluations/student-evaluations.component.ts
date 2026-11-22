import { ChangeDetectionStrategy, Component, OnInit, computed, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink } from '@angular/router';
import {
  StudentAcademicStatus,
  StudentEvaluationItem,
  StudentService
} from '@core/services/student.service';
import { SearchBarComponent } from '@shared/components/search-bar/search-bar.component';
import { FilterPopoverComponent } from '@shared/components/filter-popover/filter-popover.component';
import { VerdictBadgeComponent } from '@shared/components/verdict-badge/verdict-badge.component';
import { SkeletonLoaderComponent } from '@shared/components/skeleton/skeleton-loader.component';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { DateTextPipe } from '@shared/pipes/date-text.pipe';
import {
  LucideCheck,
  LucideChevronDown,
  LucideChevronUp,
  LucideHistory,
  LucideInbox,
  LucideLaptop,
  LucideTerminal,
  LucideX
} from '@lucide/angular';

export type AcademicStatusFilter = 'all' | StudentAcademicStatus;
export type ActivityPurposeFilter = 'all' | 'exam' | 'class';

@Component({
  selector: 'student-evaluations',
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    CommonModule,
    RouterLink,
    SearchBarComponent,
    FilterPopoverComponent,
    VerdictBadgeComponent,
    SkeletonLoaderComponent,
    MachineDataDirective,
    DateTextPipe,
    LucideCheck,
    LucideChevronDown,
    LucideChevronUp,
    LucideHistory,
    LucideInbox,
    LucideLaptop,
    LucideTerminal,
    LucideX
  ],
  templateUrl: './student-evaluations.component.html',
  styleUrl: './student-evaluations.component.scss'
})
export class StudentEvaluationsComponent implements OnInit {
  private studentService = inject(StudentService);

  readonly isLoading = signal<boolean>(true);
  readonly searchQuery = signal<string>('');
  readonly statusFilter = signal<AcademicStatusFilter>('all');
  readonly subjectFilter = signal<string>('all');
  readonly purposeFilter = signal<ActivityPurposeFilter>('all');
  readonly completedExpanded = signal<boolean>(false);

  readonly evaluations = computed<StudentEvaluationItem[]>(() => this.studentService.evaluations());

  readonly totalActivities = computed(() => this.evaluations().length);

  readonly subjectsMap = computed(() => {
    const map = new Map<string, { code: string; name: string }>();
    for (const item of this.evaluations()) {
      if (!map.has(item.subject_id)) {
        map.set(item.subject_id, { code: item.subject_code, name: item.subject_name });
      }
    }
    return map;
  });

  readonly summaryStats = computed(() => {
    const list = this.evaluations();
    const exams = list.filter(item => item.purpose === 'exam' && item.academic_status !== 'completed').length;
    const inProgress = list.filter(item => item.academic_status === 'in_progress').length;
    const completed = list.filter(item => item.academic_status === 'completed').length;
    const examPlural = exams === 1 ? '1 examen próximo' : `${exams} exámenes próximos`;
    return `${examPlural} · ${inProgress} en curso · ${completed} completadas en todas sus materias.`;
  });

  readonly mostUrgentActivity = computed<{ label: string; exerciseId: string; title: string } | null>(() => {
    const list = this.evaluations().filter(item => item.academic_status !== 'completed');
    if (list.length === 0) return null;

    // Prioridad: examen con due_date más cercano o práctica en progreso
    const sorted = [...list].sort((a, b) => {
      if (a.purpose === 'exam' && b.purpose !== 'exam') return -1;
      if (b.purpose === 'exam' && a.purpose !== 'exam') return 1;
      if (a.due_date && b.due_date) {
        return new Date(a.due_date).getTime() - new Date(b.due_date).getTime();
      }
      if (a.due_date) return -1;
      if (b.due_date) return 1;
      return 0;
    });

    const top = sorted[0];
    return {
      label: `Continuar ${top.subject_code} →`,
      exerciseId: top.exercise_id,
      title: top.title
    };
  });

  readonly activeFilterCount = computed(() => {
    let count = 0;
    if (this.statusFilter() !== 'all') count++;
    if (this.subjectFilter() !== 'all') count++;
    if (this.purposeFilter() !== 'all') count++;
    return count;
  });

  readonly hasActiveFilters = computed(() => {
    return this.activeFilterCount() > 0 || this.searchQuery().trim().length > 0;
  });

  readonly filteredEvaluations = computed<StudentEvaluationItem[]>(() => {
    const status = this.statusFilter();
    const subject = this.subjectFilter();
    const purpose = this.purposeFilter();
    const query = this.searchQuery().trim().toLowerCase();

    return this.evaluations().filter((item) => {
      if (status !== 'all' && item.academic_status !== status) {
        return false;
      }
      if (subject !== 'all' && item.subject_id !== subject) {
        return false;
      }
      if (purpose !== 'all' && item.purpose !== purpose) {
        return false;
      }
      if (!query) {
        return true;
      }
      return (
        item.title.toLowerCase().includes(query) ||
        item.subject_name.toLowerCase().includes(query) ||
        item.subject_code.toLowerCase().includes(query)
      );
    });
  });

  readonly filteredCount = computed(() => this.filteredEvaluations().length);

  // Jerarquía en 3 bloques
  readonly upcomingAndDue = computed(() => {
    return this.filteredEvaluations()
      .filter(item => item.academic_status !== 'completed' && (item.due_date || item.purpose === 'exam'))
      .sort((a, b) => {
        if (!a.due_date) return 1;
        if (!b.due_date) return -1;
        return new Date(a.due_date).getTime() - new Date(b.due_date).getTime();
      });
  });

  readonly openPractices = computed(() => {
    return this.filteredEvaluations().filter(
      item => item.academic_status !== 'completed' && !item.due_date && item.purpose !== 'exam'
    );
  });

  readonly completedActivities = computed(() => {
    return this.filteredEvaluations().filter(item => item.academic_status === 'completed');
  });

  readonly isCompletedSectionOpen = computed(() => {
    return this.completedExpanded() || this.statusFilter() === 'completed';
  });

  toggleCompleted(): void {
    this.completedExpanded.update(v => !v);
  }

  clearAllFilters(): void {
    this.statusFilter.set('all');
    this.subjectFilter.set('all');
    this.purposeFilter.set('all');
    this.searchQuery.set('');
  }

  setStatus(status: AcademicStatusFilter): void {
    this.statusFilter.set(status);
  }

  setSubject(subjectId: string): void {
    this.subjectFilter.set(subjectId);
  }

  setPurpose(purpose: ActivityPurposeFilter): void {
    this.purposeFilter.set(purpose);
  }

  getUrgencyLabel(dueDateStr?: string | null): string {
    if (!dueDateStr) return '';
    const now = new Date();
    const due = new Date(dueDateStr);
    const diffMs = due.getTime() - now.getTime();
    if (diffMs < 0) return 'Vencida';
    const diffHours = diffMs / (1000 * 60 * 60);
    if (diffHours <= 24) return 'Hoy';
    if (diffHours <= 48) return 'Mañana';
    const days = Math.ceil(diffHours / 24);
    return `En ${days} días`;
  }

  statusLabel(status: StudentAcademicStatus | 'all'): string {
    if (status === 'all') return 'Todos';
    switch (status) {
      case 'completed': return 'Cumplida';
      case 'in_progress': return 'En curso';
      case 'pending': return 'Pendiente';
      case 'overdue': return 'Vencida';
      default: return 'Disponible';
    }
  }

  isIdeExercise(item: StudentEvaluationItem): boolean {
    return item.environment_type === 'IDE_PERSISTENTE' || item.type === 'project';
  }

  actionLabel(item: StudentEvaluationItem): string {
    switch (item.academic_status) {
      case 'in_progress':
        return 'Continuar';
      case 'completed':
        return 'Ver entrega →';
      default:
        return 'Resolver';
    }
  }

  ngOnInit(): void {
    if (this.studentService.evaluations().length > 0) {
      this.isLoading.set(false);
      return;
    }
    void this.loadEvaluations();
  }

  private async loadEvaluations(): Promise<void> {
    try {
      await this.studentService.loadEvaluations();
    } finally {
      this.isLoading.set(false);
    }
  }
}
