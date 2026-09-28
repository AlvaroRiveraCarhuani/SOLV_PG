import { Component, OnInit, inject, signal, computed, HostListener } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { 
  AdminCoursesService, 
  AdminCourseItem, 
  AcademicPeriod, 
  DockerTemplateItem 
} from '../services/admin-courses.service';
import { CourseCreateModalComponent } from './components/course-create-modal/course-create-modal.component';
import { CourseReassignModalComponent } from './components/course-reassign-modal/course-reassign-modal.component';
import { ConfirmModalComponent } from '@shared/components/confirm-modal/confirm-modal.component';
import { ModalShellComponent } from '@shared/components/modal-shell/modal-shell.component';
import { FormFieldComponent } from '@shared/components/form-field/form-field.component';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { ComboboxComponent, ComboboxOption } from '@shared/components/combobox/combobox.component';
import { SearchBarComponent } from '@shared/components/search-bar/search-bar.component';
import { KpiCardComponent, KpiGridComponent } from '@shared/components/kpi-card/kpi-card.component';
import { StatusTabsComponent, StatusTabItem } from '@shared/components/status-tabs/status-tabs.component';
import { PaginationBarComponent } from '@shared/components/pagination-bar/pagination-bar.component';
import { 
  LucideBookOpen, 
  LucidePlus, 
  LucideCheckCircle, 
  LucideAlertCircle, 
  LucideLayers, 
  LucideArrowRightLeft, 
  LucideArchive, 
  LucideArchiveRestore, 
  LucideRefreshCw,
  LucideAlertTriangle,
  LucidePencil,
  LucideX
} from '@lucide/angular';

interface ToastState {
  text: string;
  isError?: boolean;
}

@Component({
  selector: 'admin-courses',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    MachineDataDirective,
    ComboboxComponent,
    SearchBarComponent,
    KpiCardComponent,
    KpiGridComponent,
    StatusTabsComponent,
    PaginationBarComponent,
    CourseCreateModalComponent,
    CourseReassignModalComponent,
    ConfirmModalComponent,
    ModalShellComponent,
    FormFieldComponent,
    LucideBookOpen,
    LucidePlus,
    LucideCheckCircle,
    LucideAlertCircle,
    LucideLayers,
    LucideArrowRightLeft,
    LucideArchive,
    LucideArchiveRestore,
    LucideAlertTriangle,
    LucidePencil,
  ],
  templateUrl: './admin-courses.component.html',
  styleUrls: ['./admin-courses.component.scss']
})
export class AdminCoursesComponent implements OnInit {
  coursesService = inject(AdminCoursesService);

  // Filtros de búsqueda
  searchQuery = signal<string>('');
  statusFilter = signal<'all' | 'active' | 'unassigned' | 'archived'>('all');

  periodHeaderComboboxOptions = computed<ComboboxOption[]>(() => {
    return [
      { id: '', label: 'Todos los periodos', value: '' },
      ...this.coursesService.periods().map(p => ({
        id: p.id,
        label: `${p.name} (${p.code})`,
        value: p.id,
        badge: p.is_active ? 'Actual' : undefined,
        badgeVariant: p.is_active ? ('official' as const) : undefined
      }))
    ];
  });

  selectedPeriodHeaderLabel = computed<string>(() => {
    const active = this.coursesService.activePeriod();
    if (!active) return 'Todos los periodos';
    return `${active.name} (${active.code})`;
  });

  onPeriodHeaderSelected(opt: ComboboxOption): void {
    this.onPeriodChange(opt.value || '');
  }

  // Paginación
  currentPage = signal<number>(1);
  pageSize = signal<number>(10);
  readonly pageSizeOptions = [10, 25, 50];

  // Control de modales
  showCreateCourseModal = signal<boolean>(false);
  selectedCourseForReassign = signal<AdminCourseItem | null>(null);

  // Toast flotante
  toastMessage = signal<ToastState | null>(null);

  // Cursos filtrados por periodo, estado y búsqueda
  filteredCourses = computed(() => {
    const q = this.searchQuery().toLowerCase().trim();
    const filter = this.statusFilter();
    const activePeriod = this.coursesService.activePeriod();
    let list = this.coursesService.courses();

    // Filtrar por periodo activo si está seleccionado
    if (activePeriod) {
      list = list.filter(c => !c.academic_period_id || c.academic_period_id === activePeriod.id);
    }

    if (filter === 'active') {
      list = list.filter(c => !c.is_archived);
    } else if (filter === 'unassigned') {
      list = list.filter(c => !c.teacher_id || c.teacher_name === 'Sin asignar');
    } else if (filter === 'archived') {
      list = list.filter(c => c.is_archived);
    }

    if (q) {
      list = list.filter(c => 
        c.name.toLowerCase().includes(q) || 
        c.code.toLowerCase().includes(q) ||
        c.teacher_name.toLowerCase().includes(q)
      );
    }

    return list;
  });

  // Paginación reactiva calculada
  totalPages = computed(() => {
    return Math.ceil(this.filteredCourses().length / this.pageSize()) || 1;
  });

  paginatedCourses = computed(() => {
    const start = (this.currentPage() - 1) * this.pageSize();
    return this.filteredCourses().slice(start, start + this.pageSize());
  });

  paginationDisplay = computed(() => {
    const total = this.filteredCourses().length;
    if (total === 0) return { from: 0, to: 0, total: 0 };
    const from = (this.currentPage() - 1) * this.pageSize() + 1;
    const to = Math.min(this.currentPage() * this.pageSize(), total);
    return { from, to, total };
  });

  displayedPages = computed(() => {
    const total = this.totalPages();
    const current = this.currentPage();
    const pages: number[] = [];
    const maxVisible = 5;

    let start = Math.max(1, current - Math.floor(maxVisible / 2));
    let end = Math.min(total, start + maxVisible - 1);
    if (end - start + 1 < maxVisible) {
      start = Math.max(1, end - maxVisible + 1);
    }

    for (let i = start; i <= end; i++) {
      pages.push(i);
    }
    return pages;
  });

  // Métricas 100% enfocadas en Cursos y Ciclo Académico
  periodKpis = computed(() => {
    const activePeriod = this.coursesService.activePeriod();
    let allInPeriod = this.coursesService.courses();
    if (activePeriod) {
      allInPeriod = allInPeriod.filter(c => !c.academic_period_id || c.academic_period_id === activePeriod.id);
    }

    const totalCourses = allInPeriod.length;
    const unassignedCourses = allInPeriod.filter(c => !c.teacher_id || c.teacher_name === 'Sin asignar').length;
    const archivedCourses = allInPeriod.filter(c => c.is_archived).length;

    return {
      totalCourses,
      unassignedCourses,
      archivedCourses
    };
  });

  coursesCounts = computed(() => {
    const kpis = this.periodKpis();
    return {
      all: kpis.totalCourses,
      active: kpis.totalCourses - kpis.archivedCourses,
      unassigned: kpis.unassignedCourses,
      archived: kpis.archivedCourses
    };
  });

  courseStatusTabs = computed<StatusTabItem[]>(() => [
    { id: 'all', label: 'Todos', count: this.coursesCounts().all },
    { id: 'active', label: 'Activos', count: this.coursesCounts().active, badgeVariant: 'active' },
    { id: 'unassigned', label: 'Sin Docente', count: this.coursesCounts().unassigned, badgeVariant: 'warning' },
    { id: 'archived', label: 'Archivados', count: this.coursesCounts().archived, badgeVariant: 'neutral' }
  ]);

  ngOnInit(): void {
    this.coursesService.loadAll();
  }

  onPeriodChange(periodId: string): void {
    if (!periodId) {
      this.coursesService.setActivePeriod(null);
      this.currentPage.set(1);
      return;
    }
    const period = this.coursesService.periods().find(p => p.id === periodId);
    if (period) {
      this.coursesService.setActivePeriod(period);
      this.currentPage.set(1);
    }
  }

  onSearchChange(val: string): void {
    this.searchQuery.set(val);
    this.currentPage.set(1);
  }

  onStatusFilterChange(filter: string): void {
    this.statusFilter.set(filter as 'all' | 'active' | 'unassigned' | 'archived');
    this.currentPage.set(1);
  }

  goToPage(page: number): void {
    if (page >= 1 && page <= this.totalPages()) {
      this.currentPage.set(page);
    }
  }

  onPageSizeChange(size: number): void {
    this.pageSize.set(size);
    this.currentPage.set(1);
  }

  reload(): void {
    this.coursesService.loadAll();
    this.showToast('Datos actualizados desde el servidor.');
  }

  // Modales
  openCreateCourseModal(): void {
    this.showCreateCourseModal.set(true);
  }

  closeCreateCourseModal(): void {
    this.showCreateCourseModal.set(false);
  }

  onCourseCreated(): void {
    this.closeCreateCourseModal();
    this.showToast('Curso institucional registrado exitosamente.');
    this.coursesService.loadAll();
  }

  openReassignModal(course: AdminCourseItem): void {
    this.selectedCourseForReassign.set(course);
  }

  closeReassignModal(): void {
    this.selectedCourseForReassign.set(null);
  }

  onReassignCompleted(): void {
    this.closeReassignModal();
    this.showToast('Docente reasignado con éxito.');
  }

  courseToToggleArchive = signal<AdminCourseItem | null>(null);

  requestToggleArchive(course: AdminCourseItem): void {
    this.courseToToggleArchive.set(course);
  }

  cancelToggleArchive(): void {
    this.courseToToggleArchive.set(null);
  }

  executeToggleArchive(course: AdminCourseItem): void {
    const nextState = !course.is_archived;
    this.coursesService.toggleArchiveCourse(course.id, nextState).subscribe({
      next: () => {
        this.cancelToggleArchive();
        this.showToast(`Materia ${nextState ? 'archivada' : 'activada'} correctamente.`);
      },
      error: () => {
        this.cancelToggleArchive();
        this.showToast('No se pudo cambiar el estado de la materia.', true);
      }
    });
  }

  // Modal de edición de curso
  editingCourse = signal<AdminCourseItem | null>(null);
  editCourseName = signal<string>('');
  editCourseCode = signal<string>('');
  isSubmittingEditCourse = signal<boolean>(false);
  editCourseError = signal<string | null>(null);

  @HostListener('document:keydown.escape')
  handleEscape(): void {
    if (this.courseToToggleArchive()) {
      this.cancelToggleArchive();
    } else if (this.selectedCourseForReassign()) {
      this.closeReassignModal();
     } else if (this.showCreateCourseModal()) {
       this.closeCreateCourseModal();
     }
  }

  openEditCourseModal(course: AdminCourseItem): void {
    this.editingCourse.set(course);
    this.editCourseName.set(course.name);
    this.editCourseCode.set(course.code);
    this.editCourseError.set(null);
  }

  closeEditCourseModal(): void {
    this.editingCourse.set(null);
    this.editCourseError.set(null);
  }

  saveEditCourse(course: AdminCourseItem): void {
    const name = this.editCourseName().trim();
    const code = this.editCourseCode().trim().toUpperCase();

    if (!name || !code) {
      this.editCourseError.set('Nombre y código de materia son obligatorios.');
      return;
    }

    this.isSubmittingEditCourse.set(true);
    this.editCourseError.set(null);

    this.coursesService.updateCourse(course.id, { name, code }).subscribe({
      next: () => {
        this.isSubmittingEditCourse.set(false);
        this.closeEditCourseModal();
        this.showToast(`Materia "${code}" actualizada con éxito.`);
      },
      error: (err: any) => {
        this.isSubmittingEditCourse.set(false);
        const msg = err.error?.error || 'Error al actualizar la materia.';
        this.editCourseError.set(msg);
      }
    });
  }

  showToast(text: string, isError = false): void {
    this.toastMessage.set({ text, isError });
    setTimeout(() => {
      this.toastMessage.set(null);
    }, 4000);
  }
}
