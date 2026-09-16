import { Component, inject, signal, computed, OnInit, OnDestroy, HostListener } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { AdminTeachersService, TeacherCourse } from '../services/admin-teachers.service';
import { TeacherInviteModalComponent } from './components/teacher-invite-modal/teacher-invite-modal.component';
import { TeacherItem, TeacherInvitationPayload } from '@core/models/admin.model';
import { 
  LucideUserPlus, 
  LucideSearch, 
  LucideLock, 
  LucideCheckCircle, 
  LucideClock, 
  LucideAlertCircle, 
  LucideSend, 
  LucideRotateCcw,
  LucideCopy,
  LucideTrash2,
  LucideBookOpen,
  LucideUsers,
  LucideChevronLeft,
  LucideChevronRight,
  LucideArrowRightLeft,
  LucidePlus
} from '@lucide/angular';

interface TokenFeedbackData {
  title: string;
  email: string;
  inviteUrl: string;
}

interface ToastData {
  text: string;
  isError: boolean;
}

@Component({
  selector: 'solv-admin-teachers',
  standalone: true,
  imports: [
    CommonModule, 
    FormsModule, 
    TeacherInviteModalComponent,
    LucideUserPlus, 
    LucideSearch, 
    LucideLock, 
    LucideCheckCircle, 
    LucideClock, 
    LucideAlertCircle, 
    LucideSend, 
    LucideRotateCcw,
    LucideCopy,
    LucideTrash2,
    LucideBookOpen,
    LucideUsers,
    LucideChevronLeft,
    LucideChevronRight,
    LucideArrowRightLeft,
    LucidePlus
  ],
  templateUrl: './admin-teachers.component.html',
  styleUrl: './admin-teachers.component.scss'
})
export class AdminTeachersComponent implements OnInit, OnDestroy {
  private teachersService = inject(AdminTeachersService);

  teachers = this.teachersService.teachers;
  isLoading = this.teachersService.isLoading;

  showInviteModal = signal<boolean>(false);
  searchTerm = signal<string>('');
  statusFilter = signal<string>('all');
  originFilter = signal<string>('all');

  // Paginación
  currentPage = signal<number>(1);
  pageSize = 10;

  // Acciones y feedback
  activeMenuId = signal<string | null>(null);
  tokenFeedback = signal<TokenFeedbackData | null>(null);
  copied = signal<boolean>(false);
  toastMessage = signal<ToastData | null>(null);

  // Modal de materias asignadas y reasignación
  selectedTeacherForCourses = signal<TeacherItem | null>(null);
  teacherCourses = signal<TeacherCourse[]>([]);
  isLoadingCourses = signal<boolean>(false);

  // Estados de reasignación activa
  reassigningCourse = signal<TeacherCourse | null>(null);
  selectedNewTeacherId = signal<string>('');
  reassignReason = signal<string>('');
  isSubmittingReassign = signal<boolean>(false);

  // Estados para asignar materia a docente sin materias
  isAssigningNewCourse = signal<boolean>(false);
  selectedCourseToAssign = signal<string>('');
  availableTenantCourses = signal<{ id: string; name: string; teacher_name: string }[]>([]);
  isLoadingAvailableCourses = signal<boolean>(false);

  private searchDebounceTimer?: ReturnType<typeof setTimeout>;
  private toastDismissTimer?: ReturnType<typeof setTimeout>;
  private copyResetTimer?: ReturnType<typeof setTimeout>;

  // Carga académica total del docente seleccionado
  totalStudentsForTeacher = computed(() => {
    return this.teacherCourses().reduce((sum, c) => sum + (c.students_count || 0), 0);
  });

  // Lista de docentes activos elegibles como nuevo titular (excluyendo el docente actual)
  activeTeachersList = computed(() => {
    const currentId = this.selectedTeacherForCourses()?.id;
    return this.teachers().filter(t => t.status === 'active' && t.id !== currentId);
  });

  // Contadores de estado calculados
  counts = computed(() => {
    const list = this.teachers();
    return {
      all: list.length,
      active: list.filter(t => t.status === 'active').length,
      noCourses: list.filter(t => t.status === 'active' && (t.active_courses ?? 0) === 0).length,
      pending: list.filter(t => t.status === 'pending').length,
      expired: list.filter(t => t.status === 'expired').length
    };
  });

  // Lista filtrada en cliente por si la búsqueda/estado cambia en vista
  filteredTeachersList = computed(() => {
    const term = this.searchTerm().toLowerCase().trim();
    const status = this.statusFilter();
    const origin = this.originFilter();

    return this.teachers().filter(t => {
      if (status === 'no_courses') {
        if (t.status !== 'active' || (t.active_courses ?? 0) > 0) return false;
      } else if (status !== 'all' && t.status !== status) {
        return false;
      }
      if (origin !== 'all' && t.origin !== origin) return false;
      if (term) {
        const matchName = t.full_name.toLowerCase().includes(term);
        const matchEmail = t.email.toLowerCase().includes(term);
        if (!matchName && !matchEmail) return false;
      }
      return true;
    });
  });

  // Lista paginada
  paginatedTeachers = computed(() => {
    const list = this.filteredTeachersList();
    const start = (this.currentPage() - 1) * this.pageSize;
    return list.slice(start, start + this.pageSize);
  });

  totalPages = computed(() => {
    return Math.max(1, Math.ceil(this.filteredTeachersList().length / this.pageSize));
  });

  displayedPages = computed(() => {
    const total = this.totalPages();
    const current = this.currentPage();
    const pages: number[] = [];

    const start = Math.max(1, current - 2);
    const end = Math.min(total, current + 2);

    for (let i = start; i <= end; i++) {
      pages.push(i);
    }
    return pages;
  });

  paginationDisplay = computed(() => {
    const total = this.filteredTeachersList().length;
    if (total === 0) return { from: 0, to: 0 };
    const from = (this.currentPage() - 1) * this.pageSize + 1;
    const to = Math.min(this.currentPage() * this.pageSize, total);
    return { from, to };
  });

  ngOnInit(): void {
    this.loadTeachers();
  }

  ngOnDestroy(): void {
    if (this.searchDebounceTimer) clearTimeout(this.searchDebounceTimer);
    if (this.toastDismissTimer) clearTimeout(this.toastDismissTimer);
    if (this.copyResetTimer) clearTimeout(this.copyResetTimer);
  }

  @HostListener('document:click')
  onDocumentClick(): void {
    this.activeMenuId.set(null);
  }

  loadTeachers(): void {
    const statusParam = this.statusFilter() === 'no_courses' ? 'active' : this.statusFilter();
    this.teachersService.fetchTeachers(
      this.searchTerm(),
      statusParam,
      this.originFilter()
    );
  }

  onSearchChange(value: string): void {
    this.searchTerm.set(value);
    this.currentPage.set(1);
    if (this.searchDebounceTimer) {
      clearTimeout(this.searchDebounceTimer);
    }
    this.searchDebounceTimer = setTimeout(() => {
      this.loadTeachers();
    }, 300);
  }

  clearSearch(): void {
    this.searchTerm.set('');
    this.currentPage.set(1);
    this.loadTeachers();
  }

  onStatusChange(value: string): void {
    this.statusFilter.set(value);
    this.currentPage.set(1);
    this.loadTeachers();
  }

  onOriginChange(value: string): void {
    this.originFilter.set(value);
    this.currentPage.set(1);
    this.loadTeachers();
  }

  goToPage(page: number): void {
    if (page >= 1 && page <= this.totalPages()) {
      this.currentPage.set(page);
    }
  }

  openInviteModal(): void {
    this.showInviteModal.set(true);
  }

  getInitials(name: string): string {
    if (!name) return 'PR';
    return name
      .split(' ')
      .slice(0, 2)
      .map(w => w[0] || '')
      .join('')
      .toUpperCase();
  }

  toggleMenu(teacherId: string, event: Event): void {
    event.stopPropagation();
    this.activeMenuId.update(current => current === teacherId ? null : teacherId);
  }

  resendInvitation(teacher: TeacherItem): void {
    this.teachersService.resendInvitation(teacher.id).subscribe(res => {
      this.showToast(`Invitación reenviada a ${teacher.email}`);
      if (res && res.invite_url) {
        this.tokenFeedback.set({
          title: 'Invitación Reenviada Exitosamente',
          email: teacher.email,
          inviteUrl: res.invite_url
        });
      }
    });
  }

  renewInvitation(teacher: TeacherItem): void {
    this.teachersService.renewInvitation(teacher.id).subscribe(res => {
      this.showToast(`Invitación renovada por 72 horas para ${teacher.email}`);
      if (res && res.invite_url) {
        this.tokenFeedback.set({
          title: 'Invitación Renovada por 72 Horas',
          email: teacher.email,
          inviteUrl: res.invite_url
        });
      }
    });
  }

  revokeInvitation(teacher: TeacherItem): void {
    this.teachersService.deleteInvitation(teacher.id).subscribe(success => {
      if (success) {
        this.showToast('Invitación revocada correctamente');
      } else {
        this.showToast('No se pudo revocar la invitación', true);
      }
    });
  }

  onInviteSubmitted(payload: TeacherInvitationPayload): void {
    this.teachersService.inviteTeacher(payload).subscribe(res => {
      this.showInviteModal.set(false);
      this.showToast(`Invitación enviada a ${payload.email}`);
      if (res && res.invite_url) {
        this.tokenFeedback.set({
          title: 'Invitación Emitida Exitosamente',
          email: res.email,
          inviteUrl: res.invite_url
        });
      }
    });
  }

  copyTeacherEmail(email: string): void {
    if (navigator?.clipboard) {
      navigator.clipboard.writeText(email);
      this.showToast('Correo copiado al portapapeles');
    }
    this.activeMenuId.set(null);
  }

  openTeacherCoursesModal(teacher: TeacherItem): void {
    this.activeMenuId.set(null);
    this.cancelReassign();
    this.cancelAssignCourseToTeacher();
    this.selectedTeacherForCourses.set(teacher);
    this.reloadTeacherCourses(teacher.id);
  }

  closeTeacherCoursesModal(): void {
    this.selectedTeacherForCourses.set(null);
    this.cancelReassign();
    this.cancelAssignCourseToTeacher();
  }

  startReassign(course: TeacherCourse): void {
    this.reassigningCourse.set(course);
    this.selectedNewTeacherId.set('');
    this.reassignReason.set('');
  }

  cancelReassign(): void {
    this.reassigningCourse.set(null);
    this.selectedNewTeacherId.set('');
    this.reassignReason.set('');
  }

  confirmReassignCourse(): void {
    const course = this.reassigningCourse();
    const targetTeacherId = this.selectedNewTeacherId();
    const reason = this.reassignReason();
    const currentTeacher = this.selectedTeacherForCourses();

    if (!course || !targetTeacherId) return;

    this.isSubmittingReassign.set(true);
    this.teachersService.reassignCourseTeacher(course.id, targetTeacherId, reason)
      .subscribe({
        next: () => {
          this.isSubmittingReassign.set(false);
          this.cancelReassign();
          this.showToast('Materia reasignada exitosamente');
          if (currentTeacher) {
            this.reloadTeacherCourses(currentTeacher.id);
          }
          this.loadTeachers();
        },
        error: () => {
          this.isSubmittingReassign.set(false);
          this.showToast('No se pudo reasignar la materia', true);
        }
      });
  }

  startAssignCourseToTeacher(): void {
    this.isAssigningNewCourse.set(true);
    this.selectedCourseToAssign.set('');
    this.reassignReason.set('');
    this.isLoadingAvailableCourses.set(true);
    this.teachersService.getAllAvailableCourses().subscribe(courses => {
      this.availableTenantCourses.set(courses);
      this.isLoadingAvailableCourses.set(false);
    });
  }

  cancelAssignCourseToTeacher(): void {
    this.isAssigningNewCourse.set(false);
    this.selectedCourseToAssign.set('');
    this.reassignReason.set('');
  }

  confirmAssignCourseToTeacher(): void {
    const courseId = this.selectedCourseToAssign();
    const teacher = this.selectedTeacherForCourses();
    const reason = this.reassignReason();

    if (!courseId || !teacher) return;

    this.isSubmittingReassign.set(true);
    this.teachersService.reassignCourseTeacher(courseId, teacher.id, reason)
      .subscribe({
        next: () => {
          this.isSubmittingReassign.set(false);
          this.cancelAssignCourseToTeacher();
          this.showToast('Materia asignada al docente exitosamente');
          this.reloadTeacherCourses(teacher.id);
          this.loadTeachers();
        },
        error: () => {
          this.isSubmittingReassign.set(false);
          this.showToast('No se pudo asignar la materia', true);
        }
      });
  }

  private reloadTeacherCourses(teacherId: string): void {
    this.isLoadingCourses.set(true);
    this.teachersService.getTeacherCourses(teacherId).subscribe(courses => {
      this.teacherCourses.set(courses);
      this.isLoadingCourses.set(false);
    });
  }

  copyTokenUrl(url: string): void {
    if (navigator?.clipboard) {
      navigator.clipboard.writeText(url);
    }
    this.copied.set(true);
    if (this.copyResetTimer) clearTimeout(this.copyResetTimer);
    this.copyResetTimer = setTimeout(() => {
      this.copied.set(false);
    }, 2500);
  }

  showToast(text: string, isError = false): void {
    this.toastMessage.set({ text, isError });
    if (this.toastDismissTimer) clearTimeout(this.toastDismissTimer);
    this.toastDismissTimer = setTimeout(() => {
      this.toastMessage.set(null);
    }, 3500);
  }
}
