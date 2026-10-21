import { Component, input, output } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { DateTextPipe } from '@shared/pipes/date-text.pipe';
import { TimeTravelReplayComponent } from '@shared/components/time-travel-replay/time-travel-replay.component';
import { ComplexityBenchmarkComponent } from '@shared/components/complexity-benchmark/complexity-benchmark.component';
import {
  TeacherSubmissionReviewDTO,
  SubmissionComment,
  SubmissionTimeline,
  BenchmarkReport
} from '../../../../models/teacher.models';
import { DiffLine } from '@shared/utils/diff.utils';

type BenchmarkPreset = 'small' | 'standard' | 'stress';

@Component({
  selector: 'speed-grader-code-viewer',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    DateTextPipe,
    TimeTravelReplayComponent,
    ComplexityBenchmarkComponent
  ],
  templateUrl: './speed-grader-code-viewer.component.html',
  styleUrl: './speed-grader-code-viewer.component.scss'
})
export class SpeedGraderCodeViewerComponent {
  readonly review = input.required<TeacherSubmissionReviewDTO>();
  readonly showReplayPlayer = input<boolean>(false);
  readonly showBenchmark = input<boolean>(false);
  readonly showDiffView = input<boolean>(false);
  readonly timeline = input<SubmissionTimeline | null>(null);
  readonly isLoadingTimeline = input<boolean>(false);
  readonly benchmarkReport = input<BenchmarkReport | null>(null);
  readonly isLoadingBenchmark = input<boolean>(false);
  readonly selectedLineNumber = input<number | null>(null);
  readonly newCommentText = input<string>('');
  readonly isAddingComment = input<boolean>(false);
  readonly codeLines = input<string[]>([]);
  readonly diffLines = input<DiffLine[]>([]);
  readonly commentsByLine = input<Map<number, SubmissionComment[]>>(new Map());
  readonly starterBoilerplate = input<string>('');

  readonly lineSelected = output<number | null>();
  readonly commentSubmitted = output<number>();
  readonly newCommentTextChange = output<string>();
  readonly toggleDiff = output<void>();
  readonly toggleReplay = output<void>();
  readonly toggleBenchmark = output<void>();
  readonly runBenchmark = output<BenchmarkPreset>();
}
