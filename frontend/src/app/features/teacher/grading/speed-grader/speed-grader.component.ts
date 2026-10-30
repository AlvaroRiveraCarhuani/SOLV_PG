import { Component, OnInit, OnDestroy, inject, signal, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, Router } from '@angular/router';
import { HttpClient } from '@angular/common/http';
import { RubricDTO } from '../../exercise-editor/exercise-editor.store';
import { TeacherGradingService } from '../../services/teacher-grading.service';
import { HotkeysService } from '@core/services/hotkeys.service';
import { SubmissionComment, SubmissionTimeline, LiveWorkspaceSession } from '../../models/teacher.models';
import { computeLineDiff, DiffLine } from '@shared/utils/diff.utils';
import { SkeletonLoaderComponent } from '@shared/components/skeleton/skeleton-loader.component';
import { ShadowTerminalModalComponent } from '../../aula/shadow-terminal-modal/shadow-terminal-modal.component';
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
    SkeletonLoaderComponent,
    ShadowTerminalModalComponent,
    KeystrokeReplayModalComponent,
    SpeedGraderHeaderComponent,
    SpeedGraderCodeViewerComponent,
    SpeedGraderEvalPanelComponent,
    SpeedGraderOverrideModalComponent
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
  comments = this.gradingService.comments;
  isLoading = this.gradingService.isLoading;
  isRunningEphemeral = this.gradingService.isRunningEphemeral;
  ephemeralResult = this.gradingService.ephemeralResult;

  // Rubrica
  rubric = signal<RubricDTO | null>(null);
  selections = signal<Record<string, string>>({});
  rubricComments = signal<string>('');
  isSubmittingRubric = signal<boolean>(false);
  rubricSuccessMessage = signal<string | null>(null);

  calculatedRubricScore = computed(() => {
    const rub = this.rubric();
    if (!rub || !rub.criteria) return 0;
    const sel = this.selections();
    let total = 0;
    for (const c of rub.criteria) {
      const selectedName = sel[c.id || ''] || (c.levels[0] ? c.levels[0].name : '');
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

    const payload = {
      selections: this.selections(),
      comments: this.rubricComments()
    };

    this.http.post<{ data?: { score?: number } }>(`/api/v1/submissions/${id}/rubric-evaluation`, payload).subscribe({
      next: (res) => {
        this.isSubmittingRubric.set(false);
        const score = res?.data?.score ?? this.calculatedRubricScore();
        this.rubricSuccessMessage.set(`Calificación guardada exitosamente (${score} / 100 pts)`);
        const currentRev = this.review();
        if (currentRev) {
          this.gradingService.currentReview.set({
            ...currentRev,
            score: Math.round(score),
            verdict: 'AC',
            manual_override: true,
            override_reason: 'Evaluado por rúbrica'
          });
        }
        if (this.rubricComments().trim()) {
          this.gradingService.getComments(id).subscribe();
        }
      },
      error: () => {
        this.isSubmittingRubric.set(false);
      }
    });
  }

  showDiffView = signal<boolean>(false);
  starterBoilerplate = signal<string>('// Plantilla base del laboratorio\n');

  // Time-Travel Replay State
  showReplayPlayer = signal<boolean>(false);
  timeline = signal<SubmissionTimeline | null>(null);
  isLoadingTimeline = signal<boolean>(false);

  selectedLineNumber = signal<number | null>(null);
  newCommentText = signal<string>('');
  isAddingComment = signal<boolean>(false);

  showOverrideModal = signal<boolean>(false);
  overrideVerdict = signal<string>('AC');
  overrideScore = signal<number>(100);
  overrideReason = signal<string>('');
  overrideNotifyStudent = signal<boolean>(true);
  overrideError = signal<string | null>(null);
  isSubmittingOverride = signal<boolean>(false);

  showShadowTerminal = signal<boolean>(false);
  showKeystrokeReplay = signal<boolean>(false);

  copiedIndex = signal<number | null>(null);

  openKeystrokeReplay(): void {
    this.showKeystrokeReplay.set(true);
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

  currentLiveSession = computed<LiveWorkspaceSession | null>(() => {
    const rev = this.review();
    if (!rev) return null;
    return {
      workspace_id: rev.id,
      container_id: rev.student_id ? `workspace-${rev.student_id.slice(0, 8)}` : '',
      student_id: rev.student_id,
      student_name: rev.student_name,
      student_email: rev.student_email,
      subject_id: rev.subject_id,
      subject_name: rev.subject_name,
      status: 'running',
      memory_limit_mb: 256,
      oom_strikes: 0,
      last_heartbeat: rev.submitted_at,
      is_attached: true
    };
  });

  openShadowTerminal(): void {
    this.showShadowTerminal.set(true);
  }

  closeShadowTerminal(): void {
    this.showShadowTerminal.set(false);
  }

  codeLines = computed(() => {
    return (this.review()?.code || '').split('\n');
  });

  diffLines = computed<DiffLine[]>(() => {
    return computeLineDiff(this.starterBoilerplate(), this.review()?.code || '');
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
        action: () => this.selectLine(1)
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
          } else if (this.selectedLineNumber()) {
            this.submitComment(this.selectedLineNumber()!);
          }
        }
      })
    );
  }

  loadSubmission(id: string): void {
    this.selectedLineNumber.set(null);
    this.newCommentText.set('');
    this.overrideError.set(null);
    this.timeline.set(null);
    this.rubric.set(null);
    this.selections.set({});
    this.rubricComments.set('');
    this.rubricSuccessMessage.set(null);

    this.gradingService.getSubmissionReview(id).subscribe({
      next: (rev) => {
        if (rev) {
          const code = rev.code || '';
          if (/^\s*(import\s+|def\s+|class\s+.*:|print\(|from\s+.*import)/m.test(code)) {
            this.starterBoilerplate.set('def busqueda_binaria(arr, x):\n    # Escriba su solución aquí\n    pass\n');
          } else if (/^\s*(#include|int\s+main)/m.test(code)) {
            this.starterBoilerplate.set('#include <iostream>\nusing namespace std;\n\nint main() {\n    // Escriba su solución aquí\n    return 0;\n}');
          } else {
            this.starterBoilerplate.set('// Plantilla base del laboratorio\n');
          }
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
      error: () => {}
    });
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

  selectLine(lineNumber: number | null): void {
    if (lineNumber === null || this.selectedLineNumber() === lineNumber) {
      this.selectedLineNumber.set(null);
    } else {
      this.selectedLineNumber.set(lineNumber);
      this.newCommentText.set('');
    }
  }

  submitComment(lineNumber: number): void {
    const text = this.newCommentText().trim();
    if (!text) return;

    const id = this.submissionId();
    this.isAddingComment.set(true);

    this.gradingService.addComment(id, {
      line_number: lineNumber,
      comment: text
    }).subscribe({
      next: () => {
        this.newCommentText.set('');
        this.selectedLineNumber.set(null);
        this.isAddingComment.set(false);
      },
      error: () => {
        this.isAddingComment.set(false);
      }
    });
  }

  runEphemeralSandbox(): void {
    const curr = this.review();
    if (!curr) return;

    this.gradingService.runEphemeral(curr.id, {
      code: curr.code,
      language: 'python'
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
    this.overrideVerdict.set(curr?.verdict || 'AC');
    this.overrideScore.set(curr?.score || 100);
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

    this.isSubmittingOverride.set(true);
    this.gradingService.overrideSubmission(this.submissionId(), {
      verdict: this.overrideVerdict(),
      override_reason: reason,
      score: this.overrideScore()
    }).subscribe({
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
