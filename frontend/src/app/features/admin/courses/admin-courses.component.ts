import { Component, OnInit, inject, signal, computed, HostListener } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { 
  AdminCoursesService, 
  AdminCourseItem, 
  AcademicPeriod, 
  DockerTemplateItem 
} from '../services/admin-courses.service';
import { PeriodManageModalComponent } from './components/period-manage-modal/period-manage-modal.component';
import { CourseCreateModalComponent } from './components/course-create-modal/course-create-modal.component';
import { CourseReassignModalComponent } from './components/course-reassign-modal/course-reassign-modal.component';
import { ConfirmModalComponent } from '@shared/components/confirm-modal/confirm-modal.component';
import { 
  LucideBookOpen, 
  LucideSearch, 
  LucidePlus, 
  LucideCalendar, 
  LucideCheckCircle, 
  LucideAlertCircle, 
  LucideLayers, 
  LucideArrowRightLeft, 
  LucideArchive, 
  LucideArchiveRestore, 
  LucideRefreshCw,
  LucideChevronLeft,
  LucideChevronRight,
  LucideAlertTriangle,
  LucidePencil,
  LucideX
} from '@lucide/angular';

interface ToastState {
  text: string;
  isError?: boolean;
}

@Component({
  selector: 'solv-admin-courses',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    PeriodManageModalComponent,
    CourseCreateModalComponent,
    CourseReassignModalComponent,
    ConfirmModalComponent,
    LucideBookOpen,
    LucideSearch,
    LucidePlus,
    LucideCalendar,
    LucideCheckCircle,
    LucideAlertCircle,
    LucideLayers,
    LucideArrowRightLeft,
    LucideArchive,
    LucideArchiveRestore,
    LucideChevronLeft,
    LucideChevronRight,
    LucideAlertTriangle,
    LucidePencil,
    LucideX
  ],
  templateUrl: './admin-courses.component.html',
  styleUrls: ['./admin-courses.component.scss']
})
export class AdminCoursesComponent implements OnInit {
  coursesService = inject(AdminCoursesService);

  // Filtros de búsqueda
  searchQuery = signal<string>('');
  statusFilter = signal<'all' | 'active' | 'unassigned' | 'archived'>('all');

  // Paginación
  currentPage = signal<number>(1);
  pageSize = signal<number>(10);
  readonly pageSizeOptions = [10, 25, 50];

  // Control de modales
  showPeriodsModal = signal<boolean>(false);
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

  onStatusFilterChange(filter: 'all' | 'active' | 'unassigned' | 'archived'): void {
    this.statusFilter.set(filter);
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
  openPeriodsModal(): void {
    this.showPeriodsModal.set(true);
  }

  closePeriodsModal(): void {
    this.showPeriodsModal.set(false);
  }

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
    } else if (this.editingCourse()) {
      this.closeEditCourseModal();
    } else if (this.selectedCourseForReassign()) {
      this.closeReassignModal();
    } else if (this.showCreateCourseModal()) {
      this.closeCreateCourseModal();
    } else if (this.showPeriodsModal()) {
      this.closePeriodsModal();
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
