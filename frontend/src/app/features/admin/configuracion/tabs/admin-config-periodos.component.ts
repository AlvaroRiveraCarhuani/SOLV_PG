import { Component, inject, signal, computed, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { HttpClient } from '@angular/common/http';
import { formatSolvDate } from '@shared/pipes/date-text.pipe';
import { Observable, forkJoin, map, catchError, of } from 'rxjs';
import {
  AdminConfigPeriodosService,
  AcademicPeriodConfig,
  CreatePeriodPayload,
  periodLifecycle,
  PeriodLifecycle
} from '../admin-config-periodos.service';
import { KpiCardComponent, KpiGridComponent } from '@shared/components/kpi-card/kpi-card.component';
import { ModalShellComponent } from '@shared/components/modal-shell/modal-shell.component';
import { FormFieldComponent } from '@shared/components/form-field/form-field.component';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { PaginationBarComponent } from '@shared/components/pagination-bar/pagination-bar.component';
import {
  LucideCalendar,
  LucidePlus,
  LucideArchive,
  LucidePencil,
  LucideTrash2,
  LucideAlertTriangle,
  LucideCheckCircle2,
  LucideAlertCircle,
  LucideRocket
} from '@lucide/angular';

interface PeriodRow extends AcademicPeriodConfig {
  lifecycle: PeriodLifecycle;
}

interface SubjectCountRow {
  academic_period_id?: string;
}

@Component({
  selector: 'admin-config-periodos',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    MachineDataDirective,
    KpiCardComponent,
    KpiGridComponent,
    ModalShellComponent,
    FormFieldComponent,
    PaginationBarComponent,
    LucideCalendar,
    LucidePlus,
    LucideArchive,
    LucidePencil,
    LucideTrash2,
    LucideAlertTriangle,
    LucideCheckCircle2,
    LucideAlertCircle,
    LucideRocket
  ],
  templateUrl: './admin-config-periodos.component.html',
  styleUrls: ['./admin-config-periodos.component.scss']
})
export class AdminConfigPeriodosComponent implements OnInit {
  private readonly periodosService = inject(AdminConfigPeriodosService);
  private readonly http = inject(HttpClient);

  readonly periods = this.periodosService.periods;
  readonly isLoading = this.periodosService.isLoading;
  readonly loadError = this.periodosService.error;

  readonly rows = computed<PeriodRow[]>(() =>
    [...this.periods()]
      .map((p) => ({ ...p, lifecycle: periodLifecycle(p) }))
      .sort((a, b) => {
        const rank = (l: PeriodLifecycle) => (l === 'activo' ? 0 : l === 'proximo' ? 1 : 2);
        return rank(a.lifecycle) - rank(b.lifecycle) || b.start_date.localeCompare(a.start_date);
      })
  );

  readonly activeCount = computed(() => this.rows().filter((r) => r.lifecycle === 'activo').length);
  readonly nextUpCount = computed(() => this.rows().filter((r) => r.lifecycle === 'proximo').length);
  readonly archivedCount = computed(() => this.rows().filter((r) => r.lifecycle === 'archivado').length);

  // ------------------------------------------------------------------
  // Paginación (estilo Docentes: client-side, 10 por página)
  // ------------------------------------------------------------------
  readonly currentPage = signal(1);
  readonly pageSize = 10;

  readonly paginatedRows = computed(() => {
    const start = (this.currentPage() - 1) * this.pageSize;
    return this.rows().slice(start, start + this.pageSize);
  });

  readonly totalPages = computed(() => Math.max(1, Math.ceil(this.rows().length / this.pageSize)));

  readonly paginationDisplay = computed(() => {
    const total = this.rows().length;
    if (total === 0) return { from: 0, to: 0 };
    const from = (this.currentPage() - 1) * this.pageSize + 1;
    const to = Math.min(this.currentPage() * this.pageSize, total);
    return { from, to };
  });

  goToPage(page: number): void {
    if (page >= 1 && page <= this.totalPages()) {
      this.currentPage.set(page);
    }
  }

  // ------------------------------------------------------------------
  // Toast
  // ------------------------------------------------------------------
  readonly toast = signal<{ type: 'success' | 'error'; message: string } | null>(null);
  private toastTimer?: ReturnType<typeof setTimeout>;

  showToast(message: string, type: 'success' | 'error' = 'success'): void {
    this.toast.set({ message, type });
    if (this.toastTimer) clearTimeout(this.toastTimer);
    this.toastTimer = setTimeout(() => this.toast.set(null), 4000);
  }

  // ------------------------------------------------------------------
  // Alta / edición
  // ------------------------------------------------------------------
  readonly formOpen = signal(false);
  readonly editingId = signal<string | null>(null);
  readonly name = signal('');
  readonly code = signal('');
  readonly startDate = signal('');
  readonly endDate = signal('');
  readonly activateNow = signal(false);
  readonly isSubmitting = signal(false);
  readonly formError = signal<string | null>(null);

  readonly isFormValid = computed(
    () =>
      this.name().trim().length > 0 &&
      this.code().trim().length > 0 &&
      this.startDate() !== '' &&
      this.endDate() !== '' &&
      this.endDate() >= this.startDate()
  );

  readonly formTitle = computed(() =>
    this.editingId() ? 'Editar Período Académico' : 'Nuevo Período Académico'
  );

  openCreate(): void {
    this.editingId.set(null);
    this.name.set('');
    this.code.set('');
    this.startDate.set('');
    this.endDate.set('');
    this.activateNow.set(false);
    this.formError.set(null);
    this.formOpen.set(true);
  }

  openEdit(row: PeriodRow): void {
    this.editingId.set(row.id);
    this.name.set(row.name);
    this.code.set(row.code);
    this.startDate.set(row.start_date.slice(0, 10));
    this.endDate.set(row.end_date.slice(0, 10));
    this.activateNow.set(row.is_active);
    this.formError.set(null);
    this.formOpen.set(true);
  }

  closeForm(): void {
    this.formOpen.set(false);
    this.editingId.set(null);
  }

  submitForm(): void {
    if (!this.isFormValid() || this.isSubmitting()) return;

    const payload: CreatePeriodPayload = {
      name: this.name().trim(),
      code: this.code().trim(),
      start_date: this.startDate(),
      end_date: this.endDate()
    };

    this.isSubmitting.set(true);
    this.formError.set(null);

    let call$: Observable<AcademicPeriodConfig>;
    if (this.editingId()) {
      call$ = this.periodosService.updatePeriod(this.editingId()!, {
        ...payload,
        is_active: this.activateNow()
      });
    } else {
      call$ = this.periodosService.createAndActivate({ ...payload, is_active: this.activateNow() });
    }

    call$.subscribe({
      next: (period) => {
        this.isSubmitting.set(false);
        this.closeForm();
        this.showToast(
          this.editingId()
            ? `Período "${period.name}" actualizado exitosamente.`
            : `Período "${period.name}" registrado exitosamente.`
        );
      },
      error: (err) => {
        this.isSubmitting.set(false);
        this.formError.set(this.periodosService.resolveError(err));
      }
    });
  }

  // ------------------------------------------------------------------
  // Archivo fuerte (confirmación por tipeo del código)
  // ------------------------------------------------------------------
  readonly archiving = signal<PeriodRow | null>(null);
  readonly archiveConfirmation = signal('');
  readonly isArchiving = signal(false);

  readonly archiveCodeMatches = computed(
    () => this.archiving() !== null && this.archiveConfirmation().trim() === this.archiving()!.code
  );

  readonly archiveImpactedCourses = computed(() => {
    const target = this.archiving();
    return target ? this.courseCountByPeriod()[target.id] ?? 0 : 0;
  });

  openArchive(row: PeriodRow): void {
    this.archiving.set(row);
    this.archiveConfirmation.set('');
  }

  closeArchive(): void {
    this.archiving.set(null);
    this.archiveConfirmation.set('');
  }

  confirmArchive(): void {
    const target = this.archiving();
    if (!target || !this.archiveCodeMatches() || this.isArchiving()) return;

    this.isArchiving.set(true);
    this.periodosService.archivePeriod(target.id, this.archiveConfirmation().trim()).subscribe({
      next: () => {
        this.isArchiving.set(false);
        this.closeArchive();
        this.showToast(
          `Período "${target.name}" archivado formalmente. Sus materias quedaron en modo solo lectura (:ro).`
        );
      },
      error: (err) => {
        this.isArchiving.set(false);
        this.showToast(this.periodosService.resolveError(err), 'error');
      }
    });
  }

  // ------------------------------------------------------------------
  // Activación
  // ------------------------------------------------------------------
  readonly activating = signal<PeriodRow | null>(null);
  readonly isActivating = signal(false);

  readonly activatingExpired = computed(() => {
    const target = this.activating();
    return target ? periodLifecycle(target) === 'archivado' : false;
  });

  openActivate(row: PeriodRow): void {
    this.activating.set(row);
  }

  closeActivate(): void {
    this.activating.set(null);
  }

  confirmActivate(): void {
    const target = this.activating();
    if (!target || this.isActivating()) return;

    this.isActivating.set(true);
    this.periodosService.updatePeriod(target.id, { is_active: true }).subscribe({
      next: () => {
        this.isActivating.set(false);
        this.closeActivate();
        this.showToast(`Período "${target.name}" activado como ciclo lectivo vigente.`);
      },
      error: (err) => {
        this.isActivating.set(false);
        this.closeActivate();
        this.showToast(this.periodosService.resolveError(err), 'error');
      }
    });
  }

  // ------------------------------------------------------------------
  // Borrado
  // ------------------------------------------------------------------
  readonly deleting = signal<PeriodRow | null>(null);
  readonly isDeleting = signal(false);

  readonly deletingHasCourses = computed(() => {
    const target = this.deleting();
    return target ? (this.courseCountByPeriod()[target.id] ?? 0) > 0 : false;
  });

  openDelete(row: PeriodRow): void {
    this.deleting.set(row);
  }

  closeDelete(): void {
    this.deleting.set(null);
  }

  confirmDelete(): void {
    const target = this.deleting();
    if (!target || this.isDeleting()) return;

    this.isDeleting.set(true);
    this.periodosService.deletePeriod(target.id).subscribe({
      next: () => {
        this.isDeleting.set(false);
        this.closeDelete();
        this.showToast(`Período "${target.name}" eliminado.`);
      },
      error: (err) => {
        this.isDeleting.set(false);
        this.closeDelete();
        this.showToast(this.periodosService.resolveError(err), 'error');
      }
    });
  }

  // ------------------------------------------------------------------
  // Conteo de materias por período (para modal de archivo y borrado)
  // ------------------------------------------------------------------
  readonly courseCountByPeriod = signal<Record<string, number>>({});

  ngOnInit(): void {
    this.periodosService.fetchPeriods().subscribe(() => this.loadCourseCounts());
  }

  private loadCourseCounts(): void {
    this.fetchSubjectCounts().subscribe((counts) => this.courseCountByPeriod.set(counts));
  }

  private fetchSubjectCounts(): Observable<Record<string, number>> {
    return this.http
      .get<{ data?: SubjectCountRow[] } | SubjectCountRow[]>('/api/v1/subjects')
      .pipe(
        map((res) => {
          const list = Array.isArray(res) ? res : res.data ?? [];
          const counts: Record<string, number> = {};
          for (const s of list) {
            if (s?.academic_period_id) {
              counts[s.academic_period_id] = (counts[s.academic_period_id] ?? 0) + 1;
            }
          }
          return counts;
        }),
        catchError(() => of({}))
      );
  }

  lifecycleLabel(lifecycle: PeriodLifecycle): string {
    return lifecycle === 'activo' ? 'Activo' : lifecycle === 'proximo' ? 'Próximo' : 'Archivado';
  }

  formatDate(iso: string): string {
    return formatSolvDate(iso, 'daymonthyear') ?? iso;
  }

  trackByPeriod(index: number, row: PeriodRow): string {
    return row.id;
  }
}
