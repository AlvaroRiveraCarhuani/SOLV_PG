import { Component, OnInit, signal, computed, inject, HostListener } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { 
  AdminStudentsService, 
  AdminStudentItem, 
  AdminStudentCourseItem, 
  SubjectOption,
  AcademicPeriodOption,
  CreateStudentDTO
} from '../services/admin-students.service';
import { StudentCoursesModalComponent } from './components/student-courses-modal/student-courses-modal.component';
import { StudentResetOOMModalComponent } from './components/student-reset-oom-modal/student-reset-oom-modal.component';
import { StudentCreateModalComponent } from './components/student-create-modal/student-create-modal.component';
import { ConfirmModalComponent } from '@shared/components/confirm-modal/confirm-modal.component';
import { ComboboxComponent, ComboboxOption } from '@shared/components/combobox/combobox.component';
import { SearchBarComponent } from '@shared/components/search-bar/search-bar.component';
import { KpiCardComponent, KpiGridComponent } from '@shared/components/kpi-card/kpi-card.component';
import { PaginationBarComponent } from '@shared/components/pagination-bar/pagination-bar.component';
import { 
  LucideUsers, 
  LucideCpu, 
  LucideAlertTriangle, 
  LucideShieldAlert, 
  LucideRotateCcw, 
  LucideBookOpen, 
  LucideCheckCircle2,
  LucideUserPlus,
  LucideUserX,
  LucideUserCheck,
  LucideCopy,
  LucideCheck,
  LucideMoreVertical
} from '@lucide/angular';

@Component({
  selector: 'admin-students',
  standalone: true,
  imports: [
    CommonModule, 
    FormsModule, 
    ComboboxComponent,
    SearchBarComponent,
    KpiCardComponent,
    KpiGridComponent,
    PaginationBarComponent,
    StudentCoursesModalComponent,
    StudentResetOOMModalComponent,
    StudentCreateModalComponent,
    ConfirmModalComponent,
    LucideUsers, 
    LucideCpu, 
    LucideAlertTriangle, 
    LucideShieldAlert, 
    LucideRotateCcw, 
    LucideBookOpen, 
    LucideCheckCircle2,
    LucideUserPlus,
    LucideUserX,
    LucideUserCheck,
    LucideCopy,
    LucideCheck,
    LucideMoreVertical
  ],
  templateUrl: './admin-students.component.html',
  styleUrls: ['./admin-students.component.scss']
})
export class AdminStudentsComponent implements OnInit {
  private studentsService = inject(AdminStudentsService);

  // Estados reactivos principales
  students = signal<AdminStudentItem[]>([]);
  subjects = signal<SubjectOption[]>([]);
  academicPeriods = signal<AcademicPeriodOption[]>([]);
  isLoading = signal<boolean>(true);

  // Filtros
  searchTerm = signal<string>('');
  selectedPeriod = signal<string>('');
  selectedSubject = signal<string>('all');
  selectedStatus = signal<string>('all');

  periodComboboxOptions = computed<ComboboxOption[]>(() => {
    return [
      { id: 'all', label: 'Todos los periodos', value: 'all' },
      ...this.academicPeriods().map(p => ({
        id: p.id,
        label: `${p.code}${p.is_active ? ' • Vigente' : ''}`,
        value: p.id
      }))
    ];
  });

  selectedPeriodLabel = computed<string>(() => {
    const id = this.selectedPeriod();
    if (!id || id === 'all') return 'Todos los periodos';
    const match = this.academicPeriods().find(p => p.id === id);
    return match ? `${match.code}${match.is_active ? ' • Vigente' : ''}` : 'Todos los periodos';
  });

  onPeriodSelected(opt: ComboboxOption): void {
    this.selectedPeriod.set(opt.value || 'all');
    this.onFilterChange();
  }

  subjectComboboxOptions = computed<ComboboxOption[]>(() => {
    return [
      { id: 'all', label: 'Todas las materias', value: 'all' },
      ...this.subjects().map(s => ({
        id: s.id,
        label: `${s.code} - ${s.name}`,
        value: s.id
      }))
    ];
  });

  selectedSubjectLabel = computed<string>(() => {
    const id = this.selectedSubject();
    if (!id || id === 'all') return 'Todas las materias';
    const match = this.subjects().find(s => s.id === id);
    return match ? `${match.code} - ${match.name}` : 'Todas las materias';
  });

  onSubjectSelected(opt: ComboboxOption): void {
    this.selectedSubject.set(opt.value || 'all');
    this.onFilterChange();
  }

  statusComboboxOptions = computed<ComboboxOption[]>(() => [
    { id: 'all', label: 'Todos los estados', value: 'all' },
    { id: 'enrolled', label: 'Cursando materias', value: 'enrolled' },
    { id: 'inactive', label: 'Sin materias en el periodo', value: 'inactive' },
    { id: 'running', label: 'Con entornos activos', value: 'running' },
    { id: 'strikes', label: 'Con penalizaciones OOM', value: 'strikes' },
    { id: 'blocked', label: 'Bloqueados (3 Strikes)', value: 'blocked' },
    { id: 'suspended', label: 'Cuentas suspendidas', value: 'suspended' }
  ]);

  selectedStatusLabel = computed<string>(() => {
    const id = this.selectedStatus();
    const match = this.statusComboboxOptions().find(o => o.value === id);
    return match ? match.label : 'Todos los estados';
  });

  onStatusSelected(opt: ComboboxOption): void {
    this.selectedStatus.set(opt.value || 'all');
    this.currentPage.set(1);
  }

  // Paginación
  currentPage = signal<number>(1);
  readonly pageSize = 8;

  // Menú contextual de acciones
  activeMenuStudentId = signal<string | null>(null);

  @HostListener('document:click')
  onDocumentClick() {
    this.activeMenuStudentId.set(null);
  }

  @HostListener('document:keydown.escape')
  onEscape() {
    this.activeMenuStudentId.set(null);
    this.showCreateModal.set(false);
    this.studentToToggleStatus.set(null);
    this.studentForReset.set(null);
    this.studentForCourses.set(null);
  }

  toggleActionsMenu(studentId: string, event: Event) {
    event.stopPropagation();
    if (this.activeMenuStudentId() === studentId) {
      this.activeMenuStudentId.set(null);
    } else {
      this.activeMenuStudentId.set(studentId);
    }
  }

  // Modales
  showCreateModal = signal<boolean>(false);
  isCreatingStudent = signal<boolean>(false);

  studentToToggleStatus = signal<AdminStudentItem | null>(null);
  isTogglingStatus = signal<boolean>(false);

  studentForReset = signal<AdminStudentItem | null>(null);
  isResetting = signal<boolean>(false);

  studentForCourses = signal<AdminStudentItem | null>(null);
  studentCourses = signal<AdminStudentCourseItem[]>([]);
  isLoadingCourses = signal<boolean>(false);
  actionInProgressWorkspaceId = signal<string | null>(null);
  copiedEmail = signal<string | null>(null);

  // Feedback / Toasts
  toastMessage = signal<string | null>(null);
  toastType = signal<'success' | 'error'>('success');

  // KPIs computados
  totalStudents = computed(() => this.students().length);
  enrolledStudentsCount = computed(() => 
    this.students().filter(s => s.academic_status === 'enrolled').length
  );
  inactiveStudentsCount = computed(() => 
    this.students().filter(s => s.academic_status === 'inactive').length
  );
  activeWorkspacesTotal = computed(() => 
    this.students().reduce((sum, s) => sum + s.active_workspaces_count, 0)
  );
  strikesStudentsCount = computed(() => 
    this.students().filter(s => s.oom_strike_count > 0 && s.oom_strike_count < 3).length
  );
  blockedStudentsCount = computed(() => 
    this.students().filter(s => s.oom_strike_count >= 3).length
  );

  // Filtrado reactivo en memoria
  filteredStudents = computed(() => {
    let list = this.students();
    const search = this.searchTerm().trim().toLowerCase();
    const status = this.selectedStatus();

    if (search) {
      list = list.filter(s => 
        (s.first_name + ' ' + s.last_name).toLowerCase().includes(search) ||
        s.email.toLowerCase().includes(search)
      );
    }

    if (status !== 'all') {
      switch (status) {
        case 'enrolled':
          list = list.filter(s => s.academic_status === 'enrolled');
          break;
        case 'inactive':
          list = list.filter(s => s.academic_status === 'inactive');
          break;
        case 'suspended':
          list = list.filter(s => s.status === 'suspended');
          break;
        case 'blocked':
          list = list.filter(s => s.oom_strike_count >= 3);
          break;
        case 'strikes':
          list = list.filter(s => s.oom_strike_count > 0);
          break;
        case 'running':
          list = list.filter(s => s.active_workspaces_count > 0);
          break;
        case 'idle':
          list = list.filter(s => s.active_workspaces_count === 0 && s.oom_strike_count === 0);
          break;
      }
    }

    return list;
  });

  // Paginación computada
  paginatedStudents = computed(() => {
    const start = (this.currentPage() - 1) * this.pageSize;
    return this.filteredStudents().slice(start, start + this.pageSize);
  });

  totalPages = computed(() => 
    Math.ceil(this.filteredStudents().length / this.pageSize) || 1
  );

  paginationFrom = computed(() => {
    if (this.filteredStudents().length === 0) return 0;
    return (this.currentPage() - 1) * this.pageSize + 1;
  });

  paginationTo = computed(() => {
    return Math.min(this.currentPage() * this.pageSize, this.filteredStudents().length);
  });

  onSearchChange(term: string): void {
    this.searchTerm.set(term);
    this.currentPage.set(1);
  }

  ngOnInit(): void {
    this.loadAcademicPeriods();
    this.loadSubjects();
  }

  loadAcademicPeriods(): void {
    this.studentsService.getAcademicPeriods().subscribe({
      next: (periods) => {
        this.academicPeriods.set(periods);
        const active = periods.find(p => p.is_active);
        if (active && !this.selectedPeriod()) {
          this.selectedPeriod.set(active.id);
        } else if (!this.selectedPeriod() && periods.length > 0) {
          this.selectedPeriod.set(periods[0].id);
        }
        this.loadStudents();
      },
      error: () => {
        this.academicPeriods.set([]);
        this.loadStudents();
      }
    });
  }

  loadSubjects(): void {
    this.studentsService.getSubjects().subscribe({
      next: (subs) => this.subjects.set(subs),
      error: () => this.subjects.set([])
    });
  }

  loadStudents(): void {
    this.isLoading.set(true);
    const subjectId = this.selectedSubject() !== 'all' ? this.selectedSubject() : undefined;
    const periodId = this.selectedPeriod() && this.selectedPeriod() !== 'all' ? this.selectedPeriod() : undefined;
    
    this.studentsService.getStudents(undefined, subjectId, undefined, periodId).subscribe({
      next: (data) => {
        this.students.set(data);
        this.isLoading.set(false);
      },
      error: () => {
        this.isLoading.set(false);
        this.showToast('No se pudo cargar el directorio de estudiantes', 'error');
      }
    });
  }

  onFilterChange(): void {
    this.currentPage.set(1);
    this.loadStudents();
  }

  goToPage(page: number): void {
    if (page >= 1 && page <= this.totalPages()) {
      this.currentPage.set(page);
    }
  }

  // Acciones de Materias
  openCoursesModal(student: AdminStudentItem): void {
    this.studentForCourses.set(student);
    this.isLoadingCourses.set(true);
    this.studentCourses.set([]);

    this.studentsService.getStudentCourses(student.id).subscribe({
      next: (courses) => {
        this.studentCourses.set(courses);
        this.isLoadingCourses.set(false);
      },
      error: () => {
        this.isLoadingCourses.set(false);
        this.showToast('Error al consultar materias del estudiante', 'error');
      }
    });
  }

  closeCoursesModal(): void {
    this.studentForCourses.set(null);
    this.studentCourses.set([]);
  }

  // Acciones de Reseteo OOM
  openResetOOMModal(student: AdminStudentItem): void {
    this.studentForReset.set(student);
  }

  closeResetOOMModal(): void {
    this.studentForReset.set(null);
  }

  confirmResetOOM(reason: string): void {
    const student = this.studentForReset();
    if (!student) return;

    this.isResetting.set(true);
    this.studentsService.resetStudentOOM(student.id, reason).subscribe({
      next: (res) => {
        this.isResetting.set(false);
        this.closeResetOOMModal();
        this.showToast(`Penalizaciones restablecidas (${res.workspaces_reset_count} espacios actualizados)`, 'success');
        this.loadStudents();
      },
      error: (err) => {
        this.isResetting.set(false);
        const detail = err.error?.error || err.error?.message || 'Error al restablecer penalizaciones.';
        this.showToast(detail, 'error');
      }
    });
  }

  // Acciones de Creación de Estudiante
  openCreateModal(): void {
    this.showCreateModal.set(true);
  }

  closeCreateModal(): void {
    this.showCreateModal.set(false);
  }

  onStudentCreated(dto: CreateStudentDTO): void {
    this.isCreatingStudent.set(true);
    this.studentsService.createStudent(dto).subscribe({
      next: (newStudent) => {
        this.isCreatingStudent.set(false);
        this.closeCreateModal();
        this.showToast(`Estudiante ${newStudent.first_name} ${newStudent.last_name} registrado correctamente.`, 'success');
        this.loadStudents();
      },
      error: (err) => {
        this.isCreatingStudent.set(false);
        const detail = err.error?.error || err.error?.message || 'Error al registrar estudiante institucional.';
        this.showToast(detail, 'error');
      }
    });
  }

  // Acciones de Suspensión / Reactivación de Cuenta
  requestToggleStatus(student: AdminStudentItem): void {
    this.studentToToggleStatus.set(student);
  }

  cancelToggleStatus(): void {
    this.studentToToggleStatus.set(null);
  }

  confirmToggleStatus(student: AdminStudentItem): void {
    const nextStatus: 'active' | 'suspended' = student.status === 'suspended' ? 'active' : 'suspended';
    const reason = nextStatus === 'suspended' ? 'Suspensión administrativa preventiva' : '';

    this.isTogglingStatus.set(true);
    this.studentsService.updateStudentStatus(student.id, nextStatus, reason).subscribe({
      next: () => {
        this.isTogglingStatus.set(false);
        this.cancelToggleStatus();
        this.showToast(
          nextStatus === 'active' 
            ? `Acceso restablecido para ${student.first_name} ${student.last_name}.` 
            : `Cuenta de ${student.first_name} ${student.last_name} suspendida.`,
          'success'
        );
        this.loadStudents();
      },
      error: (err) => {
        this.isTogglingStatus.set(false);
        const detail = err.error?.error || err.error?.message || 'Error al actualizar estado del estudiante.';
        this.showToast(detail, 'error');
      }
    });
  }

  // Acciones Rápidas sobre Contenedores Docker (desde el modal de materias)
  onRestartWorkspace(workspaceId: string): void {
    this.actionInProgressWorkspaceId.set(workspaceId);
    this.studentsService.restartWorkspace(workspaceId).subscribe({
      next: () => {
        this.actionInProgressWorkspaceId.set(null);
        this.showToast('Entorno reiniciado en el servidor.', 'success');
        if (this.studentForCourses()) {
          this.openCoursesModal(this.studentForCourses()!);
        }
        this.loadStudents();
      },
      error: (err) => {
        this.actionInProgressWorkspaceId.set(null);
        const detail = err.error?.error || err.error?.message || 'Error al reiniciar entorno de trabajo.';
        this.showToast(detail, 'error');
      }
    });
  }

  onPauseWorkspace(workspaceId: string): void {
    this.actionInProgressWorkspaceId.set(workspaceId);
    this.studentsService.pauseWorkspace(workspaceId).subscribe({
      next: () => {
        this.actionInProgressWorkspaceId.set(null);
        this.showToast('Entorno pausado y memoria RAM liberada.', 'success');
        if (this.studentForCourses()) {
          this.openCoursesModal(this.studentForCourses()!);
        }
        this.loadStudents();
      },
      error: (err) => {
        this.actionInProgressWorkspaceId.set(null);
        const detail = err.error?.error || err.error?.message || 'Error al pausar entorno de trabajo.';
        this.showToast(detail, 'error');
      }
    });
  }

  copyEmail(email: string): void {
    if (!navigator.clipboard) return;
    navigator.clipboard.writeText(email).then(() => {
      this.copiedEmail.set(email);
      setTimeout(() => {
        if (this.copiedEmail() === email) {
          this.copiedEmail.set(null);
        }
      }, 2000);
    });
  }

  resetFilters(): void {
    this.searchTerm.set('');
    this.selectedSubject.set('all');
    this.selectedStatus.set('all');
    this.currentPage.set(1);
  }

  getInitials(firstName: string, lastName: string): string {
    const first = firstName ? firstName.trim().charAt(0).toUpperCase() : '';
    const last = lastName ? lastName.trim().charAt(0).toUpperCase() : '';
    return first + last || 'E';
  }

  formatDate(dateStr?: string | null): string {
    if (!dateStr) return 'Sin registros';
    try {
      const d = new Date(dateStr);
      return d.toLocaleDateString('es-ES', { 
        day: '2-digit', 
        month: 'short', 
        hour: '2-digit', 
        minute: '2-digit' 
      });
    } catch {
      return dateStr;
    }
  }

  showToast(msg: string, type: 'success' | 'error' = 'success'): void {
    this.toastMessage.set(msg);
    this.toastType.set(type);
    setTimeout(() => {
      this.toastMessage.set(null);
    }, 4500);
  }
}
