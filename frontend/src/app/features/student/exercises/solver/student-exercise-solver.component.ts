import { Component, DestroyRef, OnInit, computed, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, RouterLink } from '@angular/router';
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
import { marked } from 'marked';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { CodeEditorComponent } from '@shared/components/code-editor/code-editor.component';
import { ComboboxComponent, ComboboxOption } from '@shared/components/combobox/combobox.component';
import { VerdictBadgeComponent } from '@shared/components/verdict-badge/verdict-badge.component';
import { DateTextPipe } from '@shared/pipes/date-text.pipe';
import { PracticeEnvironment } from '@core/models/environment.model';
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
  readonly isSubmitting = signal<boolean>(false);
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
   * Entrega de un examen: el primer clic pide confirmación porque el volumen
   * queda en solo lectura y no se puede seguir editando (ADR-005 §5.2).
   */
  readonly examSubmitPending = signal<boolean>(false);

  /** Avisos de avance del docente que corresponden a esta práctica. */
  readonly practiceAdvances = computed<PendingClassAdvance[]>(() =>
    this.advances.pending().filter(advance => advance.exercise_id === this.exerciseId())
  );
  readonly selectedAdvance = signal<PendingClassAdvance | null>(null);
  readonly isDismissingAdvance = signal<boolean>(false);

  readonly languageOptions = computed<ComboboxOption[]>(() => {
    const baseLang = (this.exercise()?.language || 'python').toLowerCase();
    const candidates = Array.from(new Set([baseLang, 'python', 'cpp', 'c', 'java', 'javascript']));
    return candidates.map((lang) => ({
      id: lang,
      label: lang.toUpperCase(),
      value: lang
    }));
  });

  readonly renderedDescription = computed<string>(() => {
    const raw = this.exercise()?.description || '';
    if (!raw.trim()) {
      return '<p>Sin descripción disponible.</p>';
    }
    return marked.parse(raw) as string;
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

  readonly canResubmitProject = computed<boolean>(() => this.exercise()?.allow_resubmit ?? true);

  readonly isExamProject = computed<boolean>(() => (this.exercise()?.purpose ?? 'class') === 'exam');

  readonly isExamSubmitted = computed<boolean>(() => {
    return this.isExamProject() && !!this.lastSubmissionId();
  });

  readonly canSubmitProject = computed<boolean>(() => {
    if (this.isExamSubmitted()) {
      return false;
    }
    if (!this.environmentIsRunning() || this.isSubmittingProject()) {
      return false;
    }
    // En un examen la entrega es final: el volumen queda en solo lectura, así
    // que no hay segunda entrega aunque el ejercicio la permita.
    if (this.projectSubmission() && (this.isExamProject() || !this.canResubmitProject())) {
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
    this.selectedLanguage.set(lang);
    if (!prevSub?.code) {
      this.sourceCode.set(data.boilerplate || '');
    }
  }

  onLanguageChange(lang: string): void {
    if (!lang) return;
    this.selectedLanguage.set(lang.toLowerCase());
  }

  onCodeChange(nextCode: string): void {
    const prevLen = this.sourceCode().length;
    const delta = nextCode.length - prevLen;
    this.sourceCode.set(nextCode);

    if (delta !== 0) {
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

    const tab = typeof window !== 'undefined' ? window.open('', '_blank') : null;
    if (tab) {
      tab.opener = null;
    }

    this.isStartingEnvironment.set(true);
    this.environmentError.set(null);

    try {
      const environment = await this.studentService.startEnvironment(subjectId, ex.id);
      this.environment.set(environment);

      const url = environment.access_url;
      if (!url) {
        tab?.close();
        this.environmentError.set('Tu entorno se está preparando. Probá de nuevo en unos segundos.');
        return;
      }

      if (tab) {
        tab.location.href = url;
      } else {
        window.open(url, '_blank');
      }
    } catch (error) {
      tab?.close();
      const backendMessage = (error as { error?: { message?: unknown } } | null)?.error?.message;
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
   * una confirmación porque el entorno se cierra para ese alumno.
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

  async submitSolution(): Promise<void> {
    const ex = this.exercise();
    const code = this.sourceCode();
    if (!ex || !code.trim() || this.isSubmitting()) {
      return;
    }

    this.isSubmitting.set(true);
    this.submitError.set(null);

    try {
      const evalResult = await this.studentService.evaluateExercise({
        exercise_id: ex.id,
        language: this.selectedLanguage(),
        source_code_b64: encodeUtf8Base64(code)
      });
      this.lastResult.set(evalResult);

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
        if (this.keystrokeBuffer.length > 0) {
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
