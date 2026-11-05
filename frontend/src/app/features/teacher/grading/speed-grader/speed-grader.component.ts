import { Component, OnInit, OnDestroy, inject, signal, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, Router } from '@angular/router';
import { HttpClient } from '@angular/common/http';
import { RubricDTO } from '../../exercise-editor/exercise-editor.store';
import { TeacherGradingService } from '../../services/teacher-grading.service';
import { HotkeysService } from '@core/services/hotkeys.service';
import { LucideAlertCircle, LucideExternalLink, LucidePanelLeftOpen } from '@lucide/angular';
import {
  ProjectFile,
  ReviewEnvironmentAccess,
  ReviewMode,
  reviewModeFromEnvironment,
  SubmissionComment,
  SubmissionTimeline
} from '../../models/teacher.models';
import { DateTextPipe } from '@shared/pipes/date-text.pipe';
import { commentsForFile, commentsOutsideFiles, globalToLocalLine } from './review-line-map';
import { ReviewFileTreeComponent } from '@shared/components/review-file-tree/review-file-tree.component';
import { SkeletonLoaderComponent } from '@shared/components/skeleton/skeleton-loader.component';
import { KeystrokeReplayModalComponent } from './keystroke-replay-modal/keystroke-replay-modal.component';
import { SpeedGraderHeaderComponent } from './components/speed-grader-header/speed-grader-header.component';
import { SpeedGraderCodeViewerComponent } from './components/speed-grader-code-viewer/speed-grader-code-viewer.component';
import { SpeedGraderEvalPanelComponent } from './components/speed-grader-eval-panel/speed-grader-eval-panel.component';
import { SpeedGraderOverrideModalComponent } from './components/speed-grader-override-modal/speed-grader-override-modal.component';
import { VERDICT_OVERRIDE_OPTIONS } from '@core/models/verdict.model';

@Component({
  selector: 'speed-grader',
  standalone: true,
  imports: [
    CommonModule,
    LucideAlertCircle,
    LucideExternalLink,
    LucidePanelLeftOpen,
    SkeletonLoaderComponent,
    KeystrokeReplayModalComponent,
    ReviewFileTreeComponent,
    SpeedGraderHeaderComponent,
    SpeedGraderCodeViewerComponent,
    SpeedGraderEvalPanelComponent,
    SpeedGraderOverrideModalComponent,
    DateTextPipe
  ],
  templateUrl: './speed-grader.component.html',
  styleUrl: './speed-grader.component.scss'
})
export class SpeedGraderComponent implements OnInit, OnDestroy {
  private route = inject(ActivatedRoute);
  private router = inject(Router);
  private gradingService = inject(TeacherGradingService);
  private hotkeysService = inject(HotkeysService);
  private http = inject(HttpClient);
  private unregisterFns: Array<() => void> = [];

  readonly verdictOptions: { id: string; label: string; value: string }[] = VERDICT_OVERRIDE_OPTIONS;

  submissionId = signal<string>('');
  review = this.gradingService.currentReview;
  readonly reviewMode = computed<ReviewMode>(() =>
    reviewModeFromEnvironment(this.review()?.environment_type)
  );
  readonly isJudge = computed(() => this.reviewMode() === 'judge');
  readonly isProject = computed(() => this.reviewMode() === 'project');

  // Archivos del proyecto (solo IDE). En juez el payload llega con files = [].
  readonly selectedFilePath = signal<string | null>(null);
  readonly treeCollapsed = signal<boolean>(false);
  readonly files = computed<ProjectFile[]>(() => this.review()?.files ?? []);
  readonly filesAvailable = computed(
    () => this.review()?.files_available === true && this.files().length > 0
  );
  readonly showFileTree = computed(() => this.isProject() && this.filesAvailable());
  readonly activeFile = computed<ProjectFile | null>(() => {
    if (!this.showFileTree()) return null;
    const files = this.files();
    return files.find((file) => file.path === this.selectedFilePath()) ?? files[0] ?? null;
  });

  /** Base del diff: boilerplate en juez, archivo homónimo del proyecto base en IDE. */
  readonly activeBaseFile = computed<ProjectFile | null>(() => {
    const file = this.activeFile();
    if (!file) return null;
    return (this.review()?.base_files ?? []).find((base) => base.path === file.path) ?? null;
  });
  readonly diffOriginal = computed<string>(() =>
    this.isProject() ? (this.activeBaseFile()?.content ?? '') : this.starterBoilerplate()
  );
  readonly diffModified = computed<string>(() =>
    this.isProject() ? (this.activeFile()?.content ?? '') : (this.review()?.code ?? '')
  );
  readonly diffLanguage = computed<string>(() =>
    this.isProject() ? (this.activeFile()?.language ?? 'plaintext') : (this.review()?.language ?? 'plaintext')
  );
  readonly canShowDiff = computed<boolean>(() => {
    if (this.isProject()) return !!this.activeBaseFile();
    return this.starterBoilerplate().trim().length > 0;
  });
  readonly activeFileComments = computed<SubmissionComment[]>(() => {
    const file = this.activeFile();
    return file ? commentsForFile(this.comments(), file) : [];
  });
  readonly generalComments = computed<SubmissionComment[]>(() => {
    if (!this.isProject()) return [];
    return commentsOutsideFiles(this.comments(), this.files());
  });

  comments = this.gradingService.comments;
  isLoading = this.gradingService.isLoading;
  isRunningEphemeral = this.gradingService.isRunningEphemeral;
  ephemeralResult = this.gradingService.ephemeralResult;
  isStartingLiveApp = this.gradingService.isStartingReviewEnvironment;
  liveApp = signal<ReviewEnvironmentAccess | null>(null);
  liveAppError = signal<string | null>(null);

  // Rubrica
  rubric = signal<RubricDTO | null>(null);
  selections = signal<Record<string, string>>({});
  rubricComments = signal<string>('');
  isSubmittingRubric = signal<boolean>(false);
  rubricSuccessMessage = signal<string | null>(null);
  rubricErrorMessage = signal<string | null>(null);

  calculatedRubricScore = computed(() => {
    const rub = this.rubric();
    if (!rub || !rub.criteria) return 0;
    const sel = this.selections();
    let total = 0;
    for (const c of rub.criteria) {
      if (!c.levels || c.levels.length === 0) continue;
      const selectedName = sel[c.id || ''] || c.levels[0].name;
      const lvl = c.levels.find(l => l.name === selectedName);
      if (lvl) {
        total += (c.weight / 100.0) * lvl.score;
      }
    }
    return Math.round(total * 100) / 100;
  });

  onLevelSelected(criterionId: string, levelName: string): void {
    this.selections.update(curr => ({ ...curr, [criterionId]: levelName }));
  }

  saveRubricEvaluation(): void {
    const id = this.submissionId();
    if (!id || !this.rubric()) return;

    this.isSubmittingRubric.set(true);
    this.rubricSuccessMessage.set(null);
    this.rubricErrorMessage.set(null);

    const payload = {
      selections: this.selections(),
      comments: this.rubricComments()
    };

    this.http.post<{ data?: { score?: number } }>(`/api/v1/submissions/${id}/rubric-evaluation`, payload).subscribe({
      next: (res) => {
        this.isSubmittingRubric.set(false);
        const score = res?.data?.score ?? this.calculatedRubricScore();
        this.rubricSuccessMessage.set(`Calificación guardada: ${score} / 100 pts. La entrega quedó en la libreta.`);
        // La rúbrica no escribe veredicto: se relee la entrega para mostrar lo
        // que quedó guardado, sin inventar un resultado del juez ni un ajuste.
        this.gradingService.getSubmissionReview(id).subscribe({ error: () => undefined });
        if (this.rubricComments().trim()) {
          this.gradingService.getComments(id).subscribe();
        }
      },
      error: (err) => {
        this.isSubmittingRubric.set(false);
        this.rubricErrorMessage.set(
          err?.error?.message || 'No se pudo guardar la calificación. Revisá la libreta e intentá de nuevo.'
        );
      }
    });
  }

  showDiffView = signal<boolean>(false);
  starterBoilerplate = computed<string>(() => this.review()?.boilerplate ?? '');

  // Time-Travel Replay State
  showReplayPlayer = signal<boolean>(false);
  timeline = signal<SubmissionTimeline | null>(null);
  isLoadingTimeline = signal<boolean>(false);

  selectedLineNumber = signal<number | null>(null);
  selectedLineEndNumber = signal<number | null>(null);
  newCommentText = signal<string>('');
  isAddingComment = signal<boolean>(false);

  showOverrideModal = signal<boolean>(false);
  overrideVerdict = signal<string>('AC');
  overrideScore = signal<number>(100);
  overrideReason = signal<string>('');
  overrideNotifyStudent = signal<boolean>(true);
  overrideError = signal<string | null>(null);
  isSubmittingOverride = signal<boolean>(false);

  showKeystrokeReplay = signal<boolean>(false);

  copiedIndex = signal<number | null>(null);

  openKeystrokeReplay(): void {
    this.showKeystrokeReplay.set(true);
  }

  /**
   * Abre el preview en una pestaña nueva. La pestaña se reserva en el clic para
   * que el navegador no bloquee la apertura cuando responde la API; si la API
   * falla, la pestaña vacía se cierra.
   */
  openLiveApplication(): void {
    const active = this.liveApp();
    if (active) {
      this.openTab(active.url);
      return;
    }
    if (this.isStartingLiveApp()) return;

    this.liveAppError.set(null);
    const tab = typeof window !== 'undefined' ? window.open('about:blank', '_blank') : null;
    if (tab) {
      try {
        tab.opener = null;
      } catch {
        // La pestaña ya no es accesible: el enlace del aviso sigue disponible.
      }
    }

    this.gradingService.startReviewEnvironment(this.submissionId()).subscribe({
      next: (access) => {
        this.liveApp.set(access);
        if (tab && !tab.closed) {
          tab.location.replace(access.url);
        } else {
          this.openTab(access.url);
        }
      },
      error: (err) => {
        if (tab && !tab.closed) {
          tab.close();
        }
        this.liveAppError.set(
          err?.error?.message || 'No se pudo abrir el entorno de prueba. Revisá la configuración del laboratorio.'
        );
      }
    });
  }

  private openTab(url: string): void {
    if (typeof window === 'undefined' || !url) return;
    window.open(url, '_blank', 'noopener,noreferrer');
  }

  closeKeystrokeReplay(): void {
    this.showKeystrokeReplay.set(false);
  }

  getComplexityClass(complexity: string): string {
    switch (complexity) {
      case 'O(1)':
      case 'O(log N)':
      case 'O(N)':
        return 'badge-success';
      case 'O(N log N)':
        return 'badge-warning';
      case 'O(N²)':
      case 'O(N³)':
      case 'O(2^N)':
        return 'badge-danger';
      default:
        return 'badge-secondary';
    }
  }

  copyCaseInput(input: string, index: number): void {
    if (typeof navigator !== 'undefined' && navigator.clipboard) {
      navigator.clipboard.writeText(input);
      this.copiedIndex.set(index);
      setTimeout(() => {
        if (this.copiedIndex() === index) {
          this.copiedIndex.set(null);
        }
      }, 2000);
    }
  }

  codeLines = computed(() => {
    return (this.review()?.code || '').split('\n');
  });

  currentCommentRange = computed<{ line: number; endLine: number | null } | null>(() => {
    const line = this.selectedLineNumber();
    if (line === null) return null;
    const end = this.selectedLineEndNumber();
    return { line, endLine: end !== null && end > line ? end : null };
  });

  commentsByLine = computed(() => {
    const map = new Map<number, SubmissionComment[]>();
    for (const c of this.comments()) {
      const lineComments = map.get(c.line_number) || [];
      lineComments.push(c);
      map.set(c.line_number, lineComments);
    }
    return map;
  });

  ngOnInit(): void {
    this.hotkeysService.setScope('speed-grader');
    this.registerHotkeys();

    this.route.paramMap.subscribe(params => {
      const id = params.get('submissionId');
      if (id) {
        this.submissionId.set(id);
        this.loadSubmission(id);
      }
    });
  }

  ngOnDestroy(): void {
    for (const unregister of this.unregisterFns) {
      unregister();
    }
    this.hotkeysService.setScope('global');
  }

  private registerHotkeys(): void {
    // J: Next submission
    this.unregisterFns.push(
      this.hotkeysService.register({
        key: 'j',
        description: 'Siguiente entrega de alumno',
        category: 'Navegación',
        scope: 'speed-grader',
        action: () => {
          const nextId = this.review()?.next_submission_id;
          if (nextId) this.router.navigate(['/teacher/revision', nextId]);
        }
      })
    );

    // K: Previous submission
    this.unregisterFns.push(
      this.hotkeysService.register({
        key: 'k',
        description: 'Entrega anterior de alumno',
        category: 'Navegación',
        scope: 'speed-grader',
        action: () => {
          const prevId = this.review()?.prev_submission_id;
          if (prevId) this.router.navigate(['/teacher/revision', prevId]);
        }
      })
    );

    // A: Modificar Nota / Auto-aprobar
    this.unregisterFns.push(
      this.hotkeysService.register({
        key: 'a',
        description: 'Abrir auditoría / Calificar',
        category: 'SpeedGrader',
        scope: 'speed-grader',
        action: () => this.openOverrideModal()
      })
    );

    // C: Comentar en la primera línea seleccionada
    this.unregisterFns.push(
      this.hotkeysService.register({
        key: 'c',
        description: 'Enfocar comentario in-line',
        category: 'SpeedGrader',
        scope: 'speed-grader',
        action: () => this.selectLine(this.defaultCommentLine())
      })
    );

    // D: Alternar Visor de Diferencias (Diff vs Boilerplate)
    this.unregisterFns.push(
      this.hotkeysService.register({
        key: 'd',
        description: 'Alternar vista de diferencias (Diff)',
        category: 'SpeedGrader',
        scope: 'speed-grader',
        action: () => {
          if (!this.canShowDiff()) return;
          this.showReplayPlayer.set(false);
          this.showDiffView.update(v => !v);
        }
      })
    );

    // T: Alternar Time-Travel Replay de Escritura
    this.unregisterFns.push(
      this.hotkeysService.register({
        key: 't',
        description: 'Alternar reproductor Time-Travel Replay',
        category: 'SpeedGrader',
        scope: 'speed-grader',
        action: () => this.toggleReplay()
      })
    );

    // Ctrl+Enter: Guardar nota en modal o comentario
    this.unregisterFns.push(
      this.hotkeysService.register({
        key: 'ctrl+enter',
        description: 'Guardar calificación / Enviar comentario',
        category: 'Acciones',
        scope: 'speed-grader',
        allowInInput: true,
        action: () => {
          if (this.showOverrideModal()) {
            this.submitOverride();
          } else if (this.currentCommentRange()) {
            this.submitComment(this.currentCommentRange()!);
          }
        }
      })
    );
  }

  loadSubmission(id: string): void {
    this.liveApp.set(null);
    this.liveAppError.set(null);
    this.selectedLineNumber.set(null);
    this.selectedLineEndNumber.set(null);
    this.newCommentText.set('');
    this.overrideError.set(null);
    this.timeline.set(null);
    this.rubric.set(null);
    this.selections.set({});
    this.rubricComments.set('');
    this.rubricSuccessMessage.set(null);
    this.rubricErrorMessage.set(null);
    this.showDiffView.set(false);
    this.showReplayPlayer.set(false);
    this.selectedFilePath.set(null);
    this.treeCollapsed.set(false);

    this.gradingService.getSubmissionReview(id).subscribe({
      next: (rev) => {
        if (!rev) return;
        if (rev.environment_type === 'IDE_PERSISTENTE') {
          this.selectedFilePath.set(rev.files?.[0]?.path ?? null);
          if (rev.exercise_id) {
            this.loadRubric(rev.exercise_id);
          }
        }
      }
    });
  }

  loadRubric(exerciseId: string): void {
    this.http.get<unknown>(`/api/v1/exercises/${exerciseId}/rubric`).subscribe({
      next: (res) => {
        const data = (res as { data?: RubricDTO }).data ?? (res as RubricDTO);
        if (data && data.criteria && data.criteria.length > 0) {
          this.rubric.set(data);
          const initialSel: Record<string, string> = {};
          for (const c of data.criteria) {
            if (c.id && c.levels && c.levels.length > 0) {
              initialSel[c.id] = c.levels[0].name;
            }
          }
          this.selections.set(initialSel);
        }
      },
      error: (err) => {
        if (err?.status && err.status !== 404) {
          this.rubricErrorMessage.set(
            'No se pudo cargar la rúbrica de la práctica. Recargá la página para reintentar.'
          );
        }
      }
    });
  }

  toggleDiffView(): void {
    if (!this.canShowDiff()) return;
    this.showReplayPlayer.set(false);
    this.showDiffView.update(v => !v);
  }

  toggleReplay(): void {
    const nextVal = !this.showReplayPlayer();
    this.showReplayPlayer.set(nextVal);
    if (nextVal) {
      this.showDiffView.set(false);
      if (!this.timeline()) {
        this.loadTimeline();
      }
    }
  }

  loadTimeline(): void {
    const id = this.submissionId();
    if (!id) return;

    this.isLoadingTimeline.set(true);
    this.gradingService.getSubmissionTimeline(id).subscribe({
      next: (data) => {
        this.timeline.set(data);
        this.isLoadingTimeline.set(false);
      },
      error: () => {
        this.isLoadingTimeline.set(false);
      }
    });
  }

  

  openShortcutsGuide(): void {
    this.hotkeysService.openHelpModal();
  }

  selectFile(path: string): void {
    if (this.selectedFilePath() === path) return;
    this.selectedFilePath.set(path);
    this.selectedLineNumber.set(null);
    this.selectedLineEndNumber.set(null);
    this.newCommentText.set('');
    if (this.showDiffView() && !this.canShowDiff()) {
      this.showDiffView.set(false);
    }
  }

  private defaultCommentLine(): number {
    const file = this.activeFile();
    return file ? file.start_line : 1;
  }

  selectLine(lineNumber: number | null): void {
    if (lineNumber === null || this.selectedLineNumber() === lineNumber) {
      this.selectedLineNumber.set(null);
      this.selectedLineEndNumber.set(null);
    } else {
      this.selectedLineNumber.set(lineNumber);
      this.selectedLineEndNumber.set(null);
      this.newCommentText.set('');
    }
  }

  /** El visor emite líneas globales, ya traducidas desde el archivo activo. */
  selectLineRange(range: { start: number; end: number }): void {
    if (range.end <= range.start) {
      this.selectLine(range.start);
      return;
    }
    this.selectedLineNumber.set(range.start);
    this.selectedLineEndNumber.set(range.end);
    this.newCommentText.set('');
  }

  submitComment(range: { line: number; endLine: number | null }): void {
    const text = this.newCommentText().trim();
    if (!text) return;

    const id = this.submissionId();
    this.isAddingComment.set(true);

    // En IDE se envía también el ancla por archivo: el backend la persiste y
    // valida que coincida con la línea global que emite el visor.
    const file = this.isProject() ? this.activeFile() : null;
    const localLine = file ? globalToLocalLine(file, range.line) : null;
    const localEnd = file && range.endLine !== null ? globalToLocalLine(file, range.endLine) : null;

    this.gradingService.addComment(id, {
      line_number: range.line,
      ...(range.endLine !== null ? { end_line_number: range.endLine } : {}),
      ...(file && localLine !== null ? { file_path: file.path, local_line_number: localLine } : {}),
      ...(localEnd !== null ? { local_end_line_number: localEnd } : {}),
      comment: text
    }).subscribe({
      next: () => {
        this.newCommentText.set('');
        this.selectedLineNumber.set(null);
        this.selectedLineEndNumber.set(null);
        this.isAddingComment.set(false);
      },
      error: () => {
        this.isAddingComment.set(false);
      }
    });
  }

  runEphemeralSandbox(): void {
    const curr = this.review();
    if (!curr || !this.isJudge()) return;

    this.gradingService.runEphemeral(curr.id, {
      code: curr.code ?? '',
      language: curr.language || 'python'
    }).subscribe({
      next: () => {
        setTimeout(() => {
          const el = document.querySelector('.ephemeral-result-box');
          if (el) {
            el.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
          }
        }, 100);
      }
    });
  }

  openOverrideModal(): void {
    const curr = this.review();
    // En un proyecto no hay veredicto que proponer: solo se ajusta la nota.
    this.overrideVerdict.set(curr?.verdict || (this.isProject() ? '' : 'AC'));
    this.overrideScore.set(curr?.score ?? 100);
    this.overrideReason.set(curr?.override_reason || '');
    this.overrideError.set(null);
    this.showOverrideModal.set(true);
  }

  closeOverrideModal(): void {
    this.showOverrideModal.set(false);
  }

  submitOverride(): void {
    this.overrideError.set(null);
    const reason = this.overrideReason().trim() || 'Ajuste manual docente';
    const feedback = this.overrideReason().trim();
    const score = this.overrideScore();

    this.isSubmittingOverride.set(true);
    // Un proyecto se califica, no se le asigna un veredicto: la nota viaja por
    // la revisión docente, que es la misma vía que usa la libreta.
    const request$ = this.isProject()
      ? this.gradingService.updateReviewScore(this.submissionId(), {
          status: 'graded',
          score,
          ...(feedback ? { feedback_summary: feedback } : {})
        })
      : this.gradingService.overrideSubmission(this.submissionId(), {
          verdict: this.overrideVerdict(),
          override_reason: reason,
          score
        });

    request$.subscribe({
      next: () => {
        this.isSubmittingOverride.set(false);
        this.showOverrideModal.set(false);
      },
      error: (err) => {
        this.isSubmittingOverride.set(false);
        this.overrideError.set(err.error?.message || 'Error al actualizar calificación.');
      }
    });
  }
}
