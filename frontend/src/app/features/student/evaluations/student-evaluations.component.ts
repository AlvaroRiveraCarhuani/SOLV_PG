import { ChangeDetectionStrategy, Component, OnInit, computed, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink } from '@angular/router';
import {
  LucideAlertTriangle,
  LucideCheckCircle2,
  LucideClock,
  LucideLoader2
} from '@lucide/angular';
import {
  StudentAcademicStatus,
  StudentEvaluationItem,
  StudentService
} from '@core/services/student.service';
import { KpiCardComponent, KpiGridComponent } from '@shared/components/kpi-card/kpi-card.component';
import { SearchBarComponent } from '@shared/components/search-bar/search-bar.component';
import { StatusTabItem, StatusTabsComponent } from '@shared/components/status-tabs/status-tabs.component';
import { ComboboxComponent, ComboboxOption } from '@shared/components/combobox/combobox.component';
import { VerdictBadgeComponent } from '@shared/components/verdict-badge/verdict-badge.component';
import { SkeletonLoaderComponent } from '@shared/components/skeleton/skeleton-loader.component';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { DateTextPipe } from '@shared/pipes/date-text.pipe';

export type AcademicStatusFilter = 'all' | StudentAcademicStatus;
export type ActivityPurposeFilter = 'all' | 'exam' | 'class';

const DIFFICULTY_LABELS: Record<string, string> = {
  easy: 'Fácil',
  medium: 'Medio',
  hard: 'Difícil'
};

const ACADEMIC_STATUS_LABELS: Record<StudentAcademicStatus, string> = {
  pending: 'Pendiente',
  in_progress: 'En progreso',
  completed: 'Completado',
  overdue: 'Vencido'
};

@Component({
  selector: 'student-evaluations',
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    CommonModule,
    RouterLink,
    LucideAlertTriangle,
    LucideCheckCircle2,
    LucideClock,
    LucideLoader2,
    KpiCardComponent,
    KpiGridComponent,
    SearchBarComponent,
    StatusTabsComponent,
    ComboboxComponent,
    VerdictBadgeComponent,
    SkeletonLoaderComponent,
    MachineDataDirective,
    DateTextPipe
  ],
  templateUrl: './student-evaluations.component.html',
  styleUrl: './student-evaluations.component.scss'
})
export class StudentEvaluationsComponent implements OnInit {
  private studentService = inject(StudentService);

  readonly isLoading = signal<boolean>(true);
  readonly statusFilter = signal<AcademicStatusFilter>('all');
  readonly subjectFilter = signal<string>('all');
  readonly purposeFilter = signal<ActivityPurposeFilter>('all');
  readonly searchQuery = signal<string>('');

  readonly evaluations = computed<StudentEvaluationItem[]>(() => this.studentService.evaluations());

  readonly counts = computed<Record<AcademicStatusFilter, number>>(() => {
    const items = this.evaluations();
    return {
      all: items.length,
      pending: items.filter((item) => item.academic_status === 'pending').length,
      in_progress: items.filter((item) => item.academic_status === 'in_progress').length,
      completed: items.filter((item) => item.academic_status === 'completed').length,
      overdue: items.filter((item) => item.academic_status === 'overdue').length
    };
  });

  readonly statusTabs = computed<StatusTabItem[]>(() => {
    const totals = this.counts();
    return [
      { id: 'all', label: 'Todas', count: totals.all },
      { id: 'pending', label: 'Pendientes', count: totals.pending },
      { id: 'in_progress', label: 'En progreso', count: totals.in_progress },
      { id: 'completed', label: 'Completadas', count: totals.completed },
      { id: 'overdue', label: 'Vencidas', count: totals.overdue }
    ];
  });

  readonly subjectOptions = computed<ComboboxOption[]>(() => {
    const labels = new Map<string, string>();
    for (const item of this.evaluations()) {
      if (!labels.has(item.subject_id)) {
        labels.set(item.subject_id, `${item.subject_code} — ${item.subject_name}`);
      }
    }
    return [
      { id: 'all', label: 'Todas las materias', value: 'all' },
      ...Array.from(labels, ([id, label]) => ({ id, label, value: id }))
    ];
  });

  readonly purposeOptions: ComboboxOption[] = [
    { id: 'all', label: 'Todas las actividades', value: 'all' },
    { id: 'exam', label: 'Evaluaciones calificadas', value: 'exam' },
    { id: 'class', label: 'Prácticas de clase', value: 'class' }
  ];

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
      return item.title.toLowerCase().includes(query);
    });
  });

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

  onStatusTabChange(tab: string): void {
    if (tab === 'all' || tab === 'pending' || tab === 'in_progress' || tab === 'completed' || tab === 'overdue') {
      this.statusFilter.set(tab);
    }
  }

  onPurposeChange(value: string): void {
    if (value === 'all' || value === 'exam' || value === 'class') {
      this.purposeFilter.set(value);
    }
  }

  difficultyLabel(difficulty?: string | null): string | null {
    if (!difficulty) {
      return null;
    }
    return DIFFICULTY_LABELS[difficulty.trim().toLowerCase()] ?? difficulty;
  }

  statusLabel(status: StudentAcademicStatus): string {
    return ACADEMIC_STATUS_LABELS[status] ?? 'Pendiente';
  }

  statusClass(status: StudentAcademicStatus): string {
    return `is-${status}`;
  }

  actionLabel(item: StudentEvaluationItem): string {
    switch (item.academic_status) {
      case 'in_progress':
        return 'Continuar';
      case 'completed':
        return 'Practicar de nuevo';
      default:
        return 'Resolver';
    }
  }
}
