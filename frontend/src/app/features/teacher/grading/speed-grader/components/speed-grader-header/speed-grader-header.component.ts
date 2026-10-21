import { Component, input, output } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink } from '@angular/router';
import {
  LucideArrowLeft,
  LucideChevronLeft,
  LucideChevronRight,
  LucidePlay,
  LucideHistory,
  LucideTrendingUp,
  LucideGitCompare,
  LucideTerminal,
  LucideEdit3,
  LucideKeyboard,
  LucideFilm
} from '@lucide/angular';
import { TeacherSubmissionReviewDTO } from '../../../../models/teacher.models';

@Component({
  selector: 'speed-grader-header',
  standalone: true,
  imports: [
    CommonModule,
    RouterLink,
    LucideArrowLeft,
    LucideChevronLeft,
    LucideChevronRight,
    LucidePlay,
    LucideHistory,
    LucideTrendingUp,
    LucideGitCompare,
    LucideTerminal,
    LucideEdit3,
    LucideKeyboard,
    LucideFilm
  ],
  templateUrl: './speed-grader-header.component.html',
  styleUrl: './speed-grader-header.component.scss'
})
export class SpeedGraderHeaderComponent {
  readonly review = input.required<TeacherSubmissionReviewDTO>();
  readonly showReplayPlayer = input<boolean>(false);
  readonly showBenchmark = input<boolean>(false);
  readonly showDiffView = input<boolean>(false);
  readonly isRunningEphemeral = input<boolean>(false);

  readonly goBack = output<void>();
  readonly openKeystrokeReplay = output<void>();
  readonly toggleReplay = output<void>();
  readonly toggleBenchmark = output<void>();
  readonly runEphemeral = output<void>();
  readonly openShadowTerminal = output<void>();
  readonly openOverrideModal = output<void>();
  readonly openShortcutsGuide = output<void>();
  readonly toggleDiff = output<void>();
}
