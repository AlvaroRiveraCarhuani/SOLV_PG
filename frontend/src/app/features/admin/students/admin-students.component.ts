import { Component, OnInit, signal, computed, inject, HostListener } from '@angular/core';
import { CommonModule } from '@angular/common';
import { 
  AdminStudentsService, 
  AdminStudentItem, 
  AdminStudentCourseItem, 
  SubjectOption, 
  AcademicPeriodOption, 
  CreateStudentDTO 
} from '../services/admin-students.service';
import { StudentKpiCardsComponent } from './components/student-kpi-cards/student-kpi-cards.component';
import { StudentTableComponent } from './components/student-table/student-table.component';
import { StudentCoursesModalComponent } from './components/student-courses-modal/student-courses-modal.component';
import { StudentResetOOMModalComponent } from './components/student-reset-oom-modal/student-reset-oom-modal.component';
import { StudentRegisterModalComponent } from './components/student-register-modal/student-register-modal.component';
import { StudentEnrollRosterModalComponent } from './components/student-enroll-roster-modal/student-enroll-roster-modal.component';
import { ConfirmModalComponent } from '@shared/components/confirm-modal/confirm-modal.component';
import { ComboboxOption } from '@shared/components/combobox/combobox.component';
import { LucideCheckCircle2, LucideAlertTriangle, LucideUserPlus, LucideBookOpen } from '@lucide/angular';

@Component({
  selector: 'admin-students',
  standalone: true,
  imports: [
    CommonModule,
    StudentKpiCardsComponent,
    StudentTableComponent,
    StudentCoursesModalComponent,
    StudentResetOOMModalComponent,
    ConfirmModalComponent,
    LucideCheckCircle2,
    LucideAlertTriangle,
    LucideUserPlus,
    LucideBookOpen,
    StudentRegisterModalComponent,
    StudentEnrollRosterModalComponent
  ],
  templateUrl: './admin-students.component.html',
  styleUrls: ['./admin-students.component.scss']
})
export class AdminStudentsComponent implements OnInit {
  private readonly studentsService = inject(AdminStudentsService);

  readonly students = signal<AdminStudentItem[]>([]);
  readonly subjects = signal<SubjectOption[]>([]);
  readonly academicPeriods = signal<AcademicPeriodOption[]>([]);
  readonly isLoading = signal<boolean>(true);

  readonly searchTerm = signal<string>('');
  readonly selectedPeriod = signal<string>('');
  readonly selectedSubject = signal<string>('all');
  readonly selectedStatus = signal<string>('all');
  readonly currentPage = signal<number>(1);
  readonly pageSize = 8;
  readonly activeMenuStudentId = signal<string | null>(null);

  readonly showRegisterModal = signal<boolean>(false);
  readonly isRegistering = signal<boolean>(false);
  readonly showRosterEnrollModal = signal<boolean>(false);
  readonly isEnrollingRoster = signal<boolean>(false);
  readonly studentToToggleStatus = signal<AdminStudentItem | null>(null);
  readonly isTogglingStatus = signal<boolean>(false);
  readonly studentForReset = signal<AdminStudentItem | null>(null);
  readonly isResetting = signal<boolean>(false);
  readonly studentForCourses = signal<AdminStudentItem | null>(null);
  readonly studentCourses = signal<AdminStudentCourseItem[]>([]);
  readonly isLoadingCourses = signal<boolean>(false);
  readonly actionInProgressWorkspaceId = signal<string | null>(null);

  readonly toastMessage = signal<string | null>(null);
  readonly toastType = signal<'success' | 'error'>('success');

  readonly periodComboboxOptions = computed<ComboboxOption[]>(() => [
    { id: 'all', label: 'Todos los periodos', value: 'all' },
    ...this.academicPeriods().map(p => ({ id: p.id, label: `${p.code}${p.is_active ? ' • Vigente' : ''}`, value: p.id }))
  ]);

  readonly selectedPeriodLabel = computed(() => {
    const match = this.academicPeriods().find(p => p.id === this.selectedPeriod());
    return match ? `${match.code}${match.is_active ? ' • Vigente' : ''}` : 'Todos los periodos';
  });

  readonly subjectComboboxOptions = computed<ComboboxOption[]>(() => [
    { id: 'all', label: 'Todas las materias', value: 'all' },
    ...this.subjects().map(s => ({ id: s.id, label: `${s.code} - ${s.name}`, value: s.id }))
  ]);

  readonly selectedSubjectLabel = computed(() => {
    const match = this.subjects().find(s => s.id === this.selectedSubject());
    return match ? `${match.code} - ${match.name}` : 'Todas las materias';
  });

  readonly statusComboboxOptions = computed<ComboboxOption[]>(() => [
    { id: 'all', label: 'Todos los estados', value: 'all' },
    { id: 'enrolled', label: 'Cursando materias', value: 'enrolled' },
    { id: 'inactive', label: 'Sin materias en el periodo', value: 'inactive' },
    { id: 'running', label: 'Con entornos activos', value: 'running' },
    { id: 'strikes', label: 'Con penalizaciones OOM', value: 'strikes' },
    { id: 'blocked', label: 'Bloqueados (3 Strikes)', value: 'blocked' },
    { id: 'suspended', label: 'Cuentas suspendidas', value: 'suspended' }
  ]);

  readonly selectedStatusLabel = computed(() =>
    this.statusComboboxOptions().find(o => o.value === this.selectedStatus())?.label ?? 'Todos los estados'
  );

  readonly totalStudents = computed(() => this.students().length);
  readonly enrolledStudentsCount = computed(() => this.students().filter(s => s.academic_status === 'enrolled').length);
  readonly inactiveStudentsCount = computed(() => this.students().filter(s => s.academic_status === 'inactive').length);
  readonly activeWorkspacesTotal = computed(() => this.students().reduce((sum, s) => sum + s.active_workspaces_count, 0));
  readonly strikesStudentsCount = computed(() => this.students().filter(s => s.oom_strike_count > 0 && s.oom_strike_count < 3).length);
  readonly blockedStudentsCount = computed(() => this.students().filter(s => s.oom_strike_count >= 3).length);

  readonly filteredStudents = computed(() => {
    const search = this.searchTerm().trim().toLowerCase();
    const status = this.selectedStatus();
    return this.students().filter(s => {
      const matchSearch = !search || (s.first_name + ' ' + s.last_name).toLowerCase().includes(search) || s.email.toLowerCase().includes(search);
      if (!matchSearch) return false;
      switch (status) {
        case 'enrolled': return s.academic_status === 'enrolled';
        case 'inactive': return s.academic_status === 'inactive';
        case 'suspended': return s.status === 'suspended';
        case 'blocked': return s.oom_strike_count >= 3;
        case 'strikes': return s.oom_strike_count > 0;
        case 'running': return s.active_workspaces_count > 0;
        default: return true;
      }
    });
  });

  readonly paginatedStudents = computed(() =>
    this.filteredStudents().slice((this.currentPage() - 1) * this.pageSize, this.currentPage() * this.pageSize)
  );
  readonly totalPages = computed(() => Math.ceil(this.filteredStudents().length / this.pageSize) || 1);
  readonly paginationFrom = computed(() => this.filteredStudents().length === 0 ? 0 : (this.currentPage() - 1) * this.pageSize + 1);
  readonly paginationTo = computed(() => Math.min(this.currentPage() * this.pageSize, this.filteredStudents().length));

  @HostListener('document:keydown.escape')
  onEscape(): void {
    this.activeMenuStudentId.set(null);
    this.showRegisterModal.set(false);
    this.showRosterEnrollModal.set(false);
    this.studentToToggleStatus.set(null);
    this.studentForReset.set(null);
    this.studentForCourses.set(null);
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

  onPeriodSelected(opt: ComboboxOption): void {
    this.selectedPeriod.set(opt.value || 'all');
    this.onFilterChange();
  }

  onSubjectSelected(opt: ComboboxOption): void {
    this.selectedSubject.set(opt.value || 'all');
    this.onFilterChange();
  }

  onStatusSelected(opt: ComboboxOption): void {
    this.selectedStatus.set(opt.value || 'all');
    this.currentPage.set(1);
  }

  onSearchChange(term: string): void {
    this.searchTerm.set(term);
    this.currentPage.set(1);
  }

  goToPage(page: number): void {
    if (page >= 1 && page <= this.totalPages()) {
      this.currentPage.set(page);
    }
  }

  toggleActionsMenu(studentId: string): void {
    this.activeMenuStudentId.update(curr => curr === studentId ? null : studentId);
  }

  copyEmail(email: string): void {
    if (!navigator.clipboard) return;
    navigator.clipboard.writeText(email);
  }

  openCoursesModal(student: AdminStudentItem): void {
    this.studentForCourses.set(student);
    this.isLoadingCourses.set(true);
    this.studentCourses.set([]);
    this.studentsService.getStudentCourses(student.id).subscribe({
      next: (courses) => { this.studentCourses.set(courses); this.isLoadingCourses.set(false); },
      error: () => { this.isLoadingCourses.set(false); this.showToast('Error al consultar materias del estudiante', 'error'); }
    });
  }

  closeCoursesModal(): void {
    this.studentForCourses.set(null);
    this.studentCourses.set([]);
  }

  openResetOOMModal(student: AdminStudentItem): void { this.studentForReset.set(student); }
  closeResetOOMModal(): void { this.studentForReset.set(null); }

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
        this.showToast(err.error?.error || err.error?.message || 'Error al restablecer penalizaciones.', 'error');
      }
    });
  }

  openRegisterModal(): void { this.showRegisterModal.set(true); }
  closeRegisterModal(): void { this.showRegisterModal.set(false); }

  onStudentCreated(dto: CreateStudentDTO): void {
    this.isRegistering.set(true);
    this.studentsService.createStudent(dto).subscribe({
      next: (newStudent) => {
        this.isRegistering.set(false);
        this.closeRegisterModal();
        this.showToast(`Estudiante ${newStudent.first_name} ${newStudent.last_name} registrado correctamente.`, 'success');
        this.loadStudents();
      },
      error: (err) => {
        this.isRegistering.set(false);
        this.showToast(err.error?.error || err.error?.message || 'Error al registrar estudiante institucional.', 'error');
      }
    });
  }

  onBatchCreated(emails: string[]): void {
    this.isRegistering.set(true);
    this.studentsService.batchCreateStudents(emails).subscribe({
      next: (res) => {
        this.isRegistering.set(false);
        this.closeRegisterModal();
        this.showToast(`${res.created_count} estudiantes creados exitosamente.`, 'success');
        this.loadStudents();
      },
      error: (err) => {
        this.isRegistering.set(false);
        this.showToast(err.error?.error || err.error?.message || 'Error al registrar estudiantes en lote.', 'error');
      }
    });
  }

  openRosterEnrollModal(): void { this.showRosterEnrollModal.set(true); }
  closeRosterEnrollModal(): void { this.showRosterEnrollModal.set(false); }

  onRosterEnrolled(payload: { subjectId: string; emails: string[] }): void {
    this.isEnrollingRoster.set(true);
    this.studentsService.rosterEnroll(payload.subjectId, payload.emails).subscribe({
      next: (res) => {
        this.isEnrollingRoster.set(false);
        this.closeRosterEnrollModal();
        this.showToast(
          `${res.enrolled_students} estudiantes matriculados (${res.created_students} cuentas nuevas creadas).`,
          'success'
        );
        this.loadStudents();
      },
      error: (err) => {
        this.isEnrollingRoster.set(false);
        this.showToast(err.error?.error || err.error?.message || 'Error al procesar la nómina del curso.', 'error');
      }
    });
  }

  requestToggleStatus(student: AdminStudentItem): void { this.studentToToggleStatus.set(student); }
  cancelToggleStatus(): void { this.studentToToggleStatus.set(null); }

  confirmToggleStatus(student: AdminStudentItem): void {
    const nextStatus: 'active' | 'suspended' = student.status === 'suspended' ? 'active' : 'suspended';
    const reason = nextStatus === 'suspended' ? 'Suspensión administrativa preventiva' : '';
    this.isTogglingStatus.set(true);
    this.studentsService.updateStudentStatus(student.id, nextStatus, reason).subscribe({
      next: () => {
        this.isTogglingStatus.set(false);
        this.cancelToggleStatus();
        this.showToast(
          nextStatus === 'active' ? `Acceso restablecido para ${student.first_name} ${student.last_name}.` : `Cuenta de ${student.first_name} ${student.last_name} suspendida.`,
          'success'
        );
        this.loadStudents();
      },
      error: (err) => {
        this.isTogglingStatus.set(false);
        this.showToast(err.error?.error || err.error?.message || 'Error al actualizar estado del estudiante.', 'error');
      }
    });
  }

  onRestartWorkspace(workspaceId: string): void {
    this.actionInProgressWorkspaceId.set(workspaceId);
    this.studentsService.restartWorkspace(workspaceId).subscribe({
      next: () => {
        this.actionInProgressWorkspaceId.set(null);
        this.showToast('Entorno reiniciado en el servidor.', 'success');
        if (this.studentForCourses()) this.openCoursesModal(this.studentForCourses()!);
        this.loadStudents();
      },
      error: (err) => {
        this.actionInProgressWorkspaceId.set(null);
        this.showToast(err.error?.error || err.error?.message || 'Error al reiniciar entorno de trabajo.', 'error');
      }
    });
  }

  onPauseWorkspace(workspaceId: string): void {
    this.actionInProgressWorkspaceId.set(workspaceId);
    this.studentsService.pauseWorkspace(workspaceId).subscribe({
      next: () => {
        this.actionInProgressWorkspaceId.set(null);
        this.showToast('Entorno pausado y memoria RAM liberada.', 'success');
        if (this.studentForCourses()) this.openCoursesModal(this.studentForCourses()!);
        this.loadStudents();
      },
      error: (err) => {
        this.actionInProgressWorkspaceId.set(null);
        this.showToast(err.error?.error || err.error?.message || 'Error al pausar entorno de trabajo.', 'error');
      }
    });
  }

  showToast(msg: string, type: 'success' | 'error' = 'success'): void {
    this.toastMessage.set(msg);
    this.toastType.set(type);
    setTimeout(() => {
      this.toastMessage.set(null);
    }, 4500);
  }
}
