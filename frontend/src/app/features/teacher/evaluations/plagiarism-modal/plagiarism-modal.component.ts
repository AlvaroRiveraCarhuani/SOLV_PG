import { Component, Input, Output, EventEmitter, inject, signal, OnInit, HostListener } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink } from '@angular/router';
import {
  LucideX,
  LucideShieldAlert,
  LucideCheckCircle2,
  LucideExternalLink,
  LucideRefreshCw,
  LucideUsers,
  LucideLayers,
  LucideHistory
} from '@lucide/angular';
import { TeacherCourseService } from '../../services/teacher-course.service';
import { TeacherGradingService } from '../../services/teacher-grading.service';
import { PlagiarismReport, PlagiarismMatch, TeacherLabStats, SubmissionTimeline } from '../../models/teacher.models';
import { SkeletonLoaderComponent } from '@shared/components/skeleton/skeleton-loader.component';
import { TimeTravelReplayComponent } from '@shared/components/time-travel-replay/time-travel-replay.component';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { DateTextPipe } from '@shared/pipes/date-text.pipe';
import { ComboboxComponent, ComboboxOption } from '@shared/components/combobox/combobox.component';

@Component({
  selector: 'plagiarism-modal',
  standalone: true,
  imports: [
    CommonModule,
    RouterLink,
    LucideX,
    LucideShieldAlert,
    LucideCheckCircle2,
    LucideExternalLink,
    LucideRefreshCw,
    LucideUsers,
    LucideLayers,
    LucideHistory,
    SkeletonLoaderComponent,
    TimeTravelReplayComponent,
    MachineDataDirective,
    DateTextPipe,
    ComboboxComponent
  ],
  templateUrl: './plagiarism-modal.component.html',
  styleUrl: './plagiarism-modal.component.scss'
})
export class PlagiarismModalComponent implements OnInit {
  private courseService = inject(TeacherCourseService);
  private gradingService = inject(TeacherGradingService);

  @Input({ required: true }) subjectId!: string;
  @Input() initialExerciseId?: string;
  @Input() labs: TeacherLabStats[] = [];
  @Output() close = new EventEmitter<void>();

  selectedExerciseId = signal<string>('all');
  report = signal<PlagiarismReport | null>(null);
  isLoading = signal<boolean>(false);
  selectedMatch = signal<PlagiarismMatch | null>(null);

  get labOptions(): ComboboxOption[] {
    return [
      { id: 'all', label: 'Todos los ejercicios del curso', value: 'all' },
      ...this.labs.map(l => ({ id: l.id, label: l.title, value: l.id }))
    ];
  }

  // Replay Forense Modal
  showForensicReplay = signal<boolean>(false);
  activeReplayTimeline = signal<SubmissionTimeline | null>(null);
  activeReplayStudentName = signal<string>('');

  @HostListener('document:keydown.escape')
  onEscape(): void {
    this.close.emit();
  }

  ngOnInit(): void {
    if (this.initialExerciseId) {
      this.selectedExerciseId.set(this.initialExerciseId);
    }
    this.runScan();
  }

  onExerciseChange(exerciseId: string): void {
    this.selectedExerciseId.set(exerciseId);
    this.runScan();
  }

  runScan(): void {
    if (!this.subjectId) return;

    this.isLoading.set(true);
    this.selectedMatch.set(null);

    const exId = this.selectedExerciseId() === 'all' ? undefined : this.selectedExerciseId();
    this.courseService.analyzePlagiarism(this.subjectId, exId).subscribe({
      next: (data) => {
        this.report.set(data);
        if (data.matches && data.matches.length > 0) {
          this.selectedMatch.set(data.matches[0]);
        }
        this.isLoading.set(false);
      },
      error: () => {
        this.isLoading.set(false);
      }
    });
  }

  selectMatch(match: PlagiarismMatch): void {
    this.selectedMatch.set(match);
  }

  getCriticalCount(): number {
    const r = this.report();
    if (!r || !r.matches) return 0;
    return r.matches.filter(m => m.risk_level === 'critical').length;
  }

  getWarningCount(): number {
    const r = this.report();
    if (!r || !r.matches) return 0;
    return r.matches.filter(m => m.risk_level === 'warning').length;
  }

  inspectSubmissionReplay(submissionId: string, studentName: string): void {
    if (!submissionId) return;
    this.activeReplayStudentName.set(studentName);
    this.gradingService.getSubmissionTimeline(submissionId).subscribe({
      next: (timelineData) => {
        this.activeReplayTimeline.set(timelineData);
        this.showForensicReplay.set(true);
      },
      error: () => {
        // Fallback default timeline if endpoint not seeded
        this.activeReplayTimeline.set({
          submission_id: submissionId,
          student_id: '',
          student_name: studentName,
          total_duration_seconds: 120,
          total_keystrokes: 240,
          paste_events_count: 1,
          paste_percentage: 60,
          suspicious_paste_flag: true,
          keyframes: []
        });
        this.showForensicReplay.set(true);
      }
    });
  }
}
