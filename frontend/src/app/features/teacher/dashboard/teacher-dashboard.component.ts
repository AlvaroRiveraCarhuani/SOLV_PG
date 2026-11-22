import { Component, OnDestroy, OnInit, computed, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Router, RouterLink } from '@angular/router';
import { LucideArrowRight } from '@lucide/angular';

import { ComboboxComponent, ComboboxOption } from '@shared/components/combobox/combobox.component';
import { SkeletonLoaderComponent } from '@shared/components/skeleton/skeleton-loader.component';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { DateTextPipe } from '@shared/pipes/date-text.pipe';

import { TeacherDashboardService } from '../services/teacher-dashboard.service';
import { TeacherCourseService } from '../services/teacher-course.service';
import { TeacherExerciseBankItem } from '../models/teacher.models';
import { SubjectCardComponent } from './subject-card/subject-card.component';

/** El estado académico cambia en días y el trabajo pendiente en horas: un minuto alcanza. */
const REFRESH_MS = 60_000;

/**
 * Pantalla "Hoy": responde "¿qué me espera hoy?" con el resumen del servidor.
 *
 * Dos superficies y una jerarquía: el canvas de materias, que se reconoce por
 * posición, y la columna de pendientes, que es la única lista accionable. No hay
 * datos de ejecución: los contenedores se miran en el aula, donde existe una
 * acción para el docente.
 */
@Component({
  selector: 'teacher-dashboard',
  standalone: true,
  imports: [
    CommonModule,
    RouterLink,
    ComboboxComponent,
    SkeletonLoaderComponent,
    MachineDataDirective,
    DateTextPipe,
    LucideArrowRight,
    SubjectCardComponent
  ],
  templateUrl: './teacher-dashboard.component.html',
  styleUrl: './teacher-dashboard.component.scss'
})
export class TeacherDashboardComponent implements OnInit, OnDestroy {
  private readonly dashboardService = inject(TeacherDashboardService);
  private readonly courseService = inject(TeacherCourseService);
  private readonly router = inject(Router);

  readonly overview = this.dashboardService.overview;
  readonly periods = this.dashboardService.periods;
  readonly isLoading = this.dashboardService.isLoading;

  readonly selectedPeriodId = signal<string>('');
  readonly loadError = signal<string | null>(null);

  /** Acceso rápido al aula: materia, práctica publicada y acción explícita. */
  readonly quickSubjectId = signal<string>('');
  readonly quickExerciseId = signal<string>('');
  readonly exerciseBank = signal<TeacherExerciseBankItem[]>([]);

  readonly today = new Date();
  readonly todayIso = this.today.toISOString().slice(0, 10);

  private refreshTimer?: ReturnType<typeof setInterval>;

  readonly subjects = computed(() => this.overview()?.subjects ?? []);
  readonly session = computed(() => this.overview()?.session ?? null);
  readonly dueSoon = computed(() => this.overview()?.due_soon ?? []);
  readonly dueSoonHidden = computed(() => this.overview()?.due_soon_hidden ?? 0);
  readonly queue = computed(
    () => this.overview()?.queue ?? { pending: 0, practices: 0, oldest_submitted_at: null }
  );

  readonly periodOptions = computed<ComboboxOption[]>(() =>
    this.periods().map(period => ({
      id: period.id,
      label: `${period.name} · ${period.code}${period.is_active ? ' (activo)' : ''}`,
      value: period.id
    }))
  );

  /** La columna solo existe con trabajo que accionar; sin ella el canvas ocupa el ancho. */
  /** Prácticas con entregas pendientes: prioridad de corrección del día. */
  readonly reviewPriorities = computed(() =>
    this.dueSoon().filter(due => due.pending_review > 0)
  );

  readonly subjectOptions = computed<ComboboxOption[]>(() =>
    this.subjects().map(subject => ({
      id: subject.id,
      label: `${subject.code} · ${subject.name}`,
      value: subject.id
    }))
  );

  readonly quickPracticeOptions = computed<ComboboxOption[]>(() => {
    const subjectId = this.quickSubjectId();
    if (!subjectId) {
      return [];
    }
    return this.exerciseBank()
      .filter(item => item.subject_id === subjectId && item.status === 'published')
      .map(item => ({ id: item.id, label: item.title, value: item.id }));
  });

  readonly showSkeleton = computed(() => this.isLoading() && this.overview() === null);

  readonly showEmpty = computed(() => {
    const current = this.overview();
    return current !== null && current.subjects.length === 0 && this.loadError() === null;
  });

  ngOnInit(): void {
    this.loadPeriods();
    this.loadExerciseBank();
    this.startRefresh();
  }

  onQuickSubjectChange(subjectId: string): void {
    this.quickSubjectId.set(subjectId);
    this.quickExerciseId.set('');
  }

  onQuickExerciseChange(exerciseId: string): void {
    this.quickExerciseId.set(exerciseId);
  }

  /** Abre el aula con el contexto ya resuelto; nunca adivina la práctica. */
  openAula(): void {
    const subjectId = this.quickSubjectId();
    if (!subjectId) {
      return;
    }
    const exerciseId = this.quickExerciseId();
    this.router.navigate(['/teacher/aula'], {
      queryParams: exerciseId ? { subject_id: subjectId, exercise_id: exerciseId } : { subject_id: subjectId }
    });
  }

  private loadExerciseBank(): void {
    this.courseService.getExerciseBank().subscribe({
      next: bank => this.exerciseBank.set(bank),
      error: () => this.exerciseBank.set([])
    });
  }

  ngOnDestroy(): void {
    this.stopRefresh();
  }

  onPeriodChange(periodId: string): void {
    this.selectedPeriodId.set(periodId);
    this.loadOverview();
  }

  reload(): void {
    this.loadError.set(null);
    this.loadOverview();
  }

  private loadPeriods(): void {
    this.dashboardService.getAcademicPeriods().subscribe({
      next: periods => {
        const active = periods.find(period => period.is_active) ?? periods[0];
        if (active) {
          this.selectedPeriodId.set(active.id);
        }
        this.loadOverview();
      },
      error: () => this.loadOverview()
    });
  }

  /** Un error de red conserva el resumen ya cargado: el docente no pierde su mapa. */
  private loadOverview(): void {
    this.dashboardService.loadOverview(this.selectedPeriodId() || undefined).subscribe({
      next: () => this.loadError.set(null),
      error: () => this.loadError.set('No se pudo actualizar el panel. Revise su conexión e intente de nuevo.')
    });
  }

  /** El refresco no mueve el foco y se detiene con la pestaña oculta. */
  private startRefresh(): void {
    this.stopRefresh();
    this.refreshTimer = setInterval(() => {
      if (typeof document !== 'undefined' && document.hidden) {
        return;
      }
      this.loadOverview();
    }, REFRESH_MS);
  }

  private stopRefresh(): void {
    if (this.refreshTimer) {
      clearInterval(this.refreshTimer);
      this.refreshTimer = undefined;
    }
  }
}
