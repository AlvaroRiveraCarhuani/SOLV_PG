import { Component, OnDestroy, OnInit, computed, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute } from '@angular/router';
import {
  LucideAlertTriangle,
  LucideCheckCircle2,
  LucideLaptop,
  LucideLoader2,
  LucideLock,
  LucideLockOpen,
  LucideMoreHorizontal,
  LucideRotateCcw,
  LucideSend,
  LucidePause,
  LucidePlay
} from '@lucide/angular';

import { DateTextPipe } from '@shared/pipes/date-text.pipe';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { DismissibleDirective } from '@shared/directives/dismissible.directive';
import { ComboboxComponent, ComboboxOption } from '@shared/components/combobox/combobox.component';
import { SearchBarComponent } from '@shared/components/search-bar/search-bar.component';
import { StatusTabItem, StatusTabsComponent } from '@shared/components/status-tabs/status-tabs.component';
import { VerdictBadgeComponent } from '@shared/components/verdict-badge/verdict-badge.component';
import { SortHeaderComponent } from '@shared/components/sort-header/sort-header.component';
import { SortDirection, SortState } from '@shared/components/sort-header/sort-header.types';
import { ConfirmModalComponent } from '@shared/components/confirm-modal/confirm-modal.component';

import { ClassSessionService } from '../services/class-session.service';
import { ClassAdvanceService } from '../services/class-advance.service';
import { TeacherDashboardService } from '../services/teacher-dashboard.service';
import { TeacherCourseService } from '../services/teacher-course.service';
import { TeacherEnvironmentService } from '../services/teacher-environment.service';
import {
  ClassSession,
  ClassSessionStudent,
  ClassSessionStudentState,
  SessionAdvance
} from '../models/teacher.models';
import { ClassAdvanceModalComponent } from './class-advance-modal/class-advance-modal.component';
import { AdvanceRecipientsModalComponent } from './advance-recipients-modal/advance-recipients-modal.component';
import { ClassHistoryComponent } from './class-history/class-history.component';
import { UnfreezeStudentModalComponent } from './unfreeze-student-modal/unfreeze-student-modal.component';

const REFRESH_MS = 5000;

/** Duraciones de clase que el docente elige al abrir. Al vencer, el aula se cierra sola. */
const DURATION_OPTIONS: ComboboxOption[] = [
  { id: '60', label: '1 hora', value: '60' },
  { id: '90', label: '1 hora 30 min', value: '90' },
  { id: '120', label: '2 horas', value: '120' },
  { id: '180', label: '3 horas', value: '180' }
];

const DEFAULT_DURATION = '90';

/**
 * La ventana de disponibilidad se abre sola al cerrar el aula. El campo de la
 * pantalla arranca en esos mismos 30 minutos y el docente decide cuánto sumar,
 * sin bloques fijos: si la clase necesita 20, escribe 20.
 */
const DEFAULT_GRACE_MINUTES = 30;
const MIN_GRACE_MINUTES = 1;
const MAX_GRACE_MINUTES = 480;

const STATE_LABELS: Record<ClassSessionStudentState, string> = {
  trabajando: 'Trabajando',
  entregado: 'Entregó',
  trabado: 'Se trabó',
  sin_empezar: 'Sin empezar',
  ausente: 'Salió'
};

/**
 * Pantalla "Aula": responde "¿quién está, quién falta y quién se trabó?" mientras
 * ocurre la clase.
 *
 * El aula es una envoltura opcional de la práctica. Sin sesión abierta la
 * pantalla ofrece abrirla y, si hay entornos encendidos, los muestra sin lista
 * de faltantes: no hay conjunto contra el cual contar.
 */
@Component({
  selector: 'teacher-aula',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    LucideAlertTriangle,
    LucideCheckCircle2,
    LucideLaptop,
    LucideLoader2,
    LucideLock,
    LucideLockOpen,
    LucideMoreHorizontal,
    LucideRotateCcw,
    LucideSend,
    LucidePause,
    LucidePlay,
    DateTextPipe,
    MachineDataDirective,
    DismissibleDirective,
    ComboboxComponent,
    SearchBarComponent,
    StatusTabsComponent,
    VerdictBadgeComponent,
    SortHeaderComponent,
    ClassAdvanceModalComponent,
    AdvanceRecipientsModalComponent,
    ClassHistoryComponent,
    UnfreezeStudentModalComponent,
    ConfirmModalComponent
  ],
  templateUrl: './aula.component.html',
  styleUrl: './aula.component.scss'
})
export class AulaComponent implements OnInit, OnDestroy {
  private readonly route = inject(ActivatedRoute);
  private readonly sessionService = inject(ClassSessionService);
  private readonly advanceService = inject(ClassAdvanceService);
  private readonly dashboardService = inject(TeacherDashboardService);
  private readonly courseService = inject(TeacherCourseService);
  private readonly environment = inject(TeacherEnvironmentService);

  readonly session = this.sessionService.session;
  readonly board = this.sessionService.board;
  readonly isLoading = this.sessionService.isLoading;
  readonly isOpening = this.sessionService.isOpening;
  readonly error = this.sessionService.error;
  readonly lastClosedAt = this.sessionService.lastClosedAt;
  readonly lastClosedSession = this.sessionService.lastClosedSession;
  readonly history = this.sessionService.history;
  readonly isLoadingHistory = this.sessionService.isLoadingHistory;
  readonly historyError = this.sessionService.historyError;
  /** Aula cuyo historial se está mirando: la abierta o la última cerrada. */
  readonly historySession = signal<ClassSession | null>(null);
  readonly isChangingFreeze = this.sessionService.isChangingFreeze;
  readonly isExtendingGrace = this.sessionService.isExtendingGrace;
  readonly freezeReport = this.sessionService.lastFreezeReport;
  readonly isFreezingExam = this.sessionService.isFreezingExam;
  readonly closedBoard = this.sessionService.closedBoard;
  readonly isLoadingClosedBoard = this.sessionService.isLoadingClosedBoard;
  /** Alumno al que se le va a devolver la escritura, si el docente lo pidió. */
  readonly unfreezeTarget = signal<ClassSessionStudent | null>(null);

  readonly lastEmission = this.advanceService.lastEmission;
  readonly sessionAdvances = this.advanceService.sessionAdvances;
  readonly isLoadingAdvances = this.advanceService.isLoadingAdvances;

  readonly subjects = this.dashboardService.courses;

  readonly activeTab = signal<string>('todos');
  readonly searchTerm = signal<string>('');
  readonly selectedSubjectId = signal<string>('');
  readonly selectedExerciseId = signal<string>('');
  readonly selectedDuration = signal<string>(DEFAULT_DURATION);
  readonly durationOptions = DURATION_OPTIONS;
  readonly practiceOptions = signal<ComboboxOption[]>([]);
  readonly isAdvanceModalOpen = signal<boolean>(false);
  readonly advanceDetail = signal<SessionAdvance | null>(null);
  readonly graceMinutes = signal<number | null>(DEFAULT_GRACE_MINUTES);
  readonly graceDeadline = signal<string>('');
  readonly graceError = signal<string>('');
  readonly isOpeningIde = signal<boolean>(false);
  readonly ideError = signal<string | null>(null);
  /** Menú del cierre: la acción que no se usa en caliente no ocupa un botón fijo. */
  readonly isMenuOpen = signal<boolean>(false);

  private refreshTimer?: ReturnType<typeof setInterval>;
  /** Práctica pedida por URL: se aplica cuando llegan las opciones de su materia. */
  private pendingExerciseId = '';

  /** Conteos del tablero. Sin aula, todo cero: la vista no inventa cifras. */
  readonly summary = computed(() => this.board()?.summary ?? {
    enrolled: 0,
    present: 0,
    not_started: 0,
    working: 0,
    submitted: 0,
    stuck: 0,
    absent: 0,
    without_submission: 0
  });

  readonly hasExceptions = computed(() => {
    const s = this.summary();
    return (
      s.stuck > 0 ||
      s.not_started > 0 ||
      s.without_submission > 0 ||
      s.absent > 0 ||
      this.studentsWithoutEnvironment() > 0
    );
  });

  /**
   * Solo se marca al alumno que está en el aula y, aun así, no tiene entorno
   * vivo: está presente y no puede trabajar. Quien todavía no empezó no es una
   * excepción, y quien entregó ya no necesita el entorno. Es la única
   * afirmación sobre contenedores que hace la pantalla, y sale del tablero.
   */
  showsEnvironmentWarning(student: ClassSessionStudent): boolean {
    return !student.environment_id && student.state === 'trabajando';
  }

  readonly studentsWithoutEnvironment = computed(
    () => (this.board()?.students ?? []).filter(student => this.showsEnvironmentWarning(student)).length
  );

  /**
   * Solo se ofrecen los estados con alguien adentro: seis pestañas con cinco
   * ceros ocupan una franja entera sin filtrar nada. El estado activo se
   * conserva aunque llegue a cero, para que el filtro elegido no desaparezca solo.
   */
  readonly tabs = computed<StatusTabItem[]>(() => {
    const s = this.summary();
    const active = this.activeTab();
    const all: StatusTabItem[] = [
      { id: 'todos', label: 'Todos', count: (this.board()?.students ?? []).length },
      { id: 'trabado', label: 'Trabados', count: s.stuck, badgeVariant: s.stuck > 0 ? 'error' : 'default' },
      { id: 'trabajando', label: 'Trabajando', count: s.working },
      { id: 'entregado', label: 'Entregaron', count: s.submitted },
      { id: 'sin_empezar', label: 'Sin empezar', count: s.not_started },
      { id: 'ausente', label: 'Salieron', count: s.absent }
    ];
    return all.filter(tab => tab.id === 'todos' || (tab.count ?? 0) > 0 || tab.id === active);
  });

  readonly sortState = signal<SortState>({ column: null, direction: 'none' });

  onSortChange(event: { column: string; direction: SortDirection }): void {
    this.sortState.set({
      column: event.direction === 'none' ? null : event.column,
      direction: event.direction
    });
  }

  readonly visibleStudents = computed<ClassSessionStudent[]>(() => {
    const students = this.board()?.students ?? [];
    const tab = this.activeTab();
    const term = this.searchTerm().toLowerCase().trim();
    const sort = this.sortState();

    let list = students.filter(student => {
      if (tab !== 'todos' && student.state !== tab) {
        return false;
      }
      if (!term) {
        return true;
      }
      return student.name.toLowerCase().includes(term) || student.email.toLowerCase().includes(term);
    });

    if (sort.column && sort.direction !== 'none') {
      const modifier = sort.direction === 'asc' ? 1 : -1;
      list = [...list].sort((a, b) => {
        switch (sort.column) {
          case 'name':
            return (a.name || '').localeCompare(b.name || '', undefined, { sensitivity: 'base' }) * modifier;
          case 'state':
            return (a.state || '').localeCompare(b.state || '', undefined, { sensitivity: 'base' }) * modifier;
          case 'attempts':
            return (a.attempts - b.attempts) * modifier;
          case 'time': {
            const tA = a.submitted_at ? new Date(a.submitted_at).getTime() : 0;
            const tB = b.submitted_at ? new Date(b.submitted_at).getTime() : 0;
            return (tA - tB) * modifier;
          }
          default:
            return 0;
        }
      });
    }

    return list;
  });

  /** El cierre de un examen deja los volúmenes en solo lectura; una clase no. */
  readonly isExamSession = computed(() => {
    const aula = this.session();
    return !!aula && aula.purpose === 'exam';
  });

  readonly lastClosedIsExam = computed(() => this.lastClosedSession()?.purpose === 'exam');

  /** Alumnos del examen cerrado con entorno: los únicos que se pueden reabrir. */
  readonly examStudents = computed<ClassSessionStudent[]>(() =>
    (this.closedBoard()?.students ?? []).filter(student => !!student.environment_id)
  );

  readonly frozenCount = computed(
    () => this.examStudents().filter(student => student.environment_status === 'frozen').length
  );

  readonly subjectOptions = computed<ComboboxOption[]>(() =>
    this.subjects().map(subject => ({
      id: subject.id,
      label: `${subject.code} · ${subject.name}`,
      value: subject.id
    }))
  );

  ngOnInit(): void {
    this.loadCurrentAula();
    this.loadSubjects();
  }

  ngOnDestroy(): void {
    this.stopRefresh();
  }

  /** Abre el aula o, si no hay ninguna, deja la pantalla lista para abrirla. */
  loadCurrentAula(): void {
    this.sessionService.loadCurrent().subscribe({
      next: session => {
        if (session) {
          this.startSession(session);
        } else {
          this.loadLastClosedAula();
        }
      },
      error: () => this.stopRefresh()
    });
  }

  /** Sin aula abierta, recupera la última cerrada para su historial y su ventana. */
  private loadLastClosedAula(): void {
    this.sessionService.loadLastClosed().subscribe({
      next: closed => {
        if (closed) {
          this.historySession.set(closed);
          this.refreshHistory();
          if (closed.purpose === 'exam') {
            this.loadClosedExam(closed.id);
          }
        }
        this.stopRefresh();
      },
      error: () => this.stopRefresh()
    });
  }

  onSubjectChange(subjectId: string): void {
    this.selectedSubjectId.set(subjectId);
    this.selectedExerciseId.set('');
    this.loadPractices(subjectId);
  }

  onExerciseChange(exerciseId: string): void {
    this.selectedExerciseId.set(exerciseId);
  }

  onDurationChange(duration: string): void {
    this.selectedDuration.set(duration || DEFAULT_DURATION);
  }

  openAula(): void {
    const subjectId = this.selectedSubjectId();
    if (!subjectId) {
      return;
    }
    this.sessionService
      .openAula(subjectId, this.selectedExerciseId() || null, Number(this.selectedDuration()))
      .subscribe({
        next: session => this.startSession(session),
        error: () => undefined
      });
  }

  readonly isCloseConfirmOpen = signal(false);
  readonly isClosing = signal(false);
  readonly isPauseConfirmOpen = signal(false);

  togglePauseClick(): void {
    const aula = this.session();
    // La pausa en vivo vale para cualquier aula abierta: deja los entornos de
    // los alumnos en solo lectura hasta que se reanude.
    if (!aula) {
      return;
    }
    if (aula.frozen_at) {
      this.sessionService.setFrozen(aula.id, false).subscribe({
        next: () => this.refreshHistory(),
        error: () => undefined
      });
    } else {
      this.isPauseConfirmOpen.set(true);
    }
  }

  cancelPause(): void {
    this.isPauseConfirmOpen.set(false);
  }

  confirmPause(): void {
    const aula = this.session();
    if (!aula) {
      this.isPauseConfirmOpen.set(false);
      return;
    }
    this.sessionService.setFrozen(aula.id, true).subscribe({
      next: () => {
        this.isPauseConfirmOpen.set(false);
        this.refreshHistory();
      },
      error: () => {
        this.isPauseConfirmOpen.set(false);
      }
    });
  }

  openCloseConfirm(): void {
    this.isCloseConfirmOpen.set(true);
  }

  cancelCloseAula(): void {
    this.isCloseConfirmOpen.set(false);
  }

  confirmCloseAula(): void {
    const aula = this.session();
    if (!aula) {
      this.isCloseConfirmOpen.set(false);
      return;
    }
    this.isClosing.set(true);
    this.sessionService.closeAula(aula.id).subscribe({
      next: closed => {
        this.isClosing.set(false);
        this.isCloseConfirmOpen.set(false);
        this.stopRefresh();
        this.activeTab.set('todos');
        this.resetGraceForm();
        this.historySession.set(closed);
        this.refreshHistory();
        // Un examen cerrado conserva su lista de entornos para el conteo y la
        // excepción individual, aunque el tablero en vivo ya no aplique.
        if (closed.purpose === 'exam') {
          this.loadClosedExam(closed.id);
        }
      },
      error: () => {
        this.isClosing.set(false);
        this.isCloseConfirmOpen.set(false);
      }
    });
  }

  refreshBoard(): void {
    const aula = this.session();
    if (aula) {
      this.sessionService.loadBoard(aula.id).subscribe({ error: () => undefined });
    }
  }

  /** Un solo refresco para la pantalla: tablero, avances e historial. */
  refreshAll(): void {
    this.refreshBoard();
    this.refreshAdvances();
    this.refreshHistory();
  }

  setTab(tabId: string): void {
    this.activeTab.set(tabId);
  }

  onSearch(term: string): void {
    this.searchTerm.set(term);
  }

  stateLabel(state: ClassSessionStudentState): string {
    return STATE_LABELS[state] ?? state;
  }

  /**
   * Emitir exige el aula abierta, sin pausa y no ser examen: en exámenes la emisión está bloqueada.
   */
  readonly canEmit = computed(() => {
    const aula = this.session();
    return !!aula && aula.status === 'open' && !aula.frozen_at && aula.purpose !== 'exam';
  });

  private startSession(session: ClassSession): void {
    this.stopRefresh();
    this.historySession.set(session);
    this.refreshHistory();
    this.sessionService.loadBoard(session.id).subscribe({ error: () => undefined });
    // La emisión elige entre las prácticas publicadas de la materia del aula.
    this.loadPractices(session.subject_id);
    this.advanceService.clearNotice();
    this.refreshAdvances();

    // Una clase se mira cada 5 segundos: es el ritmo de la presencialidad.
    this.refreshTimer = setInterval(() => {
      if (typeof document !== 'undefined' && document.hidden) {
        return;
      }
      this.sessionService.loadBoard(session.id).subscribe({
        // Al vencer la hora de fin, el servidor devuelve la sesión cerrada y la
        // pantalla deja el modo en vivo sin esperar a que el docente cierre.
        next: () => {
          if (!this.sessionService.session()) {
            this.onAulaEnded();
          }
        },
        error: () => undefined
      });
    }, REFRESH_MS);
  }

  /**
   * Abre el entorno de la práctica que la clase está dando. La pestaña se reserva
   * antes de pedir el entorno, para que el navegador no la tome por ventana
   * emergente y para cerrarla si la apertura falla.
   */
  openIde(): void {
    const aula = this.session();
    if (!aula?.exercise_id || this.isOpeningIde()) {
      return;
    }

    const tab = window.open('/entorno.html', '_blank');
    this.ideError.set(null);
    this.isOpeningIde.set(true);

    this.environment.openPracticeIDE(aula.subject_id, aula.exercise_id).subscribe({
      next: session => {
        this.isOpeningIde.set(false);
        if (!session?.access_url) {
          tab?.close();
          this.ideError.set('El entorno arrancó pero todavía no tiene una URL de acceso.');
          return;
        }
        if (tab) {
          tab.opener = null;
          tab.location.href = session.access_url;
          return;
        }
        // Sin pestaña disponible (el navegador bloqueó la ventana emergente), el
        // entorno ya está corriendo: se abre en esta misma pestaña.
        window.location.assign(session.access_url);
      },
      error: err => {
        this.isOpeningIde.set(false);
        tab?.close();
        this.ideError.set(this.environment.describePracticeIDEError(err));
      }
    });
  }

  openAdvanceModal(): void {
    if (!this.canEmit()) {
      return;
    }
    this.advanceService.clearError();
    this.isAdvanceModalOpen.set(true);
  }

  closeAdvanceModal(): void {
    this.isAdvanceModalOpen.set(false);
  }

  /** El aviso ya quedó para los alumnos: se refresca el tablero y se cierra. */
  onAdvanceEmitted(): void {
    this.isAdvanceModalOpen.set(false);
    this.refreshBoard();
    this.refreshAdvances();
    this.refreshHistory();
  }

  /** Historial del aula abierta o de la última cerrada. */
  refreshHistory(): void {
    const session = this.historySession();
    if (!session) {
      return;
    }
    this.sessionService.loadHistory(session.id).subscribe({ error: () => undefined });
  }

  /** Estado de cada destinatario de las emisiones de esta aula. */
  refreshAdvances(): void {
    const aula = this.session();
    if (!aula) {
      return;
    }
    this.advanceService.loadSessionAdvances(aula.id).subscribe({ error: () => undefined });
  }

  openAdvanceDetail(advance: SessionAdvance): void {
    this.advanceDetail.set(advance);
  }

  closeAdvanceDetail(): void {
    this.advanceDetail.set(null);
  }

  dismissEmissionNotice(): void {
    this.advanceService.clearNotice();
  }

  /** Tablero del examen cerrado: estado de cada entorno y hora de la lista. */
  private loadClosedExam(sessionId: string): void {
    this.sessionService.loadClosedBoard(sessionId).subscribe({ error: () => undefined });
  }

  /** Reintenta el congelamiento sin reabrir la clase; el conteo se actualiza. */
  freezeExamAgain(closed: ClassSession): void {
    this.sessionService.freezeExam(closed.id).subscribe({
      next: () => {
        this.loadClosedExam(closed.id);
        this.refreshHistory();
      },
      error: () => undefined
    });
  }

  openUnfreeze(student: ClassSessionStudent): void {
    this.unfreezeTarget.set(student);
  }

  closeUnfreeze(): void {
    this.unfreezeTarget.set(null);
  }

  /** La escritura volvió a un alumno: se relee el estado real de los entornos. */
  onStudentUnfrozen(): void {
    const closed = this.lastClosedSession();
    this.unfreezeTarget.set(null);
    if (closed) {
      this.loadClosedExam(closed.id);
      this.refreshHistory();
    }
  }

  /** Una ventana con hora pasada ya no deja aplicar, aunque conserve el valor. */
  isGraceActive(session: ClassSession): boolean {
    return !!session.grace_until && new Date(session.grace_until).getTime() > Date.now();
  }

  onGraceMinutesChange(minutes: number | null): void {
    this.graceMinutes.set(minutes);
    this.graceError.set('');
  }

  onGraceDeadlineChange(value: string): void {
    this.graceDeadline.set(value);
    this.graceError.set('');
  }

  	/**
	 * Suma minutos al final vigente de la ventana del aula cerrada. Se calcula y
	 * se manda la hora final, no el delta: un clic repetido o un reintento manda
	 * la misma hora y la ventana no se corre dos veces.
	 */
	addGrace(): void {
		const closed = this.lastClosedSession();
		const minutes = this.graceMinutes();
		if (!closed || this.isExtendingGrace()) {
			return;
		}
		if (minutes === null || !Number.isInteger(minutes) || minutes < MIN_GRACE_MINUTES || minutes > MAX_GRACE_MINUTES) {
			this.graceError.set(`Escribí un número entero entre ${MIN_GRACE_MINUTES} y ${MAX_GRACE_MINUTES} minutos.`);
			return;
		}
		const now = Date.now();
		const current = closed.grace_until ? new Date(closed.grace_until).getTime() : Number.NaN;
		// Una ventana vencida se reabre desde ahora, igual que hacía el servidor.
		const base = Number.isNaN(current) || current < now ? now : current;
		const until = new Date(base + minutes * 60_000);
		this.graceError.set('');
		this.sessionService.setGraceUntil(closed.id, until.toISOString()).subscribe({
			next: () => this.refreshHistory(),
			error: () => this.graceFailed()
		});
	}

  /** Fija una fecha límite concreta para una tarea. */
  applyGraceDeadline(): void {
    const closed = this.lastClosedSession();
    const value = this.graceDeadline();
    if (!closed || !value) {
      return;
    }
    const until = new Date(value);
    if (Number.isNaN(until.getTime())) {
      this.graceError.set('Elegí una fecha y una hora válidas.');
      return;
    }
    this.graceError.set('');
    		this.sessionService.setGraceUntil(closed.id, until.toISOString()).subscribe({
      next: () => this.refreshHistory(),
      error: () => this.graceFailed()
    });
  }

  /** Cierra la ventana en el momento, sin esperar a que venza. */
  closeGraceNow(): void {
    const closed = this.lastClosedSession();
    if (!closed) {
      return;
    }
    this.graceError.set('');
    this.sessionService.closeGrace(closed.id).subscribe({
      next: () => this.refreshHistory(),
      error: () => this.graceFailed()
    });
  }

  /** El motivo del servidor se muestra junto a la ventana, no al pie del aula. */
  private graceFailed(): void {
    this.graceError.set(this.sessionService.error() ?? 'No se pudo ajustar la ventana.');
    this.sessionService.clearError();
  }

  /** Cada aula cerrada vuelve a arrancar con la ventana automática de 30 minutos. */
  private resetGraceForm(): void {
    this.graceMinutes.set(DEFAULT_GRACE_MINUTES);
    this.graceDeadline.set('');
    this.graceError.set('');
  }

  /** El aula dejó el modo en vivo: se vuelve al estado sin sesión. */
  private onAulaEnded(): void {
    this.stopRefresh();
    this.activeTab.set('todos');
    this.resetGraceForm();
    const closed = this.lastClosedSession();
    if (closed) {
      this.historySession.set(closed);
      this.refreshHistory();
    }
  }

  private stopRefresh(): void {
    if (this.refreshTimer) {
      clearInterval(this.refreshTimer);
      this.refreshTimer = undefined;
    }
  }	/**
	 * Materias del docente y, si la URL trae una, la preselección del formulario.
	 * Un identificador que no es suyo no se convierte en error de permiso: cae
	 * al comportamiento sin preselección. Con aula abierta la URL no manda.
	 */
	private loadSubjects(): void {
		this.dashboardService.getCourses().subscribe({
			next: courses => {
				const preselected = this.route.snapshot.queryParamMap.get('subject_id') ?? '';
				if (preselected && !this.session() && courses.some(course => course.id === preselected)) {
					this.pendingExerciseId = this.route.snapshot.queryParamMap.get('exercise_id') ?? '';
					this.onSubjectChange(preselected);
				}
			},
			error: () => undefined
		});
	}

  /** Prácticas publicadas de la materia elegida, desde el banco del docente. */
  private loadPractices(subjectId: string): void {
    this.courseService.getExerciseBank().subscribe({
      next: bank => {
        const options = bank
          .filter(item => item.subject_id === subjectId && item.status === 'published')
          .map(item => ({
            id: item.id,						label: item.title,
						value: item.id
					}));
				this.practiceOptions.set(options);
				this.applyPendingExercise(options);
			},
			error: () => this.practiceOptions.set([])
		});
	}

	/**
	 * La práctica de la URL se elige solo si pertenece a la materia y está
	 * publicada: el enlace viejo o de otra materia no deja un formulario cojo.
	 */
	private applyPendingExercise(options: ComboboxOption[]): void {
		const pending = this.pendingExerciseId;
		this.pendingExerciseId = '';
		if (pending && options.some(option => option.value === pending)) {
			this.selectedExerciseId.set(pending);
		}
	}
}
