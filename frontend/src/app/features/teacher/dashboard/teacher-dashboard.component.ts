import { Component, OnDestroy, OnInit, computed, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink } from '@angular/router';

import { ComboboxComponent, ComboboxOption } from '@shared/components/combobox/combobox.component';
import { SkeletonLoaderComponent } from '@shared/components/skeleton/skeleton-loader.component';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { DateTextPipe } from '@shared/pipes/date-text.pipe';

import { TeacherDashboardService } from '../services/teacher-dashboard.service';
import { PendingColumnComponent } from './pending-column/pending-column.component';
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
    SubjectCardComponent,
    PendingColumnComponent
  ],
  templateUrl: './teacher-dashboard.component.html',
  styleUrl: './teacher-dashboard.component.scss'
})
export class TeacherDashboardComponent implements OnInit, OnDestroy {
  private readonly dashboardService = inject(TeacherDashboardService);

  readonly overview = this.dashboardService.overview;
  readonly periods = this.dashboardService.periods;
  readonly isLoading = this.dashboardService.isLoading;

  readonly selectedPeriodId = signal<string>('');
  readonly loadError = signal<string | null>(null);

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
  readonly hasPending = computed(() => {
    const current = this.overview();
    if (!current) {
      return false;
    }
    return current.session !== null || current.due_soon.length > 0 || current.queue.pending > 0;
  });

  readonly showSkeleton = computed(() => this.isLoading() && this.overview() === null);

  readonly showEmpty = computed(() => {
    const current = this.overview();
    return current !== null && current.subjects.length === 0 && this.loadError() === null;
  });

  ngOnInit(): void {
    this.loadPeriods();
    this.startRefresh();
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
      error: () => this.loadError.set('No se pudo actualizar el panel. Revisa tu conexión e intenta de nuevo.')
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
