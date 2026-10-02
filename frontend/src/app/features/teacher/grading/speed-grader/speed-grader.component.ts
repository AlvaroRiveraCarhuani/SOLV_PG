import { Component, OnInit, OnDestroy, inject, signal, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { FormsModule } from '@angular/forms';
import { 
  LucideArrowLeft, 
  LucideChevronLeft, 
  LucideChevronRight, 
  LucidePlay, 
  LucidePause,
  LucideRotateCcw,
  LucideEdit3, 
  LucideTerminal, 
  LucideCheckCircle, 
  LucideXCircle, 
  LucideAlertTriangle, 
  LucideKeyboard,
  LucideGitCompare,
  LucideHistory
} from '@lucide/angular';
import { TeacherGradingService } from '../../services/teacher-grading.service';
import { HotkeysService } from '@core/services/hotkeys.service';
import { SubmissionComment, SubmissionTimeline, TimelineKeyframe } from '../../models/teacher.models';
import { DateTextPipe } from '@shared/pipes/date-text.pipe';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { SkeletonLoaderComponent } from '@shared/components/skeleton/skeleton-loader.component';
import { computeLineDiff, DiffLine } from '@shared/utils/diff.utils';

@Component({
  selector: 'speed-grader',
  standalone: true,
  imports: [
    CommonModule,
    RouterLink,
    FormsModule,
    LucideArrowLeft,
    LucideChevronLeft,
    LucideChevronRight,
    LucidePlay,
    LucidePause,
    LucideRotateCcw,
    LucideEdit3,
    LucideTerminal,
    LucideCheckCircle,
    LucideXCircle,
    LucideAlertTriangle,
    LucideKeyboard,
    LucideGitCompare,
    LucideHistory,
    DateTextPipe,
    MachineDataDirective,
    SkeletonLoaderComponent
  ],
  templateUrl: './speed-grader.component.html',
  styleUrl: './speed-grader.component.scss'
})
export class SpeedGraderComponent implements OnInit, OnDestroy {
  private route = inject(ActivatedRoute);
  private router = inject(Router);
  private gradingService = inject(TeacherGradingService);
  private hotkeysService = inject(HotkeysService);

  private unregisterFns: Array<() => void> = [];
  private replayTimer: any = null;

  submissionId = signal<string>('');
  review = this.gradingService.currentReview;
  comments = this.gradingService.comments;
  isLoading = this.gradingService.isLoading;
  isRunningEphemeral = this.gradingService.isRunningEphemeral;
  ephemeralResult = this.gradingService.ephemeralResult;

  showDiffView = signal<boolean>(false);
  starterBoilerplate = signal<string>('#include <iostream>\nusing namespace std;\n\nint main() {\n    // Escribe tu solución aquí\n    return 0;\n}');

  // Time-Travel Replay State
  showReplayPlayer = signal<boolean>(false);
  isPlayingReplay = signal<boolean>(false);
  replaySpeed = signal<number>(1);
  currentKeyframeIndex = signal<number>(0);
  timeline = signal<SubmissionTimeline | null>(null);
  isLoadingTimeline = signal<boolean>(false);

  selectedLineNumber = signal<number | null>(null);
  newCommentText = signal<string>('');
  isAddingComment = signal<boolean>(false);

  showOverrideModal = signal<boolean>(false);
  overrideVerdict = signal<string>('AC');
  overrideScore = signal<number>(100);
  overrideReason = signal<string>('');
  overrideError = signal<string | null>(null);
  isSubmittingOverride = signal<boolean>(false);

  activeCodeContent = computed<string>(() => {
    if (this.showReplayPlayer()) {
      const tl = this.timeline();
      if (tl && tl.keyframes.length > 0) {
        const idx = Math.min(this.currentKeyframeIndex(), tl.keyframes.length - 1);
        return tl.keyframes[idx]?.content ?? '';
      }
    }
    return this.review()?.code || '';
  });

  codeLines = computed(() => {
    return this.activeCodeContent().split('\n');
  });

  diffLines = computed<DiffLine[]>(() => {
    return computeLineDiff(this.starterBoilerplate(), this.activeCodeContent());
  });

  currentKeyframe = computed<TimelineKeyframe | null>(() => {
    const tl = this.timeline();
    if (!tl || tl.keyframes.length === 0) return null;
    const idx = Math.min(this.currentKeyframeIndex(), tl.keyframes.length - 1);
    return tl.keyframes[idx] ?? null;
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
    this.pauseReplay();
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
    this.pauseReplay();
    this.gradingService.getSubmissionReview(id).subscribe();
  }

  toggleReplay(): void {
    const nextVal = !this.showReplayPlayer();
    this.showReplayPlayer.set(nextVal);
    if (nextVal) {
      this.showDiffView.set(false);
      if (!this.timeline()) {
        this.loadTimeline();
      }
    } else {
      this.pauseReplay();
    }
  }

  loadTimeline(): void {
    const id = this.submissionId();
    if (!id) return;

    this.isLoadingTimeline.set(true);
    this.gradingService.getSubmissionTimeline(id).subscribe({
      next: (data) => {
        this.timeline.set(data);
        this.currentKeyframeIndex.set(data.keyframes.length - 1);
        this.isLoadingTimeline.set(false);
      },
      error: () => {
        this.isLoadingTimeline.set(false);
      }
    });
  }

  playReplay(): void {
    const tl = this.timeline();
    if (!tl || tl.keyframes.length === 0) return;

    if (this.currentKeyframeIndex() >= tl.keyframes.length - 1) {
      this.currentKeyframeIndex.set(0);
    }

    this.isPlayingReplay.set(true);
    if (this.replayTimer) clearInterval(this.replayTimer);

    const intervalMs = Math.max(50, Math.round(300 / this.replaySpeed()));
    this.replayTimer = setInterval(() => {
      const nextIdx = this.currentKeyframeIndex() + 1;
      if (nextIdx < tl.keyframes.length) {
        this.currentKeyframeIndex.set(nextIdx);
      } else {
        this.pauseReplay();
      }
    }, intervalMs);
  }

  pauseReplay(): void {
    this.isPlayingReplay.set(false);
    if (this.replayTimer) {
      clearInterval(this.replayTimer);
      this.replayTimer = null;
    }
  }

  resetReplay(): void {
    this.pauseReplay();
    this.currentKeyframeIndex.set(0);
  }

  setReplaySpeed(speed: number): void {
    this.replaySpeed.set(speed);
    if (this.isPlayingReplay()) {
      this.playReplay();
    }
  }

  seekKeyframe(index: number): void {
    this.currentKeyframeIndex.set(index);
  }

  openShortcutsGuide(): void {
    this.hotkeysService.openHelpModal();
  }

  selectLine(lineNumber: number): void {
    if (this.selectedLineNumber() === lineNumber) {
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
    }).subscribe();
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
    const reason = this.overrideReason().trim();

    if (reason.length < 10) {
      this.overrideError.set('La justificación debe tener al menos 10 caracteres para auditoría.');
      return;
    }

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
