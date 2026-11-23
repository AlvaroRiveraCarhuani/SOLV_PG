import { Component, inject, signal, computed, viewChild, OnInit, OnDestroy, HostListener } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { AdminTeachersService, TeacherCourse } from '../services/admin-teachers.service';
import { TeacherInviteModalComponent } from './components/teacher-invite-modal/teacher-invite-modal.component';
import { ConfirmModalComponent } from '@shared/components/confirm-modal/confirm-modal.component';
import { ContextHelpService } from '@shared/help/context-help.service';
import { ProductTourService } from '@shared/help/product-tour.service';
import { InteractiveTutorialService } from '@shared/help/interactive-tutorial.service';
import { buildTeachersTour, buildTeachersTutorial } from './teachers-help.definitions';
import { TeacherTableComponent } from './components/teacher-table/teacher-table.component';
import { TeacherInviteFeedbackComponent } from './components/teacher-invite-feedback/teacher-invite-feedback.component';
import { DrawerComponent } from '@shared/components/drawer/drawer.component';
import { FormFieldComponent } from '@shared/components/form-field/form-field.component';
import { ComboboxComponent, ComboboxOption } from '@shared/components/combobox/combobox.component';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { DateTextPipe } from '@shared/pipes/date-text.pipe';
import { StatusTabItem } from '@shared/components/status-tabs/status-tabs.component';
import { SortDirection, SortState } from '@shared/components/sort-header/sort-header.types';
import {
  TeacherItem,
  TeacherInvitationPayload,
  DirectTeacherPayload,
  BatchDirectTeacherPayload
} from '@core/models/admin.model';
import {
  LucideUserPlus,
  LucideCheckCircle,
  LucideAlertCircle,
  LucideAlertTriangle,
  LucideBookOpen,
  LucideUsers,
  LucideArrowRightLeft,
  LucideUserMinus
} from '@lucide/angular';
import { formatSolvDate } from '@shared/pipes/date-text.pipe';

const ENGLISH_MONTH_INDEX: Record<string, number> = {
  jan: 0, feb: 1, mar: 2, apr: 3, may: 4, jun: 5,
  jul: 6, aug: 7, sep: 8, oct: 9, nov: 10, dec: 11
};

interface ToastData {
  text: string;
  isError: boolean;
}

interface TokenFeedbackData {
  title: string;
  email: string;
  inviteUrl: string;
}

const TEACHERS_HELP_KEY = 'teachers-invitations';

@Component({
  selector: 'admin-teachers',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    MachineDataDirective,
    DateTextPipe,
    TeacherInviteModalComponent,
    ConfirmModalComponent,
    TeacherTableComponent,
    TeacherInviteFeedbackComponent,
    DrawerComponent,
    FormFieldComponent,
    ComboboxComponent,
    LucideUserPlus,
    LucideCheckCircle,
    LucideAlertCircle,
    LucideAlertTriangle,
    LucideBookOpen,
    LucideUsers,
    LucideArrowRightLeft,
    LucideUserMinus
  ],
  templateUrl: './admin-teachers.component.html',
  styleUrl: './admin-teachers.component.scss'
})
export class AdminTeachersComponent implements OnInit, OnDestroy {
  private teachersService = inject(AdminTeachersService);
  private readonly help = inject(ContextHelpService);
  private readonly productTour = inject(ProductTourService);
  private readonly tutorial = inject(InteractiveTutorialService);

  teachers = this.teachersService.teachers;
  isLoading = this.teachersService.isLoading;

  showInviteModal = signal<boolean>(false);
  /** Modal de alta visible, para que el tutorial compruebe el correo real. */
  readonly inviteModal = viewChild(TeacherInviteModalComponent);
  /** Se registró un docente en esta visita; lo lee el tutorial. */
  readonly registeredInSession = signal<boolean>(false);
  searchTerm = signal<string>('');
  statusFilter = signal<string>('all');
  originFilter = signal<string>('all');
  sortState = signal<SortState>({ column: null, direction: 'none' });

  onSortChange(event: { column: string; direction: SortDirection }): void {
    this.sortState.set({
      column: event.direction === 'none' ? null : event.column,
      direction: event.direction
    });
    this.currentPage.set(1);
  }

  originComboboxOptions = computed<ComboboxOption[]>(() => [
    { id: 'all', label: 'Todos los orígenes', value: 'all' },
    { id: 'manual', label: 'Manual', value: 'manual' },
    { id: 'bulk_import', label: 'Importación masiva', value: 'bulk_import' },
    { id: 'gclassroom', label: 'Google Classroom', value: 'gclassroom' }
  ]);

  selectedOriginLabel = computed<string>(() => {
    const val = this.originFilter();
    const match = this.originComboboxOptions().find(o => o.value === val);
    return match ? match.label : 'Todos los orígenes';
  });

  formatInvitedAt(value: string): string {
    const legacyDate = /^(\d{2})-([A-Za-z]{3})-(\d{4})$/.exec(value);
    if (legacyDate) {
      const monthIndex = ENGLISH_MONTH_INDEX[legacyDate[2].toLowerCase()] ?? -1;
      if (monthIndex >= 0) {
        const iso = `${legacyDate[3]}-${String(monthIndex + 1).padStart(2, '0')}-${legacyDate[1]}`;
        return formatSolvDate(iso, 'daymonthyear') ?? value;
      }
      return value;
    }
    return formatSolvDate(value, 'daymonthyear') ?? value;
  }

  onOriginChange(value: string): void {
    this.originFilter.set(value);
    this.currentPage.set(1);
    this.loadTeachers();
  }

  // Drawer Lateral de Detalle y Reasignación
  selectedTeacherForDrawer = signal<TeacherItem | null>(null);

  openTeacherDrawer(teacher: TeacherItem): void {
    this.activeMenuId.set(null);
    this.cancelReassign();
    this.selectedTeacherForDrawer.set(teacher);
    this.reloadTeacherCourses(teacher.id);
  }

  closeTeacherDrawer(): void {
    this.selectedTeacherForDrawer.set(null);
    this.cancelReassign();
  }

  getTeacherOriginLabel(origin: string | undefined): string {
    if (origin === 'gclassroom') return 'Google Classroom';
    if (origin === 'bulk_import') return 'Importación masiva';
    return 'Manual';
  }

  activeTeacherComboboxOptions = computed<ComboboxOption[]>(() => {
    return this.activeTeachersList().map(t => ({
      id: t.id,
      label: t.full_name,
      value: t.id,
      meta: t.email
    }));
  });

  selectedNewTeacherLabel = computed<string>(() => {
    const id = this.selectedNewTeacherId();
    if (!id) return 'Seleccione el nuevo docente...';
    const match = this.activeTeachersList().find(t => t.id === id);
    return match ? match.full_name : 'Seleccione el nuevo docente...';
  });

  onNewTeacherSelected(opt: ComboboxOption): void {
    this.selectedNewTeacherId.set(opt.value || '');
  }

  // Paginación
  currentPage = signal<number>(1);
  pageSize = 10;

  // Acciones y feedback
  activeMenuId = signal<string | null>(null);
  tokenFeedback = signal<TokenFeedbackData | null>(null);
  copied = signal<boolean>(false);
  toastMessage = signal<ToastData | null>(null);
  teacherToDelete = signal<TeacherItem | null>(null);

  // Materias asignadas y reasignación en Drawer
  teacherCourses = signal<TeacherCourse[]>([]);
  isLoadingCourses = signal<boolean>(false);

  // Estados de reasignación activa
  reassigningCourse = signal<TeacherCourse | null>(null);
  selectedNewTeacherId = signal<string>('');
  reassignReason = signal<string>('');
  isSubmittingReassign = signal<boolean>(false);

  private searchDebounceTimer?: ReturnType<typeof setTimeout>;
  private toastDismissTimer?: ReturnType<typeof setTimeout>;
  private copyResetTimer?: ReturnType<typeof setTimeout>;

  // Carga académica total del docente seleccionado en drawer
  totalStudentsForTeacher = computed(() => {
    return this.teacherCourses().reduce((sum, c) => sum + (c.students_count || 0), 0);
  });

  // Lista de otros docentes activos elegibles como nuevo titular (excluyendo el docente actual)
  activeTeachersList = computed(() => {
    const currentId = this.selectedTeacherForDrawer()?.id;
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

  // 4 Pestañas de estado limpias con contadores reales en tono neutro
  teacherStatusTabs = computed<StatusTabItem[]>(() => {
    const c = this.counts();
    return [
      { id: 'all', label: 'Todos', count: c.all, badgeVariant: 'neutral' },
      { id: 'active', label: 'Activos', count: c.active, badgeVariant: 'neutral' },
      { id: 'no_courses', label: 'Sin materias', count: c.noCourses, badgeVariant: 'neutral' },
      { id: 'invitations', label: 'Invitaciones', count: c.pending + c.expired, badgeVariant: 'neutral' }
    ];
  });

  // Lista filtrada y ordenada en cliente
  filteredTeachersList = computed(() => {
    const term = this.searchTerm().toLowerCase().trim();
    const status = this.statusFilter();
    const origin = this.originFilter();
    const sort = this.sortState();

    let list = this.teachers().filter(t => {
      if (status === 'no_courses') {
        if (t.status !== 'active' || (t.active_courses ?? 0) > 0) return false;
      } else if (status === 'invitations') {
        if (t.status !== 'pending' && t.status !== 'expired') return false;
      } else if (status !== 'all' && t.status !== status) {
        return false;
      }
      if (origin !== 'all' && (t.origin as string) !== origin) return false;
      if (term) {
        const matchName = t.full_name.toLowerCase().includes(term);
        const matchEmail = t.email.toLowerCase().includes(term);
        if (!matchName && !matchEmail) return false;
      }
      return true;
    });

    if (sort.column && sort.direction !== 'none') {
      const modifier = sort.direction === 'asc' ? 1 : -1;
      list = [...list].sort((a, b) => {
        switch (sort.column) {
          case 'name':
            return (a.full_name || '').localeCompare(b.full_name || '', undefined, { sensitivity: 'base' }) * modifier;
          case 'status':
            return (a.status || '').localeCompare(b.status || '', undefined, { sensitivity: 'base' }) * modifier;
          case 'courses':
            return ((a.active_courses ?? 0) - (b.active_courses ?? 0)) * modifier;
          case 'date': {
            const dA = a.invited_at ? new Date(a.invited_at).getTime() : 0;
            const dB = b.invited_at ? new Date(b.invited_at).getTime() : 0;
            return (dA - dB) * modifier;
          }
          default:
            return 0;
        }
      });
    }

    return list;
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

  ngOnInit(): void {
    this.help.registerScreen(TEACHERS_HELP_KEY);
    this.help.registerGuides(TEACHERS_HELP_KEY, {
      tour: {
        label: 'Recorrido visual',
        hint: 'Señala los contadores, el alta y las acciones por fila.',
        run: () => void this.productTour.start(buildTeachersTour())
      },
      tutorial: {
        label: 'Tutorial guiado',
        hint: 'Registra un docente y comprueba el correo antes de continuar.',
        run: () => this.startTeachersTutorial()
      }
    });
    this.loadTeachers();
  }

  ngOnDestroy(): void {
    this.help.unregisterGuides(TEACHERS_HELP_KEY);
    this.help.unregisterScreen(TEACHERS_HELP_KEY);
    if (this.searchDebounceTimer) clearTimeout(this.searchDebounceTimer);
    if (this.toastDismissTimer) clearTimeout(this.toastDismissTimer);
    if (this.copyResetTimer) clearTimeout(this.copyResetTimer);
  }

  @HostListener('document:keydown.escape')
  onEscapeKey(): void {
    if (this.showInviteModal()) {
      this.showInviteModal.set(false);
    }
    if (this.tokenFeedback()) {
      this.tokenFeedback.set(null);
    }
    if (this.selectedTeacherForDrawer()) {
      this.closeTeacherDrawer();
    }
    if (this.teacherToDelete()) {
      this.cancelDeleteTeacher();
    }
    this.activeMenuId.set(null);
  }

  loadTeachers(): void {
    this.teachersService.fetchTeachers(
      this.searchTerm(),
      undefined,
      this.originFilter() !== 'all' ? this.originFilter() : undefined
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
  }

  goToPage(page: number): void {
    if (page >= 1 && page <= this.totalPages()) {
      this.currentPage.set(page);
    }
  }

  /** Tutorial del alta: exige modal abierto, correo válido y registro confirmado. */
  startTeachersTutorial(): void {
    this.tutorial.start(
      buildTeachersTutorial({
        inviteModalOpen: () => this.showInviteModal(),
        emailValid: () => this.inviteModal()?.isValidEmail() ?? false,
        hasRegistered: () => this.registeredInSession()
      })
    );
  }

  openInviteModal(): void {
    this.showInviteModal.set(true);
  }

  toggleMenu(teacherId: string): void {
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

  requestDeleteTeacher(teacher: TeacherItem): void {
    this.activeMenuId.set(null);
    this.teacherToDelete.set(teacher);
  }

  cancelDeleteTeacher(): void {
    this.teacherToDelete.set(null);
  }

  executeDeleteTeacher(teacher: TeacherItem): void {
    this.teachersService.deleteTeacher(teacher.id).subscribe(success => {
      this.cancelDeleteTeacher();
      if (success) {
        if (this.selectedTeacherForDrawer()?.id === teacher.id) {
          this.closeTeacherDrawer();
        }
        this.showToast(`Rol docente de ${teacher.full_name} revocado exitosamente`);
      } else {
        this.showToast('No se pudo revocar el rol docente', true);
      }
    });
  }

  getRevokeConfirmMessage(teacher: TeacherItem): string {
    const courses = teacher.active_courses ?? 0;
    if (courses > 0) {
      return `¿Confirma la revocación del rol docente de ${teacher.full_name} (${teacher.email})? La cuenta pasará a tener rol de estudiante. ADVERTENCIA: Sus ${courses} ${courses === 1 ? 'materia asignada quedará' : 'materias asignadas quedarán'} sin docente titular. Se recomienda reasignar las materias antes de revocar el rol.`;
    }
    return `¿Confirma la revocación del rol docente de ${teacher.full_name} (${teacher.email})? La cuenta pasará a tener rol de estudiante y perderá el acceso a las funciones docentes.`;
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

  onDirectTeacherSubmitted(payload: DirectTeacherPayload): void {
    this.teachersService.directRegisterTeacher(payload).subscribe(res => {
      this.showInviteModal.set(false);
      if (res) {
        this.registeredInSession.set(true);
        this.showToast(`Docente ${payload.email} registrado exitosamente`);
        this.loadTeachers();
      } else {
        this.showToast('No se pudo completar el registro del docente', true);
      }
    });
  }

  onBatchTeachersSubmitted(payload: BatchDirectTeacherPayload): void {
    this.teachersService.batchDirectRegister(payload).subscribe(res => {
      this.showInviteModal.set(false);
      if (res) {
        this.registeredInSession.set(true);
        this.showToast(`${res.registered_count} docentes registrados exitosamente`);
        this.loadTeachers();
      } else {
        this.showToast('No se pudo procesar la importación masiva', true);
      }
    });
  }

  copyTeacherEmail(email: string): void {
    this.showToast('Correo copiado al portapapeles');
    this.activeMenuId.set(null);
  }

  toggleReassignCourse(course: TeacherCourse): void {
    if (this.reassigningCourse()?.id === course.id) {
      this.cancelReassign();
    } else {
      this.startReassign(course);
    }
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
    const currentTeacher = this.selectedTeacherForDrawer();

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
