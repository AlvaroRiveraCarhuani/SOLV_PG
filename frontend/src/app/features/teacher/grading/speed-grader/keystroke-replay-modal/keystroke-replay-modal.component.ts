import { Component, OnInit, OnDestroy, inject, input, output, signal, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { 
  LucidePlay, 
  LucidePause, 
  LucideSkipForward, 
  LucideSkipBack, 
  LucideAlertTriangle, 
  LucideClock, 
  LucideClipboard, 
  LucidePercent, 
  LucideFilm,
  LucideCheckCircle
} from '@lucide/angular';
import { TeacherGradingService } from '../../../services/teacher-grading.service';
import { SubmissionKeystrokeEvent, SubmissionKeystrokeReport } from '../../../models/teacher.models';
import { ModalShellComponent } from '@shared/components/modal-shell/modal-shell.component';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { SkeletonLoaderComponent } from '@shared/components/skeleton/skeleton-loader.component';

interface CodeReplaySegment {
  text: string;
  isPaste: boolean;
  timestamp: number;
}

@Component({
  selector: 'keystroke-replay-modal',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    LucidePlay,
    LucidePause,
    LucideSkipForward,
    LucideSkipBack,
    LucideAlertTriangle,
    LucideClock,
    LucideClipboard,
    LucidePercent,
    LucideFilm,
    LucideCheckCircle,
    ModalShellComponent,
    MachineDataDirective,
    SkeletonLoaderComponent
  ],
  templateUrl: './keystroke-replay-modal.component.html',
  styleUrl: './keystroke-replay-modal.component.scss'
})
export class KeystrokeReplayModalComponent implements OnInit, OnDestroy {
  readonly submissionId = input.required<string>();
  readonly close = output<void>();

  private gradingService = inject(TeacherGradingService);

  report = signal<SubmissionKeystrokeReport | null>(null);
  isLoading = signal<boolean>(true);
  error = signal<string | null>(null);

  // Playback state
  currentTimeMS = signal<number>(0);
  isPlaying = signal<boolean>(false);
  playbackSpeed = signal<number>(1); // 0.5, 1, 2, 4
  private playIntervalId: ReturnType<typeof setInterval> | null = null;

  ngOnInit(): void {
    this.loadKeystrokeEvents();
  }

  ngOnDestroy(): void {
    this.pause();
  }

  loadKeystrokeEvents(): void {
    const id = this.submissionId();
    if (!id) return;

    this.isLoading.set(true);
    this.error.set(null);

    this.gradingService.getKeystrokeEvents(id).subscribe({
      next: (rep) => {
        this.report.set(rep);
        this.isLoading.set(false);
        this.currentTimeMS.set(0);
      },
      error: () => {
        this.isLoading.set(false);
        this.error.set('No se pudieron cargar los eventos de auditoría de escritura.');
      }
    });
  }

  totalDurationMS = computed<number>(() => {
    const rep = this.report();
    if (!rep || rep.events.length === 0) return 0;
    return rep.total_time_ms > 0 ? rep.total_time_ms : rep.events[rep.events.length - 1].timestamp_ms;
  });

  // Events filtered up to the current timestamp
  activeEvents = computed<SubmissionKeystrokeEvent[]>(() => {
    const rep = this.report();
    if (!rep) return [];
    const t = this.currentTimeMS();
    return rep.events.filter(e => e.timestamp_ms <= t);
  });

  // Reconstruct code state with segments tracking paste origin
  codeSegments = computed<CodeReplaySegment[]>(() => {
    const events = this.activeEvents();
    if (events.length === 0) {
      return [{ text: '', isPaste: false, timestamp: 0 }];
    }

    const segments: CodeReplaySegment[] = [];
    for (const ev of events) {
      const isPaste = !!(ev.event_type === 'paste' || ev.paste_source_detected);
      if (ev.event_type === 'delete') {
        if (segments.length > 0) {
          const last = segments[segments.length - 1];
          if (last.text.length > 0) {
            last.text = last.text.slice(0, -1);
          }
        }
      } else {
        segments.push({
          text: ev.content,
          isPaste,
          timestamp: ev.timestamp_ms
        });
      }
    }
    return segments;
  });

  formattedCurrentTime = computed<string>(() => {
    const s = Math.floor(this.currentTimeMS() / 1000);
    const mins = Math.floor(s / 60);
    const secs = s % 60;
    return `${mins.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`;
  });

  formattedTotalTime = computed<string>(() => {
    const s = Math.floor(this.totalDurationMS() / 1000);
    const mins = Math.floor(s / 60);
    const secs = s % 60;
    return `${mins.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`;
  });

  // Playback controls
  togglePlay(): void {
    if (this.isPlaying()) {
      this.pause();
    } else {
      this.play();
    }
  }

  play(): void {
    if (this.currentTimeMS() >= this.totalDurationMS()) {
      this.currentTimeMS.set(0);
    }
    this.isPlaying.set(true);
    const stepMS = 100;
    this.playIntervalId = setInterval(() => {
      const next = this.currentTimeMS() + (stepMS * this.playbackSpeed());
      if (next >= this.totalDurationMS()) {
        this.currentTimeMS.set(this.totalDurationMS());
        this.pause();
      } else {
        this.currentTimeMS.set(next);
      }
    }, stepMS);
  }

  pause(): void {
    this.isPlaying.set(false);
    if (this.playIntervalId) {
      clearInterval(this.playIntervalId);
      this.playIntervalId = null;
    }
  }

  stepForward(): void {
    this.pause();
    const rep = this.report();
    if (!rep) return;
    const current = this.currentTimeMS();
    const nextEvent = rep.events.find(e => e.timestamp_ms > current);
    if (nextEvent) {
      this.currentTimeMS.set(nextEvent.timestamp_ms);
    } else {
      this.currentTimeMS.set(this.totalDurationMS());
    }
  }

  stepBackward(): void {
    this.pause();
    const rep = this.report();
    if (!rep) return;
    const current = this.currentTimeMS();
    const prevEvents = rep.events.filter(e => e.timestamp_ms < current);
    if (prevEvents.length > 0) {
      this.currentTimeMS.set(prevEvents[prevEvents.length - 1].timestamp_ms);
    } else {
      this.currentTimeMS.set(0);
    }
  }

  setSpeed(speed: number): void {
    this.playbackSpeed.set(speed);
    if (this.isPlaying()) {
      this.pause();
      this.play();
    }
  }

  onSliderChange(event: Event): void {
    const val = Number((event.target as HTMLInputElement).value);
    this.currentTimeMS.set(val);
  }
}
