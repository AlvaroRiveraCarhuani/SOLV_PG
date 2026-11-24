import { Component, DestroyRef, HostListener, OnInit, computed, inject, signal } from '@angular/core';
import { CommonModule, Location } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import {
  LucideAlertTriangle,
  LucideArrowLeft,
  LucideCheck,
  LucideCheckCircle2,
  LucideChevronDown,
  LucideChevronUp,
  LucideColumns2,
  LucideCopy,
  LucideExternalLink,
  LucideFileCode,
  LucideFileText,
  LucideLayers,
  LucideLoader2,
  LucideMaximize2,
  LucidePlay,
  LucideRefreshCw,
  LucideRotateCcw,
  LucideShield,
  LucideTerminal,
  LucideX
} from '@lucide/angular';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { CodeEditorComponent } from '@shared/components/code-editor/code-editor.component';
import { ComboboxComponent, ComboboxOption } from '@shared/components/combobox/combobox.component';
import { VerdictBadgeComponent } from '@shared/components/verdict-badge/verdict-badge.component';
import { ConfirmModalComponent } from '@shared/components/confirm-modal/confirm-modal.component';
import { DateTextPipe } from '@shared/pipes/date-text.pipe';
import { EnvironmentLabelPipe } from '@shared/pipes/environment-label.pipe';
import { EnvironmentState, PracticeEnvironment } from '@core/models/environment.model';
import { RubricDTO } from '@core/models/rubric.model';
import {
  StudentEvaluationResultDTO,
  StudentPublicExerciseDTO,
  StudentPublicTestCaseDTO,
  StudentService,
  PracticeEnvironmentProjectSubmissionDTO
} from '@core/services/student.service';
import { AuthService } from '@core/services/auth.service';
import {
  ClassAdvanceNoticeService,
  PendingClassAdvance
} from '../../services/class-advance-notice.service';
import { AdvanceConfirmModalComponent } from './advance-confirm-modal/advance-confirm-modal.component';
import { renderStatementMarkdown } from './statement-markdown.utils';

interface KeystrokeEventPayload {
  event_type: 'insert' | 'delete' | 'paste';
  timestamp_ms: number;
  position: number;
  content: string;
}

function encodeUtf8Base64(text: string): string {
  const bytes = new TextEncoder().encode(text);
  let binary = '';
  for (const byte of bytes) {
    binary += String.fromCharCode(byte);
  }
  return btoa(binary);
}

function normalizeNewlines(text?: string | null): string {
  if (!text) return '';
  return text.includes('\\n') ? text.replace(/\\n/g, '\n') : text;
}

export function getFileExtension(lang: string): string {
  const l = (lang || '').toLowerCase().trim();
  const map: Record<string, string> = {
    python: 'py',
    py: 'py',
    javascript: 'js',
    js: 'js',
    typescript: 'ts',
    ts: 'ts',
    golang: 'go',
    go: 'go',
    cpp: 'cpp',
    'c++': 'cpp',
    c: 'c',
    java: 'java',
    rust: 'rs',
    rs: 'rs',
    csharp: 'cs',
    'c#': 'cs',
    php: 'php',
    ruby: 'rb',
    sql: 'sql'
  };
  return map[l] || l || 'txt';
}

export function formatCPUQuota(cpuMillis: number): string {
  if (!Number.isFinite(cpuMillis) || cpuMillis <= 0) return '';
  const cores = cpuMillis / 1000;
  if (Number.isInteger(cores)) return cores === 1 ? '1 núcleo' : `${cores} núcleos`;
  return `${cores.toFixed(1)} núcleos`;
}

@Component({
  selector: 'student-exercise-solver',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    RouterLink,
    DateTextPipe,
    MachineDataDirective,
    CodeEditorComponent,
    ComboboxComponent,
    VerdictBadgeComponent,
    ConfirmModalComponent,
    AdvanceConfirmModalComponent,
    LucideAlertTriangle,
    LucideArrowLeft,
    LucideCheck,
    LucideCheckCircle2,
    LucideChevronDown,
    LucideChevronUp,
    LucideColumns2,
    LucideCopy,
    LucideExternalLink,
    LucideFileCode,
    LucideFileText,
    LucideLayers,
    LucideLoader2,
    LucideMaximize2,
    LucidePlay,
    LucideRefreshCw,
    LucideRotateCcw,
    LucideShield,
    LucideTerminal,
    LucideX
  ],
  templateUrl: './student-exercise-solver.component.html',
  styleUrl: './student-exercise-solver.component.scss'
})
export class StudentExerciseSolverComponent implements OnInit {
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly location = inject(Location);
  private readonly studentService = inject(StudentService);
  private readonly authService = inject(AuthService);
  private readonly destroyRef = inject(DestroyRef);
  private readonly advances = inject(ClassAdvanceNoticeService);

  readonly exerciseId = signal<string>('');
  readonly exercise = signal<StudentPublicExerciseDTO | null>(null);
  readonly isLoading = signal<boolean>(false);
  readonly loadError = signal<string | null>(null);

  // Rúbrica para Entorno de Desarrollo
  readonly rubric = signal<RubricDTO | null>(null);
  readonly activeCriterionIndex = signal<number | null>(null);

  readonly selectedLanguage = signal<string>('python');
  readonly sourceCode = signal<string>('');
  readonly isTesting = signal<boolean>(false);
  readonly isSubmitting = signal<boolean>(false);
  readonly isTestRun = signal<boolean>(false);
  readonly submitError = signal<string | null>(null);
  readonly lastResult = signal<StudentEvaluationResultDTO | null>(null);
  readonly lastSubmissionId = signal<string>('');

  // Consola inferior plegable y pestañas
  readonly consoleOpen = signal<boolean>(true);
  readonly activeConsoleTab = signal<'test' | 'submit'>('test');
  readonly copiedCaseIndex = signal<number | null>(null);

  // Layout interactivo: modo de visualización y splitters redimensionables
  readonly viewMode = signal<'both' | 'statement' | 'editor'>('both');
  readonly splitRatio = signal<number>(42);
  readonly isDraggingSplitter = signal<boolean>(false);
  readonly consoleHeight = signal<number>(248);
  readonly isDraggingConsoleSplitter = signal<boolean>(false);

  // Modal de confirmación para examen
  readonly examConfirmModalOpen = signal<boolean>(false);
  readonly isConfirmingProject = signal<boolean>(false);

  private readonly keystrokeBuffer: KeystrokeEventPayload[] = [];

  // IDE Persistente
  readonly environment = signal<PracticeEnvironment | null>(null);
  readonly isStartingEnvironment = signal<boolean>(false);
  readonly environmentError = signal<string | null>(null);
  readonly isSubmittingProject = signal<boolean>(false);
  readonly projectError = signal<string | null>(null);
  readonly projectSubmission = signal<PracticeEnvironmentProjectSubmissionDTO | null>(null);
  readonly projectSubmittedAt = signal<Date | null>(null);
  readonly examSubmitPending = signal<boolean>(false);

  readonly practiceAdvances = computed<PendingClassAdvance[]>(() =>
    this.advances.pending().filter(advance => advance.exercise_id === this.exerciseId())
  );
  readonly selectedAdvance = signal<PendingClassAdvance | null>(null);
  readonly isDismissingAdvance = signal<boolean>(false);

  readonly allowedLanguages = computed<string[]>(() => {
    const ex = this.exercise();
    const base = (ex?.language || 'python').toLowerCase();
    const declared = (ex?.allowed_languages ?? [])
      .map(lang => (lang || '').toLowerCase())
      .filter(Boolean);
    if (declared.length === 0) return [base];
    return Array.from(new Set([base, ...declared]));
  });

  readonly languageOptions = computed<ComboboxOption[]>(() =>
    this.allowedLanguages().map((lang) => ({
      id: lang,
      label: lang.toUpperCase(),
      value: lang
    }))
  );

  readonly fileExtension = computed<string>(() => {
    return getFileExtension(this.selectedLanguage());
  });

  readonly fileName = computed<string>(() => {
    return `solucion.${this.fileExtension()}`;
  });

  readonly renderedDescription = computed<string>(() => {
    return renderStatementMarkdown(this.exercise()?.description || '');
  });

  readonly exampleCases = computed<StudentPublicTestCaseDTO[]>(() => {
    return this.exercise()?.public_test_cases ?? [];
  });

  readonly constraintsList = computed<string[]>(() => {
    return (this.exercise()?.constraints ?? []).map((constraint) =>
      constraint
        .replace(/^Forbidden import: /, 'Importación prohibida: ')
        .replace(/^Forbidden function: /, 'Función prohibida: ')
    );
  });

  readonly isIdeExercise = computed<boolean>(() => {
    const ex = this.exercise();
    return (
      ex?.type === 'project' ||
      ex?.type === 'environment' ||
      ex?.environment_type === 'IDE_PERSISTENTE' ||
      ex?.environment_type === 'environment'
    );
  });

  readonly environmentIsRunning = computed<boolean>(() => this.environment()?.status === 'running');
  readonly appUrl = computed<string | null>(() => this.environment()?.app_url ?? null);
  readonly ideUrl = computed<string | null>(() => this.environment()?.access_url ?? null);
  readonly popupBlocked = signal<boolean>(false);

  readonly examClosed = signal<boolean>(false);
  readonly classPaused = signal<boolean>(false);

  readonly diskNotice = computed<{ tone: 'warning' | 'full'; text: string } | null>(() => {
    const environment = this.environment();
    if (!environment || (environment.state !== 'disk_warning' && environment.state !== 'disk_full')) {
      return null;
    }
    const used = environment.disk_used_mb ?? 0;
    const limit = environment.disk_limit_mb ?? 0;
    if (environment.state === 'disk_full') {
      return {
        tone: 'full',
        text: `Su espacio de trabajo está lleno (${used} MB de ${limit} MB). Elimine archivos generados para liberar espacio y vuelva a entregar; su entorno continúa disponible.`
      };
    }
    const percent = limit > 0 ? Math.round((used * 100) / limit) : 0;
    return {
      tone: 'warning',
      text: `Su espacio de trabajo está al ${percent}% (${used} MB de ${limit} MB). El límite es orientativo: puede continuar trabajando y eliminar archivos generados para liberar espacio.`
    };
  });

  readonly resourceQuota = computed<{
    declaredMB: number;
    grantedMB: number;
    cpuText: string;
    reduced: boolean;
  } | null>(() => {
    const environment = this.environment();
    if (!environment) return null;
    const declaredMB = environment.memory_profile_mb ?? 0;
    const grantedMB = environment.memory_limit_mb ?? 0;
    const cpuText = formatCPUQuota(environment.cpu_millis ?? 0);
    if (grantedMB <= 0 && cpuText === '') return null;
    return {
      declaredMB,
      grantedMB,
      cpuText,
      reduced: declaredMB > 0 && grantedMB > 0 && grantedMB < declaredMB
    };
  });

  readonly purgingCaches = signal<boolean>(false);

  readonly environmentState = computed<EnvironmentState>(() => {
    if (this.examClosed()) {
      return 'closed';
    }
    const env = this.environment();
    if (env?.state) {
      return env.state;
    }
    if (!env) {
      return 'preparing';
    }
    if (env.status === 'frozen') {
      return 'closed';
    }
    if (env.status === 'failed' || env.status === 'oom_killed') {
      return 'failed';
    }
    return env.status === 'running' ? 'ready' : 'preparing';
  });

  readonly showsClassPausedNotice = computed<boolean>(() => {
    if (this.isExamProject()) {
      return false;
    }
    return this.classPaused() || this.environment()?.status === 'frozen';
  });

  readonly canResubmitProject = computed<boolean>(() => this.exercise()?.allow_resubmit ?? true);
  readonly isExamProject = computed<boolean>(() => (this.exercise()?.purpose ?? 'class') === 'exam');
  readonly examBannerDismissed = signal<boolean>(false);

  readonly isExamExpired = computed<boolean>(() => {
    const due = this.exercise()?.due_date;
    if (!due) return false;
    return new Date(due).getTime() < Date.now();
  });

  readonly isExamSubmitted = computed<boolean>(() => {
    if (!this.isExamProject() || this.isIdeExercise()) {
      return false;
    }
    if (this.isExamExpired()) {
      return true;
    }
    const hasSubmitted = !!this.lastSubmissionId();
    if (!hasSubmitted) {
      return false;
    }
    return !this.canResubmitProject();
  });

  readonly projectSubmissionId = computed<string>(() => {
    return this.projectSubmission()?.submission_id ?? this.lastSubmissionId();
  });

  readonly hasProjectSubmission = computed<boolean>(() => this.projectSubmissionId() !== '');

  readonly canSubmitProject = computed<boolean>(() => {
    if (!this.environmentIsRunning() || this.isSubmittingProject()) {
      return false;
    }
    if (this.showsClassPausedNotice()) {
      return false;
    }
    if (this.isExamProject() && this.isExamExpired()) {
      return false;
    }
    if (this.projectSubmission() && !this.canResubmitProject()) {
      return false;
    }
    return true;
  });

  readonly timeLimitLabel = computed<string>(() => {
    const ms = this.exercise()?.time_limit_ms || 1000;
    if (ms >= 1000) {
      const seconds = ms / 1000;
      return `${Number.isInteger(seconds) ? seconds : seconds.toFixed(1)} s`;
    }
    return `${ms} ms`;
  });

  readonly difficultyLabel = computed<string>(() => {
    switch ((this.exercise()?.difficulty || '').toLowerCase()) {
      case 'easy':
      case 'facil':
      case 'fácil':
        return 'Fácil';
      case 'medium':
      case 'medio':
        return 'Medio';
      case 'hard':
      case 'dificil':
      case 'difícil':
        return 'Difícil';
      default:
        return this.exercise()?.difficulty || '';
    }
  });

  readonly rubricTotalWeight = computed<number>(() => {
    const list = this.rubric()?.criteria || [];
    return list.reduce((sum, c) => sum + (c.weight || 0), 0) || 100;
  });

  // Salvaguarda anti-olvido
  readonly allExampleCasesPassed = computed<boolean>(() => {
    const res = this.lastResult();
    if (!res || !this.isTestRun()) return false;
    if (res.verdict !== 'AC') return false;
    if (!res.case_results || res.case_results.length === 0) return true;
    return res.case_results.every(c => c.verdict === 'AC');
  });

  readonly showsAntiForgetNotice = computed<boolean>(() => {
    return this.isTestRun() && this.allExampleCasesPassed() && !this.lastSubmissionId();
  });

  readonly examModalTitle = computed<string>(() => {
    return this.isIdeExercise() ? 'Confirmar entrega del proyecto' : 'Confirmar entrega del examen';
  });

  readonly examModalMessage = computed<string>(() => {
    if (this.canResubmitProject()) {
      return 'Este examen permite reenvíos hasta la fecha límite. Al confirmar, se registrará una versión oficial de su solución y podrá continuar editando mientras el examen permanezca abierto.';
    }
    return 'Atención: este examen es de intento único. Al confirmar la entrega, su solución quedará registrada de forma definitiva y no podrá realizar modificaciones posteriores.';
  });

  @HostListener('window:keydown', ['$event'])
  handleGlobalShortcuts(event: KeyboardEvent): void {
    if ((event.ctrlKey || event.metaKey) && event.key === 'Enter') {
      if (!this.isIdeExercise() && !this.isTesting() && !this.isSubmitting() && this.sourceCode().trim()) {
        event.preventDefault();
        void this.testCode();
      }
    }
  }

  toggleConsole(): void {
    this.consoleOpen.update(v => !v);
  }

  setConsoleTab(tab: 'test' | 'submit'): void {
    this.activeConsoleTab.set(tab);
    if (!this.consoleOpen()) {
      this.consoleOpen.set(true);
    }
  }

  copyInput(inputStr: string, index: number): void {
    if (typeof navigator !== 'undefined' && navigator.clipboard) {
      void navigator.clipboard.writeText(inputStr);
      this.copiedCaseIndex.set(index);
      setTimeout(() => this.copiedCaseIndex.set(null), 2000);
    }
  }

  setViewMode(mode: 'both' | 'statement' | 'editor'): void {
    this.viewMode.set(mode);
  }

  onSplitterMouseDown(event: MouseEvent): void {
    event.preventDefault();
    this.isDraggingSplitter.set(true);

    const onMouseMove = (moveEvent: MouseEvent) => {
      const container = document.querySelector('.workbench-split-layout') as HTMLElement;
      if (!container) return;
      const rect = container.getBoundingClientRect();
      const currentX = moveEvent.clientX - rect.left;
      const percent = (currentX / rect.width) * 100;
      this.splitRatio.set(Math.max(20, Math.min(80, Math.round(percent))));
    };

    const onMouseUp = () => {
      this.isDraggingSplitter.set(false);
      window.removeEventListener('mousemove', onMouseMove);
      window.removeEventListener('mouseup', onMouseUp);
    };

    window.addEventListener('mousemove', onMouseMove);
    window.addEventListener('mouseup', onMouseUp);
  }

  onConsoleSplitterMouseDown(event: MouseEvent): void {
    event.preventDefault();
    if (!this.consoleOpen()) {
      this.consoleOpen.set(true);
    }
    this.isDraggingConsoleSplitter.set(true);

    const onMouseMove = (moveEvent: MouseEvent) => {
      const container = document.querySelector('.workbench-right-pane') as HTMLElement;
      if (!container) return;
      const rect = container.getBoundingClientRect();
      const newHeight = rect.bottom - moveEvent.clientY;
      this.consoleHeight.set(Math.max(120, Math.min(Math.round(rect.height * 0.75), Math.round(newHeight))));
    };

    const onMouseUp = () => {
      this.isDraggingConsoleSplitter.set(false);
      window.removeEventListener('mousemove', onMouseMove);
      window.removeEventListener('mouseup', onMouseUp);
    };

    window.addEventListener('mousemove', onMouseMove);
    window.addEventListener('mouseup', onMouseUp);
  }

  goBack(): void {
    if (typeof window !== 'undefined' && (window.history.state?.navigationId ?? 0) > 1) {
      this.location.back();
    } else {
      const subjectId = this.exercise()?.subject_id;
      void this.router.navigate(subjectId ? ['/student/courses', subjectId] : ['/student']);
    }
  }

  ngOnInit(): void {
    this.advances.start();
    this.destroyRef.onDestroy(() => this.advances.stop());

    this.route.paramMap.pipe(takeUntilDestroyed(this.destroyRef)).subscribe((params) => {
      const id = params.get('exerciseId') || params.get('id') || '';
      this.exerciseId.set(id);
      if (id) {
        void this.loadPublicExercise(id);
      }
    });
  }

  async loadPublicExercise(id: string): Promise<void> {
    this.isLoading.set(true);
    this.loadError.set(null);
    const data = await this.studentService.getPublicExercise(id);
    this.isLoading.set(false);

    if (!data) {
      this.loadError.set('No se pudo cargar el ejercicio solicitado.');
      return;
    }

    // Normalizar saltos de línea escapados en descripción
    data.description = normalizeNewlines(data.description);
    if (data.boilerplate) {
      data.boilerplate = normalizeNewlines(data.boilerplate);
    }
    if (data.boilerplates) {
      for (const lang of Object.keys(data.boilerplates)) {
        data.boilerplates[lang] = normalizeNewlines(data.boilerplates[lang]);
      }
    }

    this.exercise.set(data);
    try {
      this.examBannerDismissed.set(sessionStorage.getItem(`solv_exam_banner_dismissed_${id}`) === 'true');
    } catch {
      this.examBannerDismissed.set(false);
    }

    // Cargar historial previo de este ejercicio si existe
    const history = this.studentService.submissionsHistory ? this.studentService.submissionsHistory() : [];
    const prevSub = history.find((s) => s.exercise_id === id);
    if (prevSub) {
      this.lastSubmissionId.set(prevSub.id);
      if (prevSub.code && !this.isIdeExercise()) {
        this.sourceCode.set(normalizeNewlines(prevSub.code));
      }
    } else if (this.studentService.loadSubmissionsHistory) {
      void this.studentService.loadSubmissionsHistory().then((h) => {
        const found = h?.find((s) => s.exercise_id === id);
        if (found) {
          this.lastSubmissionId.set(found.id);
          if (found.code && !this.isIdeExercise()) {
            this.sourceCode.set(normalizeNewlines(found.code));
          }
        }
      });
    }

    // Si es IDE_PERSISTENTE, cargar rúbrica oficial
    if (this.isIdeExercise()) {
      void this.studentService.getExerciseRubric(id).then(r => this.rubric.set(r));
      return;
    }

    const lang = (data.language || 'python').toLowerCase();
    const allowed = (data.allowed_languages ?? [])
      .map(candidate => (candidate || '').toLowerCase())
      .filter(Boolean);
    const preferred = allowed.includes(lang) ? lang : (allowed[0] || lang);
    this.selectedLanguage.set(preferred);
    if (!prevSub?.code) {
      const code = data.boilerplates?.[preferred] ?? (preferred === lang ? data.boilerplate || '' : '');
      this.sourceCode.set(normalizeNewlines(code));
    }
  }

  onLanguageChange(lang: string): void {
    if (!lang) return;
    const next = lang.toLowerCase();
    const previous = this.selectedLanguage();
    if (previous === next) return;

    const base = (this.exercise()?.language || '').toLowerCase();
    const starters = this.exercise()?.boilerplates ?? {};
    const starterFor = (language: string): string =>
      normalizeNewlines(starters[language] ?? (language === base ? this.exercise()?.boilerplate ?? '' : ''));
    const current = this.sourceCode();
    if (!current.trim() || current === starterFor(previous)) {
      this.sourceCode.set(starterFor(next));
    }
    this.selectedLanguage.set(next);
  }

  onCodeChange(nextCode: string): void {
    const prevLen = this.sourceCode().length;
    const delta = nextCode.length - prevLen;
    this.sourceCode.set(nextCode);

    if (delta !== 0 && this.exercise()?.purpose === 'exam') {
      const lines = nextCode.split('\n');
      const isPaste = Math.abs(delta) > 25;
      this.keystrokeBuffer.push({
        event_type: isPaste ? 'paste' : (delta > 0 ? 'insert' : 'delete'),
        timestamp_ms: Date.now(),
        position: lines.length - 1,
        content: isPaste ? nextCode : ''
      });
    }
  }

  resetBoilerplate(): void {
    this.sourceCode.set(normalizeNewlines(this.exercise()?.boilerplate || ''));
  }

  openAdvance(advance: PendingClassAdvance): void {
    this.selectedAdvance.set(advance);
  }

  closeAdvanceModal(): void {
    this.selectedAdvance.set(null);
  }

  async dismissAdvance(advance: PendingClassAdvance): Promise<void> {
    if (this.isDismissingAdvance()) return;
    this.isDismissingAdvance.set(true);
    try {
      await this.advances.dismiss(advance.id);
    } finally {
      this.isDismissingAdvance.set(false);
    }
  }

  async openEnvironment(): Promise<void> {
    const ex = this.exercise();
    if (!ex || this.isStartingEnvironment()) return;
    const subjectId = ex.subject_id;
    if (!subjectId) {
      this.environmentError.set('El ejercicio no tiene una materia asociada para abrir el entorno.');
      return;
    }

    const preparacion = typeof window !== 'undefined'
      ? `/entorno.html?volver=${encodeURIComponent(window.location.pathname + window.location.search)}`
      : '/entorno.html';
    const tab = typeof window !== 'undefined' ? window.open(preparacion, '_blank') : null;
    if (tab) {
      tab.opener = null;
    }

    this.popupBlocked.set(!tab);
    this.isStartingEnvironment.set(true);
    this.environmentError.set(null);
    this.examClosed.set(false);
    this.classPaused.set(false);

    try {
      const environment = await this.studentService.startEnvironment(subjectId, ex.id);
      this.environment.set(environment);

      const url = environment.access_url;
      if (!url) {
        tab?.close();
        this.environmentError.set('Su entorno se está preparando. Intente de nuevo en unos segundos.');
        return;
      }

      if (!tab) return;
      tab.location.href = url;
    } catch (error) {
      tab?.close();
      const body = (error as { error?: { message?: unknown; error?: unknown } } | null)?.error;
      const rawCode = body?.error;
      const code = typeof rawCode === 'string'
        ? rawCode
        : (rawCode as { code?: unknown } | null | undefined)?.code;
      if (
        code === 'environment_frozen' ||
        code === 'environment_frozen_exam_paused' ||
        code === 'environment_frozen_class'
      ) {
        if (this.isExamProject()) {
          this.examClosed.set(true);
        } else {
          this.classPaused.set(true);
        }
        return;
      }
      const backendMessage = body?.message;
      this.environmentError.set(
        typeof backendMessage === 'string' && backendMessage.trim()
          ? backendMessage
          : 'No se pudo abrir su entorno. Intente de nuevo en unos segundos.'
      );
    } finally {
      this.isStartingEnvironment.set(false);
    }
  }

  dismissExamBanner(): void {
    this.examBannerDismissed.set(true);
    const id = this.exercise()?.id;
    if (id) {
      try {
        sessionStorage.setItem(`solv_exam_banner_dismissed_${id}`, 'true');
      } catch {
        // storage unavailable
      }
    }
  }

  requestSubmitProject(): void {
    if (!this.canSubmitProject()) return;
    if (this.isExamProject()) {
      this.isConfirmingProject.set(true);
      this.examConfirmModalOpen.set(true);
      return;
    }
    void this.submitProject();
  }

  requestSubmitEvaluation(): void {
    if (!this.sourceCode().trim() || this.isSubmitting() || this.isTesting()) return;
    if (this.isExamProject()) {
      this.isConfirmingProject.set(false);
      this.examConfirmModalOpen.set(true);
      return;
    }
    void this.submitSolution();
  }

  onConfirmExamSubmit(): void {
    this.examConfirmModalOpen.set(false);
    if (this.isConfirmingProject()) {
      void this.submitProject();
    } else {
      void this.submitSolution();
    }
  }

  onCancelExamSubmit(): void {
    this.examConfirmModalOpen.set(false);
  }

  async submitProject(): Promise<void> {
    const ex = this.exercise();
    const environment = this.environment();
    if (!ex || !environment || !this.canSubmitProject()) return;

    this.isSubmittingProject.set(true);
    this.projectError.set(null);

    try {
      const submission = await this.studentService.submitPracticeEnvironmentProject(environment.id, {
        exercise_id: ex.id
      });
      this.examSubmitPending.set(false);
      this.projectSubmission.set(submission);
      this.projectSubmittedAt.set(new Date());
    } catch {
      this.projectError.set('No se pudo registrar la entrega del proyecto.');
    } finally {
      this.isSubmittingProject.set(false);
    }
  }

  async purgeCaches(): Promise<void> {
    const environment = this.environment();
    if (!environment?.id || this.purgingCaches()) return;
    this.purgingCaches.set(true);
    try {
      const result = await this.studentService.purgeEnvironmentCaches(environment.id);
      this.environment.set({
        ...environment,
        disk_used_mb: result.used_mb,
        disk_limit_mb: result.limit_mb,
        disk_state: (result.state as 'ok' | 'warning' | 'full') ?? 'ok',
        state: result.state === 'warning' ? 'disk_warning' : result.state === 'full' ? 'disk_full' : 'ready'
      });
    } catch {
      this.environmentError.set('No se pudieron eliminar los archivos generados.');
    } finally {
      this.purgingCaches.set(false);
    }
  }

  // Primitiva 1: Probar código (Run)
  async testCode(): Promise<void> {
    const ex = this.exercise();
    const code = this.sourceCode();
    if (!ex || !code.trim() || this.isTesting() || this.isSubmitting()) return;
    if (this.isExamProject() && this.isExamExpired()) {
      this.submitError.set('El plazo de entrega para este examen ha finalizado.');
      return;
    }

    this.isTesting.set(true);
    this.submitError.set(null);

    try {
      const evalResult = await this.studentService.evaluateExercise({
        exercise_id: ex.id,
        language: this.selectedLanguage(),
        source_code_b64: encodeUtf8Base64(code),
        mode: 'test'
      });
      this.lastResult.set(evalResult);
      this.isTestRun.set(true);
      this.setConsoleTab('test');
    } catch {
      this.submitError.set('No se pudo probar la solución con los casos de ejemplo.');
    } finally {
      this.isTesting.set(false);
    }
  }

  // Primitiva 2: Enviar a evaluación / Entregar examen (Submit)
  async submitSolution(): Promise<void> {
    const ex = this.exercise();
    const code = this.sourceCode();
    if (!ex || !code.trim() || this.isSubmitting() || this.isTesting()) return;
    if (this.isExamProject() && this.isExamExpired()) {
      this.submitError.set('El plazo de entrega para este examen ha finalizado.');
      return;
    }
    if (this.isExamProject() && !this.canResubmitProject() && this.lastSubmissionId()) {
      this.submitError.set('Este examen no admite reentregas.');
      return;
    }

    this.isSubmitting.set(true);
    this.submitError.set(null);

    try {
      const evalResult = await this.studentService.evaluateExercise({
        exercise_id: ex.id,
        language: this.selectedLanguage(),
        source_code_b64: encodeUtf8Base64(code),
        mode: 'submit'
      });
      this.lastResult.set(evalResult);
      this.isTestRun.set(false);
      this.setConsoleTab('submit');

      const studentId = this.authService.currentUser()?.id || '00000000-0000-0000-0000-000000000301';
      const subResp = await this.studentService.submitExercise({
        exercise_id: ex.id,
        student_id: studentId,
        code,
        verdict: evalResult.verdict,
        execution_time_ms: evalResult.execution_time_ms || 1,
        memory_used_mb: Math.max(1, Math.round(evalResult.memory_used_mb || 12)),
        generated_cases: evalResult.generated_cases,
        evaluation_receipt: evalResult.evaluation_receipt
      });

      if (subResp.id) {
        this.lastSubmissionId.set(subResp.id);
        if (this.exercise()?.purpose === 'exam' && this.keystrokeBuffer.length > 0) {
          const eventsToFlush = [...this.keystrokeBuffer];
          this.keystrokeBuffer.length = 0;
          await this.studentService.saveKeystrokeEvents(subResp.id, eventsToFlush);
        }
      }
    } catch {
      this.submitError.set('No se pudo evaluar la solución en el Entorno de Evaluación.');
    } finally {
      this.isSubmitting.set(false);
    }
  }
}
