import { 
  Component, 
  ChangeDetectionStrategy, 
  input, 
  output, 
  signal, 
  computed, 
  effect, 
  OnDestroy, 
  HostListener 
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { 
  LucideHistory, 
  LucidePlay, 
  LucidePause, 
  LucideRotateCcw, 
  LucideChevronLeft, 
  LucideChevronRight, 
  LucideGitCompare, 
  LucideAlertTriangle, 
  LucideAlertCircle, 
  LucideCheckCircle, 
  LucideLayers, 
  LucideX 
} from '@lucide/angular';
import { SubmissionTimeline, TimelineKeyframe } from '../../../features/teacher/models/teacher.models';
import { SparklineComponent } from '../sparkline/sparkline.component';
import { MachineDataDirective } from '../../directives/machine-data.directive';
import { computeLineDiff, DiffLine } from '../../utils/diff.utils';

@Component({
  selector: 'time-travel-replay',
  standalone: true,
  imports: [
    CommonModule,
    LucideHistory,
    LucidePlay,
    LucidePause,
    LucideRotateCcw,
    LucideChevronLeft,
    LucideChevronRight,
    LucideGitCompare,
    LucideAlertTriangle,
    LucideAlertCircle,
    LucideCheckCircle,
    LucideLayers,
    LucideX,
    SparklineComponent,
    MachineDataDirective
  ],
  templateUrl: './time-travel-replay.component.html',
  styleUrl: './time-travel-replay.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class TimeTravelReplayComponent implements OnDestroy {
  readonly timeline = input<SubmissionTimeline | null>(null);
  readonly initialBoilerplate = input<string>('');
  readonly mode = input<'embedded' | 'drawer' | 'modal'>('embedded');
  readonly title = input<string>('Time-Travel Replay & AST Timeline');

  readonly close = output<void>();
  readonly keyframeChange = output<TimelineKeyframe>();

  readonly currentKeyframeIndex = signal<number>(0);
  readonly isPlaying = signal<boolean>(false);
  readonly speed = signal<number>(1);
  readonly showDiff = signal<boolean>(false);

  private playbackTimer?: ReturnType<typeof setInterval>;

  constructor() {
    effect(() => {
      const tl = this.timeline();
      if (tl && tl.keyframes.length > 0) {
        this.currentKeyframeIndex.set(tl.keyframes.length - 1);
      }
    });
  }

  ngOnDestroy(): void {
    this.stopPlayback();
  }

  @HostListener('document:keydown', ['$event'])
  handleKeyboardEvent(event: KeyboardEvent): void {
    // Only listen if not typing in an input/textarea
    const target = event.target as HTMLElement;
    if (target && (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA')) {
      return;
    }

    if (event.code === 'Space') {
      event.preventDefault();
      this.togglePlayPause();
    } else if (event.code === 'ArrowLeft') {
      event.preventDefault();
      this.stepBackward();
    } else if (event.code === 'ArrowRight') {
      event.preventDefault();
      this.stepForward();
    } else if (event.code === 'Home') {
      event.preventDefault();
      this.resetPlayback();
    } else if (event.code === 'Escape' && (this.mode() === 'modal' || this.mode() === 'drawer')) {
      this.close.emit();
    }
  }

  readonly maxKeyframeIndex = computed(() => {
    const list = this.timeline()?.keyframes;
    if (!list || list.length === 0) return 0;
    return list.length - 1;
  });

  readonly isAtEnd = computed(() => {
    return this.currentKeyframeIndex() >= this.maxKeyframeIndex();
  });

  readonly currentKeyframe = computed<TimelineKeyframe | null>(() => {
    const list = this.timeline()?.keyframes;
    if (!list || list.length === 0) return null;
    const idx = Math.min(this.currentKeyframeIndex(), list.length - 1);
    return list[idx] ?? null;
  });

  readonly currentCodeContent = computed<string>(() => {
    return this.currentKeyframe()?.content || '';
  });

  readonly codeLines = computed<string[]>(() => {
    return this.currentCodeContent().split('\n');
  });

  readonly diffLines = computed<DiffLine[]>(() => {
    return computeLineDiff(this.initialBoilerplate(), this.currentCodeContent());
  });

  readonly astHistoryData = computed<number[]>(() => {
    const list = this.timeline()?.keyframes;
    if (!list || list.length === 0) return [];
    return list.map(k => k.ast_node_count ?? (k.content.length / 10));
  });

  togglePlayPause(): void {
    if (this.isPlaying()) {
      this.pausePlayback();
    } else {
      this.startPlayback();
    }
  }

  startPlayback(): void {
    const maxIdx = this.maxKeyframeIndex();
    if (this.currentKeyframeIndex() >= maxIdx) {
      this.currentKeyframeIndex.set(0);
    }

    this.isPlaying.set(true);
    this.scheduleNextTick();
  }

  pausePlayback(): void {
    this.isPlaying.set(false);
    this.stopPlayback();
  }

  stopPlayback(): void {
    if (this.playbackTimer) {
      clearInterval(this.playbackTimer);
      this.playbackTimer = undefined;
    }
  }

  resetPlayback(): void {
    this.pausePlayback();
    this.seek(0);
  }

  stepForward(): void {
    const next = this.currentKeyframeIndex() + 1;
    if (next <= this.maxKeyframeIndex()) {
      this.seek(next);
    }
  }

  stepBackward(): void {
    const prev = this.currentKeyframeIndex() - 1;
    if (prev >= 0) {
      this.seek(prev);
    }
  }

  seek(index: number): void {
    const clamped = Math.max(0, Math.min(index, this.maxKeyframeIndex()));
    this.currentKeyframeIndex.set(clamped);
    const kf = this.currentKeyframe();
    if (kf) {
      this.keyframeChange.emit(kf);
    }
  }

  onSliderInput(event: Event): void {
    const value = +(event.target as HTMLInputElement).value;
    this.seek(value);
  }

  setSpeed(multiplier: number): void {
    this.speed.set(multiplier);
    if (this.isPlaying()) {
      this.stopPlayback();
      this.scheduleNextTick();
    }
  }

  toggleDiff(): void {
    this.showDiff.update(v => !v);
  }

  getKeyframePercent(index: number): number {
    const max = this.maxKeyframeIndex();
    if (max === 0) return 0;
    return Math.round((index / max) * 100);
  }

  formatDuration(seconds: number): string {
    if (!seconds || seconds <= 0) return '0s';
    if (seconds < 60) return `${Math.round(seconds)}s`;
    const mins = Math.floor(seconds / 60);
    const rem = Math.round(seconds % 60);
    return `${mins}m ${rem}s`;
  }

  formatOffset(offsetMs: number): string {
    if (!offsetMs || offsetMs <= 0) return '0s';
    const totalSec = Math.round(offsetMs / 1000);
    return this.formatDuration(totalSec);
  }

  private scheduleNextTick(): void {
    const interval = Math.max(40, Math.round(350 / this.speed()));
    this.playbackTimer = setInterval(() => {
      const nextIdx = this.currentKeyframeIndex() + 1;
      if (nextIdx <= this.maxKeyframeIndex()) {
        this.seek(nextIdx);
      } else {
        this.pausePlayback();
      }
    }, interval);
  }
}
