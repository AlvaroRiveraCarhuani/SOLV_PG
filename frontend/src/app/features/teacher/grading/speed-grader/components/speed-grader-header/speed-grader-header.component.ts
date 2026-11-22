import { Component, input, output, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink } from '@angular/router';
import {
  LucideArrowLeft,
  LucideChevronLeft,
  LucideChevronRight,
  LucideChevronDown,
  LucidePlay,
  LucideHistory,
  LucideGitCompare,
  LucideEdit3,
  LucideKeyboard,
  LucideFilm,
  LucideSlidersHorizontal,
  LucideLoader2,
  LucideExternalLink
} from '@lucide/angular';
import { ReviewMode, TeacherSubmissionReviewDTO } from '../../../../models/teacher.models';

import { EnvironmentLabelPipe } from '@shared/pipes/environment-label.pipe';

@Component({
  selector: 'speed-grader-header',
  standalone: true,
  imports: [
    EnvironmentLabelPipe,
    CommonModule,
    RouterLink,
    LucideArrowLeft,
    LucideChevronLeft,
    LucideChevronRight,
    LucideChevronDown,
    LucidePlay,
    LucideHistory,
    LucideGitCompare,
    LucideEdit3,
    LucideKeyboard,
    LucideFilm,
    LucideSlidersHorizontal,
    LucideLoader2,
    LucideExternalLink
  ],
  templateUrl: './speed-grader-header.component.html',
  styleUrl: './speed-grader-header.component.scss'
})
export class SpeedGraderHeaderComponent {
  readonly review = input.required<TeacherSubmissionReviewDTO>();
  readonly mode = input.required<ReviewMode>();
  readonly showReplayPlayer = input<boolean>(false);
  readonly showDiffView = input<boolean>(false);
  /** Hay una base real contra la que comparar (boilerplate o archivo base IDE). */
  readonly diffAvailable = input<boolean>(true);
  readonly isRunningEphemeral = input<boolean>(false);
  readonly isStartingLiveApp = input<boolean>(false);

  showToolsMenu = signal<boolean>(false);

  readonly openKeystrokeReplay = output<void>();
  readonly toggleReplay = output<void>();
  readonly runEphemeral = output<void>();
  readonly openLiveApp = output<void>();
  readonly openOverrideModal = output<void>();
  readonly openShortcutsGuide = output<void>();
  readonly toggleDiff = output<void>();

  /** Iniciales del alumno para el avatar del paginador. */
  initials(name: string): string {
    const parts = (name || '')
      .trim()
      .split(/\s+/)
      .filter(Boolean);
    if (parts.length === 0) {
      return '—';
    }
    const first = parts[0].charAt(0);
    const second = parts.length > 1 ? parts[1].charAt(0) : '';
    return `${first}${second}`.toUpperCase();
  }
}
