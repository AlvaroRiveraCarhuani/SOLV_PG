import { Component, DestroyRef, OnInit, computed, inject, signal } from '@angular/core';
import { CommonModule, Location } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import {
  LucideArrowLeft,
  LucidePlay,
  LucideRotateCcw,
  LucideFileCode,
  LucideTerminal,
  LucideFileText,
  LucideClock,
  LucideAlertTriangle,
  LucideCheckCircle2,
  LucideLoader2,
  LucideLayers,
  LucideShield,
  LucideExternalLink,
  LucideRefreshCw
} from '@lucide/angular';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { CodeEditorComponent } from '@shared/components/code-editor/code-editor.component';
import { ComboboxComponent, ComboboxOption } from '@shared/components/combobox/combobox.component';
import { VerdictBadgeComponent } from '@shared/components/verdict-badge/verdict-badge.component';
import { DateTextPipe } from '@shared/pipes/date-text.pipe';
import { EnvironmentState, PracticeEnvironment } from '@core/models/environment.model';
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

import { EnvironmentLabelPipe } from '@shared/pipes/environment-label.pipe';

/** Cuota de CPU en milésimas como texto legible; vacío si no hay cuota guardada. */
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
    EnvironmentLabelPipe,
    CommonModule,
    FormsModule,
    RouterLink,
    LucideArrowLeft,
    LucidePlay,
    LucideRotateCcw,
    LucideFileCode,
    LucideTerminal,
    LucideFileText,
    LucideClock,
    LucideAlertTriangle,
    LucideCheckCircle2,
    LucideLoader2,
    LucideLayers,
    LucideShield,
    LucideExternalLink,
    LucideRefreshCw,
    MachineDataDirective,
    CodeEditorComponent,
    ComboboxComponent,
    VerdictBadgeComponent,
    DateTextPipe,
    AdvanceConfirmModalComponent
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

  readonly selectedLanguage = signal<string>('python');
  readonly sourceCode = signal<string>('');
  readonly isTesting = signal<boolean>(false);
  readonly isSubmitting = signal<boolean>(false);
  readonly isTestRun = signal<boolean>(false);
  readonly submitError = signal<string | null>(null);
  readonly lastResult = signal<StudentEvaluationResultDTO | null>(null);
  readonly lastSubmissionId = signal<string>('');

  private readonly keystrokeBuffer: KeystrokeEventPayload[] = [];

  // IDE Persistente: environment del alumno, archivos del proyecto y entrega a SpeedGrader.
  readonly environment = signal<PracticeEnvironment | null>(null);
  readonly isStartingEnvironment = signal<boolean>(false);
  readonly environmentError = signal<string | null>(null);
  readonly isSubmittingProject = signal<boolean>(false);
  readonly projectError = signal<string | null>(null);
  readonly projectSubmission = signal<PracticeEnvironmentProjectSubmissionDTO | null>(null);
  readonly projectSubmittedAt = signal<Date | null>(null);
  /**
   * Entrega de un examen: el primer clic pide confirmación porque la entrega
   * queda registrada como una versión. El entorno sigue escribible y se puede
   * volver a entregar mientras el examen esté abierto (ADR-005 §5.2).
   */
  readonly examSubmitPending = signal<boolean>(false);

  /** Avisos de avance del docente que corresponden a esta práctica. */
  readonly practiceAdvances = computed<PendingClassAdvance[]>(() =>
    this.advances.pending().filter(advance => advance.exercise_id === this.exerciseId())
  );
  readonly selectedAdvance = signal<PendingClassAdvance | null>(null);
  readonly isDismissingAdvance = signal<boolean>(false);

  /**
   * Lenguajes que el entorno declaró para esta práctica. Sin declaración se
   * conserva el contrato anterior: solo el lenguaje del ejercicio.
   */
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

  readonly renderedDescription = computed<string>(() => {
    return renderStatementMarkdown(this.exercise()?.description || '');
  });

  readonly exampleCases = computed<StudentPublicTestCaseDTO[]>(() => {
    return this.exercise()?.public_test_cases ?? [];
  });

  readonly constraintsList = computed<string[]>(() => {
    return this.exercise()?.constraints ?? [];
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
  /** URL de la aplicación del alumno, cuando la práctica declara app_port. */
  readonly appUrl = computed<string | null>(() => this.environment()?.app_url ?? null);
  /** URL del IDE, para cuando no hay pestaña nueva donde abrirlo. */
  readonly ideUrl = computed<string | null>(() => this.environment()?.access_url ?? null);
  /**
   * El navegador no permitió abrir la pestaña del entorno. En ese caso la
   * página deja los enlaces a la vista en lugar de intentar abrirla de nuevo.
   */
  readonly popupBlocked = signal<boolean>(false);

  /**
   * Estado que ve el alumno. Sale del campo calculado por el servidor; si no
   * viene (respuesta vieja), se deduce del estado guardado.
   */
  /**
   * El examen ya terminó: se sabe al intentar abrirlo, aunque no haya entorno
   * cargado. Sin esto el alumno solo veía un error y nunca "Examen cerrado".
   */
  readonly examClosed = signal<boolean>(false);

  /**
   * La clase está en pausa: el laboratorio de una práctica quedó en solo
   * lectura aunque no sea un examen. La escritura vuelve cuando el docente
   * reanuda el aula.
   */
  readonly classPaused = signal<boolean>(false);

  /**
   * Aviso de espacio antes del bloqueo: la cuota es blanda y el alumno tiene que
   * poder liberar lugar mientras sigue trabajando.
   */
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
        text: `Tu espacio está lleno (${used} MB de ${limit} MB). Borrá archivos generados, como node_modules, y volvé a entregar; tu editor sigue abierto.`
      };
    }
    const percent = limit > 0 ? Math.round((used * 100) / limit) : 0;
    return {
      tone: 'warning',
      text: `Tu espacio está al ${percent}% (${used} MB de ${limit} MB). El límite es blando: podés seguir trabajando y borrar archivos generados para liberar lugar.`
    };
  });

  /**
   * Los tres números de cuota del entorno: perfil declarado por la plantilla,
   * memoria concedida por la admisión y CPU aplicada. La advertencia aparece
   * solo cuando la concedida quedó por debajo de la declarada.
   */
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

  /**
   * Limpia las carpetas que se regeneran y actualiza el aviso con la medición
   * que devuelve el servidor, sin recargar la pantalla.
   */
  async purgeCaches(): Promise<void> {
    const environment = this.environment();
    if (!environment?.id || this.purgingCaches()) {
      return;
    }
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
    } catch (error) {
      const message = (error as { error?: { message?: unknown } } | null)?.error?.message;
      this.environmentError.set(
        typeof message === 'string' && message.trim() ? message : 'No se pudieron borrar los archivos generados.'
      );
    } finally {
      this.purgingCaches.set(false);
    }
  }

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

  readonly canResubmitProject = computed<boolean>(() => this.exercise()?.allow_resubmit ?? true);

  readonly isExamProject = computed<boolean>(() => (this.exercise()?.purpose ?? 'class') === 'exam');

  /** El aviso de solo lectura de una práctica pausada no es el de un examen. */
  readonly showsClassPausedNotice = computed<boolean>(() => {
    if (this.isExamProject()) {
      return false;
    }
    return this.classPaused() || this.environment()?.status === 'frozen';
  });

  readonly isExamExpired = computed<boolean>(() => {
    const due = this.exercise()?.due_date;
    if (!due) return false;
    return new Date(due).getTime() < Date.now();
  });

  /**
   * Entrega de examen en el Entorno de Evaluación: si el examen venció o si
   * no admite reentregas y ya tiene entrega, se bloquea. Con reentregas permitidas
   * y dentro del plazo, el estudiante puede corregir y volver a entregar.
   */
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

  /** Identificador de la entrega del proyecto, para abrir su historial. */
  readonly projectSubmissionId = computed<string>(() => {
    return this.projectSubmission()?.submission_id ?? this.lastSubmissionId();
  });

  readonly hasProjectSubmission = computed<boolean>(() => this.projectSubmissionId() !== '');

  readonly canSubmitProject = computed<boolean>(() => {
    if (!this.environmentIsRunning() || this.isSubmittingProject()) {
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

  /**
   * El límite de tiempo pertenece al juez: mostrarlo en una práctica de IDE
   * habla de un recurso que esa práctica no usa. La memoria del contenedor es
   * del nodo, no del alumno, y tampoco se muestra.
   */
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

  readonly backCourseLink = computed<string[]>(() => {
    const subjectId = this.exercise()?.subject_id;
    return subjectId ? ['/student/courses', subjectId] : ['/student'];
  });

  goBack(): void {
    if (typeof window !== 'undefined' && (window.history.state?.navigationId ?? 0) > 1) {
      this.location.back();
    } else {
      void this.router.navigate(this.backCourseLink());
    }
  }

  ngOnInit(): void {
    // El aviso llega a la pantalla donde el alumno trabaja la práctica, no solo
    // al laboratorio por materia. La consulta corre con el mismo ciclo que en
    // esa pantalla (ADR-058 §8).
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

    this.exercise.set(data);

    const history = this.studentService.submissionsHistory ? this.studentService.submissionsHistory() : [];
    const prevSub = history.find((s) => s.exercise_id === id);
    if (prevSub) {
      this.lastSubmissionId.set(prevSub.id);
      if (prevSub.code && !this.isIdeExercise()) {
        this.sourceCode.set(prevSub.code);
      }
    } else if (this.studentService.loadSubmissionsHistory) {
      void this.studentService.loadSubmissionsHistory().then((h) => {
        const found = h?.find((s) => s.exercise_id === id);
        if (found) {
          this.lastSubmissionId.set(found.id);
          if (found.code && !this.isIdeExercise()) {
            this.sourceCode.set(found.code);
          }
        }
      });
    }

    if (this.isIdeExercise()) {
      return;
    }

    const lang = (data.language || 'python').toLowerCase();
    const allowed = (data.allowed_languages ?? [])
      .map(candidate => (candidate || '').toLowerCase())
      .filter(Boolean);
    const preferred = allowed.includes(lang) ? lang : (allowed[0] || lang);
    this.selectedLanguage.set(preferred);
    if (!prevSub?.code) {
      this.sourceCode.set(data.boilerplates?.[preferred] ?? (preferred === lang ? data.boilerplate || '' : ''));
    }
  }

  onLanguageChange(lang: string): void {
    if (!lang) return;
    const next = lang.toLowerCase();
    const previous = this.selectedLanguage();
    if (previous === next) return;

    // El alumno recibe el código de partida del lenguaje elegido; si ya escribió
    // su solución, cambiar de lenguaje no la borra.
    const base = (this.exercise()?.language || '').toLowerCase();
    const starters = this.exercise()?.boilerplates ?? {};
    const starterFor = (language: string): string =>
      starters[language] ?? (language === base ? this.exercise()?.boilerplate ?? '' : '');
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

    // Solo el examen registra tecleo: la entrega de práctica no se supervisa.
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
    this.sourceCode.set(this.exercise()?.boilerplate || '');
  }

  openAdvance(advance: PendingClassAdvance): void {
    this.selectedAdvance.set(advance);
  }

  closeAdvanceModal(): void {
    this.selectedAdvance.set(null);
  }

  async dismissAdvance(advance: PendingClassAdvance): Promise<void> {
    if (this.isDismissingAdvance()) {
      return;
    }
    this.isDismissingAdvance.set(true);
    try {
      await this.advances.dismiss(advance.id);
    } catch {
      // Si el descarte falla, el aviso sigue visible y se puede reintentar.
    } finally {
      this.isDismissingAdvance.set(false);
    }
  }

  /**
   * Abre el entorno de la práctica en un solo paso, como el docente: pide el
   * arranque (que reanuda si ya existía) y lleva la pestaña a la URL cuando el
   * backend la entrega. La pestaña se abre dentro del gesto del usuario para
   * que el navegador no la bloquee.
   */
  async openEnvironment(): Promise<void> {
    const ex = this.exercise();
    if (!ex || this.isStartingEnvironment()) {
      return;
    }
    const subjectId = ex.subject_id;
    if (!subjectId) {
      this.environmentError.set('El ejercicio no tiene una materia asociada para abrir el entorno.');
      return;
    }

    // La pestaña de preparación lleva la ruta del ejercicio: si el arranque no
    // responde, su aviso puede volver acá en vez de girar sin fin.
    const preparacion = typeof window !== 'undefined'
      ? `/entorno.html?volver=${encodeURIComponent(window.location.pathname + window.location.search)}`
      : '/entorno.html';
    const tab = typeof window !== 'undefined' ? window.open(preparacion, '_blank') : null;
    if (tab) {
      tab.opener = null;
    }

    // Sin pestaña nueva no hay dónde seguir el arranque: la página muestra los
    // enlaces al IDE y a la aplicación, que no dependen de una ventana abierta.
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
        this.environmentError.set('Tu entorno se está preparando. Probá de nuevo en unos segundos.');
        return;
      }

      if (!tab) {
        return;
      }
      tab.location.href = url;
    } catch (error) {
      tab?.close();
      const body = (error as { error?: { message?: unknown; error?: unknown } } | null)?.error;
      // El backend responde { error: <código>, message } y algunos clientes lo
      // envuelven como { error: { code } }: se aceptan las dos formas.
      const rawCode = body?.error;
      const code = typeof rawCode === 'string'
        ? rawCode
        : (rawCode as { code?: unknown } | null | undefined)?.code;
      if (
        code === 'environment_frozen' ||
        code === 'environment_frozen_exam_paused' ||
        code === 'environment_frozen_class'
      ) {
        // El volumen congelado tiene su propio aviso de estado, no un error suelto.
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
          : 'No se pudo abrir tu entorno. Intentá de nuevo en unos segundos.'
      );
    } finally {
      this.isStartingEnvironment.set(false);
    }
  }

  /**
   * Entrada del botón de entrega. En un examen no entrega al primer clic: pide
   * una confirmación porque la entrega queda registrada como versión.
   */
  requestSubmitProject(): void {
    if (!this.canSubmitProject()) {
      return;
    }
    if (this.isExamProject() && !this.examSubmitPending()) {
      this.examSubmitPending.set(true);
      return;
    }
    void this.submitProject();
  }

  cancelExamSubmit(): void {
    this.examSubmitPending.set(false);
  }

  async submitProject(): Promise<void> {
    const ex = this.exercise();
    const environment = this.environment();
    if (!ex || !environment || !this.canSubmitProject()) {
      return;
    }

    this.isSubmittingProject.set(true);
    this.projectError.set(null);

    try {
      // La entrega es el volumen de la práctica: el backend lo fotografía tal
      // como está. No se mandan archivos desde el navegador, para no pisar lo
      // que el alumno escribió en el IDE.
      const submission = await this.studentService.submitPracticeEnvironmentProject(environment.id, {
        exercise_id: ex.id
      });
      this.examSubmitPending.set(false);
      this.projectSubmission.set(submission);
      this.projectSubmittedAt.set(new Date());
    } catch {
      this.examSubmitPending.set(false);
      this.projectError.set('No se pudo registrar la entrega del proyecto.');
    } finally {
      this.isSubmittingProject.set(false);
    }
  }

  async testCode(): Promise<void> {
    const ex = this.exercise();
    const code = this.sourceCode();
    if (!ex || !code.trim() || this.isTesting() || this.isSubmitting()) {
      return;
    }
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
    } catch {
      this.submitError.set('No se pudo probar la solución con los casos de ejemplo.');
    } finally {
      this.isTesting.set(false);
    }
  }

  async submitSolution(): Promise<void> {
    const ex = this.exercise();
    const code = this.sourceCode();
    if (!ex || !code.trim() || this.isSubmitting() || this.isTesting()) {
      return;
    }
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
