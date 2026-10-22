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
  LucideCpu,
  LucideAlertTriangle,
  LucideCheckCircle2,
  LucideLoader2,
  LucideLayers,
  LucideShield,
  LucideExternalLink,
  LucideRefreshCw,
  LucidePlus,
  LucideTrash2
} from '@lucide/angular';
import { marked } from 'marked';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { CodeEditorComponent } from '@shared/components/code-editor/code-editor.component';
import { ComboboxComponent, ComboboxOption } from '@shared/components/combobox/combobox.component';
import { VerdictBadgeComponent } from '@shared/components/verdict-badge/verdict-badge.component';
import { DateTextPipe } from '@shared/pipes/date-text.pipe';
import { WorkspaceInstance } from '@core/models/workspace.model';
import {
  StudentEvaluationResultDTO,
  StudentPublicExerciseDTO,
  StudentPublicTestCaseDTO,
  StudentService,
  WorkspaceProjectSubmissionDTO
} from '@core/services/student.service';
import { AuthService } from '@core/services/auth.service';

interface KeystrokeEventPayload {
  event_type: 'keystroke' | 'paste';
  timestamp_ms: number;
  char_delta: number;
  cursor_line: number;
}

interface ProjectFile {
  name: string;
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

@Component({
  selector: 'student-exercise-solver',
  standalone: true,
  imports: [
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
    LucideCpu,
    LucideAlertTriangle,
    LucideCheckCircle2,
    LucideLoader2,
    LucideLayers,
    LucideShield,
    LucideExternalLink,
    LucideRefreshCw,
    LucidePlus,
    LucideTrash2,
    MachineDataDirective,
    CodeEditorComponent,
    ComboboxComponent,
    VerdictBadgeComponent,
    DateTextPipe
  ],
  templateUrl: './student-exercise-solver.component.html',
  styleUrl: './student-exercise-solver.component.scss'
})
export class StudentExerciseSolverComponent implements OnInit {
  private readonly route = inject(ActivatedRoute);
  private readonly studentService = inject(StudentService);
  private readonly authService = inject(AuthService);
  private readonly destroyRef = inject(DestroyRef);

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

  // IDE Persistente: workspace del alumno, archivos del proyecto y entrega a SpeedGrader.
  readonly workspace = signal<WorkspaceInstance | null>(null);
  readonly isStartingWorkspace = signal<boolean>(false);
  readonly workspaceError = signal<string | null>(null);
  readonly projectFiles = signal<ProjectFile[]>([]);
  readonly selectedFileName = signal<string>('');
  readonly newFileName = signal<string>('');
  readonly isSubmittingProject = signal<boolean>(false);
  readonly projectError = signal<string | null>(null);
  readonly projectSubmission = signal<WorkspaceProjectSubmissionDTO | null>(null);
  readonly projectSubmittedAt = signal<Date | null>(null);

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
      ex?.type === 'workspace' ||
      ex?.environment_type === 'IDE_PERSISTENTE' ||
      ex?.environment_type === 'workspace'
    );
  });

  readonly workspaceIsRunning = computed<boolean>(() => this.workspace()?.status === 'running');

  readonly projectFilesCount = computed<number>(() => this.projectFiles().length);

  readonly selectedFileContent = computed<string>(() => {
    const name = this.selectedFileName();
    return this.projectFiles().find(file => file.name === name)?.content ?? '';
  });

  readonly projectEditorLanguage = computed<string>(() => {
    const name = this.selectedFileName().toLowerCase();
    const extension = name.includes('.') ? name.split('.').pop() ?? '' : '';
    const byExtension: Record<string, string> = {
      py: 'python',
      sql: 'sql',
      js: 'javascript',
      ts: 'typescript',
      cpp: 'cpp',
      c: 'c',
      java: 'java',
      go: 'go',
      json: 'json',
      md: 'markdown',
      html: 'html',
      css: 'css',
      sh: 'shell'
    };
    return byExtension[extension] || 'plaintext';
  });

  readonly canResubmitProject = computed<boolean>(() => this.exercise()?.allow_resubmit ?? true);

  readonly canSubmitProject = computed<boolean>(() => {
    if (!this.workspaceIsRunning() || this.isSubmittingProject()) {
      return false;
    }
    const hasFiles = this.projectFiles().some(file => file.name.trim() !== '' && file.content.trim() !== '');
    if (!hasFiles) {
      return false;
    }
    if (this.projectSubmission() && !this.canResubmitProject()) {
      return false;
    }
    return true;
  });

  readonly effectiveMemoryLimitMb = computed<number>(() => {
    const ex = this.exercise();
    return ex?.memory_limit_mb || ex?.memory_mb || 128;
  });

  readonly effectiveTimeLimitMs = computed<number>(() => {
    return this.exercise()?.time_limit_ms || 1000;
  });

  readonly backCourseLink = computed<string[]>(() => {
    const subjectId = this.exercise()?.subject_id;
    return subjectId ? ['/student/courses', subjectId] : ['/student'];
  });

  ngOnInit(): void {
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

    if (this.isIdeExercise()) {
      this.ensureDefaultProjectFile();
      return;
    }

    const lang = (data.language || 'python').toLowerCase();
    this.selectedLanguage.set(lang);
    this.sourceCode.set(data.boilerplate || '');
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
      this.keystrokeBuffer.push({
        event_type: Math.abs(delta) > 25 ? 'paste' : 'keystroke',
        timestamp_ms: Date.now(),
        char_delta: delta,
        cursor_line: nextCode.split('\n').length
      });
    }
  }

  resetBoilerplate(): void {
    this.sourceCode.set(this.exercise()?.boilerplate || '');
  }

  private defaultProjectFileName(): string {
    const ex = this.exercise();
    const language = (ex?.language || '').toLowerCase();
    if (language === 'sql' || language === 'postgresql') {
      return 'solution.sql';
    }
    const extensionByLanguage: Record<string, string> = {
      python: 'py',
      cpp: 'cpp',
      c: 'c',
      java: 'java',
      javascript: 'js',
      typescript: 'ts',
      go: 'go'
    };
    return `main.${extensionByLanguage[language] || 'py'}`;
  }

  private ensureDefaultProjectFile(): void {
    if (this.projectFiles().length > 0) {
      return;
    }
    const name = this.defaultProjectFileName();
    const boilerplate = this.exercise()?.boilerplate?.trim() ? this.exercise()?.boilerplate ?? '' : '';
    this.projectFiles.set([{ name, content: boilerplate }]);
    this.selectedFileName.set(name);
  }

  async startIdeWorkspace(): Promise<void> {
    const ex = this.exercise();
    if (!ex || this.isStartingWorkspace()) {
      return;
    }
    const subjectId = ex.subject_id;
    if (!subjectId) {
      this.workspaceError.set('El ejercicio no tiene una materia asociada para iniciar el workspace.');
      return;
    }

    this.isStartingWorkspace.set(true);
    this.workspaceError.set(null);

    try {
      const workspace = await this.studentService.startWorkspace(subjectId, ex.id);
      this.workspace.set(workspace);
      this.ensureDefaultProjectFile();
    } catch {
      this.workspaceError.set('No se pudo iniciar el IDE. Intentá de nuevo en unos segundos.');
    } finally {
      this.isStartingWorkspace.set(false);
    }
  }

  openIdeInNewTab(): void {
    const url = this.workspace()?.access_url;
    if (!url) {
      return;
    }
    window.open(url, '_blank', 'noopener');
  }

  selectProjectFile(name: string): void {
    this.selectedFileName.set(name);
  }

  onProjectFileChange(content: string): void {
    const name = this.selectedFileName();
    this.projectFiles.update(files =>
      files.map(file => (file.name === name ? { ...file, content } : file))
    );
  }

  addProjectFile(): void {
    const name = this.newFileName().trim().replace(/\s+/g, '-');
    if (!name) {
      return;
    }
    if (this.projectFiles().some(file => file.name === name)) {
      this.selectedFileName.set(name);
      this.newFileName.set('');
      return;
    }
    this.projectFiles.update(files => [...files, { name, content: '' }]);
    this.selectedFileName.set(name);
    this.newFileName.set('');
  }

  removeProjectFile(name: string): void {
    const remaining = this.projectFiles().filter(file => file.name !== name);
    if (remaining.length === 0) {
      return;
    }
    this.projectFiles.set(remaining);
    if (this.selectedFileName() === name) {
      this.selectedFileName.set(remaining[0].name);
    }
  }

  async submitProject(): Promise<void> {
    const ex = this.exercise();
    const workspace = this.workspace();
    if (!ex || !workspace || !this.canSubmitProject()) {
      return;
    }

    const files: Record<string, string> = {};
    for (const file of this.projectFiles()) {
      const name = file.name.trim();
      if (name) {
        files[name] = file.content;
      }
    }

    this.isSubmittingProject.set(true);
    this.projectError.set(null);

    try {
      const submission = await this.studentService.submitWorkspaceProject(workspace.id, {
        exercise_id: ex.id,
        files
      });
      this.projectSubmission.set(submission);
      this.projectSubmittedAt.set(new Date());
    } catch {
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
        generated_cases: evalResult.generated_cases
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
      this.submitError.set('No se pudo evaluar la solución en el Juez Virtual.');
    } finally {
      this.isSubmitting.set(false);
    }
  }
}
