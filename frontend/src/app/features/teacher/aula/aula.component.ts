import { Component, OnDestroy, OnInit, computed, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import {
  LucideAlertTriangle,
  LucideCheckCircle2,
  LucideRotateCcw,
  LucideTerminal
} from '@lucide/angular';

import { DateTextPipe } from '@shared/pipes/date-text.pipe';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { ComboboxComponent, ComboboxOption } from '@shared/components/combobox/combobox.component';
import { SearchBarComponent } from '@shared/components/search-bar/search-bar.component';
import { StatusTabItem, StatusTabsComponent } from '@shared/components/status-tabs/status-tabs.component';
import { VerdictBadgeComponent } from '@shared/components/verdict-badge/verdict-badge.component';

import { ClassSessionService } from '../services/class-session.service';
import { TeacherDashboardService } from '../services/teacher-dashboard.service';
import { TeacherCourseService } from '../services/teacher-course.service';
import { TeacherLiveService } from '../services/teacher-live.service';
import {
  ClassSession,
  ClassSessionStudent,
  ClassSessionStudentState,
  LiveWorkspaceSession
} from '../models/teacher.models';
import { AulaTelemetryWidgetComponent } from './aula-telemetry-widget/aula-telemetry-widget.component';
import { ShadowTerminalModalComponent } from './shadow-terminal-modal/shadow-terminal-modal.component';

const REFRESH_MS = 5000;

/** Duraciones de clase que el docente elige al abrir. Al vencer, el aula se cierra sola. */
const DURATION_OPTIONS: ComboboxOption[] = [
  { id: '60', label: '1 hora', value: '60' },
  { id: '90', label: '1 hora 30 min', value: '90' },
  { id: '120', label: '2 horas', value: '120' },
  { id: '180', label: '3 horas', value: '180' }
];

const DEFAULT_DURATION = '90';

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
    LucideRotateCcw,
    LucideTerminal,
    DateTextPipe,
    MachineDataDirective,
    ComboboxComponent,
    SearchBarComponent,
    StatusTabsComponent,
    VerdictBadgeComponent,
    AulaTelemetryWidgetComponent,
    ShadowTerminalModalComponent
  ],
  templateUrl: './aula.component.html',
  styleUrl: './aula.component.scss'
})
export class AulaComponent implements OnInit, OnDestroy {
  private readonly sessionService = inject(ClassSessionService);
  private readonly dashboardService = inject(TeacherDashboardService);
  private readonly courseService = inject(TeacherCourseService);
  private readonly liveService = inject(TeacherLiveService);

  readonly session = this.sessionService.session;
  readonly board = this.sessionService.board;
  readonly isLoading = this.sessionService.isLoading;
  readonly isOpening = this.sessionService.isOpening;
  readonly error = this.sessionService.error;
  readonly lastClosedAt = this.sessionService.lastClosedAt;

  readonly liveSessions = this.liveService.liveSessions;
  readonly isPollingActive = this.liveService.isPollingActive;
  readonly pollingRateMs = this.liveService.pollingIntervalMs;

  readonly subjects = this.dashboardService.courses;

  readonly activeTab = signal<string>('todos');
  readonly searchTerm = signal<string>('');
  readonly selectedSubjectId = signal<string>('');
  readonly selectedExerciseId = signal<string>('');
  readonly selectedDuration = signal<string>(DEFAULT_DURATION);
  readonly durationOptions = DURATION_OPTIONS;
  readonly practiceOptions = signal<ComboboxOption[]>([]);
  readonly selectedWorkspace = signal<LiveWorkspaceSession | null>(null);

  private refreshTimer?: ReturnType<typeof setInterval>;

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
    return s.stuck > 0 || s.not_started > 0 || s.without_submission > 0 || s.absent > 0;
  });

  readonly tabs = computed<StatusTabItem[]>(() => {
    const s = this.summary();
    const students = this.board()?.students ?? [];
    return [
      { id: 'todos', label: 'Todos', count: students.length },
      { id: 'trabado', label: 'Trabados', count: s.stuck, badgeVariant: s.stuck > 0 ? 'error' : 'default' },
      { id: 'sin_empezar', label: 'Sin empezar', count: s.not_started },
      { id: 'ausente', label: 'Salieron', count: s.absent },
      { id: 'entregado', label: 'Entregaron', count: s.submitted },
      { id: 'trabajando', label: 'Trabajando', count: s.working }
    ];
  });

  readonly visibleStudents = computed<ClassSessionStudent[]>(() => {
    const students = this.board()?.students ?? [];
    const tab = this.activeTab();
    const term = this.searchTerm().toLowerCase().trim();

    return students.filter(student => {
      if (tab !== 'todos' && student.state !== tab) {
        return false;
      }
      if (!term) {
        return true;
      }
      return student.name.toLowerCase().includes(term) || student.email.toLowerCase().includes(term);
    });
  });

  /** Entornos de la clase abierta: la telemetría se mira una materia por vez. */
  readonly sessionEnvironments = computed<LiveWorkspaceSession[]>(() => {
    const aula = this.session();
    const sessions = this.liveSessions();
    if (!aula) {
      return sessions;
    }
    return sessions.filter(item => item.subject_id === aula.subject_id);
  });

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
    this.liveService.pausePolling();
  }

  /** Abre el aula o, si no hay ninguna, deja la pantalla lista para abrirla. */
  loadCurrentAula(): void {
    this.sessionService.loadCurrent().subscribe({
      next: session => {
        if (session) {
          this.startSession(session);
        } else {
          this.watchEnvironments();
        }
      },
      error: () => this.watchEnvironments()
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

  closeAula(): void {
    const aula = this.session();
    if (!aula) {
      return;
    }
    this.sessionService.closeAula(aula.id).subscribe({
      next: () => {
        this.stopRefresh();
        this.activeTab.set('todos');
        this.watchEnvironments();
      },
      error: () => undefined
    });
  }

  refreshBoard(): void {
    const aula = this.session();
    if (aula) {
      this.sessionService.loadBoard(aula.id).subscribe({ error: () => undefined });
    }
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

  openConsole(student: ClassSessionStudent): void {
    const workspace = this.liveSessions().find(item => item.workspace_id === student.workspace_id);
    if (workspace) {
      this.selectedWorkspace.set(workspace);
    }
  }

  closeConsole(): void {
    this.selectedWorkspace.set(null);
  }

  setPollingRate(rateMs: number): void {
    this.liveService.setPollingInterval(rateMs);
  }

  refreshEnvironments(): void {
    this.liveService.loadLiveSessions().subscribe();
  }

  private startSession(session: ClassSession): void {
    this.stopRefresh();
    this.sessionService.loadBoard(session.id).subscribe({ error: () => undefined });

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

    this.liveService.setPollingInterval(REFRESH_MS);
  }

  /** El aula dejó el modo en vivo: se vuelve al estado sin sesión. */
  private onAulaEnded(): void {
    this.stopRefresh();
    this.activeTab.set('todos');
    this.watchEnvironments();
  }

  /** Sin aula abierta solo se miran contenedores: no hay faltantes que calcular. */
  private watchEnvironments(): void {
    this.stopRefresh();
    this.liveService.setPollingInterval(REFRESH_MS);
  }

  private stopRefresh(): void {
    if (this.refreshTimer) {
      clearInterval(this.refreshTimer);
      this.refreshTimer = undefined;
    }
  }

  private loadSubjects(): void {
    this.dashboardService.getCourses().subscribe({ error: () => undefined });
  }

  /** Prácticas publicadas de la materia elegida, desde el banco del docente. */
  private loadPractices(subjectId: string): void {
    this.courseService.getExerciseBank().subscribe({
      next: bank => {
        const options = bank
          .filter(item => item.subject_id === subjectId && item.status === 'published')
          .map(item => ({
            id: item.id,
            label: item.title,
            value: item.id
          }));
        this.practiceOptions.set(options);
      },
      error: () => this.practiceOptions.set([])
    });
  }
}
